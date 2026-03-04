package admin

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

// captchaImage 生成一次性验证码图片（Base64）并返回给前端。
// 参数：w 为响应写入器，r 为请求对象。
// 返回：成功返回 captcha_id 与 image_base64。
// 异常：验证码服务不可用或生成失败时返回 503。
func (s *Server) captchaImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
		return
	}
	key := clientKey(r)
	preset := r.URL.Query().Get("preset")
	if s.captcha == nil {
		if s.runtime != nil {
			s.runtime.Logger().Warn("验证码服务不可用", zap.String("client", key))
		}
		writeError(w, http.StatusServiceUnavailable, "captcha_unavailable", "captcha unavailable", r)
		return
	}
	id, imageBase64, err := s.captcha.Generate(preset)
	if err != nil {
		if s.runtime != nil {
			s.runtime.Logger().Error("生成验证码失败", zap.String("client", key), zap.String("preset", preset), zap.Error(err))
		}
		writeError(w, http.StatusServiceUnavailable, "captcha_unavailable", "captcha unavailable", r)
		return
	}
	if s.runtime != nil {
		s.runtime.Logger().Debug("生成验证码成功", zap.String("client", key), zap.String("preset", preset), zap.String("captcha_id", id))
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"captcha_id":   id,
		"image_base64": imageBase64,
		"preset":       preset,
	})
}

// login 验证管理员账号密码并返回控制面 Token。
// 参数：w 为响应写入器，r 为请求对象。
// 返回：成功返回 token。
// 异常：首次登录失败后将要求验证码；验证码缺失或错误时返回 400。
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
		return
	}

	var creds struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		CaptchaID   string `json:"captcha_id"`
		CaptchaCode string `json:"captcha_code"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10))
	if err := decoder.Decode(&creds); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json", r)
		return
	}

	auth := s.manager.CurrentConfig().ControlPlane.Auth
	if auth.Username == "" || auth.Password == "" {
		writeError(w, http.StatusServiceUnavailable, "auth_not_configured", "admin auth not configured", r)
		return
	}

	key := clientKey(r)
	captchaRequired := s.guard != nil && s.guard.RequireCaptcha(key)
	if s.runtime != nil {
		s.runtime.Logger().Debug("管理员登录请求", zap.String("client", key), zap.Bool("captcha_required", captchaRequired))
	}
	if captchaRequired {
		if s.captcha == nil {
			if s.runtime != nil {
				s.runtime.Logger().Warn("验证码服务不可用", zap.String("client", key))
			}
			writeError(w, http.StatusServiceUnavailable, "captcha_unavailable", "captcha unavailable", r)
			return
		}
		if creds.CaptchaID == "" || creds.CaptchaCode == "" {
			if s.runtime != nil {
				s.runtime.Logger().Warn("缺少验证码参数", zap.String("client", key))
			}
			writeError(w, http.StatusBadRequest, "captcha_required", "captcha required", r)
			return
		}
		if !s.captcha.Verify(creds.CaptchaID, creds.CaptchaCode) {
			if s.guard != nil {
				s.guard.OnFailure(key)
			}
			if s.runtime != nil {
				s.runtime.Logger().Warn("验证码校验失败", zap.String("client", key))
			}
			writeError(w, http.StatusBadRequest, "invalid_captcha", "invalid captcha", r)
			return
		}
	}

	if creds.Username != auth.Username || creds.Password != auth.Password {
		if s.guard != nil {
			s.guard.OnFailure(key)
		}
		if s.runtime != nil {
			s.runtime.Logger().Warn("管理员登录失败", zap.String("client", key))
		}
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid username or password", r)
		return
	}

	if s.guard != nil {
		s.guard.OnSuccess(key)
	}
	if s.runtime != nil {
		s.runtime.Logger().Info("管理员登录成功", zap.String("client", key))
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"token": auth.Token,
	})
}

func (s *Server) loginCaptchaRequirement(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
		return
	}
	key := clientKey(r)
	required := s.guard != nil && s.guard.RequireCaptcha(key)
	writeJSON(w, http.StatusOK, map[string]bool{
		"required": required,
	})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
		return
	}
	key := clientKey(r)
	if s.runtime != nil {
		s.runtime.Logger().Info("管理员登出", zap.String("client", key))
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "logged_out",
	})
}
