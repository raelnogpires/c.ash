// Package desktop exposes a narrow, user-safe Wails façade.
package desktop

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"c.ash/internal/application"
	"c.ash/internal/domain"
	"c.ash/internal/updater"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx            context.Context
	service        *application.Service
	updater        *updater.Manager
	onThemeChanged func(domain.Theme)
}

func New(service *application.Service, updaterManager *updater.Manager, onThemeChanged ...func(domain.Theme)) *App {
	app := &App{service: service, updater: updaterManager}
	if len(onThemeChanged) > 0 {
		app.onThemeChanged = onThemeChanged[0]
	}
	return app
}
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	if a.updater == nil {
		return
	}
	a.updater.Subscribe(func(status updater.Status) { runtime.EventsEmit(ctx, "update:status", status) })
	go func() { _, _ = a.updater.Check(context.Background(), false) }()
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
func (a *App) DeleteAccount(id string) error {
	return safe(a.service.DeleteAccount(a.context(), id))
}
func (a *App) CreateTransaction(in application.TransactionInput) (domain.Transaction, error) {
	v, err := a.service.CreateTransaction(a.context(), in)
	return v, safe(err)
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
func (a *App) ListTransactions() ([]domain.Transaction, error) {
	v, err := a.service.ListTransactions(a.context())
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
		{domain.ErrSameTransferAccount, "Escolha contas diferentes para a transferência."}, {domain.ErrTransferCategory, "Transferências não usam categoria."},
		{domain.ErrSavingsNegative, "Essa operação deixaria uma poupança com saldo negativo."}, {domain.ErrUnknownTransaction, "A movimentação não existe mais."},
		{domain.ErrTransactionActive, "A movimentação já está ativa."}, {domain.ErrTransactionTrashed, "A movimentação já foi removida."},
		{domain.ErrInvalidDueDay, "Informe um dia entre 1 e 31."}, {domain.ErrUnknownFixedExpense, "Esta despesa fixa não existe mais."},
		{domain.ErrUnknownOccurrence, "Esta previsão não existe mais."}, {domain.ErrOccurrenceClosed, "Esta previsão já foi concluída ou dispensada."},
		{domain.ErrFixedExpenseArchived, "Esta despesa fixa está arquivada."},
		{domain.ErrInvalidStatement, "O arquivo não é um PDF de extrato válido ou está protegido por senha."},
		{domain.ErrUnsupportedBank, "Escolha Itaú, Bradesco ou Inter."},
		{domain.ErrStatementEmpty, "Não encontramos movimentações nesse extrato. PDFs escaneados ainda não são compatíveis."},
		{domain.ErrStatementTooLarge, "O PDF deve ter no máximo 15 MB."},
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
