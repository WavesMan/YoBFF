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
	"YoBFF/internal/store"

	"github.com/google/uuid"
)

// Handler 封装数据平面的流量校验与反向代理能力。
type Handler struct {
	manager *config.Manager
	logs    *logging.Pipeline
	store   *store.Store
}

// NewHandler 创建数据平面请求处理器。
// 参数：manager 为配置管理器，logs 为异步日志管线，store 为配置存储。
// 返回：可挂载到 HTTP Server 的处理器。
// 异常：无。
func NewHandler(manager *config.Manager, logs *logging.Pipeline, store *store.Store) *Handler {
	return &Handler{
		manager: manager,
		logs:    logs,
		store:   store,
	}
}

// ServeHTTP 执行请求校验、路由匹配与透明转发。
// 参数：w 为响应写入器，r 为当前请求。
// 返回：无。
// 异常：校验失败返回 403，路由或上游异常返回 502。
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	reqID := uuid.New().String()
	w.Header().Set("X-Request-ID", reqID)

	host := normalizeHost(r.Host)
	siteID := ""
	if h.store != nil {
		if resolvedSiteID, err := h.store.GetSiteIDByHostname(host); err == nil {
			siteID = resolvedSiteID
		}
	}
	emitLog := func(event logging.Event) {
		if h.logs != nil {
			h.logs.Emit(event)
		}
		if h.store != nil && siteID != "" {
			_ = h.store.SaveSiteTrafficLog(siteID, store.SiteTrafficLogWrite{
				Level:      event.Level,
				EventType:  event.Type,
				Message:    event.Message,
				RequestID:  event.RequestID,
				ClientIP:   event.ClientIP,
				Host:       event.Host,
				Method:     event.Method,
				Path:       event.Path,
				StatusCode: event.Status,
				LatencyMS:  event.LatencyMS,
			})
		}
	}
	// cfg := h.manager.CurrentConfig() // 已移除未使用的配置加载

	// 1. 尝试加载站点级配置
	var siteCfg config.Config
	useSiteConfig := false
	if h.store != nil {
		if sc, err := h.store.GetSiteConfigByHostname(host); err == nil {
			siteCfg = sc
			useSiteConfig = true
		}
	}
	currentCfg := h.manager.CurrentConfig()
	trustedProxyCIDRs := currentCfg.Security.TrustedProxyCIDRs
	if useSiteConfig && len(siteCfg.Security.TrustedProxyCIDRs) > 0 {
		trustedProxyCIDRs = siteCfg.Security.TrustedProxyCIDRs
	}
	clientIP := parseClientIP(r, trustedProxyCIDRs)
	peerIP := parseIPToken(r.RemoteAddr)
	if peerIP == netip.IPv4Unspecified() {
		peerIP = clientIP
	}

	// 2. 执行 IP 访问控制
	allowed := false
	denialReason := ""

	if useSiteConfig {
		// 站点级策略：只要配置了白名单（CIDR 或 CDN），则必须命中其一
		hasCDNRules := len(siteCfg.Security.AllowedCDNProviders) > 0
		hasCIDRRules := len(siteCfg.Security.AllowedCIDRs) > 0

		if !hasCDNRules && !hasCIDRRules {
			allowed = true
		} else {
			// 优先检查 CDN 提供商
			if hasCDNRules {
				if h.manager.IsIPAllowedByProviders(clientIP, siteCfg.Security.AllowedCDNProviders) {
					allowed = true
				} else {
					denialReason = "cdn source check failed"
				}
			}

			// 若 CDN 未命中且有自定义 CIDR，检查 CIDR
			if !allowed && hasCIDRRules {
				for _, cidr := range siteCfg.Security.AllowedCIDRs {
					if prefix, err := netip.ParsePrefix(cidr); err == nil {
						if prefix.Contains(clientIP) {
							allowed = true
							break
						}
					}
				}
				if !allowed {
					// 若同时配置了 CDN 和 CIDR，且都未命中，记录更具体的失败原因
					if denialReason != "" {
						denialReason = "both cdn and ip whitelist check failed"
					} else {
						denialReason = "ip whitelist check failed"
					}
				}
			}
		}
	} else {
		// 全局策略
		allowed = h.manager.IsIPAllowed(clientIP)
		if !allowed {
			denialReason = "global ip whitelist check failed"
		}
	}

	// 检查域名授权
	hostAuthorized := h.manager.IsHostAuthorized(host)
	if !hostAuthorized && allowed {
		denialReason = "host not authorized"
	}

	if !allowed || !hostAuthorized {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)

		blockPage := renderBlockPage(reqID, clientIP.String())
		_, _ = w.Write([]byte(blockPage))

		msg := "source check failed"
		if denialReason != "" {
			msg = denialReason
		}

		emitLog(logging.Event{
			Level:     "warn",
			Type:      "blocked",
			RequestID: reqID,
			ClientIP:  clientIP.String(),
			Host:      host,
			Method:    r.Method,
			Path:      r.URL.Path,
			Status:    http.StatusForbidden,
			LatencyMS: time.Since(start).Milliseconds(),
			Message:   msg,
		})
		return
	}

	route, ok := h.manager.ResolveRoute(host)
	if !ok {
		http.Error(w, "upstream route not found", http.StatusBadGateway)
		emitLog(logging.Event{
			Level:     "error",
			Type:      "proxy",
			RequestID: reqID,
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
		emitLog(logging.Event{
			Level:     "info",
			Type:      "proxy",
			RequestID: reqID,
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
		req.Header.Set("X-Request-ID", reqID)
		appendXForwardedFor(req.Header, peerIP.String())
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
	emitLog(logging.Event{
		Level:     "info",
		Type:      "proxy",
		RequestID: reqID,
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

// parseClientIP 按受信代理链策略解析客户端 IP。
// 参数：r 为当前请求对象，trustedProxyCIDRs 为受信代理网段列表。
// 返回：解析后的 IP，失败时返回 0.0.0.0。
// 异常：无。
func parseClientIP(r *http.Request, trustedProxyCIDRs []string) netip.Addr {
	remoteIP := parseIPToken(r.RemoteAddr)
	if remoteIP == netip.IPv4Unspecified() {
		return remoteIP
	}
	if !isTrustedProxyIP(remoteIP, trustedProxyCIDRs) {
		return remoteIP
	}
	xffIPs := parseForwardedForIPs(r.Header.Get("X-Forwarded-For"))
	if len(xffIPs) > 0 {
		for idx := len(xffIPs) - 1; idx >= 0; idx-- {
			if !isTrustedProxyIP(xffIPs[idx], trustedProxyCIDRs) {
				return xffIPs[idx]
			}
		}
		return xffIPs[0]
	}
	realIP := parseIPToken(r.Header.Get("X-Real-IP"))
	if realIP != netip.IPv4Unspecified() {
		return realIP
	}
	return remoteIP
}

func parseForwardedForIPs(raw string) []netip.Addr {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	ips := make([]netip.Addr, 0, len(parts))
	for _, part := range parts {
		ip := parseIPToken(part)
		if ip == netip.IPv4Unspecified() {
			continue
		}
		ips = append(ips, ip)
	}
	return ips
}

func isTrustedProxyIP(ip netip.Addr, trustedProxyCIDRs []string) bool {
	for _, cidr := range trustedProxyCIDRs {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(cidr))
		if err != nil {
			continue
		}
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}

func parseIPToken(raw string) netip.Addr {
	value := strings.TrimSpace(raw)
	if value == "" {
		return netip.IPv4Unspecified()
	}
	if ip, err := netip.ParseAddr(value); err == nil {
		return ip.Unmap()
	}
	host, _, err := net.SplitHostPort(value)
	if err != nil {
		return netip.IPv4Unspecified()
	}
	if ip, parseErr := netip.ParseAddr(strings.TrimSpace(host)); parseErr == nil {
		return ip.Unmap()
	}
	return netip.IPv4Unspecified()
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
