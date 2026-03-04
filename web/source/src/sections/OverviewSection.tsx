import type { HealthzResponse, LogStats } from '../admin/types'

type OverviewSectionProps = {
  health: HealthzResponse | null
  logStats: LogStats | null
  logLevel: string
  loadingHealth: boolean
  loadingLogStats: boolean
}

export function OverviewSection({
  health,
  logStats,
  logLevel,
  loadingHealth,
  loadingLogStats,
}: OverviewSectionProps) {
  const healthStatus = health?.status === 'ok' ? 'success' : 'warning'
  const alertItems = []
  if (health && health.status !== 'ok') {
    alertItems.push({ level: 'error', text: '健康检查异常' })
  }
  if ((logStats?.blocked ?? 0) > 0) {
    alertItems.push({
      level: 'warning',
      text: `触发限流 ${logStats?.blocked ?? 0} 次`,
    })
  }
  if ((logStats?.dropped ?? 0) > 0) {
    alertItems.push({
      level: 'error',
      text: `请求丢弃 ${logStats?.dropped ?? 0} 次`,
    })
  }
  if (alertItems.length === 0) {
    alertItems.push({ level: 'success', text: '暂无告警' })
  }

  return (
    <>
      <div className="grid grid-3">
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">服务健康度</h3>
            <span className="badge">
              <span className={`status-dot status-${healthStatus}`} />
              {health?.status || 'unknown'}
            </span>
          </div>
          {loadingHealth ? <div className="skeleton" /> : <div className="stat-value">{health?.status || 'unknown'}</div>}
          <div className="stat-label">GET /healthz</div>
        </div>
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">日志管线统计</h3>
            <span className="pill">GET /admin/api/v1/log/stats</span>
          </div>
          {loadingLogStats ? (
            <div className="stack">
              <div className="skeleton" />
              <div className="skeleton" />
            </div>
          ) : (
            <div className="stack">
              <div className="stat-value">{logStats?.total ?? '-'}</div>
              <div className="inline">
                <span className="badge">blocked {logStats?.blocked ?? '-'}</span>
                <span className="badge">proxied {logStats?.proxied ?? '-'}</span>
                <span className="badge">dropped {logStats?.dropped ?? '-'}</span>
              </div>
            </div>
          )}
        </div>
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">运行策略</h3>
            <span className="pill">动态生效</span>
          </div>
          <div className="stack">
            <div className="inline">
              <span className="badge">日志级别</span>
              <span className="stat-value">{logLevel.toUpperCase()}</span>
            </div>
            <div className="muted">支持 Debug/Info/Warn/Error 即时切换</div>
          </div>
        </div>
      </div>
      <div className="grid grid-2">
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">告警摘要</h3>
            <span className="pill">近 5 条</span>
          </div>
          <div className="stack">
            {alertItems.map((item, index) => (
              <div className="inline" key={`${item.level}-${index}`}>
                <span className={`status-dot status-${item.level}`} />
                {item.text}
              </div>
            ))}
          </div>
        </div>
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">趋势概览</h3>
            <span className="pill">过去 1 小时</span>
          </div>
          <div className="stack">
            <div className="muted">图表区域已预留，可接入 Recharts 或 ECharts</div>
            <div className="skeleton" style={{ height: 120 }} />
          </div>
        </div>
      </div>
    </>
  )
}
