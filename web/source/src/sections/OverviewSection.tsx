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
            <div className="inline">
              <span className="status-dot status-warning" />
              限流触发，建议检查突发流量或提升策略
            </div>
            <div className="inline">
              <span className="status-dot status-error" />
              Auth 未配置，登录将无法生效
            </div>
            <div className="inline">
              <span className="status-dot status-success" />
              控制面心跳正常
            </div>
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
