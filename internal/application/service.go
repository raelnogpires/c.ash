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
	"path/filepath"
	"sort"
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
	CreditLimitCents    int64              `json:"creditLimitCents"`
	ClosingDay          int                `json:"closingDay"`
	DueDay              int                `json:"dueDay"`
	OpeningDebtCents    int64              `json:"openingDebtCents"`
	OpeningDebtDueDate  string             `json:"openingDebtDueDate"`
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
	InstallmentCount     int                    `json:"installmentCount"`
}

type CreditCardPaymentInput struct {
	AccountID      string `json:"accountId"`
	AmountCents    int64  `json:"amountCents"`
	OccurrenceDate string `json:"occurrenceDate"`
}

type CreditCardsOverview struct {
	Cards    []domain.CreditCardSummary `json:"cards"`
	Invoices []domain.CreditCardInvoice `json:"invoices"`
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
	AccountID  string        `json:"accountId"`
	Bank       importer.Bank `json:"bank"`
	FileName   string        `json:"fileName"`
	Base64Data string        `json:"base64Data"`
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

type EncryptionInput struct {
	Password     string `json:"password"`
	Confirmation string `json:"confirmation"`
}

type ChangeEncryptionPasswordInput struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
	Confirmation    string `json:"confirmation"`
}

type RecoverEncryptionInput struct {
	RecoveryKey  string `json:"recoveryKey"`
	NewPassword  string `json:"newPassword"`
	Confirmation string `json:"confirmation"`
}

type UnlockInput struct {
	Password    string `json:"password"`
	RecoveryKey string `json:"recoveryKey"`
}

type RestoreBackupInput struct {
	Path        string `json:"path"`
	Password    string `json:"password"`
	RecoveryKey string `json:"recoveryKey"`
}

func New(store *storage.Store, now Clock) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: store, now: now}
}

func (s *Service) SecurityStatus() storage.SecurityStatus { return s.store.SecurityStatus() }

func (s *Service) UnlockDatabase(ctx context.Context, in UnlockInput) error {
	return s.store.Unlock(ctx, in.Password, in.RecoveryKey)
}

func (s *Service) EnableEncryption(ctx context.Context, in EncryptionInput, version string) (storage.EncryptionResult, error) {
	return s.store.EnableEncryption(ctx, in.Password, in.Confirmation, version)
}

func (s *Service) ChangeEncryptionPassword(ctx context.Context, in ChangeEncryptionPasswordInput, version string) error {
	return s.store.ChangePassword(ctx, in.CurrentPassword, in.NewPassword, in.Confirmation, version)
}

func (s *Service) RecoverEncryption(ctx context.Context, in RecoverEncryptionInput, version string) error {
	return s.store.RecoverPassword(ctx, in.RecoveryKey, in.NewPassword, in.Confirmation, version)
}

func (s *Service) DisableEncryption(ctx context.Context, in UnlockInput, version string) error {
	return s.store.DisableEncryption(ctx, in.Password, in.RecoveryKey, version)
}

func (s *Service) BackupStatus() (storage.BackupStatus, error) {
	return s.store.BackupStatus(s.now())
}

func (s *Service) CreateBackup(ctx context.Context, path, version string) (storage.BackupInfo, error) {
	return s.store.CreateBackupAt(ctx, path, storage.BackupKindManual, version)
}

func (s *Service) RunAutomaticBackup(ctx context.Context, version string) (storage.BackupStatus, error) {
	return s.store.RunAutomaticBackup(ctx, version, s.now())
}

func (s *Service) SetBackupFolder(path string) error { return s.store.SetBackupFolder(path) }
func (s *Service) ResetBackupFolder() error          { return s.store.ResetBackupFolder() }
func (s *Service) InspectBackup(path string) (storage.BackupInfo, error) {
	return s.store.InspectBackup(path)
}

func (s *Service) RestoreBackup(ctx context.Context, in RestoreBackupInput, version string) (storage.BackupInfo, error) {
	return s.store.RestoreBackup(ctx, in.Path, storage.RestoreCredential{Password: in.Password, RecoveryKey: in.RecoveryKey}, version)
}

func (s *Service) ExportData(ctx context.Context, format storage.ExportFormat, path, version string) error {
	return s.store.ExportData(ctx, format, path, version)
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
	dashboard := domain.CalculateDashboardWithFixedExpenses(accounts, txs, occurrences, s.now())
	overview, err := s.creditCardsOverview(ctx)
	if err != nil {
		return Bootstrap{}, err
	}
	for _, invoice := range overview.Invoices {
		if invoice.Status == domain.InvoiceOpen || invoice.Status == domain.InvoiceClosed {
			dashboard.UpcomingInvoices = append(dashboard.UpcomingInvoices, invoice)
		}
	}
	if len(dashboard.UpcomingInvoices) > 3 {
		dashboard.UpcomingInvoices = dashboard.UpcomingInvoices[:3]
	}
	return Bootstrap{Profile: p, Setup: p != nil, Accounts: accounts, Categories: categories, Dashboard: dashboard, Theme: theme}, nil
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
	if err := validateAccountInput(a, in.FirstAccount, s.now()); err != nil {
		return domain.Profile{}, err
	}
	err := s.store.WithTx(ctx, func(q storage.Queries) error {
		if err := q.SaveProfile(ctx, p, a.CreatedAt); err != nil {
			return err
		}
		if err := q.InsertAccount(ctx, a, a.CreatedAt); err != nil {
			return err
		}
		return s.insertOpeningInvoice(ctx, q, a, in.FirstAccount, a.CreatedAt)
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
	if err := validateAccountInput(a, in, s.now()); err != nil {
		return domain.Account{}, err
	}
	err := s.store.WithTx(ctx, func(q storage.Queries) error {
		if err := q.InsertAccount(ctx, a, a.CreatedAt); err != nil {
			return err
		}
		return s.insertOpeningInvoice(ctx, q, a, in, a.CreatedAt)
	})
	return a, err
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
		if current.Type != in.Type && (current.Type == domain.AccountCreditCard || in.Type == domain.AccountCreditCard) {
			invoices, err := q.CreditCardInvoices(ctx, id)
			if err != nil {
				return err
			}
			if len(linked) > 0 || len(invoices) > 0 {
				return domain.ErrAccountInUse
			}
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
		if current.Type == domain.AccountCreditCard && updated.Type == domain.AccountCreditCard {
			updated.OpeningBalanceCents = current.OpeningBalanceCents
		} else if updated.Type == domain.AccountCreditCard {
			updated.OpeningBalanceCents = -in.OpeningDebtCents
		}
		updated.OpeningDate = in.OpeningDate
		updated.CreditLimitCents, updated.ClosingDay, updated.DueDay = in.CreditLimitCents, in.ClosingDay, in.DueDay
		if err := validateAccountInput(updated, in, now); err != nil {
			return err
		}
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
		if err := q.UpdateAccount(ctx, updated, at); err != nil {
			return err
		}
		if current.Type != domain.AccountCreditCard && updated.Type == domain.AccountCreditCard {
			return s.insertOpeningInvoice(ctx, q, updated, in, at)
		}
		return nil
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
		invoices, err := q.CreditCardInvoices(ctx, id)
		if err != nil {
			return err
		}
		if len(invoices) != 0 {
			return domain.ErrAccountInUse
		}
		return q.DeleteAccount(ctx, id)
	})
}

func (s *Service) CreateTransaction(ctx context.Context, in TransactionInput) (domain.Transaction, error) {
	now := s.now()
	at := now.UTC().Format(time.RFC3339Nano)
	count := in.InstallmentCount
	if count == 0 {
		count = 1
	}
	tx := domain.Transaction{ID: newID(), Kind: in.Kind, AmountCents: in.AmountCents, AccountID: in.AccountID, DestinationAccountID: in.DestinationAccountID, CategoryID: in.CategoryID, Description: strings.TrimSpace(in.Description), OccurrenceDate: in.OccurrenceDate, InstallmentCount: count, CreatedAt: at, UpdatedAt: at}
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
		account, _ := q.Account(ctx, tx.AccountID)
		if account != nil && account.Type == domain.AccountCreditCard {
			if err := s.insertPurchaseSchedule(ctx, q, *account, tx, at); err != nil {
				return err
			}
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
	extension := strings.ToLower(filepath.Ext(strings.TrimSpace(in.FileName)))
	if extension != ".pdf" && extension != ".ofx" && extension != ".csv" {
		return result, domain.ErrInvalidStatement
	}
	if len(in.Base64Data) > ((importer.MaxStatementSize+2)/3)*4+8 {
		return result, domain.ErrStatementTooLarge
	}
	data, err := base64.StdEncoding.DecodeString(in.Base64Data)
	if err != nil {
		return result, domain.ErrInvalidStatement
	}
	if len(data) > importer.MaxStatementSize {
		return result, domain.ErrStatementTooLarge
	}
	var entries []importer.Entry
	switch extension {
	case ".pdf":
		entries, err = importer.ParsePDF(data, in.Bank, s.now().In(time.Local).Year())
	case ".ofx":
		entries, err = importer.ParseOFX(data)
	case ".csv":
		entries, err = importer.ParseCSV(data)
	}
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
		if account.Type == domain.AccountCreditCard {
			return domain.ErrCardTransaction
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
		if current.InvoicePaymentID != "" {
			return domain.ErrInvoiceLocked
		}
		count := in.InstallmentCount
		if count == 0 {
			count = 1
		}
		scheduleChanged := current.Kind != in.Kind || current.AmountCents != in.AmountCents || current.AccountID != in.AccountID ||
			current.DestinationAccountID != in.DestinationAccountID || current.OccurrenceDate != in.OccurrenceDate || current.InstallmentCount != count
		if scheduleChanged {
			if err := s.ensureTransactionScheduleEditable(ctx, q, *current); err != nil {
				return err
			}
		}
		updated = *current
		updated.Kind, updated.AmountCents, updated.AccountID = in.Kind, in.AmountCents, in.AccountID
		updated.DestinationAccountID, updated.CategoryID = in.DestinationAccountID, in.CategoryID
		updated.InstallmentCount = in.InstallmentCount
		if updated.InstallmentCount == 0 {
			updated.InstallmentCount = 1
		}
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
		if scheduleChanged {
			if err := q.DeleteTransactionInstallments(ctx, id); err != nil {
				return err
			}
			account, _ := q.Account(ctx, updated.AccountID)
			if account != nil && account.Type == domain.AccountCreditCard {
				if err := s.insertPurchaseSchedule(ctx, q, *account, updated, at); err != nil {
					return err
				}
			}
		} else if err := q.UpdateTransactionInstallmentDescriptions(ctx, id, updated.Description); err != nil {
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
		if current.InvoicePaymentID != "" {
			return domain.ErrInvoiceLocked
		}
		if err := s.ensureTransactionScheduleEditable(ctx, q, *current); err != nil {
			return err
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
		if current.InvoicePaymentID != "" {
			return domain.ErrInvoiceLocked
		}
		if err := s.ensureTransactionScheduleEditable(ctx, q, *current); err != nil {
			return err
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

func (s *Service) DeleteTransactionPermanently(ctx context.Context, id string) error {
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
		return deleteTrashedTransaction(ctx, q, id)
	})
}

func (s *Service) EmptyTransactionTrash(ctx context.Context) error {
	return s.store.WithTx(ctx, func(q storage.Queries) error {
		transactions, err := q.TrashedTransactions(ctx)
		if err != nil {
			return err
		}
		for _, transaction := range transactions {
			if err := deleteTrashedTransaction(ctx, q, transaction.ID); err != nil {
				return err
			}
		}
		return nil
	})
}

func deleteTrashedTransaction(ctx context.Context, q storage.Queries, id string) error {
	if err := q.DeleteTransactionInstallments(ctx, id); err != nil {
		return err
	}
	if err := q.DeleteTransactionRevisions(ctx, id); err != nil {
		return err
	}
	return q.DeleteTransaction(ctx, id)
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

func (s *Service) ListTrashedTransactions(ctx context.Context) ([]domain.Transaction, error) {
	return s.store.TrashedTransactions(ctx)
}

func (s *Service) CreditCardsOverview(ctx context.Context) (CreditCardsOverview, error) {
	return s.creditCardsOverview(ctx)
}

func (s *Service) creditCardsOverview(ctx context.Context) (CreditCardsOverview, error) {
	result := CreditCardsOverview{Cards: []domain.CreditCardSummary{}, Invoices: []domain.CreditCardInvoice{}}
	err := s.store.WithTx(ctx, func(q storage.Queries) error {
		accounts, err := q.Accounts(ctx)
		if err != nil {
			return err
		}
		transactions, err := q.Transactions(ctx)
		if err != nil {
			return err
		}
		balances := map[string]int64{}
		for _, account := range accounts {
			balances[account.ID] = account.OpeningBalanceCents
		}
		for _, transaction := range transactions {
			domain.ApplyTransaction(balances, transaction)
		}
		for _, account := range accounts {
			if account.Type != domain.AccountCreditCard {
				continue
			}
			invoices, err := s.reconcileCardInvoices(ctx, q, account)
			if err != nil {
				return err
			}
			result.Invoices = append(result.Invoices, invoices...)
			outstanding := int64(0)
			if balances[account.ID] < 0 {
				outstanding = -balances[account.ID]
			}
			summary := domain.CreditCardSummary{Account: account, OutstandingCents: outstanding,
				AvailableLimitCents: account.CreditLimitCents - outstanding}
			for index := range invoices {
				if invoices[index].Status == domain.InvoiceOpen || invoices[index].Status == domain.InvoiceClosed {
					summary.CurrentInvoice = &invoices[index]
					break
				}
			}
			result.Cards = append(result.Cards, summary)
		}
		return nil
	})
	sort.SliceStable(result.Invoices, func(i, j int) bool { return result.Invoices[i].DueDate < result.Invoices[j].DueDate })
	return result, err
}

func (s *Service) reconcileCardInvoices(ctx context.Context, q storage.Queries, account domain.Account) ([]domain.CreditCardInvoice, error) {
	invoices, err := q.CreditCardInvoices(ctx, account.ID)
	if err != nil {
		return nil, err
	}
	nowDate, _ := domain.ParseCivilDate(s.now().In(time.Local).Format("2006-01-02"))
	at := s.now().UTC().Format(time.RFC3339Nano)
	for index := 0; index < len(invoices); index++ {
		invoice, err := hydrateInvoice(ctx, q, invoices[index])
		if err != nil {
			return nil, err
		}
		if invoice.Status == domain.InvoiceRolledOver {
			continue
		}
		closing, _ := domain.ParseCivilDate(invoice.ClosingDate)
		due, _ := domain.ParseCivilDate(invoice.DueDate)
		status := domain.InvoiceOpen
		if !nowDate.Before(closing) {
			status = domain.InvoiceClosed
		}
		if invoice.OutstandingCents == 0 && invoice.ChargesCents > 0 {
			status = domain.InvoicePaid
		}
		if nowDate.After(due) && invoice.OutstandingCents > 0 {
			nextClosing, nextDue := domain.CreditCardCycle(due, account.ClosingDay, account.DueDay)
			next, err := ensureCreditCardInvoice(ctx, q, account, nextClosing, nextDue, at)
			if err != nil {
				return nil, err
			}
			if err := q.UpdateCreditCardInvoice(ctx, next.Status, next.CarryForwardCents+invoice.OutstandingCents, next.ID, at); err != nil {
				return nil, err
			}
			if err := q.UpdateCreditCardInvoice(ctx, domain.InvoiceRolledOver, invoice.CarryForwardCents, invoice.ID, at); err != nil {
				return nil, err
			}
			invoices, err = q.CreditCardInvoices(ctx, account.ID)
			if err != nil {
				return nil, err
			}
			index = -1
			continue
		}
		if status != invoice.Status {
			if err := q.UpdateCreditCardInvoice(ctx, status, invoice.CarryForwardCents, invoice.ID, at); err != nil {
				return nil, err
			}
		}
	}
	invoices, err = q.CreditCardInvoices(ctx, account.ID)
	if err != nil {
		return nil, err
	}
	for index := range invoices {
		invoices[index], err = hydrateInvoice(ctx, q, invoices[index])
		if err != nil {
			return nil, err
		}
		if invoices[index].Status == domain.InvoiceRolledOver {
			invoices[index].OutstandingCents = 0
		}
	}
	return invoices, nil
}

func hydrateInvoice(ctx context.Context, q storage.Queries, invoice domain.CreditCardInvoice) (domain.CreditCardInvoice, error) {
	items, err := q.InvoiceInstallments(ctx, invoice.ID)
	if err != nil {
		return invoice, err
	}
	payments, err := q.InvoicePayments(ctx, invoice.ID)
	if err != nil {
		return invoice, err
	}
	invoice.Installments, invoice.Payments = items, payments
	invoice.ChargesCents = invoice.CarryForwardCents
	for _, item := range items {
		invoice.ChargesCents += item.AmountCents
	}
	for _, payment := range payments {
		invoice.PaidCents += payment.AmountCents
	}
	invoice.OutstandingCents = invoice.ChargesCents - invoice.PaidCents
	if invoice.OutstandingCents < 0 {
		invoice.OutstandingCents = 0
	}
	return invoice, nil
}

func (s *Service) PayCreditCardInvoice(ctx context.Context, invoiceID string, in CreditCardPaymentInput) (domain.CreditCardPayment, error) {
	if in.AmountCents <= 0 {
		return domain.CreditCardPayment{}, domain.ErrInvalidAmount
	}
	date, err := domain.ParseCivilDate(in.OccurrenceDate)
	if err != nil {
		return domain.CreditCardPayment{}, err
	}
	today, _ := domain.ParseCivilDate(s.now().In(time.Local).Format("2006-01-02"))
	if date.After(today) {
		return domain.CreditCardPayment{}, domain.ErrFutureDate
	}
	at := s.now().UTC().Format(time.RFC3339Nano)
	payment := domain.CreditCardPayment{ID: newID(), InvoiceID: invoiceID, AccountID: in.AccountID,
		AmountCents: in.AmountCents, OccurrenceDate: in.OccurrenceDate, CreatedAt: at}
	err = s.store.WithTx(ctx, func(q storage.Queries) error {
		invoice, err := q.CreditCardInvoice(ctx, invoiceID)
		if err != nil {
			return err
		}
		if invoice == nil {
			return domain.ErrUnknownInvoice
		}
		card, err := q.Account(ctx, invoice.AccountID)
		if err != nil {
			return err
		}
		if card == nil || card.Type != domain.AccountCreditCard {
			return domain.ErrUnknownAccount
		}
		invoices, err := s.reconcileCardInvoices(ctx, q, *card)
		if err != nil {
			return err
		}
		var payable *domain.CreditCardInvoice
		for index := range invoices {
			if invoices[index].ID == invoiceID {
				payable = &invoices[index]
				break
			}
		}
		if payable == nil {
			return domain.ErrUnknownInvoice
		}
		if payable.Status == domain.InvoicePaid || payable.Status == domain.InvoiceRolledOver || payable.OutstandingCents <= 0 {
			return domain.ErrInvoiceNotPayable
		}
		if in.AmountCents > payable.OutstandingCents {
			return domain.ErrInvoiceOverpayment
		}
		source, err := q.Account(ctx, in.AccountID)
		if err != nil {
			return err
		}
		if source == nil || source.Type == domain.AccountCreditCard {
			return domain.ErrInvalidPaymentAccount
		}
		payment.AccountName = source.Name
		payment.TransactionID = newID()
		tx := domain.Transaction{ID: payment.TransactionID, Kind: domain.Transfer, AmountCents: in.AmountCents,
			AccountID: source.ID, DestinationAccountID: card.ID, Description: "Pagamento da fatura " + card.Name,
			OccurrenceDate: in.OccurrenceDate, InstallmentCount: 1, InvoicePaymentID: payment.ID, CreatedAt: at, UpdatedAt: at}
		if err := s.prepareTransaction(ctx, q, &tx, s.now()); err != nil {
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
		if err := q.InsertTransactionRevision(ctx, tx, "create", at); err != nil {
			return err
		}
		if err := q.InsertCreditCardPayment(ctx, payment); err != nil {
			return err
		}
		remaining := payable.OutstandingCents - in.AmountCents
		status := payable.Status
		if remaining == 0 {
			status = domain.InvoicePaid
		}
		return q.UpdateCreditCardInvoice(ctx, status, payable.CarryForwardCents, payable.ID, at)
	})
	return payment, err
}

func (s *Service) ensureTransactionScheduleEditable(ctx context.Context, q storage.Queries, tx domain.Transaction) error {
	items, err := q.TransactionInstallments(ctx, tx.ID)
	if err != nil {
		return err
	}
	for _, item := range items {
		invoice, err := q.CreditCardInvoice(ctx, item.InvoiceID)
		if err != nil {
			return err
		}
		closing := time.Time{}
		if invoice != nil {
			closing, _ = domain.ParseCivilDate(invoice.ClosingDate)
		}
		today, _ := domain.ParseCivilDate(s.now().In(time.Local).Format("2006-01-02"))
		if invoice == nil || invoice.Status != domain.InvoiceOpen || !today.Before(closing) {
			return domain.ErrInvoiceLocked
		}
		payments, err := q.InvoicePayments(ctx, invoice.ID)
		if err != nil {
			return err
		}
		if len(payments) > 0 {
			return domain.ErrInvoiceLocked
		}
	}
	return nil
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
	expense := domain.FixedExpense{ID: newID(), Description: strings.TrimSpace(in.Description), AmountCents: in.AmountCents, DueDay: in.DueDay, AccountID: in.AccountID, CategoryID: in.CategoryID, OccurrenceStartAt: at, CreatedAt: at, UpdatedAt: at}
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
	now := s.now()
	at := now.UTC().Format(time.RFC3339Nano)
	return s.store.WithTx(ctx, func(q storage.Queries) error {
		current, err := q.FixedExpense(ctx, id)
		if err != nil {
			return err
		}
		if current == nil {
			return domain.ErrUnknownFixedExpense
		}
		if current.ArchivedAt != "" {
			return nil
		}
		if err := reconcileFixedExpenseOccurrences(ctx, q, []domain.FixedExpense{*current}, now); err != nil {
			return err
		}
		return q.ArchiveFixedExpense(ctx, id, at)
	})
}

func (s *Service) RestoreFixedExpense(ctx context.Context, id string) error {
	now := s.now()
	at := now.UTC().Format(time.RFC3339Nano)
	return s.store.WithTx(ctx, func(q storage.Queries) error {
		current, err := q.FixedExpense(ctx, id)
		if err != nil {
			return err
		}
		if current == nil {
			return domain.ErrUnknownFixedExpense
		}
		if current.ArchivedAt == "" {
			return nil
		}
		if err := q.RestoreFixedExpense(ctx, id, at, at); err != nil {
			return err
		}
		current.ArchivedAt = ""
		current.OccurrenceStartAt = at
		return reconcileFixedExpenseOccurrences(ctx, q, []domain.FixedExpense{*current}, now)
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
		account, _ := q.Account(ctx, tx.AccountID)
		if account != nil && account.Type == domain.AccountCreditCard {
			tx.InstallmentCount = 1
			if err := s.insertPurchaseSchedule(ctx, q, *account, tx, at); err != nil {
				return err
			}
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
	return s.store.WithTx(ctx, func(q storage.Queries) error {
		expenses, err := q.FixedExpenses(ctx)
		if err != nil {
			return err
		}
		return reconcileFixedExpenseOccurrences(ctx, q, expenses, now)
	})
}

func reconcileFixedExpenseOccurrences(ctx context.Context, q storage.Queries, expenses []domain.FixedExpense, now time.Time) error {
	at := now.UTC().Format(time.RFC3339Nano)
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
		start, err := time.Parse(time.RFC3339Nano, expense.OccurrenceStartAt)
		if err != nil {
			start = now
		}
		for month := monthStart(start.In(time.Local)); !month.After(currentMonth); month = month.AddDate(0, 1, 0) {
			key := expense.ID + ":" + month.Format("2006-01")
			if existing[key] {
				continue
			}
			if err := q.InsertFixedExpenseOccurrence(ctx, occurrenceFromExpense(expense, month, now), at); err != nil {
				return err
			}
			existing[key] = true
		}
	}
	return nil
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
	opening := in.OpeningBalanceCents
	if in.Type == domain.AccountCreditCard {
		opening = -in.OpeningDebtCents
	}
	return domain.Account{ID: id, Name: strings.TrimSpace(in.Name), Type: in.Type, OpeningBalanceCents: opening, OpeningDate: in.OpeningDate,
		CreditLimitCents: in.CreditLimitCents, ClosingDay: in.ClosingDay, DueDay: in.DueDay, CreatedAt: now.UTC().Format(time.RFC3339Nano)}
}

func validateAccountInput(account domain.Account, in AccountInput, now time.Time) error {
	if err := domain.ValidateCreditCard(account); err != nil {
		return err
	}
	if account.Type != domain.AccountCreditCard {
		return nil
	}
	if in.OpeningDebtCents < 0 {
		return domain.ErrInvalidAmount
	}
	if in.OpeningDebtCents == 0 {
		return nil
	}
	due, err := domain.ParseCivilDate(in.OpeningDebtDueDate)
	if err != nil {
		return err
	}
	opened, _ := domain.ParseCivilDate(account.OpeningDate)
	if due.Before(opened) {
		return domain.ErrBeforeOpening
	}
	_ = now
	return nil
}

func (s *Service) insertOpeningInvoice(ctx context.Context, q storage.Queries, account domain.Account, in AccountInput, at string) error {
	if account.Type != domain.AccountCreditCard || in.OpeningDebtCents == 0 {
		return nil
	}
	due, _ := domain.ParseCivilDate(in.OpeningDebtDueDate)
	closing := closingDateForDue(due, account.ClosingDay, account.DueDay)
	invoice, err := ensureCreditCardInvoice(ctx, q, account, closing, due, at)
	if err != nil {
		return err
	}
	return q.InsertCreditCardInstallment(ctx, domain.CreditCardInstallment{ID: newID(), InvoiceID: invoice.ID,
		Description: "Saldo anterior", AmountCents: in.OpeningDebtCents, InstallmentNumber: 1, InstallmentCount: 1, OpeningDebt: true}, at)
}

func closingDateForDue(due time.Time, closingDay, dueDay int) time.Time {
	month := due.Month()
	year := due.Year()
	if dueDay <= closingDay {
		month--
	}
	base := time.Date(year, month, 1, 0, 0, 0, 0, due.Location())
	last := time.Date(base.Year(), base.Month()+1, 0, 0, 0, 0, 0, base.Location()).Day()
	if closingDay > last {
		closingDay = last
	}
	return time.Date(base.Year(), base.Month(), closingDay, 0, 0, 0, 0, base.Location())
}

func (s *Service) insertPurchaseSchedule(ctx context.Context, q storage.Queries, account domain.Account, tx domain.Transaction, at string) error {
	amounts, err := domain.InstallmentAmounts(tx.AmountCents, tx.InstallmentCount)
	if err != nil {
		return err
	}
	purchase, _ := domain.ParseCivilDate(tx.OccurrenceDate)
	closing, due := domain.CreditCardCycle(purchase, account.ClosingDay, account.DueDay)
	for index, amount := range amounts {
		if index > 0 {
			closing = cardCivilMonth(closing, 1, account.ClosingDay)
			due = invoiceDueDate(closing, account.ClosingDay, account.DueDay)
		}
		invoice, err := ensureCreditCardInvoice(ctx, q, account, closing, due, at)
		if err != nil {
			return err
		}
		item := domain.CreditCardInstallment{ID: newID(), InvoiceID: invoice.ID, TransactionID: tx.ID,
			Description: tx.Description, AmountCents: amount, InstallmentNumber: index + 1, InstallmentCount: len(amounts)}
		if err := q.InsertCreditCardInstallment(ctx, item, at); err != nil {
			return err
		}
	}
	return nil
}

func ensureCreditCardInvoice(ctx context.Context, q storage.Queries, account domain.Account, closing, due time.Time, at string) (domain.CreditCardInvoice, error) {
	invoices, err := q.CreditCardInvoices(ctx, account.ID)
	if err != nil {
		return domain.CreditCardInvoice{}, err
	}
	closingText := closing.Format("2006-01-02")
	for _, invoice := range invoices {
		if invoice.ClosingDate == closingText {
			return invoice, nil
		}
	}
	invoice := domain.CreditCardInvoice{ID: newID(), AccountID: account.ID, AccountName: account.Name,
		ReferenceMonth: due.Format("2006-01"), ClosingDate: closingText, DueDate: due.Format("2006-01-02"), Status: domain.InvoiceOpen}
	return invoice, q.InsertCreditCardInvoice(ctx, invoice, at)
}

func cardCivilMonth(value time.Time, months int, day int) time.Time {
	base := time.Date(value.Year(), value.Month()+time.Month(months), 1, 0, 0, 0, 0, value.Location())
	closing, _ := domain.CreditCardCycle(base.AddDate(0, 0, -1), day, 1)
	return closing
}

func invoiceDueDate(closing time.Time, closingDay, dueDay int) time.Time {
	month := closing.Month()
	year := closing.Year()
	if dueDay <= closingDay {
		month++
	}
	base := time.Date(year, month, 1, 0, 0, 0, 0, closing.Location())
	last := time.Date(base.Year(), base.Month()+1, 0, 0, 0, 0, 0, base.Location()).Day()
	if dueDay > last {
		dueDay = last
	}
	return time.Date(base.Year(), base.Month(), dueDay, 0, 0, 0, 0, base.Location())
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
