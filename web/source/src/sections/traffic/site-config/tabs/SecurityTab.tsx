import { FiChevronDown, FiChevronRight, FiX, FiCheck } from 'react-icons/fi'
import { BsToggleOn, BsToggleOff } from 'react-icons/bs'
import type { Config, Site, SSLCertificate } from '../../../../admin/types'

type SecurityTabProps = {
  site: Site | null
  config: Config
  updateConfig: (updater: (prev: Config) => Config) => void
  filteredCerts: SSLCertificate[]
  showCertSelector: boolean
  setShowCertSelector: (show: boolean) => void
  cdnExpanded: boolean
  setCdnExpanded: (expanded: boolean) => void
}

export function SecurityTab({
  site,
  config,
  updateConfig,
  filteredCerts,
  showCertSelector,
  setShowCertSelector,
  cdnExpanded,
  setCdnExpanded,
}: SecurityTabProps) {
  return (
    <div className="panel">
      <h4>安全防护</h4>
      
      {/* HTTPS & SSL Group */}
      <div className="security-group">
        <div className="group-header">
          HTTPS 与 SSL 配置
        </div>
        <div className="toggle-row">
          <div 
            className="toggle-item"
            onClick={() => {
              // Toggle HTTPS logic
              if (config.dataPlane?.enableHttps) {
                // Disable HTTPS
                updateConfig(prev => ({
                  ...prev,
                  dataPlane: { ...prev.dataPlane, enableHttps: false }
                }))
              } else {
                // Enable HTTPS - Require certificate selection
                setShowCertSelector(true)
              }
            }}
          >
            {config.dataPlane?.enableHttps ? (
              <BsToggleOn size={24} color="#10b981" />
            ) : (
              <BsToggleOff size={24} color="#9ca3af" />
            )}
            <span>启用全局 HTTPS</span>
          </div>
          
          {/* Certificate Selection Modal/Area */}
          {showCertSelector && (
            <div className="modal-overlay" onClick={() => setShowCertSelector(false)}>
              <div className="modal-content" onClick={e => e.stopPropagation()}>
                <div className="modal-header">
                  <h4>选择 SSL 证书</h4>
                  <button className="icon-button" onClick={() => setShowCertSelector(false)}>
                    <FiX />
                  </button>
                </div>
                <div className="modal-body">
                  <p className="muted" style={{ marginBottom: '15px' }}>
                    请为 {site?.hostname} 选择一个匹配的 SSL 证书以启用 HTTPS。
                  </p>
                  
                  <div className="cert-list">
                    {filteredCerts.length === 0 ? (
                      <div className="empty-state">
                        没有找到匹配的证书，请先在证书管理中上传。
                      </div>
                    ) : (
                      filteredCerts.map(cert => (
                        <div 
                          key={cert.id} 
                          className={`cert-item ${config.dataPlane?.certId === cert.id ? 'selected' : ''}`}
                          onClick={() => {
                            updateConfig(prev => ({
                              ...prev,
                              dataPlane: { 
                                ...prev.dataPlane, 
                                enableHttps: true,
                                certId: cert.id 
                              }
                            }))
                            setShowCertSelector(false)
                          }}
                        >
                          <div className="cert-info">
                            <div className="cert-name">{cert.name}</div>
                            <div className="cert-domains">
                              {cert.domains.join(', ')}
                            </div>
                          </div>
                          {config.dataPlane?.certId === cert.id && <FiCheck />}
                        </div>
                      ))
                    )}
                  </div>
                </div>
              </div>
            </div>
          )}

          <div 
            className="toggle-item"
            onClick={() => updateConfig(prev => ({
              ...prev,
              security: { ...prev.security, enableHsts: !prev.security?.enableHsts }
            }))}
          >
            {config.security?.enableHsts ? (
              <BsToggleOn size={24} color="#10b981" />
            ) : (
              <BsToggleOff size={24} color="#9ca3af" />
            )}
            <span>启用 HSTS (强制跳转)</span>
          </div>
        </div>
      </div>

      {/* CDN Origin Protection Group - Collapsible under HTTPS context as requested */}
      <div className="security-group">
        <div 
          className="group-header collapsible" 
          onClick={() => setCdnExpanded(!cdnExpanded)}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            {cdnExpanded ? <FiChevronDown /> : <FiChevronRight />}
            <span>CDN 回源来源防护</span>
          </div>
          <div className="badge">
            {(config.security?.allowedCdnProviders || []).length} 已启用
          </div>
        </div>
        
        {cdnExpanded && (
          <div className="group-content">
            <div className="muted" style={{ marginBottom: '15px', fontSize: '13px' }}>
              启用后，网关将仅允许来自所选 CDN 厂商的回源 IP 访问。请确保您已在对应 CDN 控制台配置回源策略。
              对于需要鉴权回源的厂商（如 Cloudflare Authenticated Origin Pulls），请填写对应 API Key 或证书信息。
            </div>
            
            <div className="cdn-grid">
              {['aliyun', 'tencent', 'cloudflare'].map(provider => {
                const isEnabled = (config.security?.allowedCdnProviders || []).includes(provider)
                const settings = config.security?.cdnProviderSettings?.[provider] || {}
                
                return (
                  <div key={provider} className={`cdn-card ${isEnabled ? 'active' : ''}`}>
                    <div className="card-header">
                      <span className="provider-name">{provider.toUpperCase()}</span>
                      <div 
                        className="toggle-btn"
                        onClick={(e) => {
                          e.stopPropagation()
                          const current = config.security?.allowedCdnProviders || []
                          const next = isEnabled
                            ? current.filter(p => p !== provider)
                            : [...current, provider]
                          
                          updateConfig(prev => ({
                            ...prev,
                            security: { ...prev.security, allowedCdnProviders: next }
                          }))
                        }}
                      >
                        {isEnabled ? (
                          <BsToggleOn size={24} color="#10b981" />
                        ) : (
                          <BsToggleOff size={24} color="#9ca3af" />
                        )}
                      </div>
                    </div>
                    
                    {isEnabled && (
                      <div className="card-body">
                        <div className="form-field small">
                          <label>API Key / Token</label>
                          <input 
                            className="input small"
                            placeholder="可选配置"
                            value={settings.apiKey || ''}
                            onChange={e => {
                              const val = e.target.value
                              updateConfig(prev => {
                                const currentSettings = prev.security?.cdnProviderSettings || {}
                                return {
                                  ...prev,
                                  security: {
                                    ...prev.security,
                                    cdnProviderSettings: {
                                      ...currentSettings,
                                      [provider]: { ...currentSettings[provider], apiKey: val }
                                    }
                                  }
                                }
                              })
                            }}
                          />
                        </div>
                        <div className="form-field small">
                          <label>Option / Secret</label>
                          <input 
                            className="input small"
                            placeholder="可选配置"
                            value={settings.secretKey || ''}
                            onChange={e => {
                              const val = e.target.value
                              updateConfig(prev => {
                                const currentSettings = prev.security?.cdnProviderSettings || {}
                                return {
                                  ...prev,
                                  security: {
                                    ...prev.security,
                                    cdnProviderSettings: {
                                      ...currentSettings,
                                      [provider]: { ...currentSettings[provider], secretKey: val }
                                    }
                                  }
                                }
                              })
                            }}
                          />
                        </div>
                      </div>
                    )}
                  </div>
                )
              })}
            </div>

            <div style={{ marginTop: '20px', borderTop: '1px solid var(--border)', paddingTop: '20px' }}>
              <div className="form-field">
                <label style={{ fontWeight: 600, marginBottom: '8px', display: 'block' }}>自定义 IP 白名单 (全局规则)</label>
                <div className="muted" style={{ marginBottom: '10px', fontSize: '13px' }}>
                  在此处配置的 IP/CIDR 将作为额外规则放行（不受 CDN 限制影响）。
                  <br />
                  留空或配置 0.0.0.0/0 (IPv4) / :: (IPv6) 表示不设置额外放行规则，完全跟随上方 CDN 回源配置。
                </div>
                <textarea 
                  className="textarea"
                  rows={5}
                  placeholder="192.168.1.0/24"
                  value={(config.security?.allowedCidrs || []).join('\n')}
                  onChange={e => updateConfig(prev => ({
                    ...prev,
                    security: { 
                      ...prev.security, 
                      allowedCidrs: e.target.value.split('\n').map(l => l.trim()).filter(Boolean)
                    }
                  }))}
                />
              </div>
            </div>

            <div style={{ marginTop: '20px', borderTop: '1px solid var(--border)', paddingTop: '20px' }}>
              <div className="form-field">
                <label style={{ fontWeight: 600, marginBottom: '8px', display: 'block' }}>自定义拦截页面 (Block Page HTML)</label>
                <div className="muted" style={{ marginBottom: '10px', fontSize: '13px' }}>
                  当请求被安全策略拦截时显示的页面内容。支持 HTML。留空则使用默认拦截页面。
                </div>
                <textarea 
                  className="textarea"
                  rows={8}
                  placeholder="<html>...</html>"
                  value={config.security?.blockPageHtml || ''}
                  onChange={e => updateConfig(prev => ({
                    ...prev,
                    security: { 
                      ...prev.security, 
                      blockPageHtml: e.target.value
                    }
                  }))}
                />
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
