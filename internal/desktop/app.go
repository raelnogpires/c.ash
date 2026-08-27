// Package desktop exposes a narrow, user-safe Wails façade.
package desktop

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"c.ash/internal/application"
	"c.ash/internal/domain"
	"c.ash/internal/storage"
	"c.ash/internal/updater"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx            context.Context
	service        *application.Service
	updater        *updater.Manager
	onThemeChanged func(domain.Theme)
	version        string
}

func New(service *application.Service, updaterManager *updater.Manager, onThemeChanged ...func(domain.Theme)) *App {
	app := &App{service: service, updater: updaterManager, version: "dev"}
	if len(onThemeChanged) > 0 {
		app.onThemeChanged = onThemeChanged[0]
	}
	return app
}

func (a *App) SetVersion(version string) {
	if strings.TrimSpace(version) != "" {
		a.version = version
	}
}

type OperationResult struct {
	Cancelled bool   `json:"cancelled"`
	Success   bool   `json:"success"`
	Path      string `json:"path,omitempty"`
}

type BackupDialogResult struct {
	Cancelled bool               `json:"cancelled"`
	Success   bool               `json:"success"`
	Backup    storage.BackupInfo `json:"backup"`
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	if !a.service.SecurityStatus().Locked {
		go func() {
			status, err := a.service.RunAutomaticBackup(context.Background(), a.version)
			if err != nil {
				log.Printf("automatic backup failed: %v", redact(err.Error()))
			}
			runtime.EventsEmit(ctx, "backup:status", status)
		}()
	}
	if a.updater == nil {
		return
	}
	a.updater.Subscribe(func(status updater.Status) { runtime.EventsEmit(ctx, "update:status", status) })
	go func() { _, _ = a.updater.Check(context.Background(), false) }()
}

func (a *App) SecurityStatus() storage.SecurityStatus { return a.service.SecurityStatus() }

func (a *App) UnlockDatabase(in application.UnlockInput) (storage.SecurityStatus, error) {
	err := a.service.UnlockDatabase(a.context(), in)
	if err == nil {
		go func() {
			status, _ := a.service.RunAutomaticBackup(context.Background(), a.version)
			runtime.EventsEmit(a.context(), "backup:status", status)
		}()
	}
	return a.service.SecurityStatus(), safe(err)
}

func (a *App) EnableEncryption(in application.EncryptionInput) (storage.EncryptionResult, error) {
	result, err := a.service.EnableEncryption(a.context(), in, a.version)
	return result, safe(err)
}

func (a *App) ChangeEncryptionPassword(in application.ChangeEncryptionPasswordInput) error {
	return safe(a.service.ChangeEncryptionPassword(a.context(), in, a.version))
}

func (a *App) RecoverEncryption(in application.RecoverEncryptionInput) (storage.SecurityStatus, error) {
	err := a.service.RecoverEncryption(a.context(), in, a.version)
	return a.service.SecurityStatus(), safe(err)
}

func (a *App) DisableEncryption(in application.UnlockInput) (storage.SecurityStatus, error) {
	err := a.service.DisableEncryption(a.context(), in, a.version)
	return a.service.SecurityStatus(), safe(err)
}

func (a *App) BackupStatus() (storage.BackupStatus, error) {
	status, err := a.service.BackupStatus()
	return status, safe(err)
}

func (a *App) CreateBackup() (BackupDialogResult, error) {
	status, err := a.service.BackupStatus()
	if err != nil {
		return BackupDialogResult{}, safe(err)
	}
	path, err := runtime.SaveFileDialog(a.context(), runtime.SaveDialogOptions{Title: "Salvar backup", DefaultDirectory: status.Folder, DefaultFilename: "cash-backup.cashbackup", CanCreateDirectories: true, Filters: []runtime.FileFilter{{DisplayName: "Backup do [c]ash (*.cashbackup)", Pattern: "*.cashbackup"}}})
	if err != nil {
		return BackupDialogResult{}, safe(err)
	}
	if path == "" {
		return BackupDialogResult{Cancelled: true}, nil
	}
	backup, err := a.service.CreateBackup(a.context(), path, a.version)
	return BackupDialogResult{Success: err == nil, Backup: backup}, safe(err)
}

func (a *App) ChooseBackupFolder() (OperationResult, error) {
	status, err := a.service.BackupStatus()
	if err != nil {
		return OperationResult{}, safe(err)
	}
	path, err := runtime.OpenDirectoryDialog(a.context(), runtime.OpenDialogOptions{Title: "Escolher pasta de backups", DefaultDirectory: status.Folder, CanCreateDirectories: true})
	if err != nil {
		return OperationResult{}, safe(err)
	}
	if path == "" {
		return OperationResult{Cancelled: true}, nil
	}
	err = a.service.SetBackupFolder(path)
	return OperationResult{Success: err == nil, Path: path}, safe(err)
}

func (a *App) ResetBackupFolder() (storage.BackupStatus, error) {
	err := a.service.ResetBackupFolder()
	if err != nil {
		return storage.BackupStatus{}, safe(err)
	}
	status, err := a.service.BackupStatus()
	return status, safe(err)
}

func (a *App) InspectBackup() (BackupDialogResult, error) {
	status, _ := a.service.BackupStatus()
	path, err := runtime.OpenFileDialog(a.context(), runtime.OpenDialogOptions{Title: "Selecionar backup", DefaultDirectory: status.Folder, Filters: []runtime.FileFilter{{DisplayName: "Backup do [c]ash (*.cashbackup)", Pattern: "*.cashbackup"}}})
	if err != nil {
		return BackupDialogResult{}, safe(err)
	}
	if path == "" {
		return BackupDialogResult{Cancelled: true}, nil
	}
	backup, err := a.service.InspectBackup(path)
	return BackupDialogResult{Success: err == nil, Backup: backup}, safe(err)
}

func (a *App) RestoreBackup(in application.RestoreBackupInput) (BackupDialogResult, error) {
	backup, err := a.service.RestoreBackup(a.context(), in, a.version)
	return BackupDialogResult{Success: err == nil, Backup: backup}, safe(err)
}

func (a *App) ExportData(format string) (OperationResult, error) {
	exportFormat := storage.ExportFormat(format)
	if exportFormat != storage.ExportCSV && exportFormat != storage.ExportJSON {
		return OperationResult{}, safe(errors.New("unsupported export format"))
	}
	extension := "." + format
	path, err := runtime.SaveFileDialog(a.context(), runtime.SaveDialogOptions{Title: "Exportar dados", DefaultFilename: "cash-export" + extension, CanCreateDirectories: true, Filters: []runtime.FileFilter{{DisplayName: strings.ToUpper(format) + " (*" + extension + ")", Pattern: "*" + extension}}})
	if err != nil {
		return OperationResult{}, safe(err)
	}
	if path == "" {
		return OperationResult{Cancelled: true}, nil
	}
	if !strings.EqualFold(filepath.Ext(path), extension) {
		path += extension
	}
	err = a.service.ExportData(a.context(), exportFormat, path, a.version)
	return OperationResult{Success: err == nil, Path: path}, safe(err)
}

func (a *App) GetUpdateStatus() updater.Status {
	if a.updater == nil {
		return updater.Status{State: updater.Disabled, Message: "Atualizações indisponíveis nesta instalação."}
	}
	return a.updater.Status()
}

func (a *App) CheckForUpdates() (updater.Status, error) {
	if a.updater == nil {
		return a.GetUpdateStatus(), nil
	}
	return a.updater.Check(a.context(), true)
}

func (a *App) InstallUpdate() (updater.Status, error) {
	if a.updater == nil {
		return a.GetUpdateStatus(), errors.New("Atualizações indisponíveis nesta instalação.")
	}
	status, err := a.updater.Install(a.context())
	if err != nil {
		return status, err
	}
	go func() { time.Sleep(250 * time.Millisecond); runtime.Quit(a.context()) }()
	return status, nil
}

func (a *App) Bootstrap() (application.Bootstrap, error) {
	v, err := a.service.Bootstrap(a.context())
	return v, safe(err)
}
func (a *App) CompleteOnboarding(in application.OnboardingInput) (domain.Profile, error) {
	v, err := a.service.CompleteOnboarding(a.context(), in)
	if err == nil && a.onThemeChanged != nil {
		a.onThemeChanged(v.Theme)
	}
	return v, safe(err)
}
func (a *App) SkipOnboarding() (domain.Profile, error) {
	v, err := a.service.SkipOnboarding(a.context())
	return v, safe(err)
}
func (a *App) CreateAccount(in application.AccountInput) (domain.Account, error) {
	v, err := a.service.CreateAccount(a.context(), in)
	return v, safe(err)
}
func (a *App) UpdateAccount(id string, in application.AccountInput) (domain.Account, error) {
	v, err := a.service.UpdateAccount(a.context(), id, in)
	return v, safe(err)
}
func (a *App) AdjustAccountBalance(id string, in application.BalanceAdjustmentInput) (domain.Transaction, error) {
	v, err := a.service.AdjustAccountBalance(a.context(), id, in)
	return v, safe(err)
}
func (a *App) Planning(month string) (domain.Planning, error) {
	v, err := a.service.Planning(a.context(), month)
	return v, safe(err)
}
func (a *App) SetMonthlyBudget(in application.MonthlyBudgetInput) (domain.MonthlyBudget, error) {
	v, err := a.service.SetMonthlyBudget(a.context(), in)
	return v, safe(err)
}
func (a *App) CreateCategory(in application.CategoryInput) (domain.Category, error) {
	v, err := a.service.CreateCategory(a.context(), in)
	return v, safe(err)
}
func (a *App) RenameCategory(id string, in application.CategoryInput) (domain.Category, error) {
	v, err := a.service.RenameCategory(a.context(), id, in)
	return v, safe(err)
}
func (a *App) ArchiveCategory(id string) error {
	return safe(a.service.ArchiveCategory(a.context(), id))
}
func (a *App) RestoreCategory(id string) error {
	return safe(a.service.RestoreCategory(a.context(), id))
}
func (a *App) SaveGoal(id string, in application.GoalInput) (domain.Goal, error) {
	v, err := a.service.SaveGoal(a.context(), id, in)
	return v, safe(err)
}
func (a *App) ArchiveGoal(id string) error { return safe(a.service.ArchiveGoal(a.context(), id)) }
func (a *App) SetGoalAllocations(id string, in []application.GoalAllocationInput) (domain.Goal, error) {
	v, err := a.service.SetGoalAllocations(a.context(), id, in)
	return v, safe(err)
}
func (a *App) DeleteAccount(id string) error {
	return safe(a.service.DeleteAccount(a.context(), id))
}
func (a *App) CreateTransaction(in application.TransactionInput) (domain.Transaction, error) {
	v, err := a.service.CreateTransaction(a.context(), in)
	return v, safe(err)
}
func (a *App) TransactionOccurrences() ([]domain.TransactionOccurrence, error) {
	v, err := a.service.TransactionOccurrences(a.context())
	return v, safe(err)
}
func (a *App) ConfirmTransactionOccurrence(id string) (domain.Transaction, error) {
	v, err := a.service.ConfirmTransactionOccurrence(a.context(), id)
	return v, safe(err)
}
func (a *App) DismissTransactionOccurrence(id string) error {
	return safe(a.service.DismissTransactionOccurrence(a.context(), id))
}
func (a *App) ArchiveRecurrence(id string) error {
	return safe(a.service.ArchiveRecurrence(a.context(), id))
}
func (a *App) UpdateTransaction(id string, in application.TransactionInput) (domain.Transaction, error) {
	v, err := a.service.UpdateTransaction(a.context(), id, in)
	return v, safe(err)
}
func (a *App) TrashTransaction(id string) error {
	return safe(a.service.TrashTransaction(a.context(), id))
}
func (a *App) RestoreTransaction(id string) error {
	return safe(a.service.RestoreTransaction(a.context(), id))
}
func (a *App) DeleteTransactionPermanently(id string) error {
	return safe(a.service.DeleteTransactionPermanently(a.context(), id))
}
func (a *App) EmptyTransactionTrash() error {
	return safe(a.service.EmptyTransactionTrash(a.context()))
}
func (a *App) ListTransactions() ([]domain.Transaction, error) {
	v, err := a.service.ListTransactions(a.context())
	return v, safe(err)
}
func (a *App) SearchTransactions(filter domain.TransactionFilter) ([]domain.Transaction, error) {
	v, err := a.service.SearchTransactions(a.context(), filter)
	return v, safe(err)
}
func (a *App) ListTrashedTransactions() ([]domain.Transaction, error) {
	v, err := a.service.ListTrashedTransactions(a.context())
	return v, safe(err)
}
func (a *App) CreditCardsOverview() (application.CreditCardsOverview, error) {
	v, err := a.service.CreditCardsOverview(a.context())
	return v, safe(err)
}
func (a *App) PayCreditCardInvoice(id string, in application.CreditCardPaymentInput) (domain.CreditCardPayment, error) {
	v, err := a.service.PayCreditCardInvoice(a.context(), id, in)
	return v, safe(err)
}
func (a *App) ImportBankStatement(in application.BankStatementInput) (application.BankStatementImportResult, error) {
	v, err := a.service.ImportBankStatement(a.context(), in)
	return v, safe(err)
}
func (a *App) SetTheme(theme domain.Theme) (domain.Profile, error) {
	v, err := a.service.SetTheme(a.context(), theme)
	if err == nil && a.onThemeChanged != nil {
		a.onThemeChanged(v.Theme)
	}
	return v, safe(err)
}
func (a *App) SetBalancesHidden(hidden bool) (domain.Profile, error) {
	v, err := a.service.SetBalancesHidden(a.context(), hidden)
	return v, safe(err)
}
func (a *App) FixedExpensesOverview() (application.FixedExpensesOverview, error) {
	v, err := a.service.FixedExpensesOverview(a.context())
	return v, safe(err)
}
func (a *App) CreateFixedExpense(in application.FixedExpenseInput) (domain.FixedExpense, error) {
	v, err := a.service.CreateFixedExpense(a.context(), in)
	return v, safe(err)
}
func (a *App) UpdateFixedExpense(id string, in application.FixedExpenseInput) (domain.FixedExpense, error) {
	v, err := a.service.UpdateFixedExpense(a.context(), id, in)
	return v, safe(err)
}
func (a *App) ArchiveFixedExpense(id string) error {
	return safe(a.service.ArchiveFixedExpense(a.context(), id))
}
func (a *App) RestoreFixedExpense(id string) error {
	return safe(a.service.RestoreFixedExpense(a.context(), id))
}
func (a *App) ConfirmFixedExpenseOccurrence(id string, in application.ConfirmFixedExpenseOccurrenceInput) (domain.Transaction, error) {
	v, err := a.service.ConfirmFixedExpenseOccurrence(a.context(), id, in)
	return v, safe(err)
}
func (a *App) DismissFixedExpenseOccurrence(id string) error {
	return safe(a.service.DismissFixedExpenseOccurrence(a.context(), id))
}
func (a *App) ReopenFixedExpenseOccurrence(id string) error {
	return safe(a.service.ReopenFixedExpenseOccurrence(a.context(), id))
}
func (a *App) context() context.Context {
	if a.ctx == nil {
		return context.Background()
	}
	return a.ctx
}

func safe(err error) error {
	if err == nil {
		return nil
	}
	messages := []struct {
		target  error
		message string
	}{
		{domain.ErrBlankName, "Informe um nome."}, {domain.ErrBlankDescription, "Informe uma descrição."},
		{domain.ErrInvalidAmount, "O valor deve ser maior que zero."}, {domain.ErrInvalidDate, "Informe uma data válida."},
		{domain.ErrFutureDate, "A data não pode estar no futuro."}, {domain.ErrBeforeOpening, "A data não pode ser anterior à abertura da conta."},
		{domain.ErrInvalidAccountType, "Escolha um tipo de conta válido."}, {domain.ErrInvalidKind, "Escolha receita, despesa ou transferência."},
		{domain.ErrInvalidTheme, "Escolha um tema válido."}, {domain.ErrUnknownAccount, "A conta selecionada não existe."},
		{domain.ErrAccountInUse, "Esta conta possui movimentações e não pode ser removida."},
		{domain.ErrUnknownCategory, "A categoria selecionada não existe."}, {domain.ErrCategoryKind, "A categoria não corresponde ao tipo da movimentação."},
		{domain.ErrDuplicateCategory, "Já existe uma categoria com esse nome."}, {domain.ErrCategoryArchived, "Esta categoria está arquivada."},
		{domain.ErrSameTransferAccount, "Escolha contas diferentes para a transferência."}, {domain.ErrTransferCategory, "Transferências não usam categoria."},
		{domain.ErrSavingsNegative, "Essa operação deixaria uma poupança com saldo negativo."}, {domain.ErrUnknownTransaction, "A movimentação não existe mais."},
		{domain.ErrTransactionActive, "A movimentação já está ativa."}, {domain.ErrTransactionTrashed, "A movimentação já foi removida."},
		{domain.ErrInvalidDueDay, "Informe um dia entre 1 e 31."}, {domain.ErrUnknownFixedExpense, "Esta despesa fixa não existe mais."},
		{domain.ErrUnknownOccurrence, "Esta previsão não existe mais."}, {domain.ErrOccurrenceClosed, "Esta previsão já foi concluída ou dispensada."},
		{domain.ErrFixedExpenseArchived, "Esta despesa fixa está arquivada."},
		{domain.ErrInvalidStatement, "O arquivo não é um extrato PDF, OFX ou CSV válido. PDFs protegidos por senha não são compatíveis."},
		{domain.ErrUnsupportedBank, "Escolha Itaú, Bradesco ou Inter."},
		{domain.ErrStatementEmpty, "Não encontramos movimentações nesse extrato. Se for PDF, o arquivo não pode ser apenas uma imagem escaneada."},
		{domain.ErrStatementTooLarge, "O extrato deve ter no máximo 15 MB."},
		{domain.ErrInvalidCreditLimit, "Informe um limite de crédito maior que zero."},
		{domain.ErrInvalidInstallments, "Escolha entre 1 e 48 parcelas."},
		{domain.ErrCardTransaction, "Cartões aceitam somente despesas; pague faturas pela área de cartões."},
		{domain.ErrInvoiceLocked, "Esta compra pertence a uma fatura consolidada e não pode mais ser alterada."},
		{domain.ErrUnknownInvoice, "Esta fatura não existe mais."},
		{domain.ErrInvoiceNotPayable, "Esta fatura já foi paga ou transferida para a seguinte."},
		{domain.ErrInvalidPaymentAccount, "Escolha uma conta corrente, poupança ou dinheiro para pagar a fatura."},
		{domain.ErrInvoiceOverpayment, "O pagamento não pode superar o valor em aberto da fatura."},
		{domain.ErrOpeningBalanceLocked, "Use um ajuste de saldo para contas que já possuem movimentações."},
		{domain.ErrAdjustmentReason, "Informe o motivo do ajuste de saldo."},
		{domain.ErrNoBalanceChange, "O saldo informado já é o saldo atual da conta."},
		{domain.ErrInvalidBudget, "Revise o mês e os limites do orçamento."},
		{domain.ErrInvalidGoal, "Revise o nome, tipo e valor da meta."},
		{domain.ErrUnknownGoal, "Esta meta não existe mais."},
		{domain.ErrAllocationLimit, "As reservas desta conta não podem superar o saldo disponível."},
		{domain.ErrInvalidSplit, "As divisões devem somar exatamente o valor da movimentação."},
		{domain.ErrUnknownLedgerOccurrence, "Esta ocorrência não existe mais."},
		{storage.ErrLocked, "Desbloqueie o banco de dados para continuar."},
		{storage.ErrInvalidCredential, "Senha ou chave de recuperação incorreta."},
		{storage.ErrWeakPassword, "Use uma senha com pelo menos 12 caracteres."},
		{storage.ErrPasswordMismatch, "A confirmação da senha não corresponde."},
		{storage.ErrEncryptionEnabled, "A criptografia já está ativada."},
		{storage.ErrEncryptionDisabled, "A criptografia não está ativada."},
		{storage.ErrCorruptKeyFile, "Os metadados de criptografia estão corrompidos. Use um backup válido."},
		{storage.ErrInvalidBackup, "O arquivo de backup está inválido ou foi alterado."},
		{storage.ErrUnsupportedBackup, "Este formato de backup não é compatível com esta versão."},
		{storage.ErrNewerSchema, "Este backup foi criado por uma versão mais nova do aplicativo."},
	}
	for _, item := range messages {
		if errors.Is(err, item.target) {
			return errors.New(item.message)
		}
	}
	log.Printf("operation failed: %T: %v", err, redact(err.Error()))
	return errors.New("Não foi possível concluir a operação. Tente novamente.")
}

func redact(_ string) string { return fmt.Sprintf("diagnostic omitted to protect financial data") }
