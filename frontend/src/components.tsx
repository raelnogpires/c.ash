import {
  Archive, ArrowDownRight, ArrowRight, ArrowRightLeft, ArrowUpRight, CalendarClock,
  Check, ChevronRight, CircleDashed, Ellipsis, Equal, Eye, EyeOff, House, List,
  Palette, PanelLeftClose, PanelLeftOpen, Plus, ReceiptText, ShieldCheck,
  TriangleAlert, WalletCards, WalletMinimal, X, type LucideIcon,
} from 'lucide-react'
import { forwardRef, useEffect, useId, useRef, useState, type PropsWithChildren, type ReactNode } from 'react'
import { formatBRL, formatDate } from './format'
import type { Account, Transaction } from './types'

const iconMap = {
  archive: Archive, arrowDownRight: ArrowDownRight, arrowRight: ArrowRight,
  arrowRightLeft: ArrowRightLeft, arrowUpRight: ArrowUpRight,
  calendarClock: CalendarClock, check: Check, chevronRight: ChevronRight,
  circleDashed: CircleDashed, ellipsis: Ellipsis, equal: Equal, eye: Eye,
  eyeOff: EyeOff, house: House, list: List, palette: Palette,
  panelLeftClose: PanelLeftClose, panelLeftOpen: PanelLeftOpen, plus: Plus,
  receipt: ReceiptText, shieldCheck: ShieldCheck, warning: TriangleAlert,
  wallet: WalletCards, walletMinimal: WalletMinimal, close: X,
} satisfies Record<string, LucideIcon>

export type IconName = keyof typeof iconMap

export function Icon({ name, className }: { name: IconName; className?: string }) {
  const Glyph = iconMap[name]
  return <Glyph className={['icon', className].filter(Boolean).join(' ')} aria-hidden="true" strokeWidth={1.8}/>
}

export type ButtonState = 'idle' | 'loading' | 'error' | 'success'

export const Button = forwardRef<HTMLButtonElement, PropsWithChildren<{ kind?: 'primary' | 'secondary' | 'ghost'; loading?: boolean; state?: ButtonState } & React.ButtonHTMLAttributes<HTMLButtonElement>>>(function Button({ children, kind = 'primary', loading = false, state = 'idle', className, disabled = false, ...props }, ref) {
  const isLoading = loading || state === 'loading'
  const resolvedState = isLoading ? 'loading' : state
  return <button ref={ref} {...props} className={['button', `button--${kind}`, className].filter(Boolean).join(' ')} data-state={resolvedState} aria-busy={isLoading ? true : undefined} aria-live={isLoading ? 'polite' : undefined} disabled={disabled || isLoading}>{children}</button>
})

export function EmptyState({ title, children, action, icon = 'circleDashed' }: PropsWithChildren<{ title: string; action?: ReactNode; icon?: IconName }>) {
  return <section className="empty"><div className="empty__mark" aria-hidden="true"><Icon name={icon}/></div><h3>{title}</h3><p>{children}</p>{action}</section>
}

export interface ComboboxOption { id: string; label: string; detail?: string }

const normalize = (value: string) => value.normalize('NFD').replace(/[\u0300-\u036f]/g, '').toLocaleLowerCase('pt-BR')

export function SearchCombobox({ label, options, value, onChange, placeholder = 'Buscar…', optional = false, disabled = false, busy = false, error = '' }: { label: string; options: ComboboxOption[]; value: string; onChange(value: string): void; placeholder?: string; optional?: boolean; disabled?: boolean; busy?: boolean; error?: string }) {
  const id = useId(), inputRef = useRef<HTMLInputElement>(null)
  const selected = options.find((option) => option.id === value)
  const [query, setQuery] = useState(selected?.label ?? ''), [open, setOpen] = useState(false), [active, setActive] = useState(-1)
  useEffect(() => { setQuery(selected?.label ?? '') }, [selected?.label])
  const filtered = options.filter((option) => normalize(`${option.label} ${option.detail ?? ''}`).includes(normalize(query)))
  const choose = (option: ComboboxOption) => { onChange(option.id); setQuery(option.label); setOpen(false); requestAnimationFrame(() => inputRef.current?.focus()) }
  const keyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') { event.preventDefault(); setOpen(true); setActive((index) => event.key === 'ArrowDown' ? Math.min(index + 1, Math.max(filtered.length - 1, 0)) : index < 0 ? Math.max(filtered.length - 1, 0) : Math.max(index - 1, 0)) }
    else if (event.key === 'Home' && open) { event.preventDefault(); setActive(0) }
    else if (event.key === 'End' && open) { event.preventDefault(); setActive(Math.max(filtered.length - 1, 0)) }
    else if (event.key === 'Enter' && open && filtered[active]) { event.preventDefault(); choose(filtered[active]) }
    else if (event.key === 'Escape') { event.preventDefault(); setOpen(false); setQuery(selected?.label ?? '') }
    else if (event.key === 'Tab') { setOpen(false); if (!selected || query !== selected.label) setQuery(selected?.label ?? '') }
  }
  return <label className={`field combobox ${error ? 'field--error' : ''}`} data-state={busy ? 'loading' : error ? 'error' : 'idle'} onBlur={(event) => { if (!event.currentTarget.contains(event.relatedTarget)) { setOpen(false); setQuery(selected?.label ?? '') } }}>
    <span>{label}{optional && <em> opcional</em>}</span>
    <input ref={inputRef} role="combobox" aria-expanded={open} aria-controls={`${id}-list`} aria-autocomplete="list" aria-activedescendant={open && filtered[active] ? `${id}-${filtered[active].id}` : undefined} aria-required={!optional || undefined} aria-invalid={error ? true : undefined} aria-busy={busy ? true : undefined} aria-describedby={`${id}-error`} value={query} placeholder={placeholder} disabled={disabled || busy} onFocus={() => { setOpen(true); setActive(options.findIndex(option => option.id === value)) }} onChange={(event) => { setQuery(event.target.value); onChange(''); setOpen(true); setActive(-1) }} onKeyDown={keyDown}/>
    {busy && <span className="combobox__status" role="status">Carregando…</span>}
    {open && !disabled && !busy && <ul id={`${id}-list`} role="listbox" className="combobox__list" aria-busy={busy ? true : undefined}>
      {filtered.length ? filtered.map((option, index) => <li key={option.id} id={`${id}-${option.id}`} role="option" aria-selected={option.id === value} className={index === active ? 'active' : ''} onMouseDown={(event) => event.preventDefault()} onMouseEnter={() => setActive(index)} onClick={() => choose(option)}><strong>{option.label}</strong>{option.detail && <small>{option.detail}</small>}</li>) : <li className="combobox__empty">Nenhum resultado</li>}
    </ul>}
    <small id={`${id}-error`} className="field__error" aria-live="polite">{error}</small>
  </label>
}

function TransactionActions({ tx, onEdit, onRemove }: { tx: Transaction; onEdit?(tx: Transaction, trigger: HTMLElement): void; onRemove?(tx: Transaction, trigger: HTMLElement): void }) {
  const [open, setOpen] = useState(false), button = useRef<HTMLButtonElement>(null), menu = useRef<HTMLDivElement>(null)
  useEffect(() => { if (!open) return; menu.current?.querySelector<HTMLButtonElement>('button')?.focus() }, [open])
  const close = () => { setOpen(false); requestAnimationFrame(() => button.current?.focus()) }
  return <div className="action-menu" onBlur={(event) => { if (!event.currentTarget.contains(event.relatedTarget)) setOpen(false) }}>
    <button ref={button} type="button" className="icon-button" aria-label={`Mais ações para ${tx.description}`} aria-haspopup="menu" aria-expanded={open} onClick={() => setOpen(value => !value)}><Icon name="ellipsis"/></button>
    {open && <div ref={menu} className="action-menu__panel" role="menu" onKeyDown={(event) => { if (event.key === 'Escape') { event.preventDefault(); close(); return } if(event.key==='ArrowDown'||event.key==='ArrowUp'){event.preventDefault();const items=Array.from(event.currentTarget.querySelectorAll<HTMLButtonElement>('button'));const index=items.indexOf(document.activeElement as HTMLButtonElement);items[(index+(event.key==='ArrowDown'?1:-1)+items.length)%items.length]?.focus()} }}><button type="button" role="menuitem" onClick={() => { setOpen(false); if (button.current) onEdit?.(tx, button.current) }}>Editar</button><button type="button" role="menuitem" className="danger" onClick={() => { setOpen(false); if (button.current) onRemove?.(tx, button.current) }}>Remover</button></div>}
  </div>
}

export function AccountActions({ account, onEdit, onRemove }: { account: Account; onEdit(account: Account, trigger: HTMLElement): void; onRemove(account: Account, trigger: HTMLElement): void }) {
  const [open, setOpen] = useState(false), button = useRef<HTMLButtonElement>(null), menu = useRef<HTMLDivElement>(null)
  useEffect(() => { if (open) menu.current?.querySelector<HTMLButtonElement>('button')?.focus() }, [open])
  const close = () => { setOpen(false); requestAnimationFrame(() => button.current?.focus()) }
  return <div className="action-menu account-card__actions" onBlur={(event) => { if (!event.currentTarget.contains(event.relatedTarget)) setOpen(false) }}>
    <button ref={button} type="button" className="icon-button" aria-label={`Mais ações para ${account.name}`} aria-haspopup="menu" aria-expanded={open} onClick={() => setOpen(value => !value)}><Icon name="ellipsis"/></button>
    {open && <div ref={menu} className="action-menu__panel" role="menu" onKeyDown={(event) => { if(event.key==='Escape'){event.preventDefault();close();return}if(event.key==='ArrowDown'||event.key==='ArrowUp'){event.preventDefault();const items=Array.from(event.currentTarget.querySelectorAll<HTMLButtonElement>('button'));const index=items.indexOf(document.activeElement as HTMLButtonElement);items[(index+(event.key==='ArrowDown'?1:-1)+items.length)%items.length]?.focus()} }}>
      <button type="button" role="menuitem" onClick={() => { setOpen(false); if(button.current) onEdit(account, button.current) }}>Editar</button>
      <button type="button" role="menuitem" className="danger" onClick={() => { setOpen(false); if(button.current) onRemove(account, button.current) }}>Remover</button>
    </div>}
  </div>
}

export function TransactionList({ transactions, empty, onEdit, onRemove }: { transactions: Transaction[]; empty?: ReactNode; onEdit?(tx: Transaction, trigger: HTMLElement): void; onRemove?(tx: Transaction, trigger: HTMLElement): void }) {
  if (!transactions.length) return <>{empty}</>
  return <ul className="transaction-list" aria-label="Movimentações">
    {transactions.map((tx) => <li key={tx.id} className="transaction-row">
      <span className={`transaction-row__glyph ${tx.kind}`} aria-hidden="true"><Icon name={tx.kind === 'income' ? 'arrowUpRight' : tx.kind === 'expense' ? 'arrowDownRight' : 'arrowRightLeft'}/></span>
      <span className="transaction-row__main"><span className="transaction-row__title"><strong>{tx.description}</strong>{tx.automaticImport && <span className="import-badge">Importação automática</span>}</span><small>{tx.kind === 'transfer' ? `${tx.accountName} para ${tx.destinationAccountName}` : `${tx.accountName}${tx.categoryName ? ` · ${tx.categoryName}` : ''}`}</small></span>
      <span className="transaction-row__date">{formatDate(tx.occurrenceDate)}</span>
      <span className={`transaction-row__amount ${tx.kind}`}><span className="sr-only">{tx.kind === 'income' ? 'Receita' : tx.kind === 'expense' ? 'Despesa' : 'Transferência'}:</span>{tx.kind === 'income' ? '+ ' : tx.kind === 'expense' ? '− ' : ''}{formatBRL(tx.amountCents)}</span>
      {(onEdit || onRemove) && <TransactionActions tx={tx} onEdit={onEdit} onRemove={onRemove}/>} 
    </li>)}
  </ul>
}
