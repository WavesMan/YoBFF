import { useCallback, useEffect, useMemo, useState } from 'react'
import { FiChevronDown, FiChevronRight, FiX, FiCheck } from 'react-icons/fi'
import { BsToggleOn, BsToggleOff } from 'react-icons/bs'
import { fetchSiteCDNOriginStatus, refreshSiteCDNOrigin } from '../../../../admin/api'
import type { CDNProviderSetting, Config, Site, SiteCDNOriginStatus, SSLCertificate } from '../../../../admin/types'

type SecurityTabProps = {
  token: string
  operator: string
  siteId: string
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
  token,
  operator,
  siteId,
  site,
  config,
  updateConfig,
  filteredCerts,
  showCertSelector,
  setShowCertSelector,
  cdnExpanded,
  setCdnExpanded,
}: SecurityTabProps) {
  const [originStatus, setOriginStatus] = useState<Record<string, SiteCDNOriginStatus>>({})
  const [originStatusError, setOriginStatusError] = useState('')
  const [loadingOriginStatus, setLoadingOriginStatus] = useState(false)
  const [refreshConfirmProvider, setRefreshConfirmProvider] = useState<string | null>(null)
  const [refreshingProvider, setRefreshingProvider] = useState<string | null>(null)
  const [expandedProviders, setExpandedProviders] = useState<Record<string, boolean>>({})

  const loadOriginStatus = useCallback(async () => {
    if (!token || !siteId) return
    setLoadingOriginStatus(true)
    setOriginStatusError('')
    try {
      const data = await fetchSiteCDNOriginStatus(token, siteId)
      const next: Record<string, SiteCDNOriginStatus> = {}
      for (const item of data.items || []) {
        const key = (item.provider || '').toLowerCase()
        if (!key) continue
        next[key] = item
      }
      setOriginStatus(next)
    } catch (e) {
      setOriginStatusError(e instanceof Error ? e.message : '加载回源 IP 同步状态失败')
    } finally {
      setLoadingOriginStatus(false)
    }
  }, [siteId, token])

  const confirmRefresh = useCallback(async () => {
    if (!refreshConfirmProvider) return
    setRefreshingProvider(refreshConfirmProvider)
    try {
      await refreshSiteCDNOrigin(token, siteId, { providers: [refreshConfirmProvider] }, operator)
      setRefreshConfirmProvider(null)
      await loadOriginStatus()
    } catch (e) {
      setOriginStatusError(e instanceof Error ? e.message : '触发回源 IP 拉取失败')
    } finally {
      setRefreshingProvider(null)
    }
  }, [loadOriginStatus, operator, refreshConfirmProvider, siteId, token])

  useEffect(() => {
    if (!token || !siteId) return
    loadOriginStatus()
  }, [loadOriginStatus, siteId, token])

  const originProtectionEnabled = config.security?.originProtectionMode !== 'disabled'
  const allowedProviders = useMemo(() => config.security?.allowedCdnProviders || [], [config.security?.allowedCdnProviders])

  useEffect(() => {
    if (!cdnExpanded) return
    setExpandedProviders(prev => {
      const next = { ...prev }
      for (const provider of allowedProviders) {
        if (next[provider] == null) next[provider] = true
      }
      return next
    })
  }, [allowedProviders, cdnExpanded])

  const getEnableError = useCallback((provider: string, settings: CDNProviderSetting) => {
    if (provider === 'cloudflare') {
      return ''
    }
    if (provider === 'aliyun') {
      const accessKeyID = String(settings.apiKey || '').trim()
      const accessKeySecret = String(settings.secretKey || '').trim()
      const siteIdText = String(settings.option || '').trim()
      if (!accessKeyID) return '需要填写 AccessKeyId 才能启用'
      if (!accessKeySecret) return '需要填写 AccessKeySecret 才能启用'
      if (!siteIdText) return '需要填写 ESA SiteId 才能启用'
      return ''
    }
    if (provider === 'tencent') {
      const secretID = String(settings.apiKey || '').trim()
      const secretKey = String(settings.secretKey || '').trim()
      const zoneId = String(settings.zoneId || '').trim()
      if (!secretID) return '需要填写 SecretId 才能启用'
      if (!secretKey) return '需要填写 SecretKey 才能启用'
      if (!zoneId) return '需要填写 TEO ZoneId 才能启用'
      return ''
    }
    return ''
  }, [])

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
              启用后，网关将仅允许来自所选 CDN 厂商的回源 IP 访问，并对其他来源返回 403。
              本项目仅负责定时拉取回源 IP，云侧回源策略需您自行开启/关闭。
            </div>

            <div style={{ display: 'flex', gap: '12px', alignItems: 'center', marginBottom: '15px' }}>
              <div
                className="toggle-item"
                onClick={() => {
                  updateConfig(prev => ({
                    ...prev,
                    security: {
                      ...prev.security,
                      originProtectionMode: originProtectionEnabled ? 'disabled' : 'enforced'
                    }
                  }))
                }}
              >
                {originProtectionEnabled ? (
                  <BsToggleOn size={24} color="#10b981" />
                ) : (
                  <BsToggleOff size={24} color="#9ca3af" />
                )}
                <span>启用回源来源防护</span>
              </div>

              <button
                className="button secondary"
                onClick={loadOriginStatus}
                disabled={loadingOriginStatus}
              >
                {loadingOriginStatus ? '刷新状态中...' : '刷新同步状态'}
              </button>
            </div>

            {originStatusError && (
              <div className="alert error" style={{ marginBottom: '15px' }}>
                {originStatusError}
              </div>
            )}
            
            <div className="cdn-grid">
              {['aliyun', 'tencent', 'cloudflare'].map(provider => {
                const isEnabled = allowedProviders.includes(provider)
                const settings = config.security?.cdnProviderSettings?.[provider] || ({} as CDNProviderSetting)
                const status = originStatus[provider]
                const isExpanded = !!expandedProviders[provider]
                const enableError = getEnableError(provider, settings)
                const canEnable = !enableError
                
                return (
                  <div key={provider} className={`cdn-card ${isEnabled ? 'active' : ''}`}>
                    <div
                      className="card-header"
                      onClick={() => setExpandedProviders(prev => ({ ...prev, [provider]: !prev[provider] }))}
                      onKeyDown={(event) => {
                        if (event.key !== 'Enter' && event.key !== ' ') return
                        event.preventDefault()
                        setExpandedProviders(prev => ({ ...prev, [provider]: !prev[provider] }))
                      }}
                      role="button"
                      tabIndex={0}
                    >
                      <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                        {isExpanded ? <FiChevronDown /> : <FiChevronRight />}
                        <span className="provider-name">{provider.toUpperCase()}</span>
                      </div>
                      <button
                        type="button"
                        className="toggle-btn"
                        title={!isEnabled && !canEnable ? enableError : ''}
                        disabled={!isEnabled && !canEnable}
                        onClick={(e) => {
                          e.stopPropagation()
                          if (!isEnabled && !canEnable) {
                            setExpandedProviders(prev => ({ ...prev, [provider]: true }))
                            return
                          }
                          const current = allowedProviders
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
                          <BsToggleOff size={24} color={!canEnable ? '#d1d5db' : '#9ca3af'} />
                        )}
                      </button>
                    </div>
                    
                    {isExpanded && (
                      <div className="card-body">
                        {!isEnabled && enableError && (
                          <div className="muted" style={{ marginBottom: '10px', fontSize: '12px' }}>
                            {enableError}
                          </div>
                        )}
                        {provider === 'cloudflare' && (
                          <div className="muted" style={{ fontSize: '12px' }}>
                            无需配置，系统将从 Cloudflare 公共接口同步回源 IP
                          </div>
                        )}

                        {provider === 'aliyun' && (
                          <>
                            <div className="form-field small">
                              <label>AccessKeyId</label>
                              <input
                                className="input small"
                                placeholder="必填"
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
                              <label>AccessKeySecret</label>
                              <input
                                className="input small"
                                type="password"
                                placeholder="仅写入不回显；留空表示保留历史值"
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
                            <div className="form-field small">
                              <label>ESA SiteId</label>
                              <input
                                className="input small"
                                placeholder="必填"
                                value={settings.option || ''}
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
                                          [provider]: { ...currentSettings[provider], option: val }
                                        }
                                      }
                                    }
                                  })
                                }}
                              />
                            </div>
                            <div className="form-field small">
                              <label>ESA Endpoint</label>
                              <input
                                className="input small"
                                placeholder="可选，默认使用官方端点"
                                value={settings.endpoint || ''}
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
                                          [provider]: { ...currentSettings[provider], endpoint: val }
                                        }
                                      }
                                    }
                                  })
                                }}
                              />
                            </div>
                          </>
                        )}

                        {provider === 'tencent' && (
                          <>
                            <div className="form-field small">
                              <label>SecretId</label>
                              <input
                                className="input small"
                                placeholder="必填"
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
                              <label>SecretKey</label>
                              <input
                                className="input small"
                                type="password"
                                placeholder="仅写入不回显；留空表示保留历史值"
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
                            <div className="form-field small">
                              <label>TEO ZoneId</label>
                              <input
                                className="input small"
                                placeholder="必填"
                                value={settings.zoneId || ''}
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
                                          [provider]: { ...currentSettings[provider], zoneId: val }
                                        }
                                      }
                                    }
                                  })
                                }}
                              />
                            </div>
                            <div className="form-field small">
                              <label>TEO Endpoint</label>
                              <input
                                className="input small"
                                placeholder="可选，默认使用官方端点"
                                value={settings.endpoint || ''}
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
                                          [provider]: { ...currentSettings[provider], endpoint: val }
                                        }
                                      }
                                    }
                                  })
                                }}
                              />
                            </div>
                          </>
                        )}
                        <div className="form-field small">
                          <label>刷新间隔（秒）</label>
                          <input
                            className="input small"
                            type="number"
                            min={60}
                            max={86400}
                            step={60}
                            placeholder="默认 3600"
                            value={settings.refreshIntervalSeconds ? String(settings.refreshIntervalSeconds) : ''}
                            onChange={e => {
                              const raw = e.target.value
                              const parsed = raw === '' ? 0 : Number.parseInt(raw, 10)
                              const next = Number.isFinite(parsed) ? parsed : 0
                              updateConfig(prev => {
                                const currentSettings = prev.security?.cdnProviderSettings || {}
                                return {
                                  ...prev,
                                  security: {
                                    ...prev.security,
                                    cdnProviderSettings: {
                                      ...currentSettings,
                                      [provider]: { ...currentSettings[provider], refreshIntervalSeconds: next }
                                    }
                                  }
                                }
                              })
                            }}
                          />
                        </div>
                        <div className="form-field small">
                          <label>最大陈旧（秒）</label>
                          <input
                            className="input small"
                            type="number"
                            min={60}
                            max={604800}
                            step={60}
                            placeholder="默认：刷新间隔 × 3"
                            value={settings.maxStalenessSeconds ? String(settings.maxStalenessSeconds) : ''}
                            onChange={e => {
                              const raw = e.target.value
                              const parsed = raw === '' ? 0 : Number.parseInt(raw, 10)
                              const next = Number.isFinite(parsed) ? parsed : 0
                              updateConfig(prev => {
                                const currentSettings = prev.security?.cdnProviderSettings || {}
                                return {
                                  ...prev,
                                  security: {
                                    ...prev.security,
                                    cdnProviderSettings: {
                                      ...currentSettings,
                                      [provider]: { ...currentSettings[provider], maxStalenessSeconds: next }
                                    }
                                  }
                                }
                              })
                            }}
                          />
                        </div>

                        <div style={{ marginTop: '10px' }}>
                          <button
                            className="button secondary"
                            onClick={() => setRefreshConfirmProvider(provider)}
                            disabled={!isEnabled || refreshingProvider === provider}
                          >
                            {refreshingProvider === provider ? '拉取中...' : '立即拉取回源 IP'}
                          </button>
                        </div>

                        <div className="muted" style={{ marginTop: '10px', fontSize: '12px' }}>
                          最近尝试：{status?.last_attempt_at || '-'}，最近成功：{status?.last_success_at || '-'}，
                          连续失败：{status?.consecutive_failures ?? 0}{status?.last_error ? `，错误：${status.last_error}` : ''}
                        </div>
                      </div>
                    )}
                  </div>
                )
              })}
            </div>

            <div style={{ marginTop: '20px', borderTop: '1px solid var(--border)', paddingTop: '20px' }}>
              <div className="form-field">
                <label style={{ fontWeight: 600, marginBottom: '8px', display: 'block' }}>自定义 IP 白名单（站点规则）</label>
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

      {refreshConfirmProvider && (
        <div
          className="confirm-overlay"
          onClick={() => refreshingProvider == null && setRefreshConfirmProvider(null)}
        >
          <div
            className="confirm-dialog"
            role="dialog"
            aria-modal="true"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="confirm-title">确认拉取</div>
            <div className="confirm-message">
              确定要立即拉取 {refreshConfirmProvider.toUpperCase()} 的回源 IP 吗？拉取成功后会立即影响数据面放行判断。
            </div>
            <div className="confirm-actions">
              <button
                className="button secondary"
                onClick={() => setRefreshConfirmProvider(null)}
                disabled={refreshingProvider != null}
              >
                取消
              </button>
              <button
                className="button danger"
                onClick={confirmRefresh}
                disabled={refreshingProvider != null}
              >
                {refreshingProvider != null ? '拉取中...' : '确定拉取'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
