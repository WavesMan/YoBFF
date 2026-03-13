import type { ReactNode } from 'react';
import { FiChevronLeft, FiChevronRight, FiUser } from 'react-icons/fi';
import './Sidebar.css';

export type SidebarItem = {
  key: string;
  label: string;
  icon: ReactNode;
  group?: string;
};

type SidebarProps = {
  items: SidebarItem[];
  activeKey: string;
  onChange: (key: string) => void;
  collapsed: boolean;
  onToggle: () => void;
};

export function Sidebar({ items, activeKey, onChange, collapsed, onToggle }: SidebarProps) {
  // Group items
  const groupedItems = items.reduce((acc, item) => {
    const group = item.group || 'General';
    if (!acc[group]) acc[group] = [];
    acc[group].push(item);
    return acc;
  }, {} as Record<string, SidebarItem[]>);

  return (
    <aside className={`sidebar ${collapsed ? 'collapsed' : ''}`}>
      <div className="brand">
        <img src="/Logo.png" alt="YoBFF" className="brand-logo" />
        <span className="brand-text">YoBFF Web</span>
      </div>

      <nav className="nav-group">
        {Object.entries(groupedItems).map(([group, groupItems]) => (
          <div key={group} className="nav-group-section">
             {/* Only show title if it's not the default General group or if we want to separate sections */}
            {!collapsed && group !== 'General' && (
              <div className="nav-group-title">{group}</div>
            )}
            {groupItems.map((item) => (
              <button
                key={item.key}
                className={`nav-item ${activeKey === item.key ? 'active' : ''}`}
                onClick={() => onChange(item.key)}
                aria-current={activeKey === item.key ? 'page' : undefined}
                title={collapsed ? item.label : undefined}
              >
                <span className="nav-icon">{item.icon}</span>
                <span className="nav-label">{item.label}</span>
              </button>
            ))}
          </div>
        ))}
      </nav>

      <div className="user-profile">
        <div className="avatar">
          <FiUser />
        </div>
        <div className="user-info">
          <span className="user-name">Admin User</span>
          <span className="user-role">Administrator</span>
        </div>
      </div>

      <button 
        className="sidebar-toggle" 
        onClick={onToggle} 
        aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
      >
        {collapsed ? <FiChevronRight size={14} /> : <FiChevronLeft size={14} />}
      </button>
    </aside>
  );
}
