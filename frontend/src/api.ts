import type { Account, AccountInput, BankStatementImportResult, BankStatementInput, BootstrapData, ConfirmFixedExpenseOccurrenceInput, CreditCardPayment, CreditCardPaymentInput, CreditCardsOverview, FixedExpense, FixedExpenseInput, FixedExpensesOverview, OnboardingInput, Profile, Theme, Transaction, TransactionInput, UpdateStatus } from './types'
import * as bindings from './wailsjs/go/desktop/App'
import { application } from './wailsjs/go/models'

interface CashAPI {
  Bootstrap(): Promise<BootstrapData>
  CompleteOnboarding(input: OnboardingInput): Promise<Profile>
  SkipOnboarding(): Promise<Profile>
  CreateAccount(input: AccountInput): Promise<Account>
  UpdateAccount(id: string, input: AccountInput): Promise<Account>
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
}

export const api: CashAPI = {
  Bootstrap: async () => await bindings.Bootstrap() as unknown as BootstrapData,
  CompleteOnboarding: async (input) => await bindings.CompleteOnboarding(new application.OnboardingInput(input)) as unknown as Profile,
  SkipOnboarding: async () => await bindings.SkipOnboarding() as unknown as Profile,
  CreateAccount: async (input) => await bindings.CreateAccount(new application.AccountInput(input)) as unknown as Account,
  UpdateAccount: async (id, input) => await bindings.UpdateAccount(id, new application.AccountInput(input)) as unknown as Account,
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
}
