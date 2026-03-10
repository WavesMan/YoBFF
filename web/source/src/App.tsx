import { useCallback, useEffect, useMemo, useState } from 'react'
import { FiActivity, FiEye, FiSettings, FiShield, FiShuffle, FiSliders, FiLock } from 'react-icons/fi'
import './App.css'
import {
  applyConfig as applyConfigRequest,
  fetchCaptcha,
  fetchCdnConfig,
  fetchConfig,
  fetchConfigVersions,
  fetchHealthz,
  fetchLoginCaptchaRequirement,
  fetchLogLevel,
  fetchLogStats,
  loginAdmin,
  rollbackConfig as rollbackConfigRequest,
  logoutAdmin,
  normalizeConfig,
  reloadConfig as reloadConfigRequest,
  updateLogLevel as updateLogLevelRequest,
  validateConfig as validateConfigRequest,
} from './admin/api'
import type {
  CaptchaResponse,
  CDNStatus,
  Config,
  ConfigVersion,
  HealthzResponse,
  LoginPayload,
  LogStats,
  ValidationIssue,
} from './admin/types'
import { Sidebar } from './components/Sidebar'
import { TopBar } from './components/TopBar'
import { ConfigPanel } from './sections/ConfigPanel'
import { ObservabilitySection } from './sections/ObservabilitySection'
import { OverviewSection } from './sections/OverviewSection'
import { SecuritySection } from './sections/SecuritySection'
import { CertificatesSection } from './sections/CertificatesSection'
import { SystemSection } from './sections/SystemSection'
import { TrafficSection } from './sections/TrafficSection'

const menuItems = [
  { key: 'dashboard', label: '仪表盘', icon: <FiActivity /> },
  { key: 'traffic', label: '流量管理', icon: <FiShuffle /> },
  { key: 'security', label: '安全防护', icon: <FiShield /> },
  { key: 'certificates', label: '证书管理', icon: <FiLock /> },
  { key: 'observability', label: '观测中心', icon: <FiEye /> },
  { key: 'system', label: '系统设置', icon: <FiSliders /> },
  { key: 'config', label: '配置应用', icon: <FiSettings /> },
]

function App() {
  const [activeSection, setActiveSection] = useState('dashboard')
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const [token, setToken] = useState(() => localStorage.getItem('yobff_token'))
  const [health, setHealth] = useState<HealthzResponse | null>(null)
  const [logStats, setLogStats] = useState<LogStats | null>(null)
  const [logLevel, setLogLevel] = useState('info')
  const [config, setConfig] = useState<Config | null>(null)
  const [configDraft, setConfigDraft] = useState<Config>(normalizeConfig(null))
  const [configView, setConfigView] = useState<'apply' | 'dry-run' | 'rollback'>('apply')
  const [configVersions, setConfigVersions] = useState<ConfigVersion[]>([])
  const [versionsAutoLoaded, setVersionsAutoLoaded] = useState(false)
  const [validationIssues, setValidationIssues] = useState<ValidationIssue[]>([])
  const [validationTime, setValidationTime] = useState('')
  const [cdnStatus, setCdnStatus] = useState<Record<string, CDNStatus>>({})
  const [captcha, setCaptcha] = useState<CaptchaResponse | null>(null)
  const [captchaRequired, setCaptchaRequired] = useState(false)
  const [loginForm, setLoginForm] = useState<LoginPayload>({
    username: '',
    password: '',
    captcha_id: '',
    captcha_code: '',
  })
  const [advancedMode, setAdvancedMode] = useState(false)
  const [jsonDraft, setJsonDraft] = useState('')
  const [operatorName, setOperatorName] = useState(() => localStorage.getItem('yobff_operator') || '')
  const [statusMessage, setStatusMessage] = useState('')
  const [errorMessage, setErrorMessage] = useState('')
  const [loading, setLoading] = useState({
    health: false,
    logStats: false,
    config: false,
    logLevel: false,
    applying: false,
    validating: false,
    reload: false,
    versions: false,
    rollback: false,
  })

  const pageHeader = useMemo(() => {
    const headers: Record<string, { title: string; subtitle: string }> = {
      dashboard: {
        title: '仪表盘',
        subtitle: '核心健康度与流量态势概览',
      },
      traffic: {
        title: '流量管理',
        subtitle: '域名与转发策略的统一配置入口',
      },
      security: {
        title: '安全防护',
        subtitle: '防火墙、WAF 与访问控制策略',
      },
      certificates: {
        title: '证书管理',
        subtitle: 'SSL/TLS 证书生命周期管理',
      },
      observability: {
        title: '观测中心',
        subtitle: '日志检索、监控大盘与链路追踪',
      },
      system: {
        title: '系统设置',
        subtitle: '运行参数与管理能力的集中配置',
      },
      config: {
        title: '配置应用',
        subtitle: '预检校验、差异对比与版本回滚入口',
      },
    }
    return headers[activeSection] || {
      title: 'BFF 负载均衡网关管理端',
      subtitle: '沉浸式暗色风格，支持配置实时感知与审计流程',
    }
  }, [activeSection])

  const notificationCount = useMemo(() => {
    let count = 0
    if (errorMessage) {
      count += 1
    }
    if (statusMessage) {
      count += 1
    }
    return count
  }, [errorMessage, statusMessage])

  const diffCurrentJson = useMemo(() => JSON.stringify(config || normalizeConfig(null), null, 2), [config])
  const diffDraftJson = useMemo(() => JSON.stringify(configDraft, null, 2), [configDraft])
  const diffChanged = diffCurrentJson !== diffDraftJson

  useEffect(() => {
    const loadHealth = async () => {
      setLoading((prev) => ({ ...prev, health: true }))
      try {
        const data = await fetchHealthz()
        setHealth(data)
      } catch (error) {
        setErrorMessage(error instanceof Error ? error.message : '健康检查失败')
      } finally {
        setLoading((prev) => ({ ...prev, health: false }))
      }
    }
    loadHealth()
  }, [])

  useEffect(() => {
    if (!token) {
      return
    }
    const loadSummary = async () => {
      setLoading((prev) => ({ ...prev, logStats: true, logLevel: true }))
      try {
        const [stats, level] = await Promise.all([
          fetchLogStats(token),
          fetchLogLevel(token),
        ])
        setLogStats(stats)
        setLogLevel(level.level || 'info')
      } catch (error) {
        setErrorMessage(error instanceof Error ? error.message : '读取日志信息失败')
      } finally {
        setLoading((prev) => ({ ...prev, logStats: false, logLevel: false }))
      }
    }
    loadSummary()
  }, [token])

  useEffect(() => {
    if (!token) {
      return
    }
    const loadConfig = async () => {
      setLoading((prev) => ({ ...prev, config: true }))
      try {
        const [cfg, cdn] = await Promise.all([
          fetchConfig(token),
          fetchCdnConfig(token),
        ])
        const normalized = normalizeConfig(cfg)
        setConfig(normalized)
        setConfigDraft(normalized)
        setCdnStatus(cdn.status || {})
      } catch (error) {
        setErrorMessage(error instanceof Error ? error.message : '读取配置失败')
      } finally {
        setLoading((prev) => ({ ...prev, config: false }))
      }
    }
    loadConfig()
  }, [token])

  useEffect(() => {
    if (!token || configView !== 'rollback' || loading.versions || versionsAutoLoaded) {
      return
    }
    if (configVersions.length > 0) {
      setVersionsAutoLoaded(true)
      return
    }
    const loadVersions = async () => {
      setLoading((prev) => ({ ...prev, versions: true }))
      setErrorMessage('')
      try {
        const data = await fetchConfigVersions(token, 20)
        setConfigVersions(data.items || [])
      } catch (error) {
        setErrorMessage(error instanceof Error ? error.message : '读取配置版本失败')
      } finally {
        setLoading((prev) => ({ ...prev, versions: false }))
        setVersionsAutoLoaded(true)
      }
    }
    loadVersions()
  }, [token, configView, configVersions.length, loading.versions, versionsAutoLoaded])

  useEffect(() => {
    if (configView !== 'rollback') {
      setVersionsAutoLoaded(false)
    }
  }, [configView])

  const refreshCaptchaRequirement = useCallback(async () => {
    try {
      const data = await fetchLoginCaptchaRequirement()
      if (!data.required) {
        setCaptchaRequired(false)
        setCaptcha(null)
        setLoginForm((prev) => ({
          ...prev,
          captcha_id: '',
          captcha_code: '',
        }))
        return
      }
      setCaptchaRequired(true)
      const captchaData = await fetchCaptcha('default')
      setCaptcha(captchaData)
      setLoginForm((prev) => ({
        ...prev,
        captcha_id: captchaData.captcha_id,
        captcha_code: '',
      }))
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : '验证码状态获取失败')
    }
  }, [])

  useEffect(() => {
    if (!token) {
      refreshCaptchaRequirement()
    }
  }, [refreshCaptchaRequirement, token])

  useEffect(() => {
    if (advancedMode) {
      setJsonDraft(JSON.stringify(configDraft, null, 2))
    }
  }, [advancedMode, configDraft])

  useEffect(() => {
    setValidationIssues([])
    setValidationTime('')
  }, [configDraft, jsonDraft, advancedMode])

  const handleAllowedCidrsChange = (value: string) => {
    const cidrs = value
      .split('\n')
      .map((line) => line.trim())
      .filter(Boolean)
    const normalized = normalizeConfig(configDraft)
    setConfigDraft({
      ...normalized,
      security: {
        ...normalized.security,
        allowedCidrs: cidrs,
      },
    })
  }

  const handleBlockPageChange = (value: string) => {
    const normalized = normalizeConfig(configDraft)
    setConfigDraft({
      ...normalized,
      security: {
        ...normalized.security,
        blockPageHtml: value,
      },
    })
  }


  const handleOperatorChange = (value: string) => {
    setOperatorName(value)
    localStorage.setItem('yobff_operator', value)
  }

  const handleApplyConfig = async () => {
    if (!token) {
      setErrorMessage('请先登录后再应用配置')
      return
    }
    setLoading((prev) => ({ ...prev, applying: true }))
    setErrorMessage('')
    setStatusMessage('')
    setValidationIssues([])
    setValidationTime('')
    try {
      let payload = configDraft
      if (advancedMode) {
        payload = JSON.parse(jsonDraft) as Config
        setConfigDraft(payload)
      }
      const operatorValue = operatorName.trim() || 'unknown'
      await applyConfigRequest(token, payload, operatorValue)
      const normalized = normalizeConfig(payload)
      setConfig(normalized)
      setConfigDraft(normalized)
      setStatusMessage('配置已提交并应用')
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : '配置应用失败')
    } finally {
      setLoading((prev) => ({ ...prev, applying: false }))
    }
  }

  const handleValidateConfig = async () => {
    if (!token) {
      setErrorMessage('请先登录后再执行预检')
      return
    }
    setLoading((prev) => ({ ...prev, validating: true }))
    setErrorMessage('')
    setStatusMessage('')
    try {
      let payload = configDraft
      if (advancedMode) {
        payload = JSON.parse(jsonDraft) as Config
        setConfigDraft(payload)
      }
      const operatorValue = operatorName.trim() || 'unknown'
      const result = await validateConfigRequest(token, payload, operatorValue)
      setValidationIssues(result.errors || [])
      setValidationTime(new Date().toLocaleString())
      if (result.valid) {
        setStatusMessage('预检通过')
      } else {
        setErrorMessage('预检未通过')
      }
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : '预检失败')
    } finally {
      setLoading((prev) => ({ ...prev, validating: false }))
    }
  }

  const handleReload = async () => {
    if (!token) {
      setErrorMessage('请先登录后再热重载')
      return
    }
    setLoading((prev) => ({ ...prev, reload: true }))
    setErrorMessage('')
    setStatusMessage('')
    try {
      await reloadConfigRequest(token)
      setStatusMessage('配置热重载已触发')
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : '热重载失败')
    } finally {
      setLoading((prev) => ({ ...prev, reload: false }))
    }
  }

  const handleRefreshVersions = async () => {
    if (!token) {
      setErrorMessage('请先登录后再读取版本')
      return
    }
    setLoading((prev) => ({ ...prev, versions: true }))
    setErrorMessage('')
    setVersionsAutoLoaded(true)
    try {
      const data = await fetchConfigVersions(token, 20)
      setConfigVersions(data.items || [])
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : '读取配置版本失败')
    } finally {
      setLoading((prev) => ({ ...prev, versions: false }))
    }
  }

  const handleRollback = async (versionID: string) => {
    if (!token) {
      setErrorMessage('请先登录后再回滚')
      return
    }
    if (!versionID) {
      return
    }
    const confirmed = window.confirm(`确认回滚到版本 ${versionID} 吗？`)
    if (!confirmed) {
      return
    }
    setLoading((prev) => ({ ...prev, rollback: true }))
    setErrorMessage('')
    setStatusMessage('')
    try {
      const operatorValue = operatorName.trim() || 'unknown'
      await rollbackConfigRequest(token, { version_id: versionID }, operatorValue)
      setStatusMessage(`已回滚到版本 ${versionID}`)
      const [cfg, versions] = await Promise.all([
        fetchConfig(token),
        fetchConfigVersions(token, 20),
      ])
      const normalized = normalizeConfig(cfg)
      setConfig(normalized)
      setConfigDraft(normalized)
      setConfigVersions(versions.items || [])
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : '回滚失败')
    } finally {
      setLoading((prev) => ({ ...prev, rollback: false }))
    }
  }

  const handleLogLevelChange = async (level: string) => {
    if (!token) {
      setErrorMessage('请先登录后再更新日志级别')
      return
    }
    setErrorMessage('')
    try {
      const data = await updateLogLevelRequest(token, level)
      setLogLevel(data.level || level)
      setStatusMessage('日志级别已更新')
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : '更新日志级别失败')
    }
  }

  const handleFetchCaptcha = async () => {
    setErrorMessage('')
    try {
      const data = await fetchCaptcha('default')
      setCaptcha(data)
      setLoginForm((prev) => ({ ...prev, captcha_id: data.captcha_id }))
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : '验证码获取失败')
    }
  }

  const handleLogin = async () => {
    setErrorMessage('')
    setStatusMessage('')
    try {
      const data = await loginAdmin(loginForm)
      if (!data.token) {
        throw new Error('登录未返回 token')
      }
      localStorage.setItem('yobff_token', data.token)
      setToken(data.token)
      setStatusMessage('登录成功')
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : '登录失败')
      await refreshCaptchaRequirement()
    }
  }

  const handleLogout = async () => {
    setErrorMessage('')
    setStatusMessage('')
    if (token) {
      try {
        await logoutAdmin(token)
      } catch (error) {
        setErrorMessage(error instanceof Error ? error.message : '登出失败')
      }
    }
    localStorage.removeItem('yobff_token')
    setToken(null)
    setLogStats(null)
    setConfig(null)
    setConfigDraft(normalizeConfig(null))
    setCdnStatus({})
    setJsonDraft('')
    setConfigVersions([])
    setVersionsAutoLoaded(false)
    setStatusMessage('已退出登录')
  }

  const captchaImageSrc = captcha?.image_base64
    ? captcha.image_base64.startsWith('data:image/')
      ? captcha.image_base64
      : `data:image/png;base64,${captcha.image_base64}`
    : ''

  if (!token) {
    return (
      <div className="login-overlay">
        <div className="login-card card">
          <div className="login-header">
            <div className="brand login-brand">
              <span className="brand-icon" />
              <span className="brand-text">YoBFF</span>
            </div>
            <div className="login-title">管理端登录</div>
            <div className="muted">登录后进入管理面板进行配置与观测</div>
          </div>
          {errorMessage && (
            <div className="error-banner" role="alert">
              {errorMessage}
            </div>
          )}
          {statusMessage && (
            <div className="success-banner" role="status" aria-live="polite">
              {statusMessage}
            </div>
          )}
          <div className="stack">
            <div className="form-row">
              <div className="form-field">
                <label className="label">用户名</label>
                <input
                  className="input"
                  value={loginForm.username}
                  onChange={(event) =>
                    setLoginForm({
                      ...loginForm,
                      username: event.target.value,
                    })
                  }
                  placeholder="admin"
                />
              </div>
              <div className="form-field">
                <label className="label">密码</label>
                <input
                  className="input"
                  type="password"
                  value={loginForm.password}
                  onChange={(event) =>
                    setLoginForm({
                      ...loginForm,
                      password: event.target.value,
                    })
                  }
                  placeholder="password"
                />
              </div>
            </div>
            {captchaRequired && (
              <div className="form-row">
                <div className="form-field">
                  <label className="label">验证码</label>
                  <input
                    className="input"
                    value={loginForm.captcha_code || ''}
                    onChange={(event) =>
                      setLoginForm({
                        ...loginForm,
                        captcha_code: event.target.value,
                      })
                    }
                    placeholder="1234"
                  />
                </div>
                <div className="form-field">
                  <label className="label">验证码图片</label>
                  <div className="inline">
                    <button className="button secondary" onClick={handleFetchCaptcha}>
                      获取验证码
                    </button>
                    {captchaImageSrc && (
                      <img className="captcha-image" src={captchaImageSrc} alt="验证码" />
                    )}
                  </div>
                </div>
              </div>
            )}
            <button className="button primary" onClick={handleLogin}>
              登录并进入面板
            </button>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className={`app-shell ${sidebarCollapsed ? 'collapsed' : ''}`}>
      <Sidebar
        items={menuItems}
        activeKey={activeSection}
        onChange={setActiveSection}
        collapsed={sidebarCollapsed}
        onToggle={() => setSidebarCollapsed((prev) => !prev)}
      />
      <div className="content">
        <TopBar
          onLogout={handleLogout}
          notificationCount={notificationCount}
          isLoggedIn={Boolean(token)}
        />
        <main className="main">
          <div>
            <h1 className="page-title">{pageHeader.title}</h1>
            <p className="page-subtitle">{pageHeader.subtitle}</p>
          </div>
          {errorMessage && (
            <div className="error-banner" role="alert">
              {errorMessage}
            </div>
          )}
          {statusMessage && (
            <div className="success-banner" role="status" aria-live="polite">
              {statusMessage}
            </div>
          )}
          {!token && (
            <div className="card">
              <div className="card-header">
                <h3 className="card-title">登录状态</h3>
                <span className="pill">POST /admin/api/v1/login</span>
              </div>
              <div className="muted">请先在系统设置中完成登录，启用受保护接口的读取</div>
            </div>
          )}
          {activeSection === 'dashboard' && (
            <OverviewSection
              health={health}
              logStats={logStats}
              logLevel={logLevel}
              loadingHealth={loading.health}
              loadingLogStats={loading.logStats}
            />
          )}
          {activeSection === 'traffic' && (
            <TrafficSection
                token={token || ''}
                operator={operatorName || 'unknown'}
              />
            )}
            {activeSection === 'security' && (
            <SecuritySection
              configDraft={configDraft}
              cdnStatus={cdnStatus}
              onAllowedCidrsChange={handleAllowedCidrsChange}
              onBlockPageChange={handleBlockPageChange}
            />
          )}
          {activeSection === 'certificates' && (
            <CertificatesSection token={token || ''} />
          )}
          {activeSection === 'observability' && (
            <ObservabilitySection
              health={health}
              logStats={logStats}
              loadingHealth={loading.health}
              loadingLogStats={loading.logStats}
            />
          )}
          {activeSection === 'system' && (
            <SystemSection
              logLevel={logLevel}
              loadingLogLevel={loading.logLevel}
              onLogLevelChange={handleLogLevelChange}
              loadingReload={loading.reload}
              onReload={handleReload}
            />
          )}
          {activeSection === 'config' && (
            <ConfigPanel
              configDraft={configDraft}
              configSnapshot={config}
              advancedMode={advancedMode}
              jsonDraft={jsonDraft}
              loadingApplying={loading.applying}
              loadingValidating={loading.validating}
              loadingVersions={loading.versions}
              view={configView}
              operator={operatorName}
              validationIssues={validationIssues}
              validationTime={validationTime}
              configVersions={configVersions}
              diffCurrentJson={diffCurrentJson}
              diffDraftJson={diffDraftJson}
              diffChanged={diffChanged}
              onViewChange={(nextView) => {
                setConfigView(nextView)
                setErrorMessage('')
                setStatusMessage('')
              }}
              onOperatorChange={handleOperatorChange}
              onToggleMode={setAdvancedMode}
              onJsonChange={setJsonDraft}
              onConfigChange={setConfigDraft}
              onApply={handleApplyConfig}
              onValidate={handleValidateConfig}
              onRefreshVersions={handleRefreshVersions}
              onRollback={handleRollback}
              onReset={() => {
                setConfigDraft(normalizeConfig(config))
                setValidationIssues([])
                setValidationTime('')
                setStatusMessage('已恢复到最近一次读取的配置')
              }}
            />
          )}
          {loading.config && (
            <div className="card">
              <div className="stack">
                <div className="skeleton" />
                <div className="skeleton" />
                <div className="skeleton" />
              </div>
            </div>
          )}
        </main>
      </div>
    </div>
  )
}

export default App
