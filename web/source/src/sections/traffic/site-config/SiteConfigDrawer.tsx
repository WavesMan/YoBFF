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
  fetchSite,
  fetchSiteConfig,
  fetchSiteVersions,
  rollbackSiteConfig,
  updateSiteConfig,
  updateSite,
  validateConfig,
  fetchSiteLogStream,
  updateSiteLogStream,
  fetchCertificates
} from '../../../admin/api'
import type { Config, ConfigVersion, DomainRule, Site, SiteLogStream, SiteUpdateRequest, SSLCertificate } from '../../../admin/types'

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
  const [logs, setLogs] = useState<string[]>([])
  const [isLogConnected, setIsLogConnected] = useState(false)
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [updatingSite, setUpdatingSite] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  // New Domain Rule State
  const [newDomain, setNewDomain] = useState<DomainRule>({
    domain: '',
    upstream: '',
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
                  newDomain={newDomain}
                  setNewDomain={setNewDomain}
                  handleAddDomain={handleAddDomain}
                  handleRemoveDomain={handleRemoveDomain}
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
                />
              )}

              {activeTab === 'log' && (
                <RealtimeLogTab 
                  siteId={siteId}
                  logStream={logStream}
                  setLogStream={setLogStream}
                  logs={logs}
                  setLogs={setLogs}
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
    </>
  )
}
