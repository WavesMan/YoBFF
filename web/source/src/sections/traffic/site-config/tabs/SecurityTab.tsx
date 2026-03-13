import { useCallback, useEffect, useMemo, useState } from 'react'
import { FiChevronDown, FiChevronRight, FiCheck, FiRefreshCw } from 'react-icons/fi'
import { BsToggleOn, BsToggleOff } from 'react-icons/bs'
import { fetchSiteCDNOriginStatus } from '../../../../admin/api'
import type { CDNProviderSetting, Config, Site, SSLCertificate } from '../../../../admin/types'
import { Card, CardContent, CardHeader, CardTitle } from '../../../../components/ui/Card'
import { Button } from '../../../../components/ui/Button'
import { Input } from '../../../../components/ui/Input'
import { Badge } from '../../../../components/ui/Badge'
import { Modal } from '../../../../components/ui/Modal'
import { useToast } from '../../../../components/ui/Toast'

type SecurityTabProps = {
  token: string
  // operator: string // Removed unused prop
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

/**
 * 安全配置标签页组件
 * 
 * 管理站点的 SSL 证书配置和 CDN 回源保护策略（支持 Cloudflare/Aliyun/Tencent）。
 * 
 * @param props.token - API 认证令牌
 * @param props.siteId - 当前站点 ID
 * @param props.site - 站点详情对象
 * @param props.config - 全局配置对象
 * @param props.updateConfig - 配置更新回调函数
 * @param props.filteredCerts - 可用的 SSL 证书列表
 * @param props.showCertSelector - 是否显示证书选择器模态框
 * @param props.setShowCertSelector - 设置证书选择器显示状态
 * @param props.cdnExpanded - 是否展开 CDN 配置区域
 * @param props.setCdnExpanded - 设置 CDN 区域展开状态
 */
export function SecurityTab({
  token,
  // operator,
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
  const [loadingOriginStatus, setLoadingOriginStatus] = useState(false)
  const [expandedProviders, setExpandedProviders] = useState<Record<string, boolean>>({})
  const { error: toastError } = useToast()

  const loadOriginStatus = useCallback(async () => {
    if (!token || !siteId) return
    setLoadingOriginStatus(true)
    try {
      await fetchSiteCDNOriginStatus(token, siteId)
      // origin status usage removed from UI for now
    } catch (e) {
      toastError(e instanceof Error ? e.message : '加载回源 IP 同步状态失败')
    } finally {
      setLoadingOriginStatus(false)
    }
  }, [siteId, token, toastError])

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
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>HTTPS 与 SSL 配置</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between p-4 border rounded-lg bg-card hover:bg-accent/5 transition-colors">
            <div className="flex items-center gap-3">
              <div 
                className="cursor-pointer text-primary"
                onClick={() => {
                  if (config.dataPlane?.enableHttps) {
                    updateConfig(prev => ({
                      ...prev,
                      dataPlane: { ...prev.dataPlane, enableHttps: false }
                    }))
                  } else {
                    setShowCertSelector(true)
                  }
                }}
              >
                {config.dataPlane?.enableHttps ? (
                  <BsToggleOn size={28} className="text-primary" />
                ) : (
                  <BsToggleOff size={28} className="text-muted-foreground" />
                )}
              </div>
              <div>
                <div className="font-medium">启用全局 HTTPS</div>
                <div className="text-sm text-muted-foreground">
                  {config.dataPlane?.enableHttps 
                    ? `已启用 (证书ID: ${config.dataPlane?.certId || '未选择'})` 
                    : '开启后将强制使用 HTTPS 访问'}
                </div>
              </div>
            </div>
            <Button variant="secondary" onClick={() => setShowCertSelector(true)}>
              {config.dataPlane?.enableHttps ? '更换证书' : '选择证书并开启'}
            </Button>
          </div>

          <div className="flex items-center justify-between p-4 border rounded-lg bg-card hover:bg-accent/5 transition-colors">
            <div className="flex items-center gap-3">
              <div 
                className="cursor-pointer"
                onClick={() => updateConfig(prev => ({
                  ...prev,
                  security: { ...prev.security, enableHsts: !prev.security?.enableHsts }
                }))}
              >
                {config.security?.enableHsts ? (
                  <BsToggleOn size={28} className="text-primary" />
                ) : (
                  <BsToggleOff size={28} className="text-muted-foreground" />
                )}
              </div>
              <div>
                <div className="font-medium">启用 HSTS (强制跳转)</div>
                <div className="text-sm text-muted-foreground">
                  强制客户端使用 HTTPS 连接，防止降级攻击
                </div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader 
          className="cursor-pointer flex flex-row items-center justify-between"
          onClick={() => setCdnExpanded(!cdnExpanded)}
        >
          <div className="flex items-center gap-2">
            {cdnExpanded ? <FiChevronDown /> : <FiChevronRight />}
            <CardTitle>CDN 回源来源防护</CardTitle>
          </div>
          <Badge variant={originProtectionEnabled ? "primary" : "default"}>
            {(config.security?.allowedCdnProviders || []).length} 已启用
          </Badge>
        </CardHeader>
        
        {cdnExpanded && (
          <CardContent className="space-y-6">
            <div className="text-sm text-muted-foreground bg-muted/30 p-3 rounded-md">
              启用后，网关将仅允许来自所选 CDN 厂商的回源 IP 访问，并对其他来源返回 403。
              本项目仅负责定时拉取回源 IP，云侧回源策略需您自行开启/关闭。
            </div>

            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div
                  className="cursor-pointer"
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
                    <BsToggleOn size={28} className="text-primary" />
                  ) : (
                    <BsToggleOff size={28} className="text-muted-foreground" />
                  )}
                </div>
                <span className="font-medium">启用回源来源防护</span>
              </div>

              <Button
                variant="secondary"
                size="sm"
                onClick={loadOriginStatus}
                disabled={loadingOriginStatus}
                icon={<FiRefreshCw className={loadingOriginStatus ? "animate-spin" : ""} />}
              >
                {loadingOriginStatus ? '刷新状态中...' : '刷新同步状态'}
              </Button>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {['aliyun', 'tencent', 'cloudflare'].map(provider => {
                const isEnabled = allowedProviders.includes(provider)
                const settings = config.security?.cdnProviderSettings?.[provider] || ({} as CDNProviderSetting)
                // const status = originStatus[provider] // unused
                const isExpanded = !!expandedProviders[provider]
                const enableError = getEnableError(provider, settings)
                const canEnable = !enableError
                
                return (
                  <Card key={provider} className={`border transition-all ${isEnabled ? 'border-primary/50 bg-primary/5' : ''}`}>
                    <div
                      className="p-4 flex items-center justify-between cursor-pointer"
                      onClick={() => setExpandedProviders(prev => ({ ...prev, [provider]: !prev[provider] }))}
                    >
                      <div className="flex items-center gap-2 font-medium">
                        {isExpanded ? <FiChevronDown /> : <FiChevronRight />}
                        {provider.toUpperCase()}
                      </div>
                      <div
                        role="button"
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
                          <BsToggleOn size={24} className="text-primary" />
                        ) : (
                          <BsToggleOff size={24} className={!canEnable ? "text-muted-foreground/30" : "text-muted-foreground"} />
                        )}
                      </div>
                    </div>
                    
                    {isExpanded && (
                      <div className="px-4 pb-4 space-y-4 border-t pt-4">
                        {!isEnabled && enableError && (
                          <div className="text-xs text-destructive">
                            {enableError}
                          </div>
                        )}
                        {provider === 'cloudflare' && (
                          <div className="text-xs text-muted-foreground">
                            无需配置，系统将从 Cloudflare 公共接口同步回源 IP
                          </div>
                        )}

                        {provider === 'aliyun' && (
                          <>
                            <Input
                              label="AccessKeyId"
                              placeholder="必填"
                              value={settings.apiKey || ''}
                              onChange={e => {
                                const val = e.target.value
                                updateConfig(prev => ({
                                  ...prev,
                                  security: {
                                    ...prev.security,
                                    cdnProviderSettings: {
                                      ...prev.security?.cdnProviderSettings,
                                      [provider]: { ...settings, apiKey: val }
                                    }
                                  }
                                }))
                              }}
                            />
                            <Input
                              label="AccessKeySecret"
                              type="password"
                              placeholder="仅写入不回显；留空表示保留历史值"
                              value={settings.secretKey || ''}
                              onChange={e => {
                                const val = e.target.value
                                updateConfig(prev => ({
                                  ...prev,
                                  security: {
                                    ...prev.security,
                                    cdnProviderSettings: {
                                      ...prev.security?.cdnProviderSettings,
                                      [provider]: { ...settings, secretKey: val }
                                    }
                                  }
                                }))
                              }}
                            />
                            <Input
                              label="ESA SiteId"
                              placeholder="必填"
                              value={settings.option || ''}
                              onChange={e => {
                                const val = e.target.value
                                updateConfig(prev => ({
                                  ...prev,
                                  security: {
                                    ...prev.security,
                                    cdnProviderSettings: {
                                      ...prev.security?.cdnProviderSettings,
                                      [provider]: { ...settings, option: val }
                                    }
                                  }
                                }))
                              }}
                            />
                            <Input
                              label="ESA Endpoint"
                              placeholder="可选，默认使用官方端点"
                              value={settings.endpoint || ''}
                              onChange={e => {
                                const val = e.target.value
                                updateConfig(prev => ({
                                  ...prev,
                                  security: {
                                    ...prev.security,
                                    cdnProviderSettings: {
                                      ...prev.security?.cdnProviderSettings,
                                      [provider]: { ...settings, endpoint: val }
                                    }
                                  }
                                }))
                              }}
                            />
                          </>
                        )}

                        {provider === 'tencent' && (
                          <>
                            <Input
                              label="SecretId"
                              placeholder="必填"
                              value={settings.apiKey || ''}
                              onChange={e => {
                                const val = e.target.value
                                updateConfig(prev => ({
                                  ...prev,
                                  security: {
                                    ...prev.security,
                                    cdnProviderSettings: {
                                      ...prev.security?.cdnProviderSettings,
                                      [provider]: { ...settings, apiKey: val }
                                    }
                                  }
                                }))
                              }}
                            />
                            <Input
                              label="SecretKey"
                              type="password"
                              placeholder="仅写入不回显；留空表示保留历史值"
                              value={settings.secretKey || ''}
                              onChange={e => {
                                const val = e.target.value
                                updateConfig(prev => ({
                                  ...prev,
                                  security: {
                                    ...prev.security,
                                    cdnProviderSettings: {
                                      ...prev.security?.cdnProviderSettings,
                                      [provider]: { ...settings, secretKey: val }
                                    }
                                  }
                                }))
                              }}
                            />
                            <Input
                              label="TEO ZoneId"
                              placeholder="必填"
                              value={settings.zoneId || ''}
                              onChange={e => {
                                const val = e.target.value
                                updateConfig(prev => ({
                                  ...prev,
                                  security: {
                                    ...prev.security,
                                    cdnProviderSettings: {
                                      ...prev.security?.cdnProviderSettings,
                                      [provider]: { ...settings, zoneId: val }
                                    }
                                  }
                                }))
                              }}
                            />
                          </>
                        )}
                      </div>
                    )}
                  </Card>
                )
              })}
            </div>
          </CardContent>
        )}
      </Card>

      <Modal
        isOpen={showCertSelector}
        onClose={() => setShowCertSelector(false)}
        title="选择 SSL 证书"
      >
        <div className="space-y-4">
          <p className="text-sm text-muted-foreground">
            请为 {site?.hostname} 选择一个匹配的 SSL 证书以启用 HTTPS。
          </p>
          
          <div className="max-h-[300px] overflow-y-auto space-y-2 border rounded-md p-2">
            {filteredCerts.length === 0 ? (
              <div className="p-4 text-center text-muted-foreground">
                没有找到匹配的证书，请先在证书管理中上传。
              </div>
            ) : (
              filteredCerts.map(cert => (
                <div 
                  key={cert.id} 
                  className={`flex items-center justify-between p-3 rounded-md cursor-pointer transition-colors ${
                    config.dataPlane?.certId === cert.id 
                      ? 'bg-primary/10 border-primary' 
                      : 'hover:bg-muted'
                  }`}
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
                  <div>
                    <div className="font-medium">{cert.name}</div>
                    <div className="text-xs text-muted-foreground mt-1">
                      {cert.domains.join(', ')}
                    </div>
                  </div>
                  {config.dataPlane?.certId === cert.id && <FiCheck className="text-primary" />}
                </div>
              ))
            )}
          </div>
        </div>
      </Modal>
    </div>
  )
}
