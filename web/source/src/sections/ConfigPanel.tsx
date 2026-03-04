import type { Config } from '../admin/types'

type ConfigPanelProps = {
  configDraft: Config
  configSnapshot: Config | null
  advancedMode: boolean
  jsonDraft: string
  loadingApplying: boolean
  onToggleMode: (isAdvanced: boolean) => void
  onJsonChange: (value: string) => void
  onConfigChange: (next: Config) => void
  onApply: () => void
  onReset: () => void
}

export function ConfigPanel({
  configDraft,
  configSnapshot,
  advancedMode,
  jsonDraft,
  loadingApplying,
  onToggleMode,
  onJsonChange,
  onConfigChange,
  onApply,
  onReset,
}: ConfigPanelProps) {
  return (
    <div className="card">
      <div className="card-header">
        <h3 className="card-title">配置应用</h3>
        <span className="pill">PUT /admin/api/v1/config</span>
      </div>
      <div className="inline">
        <button
          className={`button ${advancedMode ? 'secondary' : 'primary'}`}
          onClick={() => onToggleMode(false)}
        >
          基础模式
        </button>
        <button
          className={`button ${advancedMode ? 'primary' : 'secondary'}`}
          onClick={() => onToggleMode(true)}
        >
          高级 JSON
        </button>
      </div>
      <div className="divider" />
      {advancedMode ? (
        <textarea className="textarea" value={jsonDraft} onChange={(event) => onJsonChange(event.target.value)} />
      ) : (
        <div className="stack">
          <div className="form-row">
            <div className="form-field">
              <label className="label">HTTP 监听地址</label>
              <input
                className="input"
                value={configDraft.dataPlane?.httpListenAddr || ''}
                onChange={(event) =>
                  onConfigChange({
                    ...configDraft,
                    dataPlane: {
                      ...configDraft.dataPlane,
                      httpListenAddr: event.target.value,
                    },
                  })
                }
              />
            </div>
            <div className="form-field">
              <label className="label">HTTPS 监听地址</label>
              <input
                className="input"
                value={configDraft.dataPlane?.httpsListenAddr || ''}
                onChange={(event) =>
                  onConfigChange({
                    ...configDraft,
                    dataPlane: {
                      ...configDraft.dataPlane,
                      httpsListenAddr: event.target.value,
                    },
                  })
                }
              />
            </div>
          </div>
          <div className="form-row">
            <div className="form-field">
              <label className="label">控制面地址</label>
              <input
                className="input"
                value={configDraft.controlPlane?.adminListenAddr || ''}
                onChange={(event) =>
                  onConfigChange({
                    ...configDraft,
                    controlPlane: {
                      ...configDraft.controlPlane,
                      adminListenAddr: event.target.value,
                    },
                  })
                }
              />
            </div>
            <div className="form-field">
              <label className="label">HSTS</label>
              <select
                className="select"
                value={configDraft.security?.enableHsts ? 'true' : 'false'}
                onChange={(event) =>
                  onConfigChange({
                    ...configDraft,
                    security: {
                      ...configDraft.security,
                      enableHsts: event.target.value === 'true',
                    },
                  })
                }
              >
                <option value="true">启用</option>
                <option value="false">关闭</option>
              </select>
            </div>
          </div>
        </div>
      )}
      <div className="divider" />
      <div className="inline">
        <button className="button primary" onClick={onApply} disabled={loadingApplying}>
          {loadingApplying ? '应用中...' : '应用配置'}
        </button>
        <button
          className="button secondary"
          onClick={() => {
            if (configSnapshot) {
              onReset()
            }
          }}
        >
          恢复
        </button>
      </div>
    </div>
  )
}
