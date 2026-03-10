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

type LoginLayoutProps = {
  onLoginSuccess: (token: string) => void
}

export function LoginLayout({ onLoginSuccess }: LoginLayoutProps) {
  const [captcha, setCaptcha] = useState<CaptchaResponse | null>(null)
  const [captchaRequired, setCaptchaRequired] = useState(false)
  const [loginForm, setLoginForm] = useState<LoginPayload>({
    username: '',
    password: '',
    captcha_id: '',
    captcha_code: '',
  })
  const [statusMessage, setStatusMessage] = useState('')
  const [errorMessage, setErrorMessage] = useState('')

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
      setStatusMessage('登录成功')
      onLoginSuccess(data.token)
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : '登录失败')
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
