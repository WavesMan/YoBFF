import { useEffect, useMemo, useState } from 'react'
import { FiActivity, FiEye, FiShield, FiShuffle, FiSliders } from 'react-icons/fi'
import './App.css'
import {
  applyConfig as applyConfigRequest,
  fetchCaptcha,
  fetchCdnConfig,
  fetchConfig,
  fetchHealthz,
  fetchLogLevel,
  fetchLogStats,
  loginAdmin,
  normalizeConfig,
  reloadConfig as reloadConfigRequest,
  updateLogLevel as updateLogLevelRequest,
} from './admin/api'
import type {
  CaptchaResponse,
  CDNStatus,
  Config,
  DomainRule,
  HealthzResponse,
  LoginPayload,
  LogStats,
} from './admin/types'
import { Sidebar } from './components/Sidebar'
import { TopBar } from './components/TopBar'
import { ConfigPanel } from './sections/ConfigPanel'
import { ObservabilitySection } from './sections/ObservabilitySection'
import { OverviewSection } from './sections/OverviewSection'
import { SecuritySection } from './sections/SecuritySection'
import { SystemSection } from './sections/SystemSection'
import { TrafficSection } from './sections/TrafficSection'

const menuItems = [
  { key: 'dashboard', label: '仪表盘', icon: <FiActivity /> },
  { key: 'traffic', label: '流量管理', icon: <FiShuffle /> },
  { key: 'security', label: '安全防护', icon: <FiShield /> },
  { key: 'observability', label: '观测中心', icon: <FiEye /> },
  { key: 'system', label: '系统设置', icon: <FiSliders /> },
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
  const [cdnStatus, setCdnStatus] = useState<Record<string, CDNStatus>>({})
  const [captcha, setCaptcha] = useState<CaptchaResponse | null>(null)
  const [loginForm, setLoginForm] = useState<LoginPayload>({
    username: '',
    password: '',
    captcha_id: '',
    captcha_code: '',
  })
  const [advancedMode, setAdvancedMode] = useState(false)
  const [jsonDraft, setJsonDraft] = useState('')
  const [statusMessage, setStatusMessage] = useState('')
  const [errorMessage, setErrorMessage] = useState('')
  const [loading, setLoading] = useState({
    health: false,
    logStats: false,
    config: false,
    logLevel: false,
    applying: false,
    reload: false,
  })

  const breadcrumbs = useMemo(() => {
    const current = menuItems.find((item) => item.key === activeSection)
    return `控制台 / ${current?.label || '仪表盘'}`
  }, [activeSection])

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
    if (advancedMode) {
      setJsonDraft(JSON.stringify(configDraft, null, 2))
    }
  }, [advancedMode, configDraft])

  useEffect(() => {
    if (token) {
      return
    }
    const loadCaptcha = async () => {
      setErrorMessage('')
      try {
        const data = await fetchCaptcha('default')
        setCaptcha(data)
        setLoginForm((prev) => ({ ...prev, captcha_id: data.captcha_id }))
      } catch (error) {
        setErrorMessage(error instanceof Error ? error.message : '验证码获取失败')
      }
    }
    loadCaptcha()
  }, [token])

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

  const handleAddDomain = (rule: DomainRule) => {
    const normalized = normalizeConfig(configDraft)
    setConfigDraft({
      ...normalized,
      routing: {
        ...normalized.routing,
        domains: [...(normalized.routing?.domains || []), rule],
      },
    })
  }

  const handleUpdateDomain = (index: number, rule: DomainRule) => {
    const normalized = normalizeConfig(configDraft)
    const domains = normalized.routing?.domains || []
    setConfigDraft({
      ...normalized,
      routing: {
        ...normalized.routing,
        domains: domains.map((item, idx) => (idx === index ? rule : item)),
      },
    })
  }

  const handleRemoveDomain = (index: number) => {
    const normalized = normalizeConfig(configDraft)
    const domains = normalized.routing?.domains || []
    setConfigDraft({
      ...normalized,
      routing: {
        ...normalized.routing,
        domains: domains.filter((_, idx) => idx !== index),
      },
    })
  }

  const handleApplyConfig = async () => {
    if (!token) {
      setErrorMessage('请先登录后再应用配置')
      return
    }
    setLoading((prev) => ({ ...prev, applying: true }))
    setErrorMessage('')
    setStatusMessage('')
    try {
      let payload = configDraft
      if (advancedMode) {
        payload = JSON.parse(jsonDraft) as Config
        setConfigDraft(payload)
      }
      await applyConfigRequest(token, payload)
      setStatusMessage('配置已提交并应用')
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : '配置应用失败')
    } finally {
      setLoading((prev) => ({ ...prev, applying: false }))
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
    }
  }

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
                  {captcha?.image_base64 && (
                    <img
                      className="captcha-image"
                      src={`data:image/png;base64,${captcha.image_base64}`}
                      alt="验证码"
                    />
                  )}
                </div>
              </div>
            </div>
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
        <TopBar breadcrumbs={breadcrumbs} token={token} />
        <main className="main">
          <div>
            <h1 className="page-title">BFF 负载均衡网关管理端</h1>
            <p className="page-subtitle">沉浸式暗色风格，支持配置实时感知与审计流程</p>
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
              configDraft={configDraft}
              onConfigChange={setConfigDraft}
              onAddDomain={handleAddDomain}
              onUpdateDomain={handleUpdateDomain}
              onRemoveDomain={handleRemoveDomain}
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
          {activeSection === 'observability' && <ObservabilitySection />}
          {activeSection === 'system' && (
            <SystemSection
              logLevel={logLevel}
              loadingLogLevel={loading.logLevel}
              onLogLevelChange={handleLogLevelChange}
              loadingReload={loading.reload}
              onReload={handleReload}
              loginForm={loginForm}
              onLoginFormChange={setLoginForm}
              onFetchCaptcha={handleFetchCaptcha}
              captcha={captcha}
              onLogin={handleLogin}
            />
          )}
          <ConfigPanel
            configDraft={configDraft}
            configSnapshot={config}
            advancedMode={advancedMode}
            jsonDraft={jsonDraft}
            loadingApplying={loading.applying}
            onToggleMode={setAdvancedMode}
            onJsonChange={setJsonDraft}
            onConfigChange={setConfigDraft}
            onApply={handleApplyConfig}
            onReset={() => {
              setConfigDraft(normalizeConfig(config))
              setStatusMessage('已恢复到最近一次读取的配置')
            }}
          />
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
