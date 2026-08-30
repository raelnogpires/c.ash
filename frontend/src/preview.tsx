import { useMemo, useState, type ComponentProps } from 'react'
import { createRoot } from 'react-dom/client'
import '@fontsource-variable/inter'
import App from './App'
import './styles.css'
import { createPreviewCreditCards, createPreviewData, createPreviewFixedExpenses, createPreviewOccurrences, type PreviewScenario } from './preview-data'
import type {
  Account,
  AccountInput,
  BackupDialogResult,
  BackupStatus,
  BalanceAdjustmentInput,
  BootstrapData,
  Category,
  CategoryInput,
  CreditCardPayment,
  CreditCardPaymentInput,
  EncryptionInput,
  EncryptionResult,
  FixedExpense,
  FixedExpenseInput,
  FixedExpenseOccurrence,
  Goal,
  GoalAllocationInput,
  GoalInput,
  MonthlyBudget,
  MonthlyBudgetInput,
  OnboardingInput,
  OperationResult,
  Planning,
  Profile,
  RestoreBackupInput,
  SecurityStatus,
  Theme,
  Transaction,
  TransactionFilter,
  TransactionInput,
  TransactionOccurrence,
  UnlockInput,
  UpdateStatus,
} from './types'

type PreviewAPI = NonNullable<ComponentProps<typeof App>['api']>

const clone = <T,>(value: T): T => JSON.parse(JSON.stringify(value)) as T
const previewDate = '2026-08-30'
const previewTimestamp = '2026-08-30T12:00:00Z'

const fallbackProfile = (theme: Theme): Profile => ({ displayName: 'Lia', currency: 'BRL', theme, onboardingStatus: 'skipped', balancesHidden: false })

function accountTypeLabel(account: Account) {
  return account.type === 'checking' ? 'Conta corrente' : account.type === 'savings' ? 'Poupança' : account.type === 'credit_card' ? 'Cartão de crédito' : 'Dinheiro'
}

function createAccount(input: AccountInput, index: number): Account {
  const openingDebt = input.openingDebtCents ?? 0
  return {
    id: `preview-account-${index}`,
    name: input.name || `Conta ${index}`,
    type: input.type,
    openingBalanceCents: input.type === 'credit_card' ? 0 : input.openingBalanceCents,
    openingDate: input.openingDate || previewDate,
    createdAt: previewTimestamp,
    currentBalanceCents: input.type === 'credit_card' ? -openingDebt : input.openingBalanceCents,
    creditLimitCents: input.creditLimitCents,
    closingDay: input.closingDay,
    dueDay: input.dueDay,
  }
}

export function createPreviewAPI(scenario: PreviewScenario, theme: Theme): PreviewAPI {
  let current = clone(createPreviewData(scenario, theme))
  let transactions = clone(current.dashboard.recentTransactions)
  let occurrences = clone(createPreviewOccurrences(scenario))
  let fixedOverview = clone(createPreviewFixedExpenses(scenario))
  let nextAccount = current.accounts.length + 1
  let nextCategory = current.categories.length + 1
  let nextGoal = (current.planning?.goals.length ?? 0) + 1

  const syncTransactions = () => {
    const active = transactions.filter(item => !item.deletedAt)
    current = { ...current, dashboard: { ...current.dashboard, recentTransactions: active.slice(0, 5) } }
  }
  const profile = () => clone(current.profile ?? fallbackProfile(theme))
  const accountById = (id: string) => current.accounts.find(item => item.id === id)
  const categoryById = (id: string) => current.categories.find(item => item.id === id)
  const transactionFromInput = (input: TransactionInput, id: string): Transaction => {
    const account = accountById(input.accountId)
    const destination = accountById(input.destinationAccountId)
    const category = categoryById(input.categoryId)
    return {
      id,
      kind: input.kind,
      amountCents: input.amountCents,
      accountId: input.accountId,
      accountName: account?.name ?? 'Conta',
      destinationAccountId: input.destinationAccountId || undefined,
      destinationAccountName: destination?.name,
      categoryId: input.categoryId || undefined,
      categoryName: category?.name,
      description: input.description || (input.kind === 'transfer' ? 'Transferência' : 'Novo registro'),
      occurrenceDate: input.occurrenceDate || previewDate,
      createdAt: previewTimestamp,
      updatedAt: previewTimestamp,
      installmentCount: input.installmentCount,
      subcategoryName: input.subcategoryName,
      tags: (input.tags ?? []).map((name, index) => ({ id: `preview-tag-${index}`, name })),
      origin: 'manual',
      recurrenceRuleId: input.monthlyRecurrence ? `preview-rule-${id}` : undefined,
    }
  }
  const backup = (): BackupDialogResult => ({
    cancelled: false,
    success: true,
    backup: {
      path: '/preview/c.ash/backup-2026-08-30.cashbackup',
      manifest: { formatVersion: 1, createdAt: previewTimestamp, applicationVersion: 'preview', schemaVersion: 12, encrypted: false, payloadSha256: 'preview', kind: 'manual' },
    },
  })
  const backupStatus = (): BackupStatus => ({ folder: '/preview/c.ash/backups', defaultFolder: '/preview/c.ash/backups', lastAutomaticAt: '2026-08-23T12:00:00Z', nextDueAt: '2026-08-30T12:00:00Z', automaticDue: false })
  const updateStatus = (): UpdateStatus => scenario === 'rich'
    ? { state: 'available', currentVersion: '0.2.4', availableVersion: '0.2.5', publishedAt: '2026-08-29T12:00:00Z', releaseNotes: 'Ajustes de navegação e refinamentos visuais.' }
    : { state: 'upToDate', currentVersion: '0.2.4' }

  const api: PreviewAPI = {
    Bootstrap: async () => clone(current),
    CompleteOnboarding: async (input: OnboardingInput) => {
      const first = createAccount(input.firstAccount, nextAccount++)
      current = { ...current, setup: true, profile: { displayName: input.displayName, currency: input.currency, theme: input.theme, onboardingStatus: 'completed', balancesHidden: false }, theme: input.theme, accounts: [first] }
      return profile()
    },
    SkipOnboarding: async () => {
      current = { ...current, setup: true, profile: fallbackProfile(theme) }
      return profile()
    },
    CreateAccount: async (input: AccountInput) => {
      const value = createAccount(input, nextAccount++)
      current = { ...current, accounts: [...current.accounts, value] }
      return clone(value)
    },
    UpdateAccount: async (id: string, input: AccountInput) => {
      const existing = accountById(id)
      const value = { ...(existing ?? createAccount(input, nextAccount++)), ...createAccount(input, Number(id.replace(/\D/g, '')) || nextAccount) , id }
      current = { ...current, accounts: current.accounts.map(item => item.id === id ? value : item) }
      return clone(value)
    },
    AdjustAccountBalance: async (id: string, input: BalanceAdjustmentInput) => {
      const existing = accountById(id)
      if (existing) current = { ...current, accounts: current.accounts.map(item => item.id === id ? { ...item, currentBalanceCents: input.targetBalanceCents, hasLedgerActivity: true } : item) }
      return transactionFromInput({ kind: 'expense', amountCents: Math.abs(input.targetBalanceCents - (existing?.currentBalanceCents ?? 0)), accountId: id, destinationAccountId: '', categoryId: '', description: input.reason, occurrenceDate: input.occurrenceDate }, `preview-adjustment-${id}`)
    },
    Planning: async () => clone(current.planning ?? ({ goals: [] } as Planning)),
    SetMonthlyBudget: async (input: MonthlyBudgetInput) => {
      const value: MonthlyBudget = {
        referenceMonth: input.referenceMonth,
        overallLimitCents: input.overallLimitCents,
        spentCents: current.planning?.budget?.spentCents ?? 0,
        remainingCents: input.overallLimitCents - (current.planning?.budget?.spentCents ?? 0),
        progressPercent: input.overallLimitCents ? ((current.planning?.budget?.spentCents ?? 0) / input.overallLimitCents) * 100 : 0,
        categoryLimits: input.categoryLimits.map((item, index) => ({ id: `preview-limit-${index}`, categoryId: item.categoryId, categoryName: categoryById(item.categoryId)?.name ?? item.categoryId, limitCents: item.limitCents, rollover: item.rollover, rolloverCents: 0, spentCents: 0, availableCents: item.limitCents, exceeded: false })),
      }
      current = { ...current, planning: { ...(current.planning ?? { goals: [] }), budget: value } }
      return clone(value)
    },
    CreateCategory: async (input: CategoryInput) => {
      const value: Category = { id: `preview-category-${nextCategory++}`, name: input.name, kind: input.kind, editable: true }
      current = { ...current, categories: [...current.categories, value] }
      return clone(value)
    },
    RenameCategory: async (id: string, input: CategoryInput) => {
      let value = current.categories.find(item => item.id === id)
      if (value) { value = { ...value, name: input.name }; current = { ...current, categories: current.categories.map(item => item.id === id ? value! : item) } }
      return clone(value ?? { id, name: input.name, kind: input.kind, editable: true })
    },
    ArchiveCategory: async (id: string) => { current = { ...current, categories: current.categories.map(item => item.id === id ? { ...item, archivedAt: previewTimestamp } : item) } },
    RestoreCategory: async (id: string) => { current = { ...current, categories: current.categories.map(item => item.id === id ? { ...item, archivedAt: undefined } : item) } },
    SaveGoal: async (id: string, input: GoalInput) => {
      const value: Goal = { id: id || `preview-goal-${nextGoal++}`, name: input.name, kind: input.kind, targetCents: input.targetCents, deadline: input.deadline, createdAt: previewTimestamp, updatedAt: previewTimestamp, allocatedCents: current.planning?.goals.find(item => item.id === id)?.allocatedCents ?? 0, progressPercent: 0, allocations: [] }
      value.progressPercent = value.targetCents ? (value.allocatedCents / value.targetCents) * 100 : 0
      const goals = current.planning?.goals ?? []
      current = { ...current, planning: { ...(current.planning ?? { goals: [] }), goals: goals.some(item => item.id === value.id) ? goals.map(item => item.id === value.id ? value : item) : [...goals, value] } }
      return clone(value)
    },
    ArchiveGoal: async (id: string) => { current = { ...current, planning: { ...(current.planning ?? { goals: [] }), goals: (current.planning?.goals ?? []).map(item => item.id === id ? { ...item, archivedAt: previewTimestamp } : item) } } },
    SetGoalAllocations: async (id: string, input: GoalAllocationInput[]) => {
      const allocations = input.map(item => ({ goalId: id, accountId: item.accountId, accountName: accountById(item.accountId)?.name ?? item.accountId, amountCents: item.amountCents }))
      current = { ...current, planning: { ...(current.planning ?? { goals: [] }), goals: (current.planning?.goals ?? []).map(item => item.id === id ? { ...item, allocations, allocatedCents: allocations.reduce((sum, item) => sum + item.amountCents, 0), progressPercent: item.targetCents ? allocations.reduce((sum, item) => sum + item.amountCents, 0) / item.targetCents * 100 : 0 } : item) } }
      return clone(current.planning!.goals.find(item => item.id === id)!)
    },
    DeleteAccount: async (id: string) => { current = { ...current, accounts: current.accounts.filter(item => item.id !== id) } },
    CreateTransaction: async (input: TransactionInput) => { const value = transactionFromInput(input, `preview-transaction-${transactions.length + 1}`); transactions = [value, ...transactions]; syncTransactions(); return clone(value) },
    UpdateTransaction: async (id: string, input: TransactionInput) => { const value = transactionFromInput(input, id); transactions = transactions.map(item => item.id === id ? value : item); syncTransactions(); return clone(value) },
    TransactionOccurrences: async () => clone(occurrences),
    ConfirmTransactionOccurrence: async (id: string) => { const item = occurrences.find(value => value.id === id); if (item) item.status = 'confirmed'; return clone(transactions[0] ?? transactionFromInput({ kind: 'expense', amountCents: 0, accountId: 'checking', destinationAccountId: '', categoryId: '', description: 'Ocorrência confirmada', occurrenceDate: previewDate }, `preview-occurrence-${id}`)) },
    DismissTransactionOccurrence: async (id: string) => { occurrences = occurrences.map(item => item.id === id ? { ...item, status: 'dismissed' } : item) },
    ArchiveRecurrence: async () => undefined,
    TrashTransaction: async (id: string) => { transactions = transactions.map(item => item.id === id ? { ...item, deletedAt: previewTimestamp } : item); syncTransactions() },
    RestoreTransaction: async (id: string) => { transactions = transactions.map(item => item.id === id ? { ...item, deletedAt: undefined } : item); syncTransactions() },
    DeleteTransactionPermanently: async (id: string) => { transactions = transactions.filter(item => item.id !== id); syncTransactions() },
    EmptyTransactionTrash: async () => { transactions = transactions.filter(item => !item.deletedAt); syncTransactions() },
    ListTransactions: async () => clone(transactions.filter(item => !item.deletedAt)),
    SearchTransactions: async (filter: TransactionFilter) => {
      const status = filter.status ?? 'active'
      const active = transactions.filter(item => !item.deletedAt).map(item => ({ ...item, status: 'active' as const }))
      const trashed = transactions.filter(item => item.deletedAt).map(item => ({ ...item, status: 'trashed' as const }))
      const pending: Transaction[] = occurrences.filter(item => item.status === 'pending').map(item => ({
        id: item.id,
        recurrenceRuleId: item.recurrenceRuleId,
        kind: item.kind,
        amountCents: item.amountCents,
        accountId: item.accountId,
        accountName: item.accountName,
        categoryId: item.categoryId,
        categoryName: item.categoryName,
        description: item.description,
        occurrenceDate: item.scheduledDate,
        createdAt: previewTimestamp,
        updatedAt: previewTimestamp,
        installmentCount: item.installmentCount,
        tags: item.tags,
        splits: item.splits,
        status: 'pending' as const,
        pending: true,
      }))
      const source = status === 'trashed' ? trashed : status === 'pending' ? pending : status === 'all' ? [...active, ...trashed, ...pending] : active
      const query = (filter.text ?? '').trim().toLocaleLowerCase('pt-BR')
      const tagQuery = (filter.tag ?? '').trim().toLocaleLowerCase('pt-BR')
      return clone(source.filter(item => {
        if (filter.startDate && item.occurrenceDate < filter.startDate) return false
        if (filter.endDate && item.occurrenceDate > filter.endDate) return false
        if (filter.accountId && item.accountId !== filter.accountId && item.destinationAccountId !== filter.accountId) return false
        if (filter.categoryId && item.categoryId !== filter.categoryId && !(item.splits ?? []).some(split => split.categoryId === filter.categoryId)) return false
        if (filter.subcategoryId && item.subcategoryId !== filter.subcategoryId) return false
        if (filter.kind && item.kind !== filter.kind) return false
        if ((filter.minimumAmountCents ?? 0) > 0 && item.amountCents < filter.minimumAmountCents!) return false
        if ((filter.maximumAmountCents ?? 0) > 0 && item.amountCents > filter.maximumAmountCents!) return false
        if (filter.recurrence === 'recurring' && !item.recurrenceRuleId) return false
        if (filter.recurrence === 'nonrecurring' && item.recurrenceRuleId) return false
        if (query && !`${item.description} ${item.accountName} ${item.categoryName ?? ''} ${item.subcategoryName ?? ''}`.toLocaleLowerCase('pt-BR').includes(query)) return false
        if (tagQuery && !(item.tags ?? []).some(tag => tag.name.toLocaleLowerCase('pt-BR') === tagQuery)) return false
        return true
      }))
    },
    ListTrashedTransactions: async () => clone(transactions.filter(item => item.deletedAt)),
    CreditCardsOverview: async () => clone(createPreviewCreditCards(scenario, current.accounts)),
    PayCreditCardInvoice: async (id: string, input: CreditCardPaymentInput) => {
      const value: CreditCardPayment = { id: `preview-payment-${id}`, invoiceId: id, accountId: input.accountId, accountName: accountById(input.accountId)?.name ?? 'Conta', transactionId: `preview-card-payment-${id}`, amountCents: input.amountCents, occurrenceDate: input.occurrenceDate, createdAt: previewTimestamp }
      return value
    },
    ImportBankStatement: async () => ({ bank: 'itau', importedCount: 3, duplicateCount: 1, ignoredCount: 0 }),
    SetTheme: async (value: Theme) => { current = { ...current, theme: value, profile: { ...(current.profile ?? fallbackProfile(value)), theme: value } }; return profile() },
    SetBalancesHidden: async (hidden: boolean) => { current = { ...current, profile: { ...(current.profile ?? fallbackProfile(theme)), balancesHidden: hidden } }; return profile() },
    FixedExpensesOverview: async () => clone(fixedOverview),
    CreateFixedExpense: async (input: FixedExpenseInput) => { const value: FixedExpense = { id: `preview-fixed-${fixedOverview.expenses.length + 1}`, ...input, accountName: accountById(input.accountId)?.name ?? 'Conta', categoryName: categoryById(input.categoryId)?.name ?? 'Categoria', createdAt: previewTimestamp, updatedAt: previewTimestamp }; fixedOverview = { ...fixedOverview, expenses: [...fixedOverview.expenses, value] }; return clone(value) },
    UpdateFixedExpense: async (id: string, input: FixedExpenseInput) => { const existing = fixedOverview.expenses.find(item => item.id === id); const value: FixedExpense = { ...(existing ?? { id, createdAt: previewTimestamp, updatedAt: previewTimestamp }), ...input, accountName: accountById(input.accountId)?.name ?? 'Conta', categoryName: categoryById(input.categoryId)?.name ?? 'Categoria' }; fixedOverview = { ...fixedOverview, expenses: fixedOverview.expenses.map(item => item.id === id ? value : item) }; return clone(value) },
    ArchiveFixedExpense: async (id: string) => { fixedOverview = { ...fixedOverview, expenses: fixedOverview.expenses.map(item => item.id === id ? { ...item, archivedAt: previewTimestamp } : item) } },
    RestoreFixedExpense: async (id: string) => { fixedOverview = { ...fixedOverview, expenses: fixedOverview.expenses.map(item => item.id === id ? { ...item, archivedAt: undefined } : item) } },
    ConfirmFixedExpenseOccurrence: async (id: string) => { fixedOverview = { ...fixedOverview, occurrences: fixedOverview.occurrences.map(item => item.id === id ? { ...item, status: 'confirmed' as const, transactionId: `preview-fixed-transaction-${id}` } : item) }; return clone(transactions[0] ?? transactionFromInput({ kind: 'expense', amountCents: 0, accountId: 'checking', destinationAccountId: '', categoryId: '', description: 'Despesa fixa', occurrenceDate: previewDate }, `preview-fixed-transaction-${id}`)) },
    DismissFixedExpenseOccurrence: async (id: string) => { fixedOverview = { ...fixedOverview, occurrences: fixedOverview.occurrences.map(item => item.id === id ? { ...item, status: 'dismissed' as const } : item) } },
    ReopenFixedExpenseOccurrence: async (id: string) => { fixedOverview = { ...fixedOverview, occurrences: fixedOverview.occurrences.map(item => item.id === id ? { ...item, status: 'pending' as const } : item) } },
    GetUpdateStatus: async () => updateStatus(),
    CheckForUpdates: async () => ({ ...updateStatus(), state: 'upToDate' as const, lastCheckedAt: previewTimestamp }),
    InstallUpdate: async () => ({ ...updateStatus(), state: 'installing' as const }),
    SecurityStatus: async () => ({ enabled: false, locked: false }),
    UnlockDatabase: async (_input: UnlockInput) => ({ enabled: false, locked: false }),
    EnableEncryption: async (_input: EncryptionInput): Promise<EncryptionResult> => ({ status: { enabled: true, locked: false }, recoveryKey: 'PREVIEW-KEY-ONLY-FOR-REVIEW' }),
    ChangeEncryptionPassword: async () => undefined,
    RecoverEncryption: async () => ({ enabled: true, locked: false }),
    DisableEncryption: async () => ({ enabled: false, locked: false }),
    BackupStatus: async () => backupStatus(),
    CreateBackup: async () => backup(),
    ChooseBackupFolder: async (): Promise<OperationResult> => ({ cancelled: false, success: true, path: '/preview/c.ash/backups' }),
    ResetBackupFolder: async () => backupStatus(),
    InspectBackup: async () => backup(),
    RestoreBackup: async (_input: RestoreBackupInput) => ({ ...backup(), success: true }),
    ExportData: async () => ({ cancelled: false, success: true, path: '/preview/c.ash/export.csv' }),
  }
  return api
}

const scenarioOptions: { value: PreviewScenario; label: string; hint: string }[] = [
  { value: 'onboarding', label: 'Onboarding', hint: 'configuração inicial sem dados' },
  { value: 'rich', label: 'Completo', hint: 'dados, cartões, planejamento e atualização' },
  { value: 'empty', label: 'Vazio', hint: 'primeiro passo sem dados fictícios' },
  { value: 'negative', label: 'Atenção', hint: 'saldo negativo e orçamento excedido' },
]

const themes: { value: Theme; label: string }[] = [
  { value: 'light', label: 'Claro' },
  { value: 'dark', label: 'Escuro' },
  { value: 'gothic', label: 'Gótico' },
]

function readParam<T extends string>(key: string, values: readonly T[], fallback: T): T {
  const value = new URLSearchParams(window.location.search).get(key) as T | null
  return value && values.includes(value) ? value : fallback
}

function Preview() {
  const [scenario, setScenario] = useState<PreviewScenario>(() => readParam('scenario', ['onboarding', 'rich', 'empty', 'negative'], 'rich'))
  const [theme, setTheme] = useState<Theme>(() => readParam('theme', ['light', 'dark', 'gothic'], 'light'))
  const controlsVisible = new URLSearchParams(window.location.search).get('controls') !== '0'
  const api = useMemo(() => createPreviewAPI(scenario, theme), [scenario, theme])
  const updateQuery = (nextScenario: PreviewScenario, nextTheme: Theme) => {
    const url = new URL(window.location.href)
    url.searchParams.set('scenario', nextScenario)
    url.searchParams.set('theme', nextTheme)
    window.history.replaceState(null, '', url)
  }
  const changeScenario = (value: PreviewScenario) => { setScenario(value); updateQuery(value, theme) }
  const changeTheme = (value: Theme) => { setTheme(value); updateQuery(scenario, value) }
  return <>
    {controlsVisible && <div className="preview-controls" role="region" aria-label="Controles do preview" style={{ position: 'fixed', top: 12, right: 12, zIndex: 100, width: 'min(26rem, calc(100vw - 24px))', padding: 10, display: 'grid', gap: 8, border: '1px solid color-mix(in srgb, currentColor 18%, transparent)', borderRadius: 12, background: 'color-mix(in srgb, Canvas 88%, transparent)', color: 'CanvasText', boxShadow: '0 12px 32px color-mix(in srgb, CanvasText 14%, transparent)', backdropFilter: 'blur(16px)' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', gap: 12 }}>
        <strong style={{ fontSize: 12, letterSpacing: '.04em' }}>[c]ash · preview</strong>
        <span style={{ fontSize: 11, opacity: .64 }}>local / mock</span>
      </div>
      <label style={{ display: 'grid', gap: 4, fontSize: 11, fontWeight: 700 }}>
        Cenário
        <select value={scenario} onChange={event => changeScenario(event.target.value as PreviewScenario)} style={{ minHeight: 34, padding: '0 8px', borderRadius: 8, border: '1px solid color-mix(in srgb, currentColor 18%, transparent)', background: 'Canvas', color: 'CanvasText' }}>
          {scenarioOptions.map(option => <option value={option.value} key={option.value}>{option.label} — {option.hint}</option>)}
        </select>
      </label>
      <div role="group" aria-label="Tema do preview" style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
        {themes.map(option => <button key={option.value} type="button" aria-pressed={theme === option.value} onClick={() => changeTheme(option.value)} style={{ minHeight: 32, padding: '0 9px', borderRadius: 999, border: '1px solid color-mix(in srgb, currentColor 18%, transparent)', background: theme === option.value ? 'CanvasText' : 'transparent', color: theme === option.value ? 'Canvas' : 'CanvasText', fontSize: 11, fontWeight: 700, cursor: 'pointer' }}>{option.label}</button>)}
      </div>
    </div>}
    <App key={`${scenario}-${theme}`} api={api}/>
  </>
}

const previewRoot = createRoot(document.getElementById('preview-root')!)
previewRoot.render(<Preview />)
const previewHot = (import.meta as ImportMeta & { hot?: { dispose(callback: () => void): void } }).hot
if (previewHot) previewHot.dispose(() => previewRoot.unmount())
