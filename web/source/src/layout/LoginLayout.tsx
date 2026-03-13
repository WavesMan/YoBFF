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
import { useToast } from '../components/ui/Toast'
import { 
    RiUserLine, 
    RiLockPasswordLine, 
    RiShieldKeyholeLine,
    RiEyeLine,
    RiEyeOffLine
} from 'react-icons/ri'
import './LoginLayout.css'

type LoginLayoutProps = {
    onLoginSuccess: (token: string) => void
}

export function LoginLayout({ onLoginSuccess }: LoginLayoutProps) {
    const { success, error } = useToast()
    const [captcha, setCaptcha] = useState<CaptchaResponse | null>(null)
    const [captchaRequired, setCaptchaRequired] = useState(false)
    const [isLoading, setIsLoading] = useState(false)
    const [showPassword, setShowPassword] = useState(false)
    const [loginForm, setLoginForm] = useState<LoginPayload>({
        username: 'admin', // 默认填充以便测试，实际可为空
        password: '',
        captcha_id: '',
        captcha_code: '',
    })

    // 获取新的验证码
    const handleFetchCaptcha = useCallback(async () => {
        try {
            const data = await fetchCaptcha('default')
            setCaptcha(data)
            setLoginForm((prev) => ({ ...prev, captcha_id: data.captcha_id, captcha_code: '' }))
        } catch (err) {
            const errorObj = err instanceof Error ? err : new Error('验证码获取失败')
            error(errorObj)
            console.error('[Captcha Error]', errorObj.message)
        }
    }, [error])

    // 检查是否需要验证码
    const checkCaptcha = useCallback(async () => {
        try {
            const data = await fetchLoginCaptchaRequirement()
            if (!data.required) {
                setCaptchaRequired(false)
                setCaptcha(null)
                return
            }
            setCaptchaRequired(true)
            await handleFetchCaptcha()
        } catch (e) {
            console.error('[Check Captcha Error]', e)
        }
    }, [handleFetchCaptcha])

    useEffect(() => {
        checkCaptcha()
    }, [checkCaptcha])

    // 处理登录提交
    const handleLogin = async (e?: React.FormEvent) => {
        e?.preventDefault()
        
        if (!loginForm.username || !loginForm.password) {
            error('请输入用户名和密码')
            return
        }

        if (captchaRequired && !loginForm.captcha_code) {
            error('请输入验证码')
            return
        }

        setIsLoading(true)
        try {
            // 模拟一点延迟以展示加载动画（实际项目中可移除）
            // await new Promise(resolve => setTimeout(resolve, 800))

            const data = await loginAdmin(loginForm)
            const token = data.token
            if (!token) {
                throw new Error('登录未返回 token')
            }
            localStorage.setItem('yobff_token', token)
            success('登录成功')
            
            // 延迟跳转以展示成功状态
            setTimeout(() => {
                onLoginSuccess(token)
            }, 500)
        } catch (err) {
            const errorObj = err instanceof Error ? err : new Error('登录失败')
            error(errorObj)
            console.error('[Login Error]', errorObj.message)
            // 登录失败通常需要刷新验证码
            await checkCaptcha()
        } finally {
            setIsLoading(false)
        }
    }

    const captchaImageSrc = captcha?.image_base64
        ? captcha.image_base64.startsWith('data:image/')
            ? captcha.image_base64
            : `data:image/png;base64,${captcha.image_base64}`
        : ''

    return (
        <div className="login-page-body">
            {/* 登录卡片 */}
            <div className="login-card-modern">
                <div className="brand-section">
                    <div className="brand-logo">
                        <img src="/Logo.png" alt="YoBFF Logo" />
                    </div>
                    <h1 className="brand-title">YoBFF</h1>
                    <p className="brand-subtitle">管理端登录</p>
                </div>

                <form className="login-form" onSubmit={handleLogin}>
                    <div className="form-group">
                        <label className="form-label">用户名</label>
                        <div className="input-wrapper">
                            <RiUserLine className="input-icon" />
                            <input 
                                type="text" 
                                className="form-input" 
                                placeholder="admin" 
                                value={loginForm.username}
                                onChange={(e) => setLoginForm({ ...loginForm, username: e.target.value })}
                                required
                            />
                        </div>
                    </div>

                    <div className="form-group">
                        <label className="form-label">密码</label>
                        <div className="input-wrapper">
                            <RiLockPasswordLine className="input-icon" />
                            <input 
                                type={showPassword ? "text" : "password"}
                                className="form-input" 
                                style={{ paddingRight: '2.5rem' }}
                                placeholder="password" 
                                value={loginForm.password}
                                onChange={(e) => setLoginForm({ ...loginForm, password: e.target.value })}
                                required
                            />
                            <button
                                type="button"
                                className="password-toggle"
                                onClick={() => setShowPassword(!showPassword)}
                                tabIndex={-1}
                                title={showPassword ? "隐藏密码" : "显示密码"}
                            >
                                {showPassword ? <RiEyeOffLine /> : <RiEyeLine />}
                            </button>
                        </div>
                    </div>

                    {captchaRequired && (
                        <div className="form-group">
                            <label className="form-label">验证码</label>
                            <div className="captcha-group">
                                <div className="input-wrapper">
                                    <RiShieldKeyholeLine className="input-icon" />
                                    <input 
                                        type="text" 
                                        className="form-input" 
                                        placeholder="1234"
                                        value={loginForm.captcha_code}
                                        onChange={(e) => setLoginForm({ ...loginForm, captcha_code: e.target.value })}
                                        required
                                    />
                                </div>
                                <div 
                                    className="captcha-display" 
                                    title="点击刷新"
                                    onClick={handleFetchCaptcha}
                                >
                                    {captchaImageSrc ? (
                                        <img 
                                            src={captchaImageSrc} 
                                            alt="验证码" 
                                            className="captcha-image"
                                        />
                                    ) : (
                                        <span style={{ color: 'var(--text-tertiary)', fontSize: '0.8rem' }}>
                                            加载中...
                                        </span>
                                    )}
                                </div>
                            </div>
                        </div>
                    )}

                    <button 
                        type="submit" 
                        className={`btn-submit ${isLoading ? 'loading' : ''}`}
                        disabled={isLoading}
                    >
                        {isLoading ? (
                            <div className="spinner"></div>
                        ) : (
                            <span>登录</span>
                        )}
                    </button>
                </form>

                <div className="login-footer">
                    &copy; {new Date().getFullYear()} YoBFF Admin Panel
                </div>
            </div>
        </div>
    )
}
