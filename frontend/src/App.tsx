import { useCallback, useEffect, useRef, useState } from 'react'
import { api as defaultAPI } from './api'
import { Button, Icon } from './components'
import { Onboarding, TransactionDialog } from './forms'
import { AppShell, PageTransition, Sidebar, Toolbar, type AppView, type NavigationGroup } from './shell'
import { AccountsView, CreditCardsView } from './views/assets'
import { DashboardView } from './views/dashboard'
import { BudgetView, CategoriesView, FixedExpensesView, GoalsView } from './views/planning'
import { SettingsView, UnlockScreen, UpdateDialog } from './views/settings'
import { TransactionsView } from './views/transactions'
import type { AccountInput, BalanceAdjustmentInput, BankStatementInput, BootstrapData, ConfirmFixedExpenseOccurrenceInput, CreditCardsOverview, FixedExpense, FixedExpenseInput, FixedExpenseOccurrence, FixedExpensesOverview, OnboardingInput, SecurityStatus, Theme, Transaction, TransactionInput, TransactionOccurrence, UpdateStatus } from './types'
import { EventsOn } from './wailsjs/runtime/runtime'

type API = typeof defaultAPI

const navigationGroups: NavigationGroup[] = [
  { label: 'Principal', items: [
    { id: 'dashboard', label: 'Visão geral', icon: 'house' },
    { id: 'transactions', label: 'Movimentações', icon: 'list' },
  ] },
  { label: 'Patrimônio', items: [
    { id: 'accounts', label: 'Contas', icon: 'wallet' },
    { id: 'cards', label: 'Cartões e faturas', icon: 'cards' },
  ] },
  { label: 'Planejamento', items: [
    { id: 'fixedExpenses', label: 'Despesas fixas', icon: 'calendarClock' },
    { id: 'budget', label: 'Orçamento', icon: 'budget' },
    { id: 'goals', label: 'Metas', icon: 'target' },
  ] },
  { label: 'Organização', items: [
    { id: 'categories', label: 'Categorias', icon: 'tags' },
  ] },
]

const viewLabels: Record<AppView, string> = Object.fromEntries([
  ...navigationGroups.flatMap(group => group.items.map(item => [item.id, item.label] as const)),
  ['settings', 'Configurações'] as const,
]) as Record<AppView, string>

const systemTheme = (): Theme => window.matchMedia?.('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
const errorMessage = (error: unknown) => error instanceof Error ? error.message : String(error)

async function animateOptimisticRemoval(element: HTMLElement | null) {
  if (!element?.animate || window.matchMedia?.('(prefers-reduced-motion: reduce)').matches) return
  const animation = element.animate([
    { opacity: 1, transform: 'translateX(0) scale(1)' },
    { opacity: 0, transform: 'translateX(16px) scale(.985)' },
  ], { duration: 180, easing: 'cubic-bezier(.2,.8,.2,1)', fill: 'forwards' })
  await animation.finished.catch(() => undefined)
}

export default function App({ api = defaultAPI }: { api?: API }) {
  const [data, setData] = useState<BootstrapData | null>(null)
  const [view, setView] = useState<AppView>('dashboard')
  const [error, setError] = useState('')
  const [transactionDialog, setTransactionDialog] = useState<{ initial?: Transaction } | null>(null)
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [ledgerOccurrences,setLedgerOccurrences]=useState<TransactionOccurrence[]>([])
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
    if (view === 'transactions' && data?.setup) { api.ListTransactions().then(setTransactions).catch(err => setError(errorMessage(err)));api.TransactionOccurrences().then(setLedgerOccurrences).catch(err=>setError(errorMessage(err))) }
    if (view === 'fixedExpenses' && data?.setup) void loadFixedExpenses()
    if (view === 'cards' && data?.setup) void loadCreditCards()
  }, [api, data?.setup, loadCreditCards, loadFixedExpenses, view])
  useEffect(() => { if (data?.setup) window.scrollTo(0, 0) }, [data?.setup, view])
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
    const row = trigger.closest<HTMLElement>('li')
    const focusTarget = row?.nextElementSibling?.querySelector<HTMLElement>('.icon-button')
      ?? row?.previousElementSibling?.querySelector<HTMLElement>('.icon-button')
      ?? document.querySelector<HTMLElement>('.topbar .button')
    const before = data, beforeList = transactions
    await animateOptimisticRemoval(row)
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

  const balancesHidden = data.profile?.balancesHidden ?? false
  const title = view === 'dashboard' ? 'Visão geral' : viewLabels[view]
  const eyebrow = view === 'dashboard'
    ? (data.profile?.displayName ? `Olá, ${data.profile.displayName}` : 'Seu mês, agora')
    : undefined
  return <>
    <AppShell
      collapsed={sidebarCollapsed}
      sidebar={<Sidebar groups={navigationGroups} active={view} collapsed={sidebarCollapsed} displayName={data.profile?.displayName} onNavigate={setView} onCollapsedChange={setSidebarCollapsed}/>}
      toolbar={<Toolbar title={title} eyebrow={eyebrow} transactionDisabled={!data.accounts.length} transactionHint={!data.accounts.length ? 'Crie uma conta primeiro' : 'Atalho: Ctrl+N'} onTransaction={() => openTransaction()}/>}
    >
      {error && <div className="alert" role="alert">{error}<button onClick={() => setError('')} aria-label="Fechar aviso"><Icon name="close"/></button></div>}
      {data.setup && updateStatus?.state === 'available' && !updateDismissed && <section className="update-banner" role="status" aria-label="Atualização disponível"><div><strong>Uma atualização está disponível</strong><span>Versão {updateStatus.availableVersion}</span></div><div><Button kind="secondary" onClick={() => setUpdateDialog(true)}>Ver novidades</Button><button className="text-button muted" onClick={() => setUpdateDismissed(true)}>Depois</button></div></section>}
      <PageTransition key={view} view={view}>
        {view === 'dashboard' && <DashboardView data={data} balancesHidden={balancesHidden} onBalancesHiddenChange={setBalancesHidden} onAccounts={() => setView('accounts')} onTransactions={() => setView('transactions')} onTransaction={() => openTransaction()} onEdit={openTransaction} onRemove={removeTransaction}/>}
        {view === 'accounts' && <AccountsView data={data} create={createAccount} update={updateAccount} adjust={adjustAccount} remove={deleteAccount}/>}
        {view === 'cards' && <CreditCardsView data={data} overview={creditOverview} onPay={async(id,input)=>{await api.PayCreditCardInvoice(id,input);await load();await loadCreditCards()}} onCreateCard={()=>setView('accounts')}/>}
        {view === 'transactions' && <TransactionsView
          data={data} transactions={transactions} occurrences={ledgerOccurrences}
          search={filter=>api.SearchTransactions(filter)}
          confirmOccurrence={async id=>{await api.ConfirmTransactionOccurrence(id);setLedgerOccurrences(await api.TransactionOccurrences());await refreshTransactions()}}
          dismissOccurrence={async id=>{await api.DismissTransactionOccurrence(id);setLedgerOccurrences(await api.TransactionOccurrences())}}
          open={() => openTransaction()} onImport={importBankStatement} onEdit={openTransaction} onRemove={removeTransaction}
          listTrashed={() => api.ListTrashedTransactions()} restoreTrashed={async id => { await api.RestoreTransaction(id); await refreshTransactions() }}
          deletePermanently={id => api.DeleteTransactionPermanently(id)} emptyTrash={() => api.EmptyTransactionTrash()}/>} {/* movements */}
        {view === 'fixedExpenses' && <FixedExpensesView data={data} overview={fixedOverview} create={input => saveFixedExpense(input)} update={(expense, input) => saveFixedExpense(input, expense)} archive={async id => { await api.ArchiveFixedExpense(id); await refreshFixedExpenses() }} restore={async id => { await api.RestoreFixedExpense(id); await refreshFixedExpenses() }} confirm={confirmFixedOccurrence} dismiss={dismissFixedOccurrence}/>}
        {view === 'categories' && <CategoriesView categories={data.categories} create={async input=>{await api.CreateCategory(input);await load()}} rename={async(id,input)=>{await api.RenameCategory(id,input);await load()}} archive={async id=>{await api.ArchiveCategory(id);await load()}} restore={async id=>{await api.RestoreCategory(id);await load()}}/>}
        {view === 'budget' && <BudgetView data={data} save={async input=>{await api.SetMonthlyBudget(input);await load()}}/>}
        {view === 'goals' && <GoalsView data={data} save={async(id,input)=>{await api.SaveGoal(id,input);await load()}} allocate={async(id,input)=>{await api.SetGoalAllocations(id,input);await load()}} archive={async id=>{await api.ArchiveGoal(id);await load()}}/>}
        {view === 'settings' && <SettingsView api={api} theme={theme} security={security ?? { enabled: false, locked: false }} updateStatus={updateStatus} setTheme={async value => { try { await api.SetTheme(value); await load() } catch (err) { setError(errorMessage(err)) } }} onCheckUpdates={checkForUpdates} onShowUpdate={() => setUpdateDialog(true)} onRestored={async () => { setView('dashboard'); await load() }} onSecurityChange={setSecurity}/>}
      </PageTransition>
    </AppShell>
    {transactionDialog && <TransactionDialog accounts={data.accounts} categories={data.categories} initial={transactionDialog.initial} onClose={closeTransaction} onSubmit={saveTransaction}/>} 
    {transactionSnackbar && <div className="snackbar" role="status"><span>Movimentação removida</span><button onClick={() => void undoRemoval()}>Desfazer</button></div>}
    {dismissedOccurrence && <div className="snackbar" role="status"><span>Previsão dispensada</span><button onClick={() => void undoDismissFixedOccurrence()}>Desfazer</button></div>}
    {updateDialog && updateStatus && ['available', 'downloading', 'installing', 'error'].includes(updateStatus.state) && <UpdateDialog status={updateStatus} onClose={() => setUpdateDialog(false)} onInstall={installUpdate}/>} 
  </>
}
