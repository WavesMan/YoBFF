package admin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"YoBFF/internal/config"
	"YoBFF/internal/logging"
)

// Server 封装控制平面管理接口依赖。
type Server struct {
	manager   *config.Manager
	runtime   *logging.Runtime
	pipeline  *logging.Pipeline
	authToken string
	limiter   *rateLimiter
}

// NewServer 创建控制平面 HTTP 服务实例。
// 参数：manager 为配置管理器，runtime 为日志运行时，pipeline 为异步日志管线。
// 返回：可注册路由的服务对象。
// 异常：无。
func NewServer(manager *config.Manager, runtime *logging.Runtime, pipeline *logging.Pipeline) *Server {
	token := strings.TrimSpace(config.EnvOrDefault("ADMIN_API_TOKEN", ""))
	limitValue := parsePositiveInt(config.EnvOrDefault("ADMIN_RATE_LIMIT_PER_MIN", "60"), 60)
	limiter := newRateLimiter(limitValue, time.Minute)
	return &Server{
		manager:   manager,
		runtime:   runtime,
		pipeline:  pipeline,
		authToken: token,
		limiter:   limiter,
	}
}

// Handler 构建控制平面的路由处理器。
// 参数：无。
// 返回：包含健康检查、配置管理和日志管理接口的处理器。
// 异常：无。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.health)
	mux.HandleFunc("/api/v1/config", s.config)
	mux.HandleFunc("/api/v1/config/reload", s.reload)
	mux.HandleFunc("/api/v1/log/level", s.logLevel)
	mux.HandleFunc("/api/v1/log/stats", s.logStats)
	handler := http.Handler(mux)
	handler = withAuth(handler, s.authToken)
	handler = withRateLimit(handler, s.limiter)
	handler = withRequestID(handler)
	return handler
}

// health 返回服务存活状态。
// 参数：w 为响应写入器，r 为请求对象。
// 返回：JSON 格式状态信息。
// 异常：无。
func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
	})
}

// config 提供配置读取与在线更新能力。
// 参数：w 为响应写入器，r 为请求对象。
// 返回：GET 返回当前配置，PUT 返回更新结果。
// 异常：请求体非法或配置应用失败时返回 400。
func (s *Server) config(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.manager.CurrentConfig())
	case http.MethodPut:
		var cfg config.Config
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&cfg); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", err.Error(), r)
			return
		}
		if err := s.manager.Apply(cfg); err != nil {
			writeError(w, http.StatusBadRequest, "config_apply_failed", err.Error(), r)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "updated",
		})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
	}
}

// reload 触发从磁盘重新加载配置文件。
// 参数：w 为响应写入器，r 为请求对象。
// 返回：重载状态响应。
// 异常：重载失败时返回 400。
func (s *Server) reload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
		return
	}
	if err := s.manager.Reload(); err != nil {
		writeError(w, http.StatusBadRequest, "config_reload_failed", err.Error(), r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "reloaded",
	})
}

// logLevel 提供日志级别查询与动态调整接口。
// 参数：w 为响应写入器，r 为请求对象。
// 返回：当前级别或更新结果。
// 异常：参数非法时返回 400，方法非法时返回 405。
func (s *Server) logLevel(w http.ResponseWriter, r *http.Request) {
	if s.runtime == nil {
		writeError(w, http.StatusServiceUnavailable, "logger_unavailable", "logger runtime unavailable", r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{
			"level": s.runtime.Level(),
		})
	case http.MethodPut:
		var payload struct {
			Level string `json:"level"`
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", err.Error(), r)
			return
		}
		if err := s.runtime.SetLevel(payload.Level); err != nil {
			writeError(w, http.StatusBadRequest, "log_level_invalid", err.Error(), r)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "updated",
			"level":  s.runtime.Level(),
		})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
	}
}

// logStats 返回异步日志管线运行统计信息。
// 参数：w 为响应写入器，r 为请求对象。
// 返回：累计日志计数快照。
// 异常：方法非法时返回 405。
func (s *Server) logStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
		return
	}
	if s.pipeline == nil {
		writeError(w, http.StatusServiceUnavailable, "log_pipeline_unavailable", "log pipeline unavailable", r)
		return
	}
	writeJSON(w, http.StatusOK, s.pipeline.Snapshot())
}

// writeJSON 统一输出 JSON 响应并设置状态码。
// 参数：w 为响应写入器，status 为 HTTP 状态码，data 为响应数据。
// 返回：无。
// 异常：编码失败时静默忽略。
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(data)
}

type errorResponse struct {
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

func writeError(w http.ResponseWriter, status int, code string, message string, r *http.Request) {
	requestID := requestIDFromContext(r.Context())
	if requestID == "" {
		requestID = newRequestID()
		w.Header().Set("X-Request-ID", requestID)
	}
	writeJSON(w, status, errorResponse{
		ErrorCode: code,
		Message:   message,
		RequestID: requestID,
	})
}

type contextKey string

const requestIDKey contextKey = "request_id"

func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if requestID == "" {
			requestID = newRequestID()
		}
		w.Header().Set("X-Request-ID", requestID)
		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requestIDFromContext(ctx context.Context) string {
	value := ctx.Value(requestIDKey)
	if value == nil {
		return ""
	}
	requestID, ok := value.(string)
	if !ok {
		return ""
	}
	return requestID
}

func newRequestID() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return hex.EncodeToString(buffer)
}

func withAuth(next http.Handler, token string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token == "" {
			writeError(w, http.StatusServiceUnavailable, "auth_not_configured", "admin auth token missing", r)
			return
		}
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		if header == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized", "missing authorization header", r)
			return
		}
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeError(w, http.StatusUnauthorized, "unauthorized", "invalid authorization scheme", r)
			return
		}
		if strings.TrimSpace(parts[1]) != token {
			writeError(w, http.StatusUnauthorized, "unauthorized", "invalid token", r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withRateLimit(next http.Handler, limiter *rateLimiter) http.Handler {
	if limiter == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := clientKey(r)
		remaining, reset, allowed := limiter.Allow(key)
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limiter.limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(reset.Unix(), 10))
		if !allowed {
			writeError(w, http.StatusTooManyRequests, "rate_limited", "too many requests", r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientKey(r *http.Request) string {
	xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			value := strings.TrimSpace(parts[0])
			if value != "" {
				return value
			}
		}
	}
	xRealIP := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if xRealIP != "" {
		return xRealIP
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	if r.RemoteAddr != "" {
		return r.RemoteAddr
	}
	return "unknown"
}

type rateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	entries map[string]*rateEntry
}

type rateEntry struct {
	count int
	reset time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	if limit <= 0 {
		return nil
	}
	return &rateLimiter{
		limit:   limit,
		window:  window,
		entries: make(map[string]*rateEntry),
	}
}

func (l *rateLimiter) Allow(key string) (int, time.Time, bool) {
	if l == nil || l.limit <= 0 {
		return 0, time.Now().Add(time.Minute), true
	}
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	entry, ok := l.entries[key]
	if !ok || now.After(entry.reset) {
		entry = &rateEntry{
			count: 0,
			reset: now.Add(l.window),
		}
		l.entries[key] = entry
	}
	if entry.count >= l.limit {
		return 0, entry.reset, false
	}
	entry.count++
	remaining := l.limit - entry.count
	return remaining, entry.reset, true
}

func parsePositiveInt(value string, fallback int) int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(trimmed)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
