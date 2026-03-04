import type { ReactNode } from 'react'
import { FiChevronLeft, FiChevronRight } from 'react-icons/fi'

type SidebarItem = {
  key: string
  label: string
  icon: ReactNode
}

type SidebarProps = {
  items: SidebarItem[]
  activeKey: string
  onChange: (key: string) => void
  collapsed: boolean
  onToggle: () => void
}

export function Sidebar({ items, activeKey, onChange, collapsed, onToggle }: SidebarProps) {
  return (
    <aside className={`sidebar ${collapsed ? 'collapsed' : ''}`}>
      <div className="brand">
        <span className="brand-icon" />
        <span className="brand-text">YoBFF</span>
      </div>
      <nav className="nav-group">
        {items.map((item) => (
          <div
            key={item.key}
            className={`nav-item ${activeKey === item.key ? 'active' : ''}`}
            onClick={() => onChange(item.key)}
            role="button"
            tabIndex={0}
            aria-current={activeKey === item.key ? 'page' : undefined}
            onKeyDown={(event) => {
              if (event.key === 'Enter' || event.key === ' ') {
                onChange(item.key)
              }
            }}
          >
            <span className="nav-icon">{item.icon}</span>
            <span className="nav-label">{item.label}</span>
          </div>
        ))}
      </nav>
      <div className="nav-footer">
        <button className="button secondary sidebar-toggle" onClick={onToggle} aria-label="收起或展开侧边栏">
          <span className="nav-icon">
            {collapsed ? <FiChevronRight /> : <FiChevronLeft />}
          </span>
          <span className="nav-label">{collapsed ? '展开侧边栏' : '收起侧边栏'}</span>
        </button>
        <a className="button secondary" href="https://yobff.dev">
          查看文档
        </a>
      </div>
    </aside>
  )
}
