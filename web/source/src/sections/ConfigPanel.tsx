import type { Config, ConfigVersion, ValidationIssue } from '../admin/types'

type ConfigView = 'apply' | 'dry-run' | 'rollback'

type ConfigPanelProps = {
  configDraft: Config
  configSnapshot: Config | null
  advancedMode: boolean
  jsonDraft: string
  loadingApplying: boolean
  loadingValidating: boolean
  loadingVersions: boolean
  view: ConfigView
  operator: string
  validationIssues: ValidationIssue[]
  validationTime: string
  configVersions: ConfigVersion[]
  diffCurrentJson: string
  diffDraftJson: string
  diffChanged: boolean
  onViewChange: (view: ConfigView) => void
  onOperatorChange: (value: string) => void
  onToggleMode: (isAdvanced: boolean) => void
  onJsonChange: (value: string) => void
  onConfigChange: (next: Config) => void
  onApply: () => void
  onReset: () => void
  onValidate: () => void
  onRefreshVersions: () => void
  onRollback: (versionID: string) => void
}

export function ConfigPanel({
  configDraft,
  configSnapshot,
  advancedMode,
  jsonDraft,
  loadingApplying,
  loadingValidating,
  loadingVersions,
  view,
  operator,
  validationIssues,
  validationTime,
  configVersions,
  diffCurrentJson,
  diffDraftJson,
  diffChanged,
  onViewChange,
  onOperatorChange,
  onToggleMode,
  onJsonChange,
  onConfigChange,
  onApply,
  onReset,
  onValidate,
  onRefreshVersions,
  onRollback,
}: ConfigPanelProps) {
  return (
    <div className="card">
      <div className="card-header">
        <div>
          <h3 className="card-title">配置管理</h3>
          <div className="muted">配置预检、差异比对与版本回滚入口</div>
        </div>
        <span className="pill">/admin/api/v1/config</span>
      </div>
      <div className="inline">
        <button
          className={`button ${view === 'apply' ? 'primary' : 'secondary'}`}
          onClick={() => onViewChange('apply')}
        >
          配置应用
        </button>
        <button
          className={`button ${view === 'dry-run' ? 'primary' : 'secondary'}`}
          onClick={() => onViewChange('dry-run')}
        >
          预检与差异
        </button>
        <button
          className={`button ${view === 'rollback' ? 'primary' : 'secondary'}`}
          onClick={() => onViewChange('rollback')}
        >
          版本回滚
        </button>
      </div>
      <div className="form-row">
        <div className="form-field">
          <label className="label">操作人</label>
          <input
            className="input"
            value={operator}
            onChange={(event) => onOperatorChange(event.target.value)}
            placeholder="operator"
          />
        </div>
      </div>
      <div className="divider" />
      {view === 'apply' && (
        <div className="stack">
          <div className="inline">
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
      )}
      {view === 'dry-run' && (
        <div className="stack">
          <div className="inline">
            <span className="pill">POST /admin/api/v1/config/validate</span>
            <button className="button primary" onClick={onValidate} disabled={loadingValidating}>
              {loadingValidating ? '预检中...' : '执行预检'}
            </button>
          </div>
          {validationTime && (
            <div className="muted">最近预检：{validationTime}</div>
          )}
          {validationIssues.length > 0 ? (
            <div className="stack">
              <div className="error-banner">预检失败，共 {validationIssues.length} 项</div>
              <table className="table">
                <thead>
                  <tr>
                    <th style={{ width: '40%' }}>路径</th>
                    <th>说明</th>
                  </tr>
                </thead>
                <tbody>
                  {validationIssues.map((issue, index) => (
                    <tr key={`${issue.path}-${index}`}>
                      <td>{issue.path}</td>
                      <td>{issue.message}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            validationTime && <div className="success-banner">预检通过</div>
          )}
          <div className="inline">
            <span className="pill">Diff</span>
            <span className="muted">{diffChanged ? '检测到配置差异' : '当前与待应用配置一致'}</span>
          </div>
          <div className="diff-grid">
            <div className="diff-box">
              <div className="muted">当前配置</div>
              <pre className="log-console">{diffCurrentJson}</pre>
            </div>
            <div className="diff-box">
              <div className="muted">待应用配置</div>
              <pre className="log-console">{diffDraftJson}</pre>
            </div>
          </div>
        </div>
      )}
      {view === 'rollback' && (
        <div className="stack">
          <div className="inline">
            <span className="pill">GET /admin/api/v1/config/versions</span>
            <span className="pill">POST /admin/api/v1/config/rollback</span>
            <button className="button secondary" onClick={onRefreshVersions} disabled={loadingVersions}>
              {loadingVersions ? '刷新中...' : '刷新列表'}
            </button>
          </div>
          <table className="table">
            <thead>
              <tr>
                <th style={{ width: '22%' }}>时间</th>
                <th style={{ width: '18%' }}>操作人</th>
                <th style={{ width: '18%' }}>来源</th>
                <th>版本 ID</th>
                <th style={{ width: '14%', textAlign: 'right' }}>操作</th>
              </tr>
            </thead>
            <tbody>
              {configVersions.map((item) => (
                <tr key={item.id}>
                  <td>{item.created_at}</td>
                  <td>{item.operator || '-'}</td>
                  <td>{item.source || '-'}</td>
                  <td>{item.id}</td>
                  <td style={{ textAlign: 'right' }}>
                    <button className="button secondary" onClick={() => onRollback(item.id)}>
                      回滚
                    </button>
                  </td>
                </tr>
              ))}
              {!configVersions.length && (
                <tr>
                  <td colSpan={5} className="muted">
                    暂无历史版本
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
