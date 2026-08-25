package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"c.ash/internal/domain"
	"c.ash/internal/importer"
	"c.ash/internal/storage"
)

func testService(t *testing.T) (*Service, *storage.Store) {
	t.Helper()
	store, err := storage.Open(context.Background(), t.TempDir()+"/cash.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	now := func() time.Time { return time.Date(2026, 8, 16, 12, 0, 0, 0, time.Local) }
	return New(store, now), store
}

func TestSkipOnboarding_CreatesNoFinancialRecords(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	p, err := service.SkipOnboarding(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if p.OnboardingStatus != "skipped" {
		t.Fatalf("status=%s", p.OnboardingStatus)
	}
	if p.Theme != "" {
		t.Fatalf("skip persisted an explicit theme: %q", p.Theme)
	}
	boot, err := service.Bootstrap(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(boot.Accounts) != 0 || len(boot.Dashboard.RecentTransactions) != 0 {
		t.Fatal("skip created financial data")
	}
}

func TestWorkflow_RecalculatesDashboardAndPersistsTheme(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	p, err := service.CompleteOnboarding(ctx, OnboardingInput{DisplayName: " Ana ", Currency: "BRL", Theme: domain.ThemeGothic, FirstAccount: AccountInput{Name: "Principal", Type: domain.AccountChecking, OpeningBalanceCents: 10000, OpeningDate: "2026-08-01"}})
	if err != nil {
		t.Fatal(err)
	}
	if p.DisplayName != "Ana" {
		t.Fatalf("name=%q", p.DisplayName)
	}
	boot, _ := service.Bootstrap(ctx)
	account := boot.Accounts[0]
	for _, in := range []TransactionInput{{Kind: domain.Income, AmountCents: 5000, AccountID: account.ID, CategoryID: "salary", Description: "Salário", OccurrenceDate: "2026-08-10"}, {Kind: domain.Expense, AmountCents: 3500, AccountID: account.ID, CategoryID: "food", Description: "Mercado", OccurrenceDate: "2026-08-12"}} {
		if _, err := service.CreateTransaction(ctx, in); err != nil {
			t.Fatal(err)
		}
	}
	boot, err = service.Bootstrap(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if boot.Dashboard.AvailableBalanceCents != 11500 || boot.Dashboard.MonthlyIncomeCents != 5000 || boot.Dashboard.MonthlyExpenseCents != 3500 {
		t.Fatalf("dashboard=%+v", boot.Dashboard)
	}
	if boot.Accounts[0].CurrentBalanceCents != 11500 {
		t.Fatalf("account balance=%d", boot.Accounts[0].CurrentBalanceCents)
	}
	if _, err := service.SetTheme(ctx, domain.ThemeDark); err != nil {
		t.Fatal(err)
	}
	boot, _ = service.Bootstrap(ctx)
	if boot.Theme != domain.ThemeDark {
		t.Fatalf("theme=%s", boot.Theme)
	}
}

func TestFailedCommandsLeaveStateUnchanged(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	_, err := service.CompleteOnboarding(ctx, OnboardingInput{DisplayName: "Ana", Currency: "BRL", Theme: domain.ThemeLight, FirstAccount: AccountInput{Name: "", Type: domain.AccountCash, OpeningDate: "2026-08-01"}})
	if !errors.Is(err, domain.ErrBlankName) {
		t.Fatalf("error=%v", err)
	}
	boot, _ := service.Bootstrap(ctx)
	if boot.Profile != nil || len(boot.Accounts) != 0 {
		t.Fatal("invalid onboarding changed state")
	}
	if _, err := service.CreateTransaction(ctx, TransactionInput{Kind: domain.Expense, AmountCents: 100, AccountID: "missing", Description: "Teste", OccurrenceDate: "2026-08-16"}); !errors.Is(err, domain.ErrUnknownAccount) {
		t.Fatalf("error=%v", err)
	}
	txs, _ := service.ListTransactions(ctx)
	if len(txs) != 0 {
		t.Fatal("failed transaction was stored")
	}
}

func TestTransactionLifecycle_TransferEditTrashRestoreAndRevisions(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	_, err := service.CompleteOnboarding(ctx, OnboardingInput{DisplayName: "Ana", Currency: "BRL", Theme: domain.ThemeLight, FirstAccount: AccountInput{Name: "Principal", Type: domain.AccountChecking, OpeningBalanceCents: 1000, OpeningDate: "2026-08-01"}})
	if err != nil {
		t.Fatal(err)
	}
	savings, err := service.CreateAccount(ctx, AccountInput{Name: "Reserva", Type: domain.AccountSavings, OpeningBalanceCents: 0, OpeningDate: "2026-08-05"})
	if err != nil {
		t.Fatal(err)
	}
	boot, _ := service.Bootstrap(ctx)
	source := boot.Accounts[0]
	transfer, err := service.CreateTransaction(ctx, TransactionInput{Kind: domain.Transfer, AmountCents: 400, AccountID: source.ID, DestinationAccountID: savings.ID, OccurrenceDate: "2026-08-10"})
	if err != nil {
		t.Fatal(err)
	}
	if transfer.Description != "Transferência para Reserva" || transfer.DestinationAccountName != "Reserva" {
		t.Fatalf("transfer=%+v", transfer)
	}
	boot, _ = service.Bootstrap(ctx)
	if boot.Dashboard.AvailableBalanceCents != 1000 || boot.Dashboard.MonthlyIncomeCents != 0 || boot.Dashboard.MonthlyExpenseCents != 0 {
		t.Fatalf("transfer dashboard=%+v", boot.Dashboard)
	}
	balances := map[string]int64{}
	for _, account := range boot.Accounts {
		balances[account.ID] = account.CurrentBalanceCents
	}
	if balances[source.ID] != 600 || balances[savings.ID] != 400 {
		t.Fatalf("balances=%v", balances)
	}

	updated, err := service.UpdateTransaction(ctx, transfer.ID, TransactionInput{Kind: domain.Expense, AmountCents: 250, AccountID: source.ID, CategoryID: "food", Description: "Mercado", OccurrenceDate: "2026-08-12"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Kind != domain.Expense || updated.DestinationAccountID != "" || updated.CategoryName != "Alimentação" {
		t.Fatalf("updated=%+v", updated)
	}
	boot, _ = service.Bootstrap(ctx)
	if boot.Dashboard.AvailableBalanceCents != 750 || boot.Dashboard.MonthlyExpenseCents != 250 {
		t.Fatalf("edited dashboard=%+v", boot.Dashboard)
	}
	if err := service.TrashTransaction(ctx, transfer.ID); err != nil {
		t.Fatal(err)
	}
	if txs, _ := service.ListTransactions(ctx); len(txs) != 0 {
		t.Fatalf("trashed transaction listed: %+v", txs)
	}
	boot, _ = service.Bootstrap(ctx)
	if boot.Dashboard.AvailableBalanceCents != 1000 {
		t.Fatalf("trash balance=%d", boot.Dashboard.AvailableBalanceCents)
	}
	if err := service.RestoreTransaction(ctx, transfer.ID); err != nil {
		t.Fatal(err)
	}
	boot, _ = service.Bootstrap(ctx)
	if boot.Dashboard.AvailableBalanceCents != 750 {
		t.Fatalf("restore balance=%d", boot.Dashboard.AvailableBalanceCents)
	}
}

func TestImportStatement_IsCumulativeDeduplicatedAndEditable(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	account, err := service.CreateAccount(ctx, AccountInput{Name: "Inter", Type: domain.AccountChecking, OpeningBalanceCents: 5000, OpeningDate: "2026-08-16"})
	if err != nil {
		t.Fatal(err)
	}
	entries := []importer.Entry{
		{Kind: domain.Expense, AmountCents: 2590, Description: "Pix enviado João", Date: "2026-08-02"},
		{Kind: domain.Expense, AmountCents: 2590, Description: "Pix enviado João", Date: "2026-08-02"},
		{Kind: domain.Income, AmountCents: 100000, Description: "Salário", Date: "2026-08-05"},
	}
	first, err := service.importStatementEntries(ctx, account.ID, importer.BankInter, entries)
	if err != nil || first.ImportedCount != 3 || first.DuplicateCount != 0 {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	boot, err := service.Bootstrap(ctx)
	if err != nil || boot.Accounts[0].CurrentBalanceCents != 5000 || boot.Dashboard.MonthlyIncomeCents != 100000 || boot.Dashboard.MonthlyExpenseCents != 5180 {
		t.Fatalf("historical import changed anchored balance: boot=%+v err=%v", boot, err)
	}
	if _, err := service.UpdateAccount(ctx, account.ID, AccountInput{Name: "Inter atualizada", Type: domain.AccountChecking, OpeningBalanceCents: 5000, OpeningDate: "2026-08-16"}); err != nil {
		t.Fatalf("historical import blocked account update: %v", err)
	}
	second, err := service.importStatementEntries(ctx, account.ID, importer.BankInter, entries)
	if err != nil || second.ImportedCount != 0 || second.DuplicateCount != 3 {
		t.Fatalf("second=%+v err=%v", second, err)
	}
	txs, err := service.ListTransactions(ctx)
	if err != nil || len(txs) != 3 || !txs[0].AutomaticImport || txs[0].ImportBank != "inter" {
		t.Fatalf("transactions=%+v err=%v", txs, err)
	}
	updated, err := service.UpdateTransaction(ctx, txs[0].ID, TransactionInput{Kind: txs[0].Kind, AmountCents: txs[0].AmountCents, AccountID: account.ID, Description: "Salário de agosto", OccurrenceDate: txs[0].OccurrenceDate})
	if err != nil || !updated.AutomaticImport || updated.ImportBank != "inter" {
		t.Fatalf("updated=%+v err=%v", updated, err)
	}
	if err := service.TrashTransaction(ctx, txs[1].ID); err != nil {
		t.Fatal(err)
	}
	third, err := service.importStatementEntries(ctx, account.ID, importer.BankInter, entries)
	if err != nil || third.ImportedCount != 0 || third.DuplicateCount != 3 {
		t.Fatalf("third=%+v err=%v", third, err)
	}
}

func TestSavingsRules_RollBackInvalidMutations(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	if _, err := service.CreateAccount(ctx, AccountInput{Name: "Reserva", Type: domain.AccountSavings, OpeningBalanceCents: -1, OpeningDate: "2026-08-01"}); !errors.Is(err, domain.ErrSavingsNegative) {
		t.Fatalf("negative opening error=%v", err)
	}
	savings, err := service.CreateAccount(ctx, AccountInput{Name: "Reserva", Type: domain.AccountSavings, OpeningBalanceCents: 100, OpeningDate: "2026-08-01"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateTransaction(ctx, TransactionInput{Kind: domain.Expense, AmountCents: 101, AccountID: savings.ID, Description: "Teste", OccurrenceDate: "2026-08-10"}); !errors.Is(err, domain.ErrSavingsNegative) {
		t.Fatalf("negative expense error=%v", err)
	}
	if txs, _ := service.ListTransactions(ctx); len(txs) != 0 {
		t.Fatal("failed savings mutation was stored")
	}
	checking, _ := service.CreateAccount(ctx, AccountInput{Name: "Corrente", Type: domain.AccountChecking, OpeningBalanceCents: 100, OpeningDate: "2026-08-01"})
	if _, err := service.CreateTransaction(ctx, TransactionInput{Kind: domain.Transfer, AmountCents: 10, AccountID: checking.ID, DestinationAccountID: checking.ID, OccurrenceDate: "2026-08-10"}); !errors.Is(err, domain.ErrSameTransferAccount) {
		t.Fatalf("same transfer error=%v", err)
	}
}

func TestUpdateAccount_ValidatesLinksAndRecalculatesBalance(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	account, err := service.CreateAccount(ctx, AccountInput{Name: "Principal", Type: domain.AccountChecking, OpeningBalanceCents: 1000, OpeningDate: "2026-08-01"})
	if err != nil {
		t.Fatal(err)
	}
	tx, err := service.CreateTransaction(ctx, TransactionInput{Kind: domain.Expense, AmountCents: 300, AccountID: account.ID, Description: "Mercado", OccurrenceDate: "2026-08-10"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateAccount(ctx, "missing", AccountInput{Name: "Outra", Type: domain.AccountChecking, OpeningDate: "2026-08-01"}); !errors.Is(err, domain.ErrUnknownAccount) {
		t.Fatalf("unknown account error=%v", err)
	}
	if _, err := service.UpdateAccount(ctx, account.ID, AccountInput{Name: "Principal", Type: domain.AccountChecking, OpeningBalanceCents: 1000, OpeningDate: "2026-08-11"}); !errors.Is(err, domain.ErrBeforeOpening) {
		t.Fatalf("incompatible date error=%v", err)
	}
	if err := service.TrashTransaction(ctx, tx.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateAccount(ctx, account.ID, AccountInput{Name: "Principal", Type: domain.AccountChecking, OpeningBalanceCents: 1000, OpeningDate: "2026-08-11"}); !errors.Is(err, domain.ErrBeforeOpening) {
		t.Fatalf("trashed date error=%v", err)
	}
	if err := service.RestoreTransaction(ctx, tx.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateAccount(ctx, account.ID, AccountInput{Name: "Reserva", Type: domain.AccountSavings, OpeningBalanceCents: 200, OpeningDate: "2026-08-01"}); !errors.Is(err, domain.ErrSavingsNegative) {
		t.Fatalf("savings conversion error=%v", err)
	}
	updated, err := service.UpdateAccount(ctx, account.ID, AccountInput{Name: " Principal nova ", Type: domain.AccountChecking, OpeningBalanceCents: 2000, OpeningDate: "2026-08-01"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Principal nova" || updated.CurrentBalanceCents != 1700 || updated.CreatedAt != account.CreatedAt {
		t.Fatalf("updated=%+v", updated)
	}
}

func TestDeleteAccount_BlocksUsageAndAllowsLastAccount(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	used, err := service.CreateAccount(ctx, AccountInput{Name: "Usada", Type: domain.AccountChecking, OpeningDate: "2026-08-01"})
	if err != nil {
		t.Fatal(err)
	}
	tx, err := service.CreateTransaction(ctx, TransactionInput{Kind: domain.Income, AmountCents: 100, AccountID: used.ID, Description: "Receita", OccurrenceDate: "2026-08-10"})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.TrashTransaction(ctx, tx.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteAccount(ctx, used.ID); !errors.Is(err, domain.ErrAccountInUse) {
		t.Fatalf("used account error=%v", err)
	}
	boot, _ := service.Bootstrap(ctx)
	if len(boot.Accounts) != 1 {
		t.Fatal("failed delete changed accounts")
	}
	empty, err := service.CreateAccount(ctx, AccountInput{Name: "Vazia", Type: domain.AccountCash, OpeningBalanceCents: 500, OpeningDate: "2026-08-01"})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteAccount(ctx, empty.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteAccount(ctx, used.ID); !errors.Is(err, domain.ErrAccountInUse) {
		t.Fatalf("last used account error=%v", err)
	}
}

func TestDeleteAccount_AllowsOnlyAccountWhenUnused(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	account, err := service.CreateAccount(ctx, AccountInput{Name: "Única", Type: domain.AccountCash, OpeningBalanceCents: 500, OpeningDate: "2026-08-01"})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteAccount(ctx, account.ID); err != nil {
		t.Fatal(err)
	}
	boot, err := service.Bootstrap(ctx)
	if err != nil || len(boot.Accounts) != 0 || boot.Dashboard.AvailableBalanceCents != 0 {
		t.Fatalf("bootstrap=%+v err=%v", boot, err)
	}
}

func TestFixedExpenseWorkflow_SeparatesForecastFromRealisedBalance(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	if _, err := service.CompleteOnboarding(ctx, OnboardingInput{DisplayName: "Ana", Currency: "BRL", Theme: domain.ThemeLight, FirstAccount: AccountInput{Name: "Principal", Type: domain.AccountChecking, OpeningBalanceCents: 10000, OpeningDate: "2026-08-01"}}); err != nil {
		t.Fatal(err)
	}
	boot, err := service.Bootstrap(ctx)
	if err != nil {
		t.Fatal(err)
	}
	expense, err := service.CreateFixedExpense(ctx, FixedExpenseInput{Description: "Aluguel", AmountCents: 2500, DueDay: 20, AccountID: boot.Accounts[0].ID, CategoryID: "housing"})
	if err != nil {
		t.Fatal(err)
	}
	overview, err := service.FixedExpensesOverview(ctx)
	if err != nil || len(overview.Occurrences) != 1 || overview.Occurrences[0].Status != domain.FixedExpensePending {
		t.Fatalf("overview=%+v err=%v", overview, err)
	}
	boot, _ = service.Bootstrap(ctx)
	if boot.Dashboard.TotalBalanceCents != 10000 || boot.Dashboard.AvailableBalanceCents != 7500 || boot.Dashboard.PendingFixedExpensesCents != 2500 {
		t.Fatalf("forecast dashboard=%+v", boot.Dashboard)
	}
	tx, err := service.ConfirmFixedExpenseOccurrence(ctx, overview.Occurrences[0].ID, ConfirmFixedExpenseOccurrenceInput{AmountCents: 2600, OccurrenceDate: "2026-08-16"})
	if err != nil || tx.FixedExpenseOccurrenceID != overview.Occurrences[0].ID {
		t.Fatalf("transaction=%+v err=%v", tx, err)
	}
	boot, _ = service.Bootstrap(ctx)
	if boot.Dashboard.TotalBalanceCents != 7400 || boot.Dashboard.AvailableBalanceCents != 7400 || boot.Dashboard.PendingFixedExpensesCents != 0 {
		t.Fatalf("confirmed dashboard=%+v", boot.Dashboard)
	}
	if err := service.TrashTransaction(ctx, tx.ID); err != nil {
		t.Fatal(err)
	}
	boot, _ = service.Bootstrap(ctx)
	if boot.Dashboard.AvailableBalanceCents != 7500 || boot.Dashboard.PendingFixedExpenseCount != 1 {
		t.Fatalf("reopened dashboard=%+v", boot.Dashboard)
	}
	if err := service.RestoreTransaction(ctx, tx.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.ArchiveFixedExpense(ctx, expense.ID); err != nil {
		t.Fatal(err)
	}
	overview, _ = service.FixedExpensesOverview(ctx)
	if overview.Expenses[0].ArchivedAt == "" || overview.Occurrences[0].Status != domain.FixedExpenseConfirmed {
		t.Fatalf("archived overview=%+v", overview)
	}
}

func TestFixedExpense_DismissAndEditOnlyFutureRules(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	account, err := service.CreateAccount(ctx, AccountInput{Name: "Conta", Type: domain.AccountChecking, OpeningBalanceCents: 1000, OpeningDate: "2026-08-01"})
	if err != nil {
		t.Fatal(err)
	}
	expense, err := service.CreateFixedExpense(ctx, FixedExpenseInput{Description: "Internet", AmountCents: 100, DueDay: 31, AccountID: account.ID, CategoryID: "bills"})
	if err != nil {
		t.Fatal(err)
	}
	overview, _ := service.FixedExpensesOverview(ctx)
	occurrence := overview.Occurrences[0]
	if occurrence.DueDate != "2026-08-31" {
		t.Fatalf("due date=%s", occurrence.DueDate)
	}
	if _, err := service.UpdateFixedExpense(ctx, expense.ID, FixedExpenseInput{Description: "Internet", AmountCents: 150, DueDay: 30, AccountID: account.ID, CategoryID: "bills"}); err != nil {
		t.Fatal(err)
	}
	overview, _ = service.FixedExpensesOverview(ctx)
	if overview.Occurrences[0].ExpectedAmountCents != 100 || overview.Occurrences[0].DueDate != "2026-08-31" {
		t.Fatalf("current occurrence was rewritten: %+v", overview.Occurrences[0])
	}
	if err := service.DismissFixedExpenseOccurrence(ctx, occurrence.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.ReopenFixedExpenseOccurrence(ctx, occurrence.ID); err != nil {
		t.Fatal(err)
	}
}

func TestSetBalancesHidden_PersistsWithProfile(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	if _, err := service.SkipOnboarding(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetBalancesHidden(ctx, true); err != nil {
		t.Fatal(err)
	}
	boot, err := service.Bootstrap(ctx)
	if err != nil || boot.Profile == nil || !boot.Profile.BalancesHidden {
		t.Fatalf("profile=%+v err=%v", boot.Profile, err)
	}
}

func TestDueDate_ClampsAtEndOfMonth(t *testing.T) {
	if got := dueDate(time.Date(2026, time.February, 1, 0, 0, 0, 0, time.Local), 31); got != "2026-02-28" {
		t.Fatalf("date=%s", got)
	}
}
