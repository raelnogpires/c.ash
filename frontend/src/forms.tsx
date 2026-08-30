import { useEffect, useId, useState, type FormEvent } from 'react'
import { Button, Icon, Modal, SearchCombobox, Sheet } from './components'
import { formatBRL, parseBRL, today } from './format'
import type { Account, AccountInput, BalanceAdjustmentInput, Category, ConfirmFixedExpenseOccurrenceInput, FixedExpense, FixedExpenseInput, FixedExpenseOccurrence, OnboardingInput, Theme, Transaction, TransactionInput, TransactionKind } from './types'

export type SubmitStatus = 'idle' | 'loading' | 'error' | 'success'
type SubmitState = { busy: boolean; error: string; status?: SubmitStatus }
const errorMessage = (error: unknown) => error instanceof Error ? error.message : String(error)
const submitStatus = (state: SubmitState): SubmitStatus => state.busy ? 'loading' : state.error ? 'error' : state.status ?? 'idle'

function FormError({ id, message }: { id: string; message: string }) {
  const active = Boolean(message)
  return <p id={id} className="form-error" role={active ? 'alert' : undefined} aria-live="assertive" aria-hidden={active ? undefined : true}>{message}</p>
}

export function AccountFields({ prefix = '', initial, invalidBalance = false, balanceErrorId }: { prefix?: string; initial?: Account; invalidBalance?: boolean; balanceErrorId?: string }) {
  const id = useId()
  const nameId = `${id}-name`, typeId = `${id}-type`, balanceId = `${id}-balance`, dateId = `${id}-date`
  const [type,setType]=useState<AccountInput['type']>(initial?.type ?? 'checking')
  return <div className="form-grid">
    <label className="field field--wide" htmlFor={nameId}><span>Nome da conta</span><input id={nameId} name={`${prefix}name`} placeholder="Ex.: Conta principal" autoComplete="off" defaultValue={initial?.name ?? ''} required aria-required="true" /></label>
    <fieldset className="account-types field--wide" aria-required="true"><legend id={typeId}>Tipo de conta</legend>
      <label><input type="radio" name={`${prefix}type`} value="checking" checked={type==='checking'} onChange={()=>setType('checking')} required/><span><strong>Conta corrente</strong><small>Para pagamentos e movimentações do dia a dia.</small></span></label>
      <label><input type="radio" name={`${prefix}type`} value="savings" checked={type==='savings'} onChange={()=>setType('savings')}/><span><strong>Poupança</strong><small>Reserva que nunca pode ficar com saldo negativo.</small></span></label>
      <label><input type="radio" name={`${prefix}type`} value="cash" checked={type==='cash'} onChange={()=>setType('cash')}/><span><strong>Dinheiro</strong><small>Valores guardados ou usados fora do banco.</small></span></label>
      <label><input type="radio" name={`${prefix}type`} value="credit_card" checked={type==='credit_card'} onChange={()=>setType('credit_card')}/><span><strong>Cartão de crédito</strong><small>Compras organizadas por fatura e parcelas.</small></span></label>
    </fieldset>
    {type==='credit_card'?<>
      <label className="field" htmlFor={balanceId}><span>Limite de crédito</span><div className="money-input"><span aria-hidden="true">R$</span><input id={balanceId} name={`${prefix}limit`} inputMode="decimal" placeholder="0,00" defaultValue={initial?.creditLimitCents?(initial.creditLimitCents/100).toLocaleString('pt-BR',{minimumFractionDigits:2,maximumFractionDigits:2}):''} required/></div></label>
      <label className="field"><span>Dia do fechamento</span><input name={`${prefix}closingDay`} type="number" min="1" max="31" defaultValue={initial?.closingDay ?? ''} required/></label>
      <label className="field"><span>Dia do vencimento</span><input name={`${prefix}dueDay`} type="number" min="1" max="31" defaultValue={initial?.dueDay ?? ''} required/></label>
      <label className="field" htmlFor={dateId}><span>Começar a acompanhar em</span><input id={dateId} name={`${prefix}date`} type="date" defaultValue={initial?.openingDate ?? today()} max={today()} required/></label>
      {!initial&&<><label className="field"><span>Fatura já em aberto <em>opcional</em></span><div className="money-input"><span aria-hidden="true">R$</span><input name={`${prefix}debt`} inputMode="decimal" placeholder="0,00"/></div></label><label className="field"><span>Vencimento dessa fatura</span><input name={`${prefix}debtDueDate`} type="date"/></label></>}
    </>:<><label className="field" htmlFor={balanceId}><span>Saldo inicial</span><div className="money-input"><span aria-hidden="true">R$</span><input id={balanceId} name={`${prefix}balance`} inputMode="decimal" placeholder="0,00" defaultValue={initial?(initial.openingBalanceCents/100).toLocaleString('pt-BR',{minimumFractionDigits:2,maximumFractionDigits:2}):''} readOnly={initial?.hasLedgerActivity} required aria-required="true" aria-invalid={invalidBalance ? true : undefined} aria-describedby={invalidBalance ? balanceErrorId : undefined} /></div>{initial?.hasLedgerActivity&&<small>O saldo inicial fica bloqueado após a primeira movimentação. Use um ajuste de saldo.</small>}</label>
    <label className="field" htmlFor={dateId}><span>Data do saldo</span><input id={dateId} name={`${prefix}date`} type="date" defaultValue={initial?.openingDate ?? today()} max={today()} required aria-required="true" /></label></>}
  </div>
}

function accountFromForm(data: FormData, prefix = ''): AccountInput | string {
  const type=String(data.get(`${prefix}type`)) as AccountInput['type']
  if(type==='credit_card'){
    const limit=parseBRL(String(data.get(`${prefix}limit`)??'')),debt=parseBRL(String(data.get(`${prefix}debt`)??''))??0,due=String(data.get(`${prefix}debtDueDate`)??'')
    if(limit===null||limit<=0)return 'Informe um limite de crédito maior que zero.'
    if(debt>0&&!due)return 'Informe o vencimento da fatura em aberto.'
    return {name:String(data.get(`${prefix}name`)??'').trim(),type,openingBalanceCents:0,openingDate:String(data.get(`${prefix}date`)),creditLimitCents:limit,closingDay:Number(data.get(`${prefix}closingDay`)),dueDay:Number(data.get(`${prefix}dueDay`)),openingDebtCents:debt,openingDebtDueDate:due}
  }
  const cents = parseBRL(String(data.get(`${prefix}balance`) ?? ''))
  if (cents === null) return 'Informe um saldo válido, com até duas casas decimais.'
  return { name: String(data.get(`${prefix}name`) ?? '').trim(), type, openingBalanceCents: cents, openingDate: String(data.get(`${prefix}date`)) }
}

export function Onboarding({ initialTheme, onComplete, onSkip }: { initialTheme: Theme; onComplete(input: OnboardingInput): Promise<void>; onSkip(): Promise<void> }) {
  const [theme, setTheme] = useState<Theme>(initialTheme)
  const [state, setState] = useState<SubmitState>({ busy: false, error: '' })
  const displayNameId = useId(), errorId = useId()
  useEffect(() => {
    document.documentElement.dataset.theme = theme
    document.documentElement.style.colorScheme = theme === 'light' ? 'light' : 'dark'
  }, [theme])
  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault(); const data = new FormData(event.currentTarget); const account = accountFromForm(data, 'account-')
    if (typeof account === 'string') return setState({ busy: false, error: account })
    const reserveTargetCents=parseBRL(String(data.get('reserveTarget')))
    if(reserveTargetCents===null||reserveTargetCents<0)return setState({busy:false,error:'Informe um alvo de reserva válido.'})
    setState({ busy: true, error: '' })
    try { await onComplete({ displayName: String(data.get('displayName')).trim(), currency: 'BRL', theme, firstAccount: account, reserveTargetCents }); setState({ busy: false, error: '', status: 'success' }) }
    catch (error) { setState({ busy: false, error: errorMessage(error) }) }
  }
  const skip = async () => { setState({ busy: true, error: '' }); try { await onSkip(); setState({ busy: false, error: '', status: 'success' }) } catch (error) { setState({ busy: false, error: errorMessage(error) }) } }
  const status = submitStatus(state)
  return <main className="onboarding">
    <section className="onboarding__intro"><div className="brand brand--large"><span>[c]</span>ash</div><p className="onboarding__kicker">Seu dinheiro, no seu ritmo</p><h1>Clareza começa com um primeiro registro.</h1><p>Organize o que você tem hoje. Seus dados ficam somente neste computador.</p><div className="privacy-note"><span aria-hidden="true"><Icon name="shieldCheck"/></span><span><strong>Privado por princípio</strong><br/>Sem conta online, sem planilha, sem dados fictícios.</span></div></section>
    <section className="onboarding__panel" aria-labelledby="setup-title"><div className="step-label">Configuração inicial · 1 de 1</div><h2 id="setup-title">Vamos preparar seu espaço</h2><p>Informe os dados básicos. Você pode ajustar o tema depois.</p>
      <form onSubmit={submit} aria-busy={state.busy ? true : undefined} data-state={status} aria-describedby={state.error ? errorId : undefined}><label className="field field--wide" htmlFor={displayNameId}><span>Como podemos chamar você?</span><input id={displayNameId} name="displayName" autoFocus required aria-required="true" autoComplete="name" placeholder="Seu primeiro nome" /></label>
        <fieldset className="theme-picker"><legend>Aparência</legend>{(['light','dark','gothic'] as Theme[]).map((item) => <label key={item} className={`theme-choice theme-choice--${item}`}><input type="radio" name="theme" value={item} checked={theme===item} onChange={() => setTheme(item)} /><span className="theme-preview" aria-hidden="true"><i/><i/><i/></span><span>{item==='light'?'Claro':item==='dark'?'Escuro':'Gótico'}</span></label>)}</fieldset>
        <div className="section-rule"><span>Sua primeira conta</span></div><AccountFields prefix="account-" invalidBalance={state.error.includes('saldo válido')} balanceErrorId={errorId} />
        <div className="section-rule"><span>Reserva de emergência</span></div><label className="field field--wide"><span>Quanto você quer reservar?</span><div className="money-input"><span>R$</span><input name="reserveTarget" inputMode="decimal" defaultValue="0,00" placeholder="0,00" required aria-required="true"/></div><small>Um alvo zero não cria a meta e nenhum dinheiro será reservado automaticamente.</small></label>
        <FormError id={errorId} message={state.error} />
        <div className="form-actions"><Button type="button" kind="ghost" disabled={state.busy} loading={state.busy} state={status} onClick={skip}>Fazer isso depois</Button><Button type="submit" disabled={state.busy} loading={state.busy} state={status}>{state.busy ? 'Salvando…' : 'Começar agora'} <Icon name="arrowRight" /></Button></div>
      </form>
    </section>
  </main>
}

export function AccountForm({ initial, onSubmit, onCancel }: { initial?: Account; onSubmit(input: AccountInput): Promise<void>; onCancel?(): void }) {
  const [state, setState] = useState<SubmitState>({busy:false,error:''})
  const errorId = useId(), titleId = useId()
  const submit = async (event: FormEvent<HTMLFormElement>) => { event.preventDefault(); const form=event.currentTarget; const account = accountFromForm(new FormData(form)); if (typeof account === 'string') return setState({busy:false,error:account}); setState({busy:true,error:''}); try { await onSubmit(account); form.reset(); setState({busy:false,error:'',status:'success'}) } catch(error){ setState({busy:false,error:errorMessage(error)}) } }
  const status = submitStatus(state)
  const close = onCancel ?? (() => undefined)
  return <Sheet onClose={close} busy={state.busy} labelledBy={titleId} className="account-form-sheet"><form className="card form-card" onSubmit={submit} aria-busy={state.busy ? true : undefined} data-state={status} aria-describedby={state.error ? errorId : undefined}><div className="card__heading"><div><h2 id={titleId}>{initial?'Revise os dados':'Onde está seu dinheiro?'}</h2></div><button type="button" className="icon-button" aria-label="Fechar" disabled={state.busy} onClick={close}><Icon name="close"/></button></div><AccountFields initial={initial} invalidBalance={state.error.includes('saldo válido')} balanceErrorId={errorId}/><FormError id={errorId} message={state.error}/><div className="form-actions">{onCancel && <Button type="button" kind="ghost" disabled={state.busy} onClick={onCancel}>Cancelar</Button>}<Button disabled={state.busy} loading={state.busy} state={status}>{state.busy?'Salvando…':initial?'Salvar alterações':'Criar conta'}</Button></div></form></Sheet>
}

export function BalanceAdjustmentDialog({ account, onSubmit, onClose }: { account: Account; onSubmit(input: BalanceAdjustmentInput): Promise<void>; onClose(): void }) {
  const [target,setTarget]=useState((account.currentBalanceCents/100).toLocaleString('pt-BR',{minimumFractionDigits:2,maximumFractionDigits:2})),[state,setState]=useState<SubmitState>({busy:false,error:''})
  const targetCents=parseBRL(target),difference=(targetCents??account.currentBalanceCents)-account.currentBalanceCents,id=useId(),errorId=`${id}-error`
  const submit=async(event:FormEvent<HTMLFormElement>)=>{event.preventDefault();const data=new FormData(event.currentTarget);if(targetCents===null)return setState({busy:false,error:'Informe um saldo válido.'});const reason=String(data.get('reason')??'').trim();if(!reason)return setState({busy:false,error:'Informe o motivo do ajuste.'});setState({busy:true,error:''});try{await onSubmit({targetBalanceCents:targetCents,occurrenceDate:String(data.get('date')),reason});onClose()}catch(error){setState({busy:false,error:errorMessage(error)})}}
  return <Sheet onClose={onClose} busy={state.busy} labelledBy={`${id}-title`} className="balance-adjustment-sheet"><header><div><p className="context-label">Registro auditável</p><h2 id={`${id}-title`}>Ajustar saldo de {account.name}</h2></div><button type="button" className="icon-button" aria-label="Fechar" disabled={state.busy} onClick={onClose}><Icon name="close"/></button></header><form onSubmit={submit} aria-describedby={state.error?errorId:undefined}><dl className="settings-details"><div><dt>Saldo atual</dt><dd>{formatBRL(account.currentBalanceCents)}</dd></div><div><dt>Diferença calculada</dt><dd>{difference>=0?'+ ':''}{formatBRL(difference)}</dd></div></dl><div className="form-grid"><label className="field"><span>Saldo desejado</span><div className="money-input"><span>R$</span><input autoFocus value={target} onChange={event=>setTarget(event.target.value)} inputMode="decimal" required/></div></label><label className="field"><span>Data do ajuste</span><input name="date" type="date" defaultValue={today()} max={today()} required/></label><label className="field field--wide"><span>Motivo</span><textarea name="reason" required placeholder="Ex.: correção após conferência do extrato"/></label></div><FormError id={errorId} message={state.error}/><div className="form-actions"><Button type="button" kind="ghost" disabled={state.busy} onClick={onClose}>Cancelar</Button><Button loading={state.busy}>Registrar ajuste</Button></div></form></Sheet>
}

export function FixedExpenseForm({ accounts, categories, initial, onSubmit, onCancel }: { accounts: Account[]; categories: Category[]; initial?: FixedExpense; onSubmit(input: FixedExpenseInput): Promise<void>; onCancel(): void }) {
  const [accountId,setAccountId]=useState(initial?.accountId ?? accounts[0]?.id ?? ''), [categoryId,setCategoryId]=useState(initial?.categoryId ?? ''), [state,setState]=useState<SubmitState>({busy:false,error:''})
  const id = useId(), errorId = `${id}-error`, titleId = `${id}-title`, descriptionId = `${id}-description`, amountId = `${id}-amount`, dueDayId = `${id}-due-day`
  const accountOptions=accounts.map(account=>({id:account.id,label:account.name,detail:account.type==='checking'?'Conta corrente':account.type==='savings'?'Poupança':account.type==='credit_card'?'Cartão de crédito':'Dinheiro'})), categoryOptions=categories.filter(category=>category.kind==='expense'&&(!category.archivedAt||category.id===initial?.categoryId)).map(category=>({id:category.id,label:category.name}))
  const submit=async(event:FormEvent<HTMLFormElement>)=>{event.preventDefault();const data=new FormData(event.currentTarget);const amountCents=parseBRL(String(data.get('amount'))),dueDay=Number(data.get('dueDay'));if(amountCents===null||amountCents<=0)return setState({busy:false,error:'Informe um valor maior que zero.'});if(!accountId)return setState({busy:false,error:'Escolha uma conta.'});if(!categoryId)return setState({busy:false,error:'Escolha uma categoria.'});setState({busy:true,error:''});try{await onSubmit({description:String(data.get('description')).trim(),amountCents,dueDay,accountId,categoryId});setState({busy:false,error:'',status:'success'})}catch(error){setState({busy:false,error:errorMessage(error)})}}
  const status = submitStatus(state)
  return <Sheet onClose={onCancel} busy={state.busy} labelledBy={titleId} className="fixed-expense-sheet"><form className="card form-card fixed-expense-form" onSubmit={submit} aria-busy={state.busy ? true : undefined} data-state={status} aria-describedby={state.error ? errorId : undefined}><div className="card__heading"><div><h2 id={titleId}>{initial?'Ajuste os próximos meses':'O que vence todo mês?'}</h2></div><button type="button" className="icon-button" aria-label="Fechar" disabled={state.busy} onClick={onCancel}><Icon name="close"/></button></div><p className="form-intro">A previsão não movimenta seu saldo até você confirmar o pagamento.</p><div className="form-grid"><label className="field field--wide" htmlFor={descriptionId}><span>Descrição</span><input id={descriptionId} name="description" autoFocus required aria-required="true" defaultValue={initial?.description ?? ''} placeholder="Ex.: Aluguel, streaming, internet"/></label><label className="field" htmlFor={amountId}><span>Valor estimado</span><div className="money-input"><span aria-hidden="true">R$</span><input id={amountId} name="amount" required aria-required="true" aria-invalid={state.error === 'Informe um valor maior que zero.' ? true : undefined} aria-describedby={state.error === 'Informe um valor maior que zero.' ? errorId : undefined} inputMode="decimal" placeholder="0,00" defaultValue={initial?(initial.amountCents/100).toLocaleString('pt-BR',{minimumFractionDigits:2,maximumFractionDigits:2}):''}/></div></label><label className="field" htmlFor={dueDayId}><span>Dia do vencimento</span><input id={dueDayId} name="dueDay" type="number" min="1" max="31" required aria-required="true" defaultValue={initial?.dueDay ?? ''} placeholder="Ex.: 10"/></label><SearchCombobox label="Conta de pagamento" options={accountOptions} value={accountId} onChange={setAccountId} disabled={state.busy} error={state.error === 'Escolha uma conta.' ? state.error : ''}/><SearchCombobox label="Categoria" options={categoryOptions} value={categoryId} onChange={setCategoryId} disabled={state.busy} error={state.error === 'Escolha uma categoria.' ? state.error : ''}/></div><FormError id={errorId} message={state.error}/><div className="form-actions"><Button type="button" kind="ghost" disabled={state.busy} onClick={onCancel}>Cancelar</Button><Button disabled={state.busy} loading={state.busy} state={status}>{state.busy?'Salvando…':initial?'Salvar alterações':'Criar despesa fixa'}</Button></div></form></Sheet>
}

export function ConfirmFixedExpenseDialog({ occurrence, onClose, onSubmit }: { occurrence: FixedExpenseOccurrence; onClose(): void; onSubmit(input: ConfirmFixedExpenseOccurrenceInput): Promise<void> }) {
  const [state,setState]=useState<SubmitState>({busy:false,error:''})
  const id = useId(), amountId = `${id}-amount`, dateId = `${id}-date`, descriptionId = `${id}-description`, errorId = `${id}-error`
  const submit=async(event:FormEvent<HTMLFormElement>)=>{event.preventDefault();const data=new FormData(event.currentTarget),amountCents=parseBRL(String(data.get('amount')));if(amountCents===null||amountCents<=0)return setState({busy:false,error:'Informe um valor maior que zero.'});setState({busy:true,error:''});try{await onSubmit({amountCents,occurrenceDate:String(data.get('date'))});setState({busy:false,error:'',status:'success'});onClose()}catch(error){setState({busy:false,error:errorMessage(error)})}}
  const status = submitStatus(state)
  return <Sheet onClose={onClose} busy={state.busy} labelledBy="confirm-fixed-title" describedBy={descriptionId} className="dialog--confirm"><header><div><h2 id="confirm-fixed-title">{occurrence.description}</h2></div><button className="icon-button" type="button" disabled={state.busy} onClick={onClose} aria-label="Fechar"><Icon name="close"/></button></header><p id={descriptionId}>Registre o valor real pago. A despesa será incluída na conta {occurrence.accountName}.</p><form onSubmit={submit} aria-busy={state.busy ? true : undefined} data-state={status} aria-describedby={state.error ? errorId : undefined}><div className="form-grid"><label className="field" htmlFor={amountId}><span>Valor pago</span><div className="money-input"><span aria-hidden="true">R$</span><input id={amountId} autoFocus name="amount" inputMode="decimal" required aria-required="true" aria-invalid={state.error === 'Informe um valor maior que zero.' ? true : undefined} aria-describedby={state.error === 'Informe um valor maior que zero.' ? errorId : undefined} defaultValue={(occurrence.expectedAmountCents/100).toLocaleString('pt-BR',{minimumFractionDigits:2,maximumFractionDigits:2})}/></div></label><label className="field" htmlFor={dateId}><span>Data do pagamento</span><input id={dateId} name="date" type="date" defaultValue={today()} max={today()} required aria-required="true"/></label></div><FormError id={errorId} message={state.error}/><div className="form-actions"><Button type="button" kind="ghost" disabled={state.busy} onClick={onClose}>Cancelar</Button><Button disabled={state.busy} loading={state.busy} state={status}><Icon name="check"/>{state.busy?'Confirmando…':'Confirmar pagamento'}</Button></div></form></Sheet>
}

export function DeleteAccountDialog({ account, onClose, onConfirm }: { account: Account; onClose(): void; onConfirm(): Promise<void> }) {
  const [state,setState]=useState<SubmitState>({busy:false,error:''})
  const errorId = useId()
  const confirm=async()=>{setState({busy:true,error:''});try{await onConfirm();setState({busy:false,error:'',status:'success'})}catch(error){setState({busy:false,error:errorMessage(error)})}}
  const status = submitStatus(state)
  return <Modal onClose={onClose} busy={state.busy} labelledBy="delete-account-title" describedBy={`delete-account-description${state.error ? ` ${errorId}` : ''}`} className="dialog--confirm" alert>
    <header><div><h2 id="delete-account-title">Remover “{account.name}”?</h2></div></header>
    <p id="delete-account-description">Esta ação é permanente e não poderá ser desfeita. A conta só será removida se não possuir nenhuma movimentação vinculada.</p>
    <FormError id={errorId} message={state.error}/> 
    <div className="form-actions"><button data-dialog-initial type="button" className="button button--ghost" disabled={state.busy} onClick={onClose}>Cancelar</button><button type="button" className="button button--danger" data-state={status} aria-busy={state.busy ? true : undefined} aria-live={state.busy ? 'polite' : undefined} disabled={state.busy} onClick={()=>void confirm()}>{state.busy?'Removendo…':'Remover permanentemente'}</button></div>
  </Modal>
}

export function TransactionDialog({ accounts, categories, initial, onClose, onSubmit }: { accounts: Account[]; categories: Category[]; initial?: Transaction; onClose(): void; onSubmit(input: TransactionInput): Promise<void> }) {
  const [kind,setKind]=useState<TransactionKind>(initial?.kind ?? 'expense'), [accountId,setAccountId]=useState(initial?.accountId ?? accounts[0]?.id ?? ''), [destinationAccountId,setDestinationAccountId]=useState(initial?.destinationAccountId ?? ''), [categoryId,setCategoryId]=useState(initial?.categoryId ?? '')
  const [state,setState]=useState<SubmitState>({busy:false,error:''}),[splitEnabled,setSplitEnabled]=useState((initial?.splits?.length??0)>0)
  const id = useId(), descriptionId = `${id}-description`, amountId = `${id}-amount`, dateId = `${id}-date`, errorId = `${id}-error`
  const submit=async(event:FormEvent<HTMLFormElement>)=>{event.preventDefault();const data=new FormData(event.currentTarget);const cents=parseBRL(String(data.get('amount')));if(cents===null||cents<=0)return setState({busy:false,error:'Informe um valor maior que zero.'});if(!accountId)return setState({busy:false,error:'Escolha uma conta.'});if(kind==='transfer'&&!destinationAccountId)return setState({busy:false,error:'Escolha a conta de destino.'});const splits=kind!=='transfer'&&splitEnabled?[1,2].map(index=>({categoryId:String(data.get(`splitCategory${index}`)??''),subcategoryName:'',amountCents:parseBRL(String(data.get(`splitAmount${index}`)))??0})).filter(item=>item.categoryId&&item.amountCents>0):[];setState({busy:true,error:''});try{await onSubmit({kind,amountCents:cents,accountId,destinationAccountId:kind==='transfer'?destinationAccountId:'',categoryId:kind==='transfer'?'':categoryId,description:String(data.get('description')).trim(),occurrenceDate:String(data.get('date')),installmentCount:Number(data.get('installments')??1),subcategoryName:String(data.get('subcategory')??'').trim(),tags:String(data.get('tags')??'').split(',').map(tag=>tag.trim()).filter(Boolean),splits,monthlyRecurrence:data.get('recurrence')==='on'});setState({busy:false,error:'',status:'success'});onClose()}catch(error){setState({busy:false,error:errorMessage(error)})}}
  const filtered=categories.filter((category)=>category.kind===kind&&(!category.archivedAt||category.id===initial?.categoryId)), typeName=(account:Account)=>account.type==='checking'?'Conta corrente':account.type==='savings'?'Poupança':account.type==='credit_card'?'Cartão de crédito':'Dinheiro'
  const selectableAccounts=kind==='expense'?accounts:accounts.filter(account=>account.type!=='credit_card'), selectedAccount=accounts.find(account=>account.id===accountId)
  const accountOptions=selectableAccounts.map(account=>({id:account.id,label:account.name,detail:`${typeName(account)} · ${formatBRL(Math.abs(account.currentBalanceCents))}`})), categoryOptions=filtered.map(category=>({id:category.id,label:category.name}))
  useEffect(()=>{if(kind!=='expense'&&selectedAccount?.type==='credit_card')setAccountId(selectableAccounts[0]?.id??'')},[kind,selectedAccount?.type,selectableAccounts])
  const status = submitStatus(state)
  return <Sheet onClose={onClose} busy={state.busy} labelledBy="transaction-title"><header><div><p className="context-label">Movimentação</p><h2 id="transaction-title">{initial?'Editar registro':'Novo registro'}</h2></div><button className="icon-button" type="button" disabled={state.busy} onClick={onClose} aria-label="Fechar"><Icon name="close"/></button></header><form onSubmit={submit} aria-busy={state.busy ? true : undefined} data-state={status} aria-describedby={state.error ? errorId : undefined}>
    <div className="segmented" role="radiogroup" aria-label="Tipo de movimentação">{([['expense','Despesa'],['income','Receita'],['transfer','Transferência']] as const).map(([value,label])=><label key={value}><input type="radio" name="kind" value={value} checked={kind===value} onChange={()=>{setKind(value);setCategoryId('');if(value!=='transfer')setDestinationAccountId('')}}/><span>{label}</span></label>)}</div>
    <label className="field field--wide" htmlFor={descriptionId}><span>Descrição {kind==='transfer'&&<em>opcional</em>}</span><input id={descriptionId} autoFocus data-dialog-initial name="description" defaultValue={initial?.description ?? ''} required={kind!=='transfer'} aria-required={kind!=='transfer' || undefined} placeholder={kind==='expense'?'Ex.: Mercado':kind==='income'?'Ex.: Pagamento recebido':'Preenchida automaticamente se ficar vazia'} /></label>
    <div className="form-grid transaction-form__essentials"><label className="field" htmlFor={amountId}><span>Valor</span><div className="money-input"><span aria-hidden="true">R$</span><input id={amountId} name="amount" inputMode="decimal" required aria-required="true" aria-invalid={state.error === 'Informe um valor maior que zero.' ? true : undefined} aria-describedby={state.error === 'Informe um valor maior que zero.' ? errorId : undefined} placeholder="0,00" defaultValue={initial?(initial.amountCents/100).toLocaleString('pt-BR',{minimumFractionDigits:2,maximumFractionDigits:2}):''} /></div></label><label className="field" htmlFor={dateId}><span>Data</span><input id={dateId} name="date" type="date" defaultValue={initial?.occurrenceDate ?? today()} max={today()} required aria-required="true" /></label>
      <SearchCombobox label={kind==='transfer'?'Conta de origem':'Conta'} options={accountOptions} value={accountId} onChange={setAccountId} disabled={state.busy} error={state.error === 'Escolha uma conta.' ? state.error : ''}/>
      {kind==='transfer'?<SearchCombobox label="Conta de destino" options={accountOptions.filter(option=>option.id!==accountId)} value={destinationAccountId} onChange={setDestinationAccountId} disabled={state.busy} error={state.error === 'Escolha a conta de destino.' ? state.error : ''}/>:<SearchCombobox label="Categoria" options={[{id:'',label:'Sem categoria'},...categoryOptions]} value={categoryId} onChange={setCategoryId} optional disabled={state.busy}/>} 
      {kind==='expense'&&<label className="field"><span>Parcelas</span><input aria-label="Parcelas" name="installments" type="number" min="1" max="48" defaultValue={initial?.installmentCount??1}/><small>{selectedAccount?.type==='credit_card'?'A compra inteira conta como despesa hoje; as parcelas organizam as faturas.':'As próximas parcelas ficam pendentes até a confirmação.'}</small></label>}
    </div>
    {kind!=='transfer'&&<details className="advanced-disclosure">
      <summary>Mais detalhes</summary>
      <div className="advanced-disclosure__content">
        <div className="form-grid"><label className="field"><span>Subcategoria <em>opcional</em></span><input name="subcategory" defaultValue={initial?.subcategoryName??''}/></label><label className="field"><span>Tags <em>separadas por vírgula</em></span><input name="tags" defaultValue={initial?.tags?.map(tag=>tag.name).join(', ')??''}/></label></div>
        {selectedAccount?.type!=='credit_card'&&(!initial||initial.recurrenceRuleId)&&<label className="confirmation-check"><input name="recurrence" type="checkbox" defaultChecked={Boolean(initial?.recurrenceRuleId)}/>{initial?.recurrenceRuleId?'Manter recorrência mensal':'Repetir mensalmente'}</label>}
        <label className="confirmation-check"><input type="checkbox" checked={splitEnabled} onChange={event=>setSplitEnabled(event.target.checked)}/>Dividir entre categorias</label>
        {splitEnabled&&[1,2].map((index)=><div className="field" key={index}><span>Divisão {index}</span><select name={`splitCategory${index}`} required><option value="">Categoria</option>{filtered.map(category=><option key={category.id} value={category.id}>{category.name}</option>)}</select><input name={`splitAmount${index}`} aria-label={`Valor da divisão ${index}`} inputMode="decimal" required/></div>)}
      </div>
    </details>}
    <FormError id={errorId} message={state.error}/><div className="form-actions"><Button type="button" kind="ghost" disabled={state.busy} onClick={onClose}>Cancelar</Button><Button disabled={state.busy} loading={state.busy} state={status}>{state.busy?'Salvando…':initial?'Salvar alterações':'Registrar'}</Button></div>
  </form></Sheet>
}
