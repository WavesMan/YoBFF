import './SiteConfigDrawer.css'
import { useCallback, useEffect, useState } from 'react'
import {
  FiActivity,
  FiClock,
  FiGlobe,
  FiLock,
  FiSave,
  FiSettings,
  FiShield,
  FiX,
  FiZap,
} from 'react-icons/fi'
import {
  createLBPool,
  deleteLBPool,
  deleteLBRoute,
  deleteSiteVersion,
  fetchLBPools,
  fetchLBRoutes,
  fetchSite,
  fetchSiteConfig,
  fetchSiteVersions,
  rollbackSiteConfig,
  updateLBPool,
  upsertLBRoute,
  updateSiteConfig,
  updateSite,
  validateConfig,
  fetchSiteLogStream,
  fetchSiteLogHistory,
  updateSiteLogStream,
  fetchCertificates
} from '../../../admin/api'
import type {
  Config,
  ConfigVersion,
  LBPool,
  LBRouteRule,
  Site,
  SiteLogEntry,
  SiteLogKind,
  SiteLogStream,
  SiteUpdateRequest,
  SSLCertificate
} from '../../../admin/types'

import { BasicSettingsTab } from './tabs/BasicSettingsTab'
import { ProxyRulesTab } from './tabs/ProxyRulesTab'
import { HttpsCertTab } from './tabs/HttpsCertTab'
import { SecurityTab } from './tabs/SecurityTab'
import { RateLimitTab } from './tabs/RateLimitTab'
import { VersionHistoryTab } from './tabs/VersionHistoryTab'
import { RealtimeLogTab } from './tabs/RealtimeLogTab'

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
  const [logs, setLogs] = useState<SiteLogEntry[]>([])
  const [isLogConnected, setIsLogConnected] = useState(false)
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [updatingSite, setUpdatingSite] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [deleteVersionTarget, setDeleteVersionTarget] = useState<{ id: string; shortID: string } | null>(null)
  const [deletingVersion, setDeletingVersion] = useState(false)
  const [rollbackVersionTarget, setRollbackVersionTarget] = useState<{ id: string; shortID: string } | null>(null)
  const [rollingBackVersion, setRollingBackVersion] = useState(false)

  const [newPool, setNewPool] = useState<LBPool>({
    id: '',
    name: '',
    strategy: 'weighted_rr',
    nodes: []
  })
  const [newRoute, setNewRoute] = useState<LBRouteRule>({
    domain: '',
    poolId: '',
    fallbackPoolId: '',
    forceHttps: true
  })
  
  // Site Update Form State
  const [siteForm, setSiteForm] = useState<SiteUpdateRequest>({
    name: '',
    hostname: '',
    ip: ''
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

  const loadSite = useCallback(async () => {
    try {
      const data = await fetchSite(token, siteId)
      setSite(data)
      setSiteForm({
        name: data.name,
        hostname: data.hostname,
        ip: data.ip
      })
    } catch (err) {
      console.error('Failed to load site info', err)
    }
  }, [token, siteId])

  const loadConfig = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const [siteConfigResult, lbPoolsResult, lbRoutesResult] = await Promise.allSettled([
        fetchSiteConfig(token, siteId),
        fetchLBPools(token),
        fetchLBRoutes(token),
      ])
      if (siteConfigResult.status !== 'fulfilled') {
        throw siteConfigResult.reason
      }
      const nextConfig = siteConfigResult.value
      if (lbPoolsResult.status === 'fulfilled' && lbRoutesResult.status === 'fulfilled') {
        setConfig({
          ...nextConfig,
          loadBalancer: {
            ...nextConfig.loadBalancer,
            defaultPoolId: lbRoutesResult.value.default_pool_id || nextConfig.loadBalancer?.defaultPoolId || '',
            pools: lbPoolsResult.value.items || [],
            routes: lbRoutesResult.value.items || [],
          },
        })
      } else {
        setConfig(nextConfig)
      }
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

  const loadLogHistory = useCallback(async (params: {
    kind: SiteLogKind
    level: string
    startTime: string
    endTime: string
    limit: number
  }) => {
    const result = await fetchSiteLogHistory(token, siteId, params)
    setLogs(result.items || [])
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

  const saveLoadBalancerByAPI = useCallback(async (draft: Config) => {
    const pools = (draft.loadBalancer?.pools || []).map(pool => ({
      ...pool,
      id: (pool.id || '').trim(),
      name: (pool.name || '').trim(),
      strategy: pool.strategy || 'weighted_rr',
      nodes: (pool.nodes || []).map(node => ({
        ...node,
        id: (node.id || '').trim(),
        upstream: (node.upstream || '').trim(),
        weight: Number(node.weight ?? 0),
        enabled: node.enabled !== false,
      })),
    }))
    const routes = (draft.loadBalancer?.routes || []).map(route => ({
      ...route,
      domain: (route.domain || '').trim(),
      poolId: (route.poolId || '').trim(),
      fallbackPoolId: (route.fallbackPoolId || '').trim(),
      forceHttps: route.forceHttps !== false,
    }))
    const poolIDSet = new Set(pools.map(pool => pool.id).filter(Boolean))
    if (pools.some(pool => !pool.id || !pool.name)) {
      throw new Error('流量池ID和名称不能为空')
    }
    if (pools.some(pool => (pool.nodes || []).some(node => !node.id || !node.upstream || Number(node.weight) <= 0))) {
      throw new Error('流量池节点配置无效')
    }
    if (routes.some(route => !route.domain || !route.poolId || !poolIDSet.has(route.poolId || '') || (route.fallbackPoolId && !poolIDSet.has(route.fallbackPoolId)))) {
      throw new Error('域名绑定规则配置无效')
    }
    const [currentPools, currentRoutes] = await Promise.all([
      fetchLBPools(token),
      fetchLBRoutes(token),
    ])
    const currentPoolIDs = new Set((currentPools.items || []).map(pool => (pool.id || '').trim()).filter(Boolean))
    const currentRouteDomains = new Set((currentRoutes.items || []).map(route => (route.domain || '').trim()).filter(Boolean))
    const desiredPoolIDs = new Set(pools.map(pool => pool.id).filter(Boolean))
    const desiredRouteDomains = new Set(routes.map(route => route.domain).filter(Boolean))
    for (const pool of pools) {
      if (!pool.id) {
        continue
      }
      if (currentPoolIDs.has(pool.id)) {
        await updateLBPool(token, pool.id, pool)
      } else {
        await createLBPool(token, pool)
      }
    }
    for (const route of routes) {
      if (!route.domain) {
        continue
      }
      await upsertLBRoute(token, route.domain, route)
    }
    for (const domain of currentRouteDomains) {
      if (desiredRouteDomains.has(domain)) {
        continue
      }
      await deleteLBRoute(token, domain)
    }
    for (const poolID of currentPoolIDs) {
      if (desiredPoolIDs.has(poolID)) {
        continue
      }
      await deleteLBPool(token, poolID)
    }
    const targetDefaultPoolID = (draft.loadBalancer?.defaultPoolId || '').trim()
    const currentDefaultPoolID = (currentRoutes.default_pool_id || '').trim()
    if (targetDefaultPoolID !== currentDefaultPoolID) {
      const validation = await validateConfig(token, draft, operator)
      if (!validation.valid) {
        throw new Error('默认流量池配置验证失败: ' + validation.errors.map(e => e.message).join(', '))
      }
      await updateSiteConfig(token, siteId, draft, operator)
    }
  }, [token, operator, siteId])

  const handleSave = async () => {
    if (!config) return
    setSaving(true)
    setError('')
    setSuccess('')
    try {
      if (activeTab === 'proxy') {
        await saveLoadBalancerByAPI(config)
      } else {
        const validation = await validateConfig(token, config, operator)
        if (!validation.valid) {
          setError('配置验证失败: ' + validation.errors.map(e => e.message).join(', '))
          return
        }
        await updateSiteConfig(token, siteId, config, operator)
      }
      setSuccess('配置已保存并生效')
      setTimeout(() => setSuccess(''), 3000)
      await loadConfig()
      if (activeTab === 'version') {
        loadVersions()
      }
    } catch (err) {
      const defaultMessage = activeTab === 'proxy' ? '流量池配置保存失败' : '保存配置失败'
      setError(err instanceof Error ? err.message : defaultMessage)
    } finally {
      setSaving(false)
    }
  }

  // handleRollback 打开回滚确认弹窗，等待用户确认后再执行回滚。
  // 参数：versionId 为目标版本标识。
  // 返回：无。
  // 异常：无。
  const handleRollback = async (versionId: string) => {
    setRollbackVersionTarget({ id: versionId, shortID: versionId.substring(0, 8) })
  }

  // confirmRollbackVersion 执行回滚并刷新配置与版本列表。
  // 参数：无。
  // 返回：无。
  // 异常：请求失败时写入错误提示。
  const confirmRollbackVersion = async () => {
    if (!rollbackVersionTarget) return
    setRollingBackVersion(true)
    setLoading(true)
    setError('')
    setSuccess('')
    try {
      await rollbackSiteConfig(token, siteId, rollbackVersionTarget.id, operator)
      await loadConfig()
      setSuccess(`已回滚到版本 ${rollbackVersionTarget.id}`)
      setRollbackVersionTarget(null)
      loadVersions()
    } catch (err) {
      setError(err instanceof Error ? err.message : '回滚失败')
    } finally {
      setLoading(false)
      setRollingBackVersion(false)
    }
  }

  // handleDeleteVersion 打开删除确认弹窗，等待用户二次确认后再执行。
  // 参数：versionId 为待删除版本标识。
  // 返回：无。
  // 异常：无。
  const handleDeleteVersion = async (versionId: string) => {
    setDeleteVersionTarget({ id: versionId, shortID: versionId.substring(0, 8) })
  }

  // confirmDeleteVersion 执行历史版本删除并刷新列表。
  // 参数：无。
  // 返回：无。
  // 异常：请求失败时写入错误提示。
  const confirmDeleteVersion = async () => {
    if (!deleteVersionTarget) return
    setDeletingVersion(true)
    setError('')
    setSuccess('')
    try {
      await deleteSiteVersion(token, siteId, deleteVersionTarget.id)
      setSuccess(`已删除版本 ${deleteVersionTarget.id}`)
      setDeleteVersionTarget(null)
      loadVersions()
    } catch (err) {
      setError(err instanceof Error ? err.message : '删除版本失败')
    } finally {
      setDeletingVersion(false)
    }
  }
  
  const handleSaveSiteInfo = async () => {
    if (!site) return
    setUpdatingSite(true)
    setError('')
    setSuccess('')
    try {
      const updatedSite = await updateSite(token, siteId, siteForm)
      setSite(updatedSite)
      setSuccess('站点基本信息已更新')
      setTimeout(() => setSuccess(''), 3000)
    } catch (err) {
      setError(err instanceof Error ? err.message : '更新站点信息失败')
    } finally {
      setUpdatingSite(false)
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

  const handleAddPool = () => {
    const poolID = (newPool.id || '').trim()
    const poolName = (newPool.name || '').trim()
    if (!poolID || !poolName) return
    updateConfig(prev => ({
      ...prev,
      loadBalancer: {
        ...prev.loadBalancer,
        pools: [
          ...(prev.loadBalancer?.pools || []),
          { ...newPool, id: poolID, name: poolName },
        ]
      }
    }))
    setNewPool({ id: '', name: '', strategy: 'weighted_rr', nodes: [] })
  }

  const handleRemovePool = (index: number) => {
    const removedPoolID = (config?.loadBalancer?.pools || [])[index]?.id || ''
    updateConfig(prev => ({
      ...prev,
      loadBalancer: {
        ...prev.loadBalancer,
        pools: (prev.loadBalancer?.pools || []).filter((_, i) => i !== index),
        defaultPoolId: prev.loadBalancer?.defaultPoolId === removedPoolID
          ? ''
          : prev.loadBalancer?.defaultPoolId,
        routes: (prev.loadBalancer?.routes || []).map(route => ({
          ...route,
          poolId: route.poolId === removedPoolID ? '' : route.poolId,
          fallbackPoolId: route.fallbackPoolId === removedPoolID ? '' : route.fallbackPoolId,
        })),
      }
    }))
    setNewRoute(prev => ({
      ...prev,
      poolId: prev.poolId === removedPoolID ? '' : prev.poolId,
      fallbackPoolId: prev.fallbackPoolId === removedPoolID ? '' : prev.fallbackPoolId,
    }))
  }

  const handleAddRoute = () => {
    if (!newRoute.domain || !newRoute.poolId) return
    updateConfig(prev => ({
      ...prev,
      loadBalancer: {
        ...prev.loadBalancer,
        routes: [...(prev.loadBalancer?.routes || []), newRoute]
      }
    }))
    setNewRoute({ domain: '', poolId: '', fallbackPoolId: '', forceHttps: true })
  }

  const handleRemoveRoute = (index: number) => {
    updateConfig(prev => ({
      ...prev,
      loadBalancer: {
        ...prev.loadBalancer,
        routes: (prev.loadBalancer?.routes || []).filter((_, i) => i !== index)
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
            <FiGlobe /> 流量池管理
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
                <BasicSettingsTab 
                  site={site}
                  siteId={siteId}
                  siteForm={siteForm}
                  setSiteForm={setSiteForm}
                  config={config}
                  updateConfig={updateConfig}
                  handleSaveSiteInfo={handleSaveSiteInfo}
                  updatingSite={updatingSite}
                  loading={loading}
                />
              )}

              {activeTab === 'proxy' && (
                <ProxyRulesTab 
                  config={config}
                  updateConfig={updateConfig}
                  newPool={newPool}
                  setNewPool={setNewPool}
                  newRoute={newRoute}
                  setNewRoute={setNewRoute}
                  handleAddPool={handleAddPool}
                  handleRemovePool={handleRemovePool}
                  handleAddRoute={handleAddRoute}
                  handleRemoveRoute={handleRemoveRoute}
                />
              )}
              
              {activeTab === 'https' && (
                <HttpsCertTab 
                  config={config}
                  handleRemoveCert={handleRemoveCert}
                />
              )}

              {activeTab === 'security' && (
                <SecurityTab 
                  site={site}
                  config={config}
                  updateConfig={updateConfig}
                  filteredCerts={filteredCerts}
                  showCertSelector={showCertSelector}
                  setShowCertSelector={setShowCertSelector}
                  cdnExpanded={cdnExpanded}
                  setCdnExpanded={setCdnExpanded}
                />
              )}

              {activeTab === 'limit' && (
                <RateLimitTab 
                  config={config}
                  updateConfig={updateConfig}
                />
              )}

              {activeTab === 'version' && (
                <VersionHistoryTab 
                  versions={versions}
                  handleRollback={handleRollback}
                  handleDelete={handleDeleteVersion}
                />
              )}

              {activeTab === 'log' && (
                <RealtimeLogTab 
                  siteId={siteId}
                  logStream={logStream}
                  setLogStream={setLogStream}
                  logs={logs}
                  setLogs={setLogs}
                  loadLogHistory={loadLogHistory}
                  isLogConnected={isLogConnected}
                  setIsLogConnected={setIsLogConnected}
                  loading={loading}
                  handleUpdateLogStream={handleUpdateLogStream}
                />
              )}
            </>
          )}
        </div>
      </div>
      </div>
      {deleteVersionTarget && (
        <div
          className="confirm-overlay"
          onClick={() => !deletingVersion && setDeleteVersionTarget(null)}
        >
          <div
            className="confirm-dialog"
            role="dialog"
            aria-modal="true"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="confirm-title">确认删除</div>
            <div className="confirm-message">
              确定要删除版本「{deleteVersionTarget.shortID}」吗？此操作不可恢复。
            </div>
            <div className="confirm-actions">
              <button
                className="button secondary"
                onClick={() => setDeleteVersionTarget(null)}
                disabled={deletingVersion}
              >
                取消
              </button>
              <button
                className="button danger"
                onClick={confirmDeleteVersion}
                disabled={deletingVersion}
              >
                {deletingVersion ? '删除中...' : '确定删除'}
              </button>
            </div>
          </div>
        </div>
      )}
      {rollbackVersionTarget && (
        <div
          className="confirm-overlay"
          onClick={() => !rollingBackVersion && setRollbackVersionTarget(null)}
        >
          <div
            className="confirm-dialog"
            role="dialog"
            aria-modal="true"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="confirm-title">确认回滚</div>
            <div className="confirm-message">
              确定要回滚到版本「{rollbackVersionTarget.shortID}」吗？当前未保存的更改将丢失。
            </div>
            <div className="confirm-actions">
              <button
                className="button secondary"
                onClick={() => setRollbackVersionTarget(null)}
                disabled={rollingBackVersion}
              >
                取消
              </button>
              <button
                className="button danger"
                onClick={confirmRollbackVersion}
                disabled={rollingBackVersion}
              >
                {rollingBackVersion ? '回滚中...' : '确定回滚'}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  )
}
