import { useEffect, useState } from 'react'
import './App.css'
import {
  fetchHealthz,
  fetchLogLevel,
  fetchLogStats,
  logoutAdmin,
  reloadConfig as reloadConfigRequest,
  updateLogLevel as updateLogLevelRequest,
} from './admin/api'
import type {
  HealthzResponse,
  LogStats,
} from './admin/types'
import { CertificatesSection } from './sections/CertificatesSection'
import { ObservabilitySection } from './sections/ObservabilitySection'
import { OverviewSection } from './sections/OverviewSection'
import { SystemSection } from './sections/SystemSection'
import { TrafficSection } from './sections/TrafficSection'
import { UiDemoSection } from './sections/UiDemoSection'
import { WeaverSection } from './sections/WeaverSection'
import LoginLayout from './layout/LoginLayout'
import { MainLayout } from './layout/MainLayout'

import { ToastProvider } from './components/ui/Toast'

function App() {
  const [activeSection, setActiveSection] = useState('dashboard')
  const [token, setToken] = useState(() => localStorage.getItem('yobff_token'))
  const [health, setHealth] = useState<HealthzResponse | null>(null)
  const [logStats, setLogStats] = useState<LogStats | null>(null)
  const [logLevel, setLogLevel] = useState('info')
  const [operatorName] = useState(() => localStorage.getItem('yobff_operator') || '')
  const [statusMessage, setStatusMessage] = useState('')
  const [errorMessage, setErrorMessage] = useState('')
  const [loading, setLoading] = useState({
    health: false,
    logStats: false,
    logLevel: false,
    reload: false,
  })

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

  return (
    <ToastProvider>
      {!token ? (
        <LoginLayout onLoginSuccess={setToken} />
      ) : (
        <MainLayout
          token={token}
          onLogout={handleLogout}
          activeSection={activeSection}
          setActiveSection={setActiveSection}
          errorMessage={errorMessage}
          statusMessage={statusMessage}
        >
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
              token={token || ''}
            />
          )}
          {activeSection === 'weaver' && (
            <WeaverSection
              token={token || ''}
              operator={operatorName || 'unknown'}
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
          {activeSection === 'uidemo' && <UiDemoSection />}
        </MainLayout>
      )}
    </ToastProvider>
  )
}

export default App
