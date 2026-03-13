import { useCallback, useEffect, useState } from 'react'
import {
  fetchCaptcha,
  fetchLoginCaptchaRequirement,
  loginAdmin,
} from '../admin/api'
import type {
  CaptchaResponse,
  LoginPayload,
} from '../admin/types'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/Card'
import { Input } from '../components/ui/Input'
import { Button } from '../components/ui/Button'
import { useToast } from '../components/ui/Toast'
import { FiUser, FiLock, FiShield, FiRefreshCw } from 'react-icons/fi'

type LoginLayoutProps = {
  onLoginSuccess: (token: string) => void
}

export function LoginLayout({ onLoginSuccess }: LoginLayoutProps) {
  const { success, error } = useToast()
  const [captcha, setCaptcha] = useState<CaptchaResponse | null>(null)
  const [captchaRequired, setCaptchaRequired] = useState(false)
  const [loginForm, setLoginForm] = useState<LoginPayload>({
    username: '',
    password: '',
    captcha_id: '',
    captcha_code: '',
  })

  const checkCaptcha = useCallback(async () => {
     try {
        const data = await fetchLoginCaptchaRequirement()
        if (!data.required) {
            setCaptchaRequired(false)
            setCaptcha(null)
            return
        }
        setCaptchaRequired(true)
        const cap = await fetchCaptcha('default')
        setCaptcha(cap)
        setLoginForm(prev => ({...prev, captcha_id: cap.captcha_id}))
     } catch (e) {
         console.error(e)
     }
  }, [])

  useEffect(() => {
    checkCaptcha()
  }, [checkCaptcha])

  const handleFetchCaptcha = async () => {
    try {
      const data = await fetchCaptcha('default')
      setCaptcha(data)
      setLoginForm((prev) => ({ ...prev, captcha_id: data.captcha_id }))
    } catch (err) {
      error(err instanceof Error ? err.message : '验证码获取失败')
    }
  }

  const handleLogin = async () => {
    try {
      const data = await loginAdmin(loginForm)
      if (!data.token) {
        throw new Error('登录未返回 token')
      }
      localStorage.setItem('yobff_token', data.token)
      success('登录成功')
      onLoginSuccess(data.token)
    } catch (err) {
      error(err instanceof Error ? err.message : '登录失败')
      await checkCaptcha()
    }
  }

  const captchaImageSrc = captcha?.image_base64
    ? captcha.image_base64.startsWith('data:image/')
      ? captcha.image_base64
      : `data:image/png;base64,${captcha.image_base64}`
    : ''

  return (
    <div className="login-overlay">
      <Card className="login-card">
        <CardHeader className="login-header" style={{ flexDirection: 'column', alignItems: 'center', textAlign: 'center' }}>
          <div className="brand login-brand">
            <span className="brand-icon" />
            <span className="brand-text">YoBFF</span>
          </div>
          <CardTitle className="login-title" style={{ fontSize: '1.5rem', margin: '10px 0' }}>管理端登录</CardTitle>
          <div className="muted">登录后进入管理面板进行配置与观测</div>
        </CardHeader>
        
        <CardContent className="stack">
          <div className="form-row">
            <div className="form-field">
              <label className="label">用户名</label>
              <Input
                icon={<FiUser />}
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
              <Input
                type="password"
                icon={<FiLock />}
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
                <Input
                  icon={<FiShield />}
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
                <div className="inline" style={{ display: 'flex', gap: '10px', alignItems: 'center' }}>
                  <Button variant="secondary" size="sm" onClick={handleFetchCaptcha} title="刷新验证码">
                    <FiRefreshCw />
                  </Button>
                  {captchaImageSrc && (
                    <img 
                      className="captcha-image" 
                      src={captchaImageSrc} 
                      alt="验证码" 
                      onClick={handleFetchCaptcha}
                      style={{ cursor: 'pointer', height: '36px', borderRadius: '4px' }}
                    />
                  )}
                </div>
              </div>
            </div>
          )}
          <Button className="w-full" onClick={handleLogin} style={{ width: '100%', marginTop: '10px' }}>
            登录并进入面板
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
