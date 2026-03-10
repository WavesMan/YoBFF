import { useCallback, useEffect, useMemo, useState } from 'react'
import { FiActivity, FiEye, FiShuffle, FiSliders, FiLock } from 'react-icons/fi'
import './App.css'
import {
  fetchCaptcha,
  fetchHealthz,
  fetchLoginCaptchaRequirement,
  fetchLogLevel,
  fetchLogStats,
  loginAdmin,
  logoutAdmin,
  updateLogLevel as updateLogLevelRequest,
  reloadConfig as reloadConfigRequest,
} from './admin/api'
import type {
  CaptchaResponse,
  HealthzResponse,
  LoginPayload,
  LogStats,
} from './admin/types'
import { Sidebar } from './components/Sidebar'
import { TopBar } from './components/TopBar'
import { ObservabilitySection } from './sections/ObservabilitySection'
import { OverviewSection } from './sections/OverviewSection'
import { CertificatesSection } from './sections/CertificatesSection'
import { SystemSection } from './sections/SystemSection'
import { TrafficSection } from './sections/TrafficSection'

const menuItems = [
  { key: 'dashboard', label: '仪表盘', icon: <FiActivity /> },
  { key: 'traffic', label: '流量管理', icon: <FiShuffle /> },
  { key: 'certificates', label: '证书管理', icon: <FiLock /> },
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
  const [captcha, setCaptcha] = useState<CaptchaResponse | null>(null)
  const [captchaRequired, setCaptchaRequired] = useState(false)
  const [loginForm, setLoginForm] = useState<LoginPayload>({
    username: '',
    password: '',
    captcha_id: '',
    captcha_code: '',
  })
  const [operatorName] = useState(() => localStorage.getItem('yobff_operator') || '')
  const [statusMessage, setStatusMessage] = useState('')
  const [errorMessage, setErrorMessage] = useState('')
  const [loading, setLoading] = useState({
    health: false,
    logStats: false,
    logLevel: false,
    reload: false,
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
        </main>
      </div>
    </div>
  )
}

export default App
