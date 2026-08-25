// Package application coordinates the first vertical slice use cases.
package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"c.ash/internal/domain"
	"c.ash/internal/importer"
	"c.ash/internal/storage"
)

type Clock func() time.Time

type AccountInput struct {
	Name                string             `json:"name"`
	Type                domain.AccountType `json:"type"`
	OpeningBalanceCents int64              `json:"openingBalanceCents"`
	OpeningDate         string             `json:"openingDate"`
}

type OnboardingInput struct {
	DisplayName  string       `json:"displayName"`
	Currency     string       `json:"currency"`
	Theme        domain.Theme `json:"theme"`
	FirstAccount AccountInput `json:"firstAccount"`
}

type TransactionInput struct {
	Kind                 domain.TransactionKind `json:"kind"`
	AmountCents          int64                  `json:"amountCents"`
	AccountID            string                 `json:"accountId"`
	DestinationAccountID string                 `json:"destinationAccountId"`
	CategoryID           string                 `json:"categoryId"`
	Description          string                 `json:"description"`
	OccurrenceDate       string                 `json:"occurrenceDate"`
}

type FixedExpenseInput struct {
	Description string `json:"description"`
	AmountCents int64  `json:"amountCents"`
	DueDay      int    `json:"dueDay"`
	AccountID   string `json:"accountId"`
	CategoryID  string `json:"categoryId"`
}

type ConfirmFixedExpenseOccurrenceInput struct {
	AmountCents    int64  `json:"amountCents"`
	OccurrenceDate string `json:"occurrenceDate"`
}

type BankStatementInput struct {
	AccountID string        `json:"accountId"`
	Bank      importer.Bank `json:"bank"`
	FileName  string        `json:"fileName"`
	Base64PDF string        `json:"base64Pdf"`
}

type BankStatementImportResult struct {
	Bank           importer.Bank `json:"bank"`
	ImportedCount  int           `json:"importedCount"`
	DuplicateCount int           `json:"duplicateCount"`
	IgnoredCount   int           `json:"ignoredCount"`
}

type FixedExpensesOverview struct {
	Expenses    []domain.FixedExpense           `json:"expenses"`
	Occurrences []domain.FixedExpenseOccurrence `json:"occurrences"`
}

type Bootstrap struct {
	Profile    *domain.Profile   `json:"profile"`
	Setup      bool              `json:"setup"`
	Accounts   []domain.Account  `json:"accounts"`
	Categories []domain.Category `json:"categories"`
	Dashboard  domain.Dashboard  `json:"dashboard"`
	Theme      domain.Theme      `json:"theme"`
}

type Service struct {
	store *storage.Store
	now   Clock
}

func New(store *storage.Store, now Clock) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: store, now: now}
}

func (s *Service) Bootstrap(ctx context.Context) (Bootstrap, error) {
	p, err := s.store.Profile(ctx)
	if err != nil {
		return Bootstrap{}, err
	}
	if p != nil {
		if err := s.ensureFixedExpenseOccurrences(ctx, s.now()); err != nil {
			return Bootstrap{}, err
		}
	}
	accounts, err := s.store.Accounts(ctx)
	if err != nil {
		return Bootstrap{}, err
	}
	categories, err := s.store.Categories(ctx)
	if err != nil {
		return Bootstrap{}, err
	}
	txs, err := s.store.Transactions(ctx)
	if err != nil {
		return Bootstrap{}, err
	}
	occurrences, err := s.store.FixedExpenseOccurrences(ctx)
	if err != nil {
		return Bootstrap{}, err
	}
	balances := make(map[string]int64, len(accounts))
	accountsByID := make(map[string]domain.Account, len(accounts))
	for _, account := range accounts {
		balances[account.ID] = account.OpeningBalanceCents
		accountsByID[account.ID] = account
	}
	for _, tx := range txs {
		domain.ApplyTransactionWithAccounts(balances, accountsByID, tx)
	}
	for index := range accounts {
		accounts[index].CurrentBalanceCents = balances[accounts[index].ID]
	}
	theme := domain.Theme("")
	if p != nil {
		theme = p.Theme
	}
	return Bootstrap{Profile: p, Setup: p != nil, Accounts: accounts, Categories: categories, Dashboard: domain.CalculateDashboardWithFixedExpenses(accounts, txs, occurrences, s.now()), Theme: theme}, nil
}

func (s *Service) CompleteOnboarding(ctx context.Context, in OnboardingInput) (domain.Profile, error) {
	if strings.TrimSpace(in.DisplayName) == "" {
		return domain.Profile{}, domain.ErrBlankName
	}
	if in.Currency != "BRL" {
		return domain.Profile{}, errors.New("invalid currency")
	}
	if err := domain.ValidateTheme(in.Theme); err != nil {
		return domain.Profile{}, err
	}
	if err := domain.ValidateAccount(in.FirstAccount.Name, in.FirstAccount.Type, in.FirstAccount.OpeningDate, s.now()); err != nil {
		return domain.Profile{}, err
	}
	if in.FirstAccount.Type == domain.AccountSavings && in.FirstAccount.OpeningBalanceCents < 0 {
		return domain.Profile{}, domain.ErrSavingsNegative
	}
	p := domain.Profile{DisplayName: strings.TrimSpace(in.DisplayName), Currency: "BRL", Theme: in.Theme, OnboardingStatus: "completed"}
	a := accountFromInput(in.FirstAccount, newID(), s.now())
	err := s.store.WithTx(ctx, func(q storage.Queries) error {
		if err := q.SaveProfile(ctx, p, a.CreatedAt); err != nil {
			return err
		}
		return q.InsertAccount(ctx, a, a.CreatedAt)
	})
	return p, err
}

func (s *Service) SkipOnboarding(ctx context.Context) (domain.Profile, error) {
	// An empty theme means no explicit choice: the frontend follows the system.
	p := domain.Profile{Currency: "BRL", Theme: "", OnboardingStatus: "skipped"}
	at := s.now().UTC().Format(time.RFC3339Nano)
	return p, s.store.WithTx(ctx, func(q storage.Queries) error { return q.SaveProfile(ctx, p, at) })
}

func (s *Service) CreateAccount(ctx context.Context, in AccountInput) (domain.Account, error) {
	if err := domain.ValidateAccount(in.Name, in.Type, in.OpeningDate, s.now()); err != nil {
		return domain.Account{}, err
	}
	if in.Type == domain.AccountSavings && in.OpeningBalanceCents < 0 {
		return domain.Account{}, domain.ErrSavingsNegative
	}
	a := accountFromInput(in, newID(), s.now())
	return a, s.store.InsertAccount(ctx, a, a.CreatedAt)
}

func (s *Service) UpdateAccount(ctx context.Context, id string, in AccountInput) (domain.Account, error) {
	if err := domain.ValidateAccount(in.Name, in.Type, in.OpeningDate, s.now()); err != nil {
		return domain.Account{}, err
	}
	now := s.now()
	at := now.UTC().Format(time.RFC3339Nano)
	var updated domain.Account
	err := s.store.WithTx(ctx, func(q storage.Queries) error {
		current, err := q.Account(ctx, id)
		if err != nil {
			return err
		}
		if current == nil {
			return domain.ErrUnknownAccount
		}
		linked, err := q.AccountTransactions(ctx, id)
		if err != nil {
			return err
		}
		opening, _ := domain.ParseCivilDate(in.OpeningDate)
		for _, tx := range linked {
			if tx.AutomaticImport {
				continue
			}
			occurrence, err := domain.ParseCivilDate(tx.OccurrenceDate)
			if err != nil {
				return err
			}
			if opening.After(occurrence) {
				return domain.ErrBeforeOpening
			}
		}
		updated = *current
		updated.Name = strings.TrimSpace(in.Name)
		updated.Type = in.Type
		updated.OpeningBalanceCents = in.OpeningBalanceCents
		updated.OpeningDate = in.OpeningDate
		accounts, err := q.Accounts(ctx)
		if err != nil {
			return err
		}
		for index := range accounts {
			if accounts[index].ID == id {
				accounts[index] = updated
			}
		}
		active, err := q.Transactions(ctx)
		if err != nil {
			return err
		}
		if err := domain.ValidateSavingsBalances(accounts, active); err != nil {
			return err
		}
		updated.CurrentBalanceCents = updated.OpeningBalanceCents
		balances := map[string]int64{id: updated.OpeningBalanceCents}
		accountsByID := make(map[string]domain.Account, len(accounts))
		for _, account := range accounts {
			accountsByID[account.ID] = account
		}
		for _, tx := range active {
			domain.ApplyTransactionWithAccounts(balances, accountsByID, tx)
		}
		updated.CurrentBalanceCents = balances[id]
		return q.UpdateAccount(ctx, updated, at)
	})
	return updated, err
}

func (s *Service) DeleteAccount(ctx context.Context, id string) error {
	return s.store.WithTx(ctx, func(q storage.Queries) error {
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

func (s *Service) CreateTransaction(ctx context.Context, in TransactionInput) (domain.Transaction, error) {
	now := s.now()
	at := now.UTC().Format(time.RFC3339Nano)
	tx := domain.Transaction{ID: newID(), Kind: in.Kind, AmountCents: in.AmountCents, AccountID: in.AccountID, DestinationAccountID: in.DestinationAccountID, CategoryID: in.CategoryID, Description: strings.TrimSpace(in.Description), OccurrenceDate: in.OccurrenceDate, CreatedAt: at, UpdatedAt: at}
	err := s.store.WithTx(ctx, func(q storage.Queries) error {
		if err := s.prepareTransaction(ctx, q, &tx, now); err != nil {
			return err
		}
		active, err := q.Transactions(ctx)
		if err != nil {
			return err
		}
		accounts, err := q.Accounts(ctx)
		if err != nil {
			return err
		}
		if err := domain.ValidateSavingsBalances(accounts, append(active, tx)); err != nil {
			return err
		}
		if err := q.InsertTransaction(ctx, tx, at); err != nil {
			return err
		}
		return q.InsertTransactionRevision(ctx, tx, "create", at)
	})
	return tx, err
}

func (s *Service) ImportBankStatement(ctx context.Context, in BankStatementInput) (BankStatementImportResult, error) {
	result := BankStatementImportResult{Bank: in.Bank}
	if !in.Bank.Valid() {
		return result, domain.ErrUnsupportedBank
	}
	if !strings.HasSuffix(strings.ToLower(strings.TrimSpace(in.FileName)), ".pdf") {
		return result, domain.ErrInvalidStatement
	}
	if len(in.Base64PDF) > ((importer.MaxPDFSize+2)/3)*4+8 {
		return result, domain.ErrStatementTooLarge
	}
	data, err := base64.StdEncoding.DecodeString(in.Base64PDF)
	if err != nil {
		return result, domain.ErrInvalidStatement
	}
	if len(data) > importer.MaxPDFSize {
		return result, domain.ErrStatementTooLarge
	}
	entries, err := importer.ParsePDF(data, in.Bank, s.now().In(time.Local).Year())
	if err != nil {
		return result, err
	}
	return s.importStatementEntries(ctx, in.AccountID, in.Bank, entries)
}

func (s *Service) importStatementEntries(ctx context.Context, accountID string, bank importer.Bank, entries []importer.Entry) (BankStatementImportResult, error) {
	result := BankStatementImportResult{Bank: bank}
	now := s.now()
	at := now.UTC().Format(time.RFC3339Nano)
	err := s.store.WithTx(ctx, func(q storage.Queries) error {
		account, err := q.Account(ctx, accountID)
		if err != nil {
			return err
		}
		if account == nil {
			return domain.ErrUnknownAccount
		}
		linked, err := q.AccountTransactions(ctx, accountID)
		if err != nil {
			return err
		}
		existing := make(map[string]struct{}, len(linked))
		for _, tx := range linked {
			if tx.ImportKey != "" {
				existing[tx.ImportKey] = struct{}{}
			}
		}
		active, err := q.Transactions(ctx)
		if err != nil {
			return err
		}
		today, _ := domain.ParseCivilDate(now.In(time.Local).Format("2006-01-02"))
		occurrences := map[string]int{}
		pending := make([]domain.Transaction, 0, len(entries))
		for _, entry := range entries {
			date, dateErr := domain.ParseCivilDate(entry.Date)
			if dateErr != nil || date.After(today) {
				result.IgnoredCount++
				continue
			}
			fingerprint := importFingerprint(accountID, bank, entry)
			occurrences[fingerprint]++
			keyBytes := sha256.Sum256([]byte(fmt.Sprintf("%s|%d", fingerprint, occurrences[fingerprint])))
			key := hex.EncodeToString(keyBytes[:])
			if _, found := existing[key]; found {
				result.DuplicateCount++
				continue
			}
			tx := domain.Transaction{ID: newID(), Kind: entry.Kind, AmountCents: entry.AmountCents, AccountID: accountID, Description: strings.TrimSpace(entry.Description), OccurrenceDate: entry.Date, CreatedAt: at, UpdatedAt: at, AutomaticImport: true, ImportBank: string(bank), ImportKey: key}
			if err := s.prepareTransaction(ctx, q, &tx, now); err != nil {
				return err
			}
			pending = append(pending, tx)
			existing[key] = struct{}{}
		}
		accounts, err := q.Accounts(ctx)
		if err != nil {
			return err
		}
		if err := domain.ValidateSavingsBalances(accounts, append(active, pending...)); err != nil {
			return err
		}
		for _, tx := range pending {
			if err := q.InsertTransaction(ctx, tx, at); err != nil {
				return err
			}
			if err := q.InsertTransactionRevision(ctx, tx, "create", at); err != nil {
				return err
			}
			result.ImportedCount++
		}
		return nil
	})
	return result, err
}

func importFingerprint(accountID string, bank importer.Bank, entry importer.Entry) string {
	description := strings.ToLower(strings.Join(strings.Fields(entry.Description), " "))
	return fmt.Sprintf("%s|%s|%s|%s|%d|%s", accountID, bank, entry.Date, entry.Kind, entry.AmountCents, description)
}

func (s *Service) UpdateTransaction(ctx context.Context, id string, in TransactionInput) (domain.Transaction, error) {
	now := s.now()
	at := now.UTC().Format(time.RFC3339Nano)
	var updated domain.Transaction
	err := s.store.WithTx(ctx, func(q storage.Queries) error {
		current, err := q.Transaction(ctx, id)
		if err != nil {
			return err
		}
		if current == nil {
			return domain.ErrUnknownTransaction
		}
		if current.DeletedAt != "" {
			return domain.ErrTransactionTrashed
		}
		updated = *current
		updated.Kind, updated.AmountCents, updated.AccountID = in.Kind, in.AmountCents, in.AccountID
		updated.DestinationAccountID, updated.CategoryID = in.DestinationAccountID, in.CategoryID
		updated.Description, updated.OccurrenceDate, updated.UpdatedAt = strings.TrimSpace(in.Description), in.OccurrenceDate, at
		if err := s.prepareTransaction(ctx, q, &updated, now); err != nil {
			return err
		}
		active, err := q.Transactions(ctx)
		if err != nil {
			return err
		}
		for index := range active {
			if active[index].ID == id {
				active[index] = updated
			}
		}
		accounts, err := q.Accounts(ctx)
		if err != nil {
			return err
		}
		if err := domain.ValidateSavingsBalances(accounts, active); err != nil {
			return err
		}
		if err := q.UpdateTransaction(ctx, updated, at); err != nil {
			return err
		}
		return q.InsertTransactionRevision(ctx, updated, "update", at)
	})
	return updated, err
}

func (s *Service) TrashTransaction(ctx context.Context, id string) error {
	now := s.now()
	at := now.UTC().Format(time.RFC3339Nano)
	return s.store.WithTx(ctx, func(q storage.Queries) error {
		current, err := q.Transaction(ctx, id)
		if err != nil {
			return err
		}
		if current == nil {
			return domain.ErrUnknownTransaction
		}
		if current.DeletedAt != "" {
			return domain.ErrTransactionTrashed
		}
		active, err := q.Transactions(ctx)
		if err != nil {
			return err
		}
		remaining := make([]domain.Transaction, 0, len(active)-1)
		for _, tx := range active {
			if tx.ID != id {
				remaining = append(remaining, tx)
			}
		}
		accounts, err := q.Accounts(ctx)
		if err != nil {
			return err
		}
		if err := domain.ValidateSavingsBalances(accounts, remaining); err != nil {
			return err
		}
		if err := q.SetTransactionDeletedAt(ctx, id, at, at); err != nil {
			return err
		}
		if current.FixedExpenseOccurrenceID != "" {
			occurrence, err := q.FixedExpenseOccurrence(ctx, current.FixedExpenseOccurrenceID)
			if err != nil {
				return err
			}
			if occurrence != nil && occurrence.Status == domain.FixedExpenseConfirmed && occurrence.TransactionID == id {
				if err := q.SetFixedExpenseOccurrence(ctx, occurrence.ID, domain.FixedExpensePending, "", at); err != nil {
					return err
				}
			}
		}
		current.DeletedAt, current.UpdatedAt = at, at
		return q.InsertTransactionRevision(ctx, *current, "trash", at)
	})
}

func (s *Service) RestoreTransaction(ctx context.Context, id string) error {
	now := s.now()
	at := now.UTC().Format(time.RFC3339Nano)
	return s.store.WithTx(ctx, func(q storage.Queries) error {
		current, err := q.Transaction(ctx, id)
		if err != nil {
			return err
		}
		if current == nil {
			return domain.ErrUnknownTransaction
		}
		if current.DeletedAt == "" {
			return domain.ErrTransactionActive
		}
		active, err := q.Transactions(ctx)
		if err != nil {
			return err
		}
		restored := *current
		restored.DeletedAt = ""
		restored.UpdatedAt = at
		accounts, err := q.Accounts(ctx)
		if err != nil {
			return err
		}
		if err := domain.ValidateSavingsBalances(accounts, append(active, restored)); err != nil {
			return err
		}
		if err := q.SetTransactionDeletedAt(ctx, id, "", at); err != nil {
			return err
		}
		if current.FixedExpenseOccurrenceID != "" {
			occurrence, err := q.FixedExpenseOccurrence(ctx, current.FixedExpenseOccurrenceID)
			if err != nil {
				return err
			}
			if occurrence == nil || occurrence.Status != domain.FixedExpensePending || occurrence.TransactionID != "" {
				return domain.ErrOccurrenceClosed
			}
			if err := q.SetFixedExpenseOccurrence(ctx, occurrence.ID, domain.FixedExpenseConfirmed, id, at); err != nil {
				return err
			}
		}
		return q.InsertTransactionRevision(ctx, restored, "restore", at)
	})
}

func (s *Service) prepareTransaction(ctx context.Context, q storage.Queries, tx *domain.Transaction, now time.Time) error {
	account, err := q.Account(ctx, tx.AccountID)
	if err != nil {
		return err
	}
	if account == nil {
		return domain.ErrUnknownAccount
	}
	var destination *domain.Account
	if tx.DestinationAccountID != "" {
		destination, err = q.Account(ctx, tx.DestinationAccountID)
		if err != nil {
			return err
		}
		if destination == nil {
			return domain.ErrUnknownAccount
		}
	}
	var category *domain.Category
	if tx.CategoryID != "" {
		category, err = q.Category(ctx, tx.CategoryID)
		if err != nil {
			return err
		}
		if category == nil {
			return domain.ErrUnknownCategory
		}
	}
	tx.AccountName = account.Name
	tx.DestinationAccountName = ""
	if destination != nil {
		tx.DestinationAccountName = destination.Name
	}
	tx.CategoryName = ""
	if category != nil {
		tx.CategoryName = category.Name
	}
	if tx.Kind == domain.Transfer && tx.Description == "" && destination != nil {
		tx.Description = "Transferência para " + destination.Name
	}
	return domain.ValidateTransaction(*tx, *account, destination, category, now)
}

func (s *Service) ListTransactions(ctx context.Context) ([]domain.Transaction, error) {
	return s.store.Transactions(ctx)
}

func (s *Service) SetTheme(ctx context.Context, theme domain.Theme) (domain.Profile, error) {
	if err := domain.ValidateTheme(theme); err != nil {
		return domain.Profile{}, err
	}
	p, err := s.store.Profile(ctx)
	if err != nil {
		return domain.Profile{}, err
	}
	if p == nil {
		p = &domain.Profile{Currency: "BRL", OnboardingStatus: "skipped"}
	}
	p.Theme = theme
	return *p, s.store.SaveProfile(ctx, *p, s.now().UTC().Format(time.RFC3339Nano))
}

func (s *Service) SetBalancesHidden(ctx context.Context, hidden bool) (domain.Profile, error) {
	p, err := s.store.Profile(ctx)
	if err != nil {
		return domain.Profile{}, err
	}
	if p == nil {
		p = &domain.Profile{Currency: "BRL", OnboardingStatus: "skipped"}
	}
	p.BalancesHidden = hidden
	if err := s.store.SaveProfile(ctx, *p, s.now().UTC().Format(time.RFC3339Nano)); err != nil {
		return domain.Profile{}, err
	}
	return *p, nil
}

func (s *Service) FixedExpensesOverview(ctx context.Context) (FixedExpensesOverview, error) {
	if err := s.ensureFixedExpenseOccurrences(ctx, s.now()); err != nil {
		return FixedExpensesOverview{}, err
	}
	expenses, err := s.store.FixedExpenses(ctx)
	if err != nil {
		return FixedExpensesOverview{}, err
	}
	occurrences, err := s.store.FixedExpenseOccurrences(ctx)
	if err != nil {
		return FixedExpensesOverview{}, err
	}
	return FixedExpensesOverview{Expenses: expenses, Occurrences: occurrences}, nil
}

func (s *Service) CreateFixedExpense(ctx context.Context, in FixedExpenseInput) (domain.FixedExpense, error) {
	now := s.now()
	at := now.UTC().Format(time.RFC3339Nano)
	expense := domain.FixedExpense{ID: newID(), Description: strings.TrimSpace(in.Description), AmountCents: in.AmountCents, DueDay: in.DueDay, AccountID: in.AccountID, CategoryID: in.CategoryID, CreatedAt: at, UpdatedAt: at}
	err := s.store.WithTx(ctx, func(q storage.Queries) error {
		if err := validateFixedExpenseInput(ctx, q, expense); err != nil {
			return err
		}
		if err := q.InsertFixedExpense(ctx, expense, at); err != nil {
			return err
		}
		return q.InsertFixedExpenseOccurrence(ctx, occurrenceFromExpense(expense, monthStart(now), now), at)
	})
	if err != nil {
		return domain.FixedExpense{}, err
	}
	stored, err := s.store.FixedExpenses(ctx)
	if err != nil {
		return domain.FixedExpense{}, err
	}
	for _, item := range stored {
		if item.ID == expense.ID {
			return item, nil
		}
	}
	return domain.FixedExpense{}, domain.ErrUnknownFixedExpense
}

func (s *Service) UpdateFixedExpense(ctx context.Context, id string, in FixedExpenseInput) (domain.FixedExpense, error) {
	now := s.now()
	at := now.UTC().Format(time.RFC3339Nano)
	expense := domain.FixedExpense{ID: id, Description: strings.TrimSpace(in.Description), AmountCents: in.AmountCents, DueDay: in.DueDay, AccountID: in.AccountID, CategoryID: in.CategoryID, UpdatedAt: at}
	err := s.store.WithTx(ctx, func(q storage.Queries) error {
		current, err := q.FixedExpense(ctx, id)
		if err != nil {
			return err
		}
		if current == nil {
			return domain.ErrUnknownFixedExpense
		}
		if current.ArchivedAt != "" {
			return domain.ErrFixedExpenseArchived
		}
		if err := validateFixedExpenseInput(ctx, q, expense); err != nil {
			return err
		}
		return q.UpdateFixedExpense(ctx, expense, at)
	})
	if err != nil {
		return domain.FixedExpense{}, err
	}
	items, err := s.store.FixedExpenses(ctx)
	if err != nil {
		return domain.FixedExpense{}, err
	}
	for _, item := range items {
		if item.ID == id {
			return item, nil
		}
	}
	return domain.FixedExpense{}, domain.ErrUnknownFixedExpense
}

func (s *Service) ArchiveFixedExpense(ctx context.Context, id string) error {
	at := s.now().UTC().Format(time.RFC3339Nano)
	return s.store.WithTx(ctx, func(q storage.Queries) error {
		current, err := q.FixedExpense(ctx, id)
		if err != nil {
			return err
		}
		if current == nil {
			return domain.ErrUnknownFixedExpense
		}
		return q.SetFixedExpenseArchivedAt(ctx, id, at, at)
	})
}

func (s *Service) RestoreFixedExpense(ctx context.Context, id string) error {
	at := s.now().UTC().Format(time.RFC3339Nano)
	return s.store.WithTx(ctx, func(q storage.Queries) error {
		current, err := q.FixedExpense(ctx, id)
		if err != nil {
			return err
		}
		if current == nil {
			return domain.ErrUnknownFixedExpense
		}
		return q.SetFixedExpenseArchivedAt(ctx, id, "", at)
	})
}

func (s *Service) ConfirmFixedExpenseOccurrence(ctx context.Context, id string, in ConfirmFixedExpenseOccurrenceInput) (domain.Transaction, error) {
	now := s.now()
	at := now.UTC().Format(time.RFC3339Nano)
	var tx domain.Transaction
	err := s.store.WithTx(ctx, func(q storage.Queries) error {
		occurrence, err := q.FixedExpenseOccurrence(ctx, id)
		if err != nil {
			return err
		}
		if occurrence == nil {
			return domain.ErrUnknownOccurrence
		}
		if occurrence.Status != domain.FixedExpensePending {
			return domain.ErrOccurrenceClosed
		}
		tx = domain.Transaction{ID: newID(), Kind: domain.Expense, AmountCents: in.AmountCents, AccountID: occurrence.AccountID, CategoryID: occurrence.CategoryID, Description: occurrence.Description, OccurrenceDate: in.OccurrenceDate, FixedExpenseOccurrenceID: occurrence.ID, CreatedAt: at, UpdatedAt: at}
		if err := s.prepareTransaction(ctx, q, &tx, now); err != nil {
			return err
		}
		active, err := q.Transactions(ctx)
		if err != nil {
			return err
		}
		accounts, err := q.Accounts(ctx)
		if err != nil {
			return err
		}
		if err := domain.ValidateSavingsBalances(accounts, append(active, tx)); err != nil {
			return err
		}
		if err := q.InsertTransaction(ctx, tx, at); err != nil {
			return err
		}
		if err := q.SetFixedExpenseOccurrence(ctx, occurrence.ID, domain.FixedExpenseConfirmed, tx.ID, at); err != nil {
			return err
		}
		return q.InsertTransactionRevision(ctx, tx, "create", at)
	})
	return tx, err
}

func (s *Service) DismissFixedExpenseOccurrence(ctx context.Context, id string) error {
	at := s.now().UTC().Format(time.RFC3339Nano)
	return s.store.WithTx(ctx, func(q storage.Queries) error {
		occurrence, err := q.FixedExpenseOccurrence(ctx, id)
		if err != nil {
			return err
		}
		if occurrence == nil {
			return domain.ErrUnknownOccurrence
		}
		if occurrence.Status != domain.FixedExpensePending {
			return domain.ErrOccurrenceClosed
		}
		return q.SetFixedExpenseOccurrence(ctx, id, domain.FixedExpenseDismissed, "", at)
	})
}

func (s *Service) ReopenFixedExpenseOccurrence(ctx context.Context, id string) error {
	at := s.now().UTC().Format(time.RFC3339Nano)
	return s.store.WithTx(ctx, func(q storage.Queries) error {
		occurrence, err := q.FixedExpenseOccurrence(ctx, id)
		if err != nil {
			return err
		}
		if occurrence == nil {
			return domain.ErrUnknownOccurrence
		}
		if occurrence.Status != domain.FixedExpenseDismissed {
			return domain.ErrOccurrenceClosed
		}
		return q.SetFixedExpenseOccurrence(ctx, id, domain.FixedExpensePending, "", at)
	})
}

func (s *Service) ensureFixedExpenseOccurrences(ctx context.Context, now time.Time) error {
	at := now.UTC().Format(time.RFC3339Nano)
	return s.store.WithTx(ctx, func(q storage.Queries) error {
		expenses, err := q.FixedExpenses(ctx)
		if err != nil {
			return err
		}
		occurrences, err := q.FixedExpenseOccurrences(ctx)
		if err != nil {
			return err
		}
		existing := make(map[string]bool, len(occurrences))
		for _, occurrence := range occurrences {
			existing[occurrence.FixedExpenseID+":"+occurrence.ReferenceMonth] = true
		}
		currentMonth := monthStart(now)
		for _, expense := range expenses {
			if expense.ArchivedAt != "" {
				continue
			}
			created, err := time.Parse(time.RFC3339Nano, expense.CreatedAt)
			if err != nil {
				created = now
			}
			for month := monthStart(created.In(time.Local)); !month.After(currentMonth); month = month.AddDate(0, 1, 0) {
				key := expense.ID + ":" + month.Format("2006-01")
				if existing[key] {
					continue
				}
				if err := q.InsertFixedExpenseOccurrence(ctx, occurrenceFromExpense(expense, month, now), at); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func validateFixedExpenseInput(ctx context.Context, q storage.Queries, expense domain.FixedExpense) error {
	account, err := q.Account(ctx, expense.AccountID)
	if err != nil {
		return err
	}
	if account == nil {
		return domain.ErrUnknownAccount
	}
	category, err := q.Category(ctx, expense.CategoryID)
	if err != nil {
		return err
	}
	return domain.ValidateFixedExpense(expense.Description, expense.AmountCents, expense.DueDay, category)
}

func occurrenceFromExpense(expense domain.FixedExpense, month, now time.Time) domain.FixedExpenseOccurrence {
	at := now.UTC().Format(time.RFC3339Nano)
	return domain.FixedExpenseOccurrence{ID: newID(), FixedExpenseID: expense.ID, ReferenceMonth: month.Format("2006-01"), DueDate: dueDate(month, expense.DueDay), Description: expense.Description, ExpectedAmountCents: expense.AmountCents, AccountID: expense.AccountID, CategoryID: expense.CategoryID, Status: domain.FixedExpensePending, CreatedAt: at, UpdatedAt: at}
}

func monthStart(value time.Time) time.Time {
	local := value.In(time.Local)
	return time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, time.Local)
}

func dueDate(month time.Time, day int) string {
	lastDay := time.Date(month.Year(), month.Month()+1, 0, 0, 0, 0, 0, month.Location()).Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(month.Year(), month.Month(), day, 0, 0, 0, 0, month.Location()).Format("2006-01-02")
}

func accountFromInput(in AccountInput, id string, now time.Time) domain.Account {
	return domain.Account{ID: id, Name: strings.TrimSpace(in.Name), Type: in.Type, OpeningBalanceCents: in.OpeningBalanceCents, OpeningDate: in.OpeningDate, CreatedAt: now.UTC().Format(time.RFC3339Nano)}
}

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("random identifier: %v", err))
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return hex.EncodeToString(b[0:4]) + "-" + hex.EncodeToString(b[4:6]) + "-" + hex.EncodeToString(b[6:8]) + "-" + hex.EncodeToString(b[8:10]) + "-" + hex.EncodeToString(b[10:16])
}
