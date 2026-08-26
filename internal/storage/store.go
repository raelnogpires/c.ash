// Package storage persists application data in SQLite.
package storage

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"c.ash/internal/domain"
	"github.com/gofrs/flock"
	_ "modernc.org/sqlite"
)

var ErrAlreadyRunning = errors.New("database is already open by another application instance")

//go:embed migrations/*.sql
var migrationFiles embed.FS

type Store struct {
	db   *sql.DB
	lock *flock.Flock
}

type Queries interface {
	Profile(context.Context) (*domain.Profile, error)
	Accounts(context.Context) ([]domain.Account, error)
	Account(context.Context, string) (*domain.Account, error)
	AccountTransactions(context.Context, string) ([]domain.Transaction, error)
	Categories(context.Context) ([]domain.Category, error)
	Category(context.Context, string) (*domain.Category, error)
	Transactions(context.Context) ([]domain.Transaction, error)
	TrashedTransactions(context.Context) ([]domain.Transaction, error)
	Transaction(context.Context, string) (*domain.Transaction, error)
	FixedExpenses(context.Context) ([]domain.FixedExpense, error)
	FixedExpense(context.Context, string) (*domain.FixedExpense, error)
	FixedExpenseOccurrences(context.Context) ([]domain.FixedExpenseOccurrence, error)
	FixedExpenseOccurrence(context.Context, string) (*domain.FixedExpenseOccurrence, error)
	OccurrenceForTransaction(context.Context, string) (*domain.FixedExpenseOccurrence, error)
	CreditCardInvoices(context.Context, string) ([]domain.CreditCardInvoice, error)
	CreditCardInvoice(context.Context, string) (*domain.CreditCardInvoice, error)
	InvoiceInstallments(context.Context, string) ([]domain.CreditCardInstallment, error)
	TransactionInstallments(context.Context, string) ([]domain.CreditCardInstallment, error)
	InvoicePayments(context.Context, string) ([]domain.CreditCardPayment, error)
	SaveProfile(context.Context, domain.Profile, string) error
	SetBalancesHidden(context.Context, bool, string) error
	InsertAccount(context.Context, domain.Account, string) error
	UpdateAccount(context.Context, domain.Account, string) error
	DeleteAccount(context.Context, string) error
	InsertTransaction(context.Context, domain.Transaction, string) error
	UpdateTransaction(context.Context, domain.Transaction, string) error
	SetTransactionDeletedAt(context.Context, string, string, string) error
	DeleteTransactionRevisions(context.Context, string) error
	DeleteTransaction(context.Context, string) error
	InsertTransactionRevision(context.Context, domain.Transaction, string, string) error
	InsertFixedExpense(context.Context, domain.FixedExpense, string) error
	UpdateFixedExpense(context.Context, domain.FixedExpense, string) error
	SetFixedExpenseArchivedAt(context.Context, string, string, string) error
	InsertFixedExpenseOccurrence(context.Context, domain.FixedExpenseOccurrence, string) error
	SetFixedExpenseOccurrence(context.Context, string, domain.FixedExpenseOccurrenceStatus, string, string) error
	InsertCreditCardInvoice(context.Context, domain.CreditCardInvoice, string) error
	UpdateCreditCardInvoice(context.Context, domain.CreditCardInvoiceStatus, int64, string, string) error
	InsertCreditCardInstallment(context.Context, domain.CreditCardInstallment, string) error
	DeleteTransactionInstallments(context.Context, string) error
	UpdateTransactionInstallmentDescriptions(context.Context, string, string) error
	InsertCreditCardPayment(context.Context, domain.CreditCardPayment) error
}

type dbQueries struct{ q sqlQuerier }

type sqlQuerier interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user data directory: %w", err)
	}
	return filepath.Join(dir, "c.ash", "cash.db"), nil
}

func Open(ctx context.Context, path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}
	fileLock := flock.New(path + ".lock")
	locked, err := fileLock.TryLock()
	if err != nil {
		return nil, fmt.Errorf("acquire database lock: %w", err)
	}
	if !locked {
		return nil, ErrAlreadyRunning
	}
	dsn := "file:" + filepath.ToSlash(path) + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		_ = fileLock.Unlock()
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := ApplyMigrations(ctx, db, migrationFiles); err != nil {
		_ = db.Close()
		_ = fileLock.Unlock()
		return nil, err
	}
	return &Store{db: db, lock: fileLock}, nil
}

func (s *Store) Close() error {
	dbErr := s.db.Close()
	lockErr := s.lock.Unlock()
	return errors.Join(dbErr, lockErr)
}

func ApplyMigrations(ctx context.Context, db *sql.DB, files fs.FS) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil {
		return fmt.Errorf("create migration history: %w", err)
	}
	entries, err := fs.Glob(files, "migrations/*.sql")
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(entries)
	for _, name := range entries {
		version := strings.TrimSuffix(filepath.Base(name), ".sql")
		var exists int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, version).Scan(&exists); err != nil {
			return fmt.Errorf("read migration %s: %w", version, err)
		}
		if exists != 0 {
			continue
		}
		script, err := fs.ReadFile(files, name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", version, err)
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", version, err)
		}
		if _, err = tx.ExecContext(ctx, string(script)); err == nil {
			_, err = tx.ExecContext(ctx, `INSERT INTO schema_migrations(version, applied_at) VALUES(?, ?)`, version, time.Now().UTC().Format(time.RFC3339Nano))
		}
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", version, err)
		}
	}
	return nil
}

func (s *Store) WithTx(ctx context.Context, fn func(Queries) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(&dbQueries{q: tx}); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *Store) queries() *dbQueries { return &dbQueries{q: s.db} }
func (s *Store) Profile(ctx context.Context) (*domain.Profile, error) {
	return s.queries().Profile(ctx)
}
func (s *Store) Accounts(ctx context.Context) ([]domain.Account, error) {
	return s.queries().Accounts(ctx)
}
func (s *Store) Account(ctx context.Context, id string) (*domain.Account, error) {
	return s.queries().Account(ctx, id)
}
func (s *Store) AccountTransactions(ctx context.Context, id string) ([]domain.Transaction, error) {
	return s.queries().AccountTransactions(ctx, id)
}
func (s *Store) CreditCardInvoices(ctx context.Context, id string) ([]domain.CreditCardInvoice, error) {
	return s.queries().CreditCardInvoices(ctx, id)
}
func (s *Store) CreditCardInvoice(ctx context.Context, id string) (*domain.CreditCardInvoice, error) {
	return s.queries().CreditCardInvoice(ctx, id)
}
func (s *Store) Categories(ctx context.Context) ([]domain.Category, error) {
	return s.queries().Categories(ctx)
}
func (s *Store) Category(ctx context.Context, id string) (*domain.Category, error) {
	return s.queries().Category(ctx, id)
}
func (s *Store) Transactions(ctx context.Context) ([]domain.Transaction, error) {
	return s.queries().Transactions(ctx)
}
func (s *Store) TrashedTransactions(ctx context.Context) ([]domain.Transaction, error) {
	return s.queries().TrashedTransactions(ctx)
}
func (s *Store) Transaction(ctx context.Context, id string) (*domain.Transaction, error) {
	return s.queries().Transaction(ctx, id)
}
func (s *Store) FixedExpenses(ctx context.Context) ([]domain.FixedExpense, error) {
	return s.queries().FixedExpenses(ctx)
}
func (s *Store) FixedExpenseOccurrences(ctx context.Context) ([]domain.FixedExpenseOccurrence, error) {
	return s.queries().FixedExpenseOccurrences(ctx)
}
func (s *Store) SaveProfile(ctx context.Context, p domain.Profile, at string) error {
	return s.queries().SaveProfile(ctx, p, at)
}
func (s *Store) SetBalancesHidden(ctx context.Context, hidden bool, at string) error {
	return s.queries().SetBalancesHidden(ctx, hidden, at)
}
func (s *Store) InsertAccount(ctx context.Context, a domain.Account, at string) error {
	return s.queries().InsertAccount(ctx, a, at)
}
func (s *Store) UpdateAccount(ctx context.Context, a domain.Account, at string) error {
	return s.queries().UpdateAccount(ctx, a, at)
}
func (s *Store) DeleteAccount(ctx context.Context, id string) error {
	return s.WithTx(ctx, func(q Queries) error {
		account, err := q.Account(ctx, id)
		if err != nil {
			return err
		}
		if account == nil {
			return domain.ErrUnknownAccount
		}
		linked, err := q.AccountTransactions(ctx, id)
		if err != nil {
			return err
		}
		if len(linked) != 0 {
			return domain.ErrAccountInUse
		}
		return q.DeleteAccount(ctx, id)
	})
}
func (s *Store) InsertTransaction(ctx context.Context, t domain.Transaction, at string) error {
	return s.queries().InsertTransaction(ctx, t, at)
}
func (s *Store) UpdateTransaction(ctx context.Context, t domain.Transaction, at string) error {
	return s.queries().UpdateTransaction(ctx, t, at)
}
func (s *Store) SetTransactionDeletedAt(ctx context.Context, id, deletedAt, at string) error {
	return s.queries().SetTransactionDeletedAt(ctx, id, deletedAt, at)
}
func (s *Store) DeleteTransactionRevisions(ctx context.Context, id string) error {
	return s.queries().DeleteTransactionRevisions(ctx, id)
}
func (s *Store) DeleteTransaction(ctx context.Context, id string) error {
	return s.queries().DeleteTransaction(ctx, id)
}
func (s *Store) InsertTransactionRevision(ctx context.Context, t domain.Transaction, action, at string) error {
	return s.queries().InsertTransactionRevision(ctx, t, action, at)
}

func (q *dbQueries) Profile(ctx context.Context) (*domain.Profile, error) {
	var p domain.Profile
	err := q.q.QueryRowContext(ctx, `SELECT display_name, currency, theme, onboarding_status, balances_hidden FROM profile WHERE id = 1`).Scan(&p.DisplayName, &p.Currency, &p.Theme, &p.OnboardingStatus, &p.BalancesHidden)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &p, err
}

const accountSelect = `SELECT id, name, type, opening_balance_cents, opening_date, created_at,
credit_limit_cents, closing_day, due_day FROM accounts`

func scanAccount(scanner interface{ Scan(...any) error }) (domain.Account, error) {
	var a domain.Account
	err := scanner.Scan(&a.ID, &a.Name, &a.Type, &a.OpeningBalanceCents, &a.OpeningDate, &a.CreatedAt,
		&a.CreditLimitCents, &a.ClosingDay, &a.DueDay)
	return a, err
}

func (q *dbQueries) Accounts(ctx context.Context) ([]domain.Account, error) {
	rows, err := q.q.QueryContext(ctx, accountSelect+` ORDER BY created_at, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Account{}
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, rows.Err()
}

func (q *dbQueries) Account(ctx context.Context, id string) (*domain.Account, error) {
	a, err := scanAccount(q.q.QueryRowContext(ctx, accountSelect+` WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &a, err
}

func (q *dbQueries) AccountTransactions(ctx context.Context, id string) ([]domain.Transaction, error) {
	rows, err := q.q.QueryContext(ctx, transactionSelect+` WHERE t.account_id = ? OR t.destination_account_id = ? ORDER BY t.occurrence_date, t.created_at`, id, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Transaction{}
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, t)
	}
	return items, rows.Err()
}

func (q *dbQueries) Categories(ctx context.Context) ([]domain.Category, error) {
	rows, err := q.q.QueryContext(ctx, `SELECT id, name, kind FROM categories ORDER BY kind DESC, rowid`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Category{}
	for rows.Next() {
		var c domain.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Kind); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

func (q *dbQueries) Category(ctx context.Context, id string) (*domain.Category, error) {
	var c domain.Category
	err := q.q.QueryRowContext(ctx, `SELECT id, name, kind FROM categories WHERE id = ?`, id).Scan(&c.ID, &c.Name, &c.Kind)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &c, err
}

const transactionSelect = `SELECT t.id, t.kind, t.amount_cents, t.account_id, a.name,
COALESCE(t.destination_account_id, ''), COALESCE(destination.name, ''),
COALESCE(t.category_id, ''), COALESCE(c.name, ''), t.description,
t.occurrence_date, t.created_at, t.updated_at, COALESCE(t.deleted_at, ''),
COALESCE(t.fixed_expense_occurrence_id, ''), t.automatic_import,
COALESCE(t.import_bank, ''), COALESCE(t.import_key, ''), t.installment_count,
COALESCE(t.invoice_payment_id, '')
FROM transactions t
JOIN accounts a ON a.id=t.account_id
LEFT JOIN accounts destination ON destination.id=t.destination_account_id
LEFT JOIN categories c ON c.id=t.category_id`

func scanTransaction(scanner interface{ Scan(...any) error }) (domain.Transaction, error) {
	var t domain.Transaction
	err := scanner.Scan(&t.ID, &t.Kind, &t.AmountCents, &t.AccountID, &t.AccountName,
		&t.DestinationAccountID, &t.DestinationAccountName, &t.CategoryID, &t.CategoryName,
		&t.Description, &t.OccurrenceDate, &t.CreatedAt, &t.UpdatedAt, &t.DeletedAt, &t.FixedExpenseOccurrenceID,
		&t.AutomaticImport, &t.ImportBank, &t.ImportKey, &t.InstallmentCount, &t.InvoicePaymentID)
	return t, err
}

func (q *dbQueries) Transactions(ctx context.Context) ([]domain.Transaction, error) {
	rows, err := q.q.QueryContext(ctx, transactionSelect+` WHERE t.deleted_at IS NULL ORDER BY t.occurrence_date DESC, t.created_at DESC`)
	return scanTransactions(rows, err)
}

func (q *dbQueries) TrashedTransactions(ctx context.Context) ([]domain.Transaction, error) {
	rows, err := q.q.QueryContext(ctx, transactionSelect+` WHERE t.deleted_at IS NOT NULL ORDER BY t.deleted_at DESC, t.occurrence_date DESC, t.created_at DESC`)
	return scanTransactions(rows, err)
}

func scanTransactions(rows *sql.Rows, err error) ([]domain.Transaction, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Transaction{}
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, t)
	}
	return items, rows.Err()
}

func (q *dbQueries) Transaction(ctx context.Context, id string) (*domain.Transaction, error) {
	t, err := scanTransaction(q.q.QueryRowContext(ctx, transactionSelect+` WHERE t.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &t, err
}

func (q *dbQueries) SaveProfile(ctx context.Context, p domain.Profile, at string) error {
	_, err := q.q.ExecContext(ctx, `INSERT INTO profile(id, display_name, currency, theme, onboarding_status, balances_hidden, created_at, updated_at) VALUES(1,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET display_name=excluded.display_name, currency=excluded.currency, theme=excluded.theme, onboarding_status=excluded.onboarding_status, balances_hidden=excluded.balances_hidden, updated_at=excluded.updated_at`, p.DisplayName, p.Currency, p.Theme, p.OnboardingStatus, p.BalancesHidden, at, at)
	return err
}

func (q *dbQueries) SetBalancesHidden(ctx context.Context, hidden bool, at string) error {
	result, err := q.q.ExecContext(ctx, `UPDATE profile SET balances_hidden=?, updated_at=? WHERE id=1`, hidden, at)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err == nil && affected == 0 {
		return sql.ErrNoRows
	}
	return err
}

func (q *dbQueries) InsertAccount(ctx context.Context, a domain.Account, at string) error {
	_, err := q.q.ExecContext(ctx, `INSERT INTO accounts(id,name,type,opening_balance_cents,opening_date,created_at,updated_at,credit_limit_cents,closing_day,due_day) VALUES(?,?,?,?,?,?,?,?,?,?)`, a.ID, a.Name, a.Type, a.OpeningBalanceCents, a.OpeningDate, at, at, a.CreditLimitCents, a.ClosingDay, a.DueDay)
	return err
}

func (q *dbQueries) UpdateAccount(ctx context.Context, a domain.Account, at string) error {
	result, err := q.q.ExecContext(ctx, `UPDATE accounts SET name=?, type=?, opening_balance_cents=?, opening_date=?, credit_limit_cents=?, closing_day=?, due_day=?, updated_at=? WHERE id=?`, a.Name, a.Type, a.OpeningBalanceCents, a.OpeningDate, a.CreditLimitCents, a.ClosingDay, a.DueDay, at, a.ID)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count == 0 {
		return domain.ErrUnknownAccount
	}
	return err
}

func (q *dbQueries) DeleteAccount(ctx context.Context, id string) error {
	result, err := q.q.ExecContext(ctx, `DELETE FROM accounts WHERE id=?`, id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count == 0 {
		return domain.ErrUnknownAccount
	}
	return err
}

func (q *dbQueries) InsertTransaction(ctx context.Context, t domain.Transaction, at string) error {
	var category any
	if t.CategoryID != "" {
		category = t.CategoryID
	}
	var destination any
	if t.DestinationAccountID != "" {
		destination = t.DestinationAccountID
	}
	var fixedOccurrence any
	if t.FixedExpenseOccurrenceID != "" {
		fixedOccurrence = t.FixedExpenseOccurrenceID
	}
	var importBank, importKey any
	if t.ImportBank != "" {
		importBank = t.ImportBank
	}
	if t.ImportKey != "" {
		importKey = t.ImportKey
	}
	count := t.InstallmentCount
	if count == 0 {
		count = 1
	}
	var payment any
	if t.InvoicePaymentID != "" {
		payment = t.InvoicePaymentID
	}
	_, err := q.q.ExecContext(ctx, `INSERT INTO transactions(id,kind,amount_cents,account_id,destination_account_id,category_id,description,occurrence_date,created_at,updated_at,fixed_expense_occurrence_id,automatic_import,import_bank,import_key,installment_count,invoice_payment_id) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, t.ID, t.Kind, t.AmountCents, t.AccountID, destination, category, t.Description, t.OccurrenceDate, at, at, fixedOccurrence, t.AutomaticImport, importBank, importKey, count, payment)
	return err
}

func (q *dbQueries) UpdateTransaction(ctx context.Context, t domain.Transaction, at string) error {
	var category, destination any
	if t.CategoryID != "" {
		category = t.CategoryID
	}
	if t.DestinationAccountID != "" {
		destination = t.DestinationAccountID
	}
	count := t.InstallmentCount
	if count == 0 {
		count = 1
	}
	result, err := q.q.ExecContext(ctx, `UPDATE transactions SET kind=?, amount_cents=?, account_id=?, destination_account_id=?, category_id=?, description=?, occurrence_date=?, installment_count=?, updated_at=? WHERE id=?`, t.Kind, t.AmountCents, t.AccountID, destination, category, t.Description, t.OccurrenceDate, count, at, t.ID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err == nil && affected == 0 {
		return domain.ErrUnknownTransaction
	}
	return err
}

func (q *dbQueries) SetTransactionDeletedAt(ctx context.Context, id, deletedAt, at string) error {
	var value any
	if deletedAt != "" {
		value = deletedAt
	}
	result, err := q.q.ExecContext(ctx, `UPDATE transactions SET deleted_at=?, updated_at=? WHERE id=?`, value, at, id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count == 0 {
		return domain.ErrUnknownTransaction
	}
	return err
}

func (q *dbQueries) DeleteTransactionRevisions(ctx context.Context, id string) error {
	_, err := q.q.ExecContext(ctx, `DELETE FROM transaction_revisions WHERE transaction_id=?`, id)
	return err
}

func (q *dbQueries) DeleteTransaction(ctx context.Context, id string) error {
	result, err := q.q.ExecContext(ctx, `DELETE FROM transactions WHERE id=?`, id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count == 0 {
		return domain.ErrUnknownTransaction
	}
	return err
}

func (q *dbQueries) InsertTransactionRevision(ctx context.Context, t domain.Transaction, action, at string) error {
	snapshot, err := json.Marshal(t)
	if err != nil {
		return err
	}
	_, err = q.q.ExecContext(ctx, `INSERT INTO transaction_revisions(transaction_id, action, snapshot_json, created_at) VALUES(?,?,?,?)`, t.ID, action, string(snapshot), at)
	return err
}

const fixedExpenseSelect = `SELECT f.id, f.description, f.amount_cents, f.due_day,
f.account_id, a.name, f.category_id, c.name, COALESCE(f.archived_at, ''), f.created_at, f.updated_at
FROM fixed_expenses f
JOIN accounts a ON a.id=f.account_id
JOIN categories c ON c.id=f.category_id`

func scanFixedExpense(scanner interface{ Scan(...any) error }) (domain.FixedExpense, error) {
	var expense domain.FixedExpense
	err := scanner.Scan(&expense.ID, &expense.Description, &expense.AmountCents, &expense.DueDay,
		&expense.AccountID, &expense.AccountName, &expense.CategoryID, &expense.CategoryName,
		&expense.ArchivedAt, &expense.CreatedAt, &expense.UpdatedAt)
	return expense, err
}

func (q *dbQueries) FixedExpenses(ctx context.Context) ([]domain.FixedExpense, error) {
	rows, err := q.q.QueryContext(ctx, fixedExpenseSelect+` ORDER BY f.archived_at IS NOT NULL, f.due_day, f.description`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.FixedExpense{}
	for rows.Next() {
		item, err := scanFixedExpense(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (q *dbQueries) FixedExpense(ctx context.Context, id string) (*domain.FixedExpense, error) {
	item, err := scanFixedExpense(q.q.QueryRowContext(ctx, fixedExpenseSelect+` WHERE f.id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &item, err
}

const fixedOccurrenceSelect = `SELECT o.id, o.fixed_expense_id, o.reference_month, o.due_date, o.description,
o.expected_amount_cents, o.account_id, a.name, o.category_id, c.name, o.status,
COALESCE(o.transaction_id, ''), o.created_at, o.updated_at
FROM fixed_expense_occurrences o
JOIN accounts a ON a.id=o.account_id
JOIN categories c ON c.id=o.category_id`

func scanFixedExpenseOccurrence(scanner interface{ Scan(...any) error }) (domain.FixedExpenseOccurrence, error) {
	var occurrence domain.FixedExpenseOccurrence
	err := scanner.Scan(&occurrence.ID, &occurrence.FixedExpenseID, &occurrence.ReferenceMonth, &occurrence.DueDate,
		&occurrence.Description, &occurrence.ExpectedAmountCents, &occurrence.AccountID, &occurrence.AccountName,
		&occurrence.CategoryID, &occurrence.CategoryName, &occurrence.Status, &occurrence.TransactionID,
		&occurrence.CreatedAt, &occurrence.UpdatedAt)
	return occurrence, err
}

func (q *dbQueries) FixedExpenseOccurrences(ctx context.Context) ([]domain.FixedExpenseOccurrence, error) {
	rows, err := q.q.QueryContext(ctx, fixedOccurrenceSelect+` ORDER BY CASE o.status WHEN 'pending' THEN 0 WHEN 'confirmed' THEN 1 ELSE 2 END, o.due_date, o.description`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.FixedExpenseOccurrence{}
	for rows.Next() {
		item, err := scanFixedExpenseOccurrence(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (q *dbQueries) FixedExpenseOccurrence(ctx context.Context, id string) (*domain.FixedExpenseOccurrence, error) {
	item, err := scanFixedExpenseOccurrence(q.q.QueryRowContext(ctx, fixedOccurrenceSelect+` WHERE o.id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &item, err
}

func (q *dbQueries) OccurrenceForTransaction(ctx context.Context, transactionID string) (*domain.FixedExpenseOccurrence, error) {
	item, err := scanFixedExpenseOccurrence(q.q.QueryRowContext(ctx, fixedOccurrenceSelect+` WHERE o.transaction_id=?`, transactionID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &item, err
}

func (q *dbQueries) InsertFixedExpense(ctx context.Context, expense domain.FixedExpense, at string) error {
	_, err := q.q.ExecContext(ctx, `INSERT INTO fixed_expenses(id,description,amount_cents,due_day,account_id,category_id,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`, expense.ID, expense.Description, expense.AmountCents, expense.DueDay, expense.AccountID, expense.CategoryID, at, at)
	return err
}

func (q *dbQueries) UpdateFixedExpense(ctx context.Context, expense domain.FixedExpense, at string) error {
	result, err := q.q.ExecContext(ctx, `UPDATE fixed_expenses SET description=?, amount_cents=?, due_day=?, account_id=?, category_id=?, updated_at=? WHERE id=? AND archived_at IS NULL`, expense.Description, expense.AmountCents, expense.DueDay, expense.AccountID, expense.CategoryID, at, expense.ID)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count == 0 {
		return domain.ErrUnknownFixedExpense
	}
	return err
}

func (q *dbQueries) SetFixedExpenseArchivedAt(ctx context.Context, id, archivedAt, at string) error {
	var value any
	if archivedAt != "" {
		value = archivedAt
	}
	result, err := q.q.ExecContext(ctx, `UPDATE fixed_expenses SET archived_at=?, updated_at=? WHERE id=?`, value, at, id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count == 0 {
		return domain.ErrUnknownFixedExpense
	}
	return err
}

func (q *dbQueries) InsertFixedExpenseOccurrence(ctx context.Context, occurrence domain.FixedExpenseOccurrence, at string) error {
	_, err := q.q.ExecContext(ctx, `INSERT INTO fixed_expense_occurrences(id,fixed_expense_id,reference_month,due_date,description,expected_amount_cents,account_id,category_id,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, occurrence.ID, occurrence.FixedExpenseID, occurrence.ReferenceMonth, occurrence.DueDate, occurrence.Description, occurrence.ExpectedAmountCents, occurrence.AccountID, occurrence.CategoryID, occurrence.Status, at, at)
	return err
}

func (q *dbQueries) SetFixedExpenseOccurrence(ctx context.Context, id string, status domain.FixedExpenseOccurrenceStatus, transactionID, at string) error {
	var transaction any
	if transactionID != "" {
		transaction = transactionID
	}
	result, err := q.q.ExecContext(ctx, `UPDATE fixed_expense_occurrences SET status=?, transaction_id=?, updated_at=? WHERE id=?`, status, transaction, at, id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count == 0 {
		return domain.ErrUnknownOccurrence
	}
	return err
}

const creditCardInvoiceSelect = `SELECT i.id, i.account_id, a.name, i.reference_month,
i.closing_date, i.due_date, i.status, i.carry_forward_cents
FROM credit_card_invoices i JOIN accounts a ON a.id=i.account_id`

func scanCreditCardInvoice(scanner interface{ Scan(...any) error }) (domain.CreditCardInvoice, error) {
	var invoice domain.CreditCardInvoice
	err := scanner.Scan(&invoice.ID, &invoice.AccountID, &invoice.AccountName, &invoice.ReferenceMonth,
		&invoice.ClosingDate, &invoice.DueDate, &invoice.Status, &invoice.CarryForwardCents)
	invoice.Installments = []domain.CreditCardInstallment{}
	invoice.Payments = []domain.CreditCardPayment{}
	return invoice, err
}

func (q *dbQueries) CreditCardInvoices(ctx context.Context, accountID string) ([]domain.CreditCardInvoice, error) {
	rows, err := q.q.QueryContext(ctx, creditCardInvoiceSelect+` WHERE i.account_id=? ORDER BY i.closing_date`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.CreditCardInvoice{}
	for rows.Next() {
		item, err := scanCreditCardInvoice(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (q *dbQueries) CreditCardInvoice(ctx context.Context, id string) (*domain.CreditCardInvoice, error) {
	item, err := scanCreditCardInvoice(q.q.QueryRowContext(ctx, creditCardInvoiceSelect+` WHERE i.id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &item, err
}

func (q *dbQueries) InvoiceInstallments(ctx context.Context, invoiceID string) ([]domain.CreditCardInstallment, error) {
	rows, err := q.q.QueryContext(ctx, `SELECT x.id, x.invoice_id, COALESCE(x.transaction_id,''), x.description,
x.amount_cents, x.installment_number, x.installment_count, x.opening_debt
FROM credit_card_installments x LEFT JOIN transactions t ON t.id=x.transaction_id
WHERE x.invoice_id=? AND (x.transaction_id IS NULL OR t.deleted_at IS NULL)
ORDER BY x.installment_number, x.created_at`, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInstallments(rows)
}

func (q *dbQueries) TransactionInstallments(ctx context.Context, transactionID string) ([]domain.CreditCardInstallment, error) {
	return q.installments(ctx, ` WHERE x.transaction_id=? ORDER BY x.installment_number`, transactionID)
}

func (q *dbQueries) installments(ctx context.Context, where string, value string) ([]domain.CreditCardInstallment, error) {
	rows, err := q.q.QueryContext(ctx, `SELECT x.id, x.invoice_id, COALESCE(x.transaction_id,''), x.description,
x.amount_cents, x.installment_number, x.installment_count, x.opening_debt
FROM credit_card_installments x`+where, value)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInstallments(rows)
}

func scanInstallments(rows *sql.Rows) ([]domain.CreditCardInstallment, error) {
	items := []domain.CreditCardInstallment{}
	for rows.Next() {
		var item domain.CreditCardInstallment
		if err := rows.Scan(&item.ID, &item.InvoiceID, &item.TransactionID, &item.Description, &item.AmountCents,
			&item.InstallmentNumber, &item.InstallmentCount, &item.OpeningDebt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (q *dbQueries) InvoicePayments(ctx context.Context, invoiceID string) ([]domain.CreditCardPayment, error) {
	rows, err := q.q.QueryContext(ctx, `SELECT p.id, p.invoice_id, p.source_account_id, a.name,
p.transaction_id, p.amount_cents, p.occurrence_date, p.created_at
FROM credit_card_payments p JOIN accounts a ON a.id=p.source_account_id
WHERE p.invoice_id=? ORDER BY p.occurrence_date, p.created_at`, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.CreditCardPayment{}
	for rows.Next() {
		var item domain.CreditCardPayment
		if err := rows.Scan(&item.ID, &item.InvoiceID, &item.AccountID, &item.AccountName, &item.TransactionID,
			&item.AmountCents, &item.OccurrenceDate, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (q *dbQueries) InsertCreditCardInvoice(ctx context.Context, invoice domain.CreditCardInvoice, at string) error {
	_, err := q.q.ExecContext(ctx, `INSERT INTO credit_card_invoices(id,account_id,reference_month,closing_date,due_date,status,carry_forward_cents,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		invoice.ID, invoice.AccountID, invoice.ReferenceMonth, invoice.ClosingDate, invoice.DueDate, invoice.Status, invoice.CarryForwardCents, at, at)
	return err
}

func (q *dbQueries) UpdateCreditCardInvoice(ctx context.Context, status domain.CreditCardInvoiceStatus, carry int64, id, at string) error {
	result, err := q.q.ExecContext(ctx, `UPDATE credit_card_invoices SET status=?, carry_forward_cents=?, updated_at=? WHERE id=?`, status, carry, at, id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count == 0 {
		return domain.ErrUnknownInvoice
	}
	return err
}

func (q *dbQueries) InsertCreditCardInstallment(ctx context.Context, item domain.CreditCardInstallment, at string) error {
	var transaction any
	if item.TransactionID != "" {
		transaction = item.TransactionID
	}
	_, err := q.q.ExecContext(ctx, `INSERT INTO credit_card_installments(id,invoice_id,transaction_id,description,amount_cents,installment_number,installment_count,opening_debt,created_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		item.ID, item.InvoiceID, transaction, item.Description, item.AmountCents, item.InstallmentNumber, item.InstallmentCount, item.OpeningDebt, at)
	return err
}

func (q *dbQueries) DeleteTransactionInstallments(ctx context.Context, transactionID string) error {
	_, err := q.q.ExecContext(ctx, `DELETE FROM credit_card_installments WHERE transaction_id=?`, transactionID)
	return err
}

func (q *dbQueries) UpdateTransactionInstallmentDescriptions(ctx context.Context, transactionID, description string) error {
	_, err := q.q.ExecContext(ctx, `UPDATE credit_card_installments SET description=? WHERE transaction_id=?`, description, transactionID)
	return err
}

func (q *dbQueries) InsertCreditCardPayment(ctx context.Context, payment domain.CreditCardPayment) error {
	_, err := q.q.ExecContext(ctx, `INSERT INTO credit_card_payments(id,invoice_id,source_account_id,transaction_id,amount_cents,occurrence_date,created_at) VALUES(?,?,?,?,?,?,?)`,
		payment.ID, payment.InvoiceID, payment.AccountID, payment.TransactionID, payment.AmountCents, payment.OccurrenceDate, payment.CreatedAt)
	return err
}
