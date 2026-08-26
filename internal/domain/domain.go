// Package domain contains the financial rules used by the application.
package domain

import (
	"errors"
	"sort"
	"strings"
	"time"
)

type Theme string

const (
	ThemeLight  Theme = "light"
	ThemeDark   Theme = "dark"
	ThemeGothic Theme = "gothic"
)

type AccountType string

const (
	AccountChecking   AccountType = "checking"
	AccountSavings    AccountType = "savings"
	AccountCash       AccountType = "cash"
	AccountCreditCard AccountType = "credit_card"
)

type TransactionKind string

const (
	Income   TransactionKind = "income"
	Expense  TransactionKind = "expense"
	Transfer TransactionKind = "transfer"
)

var (
	ErrBlankName             = errors.New("blank name")
	ErrBlankDescription      = errors.New("blank description")
	ErrInvalidAmount         = errors.New("amount must be positive")
	ErrInvalidDate           = errors.New("invalid civil date")
	ErrFutureDate            = errors.New("date is in the future")
	ErrBeforeOpening         = errors.New("date is before account opening")
	ErrInvalidAccountType    = errors.New("invalid account type")
	ErrInvalidKind           = errors.New("invalid transaction kind")
	ErrInvalidTheme          = errors.New("invalid theme")
	ErrUnknownAccount        = errors.New("unknown account")
	ErrAccountInUse          = errors.New("account is in use")
	ErrUnknownCategory       = errors.New("unknown category")
	ErrCategoryKind          = errors.New("category kind does not match transaction")
	ErrSameTransferAccount   = errors.New("transfer accounts must be distinct")
	ErrTransferCategory      = errors.New("transfer cannot have category")
	ErrSavingsNegative       = errors.New("savings account cannot be negative")
	ErrUnknownTransaction    = errors.New("unknown transaction")
	ErrTransactionActive     = errors.New("transaction is already active")
	ErrTransactionTrashed    = errors.New("transaction is already trashed")
	ErrInvalidDueDay         = errors.New("invalid due day")
	ErrUnknownFixedExpense   = errors.New("unknown fixed expense")
	ErrUnknownOccurrence     = errors.New("unknown fixed expense occurrence")
	ErrOccurrenceClosed      = errors.New("fixed expense occurrence is not pending")
	ErrFixedExpenseArchived  = errors.New("fixed expense is archived")
	ErrInvalidStatement      = errors.New("invalid bank statement")
	ErrUnsupportedBank       = errors.New("unsupported bank")
	ErrStatementEmpty        = errors.New("bank statement has no transactions")
	ErrStatementTooLarge     = errors.New("bank statement is too large")
	ErrInvalidCreditLimit    = errors.New("invalid credit limit")
	ErrInvalidInstallments   = errors.New("invalid installment count")
	ErrCardTransaction       = errors.New("invalid credit card transaction")
	ErrInvoiceLocked         = errors.New("credit card invoice is locked")
	ErrUnknownInvoice        = errors.New("unknown credit card invoice")
	ErrInvoiceNotPayable     = errors.New("credit card invoice is not payable")
	ErrInvalidPaymentAccount = errors.New("invalid invoice payment account")
	ErrInvoiceOverpayment    = errors.New("invoice payment exceeds outstanding amount")
	ErrOpeningBalanceLocked  = errors.New("opening balance is locked after ledger activity")
	ErrAdjustmentReason      = errors.New("adjustment reason is required")
	ErrNoBalanceChange       = errors.New("target balance equals current balance")
	ErrInvalidBudget         = errors.New("invalid budget")
	ErrInvalidGoal           = errors.New("invalid goal")
	ErrUnknownGoal           = errors.New("unknown goal")
	ErrAllocationLimit       = errors.New("goal allocations exceed account balance")
)

type TransactionOrigin string

const (
	OriginManual       TransactionOrigin = "manual"
	OriginImport       TransactionOrigin = "import"
	OriginFixedExpense TransactionOrigin = "fixed_expense"
	OriginCardPayment  TransactionOrigin = "card_payment"
	OriginAdjustment   TransactionOrigin = "adjustment"
)

type Profile struct {
	DisplayName      string `json:"displayName"`
	Currency         string `json:"currency"`
	Theme            Theme  `json:"theme"`
	OnboardingStatus string `json:"onboardingStatus"`
	BalancesHidden   bool   `json:"balancesHidden"`
}

type Account struct {
	ID                  string      `json:"id"`
	Name                string      `json:"name"`
	Type                AccountType `json:"type"`
	OpeningBalanceCents int64       `json:"openingBalanceCents"`
	OpeningDate         string      `json:"openingDate"`
	CreatedAt           string      `json:"createdAt"`
	CurrentBalanceCents int64       `json:"currentBalanceCents"`
	CreditLimitCents    int64       `json:"creditLimitCents,omitempty"`
	ClosingDay          int         `json:"closingDay,omitempty"`
	DueDay              int         `json:"dueDay,omitempty"`
	HasLedgerActivity   bool        `json:"hasLedgerActivity"`
}

type Category struct {
	ID   string          `json:"id"`
	Name string          `json:"name"`
	Kind TransactionKind `json:"kind"`
}

type Transaction struct {
	ID                       string            `json:"id"`
	Kind                     TransactionKind   `json:"kind"`
	AmountCents              int64             `json:"amountCents"`
	AccountID                string            `json:"accountId"`
	AccountName              string            `json:"accountName"`
	DestinationAccountID     string            `json:"destinationAccountId,omitempty"`
	DestinationAccountName   string            `json:"destinationAccountName,omitempty"`
	CategoryID               string            `json:"categoryId,omitempty"`
	CategoryName             string            `json:"categoryName,omitempty"`
	Description              string            `json:"description"`
	OccurrenceDate           string            `json:"occurrenceDate"`
	CreatedAt                string            `json:"createdAt"`
	UpdatedAt                string            `json:"updatedAt"`
	DeletedAt                string            `json:"deletedAt,omitempty"`
	FixedExpenseOccurrenceID string            `json:"fixedExpenseOccurrenceId,omitempty"`
	AutomaticImport          bool              `json:"automaticImport"`
	ImportBank               string            `json:"importBank,omitempty"`
	ImportKey                string            `json:"-"`
	InstallmentCount         int               `json:"installmentCount,omitempty"`
	InvoicePaymentID         string            `json:"invoicePaymentId,omitempty"`
	Origin                   TransactionOrigin `json:"origin"`
	AdjustmentReason         string            `json:"adjustmentReason,omitempty"`
}

type CreditCardInvoiceStatus string

const (
	InvoiceOpen       CreditCardInvoiceStatus = "open"
	InvoiceClosed     CreditCardInvoiceStatus = "closed"
	InvoicePaid       CreditCardInvoiceStatus = "paid"
	InvoiceRolledOver CreditCardInvoiceStatus = "rolled_over"
)

type CreditCardInvoice struct {
	ID                string                  `json:"id"`
	AccountID         string                  `json:"accountId"`
	AccountName       string                  `json:"accountName"`
	ReferenceMonth    string                  `json:"referenceMonth"`
	ClosingDate       string                  `json:"closingDate"`
	DueDate           string                  `json:"dueDate"`
	Status            CreditCardInvoiceStatus `json:"status"`
	ChargesCents      int64                   `json:"chargesCents"`
	CarryForwardCents int64                   `json:"carryForwardCents"`
	PaidCents         int64                   `json:"paidCents"`
	OutstandingCents  int64                   `json:"outstandingCents"`
	Installments      []CreditCardInstallment `json:"installments"`
	Payments          []CreditCardPayment     `json:"payments"`
}

type CreditCardInstallment struct {
	ID                string `json:"id"`
	InvoiceID         string `json:"invoiceId"`
	TransactionID     string `json:"transactionId,omitempty"`
	Description       string `json:"description"`
	AmountCents       int64  `json:"amountCents"`
	InstallmentNumber int    `json:"installmentNumber"`
	InstallmentCount  int    `json:"installmentCount"`
	OpeningDebt       bool   `json:"openingDebt"`
}

type CreditCardPayment struct {
	ID             string `json:"id"`
	InvoiceID      string `json:"invoiceId"`
	AccountID      string `json:"accountId"`
	AccountName    string `json:"accountName"`
	TransactionID  string `json:"transactionId"`
	AmountCents    int64  `json:"amountCents"`
	OccurrenceDate string `json:"occurrenceDate"`
	CreatedAt      string `json:"createdAt"`
}

type CreditCardSummary struct {
	Account             Account            `json:"account"`
	OutstandingCents    int64              `json:"outstandingCents"`
	AvailableLimitCents int64              `json:"availableLimitCents"`
	CurrentInvoice      *CreditCardInvoice `json:"currentInvoice,omitempty"`
}

type FixedExpense struct {
	ID                string `json:"id"`
	Description       string `json:"description"`
	AmountCents       int64  `json:"amountCents"`
	DueDay            int    `json:"dueDay"`
	AccountID         string `json:"accountId"`
	AccountName       string `json:"accountName"`
	CategoryID        string `json:"categoryId"`
	CategoryName      string `json:"categoryName"`
	ArchivedAt        string `json:"archivedAt,omitempty"`
	OccurrenceStartAt string `json:"-"`
	CreatedAt         string `json:"createdAt"`
	UpdatedAt         string `json:"updatedAt"`
}

type FixedExpenseOccurrenceStatus string

const (
	FixedExpensePending   FixedExpenseOccurrenceStatus = "pending"
	FixedExpenseConfirmed FixedExpenseOccurrenceStatus = "confirmed"
	FixedExpenseDismissed FixedExpenseOccurrenceStatus = "dismissed"
)

type FixedExpenseOccurrence struct {
	ID                  string                       `json:"id"`
	FixedExpenseID      string                       `json:"fixedExpenseId"`
	ReferenceMonth      string                       `json:"referenceMonth"`
	DueDate             string                       `json:"dueDate"`
	Description         string                       `json:"description"`
	ExpectedAmountCents int64                        `json:"expectedAmountCents"`
	AccountID           string                       `json:"accountId"`
	AccountName         string                       `json:"accountName"`
	CategoryID          string                       `json:"categoryId"`
	CategoryName        string                       `json:"categoryName"`
	Status              FixedExpenseOccurrenceStatus `json:"status"`
	TransactionID       string                       `json:"transactionId,omitempty"`
	CreatedAt           string                       `json:"createdAt"`
	UpdatedAt           string                       `json:"updatedAt"`
}

type BalanceHistoryPoint struct {
	Month        string `json:"month"`
	Label        string `json:"label"`
	BalanceCents int64  `json:"balanceCents"`
}

type AccountAllocation struct {
	AccountID    string `json:"accountId"`
	AccountName  string `json:"accountName"`
	BalanceCents int64  `json:"balanceCents"`
}

type Dashboard struct {
	AvailableBalanceCents     int64                 `json:"availableBalanceCents"`
	TotalBalanceCents         int64                 `json:"totalBalanceCents"`
	PendingFixedExpensesCents int64                 `json:"pendingFixedExpensesCents"`
	PendingFixedExpenseCount  int                   `json:"pendingFixedExpenseCount"`
	MonthlyIncomeCents        int64                 `json:"monthlyIncomeCents"`
	MonthlyExpenseCents       int64                 `json:"monthlyExpenseCents"`
	RecentTransactions        []Transaction         `json:"recentTransactions"`
	BalanceHistory            []BalanceHistoryPoint `json:"balanceHistory"`
	AccountAllocations        []AccountAllocation   `json:"accountAllocations"`
	HasNegativeBalance        bool                  `json:"hasNegativeBalance"`
	CreditCardDebtCents       int64                 `json:"creditCardDebtCents"`
	UpcomingInvoices          []CreditCardInvoice   `json:"upcomingInvoices"`
	ReservedValueCents        int64                 `json:"reservedValueCents"`
	EligibleBalanceCents      int64                 `json:"eligibleBalanceCents"`
	FreeValueCents            int64                 `json:"freeValueCents"`
	SafelySpendableCents      int64                 `json:"safelySpendableCents"`
	BudgetProgressPercent     float64               `json:"budgetProgressPercent"`
	GoalProgressPercent       float64               `json:"goalProgressPercent"`
}

type CategoryBudgetLimit struct {
	ID             string `json:"id"`
	CategoryID     string `json:"categoryId"`
	CategoryName   string `json:"categoryName"`
	LimitCents     int64  `json:"limitCents"`
	Rollover       bool   `json:"rollover"`
	RolloverCents  int64  `json:"rolloverCents"`
	SpentCents     int64  `json:"spentCents"`
	AvailableCents int64  `json:"availableCents"`
	Exceeded       bool   `json:"exceeded"`
}

type MonthlyBudget struct {
	ReferenceMonth    string                `json:"referenceMonth"`
	OverallLimitCents int64                 `json:"overallLimitCents"`
	SpentCents        int64                 `json:"spentCents"`
	RemainingCents    int64                 `json:"remainingCents"`
	ProgressPercent   float64               `json:"progressPercent"`
	CategoryLimits    []CategoryBudgetLimit `json:"categoryLimits"`
}

type GoalKind string

const (
	GoalEmergencyReserve GoalKind = "emergency_reserve"
	GoalSavings          GoalKind = "savings"
)

type GoalAllocation struct {
	GoalID      string `json:"goalId"`
	AccountID   string `json:"accountId"`
	AccountName string `json:"accountName"`
	AmountCents int64  `json:"amountCents"`
}

type Goal struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Kind            GoalKind         `json:"kind"`
	TargetCents     int64            `json:"targetCents"`
	Deadline        string           `json:"deadline,omitempty"`
	ArchivedAt      string           `json:"archivedAt,omitempty"`
	CreatedAt       string           `json:"createdAt"`
	UpdatedAt       string           `json:"updatedAt"`
	AllocatedCents  int64            `json:"allocatedCents"`
	ProgressPercent float64          `json:"progressPercent"`
	Allocations     []GoalAllocation `json:"allocations"`
}

type Planning struct {
	Budget *MonthlyBudget `json:"budget,omitempty"`
	Goals  []Goal         `json:"goals"`
}

func ParseCivilDate(value string) (time.Time, error) {
	date, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil || date.Format("2006-01-02") != value {
		return time.Time{}, ErrInvalidDate
	}
	return date, nil
}

func ValidateTheme(theme Theme) error {
	switch theme {
	case ThemeLight, ThemeDark, ThemeGothic:
		return nil
	default:
		return ErrInvalidTheme
	}
}

func ValidateAccount(name string, accountType AccountType, openingDate string, now time.Time) error {
	if strings.TrimSpace(name) == "" {
		return ErrBlankName
	}
	if accountType != AccountChecking && accountType != AccountSavings && accountType != AccountCash && accountType != AccountCreditCard {
		return ErrInvalidAccountType
	}
	date, err := ParseCivilDate(openingDate)
	if err != nil {
		return err
	}
	today, _ := ParseCivilDate(now.In(time.Local).Format("2006-01-02"))
	if date.After(today) {
		return ErrFutureDate
	}
	return nil
}

func ValidateCreditCard(account Account) error {
	if account.Type != AccountCreditCard {
		return nil
	}
	if account.CreditLimitCents <= 0 {
		return ErrInvalidCreditLimit
	}
	if account.ClosingDay < 1 || account.ClosingDay > 31 || account.DueDay < 1 || account.DueDay > 31 {
		return ErrInvalidDueDay
	}
	return nil
}

func ValidateFixedExpense(description string, amountCents int64, dueDay int, category *Category) error {
	if strings.TrimSpace(description) == "" {
		return ErrBlankDescription
	}
	if amountCents <= 0 {
		return ErrInvalidAmount
	}
	if dueDay < 1 || dueDay > 31 {
		return ErrInvalidDueDay
	}
	if category == nil {
		return ErrUnknownCategory
	}
	if category.Kind != Expense {
		return ErrCategoryKind
	}
	return nil
}

func ValidateTransaction(tx Transaction, account Account, destination *Account, category *Category, now time.Time) error {
	if tx.Kind != Income && tx.Kind != Expense && tx.Kind != Transfer {
		return ErrInvalidKind
	}
	if tx.AmountCents <= 0 {
		return ErrInvalidAmount
	}
	if tx.InstallmentCount == 0 {
		tx.InstallmentCount = 1
	}
	if tx.InstallmentCount < 1 || tx.InstallmentCount > 48 {
		return ErrInvalidInstallments
	}
	if account.Type == AccountCreditCard {
		if tx.Kind != Expense || destination != nil || tx.InvoicePaymentID != "" {
			return ErrCardTransaction
		}
	} else if tx.InstallmentCount != 1 {
		return ErrInvalidInstallments
	}
	if tx.Kind != Transfer && strings.TrimSpace(tx.Description) == "" {
		return ErrBlankDescription
	}
	date, err := ParseCivilDate(tx.OccurrenceDate)
	if err != nil {
		return err
	}
	today, _ := ParseCivilDate(now.In(time.Local).Format("2006-01-02"))
	if date.After(today) {
		return ErrFutureDate
	}
	opening, err := ParseCivilDate(account.OpeningDate)
	if err != nil {
		return err
	}
	if date.Before(opening) && !tx.AutomaticImport {
		return ErrBeforeOpening
	}
	if tx.Kind == Transfer {
		if destination == nil {
			return ErrUnknownAccount
		}
		if account.ID == destination.ID {
			return ErrSameTransferAccount
		}
		if (account.Type == AccountCreditCard || destination.Type == AccountCreditCard) && tx.InvoicePaymentID == "" {
			return ErrCardTransaction
		}
		destinationOpening, err := ParseCivilDate(destination.OpeningDate)
		if err != nil {
			return err
		}
		if date.Before(destinationOpening) && !tx.AutomaticImport {
			return ErrBeforeOpening
		}
		if category != nil || tx.CategoryID != "" {
			return ErrTransferCategory
		}
		return nil
	}
	if category != nil && category.Kind != tx.Kind {
		return ErrCategoryKind
	}
	return nil
}

// CreditCardCycle returns the closing and due dates of the invoice that receives
// a purchase. The closing day itself is treated as already closed.
func CreditCardCycle(purchase time.Time, closingDay, dueDay int) (time.Time, time.Time) {
	location := purchase.Location()
	closing := civilDay(purchase.Year(), purchase.Month(), closingDay, location)
	if !purchase.Before(closing) {
		closing = civilDay(purchase.Year(), purchase.Month()+1, closingDay, location)
	}
	dueMonth := closing.Month()
	dueYear := closing.Year()
	if dueDay <= closingDay {
		dueMonth++
	}
	due := civilDay(dueYear, dueMonth, dueDay, location)
	return closing, due
}

func civilDay(year int, month time.Month, day int, location *time.Location) time.Time {
	last := time.Date(year, month+1, 0, 0, 0, 0, 0, location).Day()
	if day > last {
		day = last
	}
	return time.Date(year, month, day, 0, 0, 0, 0, location)
}

func InstallmentAmounts(total int64, count int) ([]int64, error) {
	if total <= 0 {
		return nil, ErrInvalidAmount
	}
	if count < 1 || count > 48 {
		return nil, ErrInvalidInstallments
	}
	items := make([]int64, count)
	base, remainder := total/int64(count), total%int64(count)
	for index := range items {
		items[index] = base
	}
	items[0] += remainder
	return items, nil
}

func SignedAmount(kind TransactionKind, cents int64) int64 {
	if kind == Expense {
		return -cents
	}
	if kind == Transfer {
		return 0
	}
	return cents
}

// ApplyTransaction adds one active transaction to per-account balances.
func ApplyTransaction(balances map[string]int64, tx Transaction) {
	if tx.Kind == Transfer {
		balances[tx.AccountID] -= tx.AmountCents
		balances[tx.DestinationAccountID] += tx.AmountCents
		return
	}
	balances[tx.AccountID] += SignedAmount(tx.Kind, tx.AmountCents)
}

// TransactionAffectsBalance reports whether a transaction belongs after the
// account's opening-balance anchor. Automatic imports may predate that anchor:
// they remain part of the history, but their effect is already represented by
// the opening balance entered when the local account was created.
func TransactionAffectsBalance(account Account, tx Transaction) bool {
	if !tx.AutomaticImport {
		return true
	}
	date, dateErr := ParseCivilDate(tx.OccurrenceDate)
	opening, openingErr := ParseCivilDate(account.OpeningDate)
	return dateErr != nil || openingErr != nil || !date.Before(opening)
}

// ApplyTransactionWithAccounts applies only the portions of a transaction that
// affect balances anchored by the supplied accounts.
func ApplyTransactionWithAccounts(balances map[string]int64, accounts map[string]Account, tx Transaction) {
	if tx.Kind == Transfer {
		if account, found := accounts[tx.AccountID]; !found || TransactionAffectsBalance(account, tx) {
			balances[tx.AccountID] -= tx.AmountCents
		}
		if account, found := accounts[tx.DestinationAccountID]; !found || TransactionAffectsBalance(account, tx) {
			balances[tx.DestinationAccountID] += tx.AmountCents
		}
		return
	}
	if account, found := accounts[tx.AccountID]; !found || TransactionAffectsBalance(account, tx) {
		balances[tx.AccountID] += SignedAmount(tx.Kind, tx.AmountCents)
	}
}

func ValidateSavingsBalances(accounts []Account, transactions []Transaction) error {
	balances := make(map[string]int64, len(accounts))
	types := make(map[string]AccountType, len(accounts))
	accountsByID := make(map[string]Account, len(accounts))
	for _, account := range accounts {
		balances[account.ID] = account.OpeningBalanceCents
		types[account.ID] = account.Type
		accountsByID[account.ID] = account
	}
	for _, tx := range transactions {
		if tx.DeletedAt == "" {
			ApplyTransactionWithAccounts(balances, accountsByID, tx)
		}
	}
	for id, balance := range balances {
		if types[id] == AccountSavings && balance < 0 {
			return ErrSavingsNegative
		}
	}
	return nil
}

func CalculateDashboard(accounts []Account, transactions []Transaction, now time.Time) Dashboard {
	result := Dashboard{RecentTransactions: []Transaction{}, BalanceHistory: []BalanceHistoryPoint{}, AccountAllocations: []AccountAllocation{}, UpcomingInvoices: []CreditCardInvoice{}}
	balances := make(map[string]int64, len(accounts))
	accountsByID := make(map[string]Account, len(accounts))
	for _, account := range accounts {
		balances[account.ID] = account.OpeningBalanceCents
		accountsByID[account.ID] = account
	}
	month := now.In(time.Local).Format("2006-01")
	for _, tx := range transactions {
		ApplyTransactionWithAccounts(balances, accountsByID, tx)
		if strings.HasPrefix(tx.OccurrenceDate, month+"-") && tx.Origin != OriginAdjustment {
			if tx.Kind == Income {
				result.MonthlyIncomeCents += tx.AmountCents
			} else if tx.Kind == Expense {
				result.MonthlyExpenseCents += tx.AmountCents
			}
		}
	}
	for id, balance := range balances {
		if accountsByID[id].Type != AccountCreditCard {
			result.AvailableBalanceCents += balance
		} else if balance < 0 {
			result.CreditCardDebtCents -= balance
		}
		result.TotalBalanceCents += balance
	}
	result.HasNegativeBalance = result.TotalBalanceCents < 0
	// Keep the JSON contract stable for the frontend: an empty list must be
	// encoded as [] rather than null.
	sorted := append([]Transaction{}, transactions...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].OccurrenceDate == sorted[j].OccurrenceDate {
			return sorted[i].CreatedAt > sorted[j].CreatedAt
		}
		return sorted[i].OccurrenceDate > sorted[j].OccurrenceDate
	})
	if len(sorted) > 5 {
		sorted = sorted[:5]
	}
	result.RecentTransactions = sorted
	result.BalanceHistory = calculateBalanceHistory(accounts, transactions, now)
	for _, account := range accounts {
		if account.Type == AccountCreditCard {
			continue
		}
		result.AccountAllocations = append(result.AccountAllocations, AccountAllocation{AccountID: account.ID, AccountName: account.Name, BalanceCents: account.CurrentBalanceCents})
	}
	sort.SliceStable(result.AccountAllocations, func(i, j int) bool {
		if result.AccountAllocations[i].BalanceCents == result.AccountAllocations[j].BalanceCents {
			return result.AccountAllocations[i].AccountName < result.AccountAllocations[j].AccountName
		}
		return result.AccountAllocations[i].BalanceCents > result.AccountAllocations[j].BalanceCents
	})
	return result
}

// CalculateDashboardWithFixedExpenses keeps planned expenses separate from
// realised balances while exposing the amount that remains safe to spend.
func CalculateDashboardWithFixedExpenses(accounts []Account, transactions []Transaction, occurrences []FixedExpenseOccurrence, now time.Time) Dashboard {
	result := CalculateDashboard(accounts, transactions, now)
	monthEnd := endOfMonth(now.In(time.Local))
	for _, occurrence := range occurrences {
		if occurrence.Status != FixedExpensePending {
			continue
		}
		due, err := ParseCivilDate(occurrence.DueDate)
		if err != nil || due.After(monthEnd) {
			continue
		}
		result.PendingFixedExpensesCents += occurrence.ExpectedAmountCents
		result.PendingFixedExpenseCount++
	}
	result.AvailableBalanceCents -= result.PendingFixedExpensesCents
	return result
}

func calculateBalanceHistory(accounts []Account, transactions []Transaction, now time.Time) []BalanceHistoryPoint {
	points := make([]BalanceHistoryPoint, 0, 7)
	localNow := now.In(time.Local)
	for offset := 6; offset >= 0; offset-- {
		monthStart := time.Date(localNow.Year(), localNow.Month()-time.Month(offset), 1, 0, 0, 0, 0, time.Local)
		cutoff := endOfMonth(monthStart)
		if offset == 0 {
			cutoff = localNow
		}
		balances := make(map[string]int64, len(accounts))
		accountsByID := make(map[string]Account, len(accounts))
		for _, account := range accounts {
			accountsByID[account.ID] = account
			opened, err := ParseCivilDate(account.OpeningDate)
			if err == nil && !opened.After(cutoff) {
				balances[account.ID] = account.OpeningBalanceCents
			}
		}
		for _, tx := range transactions {
			date, err := ParseCivilDate(tx.OccurrenceDate)
			if err == nil && !date.After(cutoff) {
				ApplyTransactionWithAccounts(balances, accountsByID, tx)
			}
		}
		balance := int64(0)
		for _, value := range balances {
			balance += value
		}
		points = append(points, BalanceHistoryPoint{Month: monthStart.Format("2006-01"), Label: monthStart.Format("Jan"), BalanceCents: balance})
	}
	return points
}

func accountByID(accounts []Account, id string) (Account, bool) {
	for _, account := range accounts {
		if account.ID == id {
			return account, true
		}
	}
	return Account{}, false
}

func endOfMonth(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month()+1, 0, 23, 59, 59, 0, value.Location())
}
