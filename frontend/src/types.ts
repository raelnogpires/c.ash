export type Theme = 'light' | 'dark' | 'gothic'
export type AccountType = 'checking' | 'savings' | 'cash'
export type TransactionKind = 'income' | 'expense' | 'transfer'
export type Bank = 'itau' | 'bradesco' | 'inter'
export type UpdateState = 'disabled' | 'idle' | 'checking' | 'upToDate' | 'available' | 'downloading' | 'installing' | 'error'

export interface Profile { displayName: string; currency: 'BRL'; theme: Theme | ''; onboardingStatus: 'completed' | 'skipped'; balancesHidden?: boolean }
export interface Account { id: string; name: string; type: AccountType; openingBalanceCents: number; openingDate: string; createdAt: string; currentBalanceCents: number }
export interface Category { id: string; name: string; kind: TransactionKind }
export interface Transaction { id: string; kind: TransactionKind; amountCents: number; accountId: string; accountName: string; destinationAccountId?: string; destinationAccountName?: string; categoryId?: string; categoryName?: string; description: string; occurrenceDate: string; createdAt: string; updatedAt: string; deletedAt?: string; automaticImport?: boolean; importBank?: Bank }
export interface BalanceHistoryPoint { month: string; label: string; balanceCents: number }
export interface AccountAllocation { accountId: string; accountName: string; balanceCents: number }
export interface Dashboard { availableBalanceCents: number; totalBalanceCents?: number; pendingFixedExpensesCents?: number; pendingFixedExpenseCount?: number; monthlyIncomeCents: number; monthlyExpenseCents: number; recentTransactions: Transaction[]; balanceHistory?: BalanceHistoryPoint[]; accountAllocations?: AccountAllocation[]; hasNegativeBalance: boolean }
export interface BootstrapData { profile: Profile | null; setup: boolean; accounts: Account[]; categories: Category[]; dashboard: Dashboard; theme: Theme | '' }
export interface AccountInput { name: string; type: AccountType; openingBalanceCents: number; openingDate: string }
export interface OnboardingInput { displayName: string; currency: 'BRL'; theme: Theme; firstAccount: AccountInput }
export interface TransactionInput { kind: TransactionKind; amountCents: number; accountId: string; destinationAccountId: string; categoryId: string; description: string; occurrenceDate: string }
export interface FixedExpense { id: string; description: string; amountCents: number; dueDay: number; accountId: string; accountName: string; categoryId: string; categoryName: string; archivedAt?: string; createdAt: string; updatedAt: string }
export type FixedExpenseOccurrenceStatus = 'pending' | 'confirmed' | 'dismissed'
export interface FixedExpenseOccurrence { id: string; fixedExpenseId: string; referenceMonth: string; dueDate: string; description: string; expectedAmountCents: number; accountId: string; accountName: string; categoryId: string; categoryName: string; status: FixedExpenseOccurrenceStatus; transactionId?: string; createdAt: string; updatedAt: string }
export interface FixedExpenseInput { description: string; amountCents: number; dueDay: number; accountId: string; categoryId: string }
export interface ConfirmFixedExpenseOccurrenceInput { amountCents: number; occurrenceDate: string }
export interface FixedExpensesOverview { expenses: FixedExpense[]; occurrences: FixedExpenseOccurrence[] }
export interface BankStatementInput { accountId: string; bank: Bank; fileName: string; base64Pdf: string }
export interface BankStatementImportResult { bank: Bank; importedCount: number; duplicateCount: number; ignoredCount: number }
export interface UpdateStatus { state: UpdateState; currentVersion: string; availableVersion?: string; releaseNotes?: string; publishedAt?: string; lastCheckedAt?: string; downloadedBytes?: number; totalBytes?: number; message?: string }
