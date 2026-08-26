import type { Account, AccountInput, BackupDialogResult, BackupStatus, BalanceAdjustmentInput, BankStatementImportResult, BankStatementInput, BootstrapData, ChangeEncryptionPasswordInput, ConfirmFixedExpenseOccurrenceInput, CreditCardPayment, CreditCardPaymentInput, CreditCardsOverview, EncryptionInput, EncryptionResult, FixedExpense, FixedExpenseInput, FixedExpensesOverview, Goal, GoalAllocationInput, GoalInput, MonthlyBudget, MonthlyBudgetInput, OnboardingInput, OperationResult, Planning, Profile, RecoverEncryptionInput, RestoreBackupInput, SecurityStatus, Theme, Transaction, TransactionInput, UnlockInput, UpdateStatus } from './types'
import * as bindings from './wailsjs/go/desktop/App'
import { application } from './wailsjs/go/models'

interface CashAPI {
  Bootstrap(): Promise<BootstrapData>
  CompleteOnboarding(input: OnboardingInput): Promise<Profile>
  SkipOnboarding(): Promise<Profile>
  CreateAccount(input: AccountInput): Promise<Account>
  UpdateAccount(id: string, input: AccountInput): Promise<Account>
  AdjustAccountBalance(id: string, input: BalanceAdjustmentInput): Promise<Transaction>
  Planning(month: string): Promise<Planning>
  SetMonthlyBudget(input: MonthlyBudgetInput): Promise<MonthlyBudget>
  SaveGoal(id: string, input: GoalInput): Promise<Goal>
  ArchiveGoal(id: string): Promise<void>
  SetGoalAllocations(id: string, input: GoalAllocationInput[]): Promise<Goal>
  DeleteAccount(id: string): Promise<void>
  CreateTransaction(input: TransactionInput): Promise<Transaction>
  UpdateTransaction(id: string, input: TransactionInput): Promise<Transaction>
  TrashTransaction(id: string): Promise<void>
  RestoreTransaction(id: string): Promise<void>
  DeleteTransactionPermanently(id: string): Promise<void>
  EmptyTransactionTrash(): Promise<void>
  ListTransactions(): Promise<Transaction[]>
  ListTrashedTransactions(): Promise<Transaction[]>
  CreditCardsOverview(): Promise<CreditCardsOverview>
  PayCreditCardInvoice(id: string, input: CreditCardPaymentInput): Promise<CreditCardPayment>
  ImportBankStatement(input: BankStatementInput): Promise<BankStatementImportResult>
  SetTheme(theme: Theme): Promise<Profile>
  SetBalancesHidden(hidden: boolean): Promise<Profile>
  FixedExpensesOverview(): Promise<FixedExpensesOverview>
  CreateFixedExpense(input: FixedExpenseInput): Promise<FixedExpense>
  UpdateFixedExpense(id: string, input: FixedExpenseInput): Promise<FixedExpense>
  ArchiveFixedExpense(id: string): Promise<void>
  RestoreFixedExpense(id: string): Promise<void>
  ConfirmFixedExpenseOccurrence(id: string, input: ConfirmFixedExpenseOccurrenceInput): Promise<Transaction>
  DismissFixedExpenseOccurrence(id: string): Promise<void>
  ReopenFixedExpenseOccurrence(id: string): Promise<void>
  GetUpdateStatus(): Promise<UpdateStatus>
  CheckForUpdates(): Promise<UpdateStatus>
  InstallUpdate(): Promise<UpdateStatus>
  SecurityStatus?(): Promise<SecurityStatus>
  UnlockDatabase?(input: UnlockInput): Promise<SecurityStatus>
  EnableEncryption?(input: EncryptionInput): Promise<EncryptionResult>
  ChangeEncryptionPassword?(input: ChangeEncryptionPasswordInput): Promise<void>
  RecoverEncryption?(input: RecoverEncryptionInput): Promise<SecurityStatus>
  DisableEncryption?(input: UnlockInput): Promise<SecurityStatus>
  BackupStatus?(): Promise<BackupStatus>
  CreateBackup?(): Promise<BackupDialogResult>
  ChooseBackupFolder?(): Promise<OperationResult>
  ResetBackupFolder?(): Promise<BackupStatus>
  InspectBackup?(): Promise<BackupDialogResult>
  RestoreBackup?(input: RestoreBackupInput): Promise<BackupDialogResult>
  ExportData?(format: 'csv' | 'json'): Promise<OperationResult>
}

export const api: CashAPI = {
  Bootstrap: async () => await bindings.Bootstrap() as unknown as BootstrapData,
  CompleteOnboarding: async (input) => await bindings.CompleteOnboarding(new application.OnboardingInput(input)) as unknown as Profile,
  SkipOnboarding: async () => await bindings.SkipOnboarding() as unknown as Profile,
  CreateAccount: async (input) => await bindings.CreateAccount(new application.AccountInput(input)) as unknown as Account,
  UpdateAccount: async (id, input) => await bindings.UpdateAccount(id, new application.AccountInput(input)) as unknown as Account,
  AdjustAccountBalance: async (id, input) => await bindings.AdjustAccountBalance(id, new application.BalanceAdjustmentInput(input)) as unknown as Transaction,
  Planning: async (month) => await bindings.Planning(month) as unknown as Planning,
  SetMonthlyBudget: async (input) => await bindings.SetMonthlyBudget(new application.MonthlyBudgetInput(input)) as unknown as MonthlyBudget,
  SaveGoal: async (id, input) => await bindings.SaveGoal(id, new application.GoalInput(input)) as unknown as Goal,
  ArchiveGoal: async (id) => await bindings.ArchiveGoal(id),
  SetGoalAllocations: async (id, input) => await bindings.SetGoalAllocations(id, input.map(item => new application.GoalAllocationInput(item))) as unknown as Goal,
  DeleteAccount: async (id) => await bindings.DeleteAccount(id),
  CreateTransaction: async (input) => await bindings.CreateTransaction(new application.TransactionInput(input)) as unknown as Transaction,
  UpdateTransaction: async (id, input) => await bindings.UpdateTransaction(id, new application.TransactionInput(input)) as unknown as Transaction,
  TrashTransaction: async (id) => await bindings.TrashTransaction(id),
  RestoreTransaction: async (id) => await bindings.RestoreTransaction(id),
  DeleteTransactionPermanently: async (id) => await bindings.DeleteTransactionPermanently(id),
  EmptyTransactionTrash: async () => await bindings.EmptyTransactionTrash(),
  ListTransactions: async () => await bindings.ListTransactions() as unknown as Transaction[],
  ListTrashedTransactions: async () => await bindings.ListTrashedTransactions() as unknown as Transaction[],
  CreditCardsOverview: async () => await bindings.CreditCardsOverview() as unknown as CreditCardsOverview,
  PayCreditCardInvoice: async (id, input) => await bindings.PayCreditCardInvoice(id, new application.CreditCardPaymentInput(input)) as unknown as CreditCardPayment,
  ImportBankStatement: async (input) => await bindings.ImportBankStatement(new application.BankStatementInput(input)) as unknown as BankStatementImportResult,
  SetTheme: async (theme) => await bindings.SetTheme(theme) as unknown as Profile,
  SetBalancesHidden: async (hidden) => await bindings.SetBalancesHidden(hidden) as unknown as Profile,
  FixedExpensesOverview: async () => await bindings.FixedExpensesOverview() as unknown as FixedExpensesOverview,
  CreateFixedExpense: async (input) => await bindings.CreateFixedExpense(new application.FixedExpenseInput(input)) as unknown as FixedExpense,
  UpdateFixedExpense: async (id, input) => await bindings.UpdateFixedExpense(id, new application.FixedExpenseInput(input)) as unknown as FixedExpense,
  ArchiveFixedExpense: async (id) => await bindings.ArchiveFixedExpense(id),
  RestoreFixedExpense: async (id) => await bindings.RestoreFixedExpense(id),
  ConfirmFixedExpenseOccurrence: async (id, input) => await bindings.ConfirmFixedExpenseOccurrence(id, new application.ConfirmFixedExpenseOccurrenceInput(input)) as unknown as Transaction,
  DismissFixedExpenseOccurrence: async (id) => await bindings.DismissFixedExpenseOccurrence(id),
  ReopenFixedExpenseOccurrence: async (id) => await bindings.ReopenFixedExpenseOccurrence(id),
  GetUpdateStatus: async () => await bindings.GetUpdateStatus() as unknown as UpdateStatus,
  CheckForUpdates: async () => await bindings.CheckForUpdates() as unknown as UpdateStatus,
  InstallUpdate: async () => await bindings.InstallUpdate() as unknown as UpdateStatus,
  SecurityStatus: async () => await bindings.SecurityStatus() as unknown as SecurityStatus,
  UnlockDatabase: async (input) => await bindings.UnlockDatabase({ password: input.password, recoveryKey: input.recoveryKey ?? '' }) as unknown as SecurityStatus,
  EnableEncryption: async (input) => await bindings.EnableEncryption(input) as unknown as EncryptionResult,
  ChangeEncryptionPassword: async (input) => await bindings.ChangeEncryptionPassword(input),
  RecoverEncryption: async (input) => await bindings.RecoverEncryption(input) as unknown as SecurityStatus,
  DisableEncryption: async (input) => await bindings.DisableEncryption({ password: input.password, recoveryKey: input.recoveryKey ?? '' }) as unknown as SecurityStatus,
  BackupStatus: async () => await bindings.BackupStatus() as unknown as BackupStatus,
  CreateBackup: async () => await bindings.CreateBackup() as unknown as BackupDialogResult,
  ChooseBackupFolder: async () => await bindings.ChooseBackupFolder() as unknown as OperationResult,
  ResetBackupFolder: async () => await bindings.ResetBackupFolder() as unknown as BackupStatus,
  InspectBackup: async () => await bindings.InspectBackup() as unknown as BackupDialogResult,
  RestoreBackup: async (input) => await bindings.RestoreBackup({ path: input.path, password: input.password ?? '', recoveryKey: input.recoveryKey ?? '' }) as unknown as BackupDialogResult,
  ExportData: async (format) => await bindings.ExportData(format) as unknown as OperationResult,
}
