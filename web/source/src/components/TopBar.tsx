type TopBarProps = {
  breadcrumbs: string
  token: string | null
  onLogout: () => void
}

export function TopBar({ breadcrumbs, token, onLogout }: TopBarProps) {
  return (
    <header className="topbar">
      <div className="breadcrumbs">{breadcrumbs}</div>
      <div className="topbar-actions">
        <input
          className="search-input"
          placeholder="搜索域名、配置项、证书..."
          aria-label="全局搜索"
        />
        <span className="badge">通知 2</span>
        <span className="badge">{token ? '已登录' : '未登录'}</span>
        {token && (
          <button className="button secondary" onClick={onLogout}>
            退出登录
          </button>
        )}
      </div>
    </header>
  )
}
