import type { CaptchaResponse, LoginPayload } from '../admin/types'

type SystemSectionProps = {
  logLevel: string
  loadingLogLevel: boolean
  onLogLevelChange: (level: string) => void
  loadingReload: boolean
  onReload: () => void
  loginForm: LoginPayload
  onLoginFormChange: (next: LoginPayload) => void
  onFetchCaptcha: () => void
  captcha: CaptchaResponse | null
  onLogin: () => void
}

const logLevels = ['debug', 'info', 'warn', 'error']

export function SystemSection({
  logLevel,
  loadingLogLevel,
  onLogLevelChange,
  loadingReload,
  onReload,
  loginForm,
  onLoginFormChange,
  onFetchCaptcha,
  captcha,
  onLogin,
}: SystemSectionProps) {
  const captchaSrc = captcha?.image_base64
    ? (captcha.image_base64.startsWith('data:image/')
      ? captcha.image_base64
      : `data:image/png;base64,${captcha.image_base64}`)
    : ''
  return (
    <>
      <div className="grid grid-2">
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">日志级别</h3>
            <span className="pill">GET/PUT /admin/api/v1/log/level</span>
          </div>
          <div className="inline">
            <select
              className="select"
              value={logLevel}
              onChange={(event) => onLogLevelChange(event.target.value)}
            >
              {logLevels.map((level) => (
                <option key={level} value={level}>
                  {level.toUpperCase()}
                </option>
              ))}
            </select>
            {loadingLogLevel && <span className="muted">同步中...</span>}
          </div>
        </div>
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">热重载</h3>
            <span className="pill">POST /admin/api/v1/config/reload</span>
          </div>
          <button className="button secondary" onClick={onReload} disabled={loadingReload}>
            {loadingReload ? '触发中...' : '触发热重载'}
          </button>
        </div>
      </div>
      <div className="card">
        <div className="card-header">
          <h3 className="card-title">管理员</h3>
          <span className="pill">登录与鉴权</span>
        </div>
        <div className="stack">
          <div className="form-row">
            <div className="form-field">
              <label className="label">用户名</label>
              <input
                className="input"
                value={loginForm.username}
                onChange={(event) =>
                  onLoginFormChange({
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
                  onLoginFormChange({
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
                  onLoginFormChange({
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
                <button className="button secondary" onClick={onFetchCaptcha}>
                  获取验证码
                </button>
                {captchaSrc && (
                  <img
                    src={captchaSrc}
                    alt="验证码"
                    style={{ height: 40 }}
                  />
                )}
              </div>
            </div>
          </div>
          <button className="button primary" onClick={onLogin}>
            登录并获取 Token
          </button>
        </div>
      </div>
    </>
  )
}
