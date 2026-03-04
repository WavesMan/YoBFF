import { useState } from 'react'
import type { Config, DomainRule } from '../admin/types'

type TrafficSectionProps = {
  configDraft: Config
  onConfigChange: (next: Config) => void
  onAddDomain: (rule: DomainRule) => void
  onUpdateDomain: (index: number, rule: DomainRule) => void
  onRemoveDomain: (index: number) => void
}

export function TrafficSection({
  configDraft,
  onConfigChange,
  onAddDomain,
  onUpdateDomain,
  onRemoveDomain,
}: TrafficSectionProps) {
  const [newDomain, setNewDomain] = useState({
    domain: '',
    upstream: '',
    forceHttps: true,
  })

  const handleAddDomain = () => {
    if (!newDomain.domain || !newDomain.upstream) {
      return
    }
    onAddDomain(newDomain)
    setNewDomain({ domain: '', upstream: '', forceHttps: true })
  }

  return (
    <div className="card">
      <div className="card-header">
        <h3 className="card-title">域名路由</h3>
        <span className="pill">GET/PUT /admin/api/v1/config</span>
      </div>
      <div className="stack">
        <div className="form-row">
          <div className="form-field">
            <label className="label">默认 Upstream</label>
            <input
              className="input"
              value={configDraft.routing?.defaultUpstream || ''}
              onChange={(event) =>
                onConfigChange({
                  ...configDraft,
                  routing: {
                    ...configDraft.routing,
                    defaultUpstream: event.target.value,
                  },
                })
              }
              placeholder="http://localhost:8080"
            />
          </div>
          <div className="form-field">
            <label className="label">Force HTTPS 默认开关</label>
            <select
              className="select"
              value={configDraft.dataPlane?.enableHttps ? 'true' : 'false'}
              onChange={(event) =>
                onConfigChange({
                  ...configDraft,
                  dataPlane: {
                    ...configDraft.dataPlane,
                    enableHttps: event.target.value === 'true',
                  },
                })
              }
            >
              <option value="true">启用</option>
              <option value="false">关闭</option>
            </select>
          </div>
        </div>
        <div className="divider" />
        <table className="table">
          <thead>
            <tr>
              <th style={{ width: '30%' }}>域名</th>
              <th style={{ width: '40%' }}>上游地址</th>
              <th style={{ width: '15%' }}>HTTPS</th>
              <th style={{ width: '15%', textAlign: 'right' }}>操作</th>
            </tr>
          </thead>
          <tbody>
            {(configDraft.routing?.domains || []).map((rule, index) => (
              <tr key={`${rule.domain}-${index}`}>
                <td>
                  <input
                    className="input"
                    value={rule.domain || ''}
                    onChange={(event) =>
                      onUpdateDomain(index, {
                        ...rule,
                        domain: event.target.value,
                      })
                    }
                  />
                </td>
                <td>
                  <input
                    className="input"
                    value={rule.upstream || ''}
                    onChange={(event) =>
                      onUpdateDomain(index, {
                        ...rule,
                        upstream: event.target.value,
                      })
                    }
                  />
                </td>
                <td>
                  <select
                    className="select"
                    value={rule.forceHttps ? 'true' : 'false'}
                    onChange={(event) =>
                      onUpdateDomain(index, {
                        ...rule,
                        forceHttps: event.target.value === 'true',
                      })
                    }
                  >
                    <option value="true">开启</option>
                    <option value="false">关闭</option>
                  </select>
                </td>
                <td style={{ textAlign: 'right' }}>
                  <button className="button danger" onClick={() => onRemoveDomain(index)}>
                    删除
                  </button>
                </td>
              </tr>
            ))}
            {!configDraft.routing?.domains?.length && (
              <tr>
                <td colSpan={4} className="muted">
                  暂无路由规则
                </td>
              </tr>
            )}
          </tbody>
        </table>
        <div className="form-row">
          <div className="form-field">
            <label className="label">新增域名</label>
            <input
              className="input"
              value={newDomain.domain}
              onChange={(event) => setNewDomain((prev) => ({ ...prev, domain: event.target.value }))}
            />
          </div>
          <div className="form-field">
            <label className="label">新增上游</label>
            <input
              className="input"
              value={newDomain.upstream}
              onChange={(event) => setNewDomain((prev) => ({ ...prev, upstream: event.target.value }))}
            />
          </div>
        </div>
        <div className="inline">
          <select
            className="select"
            value={newDomain.forceHttps ? 'true' : 'false'}
            onChange={(event) => setNewDomain((prev) => ({ ...prev, forceHttps: event.target.value === 'true' }))}
          >
            <option value="true">强制 HTTPS</option>
            <option value="false">不强制</option>
          </select>
          <button className="button secondary" onClick={handleAddDomain}>
            新增规则
          </button>
        </div>
      </div>
    </div>
  )
}
