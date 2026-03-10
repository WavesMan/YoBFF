import { BsToggleOn, BsToggleOff } from 'react-icons/bs'
import type { Config } from '../../../../admin/types'

type RateLimitTabProps = {
  config: Config
  updateConfig: (updater: (prev: Config) => Config) => void
}

export function RateLimitTab({ config, updateConfig }: RateLimitTabProps) {
  return (
    <div className="panel">
      <h4>流量限制</h4>
      <div className="security-group">
        <div className="group-header">请求速率限制 (Rate Limit)</div>
        <div 
          className="toggle-item"
          style={{ marginBottom: '20px' }}
          onClick={() => updateConfig(prev => ({
            ...prev,
            rateLimit: { ...prev.rateLimit, enabled: !prev.rateLimit?.enabled }
          }))}
        >
          {config.rateLimit?.enabled ? (
            <BsToggleOn size={24} color="#10b981" />
          ) : (
            <BsToggleOff size={24} color="#9ca3af" />
          )}
          <span>启用全局速率限制</span>
        </div>
        
        {config.rateLimit?.enabled && (
          <div className="group-content" style={{ borderTop: '1px solid var(--border)', paddingTop: '20px' }}>
            <div className="form-field">
              <label>每秒请求数 (Requests/Second)</label>
              <input 
                type="number"
                className="input"
                value={config.rateLimit?.requestsPerSecond || 0}
                onChange={e => updateConfig(prev => ({
                  ...prev,
                  rateLimit: { ...prev.rateLimit, requestsPerSecond: parseInt(e.target.value) || 0 }
                }))}
              />
            </div>
            <div className="form-field">
              <label>突发流量 (Burst)</label>
              <input 
                type="number"
                className="input"
                value={config.rateLimit?.burst || 0}
                onChange={e => updateConfig(prev => ({
                  ...prev,
                  rateLimit: { ...prev.rateLimit, burst: parseInt(e.target.value) || 0 }
                }))}
              />
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
