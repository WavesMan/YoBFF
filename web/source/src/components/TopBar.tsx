type TopBarProps = {
  onLogout: () => void
  notificationCount: number
  isLoggedIn: boolean
}

export function TopBar({ onLogout, notificationCount, isLoggedIn }: TopBarProps) {
  return (
    <header className="topbar">
      <div className="breadcrumbs">控制台 / 仪表盘</div>
      <div className="topbar-actions">
        <input
          className="search-input"
          placeholder="搜索域名、配置项、证书..."
          aria-label="全局搜索"
        />
        <span className="badge">通知 {notificationCount}</span>
        <span className="badge">{isLoggedIn ? '已登录' : '未登录'}</span>
        <button className="button secondary" onClick={onLogout}>
          退出登录
        </button>
      </div>
    </header>
  )
}
