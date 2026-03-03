package app

import (
	"net"
	"net/http"
	"net/url"

	"YoBFF/internal/config"
)

// WithHSTS 为处理器添加 HSTS 响应头。
// 参数：next 为下一个处理器，manager 为配置管理器。
// 返回：包装后的处理器。
// 异常：无。
func WithHSTS(next http.Handler, manager *config.Manager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg := manager.CurrentConfig()
		if cfg.Security.EnableHSTS && r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

// RedirectHTTPToHTTPS 将 HTTP 请求重定向至 HTTPS。
// 参数：next 为备用处理器（当请求已加密时使用），httpsListenAddr 为 HTTPS 监听地址。
// 返回：重定向处理器。
// 异常：无。
func RedirectHTTPToHTTPS(next http.Handler, httpsListenAddr string) http.Handler {
	targetPort := resolvePort(httpsListenAddr)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS != nil {
			next.ServeHTTP(w, r)
			return
		}
		target := &url.URL{
			Scheme:   "https",
			Host:     buildHTTPSHost(r.Host, targetPort),
			Path:     r.URL.Path,
			RawQuery: r.URL.RawQuery,
		}
		http.Redirect(w, r, target.String(), http.StatusMovedPermanently)
	})
}

// resolvePort 从地址字符串中提取端口号。
// 参数：listenAddr 为监听地址（如 :8443）。
// 返回：端口号字符串，若解析失败默认返回 443。
// 异常：无。
func resolvePort(listenAddr string) string {
	_, port, err := net.SplitHostPort(listenAddr)
	if err == nil {
		return port
	}
	return "443"
}

// buildHTTPSHost 构建重定向目标的 Host 字符串。
// 参数：sourceHost 为原始 Host，targetPort 为目标端口。
// 返回：包含目标端口的 Host 字符串。
// 异常：无。
func buildHTTPSHost(sourceHost string, targetPort string) string {
	host, _, err := net.SplitHostPort(sourceHost)
	if err != nil {
		host = sourceHost
	}
	if targetPort == "443" {
		return host
	}
	return net.JoinHostPort(host, targetPort)
}
