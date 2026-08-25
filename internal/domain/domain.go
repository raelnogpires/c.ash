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
	AccountChecking AccountType = "checking"
	AccountSavings  AccountType = "savings"
	AccountCash     AccountType = "cash"
)

type TransactionKind string

const (
	Income   TransactionKind = "income"
	Expense  TransactionKind = "expense"
	Transfer TransactionKind = "transfer"
)

var (
	ErrBlankName            = errors.New("blank name")
	ErrBlankDescription     = errors.New("blank description")
	ErrInvalidAmount        = errors.New("amount must be positive")
	ErrInvalidDate          = errors.New("invalid civil date")
	ErrFutureDate           = errors.New("date is in the future")
	ErrBeforeOpening        = errors.New("date is before account opening")
	ErrInvalidAccountType   = errors.New("invalid account type")
	ErrInvalidKind          = errors.New("invalid transaction kind")
	ErrInvalidTheme         = errors.New("invalid theme")
	ErrUnknownAccount       = errors.New("unknown account")
	ErrAccountInUse         = errors.New("account is in use")
	ErrUnknownCategory      = errors.New("unknown category")
	ErrCategoryKind         = errors.New("category kind does not match transaction")
	ErrSameTransferAccount  = errors.New("transfer accounts must be distinct")
	ErrTransferCategory     = errors.New("transfer cannot have category")
	ErrSavingsNegative      = errors.New("savings account cannot be negative")
	ErrUnknownTransaction   = errors.New("unknown transaction")
	ErrTransactionActive    = errors.New("transaction is already active")
	ErrTransactionTrashed   = errors.New("transaction is already trashed")
	ErrInvalidDueDay        = errors.New("invalid due day")
	ErrUnknownFixedExpense  = errors.New("unknown fixed expense")
	ErrUnknownOccurrence    = errors.New("unknown fixed expense occurrence")
	ErrOccurrenceClosed     = errors.New("fixed expense occurrence is not pending")
	ErrFixedExpenseArchived = errors.New("fixed expense is archived")
	ErrInvalidStatement     = errors.New("invalid bank statement")
	ErrUnsupportedBank      = errors.New("unsupported bank")
	ErrStatementEmpty       = errors.New("bank statement has no transactions")
	ErrStatementTooLarge    = errors.New("bank statement is too large")
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
}

type Category struct {
	ID   string          `json:"id"`
	Name string          `json:"name"`
	Kind TransactionKind `json:"kind"`
}

type Transaction struct {
	ID                       string          `json:"id"`
	Kind                     TransactionKind `json:"kind"`
	AmountCents              int64           `json:"amountCents"`
	AccountID                string          `json:"accountId"`
	AccountName              string          `json:"accountName"`
	DestinationAccountID     string          `json:"destinationAccountId,omitempty"`
	DestinationAccountName   string          `json:"destinationAccountName,omitempty"`
	CategoryID               string          `json:"categoryId,omitempty"`
	CategoryName             string          `json:"categoryName,omitempty"`
	Description              string          `json:"description"`
	OccurrenceDate           string          `json:"occurrenceDate"`
	CreatedAt                string          `json:"createdAt"`
	UpdatedAt                string          `json:"updatedAt"`
	DeletedAt                string          `json:"deletedAt,omitempty"`
	FixedExpenseOccurrenceID string          `json:"fixedExpenseOccurrenceId,omitempty"`
	AutomaticImport          bool            `json:"automaticImport"`
	ImportBank               string          `json:"importBank,omitempty"`
	ImportKey                string          `json:"-"`
}

type FixedExpense struct {
	ID           string `json:"id"`
	Description  string `json:"description"`
	AmountCents  int64  `json:"amountCents"`
	DueDay       int    `json:"dueDay"`
	AccountID    string `json:"accountId"`
	AccountName  string `json:"accountName"`
	CategoryID   string `json:"categoryId"`
	CategoryName string `json:"categoryName"`
	ArchivedAt   string `json:"archivedAt,omitempty"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
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
	if accountType != AccountChecking && accountType != AccountSavings && accountType != AccountCash {
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
	result := Dashboard{RecentTransactions: []Transaction{}, BalanceHistory: []BalanceHistoryPoint{}, AccountAllocations: []AccountAllocation{}}
	balances := make(map[string]int64, len(accounts))
	accountsByID := make(map[string]Account, len(accounts))
	for _, account := range accounts {
		balances[account.ID] = account.OpeningBalanceCents
		accountsByID[account.ID] = account
	}
	month := now.In(time.Local).Format("2006-01")
	for _, tx := range transactions {
		ApplyTransactionWithAccounts(balances, accountsByID, tx)
		if strings.HasPrefix(tx.OccurrenceDate, month+"-") {
			if tx.Kind == Income {
				result.MonthlyIncomeCents += tx.AmountCents
			} else if tx.Kind == Expense {
				result.MonthlyExpenseCents += tx.AmountCents
			}
		}
	}
	for _, balance := range balances {
		result.AvailableBalanceCents += balance
	}
	result.TotalBalanceCents = result.AvailableBalanceCents
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
	result.AvailableBalanceCents = result.TotalBalanceCents - result.PendingFixedExpensesCents
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
		balance := int64(0)
		for _, account := range accounts {
			opened, err := ParseCivilDate(account.OpeningDate)
			if err == nil && !opened.After(cutoff) {
				balance += account.OpeningBalanceCents
			}
		}
		for _, tx := range transactions {
			date, err := ParseCivilDate(tx.OccurrenceDate)
			account, found := accountByID(accounts, tx.AccountID)
			if err == nil && !date.After(cutoff) && (!found || TransactionAffectsBalance(account, tx)) {
				balance += SignedAmount(tx.Kind, tx.AmountCents)
			}
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
