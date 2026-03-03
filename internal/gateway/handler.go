package gateway

import (
	"errors"
	"net"
	"net/http"
	"net/http/httputil"
	"net/netip"
	"strings"
	"time"

	"YoBFF/internal/config"
	"YoBFF/internal/logging"
)

// Handler 封装数据平面的流量校验与反向代理能力。
type Handler struct {
	manager *config.Manager
	logs    *logging.Pipeline
}

// NewHandler 创建数据平面请求处理器。
// 参数：manager 为配置管理器，logs 为异步日志管线。
// 返回：可挂载到 HTTP Server 的处理器。
// 异常：无。
func NewHandler(manager *config.Manager, logs *logging.Pipeline) *Handler {
	return &Handler{
		manager: manager,
		logs:    logs,
	}
}

// ServeHTTP 执行请求校验、路由匹配与透明转发。
// 参数：w 为响应写入器，r 为当前请求。
// 返回：无。
// 异常：校验失败返回 403，路由或上游异常返回 502。
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	clientIP := parseClientIP(r.RemoteAddr)
	host := normalizeHost(r.Host)
	cfg := h.manager.CurrentConfig()

	if !h.manager.IsIPAllowed(clientIP) || !h.manager.IsHostAuthorized(host) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(cfg.Security.BlockPageHTML))
		h.logs.Emit(logging.Event{
			Level:     "warn",
			Type:      "blocked",
			ClientIP:  clientIP.String(),
			Host:      host,
			Method:    r.Method,
			Path:      r.URL.Path,
			Status:    http.StatusForbidden,
			LatencyMS: time.Since(start).Milliseconds(),
			Message:   "source check failed",
		})
		return
	}

	route, ok := h.manager.ResolveRoute(host)
	if !ok {
		http.Error(w, "upstream route not found", http.StatusBadGateway)
		h.logs.Emit(logging.Event{
			Level:     "error",
			Type:      "proxy",
			ClientIP:  clientIP.String(),
			Host:      host,
			Method:    r.Method,
			Path:      r.URL.Path,
			Status:    http.StatusBadGateway,
			LatencyMS: time.Since(start).Milliseconds(),
			Message:   "route not found",
		})
		return
	}

	if route.ForceHTTPS && r.TLS == nil {
		targetURL := "https://" + host + r.URL.RequestURI()
		http.Redirect(w, r, targetURL, http.StatusMovedPermanently)
		h.logs.Emit(logging.Event{
			Level:     "info",
			Type:      "proxy",
			ClientIP:  clientIP.String(),
			Host:      host,
			Method:    r.Method,
			Path:      r.URL.Path,
			Status:    http.StatusMovedPermanently,
			LatencyMS: time.Since(start).Milliseconds(),
			Message:   "http redirected to https",
		})
		return
	}

	recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
	proxy := httputil.NewSingleHostReverseProxy(route.Target)
	director := proxy.Director
	proxy.Director = func(req *http.Request) {
		director(req)
		req.Host = r.Host
		req.Header.Set("X-Real-IP", clientIP.String())
		appendXForwardedFor(req.Header, clientIP.String())
		if r.TLS != nil {
			req.Header.Set("X-Forwarded-Proto", "https")
		} else {
			req.Header.Set("X-Forwarded-Proto", "http")
		}
	}
	proxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, err error) {
		http.Error(writer, "bad gateway", http.StatusBadGateway)
	}

	proxy.ServeHTTP(recorder, r)
	h.logs.Emit(logging.Event{
		Level:     "info",
		Type:      "proxy",
		ClientIP:  clientIP.String(),
		Host:      host,
		Method:    r.Method,
		Path:      r.URL.Path,
		Status:    recorder.statusCode,
		LatencyMS: time.Since(start).Milliseconds(),
		Message:   "request proxied",
	})
}

// appendXForwardedFor 维护代理链路中的 X-Forwarded-For 字段。
// 参数：header 为目标请求头，ip 为当前客户端地址。
// 返回：无。
// 异常：无。
func appendXForwardedFor(header http.Header, ip string) {
	original := header.Get("X-Forwarded-For")
	if original == "" {
		header.Set("X-Forwarded-For", ip)
		return
	}
	header.Set("X-Forwarded-For", original+", "+ip)
}

// parseClientIP 从 RemoteAddr 中解析客户端 IP。
// 参数：remoteAddr 为连接地址字符串。
// 返回：解析后的 IP，失败时返回 0.0.0.0。
// 异常：无。
func parseClientIP(remoteAddr string) netip.Addr {
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr))
	if err != nil {
		host = remoteAddr
	}
	ip, parseErr := netip.ParseAddr(host)
	if parseErr != nil {
		return netip.IPv4Unspecified()
	}
	return ip.Unmap()
}

// normalizeHost 将 Host 规范化为小写且不含端口的形式。
// 参数：rawHost 为请求头 Host 原始值。
// 返回：标准化域名字符串。
// 异常：无。
func normalizeHost(rawHost string) string {
	host := strings.TrimSpace(strings.ToLower(rawHost))
	if idx := strings.IndexByte(host, ':'); idx >= 0 {
		return host[:idx]
	}
	return host
}

// statusRecorder 用于捕获代理返回状态码。
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader 记录状态码并透传到底层响应写入器。
// 参数：statusCode 为业务响应状态。
// 返回：无。
// 异常：无。
func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// Write 在未显式写入状态时兜底为 200 并输出响应体。
// 参数：data 为响应体字节。
// 返回：写入字节数与错误。
// 异常：底层响应写入器为空时返回错误。
func (r *statusRecorder) Write(data []byte) (int, error) {
	if r.statusCode == 0 {
		r.statusCode = http.StatusOK
	}
	if r.ResponseWriter == nil {
		return 0, errors.New("response writer is nil")
	}
	return r.ResponseWriter.Write(data)
}
