package app

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"YoBFF/internal/config"
)

// TestRedirectHTTPToHTTPS_RedirectsToConfiguredPort 验证 HTTP 请求会重定向到配置的 HTTPS 端口。
func TestRedirectHTTPToHTTPS_RedirectsToConfiguredPort(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("不应走 next: %s", r.URL.Path)
	})
	handler := RedirectHTTPToHTTPS(nextHandler, ":8443")

	req := httptest.NewRequest(http.MethodGet, "http://example.com:8080/hello?x=1", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusMovedPermanently {
		t.Fatalf("状态码不匹配: got=%d", res.StatusCode)
	}
	if location := res.Header.Get("Location"); location != "https://example.com:8443/hello?x=1" {
		t.Fatalf("Location 不匹配: got=%q", location)
	}
}

// TestRedirectHTTPToHTTPS_OmitsPortWhen443 验证 HTTPS 端口为 443 时省略端口。
func TestRedirectHTTPToHTTPS_OmitsPortWhen443(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("不应走 next: %s", r.URL.Path)
	})
	handler := RedirectHTTPToHTTPS(nextHandler, ":443")

	req := httptest.NewRequest(http.MethodGet, "http://example.com:8080/hello", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusMovedPermanently {
		t.Fatalf("状态码不匹配: got=%d", res.StatusCode)
	}
	if location := res.Header.Get("Location"); location != "https://example.com/hello" {
		t.Fatalf("Location 不匹配: got=%q", location)
	}
}

// TestRedirectHTTPToHTTPS_PassesThroughWhenTLS 验证已是 TLS 请求时直接透传到下游处理器。
func TestRedirectHTTPToHTTPS_PassesThroughWhenTLS(t *testing.T) {
	var nextHit bool
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextHit = true
		w.WriteHeader(http.StatusOK)
	})
	handler := RedirectHTTPToHTTPS(nextHandler, ":8443")

	req := httptest.NewRequest(http.MethodGet, "https://example.com/hello", nil)
	req.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusOK {
		t.Fatalf("状态码不匹配: got=%d", rec.Result().StatusCode)
	}
	if !nextHit {
		t.Fatalf("next 未命中")
	}
}

// TestRedirectHTTPToHTTPS_HostWithoutPort 验证 Host 不带端口时的重定向逻辑。
func TestRedirectHTTPToHTTPS_HostWithoutPort(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("不应走 next: %s", r.URL.Path)
	})
	handler := RedirectHTTPToHTTPS(nextHandler, ":8443")

	req := httptest.NewRequest(http.MethodGet, "http://example.com/hello?x=1", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusMovedPermanently {
		t.Fatalf("状态码不匹配: got=%d", res.StatusCode)
	}
	if location := res.Header.Get("Location"); location != "https://example.com:8443/hello?x=1" {
		t.Fatalf("Location 不匹配: got=%q", location)
	}
}

// TestWithHSTS_AddsHeaderWhenEnabledAndTLS 验证启用 HSTS 且为 TLS 请求时设置响应头。
func TestWithHSTS_AddsHeaderWhenEnabledAndTLS(t *testing.T) {
	manager := buildManagerForTest(t, config.Config{
		Security: config.SecurityConfig{
			EnableHSTS: true,
		},
		Routing: config.RoutingConfig{
			DefaultUpstream: "http://127.0.0.1:18080",
		},
	})

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := WithHSTS(nextHandler, manager)

	req := httptest.NewRequest(http.MethodGet, "https://example.com/hello", nil)
	req.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusOK {
		t.Fatalf("状态码不匹配: got=%d", rec.Result().StatusCode)
	}
	if value := rec.Result().Header.Get("Strict-Transport-Security"); value == "" {
		t.Fatalf("未设置 HSTS 响应头")
	}
}

// TestWithHSTS_DoesNotAddHeaderWhenDisabled 验证禁用 HSTS 时不设置响应头。
func TestWithHSTS_DoesNotAddHeaderWhenDisabled(t *testing.T) {
	manager := buildManagerForTest(t, config.Config{
		Security: config.SecurityConfig{
			EnableHSTS: false,
		},
		Routing: config.RoutingConfig{
			DefaultUpstream: "http://127.0.0.1:18080",
		},
	})

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := WithHSTS(nextHandler, manager)

	req := httptest.NewRequest(http.MethodGet, "https://example.com/hello", nil)
	req.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if value := rec.Result().Header.Get("Strict-Transport-Security"); value != "" {
		t.Fatalf("不应设置 HSTS 响应头: got=%q", value)
	}
}

// buildManagerForTest 构建用于单元测试的配置管理器，并应用指定配置。
// 参数：t 为测试上下文，cfg 为待应用配置。
// 返回：可用于读取快照的 Manager 实例。
// 异常：初始化或应用配置失败时终止当前测试。
func buildManagerForTest(t *testing.T, cfg config.Config) *config.Manager {
	t.Helper()

	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("写入配置文件失败: %v", err)
	}
	manager, err := config.NewManager(configPath)
	if err != nil {
		t.Fatalf("初始化 Manager 失败: %v", err)
	}
	if err = manager.Apply(cfg); err != nil {
		t.Fatalf("应用配置失败: %v", err)
	}
	return manager
}
