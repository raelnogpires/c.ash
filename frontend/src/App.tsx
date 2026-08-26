import { useCallback, useEffect, useId, useRef, useState } from 'react'
import { api as defaultAPI } from './api'
import { AccountActions, Button, EmptyState, Icon, TransactionList, type IconName } from './components'
import { AccountForm, BalanceAdjustmentDialog, ConfirmFixedExpenseDialog, DeleteAccountDialog, FixedExpenseForm, Onboarding, TransactionDialog } from './forms'
import { formatBRL, formatDate, parseBRL, today } from './format'
import type { Account, AccountInput, BackupInfo, BackupStatus, BalanceAdjustmentInput, Bank, BankStatementImportResult, BankStatementInput, BootstrapData, ConfirmFixedExpenseOccurrenceInput, CreditCardInvoice, CreditCardPaymentInput, CreditCardsOverview, FixedExpense, FixedExpenseInput, FixedExpenseOccurrence, FixedExpensesOverview, Goal, GoalAllocationInput, GoalInput, MonthlyBudgetInput, OnboardingInput, SecurityStatus, Theme, Transaction, TransactionInput, UpdateStatus } from './types'
import { EventsOn } from './wailsjs/runtime/runtime'

type View = 'dashboard' | 'accounts' | 'cards' | 'transactions' | 'fixedExpenses' | 'budget' | 'goals' | 'settings'
type API = typeof defaultAPI

const systemTheme = (): Theme => window.matchMedia?.('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
const errorMessage = (error: unknown) => error instanceof Error ? error.message : String(error)

export default function App({ api = defaultAPI }: { api?: API }) {
  const [data, setData] = useState<BootstrapData | null>(null)
  const [view, setView] = useState<View>('dashboard')
  const [error, setError] = useState('')
  const [transactionDialog, setTransactionDialog] = useState<{ initial?: Transaction } | null>(null)
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [fixedOverview, setFixedOverview] = useState<FixedExpensesOverview | null>(null)
  const [creditOverview, setCreditOverview] = useState<CreditCardsOverview | null>(null)
  const [transactionSnackbar, setTransactionSnackbar] = useState<{ transaction: Transaction } | null>(null)
  const [dismissedOccurrence, setDismissedOccurrence] = useState<FixedExpenseOccurrence | null>(null)
  const [fallbackTheme, setFallbackTheme] = useState<Theme>(systemTheme)
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const [updateStatus, setUpdateStatus] = useState<UpdateStatus | null>(null)
  const [updateDialog, setUpdateDialog] = useState(false)
  const [updateDismissed, setUpdateDismissed] = useState(false)
  const [security, setSecurity] = useState<SecurityStatus | null>(null)
  const dialogTrigger = useRef<HTMLElement | null>(null)

  const load = useCallback(async () => {
    try {
      setError('')
      setData(await api.Bootstrap())
    } catch (err) {
      setError(errorMessage(err))
    }
  }, [api])
  const loadFixedExpenses = useCallback(async () => {
    try {
      setFixedOverview(await api.FixedExpensesOverview())
    } catch (err) {
      setError(errorMessage(err))
    }
  }, [api])
  const loadCreditCards = useCallback(async () => { try { setCreditOverview(await api.CreditCardsOverview()) } catch(err) { setError(errorMessage(err)) } },[api])

  useEffect(() => {
    let active = true
    const initialize = async () => {
      try {
        const status = api.SecurityStatus ? await api.SecurityStatus() : { enabled: false, locked: false }
        if (!active) return
        setSecurity(status)
        if (!status.locked) await load()
      } catch (err) {
        if (active) setError(errorMessage(err))
      }
    }
    void initialize()
    return () => { active = false }
  }, [api, load])
  useEffect(() => {
    void api.GetUpdateStatus().then(setUpdateStatus).catch(() => undefined)
    const runtime = (window as typeof window & { runtime?: { EventsOn?: unknown } }).runtime
    if (!runtime?.EventsOn) return
    return EventsOn('update:status', (status: UpdateStatus) => setUpdateStatus(status))
  }, [api])
  useEffect(() => {
    const media = window.matchMedia?.('(prefers-color-scheme: dark)')
    if (!media) return
    const update = () => setFallbackTheme(media.matches ? 'dark' : 'light')
    media.addEventListener?.('change', update)
    return () => media.removeEventListener?.('change', update)
  }, [])
  useEffect(() => {
    if (view === 'transactions' && data?.setup) api.ListTransactions().then(setTransactions).catch(err => setError(errorMessage(err)))
    if (view === 'fixedExpenses' && data?.setup) void loadFixedExpenses()
    if (view === 'cards' && data?.setup) void loadCreditCards()
  }, [api, data?.setup, loadCreditCards, loadFixedExpenses, view])
  useEffect(() => { if (data?.setup) window.scrollTo(0, 0) }, [data?.setup])
  useEffect(() => {
    if (!transactionSnackbar) return
    const timeout = window.setTimeout(() => setTransactionSnackbar(null), 8000)
    return () => window.clearTimeout(timeout)
  }, [transactionSnackbar])
  useEffect(() => {
    if (!dismissedOccurrence) return
    const timeout = window.setTimeout(() => setDismissedOccurrence(null), 8000)
    return () => window.clearTimeout(timeout)
  }, [dismissedOccurrence])

  const theme = data?.theme || fallbackTheme
  useEffect(() => {
    document.documentElement.dataset.theme = theme
    document.documentElement.style.colorScheme = theme === 'light' ? 'light' : 'dark'
  }, [theme])

  const closeTransaction = useCallback(() => {
    setTransactionDialog(null)
    requestAnimationFrame(() => dialogTrigger.current?.focus())
  }, [])
  const openTransaction = useCallback((initial?: Transaction, trigger?: HTMLElement) => {
    dialogTrigger.current = trigger ?? (document.activeElement instanceof HTMLElement ? document.activeElement : null)
    setTransactionDialog({ initial })
  }, [])
  useEffect(() => {
    const listener = (event: KeyboardEvent) => {
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'n') {
        event.preventDefault()
        if (data?.accounts.length) openTransaction()
      }
    }
    window.addEventListener('keydown', listener)
    return () => window.removeEventListener('keydown', listener)
  }, [data, openTransaction])

  if (security?.locked) return <UnlockScreen api={api} onUnlocked={async status => { setSecurity(status); await load() }}/>
  if (!data && !error) return <main className="loading" aria-live="polite"><div className="brand"><span>[c]</span>ash</div><p>Preparando seu espaço…</p></main>
  if (!data) return <main className="fatal"><div className="brand"><span>[c]</span>ash</div><h1>Não conseguimos abrir seus dados</h1><p role="alert">{error}</p><Button onClick={() => void load()}>Tentar novamente</Button></main>
  if (!data.setup) return <Onboarding initialTheme={theme} onComplete={async (input: OnboardingInput) => { await api.CompleteOnboarding(input); await load() }} onSkip={async () => { await api.SkipOnboarding(); await load() }}/>

  const createAccount = async (input: AccountInput) => { await api.CreateAccount(input); await load() }
  const updateAccount = async (id: string, input: AccountInput) => { await api.UpdateAccount(id, input); await load() }
  const adjustAccount = async (id: string, input: BalanceAdjustmentInput) => { await api.AdjustAccountBalance(id, input); await load() }
  const deleteAccount = async (id: string) => { await api.DeleteAccount(id); await load() }
  const refreshTransactions = async () => { await load(); if (view === 'transactions') setTransactions(await api.ListTransactions()) }
  const refreshFixedExpenses = async () => { await load(); await loadFixedExpenses() }
  const saveTransaction = async (input: TransactionInput) => { if (transactionDialog?.initial) await api.UpdateTransaction(transactionDialog.initial.id, input); else await api.CreateTransaction(input); await refreshTransactions() }
  const importBankStatement = async (input: BankStatementInput) => { const result = await api.ImportBankStatement(input); await refreshTransactions(); return result }
  const removeTransaction = async (tx: Transaction, trigger: HTMLElement) => {
    const row = trigger.closest('li')
    const focusTarget = row?.nextElementSibling?.querySelector<HTMLElement>('.icon-button')
      ?? row?.previousElementSibling?.querySelector<HTMLElement>('.icon-button')
      ?? document.querySelector<HTMLElement>('.topbar .button')
    const before = data, beforeList = transactions
    setTransactionSnackbar({ transaction: tx })
    setTransactions(items => items.filter(item => item.id !== tx.id))
    setData(current => current ? { ...current, dashboard: { ...current.dashboard, recentTransactions: current.dashboard.recentTransactions.filter(item => item.id !== tx.id) } } : current)
    requestAnimationFrame(() => focusTarget?.focus())
    try { await api.TrashTransaction(tx.id); await refreshTransactions(); if (view === 'fixedExpenses') await loadFixedExpenses() }
    catch (err) { setData(before); setTransactions(beforeList); setTransactionSnackbar(null); setError(errorMessage(err)) }
  }
  const undoRemoval = async () => {
    if (!transactionSnackbar) return
    const tx = transactionSnackbar.transaction
    setTransactionSnackbar(null)
    try { await api.RestoreTransaction(tx.id); await refreshTransactions(); if (view === 'fixedExpenses') await loadFixedExpenses() }
    catch (err) { setError(errorMessage(err)) }
  }
  const setBalancesHidden = async (hidden: boolean) => {
    const before = data
    setData(current => current ? { ...current, profile: current.profile ? { ...current.profile, balancesHidden: hidden } : current.profile } : current)
    try {
	      const profile = await api.SetBalancesHidden(hidden)
	      setData(current => current ? { ...current, profile: { ...profile, balancesHidden: profile.balancesHidden ?? hidden } } : current)
    } catch (err) {
      setData(before)
      setError(errorMessage(err))
    }
  }
  const saveFixedExpense = async (input: FixedExpenseInput, initial?: FixedExpense) => {
    if (initial) await api.UpdateFixedExpense(initial.id, input)
    else await api.CreateFixedExpense(input)
    await refreshFixedExpenses()
  }
  const confirmFixedOccurrence = async (occurrence: FixedExpenseOccurrence, input: ConfirmFixedExpenseOccurrenceInput) => {
    await api.ConfirmFixedExpenseOccurrence(occurrence.id, input)
    await refreshFixedExpenses()
  }
  const dismissFixedOccurrence = async (occurrence: FixedExpenseOccurrence) => {
    await api.DismissFixedExpenseOccurrence(occurrence.id)
    setDismissedOccurrence(occurrence)
    await refreshFixedExpenses()
  }
  const undoDismissFixedOccurrence = async () => {
    if (!dismissedOccurrence) return
    const occurrence = dismissedOccurrence
    setDismissedOccurrence(null)
    try { await api.ReopenFixedExpenseOccurrence(occurrence.id); await refreshFixedExpenses() }
    catch (err) { setError(errorMessage(err)) }
  }
  const checkForUpdates = async () => {
    try { setUpdateStatus(await api.CheckForUpdates()) }
    catch (err) { setError(errorMessage(err)) }
  }
  const installUpdate = async () => {
    try { setUpdateStatus(await api.InstallUpdate()) }
    catch (err) { setError(errorMessage(err)); throw err }
  }

  const nav: { id: View; label: string; icon: IconName }[] = [
    { id: 'dashboard', label: 'Visão geral', icon: 'house' }, { id: 'accounts', label: 'Contas', icon: 'wallet' },
    { id: 'cards', label: 'Cartões e faturas', icon: 'receipt' },
    { id: 'transactions', label: 'Movimentações', icon: 'list' }, { id: 'fixedExpenses', label: 'Despesas fixas', icon: 'calendarClock' },
    { id: 'budget', label: 'Orçamento', icon: 'receipt' }, { id: 'goals', label: 'Metas', icon: 'wallet' },
    { id: 'settings', label: 'Configurações', icon: 'palette' },
  ]
  const balancesHidden = data.profile?.balancesHidden ?? false
  const sidebarToggleLabel = sidebarCollapsed ? 'Expandir navegação' : 'Recolher navegação'
  return <div className={`shell${sidebarCollapsed ? ' shell--collapsed' : ''}`}>
    <aside className={`sidebar${sidebarCollapsed ? ' sidebar--collapsed' : ''}`}><div className="sidebar__header"><div className="brand"><span>[c]</span>ash</div><button type="button" className="sidebar__toggle" aria-label={sidebarToggleLabel} aria-expanded={!sidebarCollapsed} aria-controls="primary-navigation" title={sidebarToggleLabel} onClick={() => setSidebarCollapsed(collapsed => !collapsed)}><Icon name={sidebarCollapsed ? 'panelLeftOpen' : 'panelLeftClose'}/></button></div><nav id="primary-navigation" className="sidebar__nav" aria-label="Principal">{nav.map(item => <button key={item.id} className={view === item.id ? 'active' : ''} aria-label={item.label} title={sidebarCollapsed ? item.label : undefined} aria-current={view === item.id ? 'page' : undefined} onClick={() => setView(item.id)}><Icon name={item.icon}/><span>{item.label}</span></button>)}</nav><div className="sidebar__footer"><span className="avatar" aria-hidden="true">{data.profile?.displayName?.[0]?.toUpperCase() || 'C'}</span><span><strong>{data.profile?.displayName || 'Meu espaço'}</strong><small>Dados locais</small></span></div></aside>
    <main className="workspace"><header className="topbar"><div>{view === 'dashboard' && <p className="eyebrow">Hoje</p>}<h1>{view === 'dashboard' ? `Olá${data.profile?.displayName ? `, ${data.profile.displayName}` : ''}` : nav.find(n => n.id === view)?.label}</h1></div><Button onClick={event => openTransaction(undefined, event.currentTarget)} disabled={!data.accounts.length} title={!data.accounts.length ? 'Crie uma conta primeiro' : 'Atalho: Ctrl+N'}><Icon name="plus"/> Nova movimentação</Button></header>
      {error && <div className="alert" role="alert">{error}<button onClick={() => setError('')} aria-label="Fechar aviso"><Icon name="close"/></button></div>}
      {data.setup && updateStatus?.state === 'available' && !updateDismissed && <section className="update-banner" role="status" aria-label="Atualização disponível"><div><strong>Uma atualização está disponível</strong><span>Versão {updateStatus.availableVersion}</span></div><div><Button kind="secondary" onClick={() => setUpdateDialog(true)}>Ver novidades</Button><button className="text-button muted" onClick={() => setUpdateDismissed(true)}>Depois</button></div></section>}
      {view === 'dashboard' && <DashboardView data={data} balancesHidden={balancesHidden} onBalancesHiddenChange={setBalancesHidden} onAccounts={() => setView('accounts')} onTransactions={() => setView('transactions')} onTransaction={() => openTransaction()} onEdit={openTransaction} onRemove={removeTransaction}/>} 
      {view === 'accounts' && <AccountsView data={data} create={createAccount} update={updateAccount} adjust={adjustAccount} remove={deleteAccount}/>}
      {view === 'cards' && <CreditCardsView data={data} overview={creditOverview} onPay={async(id,input)=>{await api.PayCreditCardInvoice(id,input);await load();await loadCreditCards()}} onCreateCard={()=>setView('accounts')}/>}
      {view === 'transactions' && <TransactionsView data={data} transactions={transactions} open={() => openTransaction()} onImport={importBankStatement} onEdit={openTransaction} onRemove={removeTransaction} listTrashed={() => api.ListTrashedTransactions()} restoreTrashed={async id => { await api.RestoreTransaction(id); await refreshTransactions() }} deletePermanently={id => api.DeleteTransactionPermanently(id)} emptyTrash={() => api.EmptyTransactionTrash()}/>}
      {view === 'fixedExpenses' && <FixedExpensesView data={data} overview={fixedOverview} create={input => saveFixedExpense(input)} update={(expense, input) => saveFixedExpense(input, expense)} archive={async id => { await api.ArchiveFixedExpense(id); await refreshFixedExpenses() }} restore={async id => { await api.RestoreFixedExpense(id); await refreshFixedExpenses() }} confirm={confirmFixedOccurrence} dismiss={dismissFixedOccurrence}/>} 
      {view === 'budget' && <BudgetView data={data} save={async input=>{await api.SetMonthlyBudget(input);await load()}}/>}
      {view === 'goals' && <GoalsView data={data} save={async(id,input)=>{await api.SaveGoal(id,input);await load()}} allocate={async(id,input)=>{await api.SetGoalAllocations(id,input);await load()}} archive={async id=>{await api.ArchiveGoal(id);await load()}}/>}
      {view === 'settings' && <SettingsView api={api} theme={theme} security={security ?? { enabled: false, locked: false }} updateStatus={updateStatus} setTheme={async value => { try { await api.SetTheme(value); await load() } catch (err) { setError(errorMessage(err)) } }} onCheckUpdates={checkForUpdates} onShowUpdate={() => setUpdateDialog(true)} onRestored={async () => { setView('dashboard'); await load() }} onSecurityChange={setSecurity}/>}
    </main>
    {transactionDialog && <TransactionDialog accounts={data.accounts} categories={data.categories} initial={transactionDialog.initial} onClose={closeTransaction} onSubmit={saveTransaction}/>} 
    {transactionSnackbar && <div className="snackbar" role="status"><span>Movimentação removida</span><button onClick={() => void undoRemoval()}>Desfazer</button></div>}
    {dismissedOccurrence && <div className="snackbar" role="status"><span>Previsão dispensada</span><button onClick={() => void undoDismissFixedOccurrence()}>Desfazer</button></div>}
    {updateDialog && updateStatus && ['available', 'downloading', 'installing', 'error'].includes(updateStatus.state) && <UpdateDialog status={updateStatus} onClose={() => setUpdateDialog(false)} onInstall={installUpdate}/>} 
  </div>
}

function DashboardView({ data, balancesHidden, onBalancesHiddenChange, onAccounts, onTransactions, onTransaction, onEdit, onRemove }: { data: BootstrapData; balancesHidden: boolean; onBalancesHiddenChange(hidden: boolean): void; onAccounts(): void; onTransactions(): void; onTransaction(): void; onEdit(tx: Transaction, trigger: HTMLElement): void; onRemove(tx: Transaction, trigger: HTMLElement): void }) {
  const d = data.dashboard, total = d.totalBalanceCents ?? d.availableBalanceCents, pending = d.pendingFixedExpensesCents ?? 0, pendingCount = d.pendingFixedExpenseCount ?? 0
  return <div className="page dashboard-page"><section className={`balance-hero stat-led-hero ${d.hasNegativeBalance ? 'negative' : ''}`} aria-labelledby="available-balance-label"><div className="balance-hero__content"><p className="balance-hero__label" id="available-balance-label"><span>Disponível após despesas fixas</span><button className="balance-visibility" type="button" aria-label={balancesHidden ? 'Mostrar saldos' : 'Ocultar saldos'} aria-pressed={balancesHidden} title={balancesHidden ? 'Mostrar saldos' : 'Ocultar saldos'} onClick={() => onBalancesHiddenChange(!balancesHidden)}><Icon name={balancesHidden ? 'eyeOff' : 'eye'}/></button></p><strong className="balance-hero__value" aria-label={balancesHidden ? 'Saldo oculto' : undefined}>{balancesHidden ? '••••••' : formatBRL(d.availableBalanceCents)}</strong></div><div className="balance-hero__meta" aria-label="Contexto do saldo disponível"><div className="balance-hero__meta-row"><span>{pendingCount ? `${pendingCount} compromisso${pendingCount === 1 ? '' : 's'} pendente${pendingCount === 1 ? '' : 's'}` : 'Compromissos pendentes'}</span><strong>{balancesHidden ? 'Oculto' : pendingCount ? formatBRL(pending) : 'Nenhum'}</strong></div><div className="balance-hero__meta-row"><span>Saldo total</span><strong>{balancesHidden ? 'Oculto' : formatBRL(total)}</strong></div></div></section>
    {d.hasNegativeBalance && <div className="warning" role="status"><Icon name="warning"/><span><strong>Seu saldo está negativo.</strong> Revise os registros e evite novos compromissos até regularizar.</span></div>}
    <section className="metrics metrics--supporting" aria-label="Resumo do mês"><article><span className="metric-icon income" aria-hidden="true"><Icon name="arrowUpRight"/></span><div><p>Receitas no mês</p><strong>+ {formatBRL(d.monthlyIncomeCents)}</strong></div></article><article><span className="metric-icon expense" aria-hidden="true"><Icon name="arrowDownRight"/></span><div><p>Despesas no mês</p><strong>− {formatBRL(d.monthlyExpenseCents)}</strong></div></article><article><span className="metric-icon neutral" aria-hidden="true"><Icon name="equal"/></span><div><p>Resultado do mês</p><strong>{formatBRL(d.monthlyIncomeCents - d.monthlyExpenseCents)}</strong></div></article></section>
    <section className="metrics metrics--supporting" aria-label="Planejamento"><article><div><p>Valor reservado</p><strong>{balancesHidden?'••••••':formatBRL(d.reservedValueCents??0)}</strong></div></article><article><div><p>Valor livre</p><strong>{balancesHidden?'••••••':formatBRL(d.freeValueCents??d.availableBalanceCents)}</strong></div></article><article><div><p>Progresso das metas</p><strong>{Math.round(d.goalProgressPercent??0)}%</strong></div></article></section>
    {(d.creditCardDebtCents??0)>0&&<section className="card invoice-dashboard"><div><h2>Cartões de crédito</h2><p>Dívida reconhecida no patrimônio</p></div><strong>{balancesHidden?'••••••':formatBRL(d.creditCardDebtCents??0)}</strong>{d.upcomingInvoices?.[0]&&<small>Próxima fatura vence {formatDate(d.upcomingInvoices[0].dueDate)} · {balancesHidden?'valor oculto':formatBRL(d.upcomingInvoices[0].outstandingCents)}</small>}</section>}
    <section className="dashboard-analysis" aria-label="Evolução e distribuição dos saldos"><BalanceTrendChart points={d.balanceHistory ?? []} hidden={balancesHidden}/><AccountAllocationChart allocations={d.accountAllocations ?? []} hidden={balancesHidden}/></section>
    <section className="card"><div className="card__heading"><div><h2>Atividade recente</h2></div>{d.recentTransactions.length > 0 && <button className="text-button" onClick={onTransactions}>Ver todas <Icon name="chevronRight"/></button>}</div><TransactionList transactions={d.recentTransactions} onEdit={onEdit} onRemove={onRemove} empty={<EmptyState icon={data.accounts.length ? 'receipt' : 'walletMinimal'} title={data.accounts.length ? 'Seu histórico começa aqui' : 'Crie sua primeira conta'} action={<Button onClick={data.accounts.length ? onTransaction : onAccounts}>{data.accounts.length ? 'Registrar movimentação' : 'Criar conta'}</Button>}>{data.accounts.length ? 'Registre uma receita, despesa ou transferência para acompanhar seu mês.' : 'Adicione onde você guarda dinheiro para começar a organizar.'}</EmptyState>}/></section></div>
}

function BalanceTrendChart({ points, hidden }: { points: NonNullable<BootstrapData['dashboard']['balanceHistory']>; hidden: boolean }) {
  if (hidden) return <section className="card analysis-card analysis-card--hidden" aria-label="Evolução do saldo"><div><h2>Evolução do saldo</h2></div><p role="status">Saldos ocultos</p></section>
  if (!points.length) return <section className="card analysis-card"><div><h2>Evolução do saldo</h2></div><p className="analysis-empty">Registre movimentações para acompanhar sua evolução.</p></section>
  const values = points.map(point => point.balanceCents), min = Math.min(...values), max = Math.max(...values), range = Math.max(max - min, 1)
  const positions = points.map((point, index) => ({ ...point, x: 24 + (512 * index) / Math.max(points.length - 1, 1), y: 24 + (112 * (max - point.balanceCents)) / range }))
  const path = positions.map((point, index) => `${index ? 'L' : 'M'}${point.x.toFixed(1)},${point.y.toFixed(1)}`).join(' ')
  return <section className="card analysis-card"><div className="card__heading"><div><h2>Evolução do saldo</h2></div><span className="analysis-total">{formatBRL(points.at(-1)?.balanceCents ?? 0)}</span></div><figure className="trend-chart"><svg viewBox="0 0 560 180" role="img" aria-labelledby="trend-title trend-description"><title id="trend-title">Evolução do saldo total nos últimos sete meses</title><desc id="trend-description">O saldo mais recente é {formatBRL(points.at(-1)?.balanceCents ?? 0)}.</desc><path className="trend-chart__baseline" d="M24 140H536"/><path className="trend-chart__line" d={path}/>{positions.map(point => <circle key={point.month} className="trend-chart__point" cx={point.x} cy={point.y} r="4"><title>{formatMonth(point.month)}: {formatBRL(point.balanceCents)}</title></circle>)}</svg><figcaption>{points.map(point => <span key={point.month}>{formatMonth(point.month)}</span>)}</figcaption></figure><ul className="sr-only">{points.map(point => <li key={point.month}>{formatMonth(point.month)}: {formatBRL(point.balanceCents)}</li>)}</ul></section>
}

function AccountAllocationChart({ allocations, hidden }: { allocations: NonNullable<BootstrapData['dashboard']['accountAllocations']>; hidden: boolean }) {
  if (hidden) return <section className="card allocation-card analysis-card--hidden" aria-label="Saldos por conta"><div><h2>Onde está seu dinheiro</h2></div><p role="status">Saldos ocultos</p></section>
  if (!allocations.length) return <section className="card allocation-card"><div><h2>Onde está seu dinheiro</h2></div><p className="analysis-empty">Suas contas aparecerão aqui.</p></section>
  const largest = Math.max(...allocations.map(item => Math.abs(item.balanceCents)), 1)
  return <section className="card allocation-card"><div><h2>Onde está seu dinheiro</h2></div><ul className="allocation-list">{allocations.map(item => <li key={item.accountId}><div><span>{item.accountName}</span><strong>{formatBRL(item.balanceCents)}</strong></div><span className={`allocation-bar ${item.balanceCents < 0 ? 'negative' : ''}`} aria-hidden="true"><i style={{ width: `${Math.max((Math.abs(item.balanceCents) / largest) * 100, 3)}%` }}/></span></li>)}</ul></section>
}

function formatMonth(month: string) { return ['jan.', 'fev.', 'mar.', 'abr.', 'mai.', 'jun.', 'jul.', 'ago.', 'set.', 'out.', 'nov.', 'dez.'][Number(month.slice(5, 7)) - 1] ?? month }

function AccountsView({ data, create, update, adjust, remove }: { data: BootstrapData; create(input: AccountInput): Promise<void>; update(id: string, input: AccountInput): Promise<void>; adjust(id: string, input: BalanceAdjustmentInput): Promise<void>; remove(id: string): Promise<void> }) {
  const [creating, setCreating] = useState(data.accounts.length === 0), [editing, setEditing] = useState<Account | undefined>(), [adjusting, setAdjusting] = useState<Account | undefined>(), [deleting, setDeleting] = useState<Account | undefined>()
  const actionTrigger = useRef<HTMLElement | null>(null), addTrigger = useRef<HTMLButtonElement>(null)
  const closeEditor = () => { setCreating(false); setEditing(undefined); requestAnimationFrame(() => actionTrigger.current?.focus()) }
  const closeDelete = () => { setDeleting(undefined); requestAnimationFrame(() => actionTrigger.current?.focus()) }
  return <div className="page page--split"><section><div className="section-heading"><p className="section-summary">{data.accounts.length ? `${data.accounts.length} ${data.accounts.length === 1 ? 'conta cadastrada' : 'contas cadastradas'}` : 'Nenhuma conta cadastrada'}</p>{!creating && !editing && <Button ref={addTrigger} kind="secondary" onClick={() => { setCreating(true); setEditing(undefined) }}><Icon name="plus"/> Adicionar</Button>}</div>{data.accounts.length ? <div className="account-grid">{data.accounts.map(account => <article className="account-card" key={account.id}><AccountActions account={account} onEdit={(item, trigger) => { actionTrigger.current = trigger; setCreating(false); setEditing(item) }} onAdjust={(item, trigger) => { actionTrigger.current=trigger; setAdjusting(item) }} onRemove={(item, trigger) => { actionTrigger.current = trigger; setDeleting(item) }}/><span>{account.type === 'checking' ? 'Conta corrente' : account.type === 'savings' ? 'Poupança' : account.type==='credit_card'?'Cartão de crédito':'Dinheiro'}</span><h3>{account.name}</h3><strong>{formatBRL(account.type==='credit_card'?Math.max(0,-account.currentBalanceCents):account.currentBalanceCents)}</strong><small>{account.type==='credit_card'?`Em aberto · limite ${formatBRL(account.creditLimitCents??0)}`:'Saldo calculado'}</small></article>)}</div> : !creating && <EmptyState icon="walletMinimal" title="Nenhuma conta ainda" action={<Button onClick={() => setCreating(true)}>Criar conta</Button>}>Crie sua primeira conta para registrar movimentações.</EmptyState>}</section>{(creating || editing) && <AccountForm key={editing?.id ?? 'new'} initial={editing} onSubmit={async input => { if (editing) await update(editing.id, input); else await create(input); closeEditor() }} onCancel={data.accounts.length ? closeEditor : undefined}/>} {adjusting&&<BalanceAdjustmentDialog account={adjusting} onClose={()=>{setAdjusting(undefined);requestAnimationFrame(()=>actionTrigger.current?.focus())}} onSubmit={input=>adjust(adjusting.id,input)}/>} {deleting && <DeleteAccountDialog account={deleting} onClose={closeDelete} onConfirm={async () => { await remove(deleting.id); setDeleting(undefined); requestAnimationFrame(() => addTrigger.current?.focus()) }}/>}</div>
}

function BudgetView({data,save}:{data:BootstrapData;save(input:MonthlyBudgetInput):Promise<void>}){
  const current=data.planning?.budget,month=current?.referenceMonth??today().slice(0,7),[overall,setOverall]=useState(current?(current.overallLimitCents/100).toLocaleString('pt-BR',{minimumFractionDigits:2}):''),[categoryId,setCategoryId]=useState(''),[categoryValue,setCategoryValue]=useState(''),[rollover,setRollover]=useState(false),[error,setError]=useState(''),[busy,setBusy]=useState(false)
  const submit=async(event:React.FormEvent)=>{event.preventDefault();const cents=parseBRL(overall);if(cents===null||cents<0)return setError('Informe um limite válido.');const categoryCents=parseBRL(categoryValue);const categoryLimits=categoryId&&categoryCents!==null?[{categoryId,limitCents:categoryCents,rollover}]:[];setBusy(true);setError('');try{await save({referenceMonth:month,overallLimitCents:cents,categoryLimits})}catch(err){setError(errorMessage(err))}finally{setBusy(false)}}
  return <div className="page page--split"><section><div className="card"><div className="card__heading"><div><h2>Progresso de {month}</h2><p>Despesas realizadas; ajustes e pagamentos de fatura não entram.</p></div></div>{current?<><progress value={Math.min(current.progressPercent,100)} max="100"/><p><strong>{formatBRL(current.spentCents)}</strong> de {formatBRL(current.overallLimitCents)} · {formatBRL(current.remainingCents)} restantes</p>{current.categoryLimits.length>0&&<ul>{current.categoryLimits.map(limit=><li key={limit.id} className={limit.exceeded?'warning':''}><strong>{limit.categoryName}</strong> · {formatBRL(limit.spentCents)} de {formatBRL(limit.limitCents+limit.rolloverCents)}{limit.exceeded?' · limite excedido':''}</li>)}</ul>}</>:<p className="analysis-empty">Defina um limite para acompanhar o mês.</p>}</div></section><form className="card form-card" onSubmit={submit}><h2>Orçamento mensal</h2><label className="field"><span>Limite geral</span><div className="money-input"><span>R$</span><input value={overall} onChange={e=>setOverall(e.target.value)} inputMode="decimal" required/></div></label><label className="field"><span>Limite por categoria <em>opcional</em></span><select value={categoryId} onChange={e=>setCategoryId(e.target.value)}><option value="">Sem limite específico</option>{data.categories.filter(c=>c.kind==='expense').map(c=><option key={c.id} value={c.id}>{c.name}</option>)}</select></label>{categoryId&&<><label className="field"><span>Limite da categoria</span><input value={categoryValue} onChange={e=>setCategoryValue(e.target.value)} inputMode="decimal" required/></label><label className="confirmation-check"><input type="checkbox" checked={rollover} onChange={e=>setRollover(e.target.checked)}/>Acumular saldo não usado</label></>}{error&&<p className="form-error" role="alert">{error}</p>}<Button loading={busy}>Salvar orçamento</Button></form></div>
}

function GoalsView({data,save,allocate,archive}:{data:BootstrapData;save(id:string,input:GoalInput):Promise<void>;allocate(id:string,input:GoalAllocationInput[]):Promise<void>;archive(id:string):Promise<void>}){
  const goals=data.planning?.goals.filter(goal=>!goal.archivedAt)??[],[editing,setEditing]=useState<Goal|undefined>(),[allocating,setAllocating]=useState<Goal|undefined>(),[error,setError]=useState('')
  const submit=async(event:React.FormEvent<HTMLFormElement>)=>{event.preventDefault();const form=new FormData(event.currentTarget),target=parseBRL(String(form.get('target')));if(target===null||target<0)return setError('Informe um alvo válido.');try{await save(editing?.id??'',{name:String(form.get('name')).trim(),kind:String(form.get('kind')) as GoalInput['kind'],targetCents:target,deadline:String(form.get('deadline'))});setEditing(undefined);setError('')}catch(err){setError(errorMessage(err))}}
  const submitAllocation=async(event:React.FormEvent<HTMLFormElement>)=>{event.preventDefault();if(!allocating)return;const form=new FormData(event.currentTarget),items=data.accounts.filter(a=>a.type!=='credit_card').map(a=>({accountId:a.id,amountCents:parseBRL(String(form.get(`account-${a.id}`)))??0}));try{await allocate(allocating.id,items);setAllocating(undefined);setError('')}catch(err){setError(errorMessage(err))}}
  return <div className="page page--split"><section><div className="section-heading"><p>{goals.length?`${goals.length} meta${goals.length===1?'':'s'} ativa${goals.length===1?'':'s'}`:'Nenhuma meta ativa'}</p><Button kind="secondary" onClick={()=>setEditing({} as Goal)}>Adicionar meta</Button></div>{goals.map(goal=><article className="card" key={goal.id}><h2>{goal.name}</h2><p>{goal.kind==='emergency_reserve'?'Reserva de emergência':'Poupança'} · {formatBRL(goal.allocatedCents)} de {formatBRL(goal.targetCents)}</p><progress value={Math.min(goal.progressPercent,100)} max="100"/><div className="settings-actions"><Button kind="secondary" onClick={()=>setAllocating(goal)}>Reservar por conta</Button><button className="text-button" onClick={()=>setEditing(goal)}>Editar</button><button className="text-button danger" onClick={()=>void archive(goal.id)}>Arquivar</button></div></article>)}</section>{editing&&<form className="card form-card" onSubmit={submit}><h2>{editing.id?'Editar meta':'Nova meta'}</h2><label className="field"><span>Nome</span><input name="name" defaultValue={editing.name??''} required autoFocus/></label><label className="field"><span>Tipo</span><select name="kind" defaultValue={editing.kind??'savings'}><option value="emergency_reserve">Reserva de emergência</option><option value="savings">Poupança</option></select></label><label className="field"><span>Valor alvo</span><input name="target" inputMode="decimal" defaultValue={editing.targetCents?(editing.targetCents/100).toLocaleString('pt-BR',{minimumFractionDigits:2}):''} required/></label><label className="field"><span>Prazo <em>opcional</em></span><input name="deadline" type="date" defaultValue={editing.deadline??''}/></label>{error&&<p className="form-error" role="alert">{error}</p>}<Button>Salvar meta</Button></form>}{allocating&&<div className="dialog-backdrop"><section className="dialog" role="dialog" aria-modal="true" aria-label={`Reservar para ${allocating.name}`}><h2>Reservar valores por conta</h2><p>Reservar não movimenta dinheiro nem cria transação.</p><form onSubmit={submitAllocation}>{data.accounts.filter(a=>a.type!=='credit_card').map(account=><label className="field" key={account.id}><span>{account.name} · saldo {formatBRL(account.currentBalanceCents)}</span><input name={`account-${account.id}`} inputMode="decimal" defaultValue={(allocating.allocations.find(x=>x.accountId===account.id)?.amountCents??0)/100}/></label>)}{error&&<p role="alert" className="form-error">{error}</p>}<div className="form-actions"><Button type="button" kind="ghost" onClick={()=>setAllocating(undefined)}>Cancelar</Button><Button>Salvar reservas</Button></div></form></section></div>}</div>
}

function CreditCardsView({data,overview,onPay,onCreateCard}:{data:BootstrapData;overview:CreditCardsOverview|null;onPay(id:string,input:CreditCardPaymentInput):Promise<void>;onCreateCard():void}){
  const [paying,setPaying]=useState<CreditCardInvoice|undefined>(),[accountId,setAccountId]=useState(''),[amount,setAmount]=useState(''),[paymentError,setPaymentError]=useState(''),[busy,setBusy]=useState(false)
  const sources=data.accounts.filter(account=>account.type!=='credit_card')
  if(!overview)return <div className="page"><section className="card"><p className="analysis-empty">Carregando cartões e faturas…</p></section></div>
  if(!overview.cards.length)return <div className="page"><EmptyState icon="wallet" title="Nenhum cartão cadastrado" action={<Button onClick={onCreateCard}>Adicionar cartão</Button>}>Cadastre limite, fechamento e vencimento para organizar compras e faturas.</EmptyState></div>
  const openPayment=(invoice:CreditCardInvoice)=>{setPaying(invoice);setAccountId(sources[0]?.id??'');setAmount((invoice.outstandingCents/100).toLocaleString('pt-BR',{minimumFractionDigits:2,maximumFractionDigits:2}));setPaymentError('')}
  const submit=async(event:React.FormEvent)=>{event.preventDefault();if(!paying)return;const cents=parseBRL(amount);if(cents===null||cents<=0)return setPaymentError('Informe um valor maior que zero.');setBusy(true);setPaymentError('');try{await onPay(paying.id,{accountId,amountCents:cents,occurrenceDate:today()});setPaying(undefined)}catch(err){setPaymentError(errorMessage(err))}finally{setBusy(false)}}
  const status=(value:CreditCardInvoice['status'])=>value==='open'?'Aberta':value==='closed'?'Fechada':value==='paid'?'Paga':'Transferida'
  return <div className="page cards-page"><section className="card-summary-grid">{overview.cards.map(card=><article className={`card credit-summary ${card.availableLimitCents<0?'negative':''}`} key={card.account.id}><span>Cartão de crédito</span><h2>{card.account.name}</h2><p>Em aberto <strong>{formatBRL(card.outstandingCents)}</strong></p><small>Limite disponível {formatBRL(card.availableLimitCents)} · fecha dia {card.account.closingDay} · vence dia {card.account.dueDay}</small></article>)}</section><section className="card invoice-list"><div className="card__heading"><div><h2>Faturas</h2><p>Compras parceladas aparecem no mês em que serão cobradas.</p></div></div>{overview.invoices.length?<ul>{overview.invoices.map(invoice=><li key={invoice.id}><div><span className="occurrence-status">{status(invoice.status)}</span><strong>{invoice.accountName} · {formatMonth(invoice.referenceMonth)}</strong><small>Fecha {formatDate(invoice.closingDate)} · vence {formatDate(invoice.dueDate)}</small>{invoice.installments.length>0&&<span>{invoice.installments.map(item=><small key={item.id}>{item.description}{item.installmentCount>1?` · ${item.installmentNumber}/${item.installmentCount}`:''} · {formatBRL(item.amountCents)}</small>)}</span>}</div><div><strong>{formatBRL(invoice.outstandingCents)}</strong>{(invoice.status==='open'||invoice.status==='closed')&&invoice.outstandingCents>0&&<Button kind="secondary" disabled={!sources.length} onClick={()=>openPayment(invoice)}>Pagar fatura</Button>}</div></li>)}</ul>:<p className="analysis-empty">As faturas aparecerão depois da primeira compra.</p>}</section>{paying&&<div className="dialog-backdrop"><section className="dialog dialog--confirm" role="dialog" aria-modal="true" aria-labelledby="pay-invoice-title"><header><h2 id="pay-invoice-title">Pagar fatura de {formatMonth(paying.referenceMonth)}</h2><button className="icon-button" onClick={()=>setPaying(undefined)} aria-label="Fechar"><Icon name="close"/></button></header><form onSubmit={submit}><label className="field"><span>Conta de pagamento</span><select value={accountId} onChange={event=>setAccountId(event.target.value)} required>{sources.map(account=><option value={account.id} key={account.id}>{account.name}</option>)}</select></label><label className="field"><span>Valor do pagamento</span><div className="money-input"><span>R$</span><input value={amount} onChange={event=>setAmount(event.target.value)} inputMode="decimal" required/></div></label>{paymentError&&<p className="form-error" role="alert">{paymentError}</p>}<div className="form-actions"><Button kind="ghost" type="button" onClick={()=>setPaying(undefined)}>Cancelar</Button><Button loading={busy}>Confirmar pagamento</Button></div></form></section></div>}</div>
}

function TransactionsView({ data, transactions, open, onImport, onEdit, onRemove, listTrashed, restoreTrashed, deletePermanently, emptyTrash }: { data: BootstrapData; transactions: Transaction[]; open(): void; onImport(input: BankStatementInput): Promise<BankStatementImportResult>; onEdit(tx: Transaction, trigger: HTMLElement): void; onRemove(tx: Transaction, trigger: HTMLElement): void; listTrashed(): Promise<Transaction[]>; restoreTrashed(id: string): Promise<void>; deletePermanently(id: string): Promise<void>; emptyTrash(): Promise<void> }) {
  const bankAccounts=data.accounts.filter(account=>account.type!=='credit_card')
  const [showImport, setShowImport] = useState(false), [accountId, setAccountId] = useState(bankAccounts[0]?.id ?? ''), [bank, setBank] = useState<Bank>('itau'), [file, setFile] = useState<File | null>(null), [importing, setImporting] = useState(false), [message, setMessage] = useState(''), [importError, setImportError] = useState('')
  const [section, setSection] = useState<'active'|'trash'>('active'), [trashed, setTrashed] = useState<Transaction[]>([]), [trashLoading, setTrashLoading] = useState(false), [trashError, setTrashError] = useState(''), [busyTransactionId, setBusyTransactionId] = useState('')
  const [deleteTarget, setDeleteTarget] = useState<{ transaction?: Transaction; trigger: HTMLElement } | null>(null)
  const fileInput = useRef<HTMLInputElement>(null)
  const trashTab = useRef<HTMLButtonElement>(null)
  const showTrash = async () => {
    setSection('trash'); setTrashLoading(true); setTrashError('')
    try { setTrashed(await listTrashed()) } catch (err) { setTrashError(errorMessage(err)) } finally { setTrashLoading(false) }
  }
  const restore = async (transaction: Transaction) => {
    setBusyTransactionId(transaction.id); setTrashError('')
    try { await restoreTrashed(transaction.id); setTrashed(items => items.filter(item => item.id !== transaction.id)) } catch (err) { setTrashError(errorMessage(err)) } finally { setBusyTransactionId('') }
  }
  const closeDelete = () => {
    const trigger = deleteTarget?.trigger
    setDeleteTarget(null)
    requestAnimationFrame(() => (trigger?.isConnected ? trigger : trashTab.current)?.focus())
  }
  const submit = async (event: React.FormEvent) => {
    event.preventDefault()
    setImportError('')
    setMessage('')
    if (!file) { setImportError('Escolha um arquivo PDF, OFX ou CSV.'); fileInput.current?.focus(); return }
    if (file.size > 15 * 1024 * 1024) { setImportError('O extrato deve ter no máximo 15 MB.'); return }
    const extension = file.name.toLowerCase().match(/\.[^.]+$/)?.[0]
    if (!extension || !['.pdf', '.ofx', '.csv'].includes(extension)) { setImportError('Escolha um arquivo PDF, OFX ou CSV.'); return }
    setImporting(true)
    try {
      const result = await onImport({ accountId, bank, fileName: file.name, base64Data: await fileAsBase64(file) })
      const parts = [`${result.importedCount} ${result.importedCount === 1 ? 'movimentação adicionada' : 'movimentações adicionadas'}`]
      if (result.duplicateCount) parts.push(`${result.duplicateCount} ${result.duplicateCount === 1 ? 'repetida ignorada' : 'repetidas ignoradas'}`)
      if (result.ignoredCount) parts.push(`${result.ignoredCount} com data inválida ou futura`)
      setMessage(parts.join(' · '))
      setFile(null)
      if (fileInput.current) fileInput.current.value = ''
    } catch (err) {
      setImportError(errorMessage(err))
    } finally {
      setImporting(false)
    }
  }
  return <div className="page transactions-page">
    <div className="transaction-tabs" role="tablist" aria-label="Movimentações"><button type="button" role="tab" aria-selected={section === 'active'} aria-controls="active-transactions" className={section === 'active' ? 'active' : ''} onClick={() => setSection('active')}>Ativas</button><button ref={trashTab} type="button" role="tab" aria-selected={section === 'trash'} aria-controls="trashed-transactions" className={section === 'trash' ? 'active' : ''} onClick={() => void showTrash()}><Icon name="trash"/> Lixeira</button></div>
    {section === 'active' ? <div id="active-transactions" role="tabpanel">
      <section className="statement-toolbar"><div><strong>Extratos bancários</strong><span>Importe arquivos PDF, OFX ou CSV; registros já importados não serão duplicados.</span></div><Button kind="secondary" onClick={() => { setShowImport(value => !value); setMessage(''); setImportError('') }}>{showImport ? 'Fechar importação' : 'Importar extrato'}</Button></section>
      {showImport && <section className="card statement-import" aria-labelledby="statement-import-title"><div className="card__heading"><div><h2 id="statement-import-title">Importar extrato</h2><p>PDFs do Itaú, Bradesco e Inter, além de arquivos OFX e CSV com colunas reconhecidas automaticamente.</p></div></div><form onSubmit={submit}><label className="field"><span>Conta</span><select value={accountId} onChange={event => setAccountId(event.target.value)} required>{bankAccounts.map(account => <option value={account.id} key={account.id}>{account.name}</option>)}</select></label><label className="field"><span>Banco</span><select value={bank} onChange={event => setBank(event.target.value as Bank)}><option value="itau">Itaú</option><option value="bradesco">Bradesco</option><option value="inter">Inter</option></select></label><label className="field statement-file" htmlFor="statement-file"><span>Arquivo do extrato</span><input id="statement-file" ref={fileInput} type="file" accept="application/pdf,application/x-ofx,application/ofx,application/vnd.intu.qfx,text/csv,text/comma-separated-values,.pdf,.ofx,.csv" onChange={event => setFile(event.target.files?.[0] ?? null)}/><small>PDF, OFX ou CSV, até 15 MB. O arquivo é processado somente neste dispositivo.</small></label><Button type="submit" disabled={!accountId}>Importar movimentações</Button></form>{importError && <p className="statement-result statement-result--error" role="alert">{importError}</p>}{message && <p className="statement-result" role="status">{message}</p>}</section>}
      <section className="card"><TransactionList transactions={transactions} onEdit={onEdit} onRemove={onRemove} empty={<EmptyState icon="receipt" title="Nenhuma movimentação" action={data.accounts.length ? <Button onClick={open}>Registrar agora</Button> : undefined}>Receitas, despesas e transferências aparecerão aqui em ordem de data.</EmptyState>}/></section>
      {importing && <div className="statement-loading" role="status" aria-live="assertive" aria-label="Importando extrato"><span className="statement-loading__spinner" aria-hidden="true"/><strong>Importando seu extrato…</strong><p>Estamos lendo e organizando as movimentações. Não feche o aplicativo.</p></div>}
    </div> : <div id="trashed-transactions" role="tabpanel" className="trash-panel">
      <div className="trash-panel__heading"><div><h2>Lixeira</h2><p>Movimentações removidas continuam aqui até você restaurá-las ou excluí-las permanentemente.</p></div>{trashed.length > 0 && <button type="button" className="button button--danger" onClick={event => setDeleteTarget({ trigger: event.currentTarget })}>Esvaziar lixeira</button>}</div>
      {trashError && <div className="alert" role="alert">{trashError}<button onClick={() => setTrashError('')} aria-label="Fechar aviso"><Icon name="close"/></button></div>}
      <section className="card">{trashLoading ? <p className="trash-loading" role="status">Carregando lixeira…</p> : <TransactionList label="Movimentações na lixeira" transactions={trashed} busyTransactionId={busyTransactionId} onRestore={transaction => void restore(transaction)} onDeletePermanently={(transaction, trigger) => setDeleteTarget({ transaction, trigger })} empty={<EmptyState icon="trash" title="A lixeira está vazia">As movimentações removidas aparecerão aqui e poderão ser recuperadas.</EmptyState>}/>}</section>
    </div>}
    {deleteTarget && <PermanentDeleteDialog
      title={deleteTarget.transaction ? `Excluir “${deleteTarget.transaction.description}” permanentemente?` : `Esvaziar a lixeira com ${trashed.length} ${trashed.length === 1 ? 'movimentação' : 'movimentações'}?`}
      description={deleteTarget.transaction ? 'A movimentação e todo o seu histórico de alterações serão apagados. Esta ação não poderá ser desfeita.' : 'Todas as movimentações e seus históricos de alterações serão apagados. Esta ação não poderá ser desfeita.'}
      confirmLabel={deleteTarget.transaction ? 'Excluir permanentemente' : 'Esvaziar lixeira'}
      onClose={closeDelete}
      onConfirm={async () => { if (deleteTarget.transaction) { await deletePermanently(deleteTarget.transaction.id); setTrashed(items => items.filter(item => item.id !== deleteTarget.transaction?.id)) } else { await emptyTrash(); setTrashed([]) } }}
    />}
  </div>
}

function PermanentDeleteDialog({ title, description, confirmLabel, onClose, onConfirm }: { title: string; description: string; confirmLabel: string; onClose(): void; onConfirm(): Promise<void> }) {
  const [busy, setBusy] = useState(false), [dialogError, setDialogError] = useState('')
  const dialog = useRef<HTMLElement>(null), cancel = useRef<HTMLButtonElement>(null), id = useId()
  useEffect(() => { cancel.current?.focus() }, [])
  const confirm = async () => { setBusy(true); setDialogError(''); try { await onConfirm(); onClose() } catch (err) { setDialogError(errorMessage(err)); setBusy(false) } }
  const keepFocus = (event: React.KeyboardEvent) => { if (event.key === 'Escape' && !busy) { event.preventDefault(); onClose(); return } if (event.key !== 'Tab' || !dialog.current) return; const items = Array.from(dialog.current.querySelectorAll<HTMLButtonElement>('button:not(:disabled)')); const first = items[0], last = items.at(-1); if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last?.focus() } else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first?.focus() } }
  return <div className="dialog-backdrop"><section ref={dialog} className="dialog dialog--confirm" role="alertdialog" aria-modal="true" aria-labelledby={`${id}-title`} aria-describedby={`${id}-description${dialogError ? ` ${id}-error` : ''}`} aria-busy={busy || undefined} onKeyDown={keepFocus}><header><h2 id={`${id}-title`}>{title}</h2></header><p id={`${id}-description`}>{description}</p>{dialogError && <p id={`${id}-error`} className="form-error" role="alert">{dialogError}</p>}<div className="form-actions"><button ref={cancel} type="button" className="button button--ghost" disabled={busy} onClick={onClose}>Cancelar</button><button type="button" className="button button--danger" disabled={busy} aria-busy={busy || undefined} onClick={() => void confirm()}>{busy ? 'Excluindo…' : confirmLabel}</button></div></section></div>
}

function fileAsBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onerror = () => reject(new Error('Não foi possível ler o arquivo.'))
    reader.onload = () => {
      const value = String(reader.result ?? '')
      const separator = value.indexOf(',')
      if (separator < 0) reject(new Error('Não foi possível ler o arquivo.'))
      else resolve(value.slice(separator + 1))
    }
    reader.readAsDataURL(file)
  })
}

function FixedExpensesView({ data, overview, create, update, archive, restore, confirm, dismiss }: { data: BootstrapData; overview: FixedExpensesOverview | null; create(input: FixedExpenseInput): Promise<void>; update(expense: FixedExpense, input: FixedExpenseInput): Promise<void>; archive(id: string): Promise<void>; restore(id: string): Promise<void>; confirm(occurrence: FixedExpenseOccurrence, input: ConfirmFixedExpenseOccurrenceInput): Promise<void>; dismiss(occurrence: FixedExpenseOccurrence): Promise<void> }) {
  const [creating, setCreating] = useState(false), [editing, setEditing] = useState<FixedExpense | undefined>(), [confirming, setConfirming] = useState<FixedExpenseOccurrence | undefined>()
  const confirmTrigger = useRef<HTMLButtonElement | null>(null), addTrigger = useRef<HTMLButtonElement | null>(null)
  if (!overview) return <div className="page"><section className="card"><p className="analysis-empty" aria-live="polite">Carregando despesas fixas…</p></section></div>
  const active = overview.expenses.filter(expense => !expense.archivedAt), archived = overview.expenses.filter(expense => expense.archivedAt), pending = overview.occurrences.filter(occurrence => occurrence.status === 'pending'), completed = overview.occurrences.filter(occurrence => occurrence.status !== 'pending' && occurrence.referenceMonth === today().slice(0, 7))
  const planned = pending.reduce((total, occurrence) => total + occurrence.expectedAmountCents, 0), confirmed = overview.occurrences.filter(occurrence => occurrence.status === 'confirmed' && occurrence.referenceMonth === today().slice(0, 7)).reduce((total, occurrence) => total + occurrence.expectedAmountCents, 0)
  const closeEditor = () => { setCreating(false); setEditing(undefined) }
  const closeConfirmation = () => { setConfirming(undefined); requestAnimationFrame(() => (confirmTrigger.current?.isConnected ? confirmTrigger.current : addTrigger.current)?.focus()) }
  return <div className="page page--split fixed-expenses-page"><section><div className="section-heading">{!creating && !editing && <Button ref={addTrigger} kind="secondary" onClick={() => setCreating(true)}><Icon name="plus"/> Adicionar</Button>}</div><section className="fixed-summary" aria-label="Resumo das despesas fixas"><article><span>Previsões pendentes</span><strong>{formatBRL(planned)}</strong></article><article><span>Confirmadas no mês</span><strong>{formatBRL(confirmed)}</strong></article><article><span>Regras ativas</span><strong>{active.length}</strong></article></section><section className="card fixed-occurrences"><div className="card__heading"><div><h2>O que falta pagar</h2></div></div>{pending.length ? <ul className="fixed-occurrence-list">{pending.map(occurrence => <li key={occurrence.id} className={occurrence.dueDate < today() ? 'overdue' : ''}><div><span className="occurrence-status">{occurrence.dueDate < today() ? 'Atrasada' : `Vence ${formatDate(occurrence.dueDate)}`}</span><strong>{occurrence.description}</strong><small>{occurrence.accountName} · {occurrence.categoryName}</small></div><div><strong>{formatBRL(occurrence.expectedAmountCents)}</strong><span className="occurrence-actions"><button className="text-button" onClick={event => { confirmTrigger.current = event.currentTarget; setConfirming(occurrence) }}><Icon name="check"/> Confirmar</button><button className="text-button muted" onClick={() => void dismiss(occurrence)}>Dispensar</button></span></div></li>)}</ul> : <p className="analysis-empty">Nenhuma despesa fixa pendente agora.</p>}</section>{completed.length > 0 && <section className="card fixed-completed"><div className="card__heading"><div><h2>Já resolvidas</h2></div></div><ul>{completed.map(occurrence => <li key={occurrence.id}><span>{occurrence.description}</span><small>{occurrence.status === 'confirmed' ? 'Confirmada' : 'Dispensada'}</small></li>)}</ul></section>}<section className="card fixed-rules"><div className="card__heading"><div><h2>Seus compromissos</h2></div></div>{active.length ? <ul>{active.map(expense => <li key={expense.id}><div><strong>{expense.description}</strong><small>Dia {expense.dueDay} · {expense.accountName} · {expense.categoryName}</small></div><div><span>{formatBRL(expense.amountCents)}</span><button className="text-button" onClick={() => { setEditing(expense); setCreating(false) }}>Editar</button><button className="text-button muted" onClick={() => void archive(expense.id)}><Icon name="archive"/> Arquivar</button></div></li>)}</ul> : <EmptyState title="Comece pelos compromissos mensais" action={<Button onClick={() => setCreating(true)}>Criar despesa fixa</Button>}>Assinaturas, aluguel e contas entram como previsão até você confirmar o pagamento.</EmptyState>}{archived.length > 0 && <details className="archived-rules"><summary>{archived.length} arquivada{archived.length === 1 ? '' : 's'}</summary><ul>{archived.map(expense => <li key={expense.id}><span>{expense.description}</span><button className="text-button" onClick={() => void restore(expense.id)}>Restaurar</button></li>)}</ul></details>}</section></section>{(creating || editing) && <FixedExpenseForm key={editing?.id ?? 'new'} accounts={data.accounts} categories={data.categories} initial={editing} onSubmit={async input => { if (editing) await update(editing, input); else await create(input); closeEditor() }} onCancel={closeEditor}/>} {confirming && <ConfirmFixedExpenseDialog occurrence={confirming} onClose={closeConfirmation} onSubmit={input => confirm(confirming, input)}/>}</div>
}

function UnlockScreen({ api, onUnlocked }: { api: API; onUnlocked(status: SecurityStatus): Promise<void> }) {
  const [recovery, setRecovery] = useState(false), [password, setPassword] = useState(''), [recoveryKey, setRecoveryKey] = useState(''), [newPassword, setNewPassword] = useState(''), [confirmation, setConfirmation] = useState(''), [busy, setBusy] = useState(false), [error, setError] = useState('')
  const submit = async (event: React.FormEvent) => {
    event.preventDefault(); setBusy(true); setError('')
    try {
      if (recovery) {
        if (!api.RecoverEncryption) throw new Error('Recuperação indisponível nesta instalação.')
        await onUnlocked(await api.RecoverEncryption({ recoveryKey, newPassword, confirmation }))
      } else {
        if (!api.UnlockDatabase) throw new Error('Desbloqueio indisponível nesta instalação.')
        await onUnlocked(await api.UnlockDatabase({ password }))
      }
    } catch (err) { setError(errorMessage(err)); setBusy(false) }
  }
  return <main className="unlock-screen"><section className="unlock-card" aria-labelledby="unlock-title"><div className="brand"><span>[c]</span>ash</div><p className="context-label">Dados protegidos</p><h1 id="unlock-title">Desbloqueie seu banco de dados</h1><p>{recovery ? 'Use a chave guardada por você para definir uma nova senha.' : 'A senha é solicitada em toda abertura. Ela nunca é armazenada.'}</p><form onSubmit={submit}>{recovery ? <><label className="field"><span>Chave de recuperação</span><textarea autoFocus value={recoveryKey} onChange={event => setRecoveryKey(event.target.value)} autoComplete="off" required/></label><label className="field"><span>Nova senha</span><input type="password" value={newPassword} onChange={event => setNewPassword(event.target.value)} minLength={12} autoComplete="new-password" required/></label><label className="field"><span>Confirme a nova senha</span><input type="password" value={confirmation} onChange={event => setConfirmation(event.target.value)} minLength={12} autoComplete="new-password" required/></label></> : <label className="field"><span>Senha</span><input autoFocus type="password" value={password} onChange={event => setPassword(event.target.value)} autoComplete="current-password" required/></label>}{error && <p className="form-error" role="alert">{error}</p>}<Button loading={busy}>{recovery ? 'Redefinir senha e entrar' : 'Desbloquear'}</Button></form><button type="button" className="text-button" disabled={busy} onClick={() => { setRecovery(value => !value); setError('') }}>{recovery ? 'Voltar para a senha' : 'Esqueci minha senha'}</button></section></main>
}

function SettingsView({ api, theme, security, updateStatus, setTheme, onCheckUpdates, onShowUpdate, onRestored, onSecurityChange }: { api: API; theme: Theme; security: SecurityStatus; updateStatus: UpdateStatus | null; setTheme(value: Theme): Promise<void>; onCheckUpdates(): Promise<void>; onShowUpdate(): void; onRestored(): Promise<void>; onSecurityChange(status: SecurityStatus): void }) {
  const checking = updateStatus?.state === 'checking'
  const label = updateStatus?.state === 'disabled' ? 'Atualizações indisponíveis neste build' : updateStatus?.state === 'available' ? `Versão ${updateStatus.availableVersion} disponível` : updateStatus?.state === 'downloading' ? 'Baixando atualização…' : updateStatus?.state === 'installing' ? 'Preparando instalação…' : updateStatus?.state === 'upToDate' ? 'Você já está usando a versão mais recente' : updateStatus?.message || `Versão ${updateStatus?.currentVersion || 'dev'}`
  const [backup, setBackup] = useState<BackupStatus | null>(null), [message, setMessage] = useState(''), [operationError, setOperationError] = useState(''), [busy, setBusy] = useState(''), [securityMode, setSecurityMode] = useState<'enable'|'change'|'recover'|'disable'|null>(null), [password, setPassword] = useState(''), [currentPassword, setCurrentPassword] = useState(''), [confirmation, setConfirmation] = useState(''), [recoveryKey, setRecoveryKey] = useState(''), [archiveRecoveryKey, setArchiveRecoveryKey] = useState(''), [savedRecovery, setSavedRecovery] = useState(false), [restore, setRestore] = useState<BackupInfo | null>(null)
  const settingsDialogTrigger = useRef<HTMLButtonElement | null>(null)
  const refreshBackup = useCallback(async () => { if (api.BackupStatus) setBackup(await api.BackupStatus()) }, [api])
  useEffect(() => { void refreshBackup().catch(err => setOperationError(errorMessage(err))) }, [refreshBackup])
  const run = async (name: string, action: () => Promise<void>) => { setBusy(name); setMessage(''); setOperationError(''); try { await action() } catch (err) { setOperationError(errorMessage(err)) } finally { setBusy('') } }
  const clearSecurityDialog = () => { setSecurityMode(null); setPassword(''); setCurrentPassword(''); setConfirmation(''); if (!savedRecovery) setRecoveryKey(''); requestAnimationFrame(() => settingsDialogTrigger.current?.focus()) }
  const openSecurity = (mode: NonNullable<typeof securityMode>, trigger: HTMLButtonElement) => { settingsDialogTrigger.current = trigger; setSecurityMode(mode) }
  const submitSecurity = async (event: React.FormEvent) => { event.preventDefault(); await run('security', async () => {
    if (securityMode === 'enable') { if (!api.EnableEncryption) return; const result = await api.EnableEncryption({ password, confirmation }); onSecurityChange(result.status); setRecoveryKey(result.recoveryKey ?? ''); setSavedRecovery(false); setMessage('Criptografia ativada.') }
    if (securityMode === 'change') { await api.ChangeEncryptionPassword?.({ currentPassword, newPassword: password, confirmation }); setMessage('Senha alterada.') ; clearSecurityDialog() }
    if (securityMode === 'recover') { if (!api.RecoverEncryption) return; onSecurityChange(await api.RecoverEncryption({ recoveryKey, newPassword: password, confirmation })); setMessage('Senha redefinida com a chave de recuperação.'); setSavedRecovery(true); clearSecurityDialog() }
    if (securityMode === 'disable') { if (!api.DisableEncryption) return; onSecurityChange(await api.DisableEncryption({ password })); setMessage('Criptografia desativada. O banco voltou a ser armazenado sem proteção.'); clearSecurityDialog() }
  }) }
  const inspect = (trigger: HTMLButtonElement) => { settingsDialogTrigger.current = trigger; return run('restore', async () => { const result = await api.InspectBackup?.(); if (result && !result.cancelled) setRestore(result.backup) }) }
  const restoreNow = () => restore && run('restore-confirm', async () => {
    const result = await api.RestoreBackup?.({ path: restore.path, password, recoveryKey: archiveRecoveryKey })
    if (!result?.success) return
    setRestore(null)
    setPassword('')
    setArchiveRecoveryKey('')
    setMessage('Backup restaurado com sucesso.')
    if (api.SecurityStatus) onSecurityChange(await api.SecurityStatus())
    await refreshBackup()
    await onRestored()
  })
  return <div className="page settings-page">
    {operationError && !securityMode && !restore && <p className="alert" role="alert">{operationError}</p>}{message && <p className="alert alert--success" role="status">{message}</p>}
    <section className="card settings-card"><h2>Escolha a atmosfera</h2><p>O conteúdo e a posição dos controles permanecem iguais em todos os temas.</p><fieldset className="settings-themes"><legend className="sr-only">Tema</legend>{(['light', 'dark', 'gothic'] as Theme[]).map(value => <label key={value}><input type="radio" name="settings-theme" value={value} checked={theme === value} onChange={() => void setTheme(value)}/><span className={`settings-swatch settings-swatch--${value}`} aria-hidden="true"><i/><i/></span><strong>{value === 'light' ? 'Claro' : value === 'dark' ? 'Escuro' : 'Gótico'}</strong><small>{value === 'light' ? 'Calmo e luminoso' : value === 'dark' ? 'Confortável à noite' : 'Carvão e vinho'}</small></label>)}</fieldset></section>
    <section className="card portability-card"><div className="settings-section-heading"><div><p className="context-label">Cópias locais</p><h2>Backup e portabilidade</h2><p>O backup automático é semanal e mantém as 12 versões automáticas mais recentes.</p></div><span className={`status-pill ${backup?.lastError ? 'status-pill--error' : ''}`}>{backup?.lastError ? 'Atenção necessária' : 'Automático'}</span></div><dl className="settings-details"><div><dt>Pasta</dt><dd title={backup?.folder}>{backup?.folder ?? 'Carregando…'}</dd></div><div><dt>Último automático</dt><dd>{backup?.lastAutomaticAt ? formatDateTime(backup.lastAutomaticAt) : 'Ainda não realizado'}</dd></div><div><dt>Próximo</dt><dd>{backup?.automaticDue ? 'Será criado na próxima abertura' : backup?.nextDueAt ? formatDateTime(backup.nextDueAt) : 'Após sete dias'}</dd></div></dl>{backup?.lastError && <p className="form-error" role="status">Última falha: {backup.lastError}</p>}<div className="settings-actions"><Button loading={busy === 'backup'} onClick={() => void run('backup', async () => { const result = await api.CreateBackup?.(); if (result && !result.cancelled) setMessage('Backup salvo com sucesso.'); await refreshBackup() })}>Criar backup agora</Button><Button kind="secondary" loading={busy === 'restore'} onClick={event => void inspect(event.currentTarget)}>Restaurar backup</Button><Button kind="ghost" onClick={() => void run('folder', async () => { const result = await api.ChooseBackupFolder?.(); if (result && !result.cancelled) await refreshBackup() })}>Escolher pasta</Button>{backup && backup.folder !== backup.defaultFolder && <button className="text-button" onClick={() => void run('folder', async () => { if (api.ResetBackupFolder) setBackup(await api.ResetBackupFolder()) })}>Usar pasta padrão</button>}</div><div className="export-panel"><div><strong>Exportar relatórios</strong><p>CSV e JSON são arquivos em texto puro, mesmo quando a criptografia está ativada. Guarde-os em local seguro; eles não podem ser usados para restauração.</p></div><div><Button kind="secondary" loading={busy === 'csv'} onClick={() => void run('csv', async () => { const result = await api.ExportData?.('csv'); if (result && !result.cancelled) setMessage('CSV exportado com sucesso.') })}>Exportar CSV</Button><Button kind="secondary" loading={busy === 'json'} onClick={() => void run('json', async () => { const result = await api.ExportData?.('json'); if (result && !result.cancelled) setMessage('JSON exportado com sucesso.') })}>Exportar JSON</Button></div></div></section>
    <section className="card security-card"><div className="settings-section-heading"><div><p className="context-label">Proteção local</p><h2>Criptografia</h2><p>{security.enabled ? 'O banco completo está protegido. A senha será exigida em toda abertura.' : 'Desativada. Seus dados continuam locais, mas o arquivo do banco pode ser lido sem senha.'}</p></div><span className={`status-pill ${security.enabled ? 'status-pill--secure' : ''}`}>{security.enabled ? 'Ativada' : 'Desativada'}</span></div><div className="settings-actions">{security.enabled ? <><Button kind="secondary" onClick={event => openSecurity('change', event.currentTarget)}>Alterar senha</Button><Button kind="secondary" onClick={event => openSecurity('recover', event.currentTarget)}>Redefinir com chave de recuperação</Button><Button kind="ghost" onClick={event => openSecurity('disable', event.currentTarget)}>Desativar criptografia</Button></> : <Button onClick={event => openSecurity('enable', event.currentTarget)}>Ativar criptografia</Button>}</div></section>
    <section className="card update-card"><div><h2>Atualizações</h2><p>{label}</p>{updateStatus?.lastCheckedAt && <small>Última verificação: {formatDateTime(updateStatus.lastCheckedAt)}</small>}</div><div className="update-card__actions">{updateStatus?.state === 'available' && <Button onClick={onShowUpdate}>Atualizar agora</Button>}<Button kind="secondary" loading={checking} onClick={() => void onCheckUpdates()} disabled={updateStatus?.state === 'disabled'}>{checking ? 'Verificando…' : 'Verificar agora'}</Button></div></section>
    <section className="card licenses-card"><details><summary>Licenças de código aberto</summary><h3>SQLCipher — BSD 3-Clause</h3><pre>{`Copyright (c) 2025, ZETETIC LLC
All rights reserved.

Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:

* Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
* Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.
* Neither the name of the ZETETIC LLC nor the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY ZETETIC LLC ''AS IS'' AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL ZETETIC LLC BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.`}</pre></details></section>
    {securityMode && !(securityMode === 'enable' && recoveryKey) && <div className="dialog-backdrop"><section className="dialog security-dialog" role={securityMode === 'disable' ? 'alertdialog' : 'dialog'} aria-modal="true" aria-labelledby="security-dialog-title"><header><div><p className="context-label">{securityMode === 'enable' ? 'Proteção do banco' : securityMode === 'change' ? 'Nova credencial' : securityMode === 'recover' ? 'Recuperação local' : 'Ação destrutiva'}</p><h2 id="security-dialog-title">{securityMode === 'enable' ? 'Ativar criptografia' : securityMode === 'change' ? 'Alterar senha' : securityMode === 'recover' ? 'Redefinir senha' : 'Desativar criptografia?'}</h2></div><button className="icon-button" aria-label="Fechar" disabled={busy === 'security'} onClick={clearSecurityDialog}><Icon name="close"/></button></header><p>{securityMode === 'disable' ? 'O banco será convertido para texto não criptografado. Um backup de segurança protegido será criado antes.' : 'Use pelo menos 12 caracteres. Não existe recuperação remota nem chave mestra.'}</p>{operationError && <p className="form-error" role="alert">{operationError}</p>}<form onSubmit={submitSecurity}>{securityMode === 'change' && <label className="field"><span>Senha atual</span><input autoFocus type="password" value={currentPassword} onChange={event => setCurrentPassword(event.target.value)} required/></label>}{securityMode === 'recover' && <label className="field"><span>Chave de recuperação</span><textarea autoFocus value={recoveryKey} onChange={event => setRecoveryKey(event.target.value)} required/></label>}<label className="field"><span>{securityMode === 'disable' ? 'Senha atual' : 'Nova senha'}</span><input autoFocus={securityMode !== 'change' && securityMode !== 'recover'} type="password" value={password} onChange={event => setPassword(event.target.value)} minLength={securityMode === 'disable' ? undefined : 12} required/></label>{securityMode !== 'disable' && <label className="field"><span>Confirme a senha</span><input type="password" value={confirmation} onChange={event => setConfirmation(event.target.value)} minLength={12} required/></label>}<div className="form-actions"><Button type="button" kind="ghost" disabled={busy === 'security'} onClick={clearSecurityDialog}>Cancelar</Button><Button loading={busy === 'security'}>{securityMode === 'disable' ? 'Desativar e converter' : securityMode === 'change' ? 'Alterar senha' : securityMode === 'recover' ? 'Redefinir senha' : 'Criptografar banco'}</Button></div></form></section></div>}
    {securityMode === 'enable' && recoveryKey && <div className="dialog-backdrop"><section className="dialog recovery-dialog" role="dialog" aria-modal="true" aria-labelledby="recovery-title"><p className="context-label">Exibida uma única vez</p><h2 id="recovery-title">Guarde sua chave de recuperação</h2><p>Ela permite redefinir a senha. O [c]ash não guarda uma cópia e não poderá recuperá-la para você.</p><output className="recovery-key">{recoveryKey}</output><label className="confirmation-check"><input type="checkbox" checked={savedRecovery} onChange={event => setSavedRecovery(event.target.checked)}/><span>Salvei a chave em um local seguro</span></label><Button disabled={!savedRecovery} onClick={() => { setRecoveryKey(''); clearSecurityDialog() }}>Concluir</Button></section></div>}
    {restore && <div className="dialog-backdrop"><section className="dialog restore-dialog" role="alertdialog" aria-modal="true" aria-labelledby="restore-title"><p className="context-label">Substituição de dados</p><h2 id="restore-title">Restaurar este backup?</h2><p>Os dados atuais serão substituídos pelos dados de {formatDateTime(restore.manifest.createdAt)}. Antes disso, criaremos um backup de segurança que nunca é apagado automaticamente.</p><dl className="settings-details"><div><dt>Versão do formato</dt><dd>{restore.manifest.formatVersion}</dd></div><div><dt>Criptografia</dt><dd>{restore.manifest.encrypted ? 'Protegido' : 'Sem criptografia'}</dd></div></dl>{restore.manifest.encrypted && <><label className="field"><span>Senha do backup, se diferente da atual</span><input autoFocus type="password" value={password} onChange={event => setPassword(event.target.value)}/></label><label className="field"><span>Ou chave de recuperação</span><textarea value={archiveRecoveryKey} onChange={event => setArchiveRecoveryKey(event.target.value)}/></label></>}{operationError && <p className="form-error" role="alert">{operationError}</p>}<div className="form-actions"><Button type="button" kind="ghost" disabled={busy === 'restore-confirm'} onClick={() => { setRestore(null); setPassword(''); setArchiveRecoveryKey(''); requestAnimationFrame(() => settingsDialogTrigger.current?.focus()) }}>Cancelar</Button><Button loading={busy === 'restore-confirm'} onClick={() => void restoreNow()}>Restaurar e substituir</Button></div></section></div>}
  </div>
}

function UpdateDialog({ status, onClose, onInstall }: { status: UpdateStatus; onClose(): void; onInstall(): Promise<void> }) {
  const [installing, setInstalling] = useState(false)
  const busy = installing || status.state === 'downloading' || status.state === 'installing'
  const progress = status.totalBytes ? Math.min(100, Math.round(((status.downloadedBytes ?? 0) / status.totalBytes) * 100)) : 0
  const install = async () => { setInstalling(true); try { await onInstall() } catch { setInstalling(false) } }
  const title = status.state === 'error' ? 'Não foi possível atualizar' : `Atualizar para ${status.availableVersion}`
  return <div className="dialog-backdrop" role="presentation"><section className="dialog update-dialog" role="dialog" aria-modal="true" aria-labelledby="update-title"><div className="dialog__heading"><div><p className="context-label">{status.state === 'error' ? 'Atualização interrompida' : 'Nova versão'}</p><h2 id="update-title">{title}</h2></div><button className="icon-button" onClick={onClose} disabled={busy} aria-label="Fechar"><Icon name="close"/></button></div>{status.state === 'downloading' ? <div className="update-dialog__progress" role="status"><span>Baixando atualização… {progress}%</span><progress value={status.downloadedBytes ?? 0} max={status.totalBytes ?? 1}/></div> : status.state === 'installing' ? <p role="status">Encerrando o aplicativo para instalar a atualização…</p> : status.state === 'error' ? <p role="alert">{status.message || 'A versão anterior foi mantida. Tente novamente mais tarde.'}</p> : <><p>O aplicativo será fechado e reaberto quando a instalação terminar.</p>{status.releaseNotes && <pre className="update-dialog__notes">{status.releaseNotes}</pre>}</>}<div className="dialog__actions"><Button kind="secondary" onClick={onClose} disabled={busy}>{status.state === 'error' ? 'Fechar' : 'Depois'}</Button>{!busy && <Button loading={installing} onClick={() => void install()}>{status.state === 'error' ? 'Tentar novamente' : 'Baixar e instalar'}</Button>}</div></section></div>
}

function formatDateTime(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat('pt-BR', { dateStyle: 'short', timeStyle: 'short' }).format(date) }
