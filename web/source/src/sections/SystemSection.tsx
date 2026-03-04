type SystemSectionProps = {
  logLevel: string
  loadingLogLevel: boolean
  onLogLevelChange: (level: string) => void
  loadingReload: boolean
  onReload: () => void
}

const logLevels = ['debug', 'info', 'warn', 'error']

export function SystemSection({
  logLevel,
  loadingLogLevel,
  onLogLevelChange,
  loadingReload,
  onReload,
}: SystemSectionProps) {
  return (
    <>
      <div className="grid grid-2">
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">日志级别</h3>
            <span className="pill">GET/PUT /admin/api/v1/log/level</span>
          </div>
          <div className="inline">
            <select
              className="select"
              value={logLevel}
              onChange={(event) => onLogLevelChange(event.target.value)}
            >
              {logLevels.map((level) => (
                <option key={level} value={level}>
                  {level.toUpperCase()}
                </option>
              ))}
            </select>
            {loadingLogLevel && <span className="muted">同步中...</span>}
          </div>
        </div>
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">热重载</h3>
            <span className="pill">POST /admin/api/v1/config/reload</span>
          </div>
          <button className="button secondary" onClick={onReload} disabled={loadingReload}>
            {loadingReload ? '触发中...' : '触发热重载'}
          </button>
        </div>
      </div>
    </>
  )
}
