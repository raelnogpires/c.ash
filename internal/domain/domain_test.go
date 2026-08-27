package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func fixedTime() time.Time { return time.Date(2026, time.August, 16, 12, 0, 0, 0, time.Local) }

func TestCalculateDashboard_CentsMonthlyBoundariesNegativeAndOrdering(t *testing.T) {
	accounts := []Account{{ID: "a", OpeningBalanceCents: 1000}}
	txs := []Transaction{
		{ID: "old", Kind: Income, AmountCents: 500, OccurrenceDate: "2026-07-31", CreatedAt: "2026-08-01T10:00:00Z"},
		{ID: "expense", Kind: Expense, AmountCents: 1800, OccurrenceDate: "2026-08-01", CreatedAt: "2026-08-01T11:00:00Z"},
		{ID: "income", Kind: Income, AmountCents: 200, OccurrenceDate: "2026-08-01", CreatedAt: "2026-08-01T12:00:00Z"},
	}
	got := CalculateDashboard(accounts, txs, fixedTime())
	if got.AvailableBalanceCents != -100 {
		t.Fatalf("balance = %d, want -100", got.AvailableBalanceCents)
	}
	if got.MonthlyIncomeCents != 200 || got.MonthlyExpenseCents != 1800 {
		t.Fatalf("monthly totals = %d/%d", got.MonthlyIncomeCents, got.MonthlyExpenseCents)
	}
	if !got.HasNegativeBalance {
		t.Fatal("expected negative balance warning")
	}
	if got.RecentTransactions[0].ID != "income" || got.RecentTransactions[1].ID != "expense" {
		t.Fatalf("unexpected ordering: %#v", got.RecentTransactions)
	}
	if SignedAmount(Income, 99) != 99 || SignedAmount(Expense, 99) != -99 {
		t.Fatal("transaction signs are incorrect")
	}
}

func TestCalculateDashboard_RecentIsLimitedToFive(t *testing.T) {
	txs := make([]Transaction, 6)
	for i := range txs {
		txs[i] = Transaction{ID: string(rune('a' + i)), Kind: Income, AmountCents: 1, OccurrenceDate: "2026-08-01", CreatedAt: time.Date(2026, 8, 1, i, 0, 0, 0, time.UTC).Format(time.RFC3339)}
	}
	if got := CalculateDashboard(nil, txs, fixedTime()); len(got.RecentTransactions) != 5 {
		t.Fatalf("recent count = %d", len(got.RecentTransactions))
	}
}

func TestCalculateDashboard_WarnsWhenAnyCashAccountIsNegative(t *testing.T) {
	accounts := []Account{{ID: "negative", Type: AccountChecking, OpeningBalanceCents: -100}, {ID: "positive", Type: AccountChecking, OpeningBalanceCents: 1000}}
	got := CalculateDashboard(accounts, nil, fixedTime())
	if got.TotalBalanceCents != 900 || !got.HasNegativeBalance {
		t.Fatalf("dashboard=%+v", got)
	}
}

func TestCalculateDashboard_EmptyRecentTransactionsEncodeAsArray(t *testing.T) {
	dashboard := CalculateDashboard(nil, nil, fixedTime())
	if dashboard.RecentTransactions == nil {
		t.Fatal("recent transactions must be an empty slice, not nil")
	}
	encoded, err := json.Marshal(dashboard)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"recentTransactions":[]`) || !strings.Contains(string(encoded), `"accountAllocations":[]`) || !strings.Contains(string(encoded), `"upcomingInvoices":[]`) {
		t.Fatalf("dashboard JSON = %s", encoded)
	}
}

func TestCalculateDashboardWithFixedExpenses_OnlyPendingDueThisMonthReduceAvailable(t *testing.T) {
	accounts := []Account{{ID: "a", Name: "Principal", OpeningBalanceCents: 10000, OpeningDate: "2026-01-01", CurrentBalanceCents: 10000}}
	occurrences := []FixedExpenseOccurrence{
		{ID: "overdue", Status: FixedExpensePending, DueDate: "2026-07-10", ExpectedAmountCents: 1200},
		{ID: "current", Status: FixedExpensePending, DueDate: "2026-08-20", ExpectedAmountCents: 2500},
		{ID: "future", Status: FixedExpensePending, DueDate: "2026-09-01", ExpectedAmountCents: 3000},
		{ID: "confirmed", Status: FixedExpenseConfirmed, DueDate: "2026-08-05", ExpectedAmountCents: 900},
	}
	got := CalculateDashboardWithFixedExpenses(accounts, nil, occurrences, fixedTime())
	if got.TotalBalanceCents != 10000 || got.PendingFixedExpensesCents != 3700 || got.AvailableBalanceCents != 6300 || got.PendingFixedExpenseCount != 2 {
		t.Fatalf("dashboard=%+v", got)
	}
}

func TestFixedExpenseJSON_OmitsOccurrenceCursor(t *testing.T) {
	expense := FixedExpense{ID: "fixed", OccurrenceStartAt: "2026-08-10T12:00:00Z"}
	encoded, err := json.Marshal(expense)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "occurrenceStartAt") || strings.Contains(string(encoded), "2026-08-10T12:00:00Z") {
		t.Fatalf("internal occurrence cursor leaked into JSON: %s", encoded)
	}
}

func TestCalculateDashboard_BuildsHistoryAndAccountAllocations(t *testing.T) {
	accounts := []Account{{ID: "a", Name: "Conta", OpeningBalanceCents: 10000, OpeningDate: "2026-04-10", CurrentBalanceCents: 9000}, {ID: "b", Name: "Reserva", OpeningBalanceCents: 2500, OpeningDate: "2026-08-01", CurrentBalanceCents: 2500}}
	txs := []Transaction{{Kind: Expense, AmountCents: 1000, OccurrenceDate: "2026-08-03"}}
	got := CalculateDashboard(accounts, txs, fixedTime())
	if len(got.BalanceHistory) != 7 || got.BalanceHistory[1].BalanceCents != 0 || got.BalanceHistory[2].BalanceCents != 10000 || got.BalanceHistory[6].BalanceCents != 11500 {
		t.Fatalf("history=%+v", got.BalanceHistory)
	}
	if len(got.AccountAllocations) != 2 || got.AccountAllocations[0].AccountName != "Conta" || got.AccountAllocations[1].AccountName != "Reserva" {
		t.Fatalf("allocations=%+v", got.AccountAllocations)
	}
}

func TestValidateFixedExpense(t *testing.T) {
	category := &Category{ID: "housing", Kind: Expense}
	if err := ValidateFixedExpense("Aluguel", 100, 31, category); err != nil {
		t.Fatal(err)
	}
	if err := ValidateFixedExpense("Aluguel", 100, 32, category); !errors.Is(err, ErrInvalidDueDay) {
		t.Fatalf("error=%v", err)
	}
	if err := ValidateFixedExpense("", 100, 1, category); !errors.Is(err, ErrBlankDescription) {
		t.Fatalf("error=%v", err)
	}
}

func TestValidateAccount_AllRules(t *testing.T) {
	tests := []struct {
		name        string
		accountName string
		kind        AccountType
		date        string
		want        error
	}{
		{"blank", "  ", AccountCash, "2026-08-16", ErrBlankName},
		{"type", "Carteira", "investment", "2026-08-16", ErrInvalidAccountType},
		{"invalid date", "Carteira", AccountCash, "16/08/2026", ErrInvalidDate},
		{"future", "Carteira", AccountCash, "2026-08-17", ErrFutureDate},
	}
	if err := ValidateAccount("Reserva", AccountSavings, "2026-08-16", fixedTime()); err != nil {
		t.Fatalf("savings rejected: %v", err)
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateAccount(tc.accountName, tc.kind, tc.date, fixedTime()); !errors.Is(err, tc.want) {
				t.Fatalf("error=%v want %v", err, tc.want)
			}
		})
	}
}

func TestCreditCardCycleAndInstallmentRounding(t *testing.T) {
	purchase, _ := ParseCivilDate("2026-08-25")
	closing, due := CreditCardCycle(purchase, 25, 2)
	if closing.Format("2006-01-02") != "2026-09-25" || due.Format("2006-01-02") != "2026-10-02" {
		t.Fatalf("cycle=%s/%s", closing.Format("2006-01-02"), due.Format("2006-01-02"))
	}
	purchase, _ = ParseCivilDate("2027-02-01")
	closing, due = CreditCardCycle(purchase, 31, 31)
	if closing.Format("2006-01-02") != "2027-02-28" || due.Format("2006-01-02") != "2027-03-31" {
		t.Fatalf("clamped cycle=%s/%s", closing.Format("2006-01-02"), due.Format("2006-01-02"))
	}
	items, err := InstallmentAmounts(100, 3)
	if err != nil || len(items) != 3 || items[0] != 34 || items[1] != 33 || items[2] != 33 {
		t.Fatalf("installments=%v err=%v", items, err)
	}
}

func TestCreditCardValidationAndDashboardSemantics(t *testing.T) {
	card := Account{ID: "card", Name: "Cartão", Type: AccountCreditCard, OpeningBalanceCents: -2000, OpeningDate: "2026-08-01", CreditLimitCents: 10000, ClosingDay: 25, DueDay: 2, CurrentBalanceCents: -5000}
	if err := ValidateCreditCard(card); err != nil {
		t.Fatal(err)
	}
	bad := card
	bad.CreditLimitCents = 0
	if !errors.Is(ValidateCreditCard(bad), ErrInvalidCreditLimit) {
		t.Fatal("invalid limit accepted")
	}
	cash := Account{ID: "cash", Name: "Conta", Type: AccountChecking, OpeningBalanceCents: 10000, OpeningDate: "2026-08-01", CurrentBalanceCents: 10000}
	purchase := Transaction{Kind: Expense, AmountCents: 3000, AccountID: card.ID, Description: "Notebook", OccurrenceDate: "2026-08-10", InstallmentCount: 3}
	got := CalculateDashboard([]Account{cash, card}, []Transaction{purchase}, fixedTime())
	if got.AvailableBalanceCents != 10000 || got.TotalBalanceCents != 5000 || got.CreditCardDebtCents != 5000 || got.MonthlyExpenseCents != 3000 || len(got.AccountAllocations) != 1 {
		t.Fatalf("dashboard=%+v", got)
	}
	if err := ValidateTransaction(purchase, card, nil, &Category{Kind: Expense}, fixedTime()); err != nil {
		t.Fatalf("card expense rejected: %v", err)
	}
	income := purchase
	income.Kind = Income
	if !errors.Is(ValidateTransaction(income, card, nil, nil, fixedTime()), ErrCardTransaction) {
		t.Fatal("card income accepted")
	}
}

func TestValidateTransaction_AllRules(t *testing.T) {
	account := Account{ID: "a", OpeningDate: "2026-08-10"}
	expense := Category{ID: "food", Kind: Expense}
	income := Category{ID: "salary", Kind: Income}
	valid := Transaction{Kind: Expense, AmountCents: 100, Description: "Café", OccurrenceDate: "2026-08-16"}
	tests := []struct {
		name     string
		mutate   func(*Transaction)
		category *Category
		want     error
	}{
		{"kind", func(tx *Transaction) { tx.Kind = "refund" }, nil, ErrInvalidKind},
		{"amount", func(tx *Transaction) { tx.AmountCents = 0 }, nil, ErrInvalidAmount},
		{"description", func(tx *Transaction) { tx.Description = " " }, nil, ErrBlankDescription},
		{"invalid date", func(tx *Transaction) { tx.OccurrenceDate = "nope" }, nil, ErrInvalidDate},
		{"future", func(tx *Transaction) { tx.OccurrenceDate = "2026-08-17" }, nil, ErrFutureDate},
		{"before opening", func(tx *Transaction) { tx.OccurrenceDate = "2026-08-09" }, nil, ErrBeforeOpening},
		{"category kind", func(tx *Transaction) {}, &income, ErrCategoryKind},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tx := valid
			tc.mutate(&tx)
			if err := ValidateTransaction(tx, account, nil, tc.category, fixedTime()); !errors.Is(err, tc.want) {
				t.Fatalf("error=%v want %v", err, tc.want)
			}
		})
	}
	if err := ValidateTransaction(valid, account, nil, &expense, fixedTime()); err != nil {
		t.Fatalf("valid transaction: %v", err)
	}
	historical := valid
	historical.OccurrenceDate = "2026-07-01"
	historical.AutomaticImport = true
	if err := ValidateTransaction(historical, account, nil, &expense, fixedTime()); err != nil {
		t.Fatalf("historical automatic import rejected: %v", err)
	}
}

func TestHistoricalAutomaticImport_IsReportedWithoutChangingAnchoredBalance(t *testing.T) {
	account := Account{ID: "a", Type: AccountSavings, OpeningBalanceCents: 1000, OpeningDate: "2026-08-16", CurrentBalanceCents: 1000}
	historical := Transaction{ID: "old", Kind: Expense, AmountCents: 9000, AccountID: "a", Description: "Histórico", OccurrenceDate: "2026-07-10", AutomaticImport: true}
	if err := ValidateSavingsBalances([]Account{account}, []Transaction{historical}); err != nil {
		t.Fatalf("historical transaction affected savings anchor: %v", err)
	}
	dashboard := CalculateDashboard([]Account{account}, []Transaction{historical}, fixedTime())
	if dashboard.TotalBalanceCents != 1000 || len(dashboard.RecentTransactions) != 1 || dashboard.RecentTransactions[0].ID != "old" {
		t.Fatalf("dashboard=%+v", dashboard)
	}
}

func TestTransfer_ValidationBalancesAndMonthlyNeutrality(t *testing.T) {
	source := Account{ID: "source", Type: AccountChecking, OpeningBalanceCents: 1000, OpeningDate: "2026-08-01"}
	destination := Account{ID: "destination", Type: AccountSavings, OpeningDate: "2026-08-10"}
	tx := Transaction{ID: "transfer", Kind: Transfer, AmountCents: 400, AccountID: source.ID, DestinationAccountID: destination.ID, Description: "", OccurrenceDate: "2026-08-16"}
	if err := ValidateTransaction(tx, source, &destination, nil, fixedTime()); err != nil {
		t.Fatal(err)
	}
	same := source
	if err := ValidateTransaction(tx, source, &same, nil, fixedTime()); !errors.Is(err, ErrSameTransferAccount) {
		t.Fatalf("same account error=%v", err)
	}
	tx.OccurrenceDate = "2026-08-09"
	if err := ValidateTransaction(tx, source, &destination, nil, fixedTime()); !errors.Is(err, ErrBeforeOpening) {
		t.Fatalf("destination opening error=%v", err)
	}
	tx.OccurrenceDate = "2026-08-16"
	balances := map[string]int64{source.ID: 1000, destination.ID: 0}
	ApplyTransaction(balances, tx)
	if balances[source.ID] != 600 || balances[destination.ID] != 400 {
		t.Fatalf("balances=%v", balances)
	}
	dashboard := CalculateDashboard([]Account{source, destination}, []Transaction{tx}, fixedTime())
	if dashboard.AvailableBalanceCents != 1000 || dashboard.MonthlyIncomeCents != 0 || dashboard.MonthlyExpenseCents != 0 {
		t.Fatalf("dashboard=%+v", dashboard)
	}
}

func TestValidateSavingsBalances_RejectsNegative(t *testing.T) {
	accounts := []Account{{ID: "s", Type: AccountSavings, OpeningBalanceCents: 100, OpeningDate: "2026-08-01"}}
	txs := []Transaction{{Kind: Expense, AmountCents: 101, AccountID: "s"}}
	if err := ValidateSavingsBalances(accounts, txs); !errors.Is(err, ErrSavingsNegative) {
		t.Fatalf("error=%v", err)
	}
	txs[0].DeletedAt = "2026-08-16T00:00:00Z"
	if err := ValidateSavingsBalances(accounts, txs); err != nil {
		t.Fatalf("trashed transaction affected balance: %v", err)
	}
}

func TestValidateTheme(t *testing.T) {
	for _, theme := range []Theme{ThemeLight, ThemeDark, ThemeGothic} {
		if err := ValidateTheme(theme); err != nil {
			t.Fatal(err)
		}
	}
	if !errors.Is(ValidateTheme("sepia"), ErrInvalidTheme) {
		t.Fatal("invalid theme accepted")
	}
}
