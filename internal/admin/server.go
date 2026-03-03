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
	mux.HandleFunc("/api/v1/config/cdn", s.cdnConfig)
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

// cdnConfig 提供 CDN 同步配置的管理与状态查询。
// 参数：w 为响应写入器，r 为请求对象。
// 返回：GET 返回配置与状态，PUT 更新配置。
// 异常：请求非法或应用失败时返回错误。
func (s *Server) cdnConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg := s.manager.CurrentConfig().CDNSync
		status := s.manager.GetCDNStatus()
		writeJSON(w, http.StatusOK, map[string]any{
			"config": cfg,
			"status": status,
		})
	case http.MethodPut:
		var payload config.CDNSyncConfig
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", err.Error(), r)
			return
		}

		fullCfg := s.manager.CurrentConfig()
		fullCfg.CDNSync = payload
		if err := s.manager.Apply(fullCfg); err != nil {
			writeError(w, http.StatusBadRequest, "config_apply_failed", err.Error(), r)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "updated",
			"config": payload,
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

// writeError 统一输出错误响应结构并补齐请求追踪信息。
// 参数：w 为响应写入器，status 为 HTTP 状态码，code 为错误码，message 为错误描述，r 为请求对象。
// 返回：无。
// 异常：无。
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

// withRequestID 为请求生成或透传 request_id，并写入响应头与上下文。
// 参数：next 为下一个处理器。
// 返回：包装后的处理器。
// 异常：无。
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

// requestIDFromContext 从上下文中读取 request_id。
// 参数：ctx 为请求上下文。
// 返回：request_id 字符串，若不存在则返回空字符串。
// 异常：无。
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

// newRequestID 生成请求追踪标识，用于跨日志与错误响应关联同一次请求。
// 参数：无。
// 返回：随机 request_id，随机源失败时回退为纳秒时间戳字符串。
// 异常：无。
func newRequestID() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return hex.EncodeToString(buffer)
}

// withAuth 对控制面请求执行 Bearer Token 鉴权。
// 参数：next 为下一个处理器，token 为期望的鉴权 Token。
// 返回：包装后的处理器。
// 异常：无，鉴权失败时直接返回固定错误结构。
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

// withRateLimit 对控制面请求执行固定窗口限流，按客户端地址计数。
// 参数：next 为下一个处理器，limiter 为限流器实例。
// 返回：包装后的处理器。
// 异常：无，触发限流时返回固定错误结构。
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

// clientKey 生成限流计数的客户端键，优先取代理链路的真实地址。
// 参数：r 为请求对象。
// 返回：用于限流的客户端标识字符串。
// 异常：无。
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

// newRateLimiter 创建固定窗口限流器。
// 参数：limit 为窗口内最大请求数，window 为统计窗口时长。
// 返回：限流器实例；当 limit 非法时返回 nil 表示不启用限流。
// 异常：无。
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

// Allow 判断给定 key 在当前窗口内是否允许请求。
// 参数：key 为客户端标识。
// 返回：remaining 为剩余配额，reset 为窗口重置时间，allowed 表示是否放行。
// 异常：无。
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

// parsePositiveInt 解析正整数文本，失败时回退为默认值。
// 参数：value 为待解析文本，fallback 为默认值。
// 返回：解析后的正整数或默认值。
// 异常：无。
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
