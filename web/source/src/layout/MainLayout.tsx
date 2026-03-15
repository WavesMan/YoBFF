import { useState, useMemo } from 'react'
import { FiActivity, FiEye, FiLayers, FiLock, FiShuffle, FiSliders } from 'react-icons/fi'
import Sidebar, { type SidebarItem } from '../components/Sidebar'
import { TopBar } from '../components/TopBar'

type MainLayoutProps = {
  token: string | null
  onLogout: () => void
  children: React.ReactNode
  activeSection: string
  setActiveSection: (section: string) => void
  errorMessage: string
  statusMessage: string
}

const menuItems: SidebarItem[] = [
  { key: 'dashboard', label: '仪表盘', icon: <FiActivity />, group: 'Overview' },
  { key: 'traffic', label: '流量管理', icon: <FiShuffle />, group: 'Traffic' },
  { key: 'certificates', label: '证书管理', icon: <FiLock />, group: 'Traffic' },
  { key: 'observability', label: '观测中心', icon: <FiEye />, group: 'System' },
  { key: 'weaver', label: '可视化实验室', icon: <FiLayers />, group: 'Labs' },
  { key: 'system', label: '系统设置', icon: <FiSliders />, group: 'System' },
  // { key: 'uidemo', label: 'UI Demo', icon: <FiLayers />, group: 'System' },
]

export function MainLayout({
  onLogout,
  children,
  activeSection,
  setActiveSection,
  errorMessage,
  statusMessage,
}: MainLayoutProps) {
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)

  const pageHeader = useMemo(() => {
    const headers: Record<string, { title: string; subtitle: string }> = {
      dashboard: {
        title: '仪表盘',
        subtitle: '核心健康度与流量态势概览',
      },
      traffic: {
        title: '流量管理',
        subtitle: '域名与转发策略的统一配置入口',
      },
      certificates: {
        title: '证书管理',
        subtitle: 'SSL/TLS 证书生命周期管理',
      },
      observability: {
        title: '观测中心',
        subtitle: '日志检索、监控大盘与链路追踪',
      },
      weaver: {
        title: '可视化实验室',
        subtitle: '可视化草稿编辑与映射试跑工作台',
      },
      system: {
        title: '系统设置',
        subtitle: '运行参数与管理能力的集中配置',
      },
      uidemo: {
        title: 'UI Component Demo',
        subtitle: 'Showcase of reusable UI components and layouts',
      },
    }
    return headers[activeSection] || {
      title: 'BFF 负载均衡网关管理端',
      subtitle: '沉浸式暗色风格，支持配置实时感知与审计流程',
    }
  }, [activeSection])

  const notificationCount = useMemo(() => {
    let count = 0
    if (errorMessage) {
      count += 1
    }
    if (statusMessage) {
      count += 1
    }
    return count
  }, [errorMessage, statusMessage])

  const breadcrumbs = useMemo(() => {
    const currentItem = menuItems.find(item => item.key === activeSection);
    return [
      { label: currentItem?.label || activeSection }
    ];
  }, [activeSection]);

  return (
    <div className={`app-shell ${sidebarCollapsed ? 'collapsed' : ''}`}>
      <Sidebar
        items={menuItems}
        activeKey={activeSection}
        onChange={setActiveSection}
        collapsed={sidebarCollapsed}
        onToggle={() => setSidebarCollapsed((prev) => !prev)}
      />
      <div className="content">
        <TopBar
          onLogout={onLogout}
          notificationCount={notificationCount}
          breadcrumbs={breadcrumbs}
        />
        <main className="main">
          <div>
            <h1 className="page-title">{pageHeader.title}</h1>
            <p className="page-subtitle">{pageHeader.subtitle}</p>
          </div>
          {errorMessage && (
            <div className="error-banner" role="alert">
              {errorMessage}
            </div>
          )}
          {statusMessage && (
            <div className="success-banner" role="status">
              {statusMessage}
            </div>
          )}
          {children}
        </main>
        <footer className="app-footer">
          <p>&copy; 2026 YoBFF - BFF Load Balancing Gateway. Licensed under the Apache License, Version 2.0.</p>
          <p>
            Available at: <a href="https://github.com/WavesMan/YoBFF" target="_blank" rel="noreferrer">https://github.com/WavesMan/YoBFF</a>
          </p>
        </footer>
      </div>
    </div>
  )
}
