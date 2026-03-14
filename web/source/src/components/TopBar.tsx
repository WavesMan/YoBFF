import React from 'react';
import { FiSearch, FiBell, FiSettings, FiLogOut } from 'react-icons/fi';
import './TopBar.css';

interface BreadcrumbItem {
  label: string;
  href?: string;
}

type TopBarProps = {
  breadcrumbs?: BreadcrumbItem[];
  onLogout?: () => void;
  notificationCount?: number;
};

export function TopBar({ breadcrumbs = [], onLogout, notificationCount = 0 }: TopBarProps) {
  return (
    <header className="topbar">
      <div className="breadcrumbs">
        <a href="/" className="breadcrumb-item">控制台</a>
        {breadcrumbs.map((item, index) => (
          <React.Fragment key={index}>
            <span className="breadcrumb-sep">/</span>
            {item.href ? (
              <a href={item.href} className="breadcrumb-item">{item.label}</a>
            ) : (
              <span className="breadcrumb-active">{item.label}</span>
            )}
          </React.Fragment>
        ))}
      </div>

      <div className="topbar-actions">
        <div className="search-box">
          <FiSearch className="search-icon" />
          <input
            className="search-input"
            placeholder="搜索域名、配置项、证书..."
            aria-label="全局搜索"
          />
        </div>
        
        <button className="icon-btn" aria-label="Notifications">
          <FiBell size={18} />
          {notificationCount > 0 && <span className="badge-dot" />}
        </button>
        
        <button className="icon-btn" aria-label="Settings">
          <FiSettings size={18} />
        </button>

        <button className="icon-btn" onClick={onLogout} aria-label="Logout" title="退出登录">
          <FiLogOut size={18} />
        </button>
      </div>
    </header>
  );
}
