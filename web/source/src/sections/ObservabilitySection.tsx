import type { HealthzResponse, LogStats } from '../admin/types'

type ObservabilitySectionProps = {
  health: HealthzResponse | null
  logStats: LogStats | null
  loadingHealth: boolean
  loadingLogStats: boolean
}

export function ObservabilitySection({
  health,
  logStats,
  loadingHealth,
  loadingLogStats,
}: ObservabilitySectionProps) {
  const isLoading = loadingHealth || loadingLogStats
  const logLines = [
    `健康检查 ${health?.status || 'unknown'}`,
    `日志总量 ${logStats?.total ?? '-'}`,
    `阻断请求 ${logStats?.blocked ?? '-'}`,
    `转发请求 ${logStats?.proxied ?? '-'}`,
    `丢弃请求 ${logStats?.dropped ?? '-'}`,
  ]

  return (
    <div className="grid grid-2">
      <div className="card">
        <div className="card-header">
          <h3 className="card-title">实时日志</h3>
          <span className="pill">WebSocket</span>
        </div>
        <div className="log-console">
          {isLoading
            ? '日志加载中...'
            : logLines.map((line, index) => (index === logLines.length - 1 ? line : `${line}\n`))}
        </div>
      </div>
      <div className="card">
        <div className="card-header">
          <h3 className="card-title">日志分析</h3>
          <span className="pill">筛选器</span>
        </div>
        <div className="stack">
          <div className="form-row">
            <div className="form-field">
              <label className="label">时间范围</label>
              <select className="select">
                <option>最近 1 小时</option>
                <option>最近 24 小时</option>
                <option>最近 7 天</option>
              </select>
            </div>
            <div className="form-field">
              <label className="label">状态码</label>
              <select className="select">
                <option>全部</option>
                <option>2xx</option>
                <option>4xx</option>
                <option>5xx</option>
              </select>
            </div>
          </div>
          {isLoading ? (
            <div className="skeleton" />
          ) : (
            <div className="inline">
              <span className="badge">total {logStats?.total ?? '-'}</span>
              <span className="badge">blocked {logStats?.blocked ?? '-'}</span>
              <span className="badge">proxied {logStats?.proxied ?? '-'}</span>
              <span className="badge">dropped {logStats?.dropped ?? '-'}</span>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
