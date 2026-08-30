import { useEffect, useId, useState, type PropsWithChildren, type ReactNode } from 'react'
import { Button, Icon, type IconName } from './components'

export type AppView = 'dashboard' | 'accounts' | 'cards' | 'transactions' | 'fixedExpenses' | 'categories' | 'budget' | 'goals' | 'settings'

export interface NavigationItem {
  id: AppView
  label: string
  icon: IconName
}

export interface NavigationGroup {
  label: string
  items: NavigationItem[]
}

function useCompactNavigation() {
  const [compact, setCompact] = useState(false)

  useEffect(() => {
    const media = window.matchMedia?.('(max-width: 59.99rem)')
    if (!media) return
    const update = () => setCompact(media.matches)
    update()
    media.addEventListener?.('change', update)
    return () => media.removeEventListener?.('change', update)
  }, [])

  return compact
}

export function Sidebar({ groups, active, collapsed, displayName, onNavigate, onCollapsedChange }: {
  groups: NavigationGroup[]
  active: AppView
  collapsed: boolean
  displayName?: string
  onNavigate(view: AppView): void
  onCollapsedChange(collapsed: boolean): void
}) {
  const compact = useCompactNavigation()
  const [moreOpen, setMoreOpen] = useState(false)
  const primaryIds: AppView[] = ['dashboard', 'transactions', 'accounts', 'budget']
  const allItems = groups.flatMap(group => group.items)
  const primaryItems = primaryIds.map(id => allItems.find(item => item.id === id)).filter((item): item is NavigationItem => Boolean(item))
  const moreItems = allItems.filter(item => !primaryIds.includes(item.id) && item.id !== 'settings')
  const navigate = (view: AppView) => {
    setMoreOpen(false)
    onNavigate(view)
  }
  const toggleLabel = collapsed ? 'Expandir navegação' : 'Recolher navegação'

  if (compact) {
    return <aside className="mobile-navigation" aria-label="Navegação compacta">
      {moreOpen && <div className="mobile-more" role="menu" aria-label="Mais destinos">
        {moreItems.map(item => <button key={item.id} type="button" role="menuitem" className={active === item.id ? 'active' : ''} onClick={() => navigate(item.id)}><Icon name={item.icon}/><span>{item.label}</span></button>)}
        <button type="button" role="menuitem" className={active === 'settings' ? 'active' : ''} onClick={() => navigate('settings')}><Icon name="palette"/><span>Configurações</span></button>
      </div>}
      <nav aria-label="Principal">
        {primaryItems.map(item => <button key={item.id} type="button" className={active === item.id ? 'active' : ''} aria-current={active === item.id ? 'page' : undefined} onClick={() => navigate(item.id)}><Icon name={item.icon}/><span>{item.label === 'Movimentações' ? 'Atividade' : item.label}</span></button>)}
        <button type="button" className={moreOpen || moreItems.some(item => item.id === active) || active === 'settings' ? 'active' : ''} aria-haspopup="menu" aria-expanded={moreOpen} onClick={() => setMoreOpen(value => !value)}><Icon name="ellipsis"/><span>Mais</span></button>
      </nav>
    </aside>
  }

  return <aside className={`sidebar${collapsed ? ' sidebar--collapsed' : ''}`}>
    <div className="sidebar__header">
      <div className="brand" aria-label="c ash"><span>[c]</span><b>ash</b></div>
      <button type="button" className="sidebar__toggle" aria-label={toggleLabel} aria-expanded={!collapsed} aria-controls="primary-navigation" title={toggleLabel} onClick={() => onCollapsedChange(!collapsed)}><Icon name={collapsed ? 'panelLeftOpen' : 'panelLeftClose'}/></button>
    </div>
    <nav id="primary-navigation" className="sidebar__nav" aria-label="Principal">
      {groups.map(group => <section className="nav-group" aria-label={group.label} key={group.label}>
        <p className="nav-group__label" aria-hidden={collapsed}>{group.label}</p>
        {group.items.map(item => <button key={item.id} type="button" className={active === item.id ? 'active' : ''} aria-label={item.label} title={collapsed ? item.label : undefined} aria-current={active === item.id ? 'page' : undefined} onClick={() => navigate(item.id)}><Icon name={item.icon}/><span>{item.label}</span></button>)}
      </section>)}
    </nav>
    <div className="sidebar__footer">
      <button type="button" className={`sidebar__settings${active === 'settings' ? ' active' : ''}`} aria-label="Configurações" title={collapsed ? 'Configurações' : undefined} aria-current={active === 'settings' ? 'page' : undefined} onClick={() => navigate('settings')}><Icon name="palette"/><span>Configurações</span></button>
      <div className="sidebar__profile">
        <span className="avatar" aria-hidden="true">{displayName?.[0]?.toUpperCase() || 'C'}</span>
        <span><strong>{displayName || 'Meu espaço'}</strong><small>Privado e local</small></span>
      </div>
    </div>
  </aside>
}

export function Toolbar({ title, eyebrow, transactionDisabled, transactionHint, onTransaction }: {
  title: string
  eyebrow?: string
  transactionDisabled?: boolean
  transactionHint?: string
  onTransaction(): void
}) {
  const hintId = useId()
  return <header className="topbar">
    <div className="topbar__title">{eyebrow && <p className="eyebrow">{eyebrow}</p>}<h1>{title}</h1></div>
    <div className="topbar__actions"><Button onClick={onTransaction} disabled={transactionDisabled} title={transactionHint ?? 'Atalho: Ctrl+N'} aria-describedby={transactionDisabled ? hintId : undefined} aria-keyshortcuts="Control+N Meta+N"><Icon name="plus"/> Nova movimentação</Button>{transactionDisabled && <span className="sr-only" id={hintId}>{transactionHint ?? 'Crie uma conta primeiro.'}</span>}</div>
  </header>
}

export function AppShell({ sidebar, toolbar, collapsed, children }: PropsWithChildren<{ sidebar: ReactNode; toolbar: ReactNode; collapsed: boolean }>) {
  return <div className={`shell${collapsed ? ' shell--collapsed' : ''}`}>{sidebar}<main className="workspace">{toolbar}{children}</main></div>
}

export function PageTransition({ view, children }: PropsWithChildren<{ view: AppView }>) {
  return <div className="workspace-view" data-view={view}>{children}</div>
}

export function Surface({ as: Component = 'section', className = '', children }: PropsWithChildren<{ as?: 'article' | 'section' | 'div'; className?: string }>) {
  return <Component className={['surface', className].filter(Boolean).join(' ')}>{children}</Component>
}

export function Section({ title, action, children, className = '' }: PropsWithChildren<{ title: string; action?: ReactNode; className?: string }>) {
  return <section className={['section', className].filter(Boolean).join(' ')}><header className="section__header"><h2>{title}</h2>{action}</header>{children}</section>
}

export function StatCard({ label, value, detail, tone = 'neutral' }: { label: string; value: ReactNode; detail?: ReactNode; tone?: 'neutral' | 'positive' | 'negative' }) {
  return <article className={`stat-card stat-card--${tone}`}><p>{label}</p><strong>{value}</strong>{detail && <small>{detail}</small>}</article>
}
