import type {
  Account,
  BootstrapData,
  Category,
  CreditCardsOverview,
  FixedExpensesOverview,
  Goal,
  MonthlyBudget,
  Theme,
  Transaction,
  TransactionOccurrence,
} from './types'

export type PreviewScenario = 'onboarding' | 'rich' | 'empty' | 'negative'

const date = '2026-08-30'
const createdAt = '2026-08-01T12:00:00Z'

const categories: Category[] = [
  { id: 'food', name: 'Alimentação', kind: 'expense', editable: true },
  { id: 'home', name: 'Casa', kind: 'expense', editable: true },
  { id: 'leisure', name: 'Lazer', kind: 'expense', editable: true },
  { id: 'salary', name: 'Salário', kind: 'income', editable: true },
]

const account = (input: Pick<Account, 'id' | 'name' | 'type' | 'currentBalanceCents'> & Partial<Account>): Account => ({
  openingBalanceCents: input.currentBalanceCents,
  openingDate: '2026-08-01',
  createdAt,
  ...input,
})

const transaction = (input: Pick<Transaction, 'id' | 'kind' | 'amountCents' | 'accountId' | 'accountName' | 'description' | 'occurrenceDate'> & Partial<Transaction>): Transaction => ({
  createdAt,
  updatedAt: createdAt,
  categoryId: input.kind === 'income' ? 'salary' : input.kind === 'transfer' ? undefined : 'food',
  categoryName: input.kind === 'income' ? 'Salário' : input.kind === 'transfer' ? undefined : 'Alimentação',
  ...input,
})

const richAccounts = (): Account[] => [
  account({ id: 'checking', name: 'Conta principal', type: 'checking', currentBalanceCents: 746300 }),
  account({ id: 'savings', name: 'Reserva', type: 'savings', currentBalanceCents: 185000 }),
  account({ id: 'cash', name: 'Carteira', type: 'cash', currentBalanceCents: 12000 }),
  account({
    id: 'card', name: 'Nubank', type: 'credit_card', currentBalanceCents: -350000,
    creditLimitCents: 800000, closingDay: 25, dueDay: 2,
  }),
]

const richTransactions = (): Transaction[] => [
  transaction({ id: 'tx-salary', kind: 'income', amountCents: 520000, accountId: 'checking', accountName: 'Conta principal', categoryId: 'salary', categoryName: 'Salário', description: 'Salário de agosto', occurrenceDate: date, origin: 'manual' }),
  transaction({ id: 'tx-rent', kind: 'expense', amountCents: 185000, accountId: 'checking', accountName: 'Conta principal', categoryId: 'home', categoryName: 'Casa', description: 'Aluguel', occurrenceDate: '2026-08-10', origin: 'fixed_expense' }),
  transaction({ id: 'tx-market', kind: 'expense', amountCents: 6420, accountId: 'checking', accountName: 'Conta principal', categoryId: 'food', categoryName: 'Alimentação', description: 'Mercado', occurrenceDate: '2026-08-21', automaticImport: true, importBank: 'itau', origin: 'import' }),
  transaction({ id: 'tx-transfer', kind: 'transfer', amountCents: 100000, accountId: 'checking', accountName: 'Conta principal', destinationAccountId: 'savings', destinationAccountName: 'Reserva', categoryId: undefined, categoryName: undefined, description: 'Reserva mensal', occurrenceDate: '2026-08-15', origin: 'manual' }),
  transaction({ id: 'tx-notebook', kind: 'expense', amountCents: 129900, accountId: 'card', accountName: 'Nubank', categoryId: 'leisure', categoryName: 'Lazer', description: 'Notebook', occurrenceDate: '2026-08-18', installmentCount: 6, tags: [{ id: 'tag-1', name: 'trabalho' }], origin: 'manual' }),
]

const richBudget: MonthlyBudget = {
  referenceMonth: '2026-08',
  overallLimitCents: 800000,
  spentCents: 510000,
  remainingCents: 290000,
  progressPercent: 63.75,
  categoryLimits: [
    { id: 'limit-food', categoryId: 'food', categoryName: 'Alimentação', limitCents: 120000, rollover: true, rolloverCents: 10000, spentCents: 86500, availableCents: 43500, exceeded: false },
    { id: 'limit-home', categoryId: 'home', categoryName: 'Casa', limitCents: 240000, rollover: false, rolloverCents: 0, spentCents: 225000, availableCents: 15000, exceeded: false },
  ],
}

const goal = (input: Pick<Goal, 'id' | 'name' | 'kind' | 'targetCents' | 'allocatedCents' | 'progressPercent'>): Goal => ({
  ...input,
  allocations: input.id === 'emergency' ? [{ goalId: input.id, accountId: 'savings', accountName: 'Reserva', amountCents: input.allocatedCents }] : [],
  createdAt,
  updatedAt: createdAt,
})

const richGoals = (): Goal[] => [
  goal({ id: 'emergency', name: 'Reserva de emergência', kind: 'emergency_reserve', targetCents: 1500000, allocatedCents: 700000, progressPercent: 46.67 }),
  goal({ id: 'travel', name: 'Viagem de fim de ano', kind: 'savings', targetCents: 500000, allocatedCents: 150000, progressPercent: 30 }),
]

export function createPreviewData(scenario: PreviewScenario, theme: Theme): BootstrapData {
  if (scenario === 'onboarding') {
    return {
      profile: { displayName: '', currency: 'BRL', theme, onboardingStatus: 'skipped', balancesHidden: false },
      setup: false,
      theme,
      accounts: [],
      categories: [],
      dashboard: {
        availableBalanceCents: 0,
        totalBalanceCents: 0,
        pendingFixedExpensesCents: 0,
        pendingFixedExpenseCount: 0,
        monthlyIncomeCents: 0,
        monthlyExpenseCents: 0,
        recentTransactions: [],
        balanceHistory: [],
        accountAllocations: [],
        upcomingInvoices: [],
        hasNegativeBalance: false,
        reservedValueCents: 0,
        freeValueCents: 0,
        safelySpendableCents: 0,
        budgetProgressPercent: 0,
        goalProgressPercent: 0,
      },
      planning: { goals: [] },
    }
  }

  if (scenario === 'empty') {
    return {
      profile: { displayName: 'Lia', currency: 'BRL', theme, onboardingStatus: 'skipped', balancesHidden: false },
      setup: true,
      theme,
      accounts: [],
      categories: [{ id: 'food', name: 'Alimentação', kind: 'expense', editable: true }],
      dashboard: {
        availableBalanceCents: 0,
        totalBalanceCents: 0,
        pendingFixedExpensesCents: 0,
        pendingFixedExpenseCount: 0,
        monthlyIncomeCents: 0,
        monthlyExpenseCents: 0,
        recentTransactions: [],
        balanceHistory: [],
        accountAllocations: [],
        upcomingInvoices: [],
        hasNegativeBalance: false,
        reservedValueCents: 0,
        freeValueCents: 0,
        safelySpendableCents: 0,
        budgetProgressPercent: 0,
        goalProgressPercent: 0,
      },
      planning: { goals: [] },
    }
  }

  if (scenario === 'negative') {
    const accounts = [
      account({ id: 'checking', name: 'Conta principal', type: 'checking', currentBalanceCents: -18340 }),
      account({ id: 'savings', name: 'Reserva', type: 'savings', currentBalanceCents: 100000 }),
    ]
    const transactions = [
      transaction({ id: 'negative-rent', kind: 'expense', amountCents: 120000, accountId: 'checking', accountName: 'Conta principal', categoryId: 'home', categoryName: 'Casa', description: 'Aluguel atrasado', occurrenceDate: '2026-08-10', origin: 'fixed_expense' }),
      transaction({ id: 'negative-fee', kind: 'expense', amountCents: 24340, accountId: 'checking', accountName: 'Conta principal', categoryId: 'home', categoryName: 'Casa', description: 'Taxa inesperada', occurrenceDate: '2026-08-22', origin: 'import', automaticImport: true, importBank: 'bradesco' }),
    ]
    return {
      profile: { displayName: 'Rui', currency: 'BRL', theme, onboardingStatus: 'completed', balancesHidden: false },
      setup: true,
      theme,
      accounts,
      categories,
      dashboard: {
        availableBalanceCents: -18340,
        totalBalanceCents: 81660,
        pendingFixedExpensesCents: 9900,
        pendingFixedExpenseCount: 1,
        monthlyIncomeCents: 320000,
        monthlyExpenseCents: 362680,
        recentTransactions: transactions,
        balanceHistory: [
          { month: '2026-02', label: 'fev.', balanceCents: 250000 },
          { month: '2026-03', label: 'mar.', balanceCents: 214000 },
          { month: '2026-04', label: 'abr.', balanceCents: 180000 },
          { month: '2026-05', label: 'mai.', balanceCents: 142000 },
          { month: '2026-06', label: 'jun.', balanceCents: 110000 },
          { month: '2026-07', label: 'jul.', balanceCents: 100000 },
          { month: '2026-08', label: 'ago.', balanceCents: 81660 },
        ],
        accountAllocations: accounts.map(item => ({ accountId: item.id, accountName: item.name, balanceCents: item.currentBalanceCents })),
        hasNegativeBalance: true,
        reservedValueCents: 100000,
        eligibleBalanceCents: -18340,
        freeValueCents: -18340,
        safelySpendableCents: 0,
        budgetProgressPercent: 108,
        goalProgressPercent: 12,
      },
      planning: {
        budget: {
          referenceMonth: '2026-08', overallLimitCents: 335000, spentCents: 362680, remainingCents: -27680, progressPercent: 108,
          categoryLimits: [{ id: 'limit-home', categoryId: 'home', categoryName: 'Casa', limitCents: 100000, rollover: false, rolloverCents: 0, spentCents: 144340, availableCents: -44340, exceeded: true }],
        },
        goals: [goal({ id: 'emergency', name: 'Reserva de emergência', kind: 'emergency_reserve', targetCents: 800000, allocatedCents: 100000, progressPercent: 12.5 })],
      },
    }
  }

  const accounts = richAccounts()
  const transactions = richTransactions()
  return {
    profile: { displayName: 'Lia', currency: 'BRL', theme, onboardingStatus: 'completed', balancesHidden: false },
    setup: true,
    theme,
    accounts,
    categories,
    dashboard: {
      availableBalanceCents: 473300,
      totalBalanceCents: 593300,
      pendingFixedExpensesCents: 9900,
      pendingFixedExpenseCount: 1,
      monthlyIncomeCents: 520000,
      monthlyExpenseCents: 327320,
      recentTransactions: transactions,
      balanceHistory: [
        { month: '2026-02', label: 'fev.', balanceCents: 360000 },
        { month: '2026-03', label: 'mar.', balanceCents: 405000 },
        { month: '2026-04', label: 'abr.', balanceCents: 448000 },
        { month: '2026-05', label: 'mai.', balanceCents: 490000 },
        { month: '2026-06', label: 'jun.', balanceCents: 528000 },
        { month: '2026-07', label: 'jul.', balanceCents: 566000 },
        { month: '2026-08', label: 'ago.', balanceCents: 593300 },
      ],
      accountAllocations: accounts.filter(item => item.type !== 'credit_card').map(item => ({ accountId: item.id, accountName: item.name, balanceCents: item.currentBalanceCents })),
      hasNegativeBalance: false,
      creditCardDebtCents: 350000,
      upcomingInvoices: [{ id: 'invoice-aug', accountId: 'card', accountName: 'Nubank', referenceMonth: '2026-09', closingDate: '2026-08-25', dueDate: '2026-09-02', status: 'open', chargesCents: 129900, carryForwardCents: 0, paidCents: 0, outstandingCents: 129900, installments: [], payments: [] }],
      reservedValueCents: 700000,
      eligibleBalanceCents: 1293300,
      freeValueCents: 593300,
      safelySpendableCents: 473300,
      budgetProgressPercent: 63.75,
      goalProgressPercent: 46.67,
    },
    planning: { budget: richBudget, goals: richGoals() },
  }
}

export function createPreviewFixedExpenses(scenario: PreviewScenario): FixedExpensesOverview {
  if (scenario === 'empty' || scenario === 'onboarding') return { expenses: [], occurrences: [] }
  const expense = (id: string, description: string, amountCents: number, dueDay: number, categoryId: string, categoryName: string) => ({ id, description, amountCents, dueDay, accountId: 'checking', accountName: 'Conta principal', categoryId, categoryName, createdAt, updatedAt: createdAt })
  const expenses = [
    expense('fixed-rent', 'Aluguel', scenario === 'negative' ? 120000 : 185000, 10, 'home', 'Casa'),
    expense('fixed-internet', 'Internet', 9900, 20, 'home', 'Casa'),
    expense('fixed-streaming', 'Streaming', 4490, 27, 'leisure', 'Lazer'),
  ]
  const occurrences = scenario === 'negative'
    ? [{ id: 'occ-internet', fixedExpenseId: 'fixed-internet', referenceMonth: '2026-08', dueDate: '2026-08-20', description: 'Internet', expectedAmountCents: 9900, accountId: 'checking', accountName: 'Conta principal', categoryId: 'home', categoryName: 'Casa', status: 'pending' as const, createdAt, updatedAt: createdAt }]
    : [
      { id: 'occ-internet', fixedExpenseId: 'fixed-internet', referenceMonth: '2026-08', dueDate: '2026-09-20', description: 'Internet', expectedAmountCents: 9900, accountId: 'checking', accountName: 'Conta principal', categoryId: 'home', categoryName: 'Casa', status: 'pending' as const, createdAt, updatedAt: createdAt },
      { id: 'occ-rent', fixedExpenseId: 'fixed-rent', referenceMonth: '2026-08', dueDate: '2026-08-10', description: 'Aluguel', expectedAmountCents: 185000, accountId: 'checking', accountName: 'Conta principal', categoryId: 'home', categoryName: 'Casa', status: 'confirmed' as const, createdAt, updatedAt: createdAt, transactionId: 'tx-rent' },
    ]
  return { expenses, occurrences }
}

export function createPreviewCreditCards(scenario: PreviewScenario, accounts: Account[]): CreditCardsOverview {
  const card = accounts.find(item => item.type === 'credit_card')
  if (scenario !== 'rich' || !card) return { cards: [], invoices: [] }
  const invoice = {
    id: 'invoice-aug', accountId: card.id, accountName: card.name, referenceMonth: '2026-09', closingDate: '2026-08-25', dueDate: '2026-09-02', status: 'open' as const,
    chargesCents: 129900, carryForwardCents: 0, paidCents: 0, outstandingCents: 129900,
    installments: [{ id: 'installment-notebook', invoiceId: 'invoice-aug', transactionId: 'tx-notebook', description: 'Notebook', amountCents: 21650, installmentNumber: 1, installmentCount: 6 }], payments: [],
  }
  return {
    cards: [{ account: card, outstandingCents: 129900, availableLimitCents: 670100, currentInvoice: invoice }],
    invoices: [invoice],
  }
}

export function createPreviewOccurrences(scenario: PreviewScenario): TransactionOccurrence[] {
  if (scenario !== 'rich') return []
  return [{
    id: 'recurrence-1', recurrenceRuleId: 'rule-1', accountId: 'checking', accountName: 'Conta principal', kind: 'expense', categoryId: 'home', categoryName: 'Casa', amountCents: 4490, description: 'Streaming', scheduledDate: '2026-09-27', status: 'pending', installmentNumber: 1, installmentCount: 1, tags: [], splits: [],
  }]
}
