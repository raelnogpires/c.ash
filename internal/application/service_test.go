package application

import (
	"context"
	"encoding/base64"
	"errors"
	"sort"
	"strings"
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

func TestTransactionTrash_ListRestorePermanentDeleteAndEmpty(t *testing.T) {
	service, store := testService(t)
	ctx := context.Background()
	account, err := service.CreateAccount(ctx, AccountInput{Name: "Principal", Type: domain.AccountChecking, OpeningBalanceCents: 10000, OpeningDate: "2026-08-01"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.CreateTransaction(ctx, TransactionInput{Kind: domain.Expense, AmountCents: 1000, AccountID: account.ID, CategoryID: "food", Description: "Almoço", OccurrenceDate: "2026-08-10"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.CreateTransaction(ctx, TransactionInput{Kind: domain.Income, AmountCents: 2000, AccountID: account.ID, CategoryID: "salary", Description: "Extra", OccurrenceDate: "2026-08-11"})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.TrashTransaction(ctx, first.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.TrashTransaction(ctx, second.ID); err != nil {
		t.Fatal(err)
	}
	trashed, err := service.ListTrashedTransactions(ctx)
	if err != nil || len(trashed) != 2 {
		t.Fatalf("trashed=%+v err=%v", trashed, err)
	}
	if err := service.RestoreTransaction(ctx, first.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteTransactionPermanently(ctx, first.ID); !errors.Is(err, domain.ErrTransactionActive) {
		t.Fatalf("active permanent delete error=%v", err)
	}
	if err := service.TrashTransaction(ctx, first.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteTransactionPermanently(ctx, first.ID); err != nil {
		t.Fatal(err)
	}
	if stored, err := store.Transaction(ctx, first.ID); err != nil || stored != nil {
		t.Fatalf("permanently deleted transaction=%+v err=%v", stored, err)
	}
	if err := service.DeleteTransactionPermanently(ctx, "missing"); !errors.Is(err, domain.ErrUnknownTransaction) {
		t.Fatalf("unknown permanent delete error=%v", err)
	}
	if err := service.EmptyTransactionTrash(ctx); err != nil {
		t.Fatal(err)
	}
	if trashed, err := service.ListTrashedTransactions(ctx); err != nil || len(trashed) != 0 {
		t.Fatalf("trash after empty=%+v err=%v", trashed, err)
	}
	if stored, err := store.Transaction(ctx, second.ID); err != nil || stored != nil {
		t.Fatalf("emptied transaction=%+v err=%v", stored, err)
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

func TestImportBankStatement_DispatchesCaseInsensitivelyAndDeduplicatesAcrossFormats(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	account, err := service.CreateAccount(ctx, AccountInput{Name: "Principal", Type: domain.AccountChecking, OpeningDate: "2026-08-01"})
	if err != nil {
		t.Fatal(err)
	}
	csvData := "Data;Descrição;Valor\n02/08/2026;PIX JOAO;-25,90\n"
	first, err := service.ImportBankStatement(ctx, BankStatementInput{AccountID: account.ID, Bank: importer.BankInter, FileName: "agosto.CSV", Base64Data: base64.StdEncoding.EncodeToString([]byte(csvData))})
	if err != nil || first.ImportedCount != 1 {
		t.Fatalf("CSV result=%+v err=%v", first, err)
	}
	ofxData := "OFXHEADER:100\nDATA:OFXSGML\nVERSION:102\n<OFX><BANKTRANLIST><STMTTRN><DTPOSTED>20260802<TRNAMT>-25.90<NAME>PIX JOAO</STMTTRN></BANKTRANLIST></OFX>"
	second, err := service.ImportBankStatement(ctx, BankStatementInput{AccountID: account.ID, Bank: importer.BankInter, FileName: "agosto.OFX", Base64Data: base64.StdEncoding.EncodeToString([]byte(ofxData))})
	if err != nil || second.ImportedCount != 0 || second.DuplicateCount != 1 {
		t.Fatalf("OFX result=%+v err=%v", second, err)
	}
	txs, err := service.ListTransactions(ctx)
	if err != nil || len(txs) != 1 || txs[0].ImportBank != "inter" {
		t.Fatalf("transactions=%+v err=%v", txs, err)
	}
}

func TestImportBankStatement_ValidatesEnvelopeAndAccount(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	validCSV := base64.StdEncoding.EncodeToString([]byte("Data;Descrição;Valor\n01/08/2026;Teste;-10,00\n"))
	tests := []struct {
		name  string
		input BankStatementInput
		want  error
	}{
		{"unsupported extension", BankStatementInput{Bank: importer.BankItau, FileName: "statement.txt", Base64Data: validCSV}, domain.ErrInvalidStatement},
		{"invalid base64", BankStatementInput{Bank: importer.BankItau, FileName: "statement.csv", Base64Data: "%%%"}, domain.ErrInvalidStatement},
		{"unknown bank", BankStatementInput{Bank: "other", FileName: "statement.csv", Base64Data: validCSV}, domain.ErrUnsupportedBank},
		{"unknown account", BankStatementInput{AccountID: "missing", Bank: importer.BankItau, FileName: "statement.csv", Base64Data: validCSV}, domain.ErrUnknownAccount},
		{"encoded size limit", BankStatementInput{Bank: importer.BankItau, FileName: "statement.ofx", Base64Data: strings.Repeat("A", ((importer.MaxStatementSize+2)/3)*4+9)}, domain.ErrStatementTooLarge},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := service.ImportBankStatement(ctx, tc.input); !errors.Is(err, tc.want) {
				t.Fatalf("error=%v want=%v", err, tc.want)
			}
		})
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
	if _, err := service.CreateTransaction(ctx, TransactionInput{Kind: domain.Expense, AmountCents: 800, AccountID: account.ID, Description: "Conta", OccurrenceDate: "2026-08-12"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateAccount(ctx, account.ID, AccountInput{Name: "Reserva", Type: domain.AccountSavings, OpeningBalanceCents: 1000, OpeningDate: "2026-08-01"}); !errors.Is(err, domain.ErrSavingsNegative) {
		t.Fatalf("savings conversion error=%v", err)
	}
	if _, err := service.UpdateAccount(ctx, account.ID, AccountInput{Name: "Principal", Type: domain.AccountChecking, OpeningBalanceCents: 2000, OpeningDate: "2026-08-01"}); !errors.Is(err, domain.ErrOpeningBalanceLocked) {
		t.Fatalf("opening balance error=%v", err)
	}
	updated, err := service.UpdateAccount(ctx, account.ID, AccountInput{Name: " Principal nova ", Type: domain.AccountChecking, OpeningBalanceCents: 1000, OpeningDate: "2026-08-01"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Principal nova" || updated.CurrentBalanceCents != -100 || updated.CreatedAt != account.CreatedAt {
		t.Fatalf("updated=%+v", updated)
	}
}

func TestAdjustAccountBalance_CreatesAuditableExcludedTransaction(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	account, err := service.CreateAccount(ctx, AccountInput{Name: "Principal", Type: domain.AccountChecking, OpeningBalanceCents: 1000, OpeningDate: "2026-08-01"})
	if err != nil {
		t.Fatal(err)
	}
	adjustment, err := service.AdjustAccountBalance(ctx, account.ID, BalanceAdjustmentInput{TargetBalanceCents: 750, OccurrenceDate: "2026-08-16", Reason: "Conferência do extrato"})
	if err != nil {
		t.Fatal(err)
	}
	if adjustment.Kind != domain.Expense || adjustment.AmountCents != 250 || adjustment.Origin != domain.OriginAdjustment || adjustment.AdjustmentReason != "Conferência do extrato" {
		t.Fatalf("adjustment=%+v", adjustment)
	}
	bootstrap, err := service.Bootstrap(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if bootstrap.Accounts[0].CurrentBalanceCents != 750 || bootstrap.Dashboard.MonthlyExpenseCents != 0 || !bootstrap.Accounts[0].HasLedgerActivity {
		t.Fatalf("bootstrap=%+v", bootstrap)
	}
	if _, err := service.AdjustAccountBalance(ctx, account.ID, BalanceAdjustmentInput{TargetBalanceCents: 750, OccurrenceDate: "2026-08-16", Reason: "igual"}); !errors.Is(err, domain.ErrNoBalanceChange) {
		t.Fatalf("same balance error=%v", err)
	}
	if _, err := service.AdjustAccountBalance(ctx, account.ID, BalanceAdjustmentInput{TargetBalanceCents: 800, OccurrenceDate: "2026-08-16"}); !errors.Is(err, domain.ErrAdjustmentReason) {
		t.Fatalf("reason error=%v", err)
	}
}

func TestPlanning_BudgetsRolloverGoalsAndAllocationLimits(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	account, err := service.CreateAccount(ctx, AccountInput{Name: "Principal", Type: domain.AccountChecking, OpeningBalanceCents: 10000, OpeningDate: "2026-07-01"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateTransaction(ctx, TransactionInput{Kind: domain.Expense, AmountCents: 2000, AccountID: account.ID, CategoryID: "food", Description: "Mercado julho", OccurrenceDate: "2026-07-10"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetMonthlyBudget(ctx, MonthlyBudgetInput{ReferenceMonth: "2026-07", OverallLimitCents: 5000, CategoryLimits: []CategoryBudgetInput{{CategoryID: "food", LimitCents: 3000, Rollover: true}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateTransaction(ctx, TransactionInput{Kind: domain.Expense, AmountCents: 2500, AccountID: account.ID, CategoryID: "food", Description: "Mercado agosto", OccurrenceDate: "2026-08-10"}); err != nil {
		t.Fatal(err)
	}
	budget, err := service.SetMonthlyBudget(ctx, MonthlyBudgetInput{ReferenceMonth: "2026-08", OverallLimitCents: 6000, CategoryLimits: []CategoryBudgetInput{{CategoryID: "food", LimitCents: 3000, Rollover: true}}})
	if err != nil {
		t.Fatal(err)
	}
	limit := budget.CategoryLimits[0]
	if budget.SpentCents != 2500 || budget.RemainingCents != 3500 || limit.RolloverCents != 1000 || limit.AvailableCents != 1500 || limit.Exceeded {
		t.Fatalf("budget=%+v", budget)
	}
	goal, err := service.SaveGoal(ctx, "", GoalInput{Name: "Reserva", Kind: domain.GoalEmergencyReserve, TargetCents: 10000})
	if err != nil {
		t.Fatal(err)
	}
	goal, err = service.SetGoalAllocations(ctx, goal.ID, []GoalAllocationInput{{AccountID: account.ID, AmountCents: 5000}})
	if err != nil || goal.AllocatedCents != 5000 || goal.ProgressPercent != 50 {
		t.Fatalf("goal=%+v err=%v", goal, err)
	}
	if _, err := service.SetGoalAllocations(ctx, goal.ID, []GoalAllocationInput{{AccountID: account.ID, AmountCents: 6000}}); !errors.Is(err, domain.ErrAllocationLimit) {
		t.Fatalf("allocation error=%v", err)
	}
	bootstrap, err := service.Bootstrap(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if bootstrap.Dashboard.EligibleBalanceCents != 5500 || bootstrap.Dashboard.ReservedValueCents != 5000 || bootstrap.Dashboard.FreeValueCents != 500 || bootstrap.Dashboard.SafelySpendableCents != 500 {
		t.Fatalf("dashboard=%+v", bootstrap.Dashboard)
	}
}

func TestTransactionCapabilities_SplitsTagsInstallmentsAndRecurrence(t *testing.T) {
	service, store := testService(t)
	ctx := context.Background()
	account, err := service.CreateAccount(ctx, AccountInput{Name: "Principal", Type: domain.AccountChecking, OpeningBalanceCents: 10000, OpeningDate: "2026-08-01"})
	if err != nil {
		t.Fatal(err)
	}
	tx, err := service.CreateTransaction(ctx, TransactionInput{Kind: domain.Expense, AmountCents: 1000, AccountID: account.ID, CategoryID: "food", Description: "Compras", OccurrenceDate: "2026-08-16", Tags: []string{"Casa", "casa", " urgente "}, Splits: []TransactionSplitInput{{CategoryID: "food", AmountCents: 600}, {CategoryID: "bills", SubcategoryName: "Energia", AmountCents: 400}}})
	if err != nil {
		t.Fatal(err)
	}
	stored, err := store.Transaction(ctx, tx.ID)
	if err != nil || stored == nil || len(stored.Tags) != 2 || len(stored.Splits) != 2 || stored.Splits[1].SubcategoryName != "Energia" {
		t.Fatalf("stored=%+v err=%v", stored, err)
	}
	if _, err := service.CreateTransaction(ctx, TransactionInput{Kind: domain.Expense, AmountCents: 1000, AccountID: account.ID, Description: "Inválida", OccurrenceDate: "2026-08-16", Splits: []TransactionSplitInput{{CategoryID: "food", AmountCents: 999}}}); !errors.Is(err, domain.ErrInvalidSplit) {
		t.Fatalf("split error=%v", err)
	}
	installment, err := service.CreateTransaction(ctx, TransactionInput{Kind: domain.Expense, AmountCents: 100, AccountID: account.ID, CategoryID: "food", Description: "Curso", OccurrenceDate: "2026-08-16", InstallmentCount: 3})
	if err != nil || installment.AmountCents != 34 {
		t.Fatalf("installment=%+v err=%v", installment, err)
	}
	occurrences, err := service.TransactionOccurrences(ctx)
	if err != nil || len(occurrences) != 2 || occurrences[0].AmountCents != 33 {
		t.Fatalf("occurrences=%+v err=%v", occurrences, err)
	}
	service.now = func() time.Time { return time.Date(2026, time.October, 20, 12, 0, 0, 0, time.Local) }
	if _, err := service.ConfirmTransactionOccurrence(ctx, occurrences[0].ID); err != nil {
		t.Fatal(err)
	}
	recurring, err := service.CreateTransaction(ctx, TransactionInput{Kind: domain.Income, AmountCents: 500, AccountID: account.ID, CategoryID: "salary", Description: "Mensal", OccurrenceDate: "2026-08-16", MonthlyRecurrence: true})
	if err != nil || recurring.RecurrenceRuleID == "" {
		t.Fatalf("recurring=%+v err=%v", recurring, err)
	}
	if got := addMonthsClamped("2024-01-31", 1); got != "2024-02-29" {
		t.Fatalf("month end=%s", got)
	}
}

func TestSearchTransactions_CombinesFiltersAcrossLedgerStatuses(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	account, err := service.CreateAccount(ctx, AccountInput{Name: "Principal", Type: domain.AccountChecking, OpeningBalanceCents: 10000, OpeningDate: "2026-08-01"})
	if err != nil {
		t.Fatal(err)
	}
	lunch, err := service.CreateTransaction(ctx, TransactionInput{Kind: domain.Expense, AmountCents: 1200, AccountID: account.ID, CategoryID: "food", Description: "Almoço equipe", OccurrenceDate: "2026-08-10", Tags: []string{"trabalho"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateTransaction(ctx, TransactionInput{Kind: domain.Income, AmountCents: 5000, AccountID: account.ID, CategoryID: "salary", Description: "Salário", OccurrenceDate: "2026-08-15"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateTransaction(ctx, TransactionInput{Kind: domain.Expense, AmountCents: 300, AccountID: account.ID, CategoryID: "food", Description: "Parcelado", OccurrenceDate: "2026-08-16", InstallmentCount: 2}); err != nil {
		t.Fatal(err)
	}
	if err := service.TrashTransaction(ctx, lunch.ID); err != nil {
		t.Fatal(err)
	}
	items, err := service.SearchTransactions(ctx, domain.TransactionFilter{Text: "equipe", AccountID: account.ID, CategoryID: "food", Tag: "TRABALHO", Kind: domain.Expense, Status: "trashed", MinimumAmountCents: 1000, MaximumAmountCents: 1300})
	if err != nil || len(items) != 1 || items[0].ID != lunch.ID {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	pending, err := service.SearchTransactions(ctx, domain.TransactionFilter{Status: "pending", Recurrence: "nonrecurring"})
	if err != nil || len(pending) != 1 || !pending[0].Pending {
		t.Fatalf("pending=%+v err=%v", pending, err)
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

func TestFixedExpenseArchiveRestore_SkipsFullyArchivedMonths(t *testing.T) {
	ctx := context.Background()
	store, err := storage.Open(ctx, t.TempDir()+"/cash.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	now := time.Date(2026, time.August, 10, 12, 0, 0, 0, time.Local)
	service := New(store, func() time.Time { return now })
	if _, err := service.CompleteOnboarding(ctx, OnboardingInput{
		DisplayName: "Ana",
		Currency:    "BRL",
		Theme:       domain.ThemeLight,
		FirstAccount: AccountInput{
			Name:                "Principal",
			Type:                domain.AccountChecking,
			OpeningDate:         "2026-08-01",
			OpeningBalanceCents: 10000,
		},
	}); err != nil {
		t.Fatal(err)
	}
	boot, err := service.Bootstrap(ctx)
	if err != nil {
		t.Fatal(err)
	}
	expense, err := service.CreateFixedExpense(ctx, FixedExpenseInput{
		Description: "Internet",
		AmountCents: 100,
		DueDay:      10,
		AccountID:   boot.Accounts[0].ID,
		CategoryID:  "bills",
	})
	if err != nil {
		t.Fatal(err)
	}
	initial, err := service.FixedExpensesOverview(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.DismissFixedExpenseOccurrence(ctx, initial.Occurrences[0].ID); err != nil {
		t.Fatal(err)
	}

	now = time.Date(2026, time.October, 20, 12, 0, 0, 0, time.Local)
	if err := service.ArchiveFixedExpense(ctx, expense.ID); err != nil {
		t.Fatal(err)
	}
	assertFixedExpenseMonths(t, store, []string{"2026-08", "2026-09", "2026-10"})
	now = time.Date(2026, time.December, 20, 12, 0, 0, 0, time.Local)
	if err := service.ArchiveFixedExpense(ctx, expense.ID); err != nil {
		t.Fatal(err)
	}
	assertFixedExpenseMonths(t, store, []string{"2026-08", "2026-09", "2026-10"})
	archivedOccurrences, err := store.FixedExpenseOccurrences(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var augustOccurrence *domain.FixedExpenseOccurrence
	for index := range archivedOccurrences {
		if archivedOccurrences[index].ReferenceMonth == "2026-08" {
			augustOccurrence = &archivedOccurrences[index]
			break
		}
	}
	if augustOccurrence == nil || augustOccurrence.Status != domain.FixedExpenseDismissed {
		t.Fatalf("archived occurrence was rewritten: %+v", augustOccurrence)
	}

	now = time.Date(2027, time.January, 5, 12, 0, 0, 0, time.Local)
	if err := service.RestoreFixedExpense(ctx, expense.ID); err != nil {
		t.Fatal(err)
	}
	assertFixedExpenseMonths(t, store, []string{"2026-08", "2026-09", "2026-10", "2027-01"})
	restored, err := store.FixedExpenses(ctx)
	if err != nil {
		t.Fatal(err)
	}
	januaryCursor := now.UTC().Format(time.RFC3339Nano)
	if restored[0].ArchivedAt != "" || restored[0].OccurrenceStartAt != januaryCursor {
		t.Fatalf("restored expense=%+v", restored[0])
	}

	if _, err := service.FixedExpensesOverview(ctx); err != nil {
		t.Fatal(err)
	}
	boot, err = service.Bootstrap(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if boot.Dashboard.TotalBalanceCents != 10000 || boot.Dashboard.PendingFixedExpensesCents != 300 || boot.Dashboard.PendingFixedExpenseCount != 3 || boot.Dashboard.AvailableBalanceCents != 9700 {
		t.Fatalf("restored dashboard includes archived-gap occurrences: %+v", boot.Dashboard)
	}
	if _, err := service.FixedExpensesOverview(ctx); err != nil {
		t.Fatal(err)
	}
	assertFixedExpenseMonths(t, store, []string{"2026-08", "2026-09", "2026-10", "2027-01"})

	now = time.Date(2027, time.February, 2, 12, 0, 0, 0, time.Local)
	if _, err := service.Bootstrap(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := service.FixedExpensesOverview(ctx); err != nil {
		t.Fatal(err)
	}
	assertFixedExpenseMonths(t, store, []string{"2026-08", "2026-09", "2026-10", "2027-01", "2027-02"})

	now = time.Date(2027, time.March, 1, 12, 0, 0, 0, time.Local)
	if err := service.RestoreFixedExpense(ctx, expense.ID); err != nil {
		t.Fatal(err)
	}
	active, err := store.FixedExpenses(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if active[0].OccurrenceStartAt != januaryCursor {
		t.Fatalf("idempotent restore moved cursor from %q to %q", januaryCursor, active[0].OccurrenceStartAt)
	}
}

func assertFixedExpenseMonths(t *testing.T, store *storage.Store, want []string) {
	t.Helper()
	occurrences, err := store.FixedExpenseOccurrences(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(occurrences))
	for index, occurrence := range occurrences {
		got[index] = occurrence.ReferenceMonth
	}
	sort.Strings(got)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("occurrence months=%v want=%v", got, want)
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

func TestCreditCardWorkflow_OpeningInvoiceInstallmentsAndPartialPayment(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	bank, err := service.CreateAccount(ctx, AccountInput{Name: "Principal", Type: domain.AccountChecking, OpeningBalanceCents: 10000, OpeningDate: "2026-08-01"})
	if err != nil {
		t.Fatal(err)
	}
	card, err := service.CreateAccount(ctx, AccountInput{Name: "Inter Gold", Type: domain.AccountCreditCard, OpeningDate: "2026-08-01",
		CreditLimitCents: 20000, ClosingDay: 25, DueDay: 2, OpeningDebtCents: 1500, OpeningDebtDueDate: "2026-09-02"})
	if err != nil {
		t.Fatal(err)
	}
	purchase, err := service.CreateTransaction(ctx, TransactionInput{Kind: domain.Expense, AmountCents: 10001, AccountID: card.ID,
		CategoryID: "shopping", Description: "Notebook", OccurrenceDate: "2026-08-10", InstallmentCount: 3})
	if err != nil {
		t.Fatal(err)
	}
	overview, err := service.CreditCardsOverview(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(overview.Cards) != 1 || len(overview.Invoices) != 3 || overview.Cards[0].OutstandingCents != 11501 {
		t.Fatalf("overview=%+v", overview)
	}
	var firstPurchaseInvoice, openingInvoice *domain.CreditCardInvoice
	for index := range overview.Invoices {
		for _, item := range overview.Invoices[index].Installments {
			if item.TransactionID == purchase.ID && item.InstallmentNumber == 1 {
				firstPurchaseInvoice = &overview.Invoices[index]
			}
			if item.OpeningDebt {
				openingInvoice = &overview.Invoices[index]
			}
		}
	}
	var firstAmount int64
	if firstPurchaseInvoice != nil {
		for _, item := range firstPurchaseInvoice.Installments {
			if item.TransactionID == purchase.ID && item.InstallmentNumber == 1 {
				firstAmount = item.AmountCents
			}
		}
	}
	if firstPurchaseInvoice == nil || firstAmount != 3335 || openingInvoice == nil || openingInvoice.ID != firstPurchaseInvoice.ID {
		t.Fatalf("purchase invoice=%+v opening=%+v", firstPurchaseInvoice, openingInvoice)
	}
	if _, err := service.PayCreditCardInvoice(ctx, openingInvoice.ID, CreditCardPaymentInput{AccountID: bank.ID, AmountCents: 4836, OccurrenceDate: "2026-08-16"}); !errors.Is(err, domain.ErrInvoiceOverpayment) {
		t.Fatalf("overpayment error=%v", err)
	}
	payment, err := service.PayCreditCardInvoice(ctx, openingInvoice.ID, CreditCardPaymentInput{AccountID: bank.ID, AmountCents: 500, OccurrenceDate: "2026-08-16"})
	if err != nil {
		t.Fatal(err)
	}
	if payment.AmountCents != 500 || payment.TransactionID == "" {
		t.Fatalf("payment=%+v", payment)
	}
	if _, err := service.UpdateTransaction(ctx, payment.TransactionID, TransactionInput{}); !errors.Is(err, domain.ErrInvoiceLocked) {
		t.Fatalf("payment edit error=%v", err)
	}
	boot, err := service.Bootstrap(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if boot.Dashboard.AvailableBalanceCents != 9500 || boot.Dashboard.TotalBalanceCents != -1501 || boot.Dashboard.CreditCardDebtCents != 11001 || boot.Dashboard.MonthlyExpenseCents != 10001 {
		t.Fatalf("dashboard=%+v", boot.Dashboard)
	}
	closedService := New(service.store, func() time.Time { return time.Date(2026, 8, 25, 12, 0, 0, 0, time.Local) })
	if _, err := closedService.CreditCardsOverview(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := closedService.UpdateTransaction(ctx, purchase.ID, TransactionInput{Kind: domain.Expense, AmountCents: 10001, AccountID: card.ID, CategoryID: "shopping", Description: "Notebook Pro", OccurrenceDate: "2026-08-10", InstallmentCount: 3}); err != nil {
		t.Fatalf("metadata edit on closed schedule: %v", err)
	}
	if _, err := closedService.UpdateTransaction(ctx, purchase.ID, TransactionInput{Kind: domain.Expense, AmountCents: 9000, AccountID: card.ID, Description: "Notebook", OccurrenceDate: "2026-08-10", InstallmentCount: 3}); !errors.Is(err, domain.ErrInvoiceLocked) {
		t.Fatalf("closed schedule edit error=%v", err)
	}
	rolledService := New(service.store, func() time.Time { return time.Date(2026, 9, 3, 12, 0, 0, 0, time.Local) })
	rolled, err := rolledService.CreditCardsOverview(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var carry int64
	for _, invoice := range rolled.Invoices {
		if invoice.ClosingDate == "2026-09-25" {
			carry = invoice.CarryForwardCents
		}
	}
	if carry != 4335 {
		t.Fatalf("carry forward=%d invoices=%+v", carry, rolled.Invoices)
	}
}
