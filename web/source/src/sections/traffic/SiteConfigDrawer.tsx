import { useCallback, useEffect, useState } from 'react'
import {
  FiActivity,
  FiClock,
  FiGlobe,
  FiLock,
  FiSave,
  FiSettings,
  FiShield,
  FiTrash2,
  FiX,
  FiZap,
  FiChevronDown,
  FiChevronRight,
  FiCheck
} from 'react-icons/fi'
import { BsToggleOn, BsToggleOff } from 'react-icons/bs'
import {
  fetchSite,
  fetchSiteConfig,
  fetchSiteVersions,
  rollbackSiteConfig,
  updateSiteConfig,
  validateConfig,
  fetchSiteLogStream,
  updateSiteLogStream,
  fetchCertificates
} from '../../admin/api'
import type { Config, ConfigVersion, DomainRule, Site, SiteLogStream, SSLCertificate } from '../../admin/types'

type SiteConfigDrawerProps = {
  token: string
  operator: string
  siteId: string
  onClose: () => void
}

export function SiteConfigDrawer({ token, operator, siteId, onClose }: SiteConfigDrawerProps) {
  const [activeTab, setActiveTab] = useState<'basic' | 'proxy' | 'security' | 'https' | 'limit' | 'version' | 'log'>('basic')
  const [site, setSite] = useState<Site | null>(null)
  const [config, setConfig] = useState<Config | null>(null)
  const [versions, setVersions] = useState<ConfigVersion[]>([])
  const [logStream, setLogStream] = useState<SiteLogStream | null>(null)
  const [logs, setLogs] = useState<string[]>([])
  const [isLogConnected, setIsLogConnected] = useState(false)
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  // New Domain Rule State
  const [newDomain, setNewDomain] = useState<DomainRule>({
    domain: '',
    upstream: '',
    forceHttps: true
  })
  

  const [cdnExpanded, setCdnExpanded] = useState(true)
  const [certs, setCerts] = useState<SSLCertificate[]>([])
  const [showCertSelector, setShowCertSelector] = useState(false)

  // Load available certificates when drawer opens or on demand
  const loadCertificates = useCallback(async () => {
    try {
      const { items } = await fetchCertificates(token, 1, 100) // Load all for selection
      setCerts(items || [])
    } catch (e) {
      console.error("Failed to load certificates", e)
    }
  }, [token])

  useEffect(() => {
    if (activeTab === 'security') {
      loadCertificates()
    }
  }, [activeTab, loadCertificates])

  // Filter certificates based on site hostname
  const filteredCerts = certs.filter(cert => {
    if (!site?.hostname || site.hostname.match(/^(\d{1,3}\.){3}\d{1,3}$/) || site.hostname.includes(':')) {
      return true // Show all if IP or no hostname
    }
    const host = site.hostname.toLowerCase()
    // Simple domain matching: cert domain should cover site hostname
    // Supports wildcards *.example.com matches sub.example.com
    return (cert.domains || []).some((d: string) => {
      const domain = d.toLowerCase()
      if (domain.startsWith('*.')) {
        const suffix = domain.substring(2)
        return host.endsWith(suffix) && host.split('.').length === suffix.split('.').length + 1
      }
      return domain === host
    })
  })

  useEffect(() => {
    let interval: ReturnType<typeof setInterval>
    if (isLogConnected && activeTab === 'log') {
      interval = setInterval(() => {
        const timestamp = new Date().toISOString()
        const methods = ['GET', 'POST', 'PUT', 'DELETE']
        const paths = ['/api/v1/users', '/login', '/dashboard', '/assets/style.css', '/api/v1/sites']
        const status = [200, 201, 400, 401, 403, 404, 500]
        
        const method = methods[Math.floor(Math.random() * methods.length)]
        const path = paths[Math.floor(Math.random() * paths.length)]
        const code = status[Math.floor(Math.random() * status.length)]
        const duration = Math.floor(Math.random() * 200) + 10
        
        const logLine = `[${timestamp}] ${method} ${path} ${code} - ${duration}ms`
        
        setLogs(prev => [logLine, ...prev].slice(0, 100))
      }, 1000)
    }
    return () => clearInterval(interval)
  }, [isLogConnected, activeTab])

  const loadSite = useCallback(async () => {
    try {
      const data = await fetchSite(token, siteId)
      setSite(data)
    } catch (err) {
      console.error('Failed to load site info', err)
    }
  }, [token, siteId])

  const loadConfig = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const data = await fetchSiteConfig(token, siteId)
      setConfig(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载配置失败')
    } finally {
      setLoading(false)
    }
  }, [token, siteId])

  const loadVersions = useCallback(async () => {
    try {
      const data = await fetchSiteVersions(token, siteId)
      setVersions(data.items || [])
    } catch (err) {
      console.error('Failed to load versions', err)
    }
  }, [token, siteId])

  const loadLogStream = useCallback(async () => {
    try {
      const data = await fetchSiteLogStream(token, siteId)
      setLogStream(data)
    } catch (err) {
      // Ignore error if log stream config doesn't exist yet
      console.log('Log stream config not found or error', err)
    }
  }, [token, siteId])

  useEffect(() => {
    if (siteId) {
      loadConfig()
      loadSite()
    }
    return () => {
      setIsLogConnected(false)
      setLogs([])
    }
  }, [siteId, loadConfig, loadSite])

  useEffect(() => {
    if (activeTab === 'version' && siteId) {
      loadVersions()
    }
    if (activeTab === 'log' && siteId) {
      loadLogStream()
    }
  }, [activeTab, siteId, loadVersions, loadLogStream])

  const handleSave = async () => {
    if (!config) return
    setSaving(true)
    setError('')
    setSuccess('')
    try {
      // Validate first
      const validation = await validateConfig(token, config, operator)
      if (!validation.valid) {
        setError('配置验证失败: ' + validation.errors.map(e => e.message).join(', '))
        return
      }
      
      await updateSiteConfig(token, siteId, config, operator)
      setSuccess('配置已保存并生效')
      setTimeout(() => setSuccess(''), 3000)
      if (activeTab === 'version') {
        loadVersions()
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存配置失败')
    } finally {
      setSaving(false)
    }
  }

  const handleRollback = async (versionId: string) => {
    if (!window.confirm('确认回滚到此版本吗？当前未保存的更改将丢失。')) return
    
    setLoading(true)
    try {
      await rollbackSiteConfig(token, siteId, versionId, operator)
      await loadConfig()
      setSuccess(`已回滚到版本 ${versionId}`)
      loadVersions()
    } catch (err) {
      setError(err instanceof Error ? err.message : '回滚失败')
    } finally {
      setLoading(false)
    }
  }
  
  const handleUpdateLogStream = async () => {
    if (!logStream) return
    try {
      await updateSiteLogStream(token, siteId, logStream.filter_query)
      setSuccess('日志过滤规则已更新')
      setTimeout(() => setSuccess(''), 3000)
    } catch (err) {
      setError(err instanceof Error ? err.message : '更新日志规则失败')
    }
  }

  // Config Update Helpers
  const updateConfig = (updater: (prev: Config) => Config) => {
    if (!config) return
    setConfig(updater(config))
  }

  const handleAddDomain = () => {
    if (!newDomain.domain || !newDomain.upstream) return
    updateConfig(prev => ({
      ...prev,
      routing: {
        ...prev.routing,
        domains: [...(prev.routing?.domains || []), newDomain]
      }
    }))
    setNewDomain({ domain: '', upstream: '', forceHttps: true })
  }

  const handleRemoveDomain = (index: number) => {
    updateConfig(prev => ({
      ...prev,
      routing: {
        ...prev.routing,
        domains: (prev.routing?.domains || []).filter((_, i) => i !== index)
      }
    }))
  }

  
  const handleRemoveCert = (index: number) => {
    updateConfig(prev => ({
      ...prev,
      certificates: (prev.certificates || []).filter((_, i) => i !== index)
    }))
  }

  if (!siteId) return null

  return (
    <>
      <div className={`drawer-overlay ${siteId ? 'open' : ''}`} onClick={onClose} />
      <div className={`drawer ${siteId ? 'open' : ''}`}>
        <div className="drawer-header">
        <div className="header-left">
          <button className="icon-btn" onClick={onClose}>
            <FiX />
          </button>
          <h3>配置中心: {site?.name || siteId}</h3>
        </div>
        <div className="header-right">
          <button 
            className="button primary" 
            onClick={handleSave} 
            disabled={saving || loading}
          >
            <FiSave /> {saving ? '保存中...' : '保存配置'}
          </button>
        </div>
      </div>
      
      <div className="drawer-body">
        <div className="config-sidebar">
          <div 
            className={`config-menu-item ${activeTab === 'basic' ? 'active' : ''}`}
            onClick={() => setActiveTab('basic')}
          >
            <FiSettings /> 基础设置
          </div>
          <div 
            className={`config-menu-item ${activeTab === 'proxy' ? 'active' : ''}`}
            onClick={() => setActiveTab('proxy')}
          >
            <FiGlobe /> 反向代理
          </div>
          <div 
            className={`config-menu-item ${activeTab === 'https' ? 'active' : ''}`}
            onClick={() => setActiveTab('https')}
          >
            <FiLock /> HTTPS 证书
          </div>
          <div 
            className={`config-menu-item ${activeTab === 'security' ? 'active' : ''}`}
            onClick={() => setActiveTab('security')}
          >
            <FiShield /> 安全防护
          </div>
          <div 
            className={`config-menu-item ${activeTab === 'limit' ? 'active' : ''}`}
            onClick={() => setActiveTab('limit')}
          >
            <FiZap /> 流量限制
          </div>
          <div 
            className={`config-menu-item ${activeTab === 'version' ? 'active' : ''}`}
            onClick={() => setActiveTab('version')}
          >
            <FiClock /> 版本控制
          </div>
          <div 
            className={`config-menu-item ${activeTab === 'log' ? 'active' : ''}`}
            onClick={() => setActiveTab('log')}
          >
            <FiActivity /> 实时日志
          </div>
        </div>
        
        <div className="config-content">
          {loading && <div className="loading">加载配置中...</div>}
          {error && <div className="alert error">{error}</div>}
          {success && <div className="alert success">{success}</div>}
          
          {!loading && config && (
            <>
              {activeTab === 'basic' && (
                <div className="panel">
                  <h4>基础设置</h4>
                  <div className="form-field">
                    <label>站点 ID</label>
                    <input className="input" defaultValue={siteId} disabled />
                  </div>
                  {site && (
                    <>
                      <div className="form-field">
                        <label>站点名称</label>
                        <input className="input" defaultValue={site.name} disabled />
                      </div>
                      <div className="form-field">
                        <label>主域名</label>
                        <input className="input" defaultValue={site.hostname} disabled />
                      </div>
                      <div className="form-field">
                        <label>绑定 IP</label>
                        <input className="input" defaultValue={site.ip} disabled />
                      </div>
                    </>
                  )}
                  <div className="divider" />
                  <h4>数据面配置</h4>
                  <div className="form-field">
                     <label>HTTPS 监听地址</label>
                     <input 
                        className="input"
                        value={config.dataPlane?.httpsListenAddr || ''}
                        onChange={e => updateConfig(prev => ({
                          ...prev,
                          dataPlane: { ...prev.dataPlane, httpsListenAddr: e.target.value }
                        }))}
                        placeholder=":443"
                     />
                  </div>
                </div>
              )}

              {activeTab === 'proxy' && (
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
              )}
              
              {activeTab === 'https' && (
                <div className="panel">
                  <h4>HTTPS 证书管理</h4>
                  <div className="muted" style={{ marginBottom: 15 }}>
                    配置需要自动申请证书的域名列表。
                  </div>
                  
                  <table className="table">
                    <thead>
                      <tr>
                        <th>域名</th>
                        <th style={{ textAlign: 'right' }}>操作</th>
                      </tr>
                    </thead>
                    <tbody>
                      {(config.certificates || []).map((cert, index) => (
                        <tr key={index}>
                          <td>{cert.domain}</td>
                          <td style={{ textAlign: 'right' }}>
                            <button className="button danger small" onClick={() => handleRemoveCert(index)}>
                              <FiTrash2 />
                            </button>
                          </td>
                        </tr>
                      ))}
                      {(!config.certificates || config.certificates.length === 0) && (
                        <tr>
                          <td colSpan={2} className="muted" style={{ textAlign: 'center' }}>
                            暂无证书配置
                          </td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </div>
              )}

              {activeTab === 'security' && (
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
                      </div>
                    )}
                  </div>
                </div>
              )}

              {activeTab === 'limit' && (
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
              )}

              {activeTab === 'version' && (
                <div className="panel">
                  <h4>历史版本</h4>
                  <div className="version-list">
                    {versions.map(v => (
                      <div key={v.id} className="version-item">
                        <div className="version-info">
                          <span className="version-id">{v.id.substring(0, 8)}</span>
                          <span className="version-date">{new Date(v.created_at).toLocaleString()}</span>
                          <span className="version-operator">{v.operator || 'unknown'}</span>
                        </div>
                        <button 
                          className="button small"
                          onClick={() => handleRollback(v.id)}
                        >
                          回滚
                        </button>
                      </div>
                    ))}
                    {versions.length === 0 && <div className="muted">暂无历史版本</div>}
                  </div>
                </div>
              )}

              {activeTab === 'log' && (
                <div className="panel">
                  <h4>实时日志</h4>
                  <div className="form-field">
                    <label>日志过滤规则 (Filter Query)</label>
                    <div style={{ display: 'flex', gap: '10px' }}>
                      <input 
                        className="input" 
                        value={logStream?.filter_query || ''}
                        onChange={e => setLogStream(prev => prev ? ({ ...prev, filter_query: e.target.value }) : { site_id: siteId, filter_query: e.target.value })}
                        placeholder="e.g. level=error"
                      />
                      <button 
                        className="button secondary"
                        onClick={handleUpdateLogStream}
                        disabled={loading}
                      >
                        更新规则
                      </button>
                    </div>
                  </div>
                  
                  <div className="log-controls" style={{ margin: '15px 0', display: 'flex', gap: '10px' }}>
                    <button 
                      className={`button ${isLogConnected ? 'danger' : 'primary'}`}
                      onClick={() => setIsLogConnected(!isLogConnected)}
                    >
                      {isLogConnected ? '断开连接' : '连接日志流'}
                    </button>
                    <button 
                      className="button secondary"
                      onClick={() => setLogs([])}
                    >
                      清空日志
                    </button>
                  </div>

                  <div className="log-viewer">
                    {logs.length === 0 ? (
                      <div className="log-empty">等待日志数据...</div>
                    ) : (
                      logs.map((log, i) => (
                        <div key={i} className="log-line">{log}</div>
                      ))
                    )}
                  </div>
                </div>
              )}
            </>
          )}
        </div>
      </div>

      <style>{`
        .drawer-overlay {
          position: fixed;
          top: 0;
          left: 0;
          width: 100%;
          height: 100%;
          background: rgba(0, 0, 0, 0.5);
          z-index: 999;
          opacity: 0;
          visibility: hidden;
          transition: all 0.3s;
        }
        .drawer-overlay.open {
          opacity: 1;
          visibility: visible;
        }
        .drawer {
          position: fixed;
          top: 0;
          right: -100%;
          width: 85%;
          height: 100vh;
          background: var(--bg-card);
          z-index: 1000;
          transition: right 0.3s ease;
          display: flex;
          flex-direction: column;
          box-shadow: -5px 0 15px rgba(0,0,0,0.5);
        }
        .drawer.open {
          right: 0;
        }
        .form-row {
          display: flex;
          gap: 10px;
          align-items: flex-end;
        }
        .drawer-header {
          padding: 15px 20px;
          border-bottom: 1px solid var(--border);
          display: flex;
          justify-content: space-between;
          align-items: center;
          background: var(--bg-surface);
        }
        .header-left {
          display: flex;
          align-items: center;
          gap: 15px;
        }
        .drawer-body {
          display: flex;
          flex: 1;
          overflow: hidden;
        }
        .config-sidebar {
          width: 200px;
          background: var(--bg-surface);
          border-right: 1px solid var(--border);
          padding: 10px 0;
        }
        .config-menu-item {
          padding: 12px 20px;
          cursor: pointer;
          display: flex;
          align-items: center;
          gap: 10px;
          color: var(--text-sub);
          transition: all 0.2s;
        }
        .config-menu-item:hover {
          background: var(--bg-hover);
          color: var(--text-main);
        }
        .config-menu-item.active {
          background: rgba(0, 122, 204, 0.1);
          color: var(--primary);
          border-right: 2px solid var(--primary);
        }
        .config-content {
          flex: 1;
          padding: 20px;
          overflow-y: auto;
          background: var(--bg-surface);
        }
        .panel {
          max-width: 800px;
        }
        .panel h4 {
          margin-bottom: 15px;
          border-bottom: 1px solid var(--border);
          padding-bottom: 10px;
        }
        .security-group {
          background: var(--bg-surface);
          border: 1px solid var(--border);
          border-radius: 6px;
          padding: 15px;
          margin-bottom: 20px;
        }
        .group-header {
          font-weight: 600;
          margin-bottom: 15px;
          color: var(--text-main);
          display: flex;
          align-items: center;
          justify-content: space-between;
        }
        .group-header.collapsible {
          cursor: pointer;
          user-select: none;
        }
        .toggle-row {
          display: flex;
          gap: 30px;
          align-items: center;
        }
        .toggle-item {
          display: flex;
          align-items: center;
          gap: 10px;
          cursor: pointer;
          user-select: none;
        }
        .cdn-grid {
          display: grid;
          grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
          gap: 15px;
        }
        .cdn-card {
          border: 1px solid var(--border);
          border-radius: 4px;
          background: var(--bg-surface-sub);
          transition: all 0.2s;
        }
        .cdn-card.active {
          border-color: var(--primary);
          background: var(--bg-surface);
          box-shadow: 0 2px 8px rgba(0,0,0,0.05);
        }
        .card-header {
          padding: 10px 15px;
          display: flex;
          justify-content: space-between;
          align-items: center;
          border-bottom: 1px solid transparent;
        }
        .cdn-card.active .card-header {
          border-bottom-color: var(--border);
        }
        .provider-name {
          font-weight: 600;
          font-size: 14px;
        }
        .card-body {
          padding: 15px;
        }
        .toggle-btn {
          cursor: pointer;
          display: flex;
        }
        .badge {
          background: var(--bg-hover);
          padding: 2px 8px;
          border-radius: 10px;
          font-size: 12px;
          color: var(--text-sub);
        }
        .alert {
          padding: 10px 15px;
          border-radius: 4px;
          margin-bottom: 20px;
        }
        .alert.error {
          background: rgba(244, 67, 54, 0.1);
          color: var(--error);
          border: 1px solid var(--error);
        }
        .alert.success {
          background: rgba(76, 175, 80, 0.1);
          color: var(--success);
          border: 1px solid var(--success);
        }
        .icon-btn {
          background: none;
          border: none;
          color: var(--text-sub);
          cursor: pointer;
          font-size: 20px;
          padding: 5px;
        }
        .icon-btn:hover {
          color: var(--text-main);
        }
        .version-item {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 10px;
          border-bottom: 1px solid var(--border);
        }
        .version-info {
          display: flex;
          gap: 15px;
          color: var(--text-sub);
        }
        .checkbox-label {
          display: flex;
          align-items: center;
          gap: 10px;
          cursor: pointer;
        }
        .input.small, .select.small {
          padding: 4px 8px;
          height: 30px;
        }
        .log-viewer {
          background: #1e1e1e;
          color: #d4d4d4;
          font-family: monospace;
          padding: 15px;
          border-radius: 4px;
          height: 400px;
          overflow-y: auto;
          font-size: 13px;
        }
        .log-line {
          margin-bottom: 4px;
          border-bottom: 1px solid #333;
          padding-bottom: 2px;
          white-space: pre-wrap;
          word-break: break-all;
        }
        .log-empty {
          color: #666;
          text-align: center;
          margin-top: 180px;
        }
      `}</style>
      </div>
    </>
  )
}
