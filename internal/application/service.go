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
	DisplayName        string       `json:"displayName"`
	Currency           string       `json:"currency"`
	Theme              domain.Theme `json:"theme"`
	FirstAccount       AccountInput `json:"firstAccount"`
	ReserveTargetCents int64        `json:"reserveTargetCents"`
}

type TransactionInput struct {
	Kind                 domain.TransactionKind  `json:"kind"`
	AmountCents          int64                   `json:"amountCents"`
	AccountID            string                  `json:"accountId"`
	DestinationAccountID string                  `json:"destinationAccountId"`
	CategoryID           string                  `json:"categoryId"`
	Description          string                  `json:"description"`
	OccurrenceDate       string                  `json:"occurrenceDate"`
	InstallmentCount     int                     `json:"installmentCount"`
	SubcategoryName      string                  `json:"subcategoryName"`
	Tags                 []string                `json:"tags"`
	Splits               []TransactionSplitInput `json:"splits"`
	MonthlyRecurrence    bool                    `json:"monthlyRecurrence"`
}

type TransactionSplitInput struct {
	CategoryID      string `json:"categoryId"`
	SubcategoryName string `json:"subcategoryName"`
	AmountCents     int64  `json:"amountCents"`
}

type BalanceAdjustmentInput struct {
	TargetBalanceCents int64  `json:"targetBalanceCents"`
	OccurrenceDate     string `json:"occurrenceDate"`
	Reason             string `json:"reason"`
}

type CategoryBudgetInput struct {
	CategoryID string `json:"categoryId"`
	LimitCents int64  `json:"limitCents"`
	Rollover   bool   `json:"rollover"`
}

type CategoryInput struct {
	Name string                 `json:"name"`
	Kind domain.TransactionKind `json:"kind"`
}

type MonthlyBudgetInput struct {
	ReferenceMonth    string                `json:"referenceMonth"`
	OverallLimitCents int64                 `json:"overallLimitCents"`
	CategoryLimits    []CategoryBudgetInput `json:"categoryLimits"`
}

type GoalInput struct {
	Name        string          `json:"name"`
	Kind        domain.GoalKind `json:"kind"`
	TargetCents int64           `json:"targetCents"`
	Deadline    string          `json:"deadline"`
}

type GoalAllocationInput struct {
	AccountID   string `json:"accountId"`
	AmountCents int64  `json:"amountCents"`
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
	Planning   domain.Planning   `json:"planning"`
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
		if err := s.ensureRecurringOccurrences(ctx, s.now()); err != nil {
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
		linked, linkedErr := s.store.AccountTransactions(ctx, accounts[index].ID)
		if linkedErr != nil {
			return Bootstrap{}, linkedErr
		}
		accounts[index].HasLedgerActivity = len(linked) > 0
	}
	theme := domain.Theme("")
	if p != nil {
		theme = p.Theme
	}
	dashboard := domain.CalculateDashboardWithFixedExpenses(accounts, txs, occurrences, s.now())
	planning, err := s.planning(ctx, s.now().In(time.Local).Format("2006-01"), accounts, txs)
	if err != nil {
		return Bootstrap{}, err
	}
	reserved := int64(0)
	targets, allocated := int64(0), int64(0)
	for _, goal := range planning.Goals {
		if goal.ArchivedAt != "" {
			continue
		}
		reserved += goal.AllocatedCents
		targets += goal.TargetCents
		allocated += goal.AllocatedCents
	}
	eligible := int64(0)
	for _, account := range accounts {
		if account.Type != domain.AccountCreditCard {
			eligible += account.CurrentBalanceCents
		}
	}
	dashboard.ReservedValueCents = reserved
	dashboard.EligibleBalanceCents = eligible
	dashboard.FreeValueCents = eligible - reserved
	dashboard.SafelySpendableCents = dashboard.FreeValueCents - dashboard.PendingFixedExpensesCents
	dashboard.AvailableBalanceCents = dashboard.SafelySpendableCents
	if planning.Budget != nil {
		dashboard.BudgetProgressPercent = planning.Budget.ProgressPercent
	}
	if targets > 0 {
		dashboard.GoalProgressPercent = float64(allocated) / float64(targets) * 100
	}
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
	return Bootstrap{Profile: p, Setup: p != nil, Accounts: accounts, Categories: categories, Dashboard: dashboard, Theme: theme, Planning: planning}, nil
}

func (s *Service) Planning(ctx context.Context, month string) (domain.Planning, error) {
	if month == "" {
		month = s.now().In(time.Local).Format("2006-01")
	}
	accounts, err := s.store.Accounts(ctx)
	if err != nil {
		return domain.Planning{}, err
	}
	transactions, err := s.store.Transactions(ctx)
	if err != nil {
		return domain.Planning{}, err
	}
	return s.planning(ctx, month, accounts, transactions)
}

func (s *Service) planning(ctx context.Context, month string, accounts []domain.Account, transactions []domain.Transaction) (domain.Planning, error) {
	if _, err := time.Parse("2006-01", month); err != nil {
		return domain.Planning{}, domain.ErrInvalidDate
	}
	budget, err := s.store.MonthlyBudget(ctx, month)
	if err != nil {
		return domain.Planning{}, err
	}
	if budget != nil {
		spentByCategory := map[string]int64{}
		for _, transaction := range transactions {
			if transaction.Kind != domain.Expense || transaction.Origin == domain.OriginAdjustment || transaction.InvoicePaymentID != "" || !strings.HasPrefix(transaction.OccurrenceDate, month+"-") {
				continue
			}
			budget.SpentCents += transaction.AmountCents
			if len(transaction.Splits) > 0 {
				for _, split := range transaction.Splits {
					spentByCategory[split.CategoryID] += split.AmountCents
				}
			} else {
				spentByCategory[transaction.CategoryID] += transaction.AmountCents
			}
		}
		budget.RemainingCents = budget.OverallLimitCents - budget.SpentCents
		if budget.OverallLimitCents > 0 {
			budget.ProgressPercent = float64(budget.SpentCents) / float64(budget.OverallLimitCents) * 100
		}
		for index := range budget.CategoryLimits {
			limit := &budget.CategoryLimits[index]
			if limit.Rollover {
				limit.RolloverCents, err = s.categoryRollover(ctx, month, limit.CategoryID, transactions)
				if err != nil {
					return domain.Planning{}, err
				}
			}
			limit.SpentCents = spentByCategory[limit.CategoryID]
			limit.AvailableCents = limit.LimitCents + limit.RolloverCents - limit.SpentCents
			limit.Exceeded = limit.AvailableCents < 0
		}
	}
	goals, err := s.store.Goals(ctx)
	if err != nil {
		return domain.Planning{}, err
	}
	for index := range goals {
		for _, allocation := range goals[index].Allocations {
			goals[index].AllocatedCents += allocation.AmountCents
		}
		if goals[index].TargetCents > 0 {
			goals[index].ProgressPercent = float64(goals[index].AllocatedCents) / float64(goals[index].TargetCents) * 100
		}
	}
	return domain.Planning{Budget: budget, Goals: goals}, nil
}

func (s *Service) categoryRollover(ctx context.Context, month, categoryID string, transactions []domain.Transaction) (int64, error) {
	current, _ := time.Parse("2006-01", month)
	return s.categoryCarryThroughMonth(ctx, current.AddDate(0, -1, 0), categoryID, transactions)
}

func (s *Service) categoryCarryThroughMonth(ctx context.Context, month time.Time, categoryID string, transactions []domain.Transaction) (int64, error) {
	budget, err := s.store.MonthlyBudget(ctx, month.Format("2006-01"))
	if err != nil || budget == nil {
		return 0, err
	}
	var limit *domain.CategoryBudgetLimit
	for index := range budget.CategoryLimits {
		if budget.CategoryLimits[index].CategoryID == categoryID {
			limit = &budget.CategoryLimits[index]
			break
		}
	}
	if limit == nil || !limit.Rollover {
		return 0, nil
	}
	prior, err := s.categoryCarryThroughMonth(ctx, month.AddDate(0, -1, 0), categoryID, transactions)
	if err != nil {
		return 0, err
	}
	spent := int64(0)
	prefix := month.Format("2006-01") + "-"
	for _, transaction := range transactions {
		if transaction.Kind != domain.Expense || transaction.Origin == domain.OriginAdjustment || transaction.InvoicePaymentID != "" || !strings.HasPrefix(transaction.OccurrenceDate, prefix) {
			continue
		}
		if len(transaction.Splits) == 0 && transaction.CategoryID == categoryID {
			spent += transaction.AmountCents
		}
		for _, split := range transaction.Splits {
			if split.CategoryID == categoryID {
				spent += split.AmountCents
			}
		}
	}
	return max(0, limit.LimitCents+prior-spent), nil
}

func validateGoalAllocationBalances(ctx context.Context, q storage.Queries, accounts []domain.Account, transactions []domain.Transaction) error {
	balances := make(map[string]int64, len(accounts))
	accountsByID := make(map[string]domain.Account, len(accounts))
	for _, account := range accounts {
		balances[account.ID] = account.OpeningBalanceCents
		accountsByID[account.ID] = account
	}
	for _, transaction := range transactions {
		domain.ApplyTransactionWithAccounts(balances, accountsByID, transaction)
	}
	goals, err := q.Goals(ctx)
	if err != nil {
		return err
	}
	reserved := map[string]int64{}
	for _, goal := range goals {
		if goal.ArchivedAt != "" {
			continue
		}
		for _, allocation := range goal.Allocations {
			reserved[allocation.AccountID] += allocation.AmountCents
		}
	}
	for accountID, amount := range reserved {
		if amount > max(balances[accountID], 0) {
			return domain.ErrAllocationLimit
		}
	}
	return nil
}

func validateLedgerState(ctx context.Context, q storage.Queries, accounts []domain.Account, transactions []domain.Transaction) error {
	if err := domain.ValidateSavingsBalances(accounts, transactions); err != nil {
		return err
	}
	return validateGoalAllocationBalances(ctx, q, accounts, transactions)
}

func (s *Service) SetMonthlyBudget(ctx context.Context, in MonthlyBudgetInput) (domain.MonthlyBudget, error) {
	if _, err := time.Parse("2006-01", in.ReferenceMonth); err != nil || in.OverallLimitCents < 0 {
		return domain.MonthlyBudget{}, domain.ErrInvalidBudget
	}
	budget := domain.MonthlyBudget{ReferenceMonth: in.ReferenceMonth, OverallLimitCents: in.OverallLimitCents, CategoryLimits: []domain.CategoryBudgetLimit{}}
	seen := map[string]bool{}
	for _, item := range in.CategoryLimits {
		if item.CategoryID == "" || item.LimitCents < 0 || seen[item.CategoryID] {
			return domain.MonthlyBudget{}, domain.ErrInvalidBudget
		}
		seen[item.CategoryID] = true
		category, err := s.store.Category(ctx, item.CategoryID)
		if err != nil || category == nil || category.Kind != domain.Expense || category.ArchivedAt != "" {
			return domain.MonthlyBudget{}, domain.ErrUnknownCategory
		}
		budget.CategoryLimits = append(budget.CategoryLimits, domain.CategoryBudgetLimit{ID: newID(), CategoryID: item.CategoryID, CategoryName: category.Name, LimitCents: item.LimitCents, Rollover: item.Rollover})
	}
	at := s.now().UTC().Format(time.RFC3339Nano)
	err := s.store.WithTx(ctx, func(q storage.Queries) error { return q.ReplaceMonthlyBudget(ctx, budget, at) })
	if err != nil {
		return domain.MonthlyBudget{}, err
	}
	planning, err := s.Planning(ctx, in.ReferenceMonth)
	if err != nil || planning.Budget == nil {
		return budget, err
	}
	return *planning.Budget, nil
}

func validateCategoryInput(in CategoryInput) error {
	if strings.TrimSpace(in.Name) == "" {
		return domain.ErrBlankName
	}
	if in.Kind != domain.Income && in.Kind != domain.Expense {
		return domain.ErrInvalidKind
	}
	return nil
}

func (s *Service) CreateCategory(ctx context.Context, in CategoryInput) (domain.Category, error) {
	if err := validateCategoryInput(in); err != nil {
		return domain.Category{}, err
	}
	category := domain.Category{ID: newID(), Name: strings.TrimSpace(in.Name), Kind: in.Kind, Editable: true}
	err := s.store.WithTx(ctx, func(q storage.Queries) error {
		existing, err := q.CategoryByName(ctx, category.Kind, category.Name)
		if err != nil {
			return err
		}
		if existing != nil {
			return domain.ErrDuplicateCategory
		}
		return q.InsertCategory(ctx, category)
	})
	return category, err
}

func (s *Service) RenameCategory(ctx context.Context, id string, in CategoryInput) (domain.Category, error) {
	if strings.TrimSpace(in.Name) == "" {
		return domain.Category{}, domain.ErrBlankName
	}
	var category domain.Category
	err := s.store.WithTx(ctx, func(q storage.Queries) error {
		current, err := q.Category(ctx, id)
		if err != nil {
			return err
		}
		if current == nil || !current.Editable {
			return domain.ErrUnknownCategory
		}
		existing, err := q.CategoryByName(ctx, current.Kind, in.Name)
		if err != nil {
			return err
		}
		if existing != nil && existing.ID != id {
			return domain.ErrDuplicateCategory
		}
		category = *current
		category.Name = strings.TrimSpace(in.Name)
		return q.UpdateCategory(ctx, category)
	})
	return category, err
}

func (s *Service) ArchiveCategory(ctx context.Context, id string) error {
	at := s.now().UTC().Format(time.RFC3339Nano)
	return s.store.WithTx(ctx, func(q storage.Queries) error {
		category, err := q.Category(ctx, id)
		if err != nil {
			return err
		}
		if category == nil || !category.Editable {
			return domain.ErrUnknownCategory
		}
		if category.ArchivedAt != "" {
			return nil
		}
		return q.SetCategoryArchivedAt(ctx, id, at)
	})
}

func (s *Service) RestoreCategory(ctx context.Context, id string) error {
	return s.store.WithTx(ctx, func(q storage.Queries) error {
		category, err := q.Category(ctx, id)
		if err != nil {
			return err
		}
		if category == nil || !category.Editable {
			return domain.ErrUnknownCategory
		}
		if category.ArchivedAt == "" {
			return nil
		}
		return q.SetCategoryArchivedAt(ctx, id, "")
	})
}

func validateGoalInput(in GoalInput) error {
	if strings.TrimSpace(in.Name) == "" || in.TargetCents < 0 || (in.Kind != domain.GoalEmergencyReserve && in.Kind != domain.GoalSavings) {
		return domain.ErrInvalidGoal
	}
	if in.Deadline != "" {
		if _, err := domain.ParseCivilDate(in.Deadline); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) SaveGoal(ctx context.Context, id string, in GoalInput) (domain.Goal, error) {
	if err := validateGoalInput(in); err != nil {
		return domain.Goal{}, err
	}
	at := s.now().UTC().Format(time.RFC3339Nano)
	goal := domain.Goal{ID: id, Name: strings.TrimSpace(in.Name), Kind: in.Kind, TargetCents: in.TargetCents, Deadline: in.Deadline, UpdatedAt: at}
	if id == "" {
		goal.ID, goal.CreatedAt = newID(), at
	}
	err := s.store.WithTx(ctx, func(q storage.Queries) error {
		if id == "" {
			return q.InsertGoal(ctx, goal, at)
		}
		return q.UpdateGoal(ctx, goal, at)
	})
	return goal, err
}

func (s *Service) ArchiveGoal(ctx context.Context, id string) error {
	at := s.now().UTC().Format(time.RFC3339Nano)
	return s.store.WithTx(ctx, func(q storage.Queries) error { return q.ArchiveGoal(ctx, id, at) })
}

func (s *Service) SetGoalAllocations(ctx context.Context, goalID string, inputs []GoalAllocationInput) (domain.Goal, error) {
	goal, err := s.store.Goal(ctx, goalID)
	if err != nil || goal == nil {
		if err != nil {
			return domain.Goal{}, err
		}
		return domain.Goal{}, domain.ErrUnknownGoal
	}
	allocations := make([]domain.GoalAllocation, 0, len(inputs))
	seen := map[string]bool{}
	for _, input := range inputs {
		if input.AmountCents < 0 || input.AccountID == "" || seen[input.AccountID] {
			return domain.Goal{}, domain.ErrInvalidGoal
		}
		seen[input.AccountID] = true
		account, err := s.store.Account(ctx, input.AccountID)
		if err != nil || account == nil || account.Type == domain.AccountCreditCard {
			return domain.Goal{}, domain.ErrUnknownAccount
		}
		allocations = append(allocations, domain.GoalAllocation{GoalID: goalID, AccountID: input.AccountID, AccountName: account.Name, AmountCents: input.AmountCents})
	}
	accounts, err := s.store.Accounts(ctx)
	if err != nil {
		return domain.Goal{}, err
	}
	transactions, err := s.store.Transactions(ctx)
	if err != nil {
		return domain.Goal{}, err
	}
	balances := make(map[string]int64, len(accounts))
	accountsByID := make(map[string]domain.Account, len(accounts))
	for _, account := range accounts {
		balances[account.ID] = account.OpeningBalanceCents
		accountsByID[account.ID] = account
	}
	for _, transaction := range transactions {
		domain.ApplyTransactionWithAccounts(balances, accountsByID, transaction)
	}
	goals, err := s.store.Goals(ctx)
	if err != nil {
		return domain.Goal{}, err
	}
	totals := map[string]int64{}
	for _, existing := range goals {
		if existing.ArchivedAt != "" || existing.ID == goalID {
			continue
		}
		for _, allocation := range existing.Allocations {
			totals[allocation.AccountID] += allocation.AmountCents
		}
	}
	for _, allocation := range allocations {
		totals[allocation.AccountID] += allocation.AmountCents
	}
	for accountID, total := range totals {
		maximum := balances[accountID]
		if maximum < 0 {
			maximum = 0
		}
		if total > maximum {
			return domain.Goal{}, domain.ErrAllocationLimit
		}
	}
	at := s.now().UTC().Format(time.RFC3339Nano)
	if err := s.store.WithTx(ctx, func(q storage.Queries) error { return q.ReplaceGoalAllocations(ctx, goalID, allocations, at) }); err != nil {
		return domain.Goal{}, err
	}
	goal.Allocations = allocations
	goal.AllocatedCents = 0
	for _, allocation := range allocations {
		goal.AllocatedCents += allocation.AmountCents
	}
	if goal.TargetCents > 0 {
		goal.ProgressPercent = float64(goal.AllocatedCents) / float64(goal.TargetCents) * 100
	}
	return *goal, nil
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
	if in.ReserveTargetCents < 0 {
		return domain.Profile{}, domain.ErrInvalidGoal
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
		if err := s.insertOpeningInvoice(ctx, q, a, in.FirstAccount, a.CreatedAt); err != nil {
			return err
		}
		if in.ReserveTargetCents > 0 {
			goal := domain.Goal{ID: newID(), Name: "Reserva de emergência", Kind: domain.GoalEmergencyReserve, TargetCents: in.ReserveTargetCents, CreatedAt: a.CreatedAt, UpdatedAt: a.CreatedAt}
			if err := q.InsertGoal(ctx, goal, a.CreatedAt); err != nil {
				return err
			}
		}
		return nil
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
		if len(linked) > 0 && in.OpeningBalanceCents != current.OpeningBalanceCents {
			return domain.ErrOpeningBalanceLocked
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
		if err := validateLedgerState(ctx, q, accounts, active); err != nil {
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

func (s *Service) AdjustAccountBalance(ctx context.Context, id string, in BalanceAdjustmentInput) (domain.Transaction, error) {
	now := s.now()
	at := now.UTC().Format(time.RFC3339Nano)
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		return domain.Transaction{}, domain.ErrAdjustmentReason
	}
	if _, err := domain.ParseCivilDate(in.OccurrenceDate); err != nil {
		return domain.Transaction{}, err
	}
	var adjustment domain.Transaction
	err := s.store.WithTx(ctx, func(q storage.Queries) error {
		account, err := q.Account(ctx, id)
		if err != nil {
			return err
		}
		if account == nil {
			return domain.ErrUnknownAccount
		}
		if account.Type == domain.AccountCreditCard {
			return domain.ErrCardTransaction
		}
		transactions, err := q.Transactions(ctx)
		if err != nil {
			return err
		}
		accounts, err := q.Accounts(ctx)
		if err != nil {
			return err
		}
		balances := make(map[string]int64, len(accounts))
		accountsByID := make(map[string]domain.Account, len(accounts))
		for _, item := range accounts {
			balances[item.ID] = item.OpeningBalanceCents
			accountsByID[item.ID] = item
		}
		for _, transaction := range transactions {
			domain.ApplyTransactionWithAccounts(balances, accountsByID, transaction)
		}
		difference := in.TargetBalanceCents - balances[id]
		if difference == 0 {
			return domain.ErrNoBalanceChange
		}
		kind, amount := domain.Income, difference
		if difference < 0 {
			kind, amount = domain.Expense, -difference
		}
		adjustment = domain.Transaction{ID: newID(), Kind: kind, AmountCents: amount, AccountID: id,
			Description: "Ajuste de saldo: " + reason, OccurrenceDate: in.OccurrenceDate, CreatedAt: at, UpdatedAt: at,
			Origin: domain.OriginAdjustment, AdjustmentReason: reason, InstallmentCount: 1}
		if err := s.prepareTransaction(ctx, q, &adjustment, now); err != nil {
			return err
		}
		if err := validateLedgerState(ctx, q, accounts, append(transactions, adjustment)); err != nil {
			return err
		}
		if err := q.InsertTransaction(ctx, adjustment, at); err != nil {
			return err
		}
		return q.InsertTransactionRevision(ctx, adjustment, "create", at)
	})
	return adjustment, err
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
	tx := domain.Transaction{ID: newID(), Kind: in.Kind, AmountCents: in.AmountCents, AccountID: in.AccountID, DestinationAccountID: in.DestinationAccountID, CategoryID: in.CategoryID, Description: strings.TrimSpace(in.Description), OccurrenceDate: in.OccurrenceDate, InstallmentCount: count, CreatedAt: at, UpdatedAt: at, Origin: domain.OriginManual, Tags: []domain.Tag{}, Splits: []domain.TransactionSplit{}}
	err := s.store.WithTx(ctx, func(q storage.Queries) error {
		account, err := q.Account(ctx, tx.AccountID)
		if err != nil || account == nil {
			if err != nil {
				return err
			}
			return domain.ErrUnknownAccount
		}
		if count > 1 && account.Type != domain.AccountCreditCard {
			if in.MonthlyRecurrence || len(in.Splits) > 0 {
				return domain.ErrInvalidInstallments
			}
			amounts, err := domain.InstallmentAmounts(in.AmountCents, count)
			if err != nil {
				return err
			}
			tx.AmountCents = amounts[0]
		}
		if err := requireActiveCategory(ctx, q, tx.CategoryID); err != nil {
			return err
		}
		if err := s.prepareTransactionDetails(ctx, q, &tx, in); err != nil {
			return err
		}
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
		if err := validateLedgerState(ctx, q, accounts, append(active, tx)); err != nil {
			return err
		}
		if err := q.InsertTransaction(ctx, tx, at); err != nil {
			return err
		}
		if err := q.SaveTransactionDetails(ctx, tx); err != nil {
			return err
		}
		if account != nil && account.Type == domain.AccountCreditCard {
			if err := s.insertPurchaseSchedule(ctx, q, *account, tx, at); err != nil {
				return err
			}
		} else if count > 1 {
			amounts, _ := domain.InstallmentAmounts(in.AmountCents, count)
			for index := 1; index < count; index++ {
				occurrence := domain.TransactionOccurrence{ID: newID(), AccountID: tx.AccountID, Kind: tx.Kind, CategoryID: tx.CategoryID, SubcategoryID: tx.SubcategoryID, AmountCents: amounts[index], Description: fmt.Sprintf("%s (%d/%d)", tx.Description, index+1, count), ScheduledDate: addMonthsClamped(tx.OccurrenceDate, index), Status: "pending", InstallmentNumber: index + 1, InstallmentCount: count, Tags: append([]domain.Tag{}, tx.Tags...), Splits: append([]domain.TransactionSplit{}, tx.Splits...)}
				if err := q.InsertTransactionOccurrence(ctx, occurrence, at); err != nil {
					return err
				}
			}
		}
		if in.MonthlyRecurrence {
			if tx.Kind == domain.Transfer || account.Type == domain.AccountCreditCard {
				return domain.ErrInvalidKind
			}
			ruleID := newID()
			tx.RecurrenceRuleID = ruleID
			if err := q.InsertRecurrenceRule(ctx, ruleID, tx, dateDay(tx.OccurrenceDate), at); err != nil {
				return err
			}
			if err := q.SetTransactionRecurrence(ctx, tx.ID, ruleID); err != nil {
				return err
			}
			for index := 1; index <= 12; index++ {
				occurrence := domain.TransactionOccurrence{ID: newID(), RecurrenceRuleID: ruleID, AccountID: tx.AccountID, Kind: tx.Kind, CategoryID: tx.CategoryID, SubcategoryID: tx.SubcategoryID, AmountCents: tx.AmountCents, Description: tx.Description, ScheduledDate: addMonthsClamped(tx.OccurrenceDate, index), Status: "pending", InstallmentNumber: 1, InstallmentCount: 1, Tags: append([]domain.Tag{}, tx.Tags...), Splits: append([]domain.TransactionSplit{}, tx.Splits...)}
				if err := q.InsertTransactionOccurrence(ctx, occurrence, at); err != nil {
					return err
				}
			}
		}
		return q.InsertTransactionRevision(ctx, tx, "create", at)
	})
	return tx, err
}

func (s *Service) prepareTransactionDetails(ctx context.Context, q storage.Queries, tx *domain.Transaction, in TransactionInput) error {
	if strings.TrimSpace(in.SubcategoryName) != "" {
		if tx.CategoryID == "" {
			return domain.ErrUnknownCategory
		}
		subcategory, err := q.EnsureSubcategory(ctx, tx.CategoryID, in.SubcategoryName)
		if err != nil {
			return err
		}
		tx.SubcategoryID, tx.SubcategoryName = subcategory.ID, subcategory.Name
	}
	seen := map[string]bool{}
	for _, name := range in.Tags {
		normalized := strings.ToLower(strings.TrimSpace(name))
		if normalized != "" && !seen[normalized] {
			seen[normalized] = true
			tx.Tags = append(tx.Tags, domain.Tag{Name: strings.TrimSpace(name)})
		}
	}
	if len(in.Splits) > 0 {
		if tx.Kind == domain.Transfer {
			return domain.ErrInvalidSplit
		}
		sum := int64(0)
		for _, item := range in.Splits {
			category, err := q.Category(ctx, item.CategoryID)
			if err != nil || category == nil || category.Kind != tx.Kind {
				return domain.ErrCategoryKind
			}
			if category.ArchivedAt != "" {
				return domain.ErrCategoryArchived
			}
			split := domain.TransactionSplit{ID: newID(), CategoryID: item.CategoryID, CategoryName: category.Name, AmountCents: item.AmountCents}
			if strings.TrimSpace(item.SubcategoryName) != "" {
				subcategory, err := q.EnsureSubcategory(ctx, item.CategoryID, item.SubcategoryName)
				if err != nil {
					return err
				}
				split.SubcategoryID, split.SubcategoryName = subcategory.ID, subcategory.Name
			}
			sum += item.AmountCents
			tx.Splits = append(tx.Splits, split)
		}
		if sum != tx.AmountCents {
			return domain.ErrInvalidSplit
		}
	}
	return nil
}

func addMonthsClamped(value string, months int) string {
	date, _ := domain.ParseCivilDate(value)
	targetMonth := date.Month() + time.Month(months)
	last := time.Date(date.Year(), targetMonth+1, 0, 0, 0, 0, 0, time.Local).Day()
	day := date.Day()
	if day > last {
		day = last
	}
	return time.Date(date.Year(), targetMonth, day, 0, 0, 0, 0, time.Local).Format("2006-01-02")
}
func dateDay(value string) int { date, _ := domain.ParseCivilDate(value); return date.Day() }

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
			tx := domain.Transaction{ID: newID(), Kind: entry.Kind, AmountCents: entry.AmountCents, AccountID: accountID, Description: strings.TrimSpace(entry.Description), OccurrenceDate: entry.Date, CreatedAt: at, UpdatedAt: at, AutomaticImport: true, ImportBank: string(bank), ImportKey: key, Origin: domain.OriginImport}
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
		if err := validateLedgerState(ctx, q, accounts, append(active, pending...)); err != nil {
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
		updated.SubcategoryID, updated.SubcategoryName = "", ""
		updated.Tags = []domain.Tag{}
		updated.Splits = []domain.TransactionSplit{}
		if err := requireActiveCategory(ctx, q, updated.CategoryID); err != nil {
			return err
		}
		if err := s.prepareTransactionDetails(ctx, q, &updated, in); err != nil {
			return err
		}
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
		if err := validateLedgerState(ctx, q, accounts, active); err != nil {
			return err
		}
		if err := q.UpdateTransaction(ctx, updated, at); err != nil {
			return err
		}
		if err := q.SaveTransactionDetails(ctx, updated); err != nil {
			return err
		}
		if current.RecurrenceRuleID != "" {
			if !in.MonthlyRecurrence {
				if err := q.ArchiveRecurrenceRule(ctx, current.RecurrenceRuleID, at); err != nil {
					return err
				}
				if err := q.SetTransactionRecurrence(ctx, updated.ID, ""); err != nil {
					return err
				}
				updated.RecurrenceRuleID = ""
			} else {
				if err := q.UpdateRecurrenceRule(ctx, current.RecurrenceRuleID, updated, dateDay(updated.OccurrenceDate), at); err != nil {
					return err
				}
				if err := q.DeletePendingRecurrenceOccurrences(ctx, current.RecurrenceRuleID); err != nil {
					return err
				}
				existing, err := q.TransactionOccurrences(ctx)
				if err != nil {
					return err
				}
				dates := map[string]bool{}
				for _, occurrence := range existing {
					if occurrence.RecurrenceRuleID == current.RecurrenceRuleID {
						dates[occurrence.ScheduledDate] = true
					}
				}
				for index := 1; index <= 12; index++ {
					date := addMonthsClamped(updated.OccurrenceDate, index)
					if dates[date] {
						continue
					}
					occurrence := domain.TransactionOccurrence{ID: newID(), RecurrenceRuleID: current.RecurrenceRuleID, AccountID: updated.AccountID, Kind: updated.Kind, CategoryID: updated.CategoryID, SubcategoryID: updated.SubcategoryID, AmountCents: updated.AmountCents, Description: updated.Description, ScheduledDate: date, Status: "pending", InstallmentNumber: 1, InstallmentCount: 1, Tags: append([]domain.Tag{}, updated.Tags...), Splits: append([]domain.TransactionSplit{}, updated.Splits...)}
					if err := q.InsertTransactionOccurrence(ctx, occurrence, at); err != nil {
						return err
					}
				}
			}
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

func requireActiveCategory(ctx context.Context, q storage.Queries, id string) error {
	if id == "" {
		return nil
	}
	category, err := q.Category(ctx, id)
	if err != nil {
		return err
	}
	if category == nil {
		return domain.ErrUnknownCategory
	}
	if category.ArchivedAt != "" {
		return domain.ErrCategoryArchived
	}
	return nil
}

func (s *Service) TransactionOccurrences(ctx context.Context) ([]domain.TransactionOccurrence, error) {
	if err := s.ensureRecurringOccurrences(ctx, s.now()); err != nil {
		return nil, err
	}
	return s.store.TransactionOccurrences(ctx)
}

func (s *Service) ensureRecurringOccurrences(ctx context.Context, now time.Time) error {
	at := now.UTC().Format(time.RFC3339Nano)
	horizon := addMonthsClamped(now.In(time.Local).Format("2006-01-02"), 12)
	return s.store.WithTx(ctx, func(q storage.Queries) error {
		rules, err := q.ActiveRecurrenceRules(ctx)
		if err != nil {
			return err
		}
		occurrences, err := q.TransactionOccurrences(ctx)
		if err != nil {
			return err
		}
		latest := map[string]domain.TransactionOccurrence{}
		for _, occurrence := range occurrences {
			if occurrence.RecurrenceRuleID != "" && occurrence.ScheduledDate > latest[occurrence.RecurrenceRuleID].ScheduledDate {
				latest[occurrence.RecurrenceRuleID] = occurrence
			}
		}
		for _, rule := range rules {
			source, err := q.TransactionForRecurrence(ctx, rule.ID)
			if err != nil {
				return err
			}
			last, found := latest[rule.ID]
			if !found {
				if source == nil {
					continue
				}
				last = domain.TransactionOccurrence{RecurrenceRuleID: rule.ID, AccountID: source.AccountID, Kind: source.Kind, CategoryID: source.CategoryID, SubcategoryID: source.SubcategoryID, AmountCents: source.AmountCents, Description: source.Description, ScheduledDate: source.OccurrenceDate, InstallmentNumber: 1, InstallmentCount: 1, Tags: source.Tags, Splits: source.Splits}
			}
			for next := nextMonthlyDate(last.ScheduledDate, rule.DayOfMonth); next <= horizon; next = nextMonthlyDate(last.ScheduledDate, rule.DayOfMonth) {
				item := last
				item.ID, item.ScheduledDate, item.Status, item.TransactionID = newID(), next, "pending", ""
				item.CreatedAt, item.UpdatedAt = at, at
				if source != nil {
					item.Tags = append([]domain.Tag{}, source.Tags...)
					item.Splits = append([]domain.TransactionSplit{}, source.Splits...)
				}
				if err := q.InsertTransactionOccurrence(ctx, item, at); err != nil {
					return err
				}
				last = item
			}
		}
		return nil
	})
}

func nextMonthlyDate(date string, day int) string {
	current, err := domain.ParseCivilDate(date)
	if err != nil {
		return date
	}
	month := time.Date(current.Year(), current.Month()+1, 1, 0, 0, 0, 0, time.Local)
	lastDay := time.Date(month.Year(), month.Month()+1, 0, 0, 0, 0, 0, time.Local).Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(month.Year(), month.Month(), day, 0, 0, 0, 0, time.Local).Format("2006-01-02")
}

func (s *Service) SearchTransactions(ctx context.Context, filter domain.TransactionFilter) ([]domain.Transaction, error) {
	active, err := s.store.Transactions(ctx)
	if err != nil {
		return nil, err
	}
	trashed, err := s.store.TrashedTransactions(ctx)
	if err != nil {
		return nil, err
	}
	occurrences, err := s.store.TransactionOccurrences(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]domain.Transaction, 0, len(active)+len(trashed)+len(occurrences))
	for _, tx := range active {
		tx.Status = "active"
		items = append(items, tx)
	}
	for _, tx := range trashed {
		tx.Status = "trashed"
		items = append(items, tx)
	}
	for _, x := range occurrences {
		if x.Status != "pending" {
			continue
		}
		items = append(items, domain.Transaction{ID: x.ID, Kind: x.Kind, AmountCents: x.AmountCents, AccountID: x.AccountID, AccountName: x.AccountName, CategoryID: x.CategoryID, CategoryName: x.CategoryName, SubcategoryID: x.SubcategoryID, Description: x.Description, OccurrenceDate: x.ScheduledDate, CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt, InstallmentCount: x.InstallmentCount, RecurrenceRuleID: x.RecurrenceRuleID, Status: "pending", Pending: true, Tags: x.Tags, Splits: x.Splits})
	}
	match := func(tx domain.Transaction) bool {
		if filter.Status != "" && filter.Status != "all" && tx.Status != filter.Status {
			return false
		}
		if filter.StartDate != "" && tx.OccurrenceDate < filter.StartDate {
			return false
		}
		if filter.EndDate != "" && tx.OccurrenceDate > filter.EndDate {
			return false
		}
		if filter.AccountID != "" && tx.AccountID != filter.AccountID && tx.DestinationAccountID != filter.AccountID {
			return false
		}
		if filter.CategoryID != "" && tx.CategoryID != filter.CategoryID {
			found := false
			for _, split := range tx.Splits {
				if split.CategoryID == filter.CategoryID {
					found = true
				}
			}
			if !found {
				return false
			}
		}
		if filter.SubcategoryID != "" && tx.SubcategoryID != filter.SubcategoryID {
			return false
		}
		if filter.Kind != "" && tx.Kind != filter.Kind {
			return false
		}
		if filter.MinimumAmountCents > 0 && tx.AmountCents < filter.MinimumAmountCents {
			return false
		}
		if filter.MaximumAmountCents > 0 && tx.AmountCents > filter.MaximumAmountCents {
			return false
		}
		if filter.Recurrence == "recurring" && tx.RecurrenceRuleID == "" {
			return false
		}
		if filter.Recurrence == "nonrecurring" && tx.RecurrenceRuleID != "" {
			return false
		}
		needle := strings.ToLower(strings.TrimSpace(filter.Text))
		if needle != "" && !strings.Contains(strings.ToLower(tx.Description+" "+tx.AccountName+" "+tx.CategoryName+" "+tx.SubcategoryName), needle) {
			return false
		}
		tagNeedle := strings.ToLower(strings.TrimSpace(filter.Tag))
		if tagNeedle != "" {
			found := false
			for _, tag := range tx.Tags {
				if strings.EqualFold(tag.Name, tagNeedle) {
					found = true
				}
			}
			if !found {
				return false
			}
		}
		return true
	}
	result := []domain.Transaction{}
	for _, tx := range items {
		if match(tx) {
			result = append(result, tx)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].OccurrenceDate == result[j].OccurrenceDate {
			return result[i].CreatedAt > result[j].CreatedAt
		}
		return result[i].OccurrenceDate > result[j].OccurrenceDate
	})
	return result, nil
}
func (s *Service) ConfirmTransactionOccurrence(ctx context.Context, id string) (domain.Transaction, error) {
	now := s.now()
	at := now.UTC().Format(time.RFC3339Nano)
	var tx domain.Transaction
	err := s.store.WithTx(ctx, func(q storage.Queries) error {
		x, err := q.TransactionOccurrence(ctx, id)
		if err != nil {
			return err
		}
		if x == nil {
			return domain.ErrUnknownLedgerOccurrence
		}
		if x.Status != "pending" {
			return domain.ErrOccurrenceClosed
		}
		tags, splits := copyOccurrenceDetails(x.Tags, x.Splits)
		tx = domain.Transaction{ID: newID(), Kind: x.Kind, AmountCents: x.AmountCents, AccountID: x.AccountID, CategoryID: x.CategoryID, SubcategoryID: x.SubcategoryID, Description: x.Description, OccurrenceDate: x.ScheduledDate, InstallmentCount: 1, CreatedAt: at, UpdatedAt: at, Origin: domain.OriginManual, RecurrenceRuleID: x.RecurrenceRuleID, Tags: tags, Splits: splits}
		if x.RecurrenceRuleID != "" && len(tx.Tags) == 0 && len(tx.Splits) == 0 {
			source, err := q.TransactionForRecurrence(ctx, x.RecurrenceRuleID)
			if err != nil {
				return err
			}
			if source != nil {
				tx.Tags, tx.Splits = copyOccurrenceDetails(source.Tags, source.Splits)
			}
		}
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
		if err := validateLedgerState(ctx, q, accounts, append(active, tx)); err != nil {
			return err
		}
		if err := q.InsertTransaction(ctx, tx, at); err != nil {
			return err
		}
		if err := q.SaveTransactionDetails(ctx, tx); err != nil {
			return err
		}
		if err := q.SetTransactionOccurrence(ctx, id, "confirmed", tx.ID, at); err != nil {
			return err
		}
		return q.InsertTransactionRevision(ctx, tx, "create", at)
	})
	return tx, err
}

func copyOccurrenceDetails(tags []domain.Tag, splits []domain.TransactionSplit) ([]domain.Tag, []domain.TransactionSplit) {
	clonedTags := append([]domain.Tag{}, tags...)
	clonedSplits := append([]domain.TransactionSplit{}, splits...)
	for index := range clonedSplits {
		clonedSplits[index].ID = newID()
	}
	return clonedTags, clonedSplits
}
func (s *Service) DismissTransactionOccurrence(ctx context.Context, id string) error {
	at := s.now().UTC().Format(time.RFC3339Nano)
	return s.store.WithTx(ctx, func(q storage.Queries) error {
		x, err := q.TransactionOccurrence(ctx, id)
		if err != nil {
			return err
		}
		if x == nil {
			return domain.ErrUnknownLedgerOccurrence
		}
		if x.Status != "pending" {
			return domain.ErrOccurrenceClosed
		}
		return q.SetTransactionOccurrence(ctx, id, "dismissed", "", at)
	})
}
func (s *Service) ArchiveRecurrence(ctx context.Context, id string) error {
	at := s.now().UTC().Format(time.RFC3339Nano)
	return s.store.WithTx(ctx, func(q storage.Queries) error { return q.ArchiveRecurrenceRule(ctx, id, at) })
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
		if err := validateLedgerState(ctx, q, accounts, remaining); err != nil {
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
		if err := validateLedgerState(ctx, q, accounts, append(active, restored)); err != nil {
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
			OccurrenceDate: in.OccurrenceDate, InstallmentCount: 1, InvoicePaymentID: payment.ID, CreatedAt: at, UpdatedAt: at, Origin: domain.OriginCardPayment}
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
		if err := validateLedgerState(ctx, q, accounts, append(active, tx)); err != nil {
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
		tx = domain.Transaction{ID: newID(), Kind: domain.Expense, AmountCents: in.AmountCents, AccountID: occurrence.AccountID, CategoryID: occurrence.CategoryID, Description: occurrence.Description, OccurrenceDate: in.OccurrenceDate, FixedExpenseOccurrenceID: occurrence.ID, CreatedAt: at, UpdatedAt: at, Origin: domain.OriginFixedExpense}
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
		if err := validateLedgerState(ctx, q, accounts, append(active, tx)); err != nil {
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
