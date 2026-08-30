import { useEffect, useId, useRef, useState, type FormEvent, type KeyboardEvent } from 'react'
import { Button, EmptyState, Icon, Modal, TransactionList } from '../components'
import { formatBRL, formatDate, parseBRL } from '../format'
import type { Bank, BankStatementImportResult, BankStatementInput, BootstrapData, Transaction, TransactionFilter, TransactionOccurrence } from '../types'

const errorMessage = (error: unknown) => error instanceof Error ? error.message : String(error)

const transactionFilterMeta: Partial<Record<keyof TransactionFilter, { label: string; control: string }>> = {
  text: { label: 'Busca', control: 'text' }, startDate: { label: 'De', control: 'startDate' }, endDate: { label: 'Até', control: 'endDate' },
  accountId: { label: 'Conta', control: 'accountId' }, categoryId: { label: 'Categoria', control: 'categoryId' }, tag: { label: 'Tag', control: 'tag' },
  kind: { label: 'Tipo', control: 'kind' }, status: { label: 'Status', control: 'status' }, minimumAmountCents: { label: 'Valor mínimo', control: 'minimum' },
  maximumAmountCents: { label: 'Valor máximo', control: 'maximum' }, recurrence: { label: 'Recorrência', control: 'recurrence' },
}

function transactionFilterValue(key: keyof TransactionFilter, value: unknown, data: BootstrapData) {
  if (key === 'accountId') return data.accounts.find(account => account.id === value)?.name ?? String(value)
  if (key === 'categoryId') return data.categories.find(category => category.id === value)?.name ?? String(value)
  if (key === 'minimumAmountCents' || key === 'maximumAmountCents') return formatBRL(Number(value))
  if (key === 'startDate' || key === 'endDate') return formatDate(String(value))
  if (key === 'kind') return value === 'income' ? 'Receita' : value === 'expense' ? 'Despesa' : value === 'transfer' ? 'Transferência' : String(value)
  if (key === 'status') return value === 'all' ? 'Todos' : value === 'trashed' ? 'Lixeira' : value === 'pending' ? 'Pendentes' : 'Realizadas'
  if (key === 'recurrence') return value === 'recurring' ? 'Recorrentes' : 'Não recorrentes'
  return String(value)
}

export function TransactionsView({ data, transactions, occurrences, search, confirmOccurrence, dismissOccurrence, open, onImport, onEdit, onRemove, listTrashed, restoreTrashed, deletePermanently, emptyTrash }: { data: BootstrapData; transactions: Transaction[]; occurrences:TransactionOccurrence[];search(filter:TransactionFilter):Promise<Transaction[]>;confirmOccurrence(id:string):Promise<void>;dismissOccurrence(id:string):Promise<void>; open(): void; onImport(input: BankStatementInput): Promise<BankStatementImportResult>; onEdit(tx: Transaction, trigger: HTMLElement): void; onRemove(tx: Transaction, trigger: HTMLElement): void; listTrashed(): Promise<Transaction[]>; restoreTrashed(id: string): Promise<void>; deletePermanently(id: string): Promise<void>; emptyTrash(): Promise<void> }) {
  const bankAccounts=data.accounts.filter(account=>account.type!=='credit_card')
  const [showImport, setShowImport] = useState(false), [accountId, setAccountId] = useState(bankAccounts[0]?.id ?? ''), [bank, setBank] = useState<Bank>('itau'), [file, setFile] = useState<File | null>(null), [importing, setImporting] = useState(false), [message, setMessage] = useState(''), [importError, setImportError] = useState('')
  const [section, setSection] = useState<'active'|'trash'>('active'), [trashed, setTrashed] = useState<Transaction[]>([]), [trashLoading, setTrashLoading] = useState(false), [trashError, setTrashError] = useState(''), [busyTransactionId, setBusyTransactionId] = useState('')
  const [deleteTarget, setDeleteTarget] = useState<{ transaction?: Transaction; trigger: HTMLElement } | null>(null)
  const [results,setResults]=useState<Transaction[]>(transactions),[filters,setFilters]=useState<TransactionFilter>({status:'active'}),[searching,setSearching]=useState(false)
  useEffect(()=>setResults(transactions),[transactions])
  const runSearch=async(next:TransactionFilter)=>{setFilters(next);setSearching(true);try{setResults(await search(next))}catch(err){setTrashError(errorMessage(err))}finally{setSearching(false)}}
  const fileInput = useRef<HTMLInputElement>(null), filterForm = useRef<HTMLFormElement>(null)
  const activeTab = useRef<HTMLButtonElement>(null), trashTab = useRef<HTMLButtonElement>(null)
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
  const submit = async (event: FormEvent) => {
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
  const activeFilters = (Object.entries(filters) as [keyof TransactionFilter, unknown][]).filter(([key, value]) => transactionFilterMeta[key] && value !== '' && value !== 0 && value !== undefined && !(key === 'status' && value === 'active'))
  const clearFilter = (key: keyof TransactionFilter) => {
    const controlName = transactionFilterMeta[key]?.control
    const control = controlName ? filterForm.current?.elements.namedItem(controlName) : null
    if (control instanceof HTMLInputElement || control instanceof HTMLSelectElement) control.value = key === 'status' ? 'active' : ''
    const next = { ...filters }
    delete next[key]
    if (key === 'status') next.status = 'active'
    void runSearch(next)
  }
  const changeTab = (next: 'active' | 'trash') => {
    if (next === 'active') setSection('active')
    else void showTrash()
  }
  const tabKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return
    event.preventDefault()
    const tabs = [activeTab.current, trashTab.current].filter((tab): tab is HTMLButtonElement => Boolean(tab))
    const current = tabs.indexOf(document.activeElement as HTMLButtonElement)
    const nextIndex = event.key === 'Home' ? 0 : event.key === 'End' ? tabs.length - 1 : (current + (event.key === 'ArrowRight' ? 1 : -1) + tabs.length) % tabs.length
    const next = tabs[nextIndex]
    next?.focus()
    changeTab(next === trashTab.current ? 'trash' : 'active')
  }
  return <div className="page transactions-page">
    <div className="transaction-tabs" role="tablist" aria-label="Movimentações" onKeyDown={tabKeyDown}><button ref={activeTab} id="active-transactions-tab" type="button" role="tab" tabIndex={section === 'active' ? 0 : -1} aria-selected={section === 'active'} aria-controls="active-transactions" className={section === 'active' ? 'active' : ''} onClick={() => changeTab('active')}>Ativas</button><button ref={trashTab} id="trashed-transactions-tab" type="button" role="tab" tabIndex={section === 'trash' ? 0 : -1} aria-selected={section === 'trash'} aria-controls="trashed-transactions" className={section === 'trash' ? 'active' : ''} onClick={() => changeTab('trash')}><Icon name="trash"/> Lixeira</button></div>
    {section === 'active' ? <div id="active-transactions" role="tabpanel" aria-labelledby="active-transactions-tab" tabIndex={0}>
      <form ref={filterForm} className="card transaction-filters" aria-label="Filtros de movimentações" aria-busy={searching||undefined} onSubmit={event=>{event.preventDefault();const form=new FormData(event.currentTarget),minimum=parseBRL(String(form.get('minimum')??''))??0,maximum=parseBRL(String(form.get('maximum')??''))??0;void runSearch({text:String(form.get('text')??''),startDate:String(form.get('startDate')??''),endDate:String(form.get('endDate')??''),accountId:String(form.get('accountId')??''),categoryId:String(form.get('categoryId')??''),tag:String(form.get('tag')??''),kind:String(form.get('kind')??'') as TransactionFilter['kind'],status:String(form.get('status')??'active') as TransactionFilter['status'],minimumAmountCents:minimum,maximumAmountCents:maximum,recurrence:String(form.get('recurrence')??'') as TransactionFilter['recurrence']})}}><div className="transaction-filters__primary"><label className="field"><span>Buscar</span><input name="text" placeholder="Descrição, conta ou categoria"/></label><label className="field"><span>Conta</span><select name="accountId"><option value="">Todas</option>{data.accounts.map(a=><option value={a.id} key={a.id}>{a.name}</option>)}</select></label><label className="field"><span>Tipo</span><select name="kind"><option value="">Todos</option><option value="income">Receita</option><option value="expense">Despesa</option><option value="transfer">Transferência</option></select></label><label className="field"><span>Status</span><select name="status" defaultValue="active"><option value="all">Todos</option><option value="active">Realizadas</option><option value="trashed">Lixeira</option><option value="pending">Pendentes</option></select></label></div><details className="transaction-filters__advanced"><summary>Mais filtros</summary><div><label className="field"><span>De</span><input name="startDate" type="date"/></label><label className="field"><span>Até</span><input name="endDate" type="date"/></label><label className="field"><span>Categoria</span><select name="categoryId"><option value="">Todas</option>{data.categories.map(c=><option value={c.id} key={c.id}>{c.name}</option>)}</select></label><label className="field"><span>Tag</span><input name="tag"/></label><label className="field"><span>Valor mínimo</span><div className="money-input"><span>R$</span><input name="minimum" inputMode="decimal"/></div></label><label className="field"><span>Valor máximo</span><div className="money-input"><span>R$</span><input name="maximum" inputMode="decimal"/></div></label><label className="field"><span>Recorrência</span><select name="recurrence"><option value="">Todas</option><option value="recurring">Recorrentes</option><option value="nonrecurring">Não recorrentes</option></select></label></div></details><div className="form-actions"><Button loading={searching}>Aplicar filtros</Button><button type="button" className="text-button" onClick={()=>{filterForm.current?.reset();void runSearch({status:'active'})}}>Limpar todos</button></div></form>
      {activeFilters.length>0&&<div className="filter-chips" aria-label="Filtros ativos">{activeFilters.map(([key,value])=><button type="button" className="filter-chip" key={key} onClick={()=>clearFilter(key)} aria-label={`Remover filtro ${transactionFilterMeta[key]?.label}`}><span><strong>{transactionFilterMeta[key]?.label}:</strong> {transactionFilterValue(key,value,data)}</span><Icon name="close"/></button>)}</div>}
      <section className="statement-toolbar"><div><strong>Extratos bancários</strong><span>Importe arquivos PDF, OFX ou CSV; registros já importados não serão duplicados.</span></div><Button kind="secondary" onClick={() => { setShowImport(value => !value); setMessage(''); setImportError('') }}>{showImport ? 'Fechar importação' : 'Importar extrato'}</Button></section>
      {showImport && <section className="card statement-import" aria-labelledby="statement-import-title"><div className="card__heading"><div><h2 id="statement-import-title">Importar extrato</h2><p>PDFs do Itaú, Bradesco e Inter, além de arquivos OFX e CSV com colunas reconhecidas automaticamente.</p></div></div><form onSubmit={submit}><label className="field"><span>Conta</span><select value={accountId} onChange={event => setAccountId(event.target.value)} required>{bankAccounts.map(account => <option value={account.id} key={account.id}>{account.name}</option>)}</select></label><label className="field"><span>Banco</span><select value={bank} onChange={event => setBank(event.target.value as Bank)}><option value="itau">Itaú</option><option value="bradesco">Bradesco</option><option value="inter">Inter</option></select></label><label className="field statement-file" htmlFor="statement-file"><span>Arquivo do extrato</span><input id="statement-file" ref={fileInput} type="file" accept="application/pdf,application/x-ofx,application/ofx,application/vnd.intu.qfx,text/csv,text/comma-separated-values,.pdf,.ofx,.csv" onChange={event => setFile(event.target.files?.[0] ?? null)}/><small>PDF, OFX ou CSV, até 15 MB. O arquivo é processado somente neste dispositivo.</small></label><Button type="submit" disabled={!accountId}>Importar movimentações</Button></form>{importError && <p className="statement-result statement-result--error" role="alert">{importError}</p>}{message && <p className="statement-result" role="status">{message}</p>}</section>}
      <section className="card"><TransactionList transactions={results} onEdit={onEdit} onRemove={onRemove} empty={<EmptyState icon="receipt" title="Nenhuma movimentação encontrada" action={data.accounts.length ? <Button onClick={open}>Registrar agora</Button> : undefined}>Revise os filtros ou registre uma nova movimentação.</EmptyState>}/></section>
      {occurrences.some(item=>item.status==='pending')&&<section className="card"><h2>Ocorrências pendentes</h2><ul>{occurrences.filter(item=>item.status==='pending').map(item=><li key={item.id}><strong>{item.description}</strong> · {formatDate(item.scheduledDate)} · {formatBRL(item.amountCents)} <Button kind="secondary" onClick={()=>void confirmOccurrence(item.id)}>Confirmar</Button><button className="text-button" onClick={()=>void dismissOccurrence(item.id)}>Dispensar</button></li>)}</ul></section>}
      {importing && <Modal onClose={()=>undefined} busy dismissible={false} labelledBy="statement-loading-title" className="statement-loading"><span className="statement-loading__spinner" aria-hidden="true"/><strong id="statement-loading-title">Importando extrato</strong><p role="status" aria-live="assertive">Estamos lendo e organizando as movimentações. Não feche o aplicativo.</p></Modal>}
    </div> : <div id="trashed-transactions" role="tabpanel" aria-labelledby="trashed-transactions-tab" tabIndex={0} className="trash-panel">
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
  const id = useId()
  const confirm = async () => { setBusy(true); setDialogError(''); try { await onConfirm(); onClose() } catch (err) { setDialogError(errorMessage(err)); setBusy(false) } }
  return <Modal onClose={onClose} busy={busy} labelledBy={`${id}-title`} describedBy={`${id}-description${dialogError ? ` ${id}-error` : ''}`} className="dialog--confirm" alert><header><h2 id={`${id}-title`}>{title}</h2></header><p id={`${id}-description`}>{description}</p>{dialogError && <p id={`${id}-error`} className="form-error" role="alert">{dialogError}</p>}<div className="form-actions"><button data-dialog-initial type="button" className="button button--ghost" disabled={busy} onClick={onClose}>Cancelar</button><button type="button" className="button button--danger" data-state={dialogError?'error':busy?'loading':'idle'} disabled={busy} aria-busy={busy || undefined} onClick={() => void confirm()}>{busy ? 'Excluindo…' : confirmLabel}</button></div></Modal>
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
