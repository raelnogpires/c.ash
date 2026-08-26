export type Theme = 'light' | 'dark' | 'gothic'
export type AccountType = 'checking' | 'savings' | 'cash' | 'credit_card'
export type TransactionKind = 'income' | 'expense' | 'transfer'
export type Bank = 'itau' | 'bradesco' | 'inter'
export type UpdateState = 'disabled' | 'idle' | 'checking' | 'upToDate' | 'available' | 'downloading' | 'installing' | 'error'

export interface Profile { displayName: string; currency: 'BRL'; theme: Theme | ''; onboardingStatus: 'completed' | 'skipped'; balancesHidden?: boolean }
export interface Account { id: string; name: string; type: AccountType; openingBalanceCents: number; openingDate: string; createdAt: string; currentBalanceCents: number; creditLimitCents?: number; closingDay?: number; dueDay?: number }
export interface Category { id: string; name: string; kind: TransactionKind }
export interface Transaction { id: string; kind: TransactionKind; amountCents: number; accountId: string; accountName: string; destinationAccountId?: string; destinationAccountName?: string; categoryId?: string; categoryName?: string; description: string; occurrenceDate: string; createdAt: string; updatedAt: string; deletedAt?: string; automaticImport?: boolean; importBank?: Bank; installmentCount?: number; invoicePaymentId?: string }
export interface BalanceHistoryPoint { month: string; label: string; balanceCents: number }
export interface AccountAllocation { accountId: string; accountName: string; balanceCents: number }
export interface Dashboard { availableBalanceCents: number; totalBalanceCents?: number; pendingFixedExpensesCents?: number; pendingFixedExpenseCount?: number; monthlyIncomeCents: number; monthlyExpenseCents: number; recentTransactions: Transaction[]; balanceHistory?: BalanceHistoryPoint[]; accountAllocations?: AccountAllocation[]; hasNegativeBalance: boolean; creditCardDebtCents?: number; upcomingInvoices?: CreditCardInvoice[] }
export interface BootstrapData { profile: Profile | null; setup: boolean; accounts: Account[]; categories: Category[]; dashboard: Dashboard; theme: Theme | '' }
export interface AccountInput { name: string; type: AccountType; openingBalanceCents: number; openingDate: string; creditLimitCents?: number; closingDay?: number; dueDay?: number; openingDebtCents?: number; openingDebtDueDate?: string }
export interface OnboardingInput { displayName: string; currency: 'BRL'; theme: Theme; firstAccount: AccountInput }
export interface TransactionInput { kind: TransactionKind; amountCents: number; accountId: string; destinationAccountId: string; categoryId: string; description: string; occurrenceDate: string; installmentCount?: number }
export type CreditCardInvoiceStatus = 'open' | 'closed' | 'paid' | 'rolled_over'
export interface CreditCardInstallment { id: string; invoiceId: string; transactionId?: string; description: string; amountCents: number; installmentNumber: number; installmentCount: number; openingDebt?: boolean }
export interface CreditCardPayment { id: string; invoiceId: string; accountId: string; accountName: string; transactionId: string; amountCents: number; occurrenceDate: string; createdAt: string }
export interface CreditCardInvoice { id: string; accountId: string; accountName: string; referenceMonth: string; closingDate: string; dueDate: string; status: CreditCardInvoiceStatus; chargesCents: number; carryForwardCents: number; paidCents: number; outstandingCents: number; installments: CreditCardInstallment[]; payments: CreditCardPayment[] }
export interface CreditCardSummary { account: Account; outstandingCents: number; availableLimitCents: number; currentInvoice?: CreditCardInvoice }
export interface CreditCardsOverview { cards: CreditCardSummary[]; invoices: CreditCardInvoice[] }
export interface CreditCardPaymentInput { accountId: string; amountCents: number; occurrenceDate: string }
export interface FixedExpense { id: string; description: string; amountCents: number; dueDay: number; accountId: string; accountName: string; categoryId: string; categoryName: string; archivedAt?: string; createdAt: string; updatedAt: string }
export type FixedExpenseOccurrenceStatus = 'pending' | 'confirmed' | 'dismissed'
export interface FixedExpenseOccurrence { id: string; fixedExpenseId: string; referenceMonth: string; dueDate: string; description: string; expectedAmountCents: number; accountId: string; accountName: string; categoryId: string; categoryName: string; status: FixedExpenseOccurrenceStatus; transactionId?: string; createdAt: string; updatedAt: string }
export interface FixedExpenseInput { description: string; amountCents: number; dueDay: number; accountId: string; categoryId: string }
export interface ConfirmFixedExpenseOccurrenceInput { amountCents: number; occurrenceDate: string }
export interface FixedExpensesOverview { expenses: FixedExpense[]; occurrences: FixedExpenseOccurrence[] }
export interface BankStatementInput { accountId: string; bank: Bank; fileName: string; base64Data: string }
export interface BankStatementImportResult { bank: Bank; importedCount: number; duplicateCount: number; ignoredCount: number }
export interface UpdateStatus { state: UpdateState; currentVersion: string; availableVersion?: string; releaseNotes?: string; publishedAt?: string; lastCheckedAt?: string; downloadedBytes?: number; totalBytes?: number; message?: string }
export interface SecurityStatus { enabled: boolean; locked: boolean }
export interface EncryptionResult { status: SecurityStatus; recoveryKey?: string }
export interface EncryptionInput { password: string; confirmation: string }
export interface ChangeEncryptionPasswordInput { currentPassword: string; newPassword: string; confirmation: string }
export interface RecoverEncryptionInput { recoveryKey: string; newPassword: string; confirmation: string }
export interface UnlockInput { password: string; recoveryKey?: string }
export interface BackupManifest { formatVersion: number; createdAt: string; applicationVersion: string; schemaVersion: number; encrypted: boolean; payloadSha256: string; kind: string }
export interface BackupInfo { path: string; manifest: BackupManifest }
export interface BackupStatus { folder: string; defaultFolder: string; lastAutomaticAt?: string; lastAutomaticPath?: string; lastError?: string; nextDueAt?: string; automaticDue: boolean }
export interface OperationResult { cancelled: boolean; success: boolean; path?: string }
export interface BackupDialogResult { cancelled: boolean; success: boolean; backup: BackupInfo }
export interface RestoreBackupInput { path: string; password?: string; recoveryKey?: string }
