import { FiTrash2 } from 'react-icons/fi'
import type { Config, DomainRule } from '../../../../admin/types'

type ProxyRulesTabProps = {
  config: Config
  updateConfig: (updater: (prev: Config) => Config) => void
  newDomain: DomainRule
  setNewDomain: (rule: DomainRule) => void
  handleAddDomain: () => void
  handleRemoveDomain: (index: number) => void
}

export function ProxyRulesTab({
  config,
  updateConfig,
  newDomain,
  setNewDomain,
  handleAddDomain,
  handleRemoveDomain,
}: ProxyRulesTabProps) {
  return (
    <div className="panel">
      <h4>默认上游</h4>
      <div className="form-field">
        <label>Default Upstream</label>
        <input 
          className="input" 
          value={config.routing?.defaultUpstream || ''}
          onChange={(e) => updateConfig(prev => ({
            ...prev,
            routing: { ...prev.routing, defaultUpstream: e.target.value }
          }))}
          placeholder="http://localhost:8080"
        />
      </div>
      
      <div className="divider" />
      
      <h4>路由规则</h4>
      <table className="table">
        <thead>
          <tr>
            <th>域名</th>
            <th>上游地址</th>
            <th>HTTPS</th>
            <th style={{ textAlign: 'right' }}>操作</th>
          </tr>
        </thead>
        <tbody>
          {(config.routing?.domains || []).map((rule, index) => (
            <tr key={index}>
              <td>
                <input 
                  className="input small" 
                  value={rule.domain} 
                  onChange={e => {
                    const newDomains = [...(config.routing?.domains || [])]
                    newDomains[index] = { ...rule, domain: e.target.value }
                    updateConfig(prev => ({ ...prev, routing: { ...prev.routing, domains: newDomains } }))
                  }}
                />
              </td>
              <td>
                <input 
                  className="input small" 
                  value={rule.upstream} 
                  onChange={e => {
                    const newDomains = [...(config.routing?.domains || [])]
                    newDomains[index] = { ...rule, upstream: e.target.value }
                    updateConfig(prev => ({ ...prev, routing: { ...prev.routing, domains: newDomains } }))
                  }}
                />
              </td>
              <td>
                <select 
                  className="select small"
                  value={rule.forceHttps ? 'true' : 'false'}
                  onChange={e => {
                    const newDomains = [...(config.routing?.domains || [])]
                    newDomains[index] = { ...rule, forceHttps: e.target.value === 'true' }
                    updateConfig(prev => ({ ...prev, routing: { ...prev.routing, domains: newDomains } }))
                  }}
                >
                  <option value="true">开启</option>
                  <option value="false">关闭</option>
                </select>
              </td>
              <td style={{ textAlign: 'right' }}>
                <button className="button danger small" onClick={() => handleRemoveDomain(index)}>
                  <FiTrash2 />
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      
      <div className="form-row" style={{ marginTop: 15 }}>
        <div className="form-field">
          <input 
            className="input" 
            placeholder="新增域名"
            value={newDomain.domain}
            onChange={e => setNewDomain({ ...newDomain, domain: e.target.value })}
          />
        </div>
        <div className="form-field">
          <input 
            className="input" 
            placeholder="上游地址"
            value={newDomain.upstream}
            onChange={e => setNewDomain({ ...newDomain, upstream: e.target.value })}
          />
        </div>
        <button className="button secondary" onClick={handleAddDomain}>添加</button>
      </div>
    </div>
  )
}
