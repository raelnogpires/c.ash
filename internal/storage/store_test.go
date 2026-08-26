package storage

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"testing/fstest"

	"c.ash/internal/domain"
)

func TestOpen_MigratesReopensAndLocks(t *testing.T) {
	ctx := context.Background()
	path := t.TempDir() + "/cash.db"
	first, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	var migrations int
	if err := first.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&migrations); err != nil || migrations != latestSchemaVersion() {
		t.Fatalf("migrations=%d err=%v", migrations, err)
	}
	categories, err := first.Categories(ctx)
	if err != nil || len(categories) != 22 {
		t.Fatalf("categories=%d err=%v", len(categories), err)
	}
	if _, err := Open(ctx, path); !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("second open error=%v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if err := ApplyMigrations(ctx, reopened.db, migrationFiles); err != nil {
		t.Fatalf("idempotent migration: %v", err)
	}
}

func TestMigration_FixedExpenseOccurrenceCursorStartsAtCreatedAt(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite3", "file:"+t.TempDir()+"/fixed-expense-upgrade.db?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	legacy := fstest.MapFS{}
	for _, name := range []string{
		"migrations/001_initial.sql",
		"migrations/002_transactions.sql",
		"migrations/003_fixed_expenses.sql",
		"migrations/004_pdf_imports.sql",
		"migrations/005_credit_cards.sql",
	} {
		legacy[name] = &fstest.MapFile{Data: mustReadMigration(t, name)}
	}
	if err := ApplyMigrations(ctx, db, legacy); err != nil {
		t.Fatal(err)
	}
	const createdAt = "2026-08-10T12:00:00Z"
	_, err = db.Exec(`INSERT INTO accounts(id,name,type,opening_balance_cents,opening_date,created_at,updated_at)
		VALUES('a','Principal','checking',1000,'2026-08-01',?,?)`, createdAt, createdAt)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO fixed_expenses(id,description,amount_cents,due_day,account_id,category_id,created_at,updated_at)
		VALUES('f','Internet',100,10,'a','bills',?,?)`, createdAt, createdAt)
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyMigrations(ctx, db, migrationFiles); err != nil {
		t.Fatal(err)
	}
	var occurrenceStartAt string
	if err := db.QueryRow(`SELECT occurrence_start_at FROM fixed_expenses WHERE id='f'`).Scan(&occurrenceStartAt); err != nil {
		t.Fatal(err)
	}
	if occurrenceStartAt != createdAt {
		t.Fatalf("occurrence_start_at=%q", occurrenceStartAt)
	}
}

func TestMigration_UpgradesExistingDataWithoutLoss(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite3", "file:"+t.TempDir()+"/upgrade.db?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	legacy := fstest.MapFS{"migrations/001_initial.sql": {Data: mustReadMigration(t, "migrations/001_initial.sql")}}
	if err := ApplyMigrations(ctx, db, legacy); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO accounts(id,name,type,opening_balance_cents,opening_date,created_at,updated_at) VALUES('a','Principal','checking',1000,'2026-08-01','x','x'); INSERT INTO transactions(id,kind,amount_cents,account_id,category_id,description,occurrence_date,created_at,updated_at) VALUES('t','expense',100,'a','food','Mercado','2026-08-02','x','x')`)
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyMigrations(ctx, db, migrationFiles); err != nil {
		t.Fatal(err)
	}
	var accounts, transactions, categories int
	if err := db.QueryRow(`SELECT (SELECT COUNT(*) FROM accounts),(SELECT COUNT(*) FROM transactions),(SELECT COUNT(*) FROM categories)`).Scan(&accounts, &transactions, &categories); err != nil {
		t.Fatal(err)
	}
	if accounts != 1 || transactions != 1 || categories != 22 {
		t.Fatalf("counts=%d/%d/%d", accounts, transactions, categories)
	}
	var foreignKeyErrors int
	rows, err := db.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		foreignKeyErrors++
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	if foreignKeyErrors != 0 {
		t.Fatalf("foreign key errors=%d", foreignKeyErrors)
	}
	if _, err := db.Exec(`INSERT INTO accounts(id,name,type,opening_balance_cents,opening_date,created_at,updated_at) VALUES('s','Reserva','savings',0,'2026-08-01','x','x')`); err != nil {
		t.Fatalf("savings constraint not upgraded: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO accounts(id,name,type,opening_balance_cents,opening_date,created_at,updated_at,credit_limit_cents,closing_day,due_day) VALUES('card','Cartão','credit_card',0,'2026-08-01','x','x',10000,25,2)`); err != nil {
		t.Fatalf("credit-card constraint not upgraded: %v", err)
	}
}

func TestOpen_CreatesPreMigrationBackup(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "cash.db")
	db, err := sql.Open("sqlite3", "file:"+path+"?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	legacy := fstest.MapFS{"migrations/001_initial.sql": {Data: mustReadMigration(t, "migrations/001_initial.sql")}}
	if err := ApplyMigrations(ctx, db, legacy); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	store, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	backups, err := filepath.Glob(filepath.Join(dir, "backups", "cash-pre-migration-*.cashbackup"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("backups=%v err=%v", backups, err)
	}
	info, err := store.InspectBackup(backups[0])
	if err != nil || info.Manifest.SchemaVersion != 1 || info.Manifest.Kind != BackupKindPreMigration {
		t.Fatalf("info=%+v err=%v", info, err)
	}
}

func mustReadMigration(t *testing.T, name string) []byte {
	t.Helper()
	data, err := migrationFiles.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestStore_ForeignKeysAndPersistence(t *testing.T) {
	ctx := context.Background()
	path := t.TempDir() + "/cash.db"
	store, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	account := domain.Account{ID: "account", Name: "Conta", Type: domain.AccountChecking, OpeningDate: "2026-08-01", CreatedAt: "2026-08-01T00:00:00Z"}
	if err := store.InsertAccount(ctx, account, account.CreatedAt); err != nil {
		t.Fatal(err)
	}
	bad := domain.Transaction{ID: "bad", Kind: domain.Expense, AmountCents: 1, AccountID: "missing", Description: "Teste", OccurrenceDate: "2026-08-01", CreatedAt: "2026-08-01T00:00:00Z"}
	if err := store.InsertTransaction(ctx, bad, bad.CreatedAt); err == nil {
		t.Fatal("foreign key accepted unknown account")
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	accounts, err := store.Accounts(ctx)
	if err != nil || len(accounts) != 1 {
		t.Fatalf("persisted accounts=%d err=%v", len(accounts), err)
	}
}

func TestStore_TransactionSoftDeleteAndRevisionSnapshots(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, t.TempDir()+"/cash.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	account := domain.Account{ID: "a", Name: "Conta", Type: domain.AccountChecking, OpeningDate: "2026-08-01", CreatedAt: "x"}
	if err := store.InsertAccount(ctx, account, "x"); err != nil {
		t.Fatal(err)
	}
	tx := domain.Transaction{ID: "t", Kind: domain.Income, AmountCents: 100, AccountID: "a", AccountName: "Conta", Description: "Receita", OccurrenceDate: "2026-08-02", CreatedAt: "x", UpdatedAt: "x"}
	if err := store.WithTx(ctx, func(q Queries) error {
		if err := q.InsertTransaction(ctx, tx, "x"); err != nil {
			return err
		}
		return q.InsertTransactionRevision(ctx, tx, "create", "x")
	}); err != nil {
		t.Fatal(err)
	}
	tx.AmountCents, tx.UpdatedAt = 125, "edited"
	if err := store.WithTx(ctx, func(q Queries) error {
		if err := q.UpdateTransaction(ctx, tx, "edited"); err != nil {
			return err
		}
		return q.InsertTransactionRevision(ctx, tx, "update", "edited")
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.WithTx(ctx, func(q Queries) error {
		if err := q.SetTransactionDeletedAt(ctx, "t", "deleted", "updated"); err != nil {
			return err
		}
		tx.DeletedAt = "deleted"
		return q.InsertTransactionRevision(ctx, tx, "trash", "updated")
	}); err != nil {
		t.Fatal(err)
	}
	if items, _ := store.Transactions(ctx); len(items) != 0 {
		t.Fatalf("deleted items=%v", items)
	}
	stored, err := store.Transaction(ctx, "t")
	if err != nil || stored.DeletedAt != "deleted" {
		t.Fatalf("stored=%+v err=%v", stored, err)
	}
	stored.DeletedAt, stored.UpdatedAt = "", "restored"
	if err := store.WithTx(ctx, func(q Queries) error {
		if err := q.SetTransactionDeletedAt(ctx, "t", "", "restored"); err != nil {
			return err
		}
		return q.InsertTransactionRevision(ctx, *stored, "restore", "restored")
	}); err != nil {
		t.Fatal(err)
	}
	var revisions int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM transaction_revisions WHERE transaction_id='t' AND json_valid(snapshot_json) AND action IN ('create','update','trash','restore')`).Scan(&revisions); err != nil {
		t.Fatal(err)
	}
	if revisions != 4 {
		t.Fatalf("revisions=%d", revisions)
	}
	if items, _ := store.Transactions(ctx); len(items) != 1 || items[0].AmountCents != 125 {
		t.Fatalf("restored items=%+v", items)
	}
}

func TestStore_TrashedTransactionsPersistAndPermanentDeleteCleansDependencies(t *testing.T) {
	ctx := context.Background()
	path := t.TempDir() + "/cash.db"
	store, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	card := domain.Account{ID: "card", Name: "Cartão", Type: domain.AccountCreditCard, OpeningDate: "2026-08-01", CreatedAt: "created", CreditLimitCents: 100000, ClosingDay: 25, DueDay: 2}
	if err := store.InsertAccount(ctx, card, "created"); err != nil {
		t.Fatal(err)
	}
	invoice := domain.CreditCardInvoice{ID: "invoice", AccountID: card.ID, ReferenceMonth: "2026-09", ClosingDate: "2026-08-25", DueDate: "2026-09-02", Status: domain.InvoiceOpen}
	older := domain.Transaction{ID: "older", Kind: domain.Expense, AmountCents: 3000, AccountID: card.ID, Description: "Compra antiga", OccurrenceDate: "2026-08-10", CreatedAt: "created", UpdatedAt: "created"}
	newer := domain.Transaction{ID: "newer", Kind: domain.Expense, AmountCents: 2000, AccountID: card.ID, Description: "Compra recente", OccurrenceDate: "2026-08-11", CreatedAt: "created", UpdatedAt: "created"}
	if err := store.WithTx(ctx, func(q Queries) error {
		if err := q.InsertCreditCardInvoice(ctx, invoice, "created"); err != nil {
			return err
		}
		for _, transaction := range []domain.Transaction{older, newer} {
			if err := q.InsertTransaction(ctx, transaction, "created"); err != nil {
				return err
			}
			if err := q.InsertTransactionRevision(ctx, transaction, "create", "created"); err != nil {
				return err
			}
		}
		if err := q.InsertCreditCardInstallment(ctx, domain.CreditCardInstallment{ID: "installment", InvoiceID: invoice.ID, TransactionID: older.ID, Description: older.Description, AmountCents: older.AmountCents, InstallmentNumber: 1, InstallmentCount: 1}, "created"); err != nil {
			return err
		}
		if err := q.SetTransactionDeletedAt(ctx, older.ID, "2026-08-16T10:00:00Z", "2026-08-16T10:00:00Z"); err != nil {
			return err
		}
		if err := q.SetTransactionDeletedAt(ctx, newer.ID, "2026-08-16T11:00:00Z", "2026-08-16T11:00:00Z"); err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	trashed, err := store.TrashedTransactions(ctx)
	if err != nil || len(trashed) != 2 || trashed[0].ID != newer.ID || trashed[1].ID != older.ID {
		t.Fatalf("trashed=%+v err=%v", trashed, err)
	}
	if err := store.WithTx(ctx, func(q Queries) error {
		if err := q.DeleteTransactionInstallments(ctx, older.ID); err != nil {
			return err
		}
		if err := q.DeleteTransactionRevisions(ctx, older.ID); err != nil {
			return err
		}
		return q.DeleteTransaction(ctx, older.ID)
	}); err != nil {
		t.Fatal(err)
	}
	var transactions, revisions, installments int
	if err := store.db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM transactions WHERE id='older'),
		(SELECT COUNT(*) FROM transaction_revisions WHERE transaction_id='older'),
		(SELECT COUNT(*) FROM credit_card_installments WHERE transaction_id='older')`).Scan(&transactions, &revisions, &installments); err != nil {
		t.Fatal(err)
	}
	if transactions != 0 || revisions != 0 || installments != 0 {
		t.Fatalf("remaining dependencies=%d/%d/%d", transactions, revisions, installments)
	}
}

func TestStore_FailedRevisionRollsBackTransaction(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, t.TempDir()+"/cash.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	account := domain.Account{ID: "a", Name: "Conta", Type: domain.AccountChecking, OpeningDate: "2026-08-01", CreatedAt: "x"}
	if err := store.InsertAccount(ctx, account, "x"); err != nil {
		t.Fatal(err)
	}
	tx := domain.Transaction{ID: "rollback", Kind: domain.Income, AmountCents: 100, AccountID: "a", Description: "Receita", OccurrenceDate: "2026-08-02", CreatedAt: "x"}
	err = store.WithTx(ctx, func(q Queries) error {
		if err := q.InsertTransaction(ctx, tx, "x"); err != nil {
			return err
		}
		return q.InsertTransactionRevision(ctx, tx, "invalid-action", "x")
	})
	if err == nil {
		t.Fatal("invalid revision unexpectedly committed")
	}
	if stored, err := store.Transaction(ctx, tx.ID); err != nil || stored != nil {
		t.Fatalf("rolled back transaction=%+v err=%v", stored, err)
	}
}

func TestStore_UpdateAndDeleteAccount(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, t.TempDir()+"/cash.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	account := domain.Account{ID: "empty", Name: "Antiga", Type: domain.AccountChecking, OpeningBalanceCents: 100, OpeningDate: "2026-08-01", CreatedAt: "created"}
	if err := store.InsertAccount(ctx, account, account.CreatedAt); err != nil {
		t.Fatal(err)
	}
	account.Name, account.Type, account.OpeningBalanceCents = "Reserva", domain.AccountSavings, 250
	if err := store.UpdateAccount(ctx, account, "updated"); err != nil {
		t.Fatal(err)
	}
	stored, err := store.Account(ctx, account.ID)
	if err != nil || stored == nil || stored.Name != "Reserva" || stored.Type != domain.AccountSavings || stored.OpeningBalanceCents != 250 || stored.CreatedAt != "created" {
		t.Fatalf("stored=%+v err=%v", stored, err)
	}
	if err := store.DeleteAccount(ctx, account.ID); err != nil {
		t.Fatal(err)
	}
	if stored, err := store.Account(ctx, account.ID); err != nil || stored != nil {
		t.Fatalf("deleted account=%+v err=%v", stored, err)
	}
}

func TestStore_DeleteAccountRejectsEveryTransactionLink(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, t.TempDir()+"/cash.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	for _, id := range []string{"active", "trashed", "source", "destination"} {
		if err := store.InsertAccount(ctx, domain.Account{ID: id, Name: id, Type: domain.AccountChecking, OpeningDate: "2026-08-01", CreatedAt: "x"}, "x"); err != nil {
			t.Fatal(err)
		}
	}
	txs := []domain.Transaction{
		{ID: "active-tx", Kind: domain.Income, AmountCents: 1, AccountID: "active", Description: "Ativa", OccurrenceDate: "2026-08-01"},
		{ID: "trashed-tx", Kind: domain.Income, AmountCents: 1, AccountID: "trashed", Description: "Lixeira", OccurrenceDate: "2026-08-01"},
		{ID: "transfer-tx", Kind: domain.Transfer, AmountCents: 1, AccountID: "source", DestinationAccountID: "destination", Description: "Transferência", OccurrenceDate: "2026-08-01"},
	}
	for _, tx := range txs {
		if err := store.InsertTransaction(ctx, tx, "x"); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.SetTransactionDeletedAt(ctx, "trashed-tx", "deleted", "deleted"); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"active", "trashed", "source", "destination"} {
		if err := store.DeleteAccount(ctx, id); !errors.Is(err, domain.ErrAccountInUse) {
			t.Fatalf("linked account %q error=%v", id, err)
		}
		if account, err := store.Account(ctx, id); err != nil || account == nil {
			t.Fatalf("account %q changed after failure: %+v err=%v", id, account, err)
		}
	}
}

func TestApplyMigrations_RollsBackFailedMigration(t *testing.T) {
	db, err := sql.Open("sqlite3", "file:"+t.TempDir()+"/rollback.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	files := fstest.MapFS{"migrations/001_bad.sql": {Data: []byte(`CREATE TABLE should_rollback(id INTEGER); INSERT INTO missing_table VALUES(1);`)}}
	if err := ApplyMigrations(context.Background(), db, files); err == nil {
		t.Fatal("bad migration succeeded")
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='should_rollback'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("migration was not rolled back")
	}
}
