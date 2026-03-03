package gateway

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"testing"

	"YoBFF/internal/config"
	"YoBFF/internal/logging"

	"go.uber.org/zap"
)

// TestHandler_BlocksWhenIPNotAllowed 验证来源 IP 不在白名单时返回 403。
func TestHandler_BlocksWhenIPNotAllowed(t *testing.T) {
	manager := buildManagerForTest(t, config.Config{
		Security: config.SecurityConfig{
			AllowedCIDRs:  []string{"10.0.0.0/8"},
			BlockPageHTML: "<html>blocked</html>",
		},
		Routing: config.RoutingConfig{
			Domains: []config.DomainRule{
				{Domain: "example.local", Upstream: "http://127.0.0.1:18080", ForceHTTPS: false},
			},
		},
	})

	logPipeline := logging.NewPipeline(16, zap.NewNop())
	defer logPipeline.Close()

	handler := NewHandler(manager, logPipeline)
	req := httptest.NewRequest(http.MethodGet, "http://example.local/hello", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	res := rec.Result()
	body, _ := io.ReadAll(res.Body)

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("状态码不匹配: got=%d", res.StatusCode)
	}
	if string(body) != "<html>blocked</html>" {
		t.Fatalf("响应体不匹配: got=%q", string(body))
	}

	stats := logPipeline.Snapshot()
	if stats.Blocked == 0 || stats.Total == 0 {
		t.Fatalf("日志统计未更新: %+v", stats)
	}
}

// TestHandler_RedirectsToHTTPSWhenForced 验证强制 HTTPS 路由规则会触发 301 重定向。
func TestHandler_RedirectsToHTTPSWhenForced(t *testing.T) {
	manager := buildManagerForTest(t, config.Config{
		Security: config.SecurityConfig{
			BlockPageHTML: "<html/>",
		},
		Routing: config.RoutingConfig{
			Domains: []config.DomainRule{
				{Domain: "example.local", Upstream: "http://127.0.0.1:18080", ForceHTTPS: true},
			},
		},
	})

	logPipeline := logging.NewPipeline(16, zap.NewNop())
	defer logPipeline.Close()

	handler := NewHandler(manager, logPipeline)
	req := httptest.NewRequest(http.MethodGet, "http://Example.Local:8080/hello?x=1", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusMovedPermanently {
		t.Fatalf("状态码不匹配: got=%d", res.StatusCode)
	}
	if location := res.Header.Get("Location"); location != "https://example.local/hello?x=1" {
		t.Fatalf("Location 不匹配: got=%q", location)
	}
}

// TestHandler_ProxiesToUpstream 验证路由匹配后会转发到上游并写入必要的代理头。
func TestHandler_ProxiesToUpstream(t *testing.T) {
	type captured struct {
		host           string
		realIP         string
		forwardedFor   string
		forwardedProto string
		path           string
	}
	var got captured

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = captured{
			host:           r.Host,
			realIP:         r.Header.Get("X-Real-IP"),
			forwardedFor:   r.Header.Get("X-Forwarded-For"),
			forwardedProto: r.Header.Get("X-Forwarded-Proto"),
			path:           r.URL.Path,
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("upstream-ok"))
	}))
	defer upstream.Close()

	manager := buildManagerForTest(t, config.Config{
		Security: config.SecurityConfig{
			BlockPageHTML: "<html/>",
		},
		Routing: config.RoutingConfig{
			Domains: []config.DomainRule{
				{Domain: "example.local", Upstream: upstream.URL, ForceHTTPS: false},
			},
		},
	})

	logPipeline := logging.NewPipeline(16, zap.NewNop())
	defer logPipeline.Close()

	handler := NewHandler(manager, logPipeline)
	req := httptest.NewRequest(http.MethodGet, "http://example.local/hello", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	res := rec.Result()
	body, _ := io.ReadAll(res.Body)

	if res.StatusCode != http.StatusOK {
		t.Fatalf("状态码不匹配: got=%d", res.StatusCode)
	}
	if string(body) != "upstream-ok" {
		t.Fatalf("响应体不匹配: got=%q", string(body))
	}
	if got.realIP != "127.0.0.1" {
		t.Fatalf("X-Real-IP 不匹配: got=%q", got.realIP)
	}
	if got.forwardedFor == "" {
		t.Fatalf("X-Forwarded-For 为空")
	}
	if got.forwardedProto != "http" {
		t.Fatalf("X-Forwarded-Proto 不匹配: got=%q", got.forwardedProto)
	}
	if got.path != "/hello" {
		t.Fatalf("Path 不匹配: got=%q", got.path)
	}

	stats := logPipeline.Snapshot()
	if stats.Proxied == 0 || stats.Total == 0 {
		t.Fatalf("日志统计未更新: %+v", stats)
	}
}

// TestHandler_ProxiesToUpstreamOverTLS 验证 TLS 请求会设置 X-Forwarded-Proto 为 https。
func TestHandler_ProxiesToUpstreamOverTLS(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Forwarded-Proto"); got != "https" {
			t.Fatalf("X-Forwarded-Proto 不匹配: got=%q", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	manager := buildManagerForTest(t, config.Config{
		Security: config.SecurityConfig{
			BlockPageHTML: "<html/>",
		},
		Routing: config.RoutingConfig{
			Domains: []config.DomainRule{
				{Domain: "example.local", Upstream: upstream.URL, ForceHTTPS: false},
			},
		},
	})

	logPipeline := logging.NewPipeline(16, zap.NewNop())
	defer logPipeline.Close()

	handler := NewHandler(manager, logPipeline)
	req := httptest.NewRequest(http.MethodGet, "https://example.local/hello", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Result().StatusCode != http.StatusOK {
		t.Fatalf("状态码不匹配: got=%d", rec.Result().StatusCode)
	}
}

// TestHandler_UpstreamErrorReturns502 验证上游不可达时返回 502。
func TestHandler_UpstreamErrorReturns502(t *testing.T) {
	manager := buildManagerForTest(t, config.Config{
		Security: config.SecurityConfig{
			BlockPageHTML: "<html/>",
		},
		Routing: config.RoutingConfig{
			Domains: []config.DomainRule{
				{Domain: "example.local", Upstream: "http://127.0.0.1:0", ForceHTTPS: false},
			},
		},
	})

	logPipeline := logging.NewPipeline(16, zap.NewNop())
	defer logPipeline.Close()

	handler := NewHandler(manager, logPipeline)
	req := httptest.NewRequest(http.MethodGet, "http://example.local/hello", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusBadGateway {
		t.Fatalf("状态码不匹配: got=%d", rec.Result().StatusCode)
	}
}

// TestParseClientIP_CoversBranches 验证 RemoteAddr 解析分支与失败兜底。
func TestParseClientIP_CoversBranches(t *testing.T) {
	if got := parseClientIP("10.0.0.1:1234"); got != netip.MustParseAddr("10.0.0.1") {
		t.Fatalf("解析结果不匹配: got=%s", got)
	}
	if got := parseClientIP("10.0.0.1"); got != netip.MustParseAddr("10.0.0.1") {
		t.Fatalf("解析结果不匹配: got=%s", got)
	}
	if got := parseClientIP("not-an-ip"); got != netip.IPv4Unspecified() {
		t.Fatalf("解析结果不匹配: got=%s", got)
	}
}

// TestStatusRecorderWrite_CoversBranches 验证 Write 的默认状态码与 nil writer 错误分支。
func TestStatusRecorderWrite_CoversBranches(t *testing.T) {
	rec := &statusRecorder{ResponseWriter: httptest.NewRecorder()}
	if _, err := rec.Write([]byte("ok")); err != nil {
		t.Fatalf("Write 失败: %v", err)
	}
	if rec.statusCode != http.StatusOK {
		t.Fatalf("statusCode 不匹配: got=%d", rec.statusCode)
	}

	empty := &statusRecorder{}
	if _, err := empty.Write([]byte("x")); err == nil {
		t.Fatalf("ResponseWriter 为空应返回错误")
	}
}

// buildManagerForTest 构建用于单元测试的配置管理器，并应用指定配置。
// 参数：t 为测试上下文，cfg 为待应用配置。
// 返回：可用于数据面处理器的 Manager 实例。
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
