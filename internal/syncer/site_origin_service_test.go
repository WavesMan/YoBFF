package syncer

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"

	"YoBFF/internal/config"
	"YoBFF/internal/store"
)

// buildManagerForTest 构造用于单测的配置管理器，避免依赖真实配置文件与外部环境。
func buildManagerForTest(t *testing.T, cfgJSON string) *config.Manager {
	t.Helper()
	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, "config.json")
	if err := os.WriteFile(configPath, []byte(cfgJSON), 0o600); err != nil {
		t.Fatalf("写入配置失败: %v", err)
	}
	manager, err := config.NewManager(configPath)
	if err != nil {
		t.Fatalf("初始化 Manager 失败: %v", err)
	}
	return manager
}

// buildStoreForTest 构造用于单测的 SQLite 存储，并在测试结束时自动关闭。
func buildStoreForTest(t *testing.T) *store.Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "store.db")
	s, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("初始化存储失败: %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})
	return s
}

// TestSiteOriginHelpers 覆盖站点回源同步内部辅助函数的关键分支。
func TestSiteOriginHelpers(t *testing.T) {
	t.Run("parseRFC3339", func(t *testing.T) {
		if !parseRFC3339("").IsZero() {
			t.Fatalf("空字符串应返回零值")
		}
		if !parseRFC3339("bad").IsZero() {
			t.Fatalf("非法时间应返回零值")
		}
		got := parseRFC3339("2026-03-12T10:11:12Z")
		if got.IsZero() || got.UTC().Format(time.RFC3339) != "2026-03-12T10:11:12Z" {
			t.Fatalf("解析结果错误: %v", got)
		}
	})

	t.Run("normalizeProviders", func(t *testing.T) {
		got := normalizeProviders([]string{" Cloudflare ", "cloudflare", "", "  ", "TENCENT"})
		if len(got) != 2 || got[0] != "cloudflare" || got[1] != "tencent" {
			t.Fatalf("normalizeProviders 结果错误: %+v", got)
		}
	})

	t.Run("parsePositiveInt", func(t *testing.T) {
		if parsePositiveInt("", 7) != 7 {
			t.Fatalf("空串应回退默认值")
		}
		if parsePositiveInt("0", 7) != 7 {
			t.Fatalf("0 应回退默认值")
		}
		if parsePositiveInt("-1", 7) != 7 {
			t.Fatalf("负数应回退默认值")
		}
		if parsePositiveInt("12x", 7) != 7 {
			t.Fatalf("包含非数字字符应回退默认值")
		}
		if parsePositiveInt("12", 7) != 12 {
			t.Fatalf("应返回解析值")
		}
	})
}

// TestNewSiteOriginService_EnvParsing 验证环境变量读取与边界值截断的逻辑。
func TestNewSiteOriginService_EnvParsing(t *testing.T) {
	t.Setenv("SITE_CDN_ORIGIN_SYNC_MAX_CONCURRENCY", "0")
	t.Setenv("SITE_CDN_ORIGIN_SYNC_TICK_SECONDS", "2")
	t.Setenv("SITE_CDN_ORIGIN_SYNC_BACKOFF_BASE_SECONDS", "3")
	t.Setenv("SITE_CDN_ORIGIN_SYNC_BACKOFF_MAX_SECONDS", "4")
	t.Setenv("SITE_CDN_ORIGIN_SYNC_JITTER_PERMILLE", "9999")

	manager := buildManagerForTest(t, `{
  "routing": { "defaultUpstream": "http://127.0.0.1:18080", "domains": [] },
  "security": { "allowedCidrs": [], "blockPageHtml": "<html/>", "enableHsts": false }
}`)
	s := buildStoreForTest(t)

	svc := NewSiteOriginService(manager, s, nil)
	if svc.logger == nil {
		t.Fatalf("logger 不应为 nil")
	}
	if svc.maxConcurrency != 6 {
		t.Fatalf("maxConcurrency 应回退默认值: got=%d", svc.maxConcurrency)
	}
	if svc.tickInterval != 2*time.Second {
		t.Fatalf("tickInterval 不匹配: %v", svc.tickInterval)
	}
	if svc.backoffBase != 3*time.Second || svc.backoffMax != 4*time.Second {
		t.Fatalf("backoff 不匹配: base=%v max=%v", svc.backoffBase, svc.backoffMax)
	}
	if svc.jitterRatio != 0.9 {
		t.Fatalf("jitterRatio 应被上限截断: %v", svc.jitterRatio)
	}
}

// TestSiteOriginService_RunGuards 验证 nil 接收者与未初始化依赖时会安全返回。
func TestSiteOriginService_RunGuards(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var nilSvc *SiteOriginService
	nilSvc.run(ctx)

	(&SiteOriginService{}).run(ctx)
	(&SiteOriginService{store: nil, manager: nil, logger: zap.NewNop()}).run(ctx)
}

// TestSiteOriginService_RunJobAndWarmCache 验证单个任务执行、快照写入与缓存预热生效。
func TestSiteOriginService_RunJobAndWarmCache(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/client/v4/ips" {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprintln(w, `{
  "success": true,
  "errors": [],
  "messages": [],
  "result": {
    "ipv4_cidrs": ["1.1.1.1/32"],
    "ipv6_cidrs": []
  }
}`)
	}))
	defer ts.Close()

	manager := buildManagerForTest(t, `{
  "routing": { "defaultUpstream": "http://127.0.0.1:18080", "domains": [] },
  "security": { "allowedCidrs": [], "blockPageHtml": "<html/>", "enableHsts": false }
}`)
	s := buildStoreForTest(t)

	site, createErr := s.CreateSite(store.Site{Name: "S1", Hostname: "s1.example.com", IP: "10.0.0.10"})
	if createErr != nil {
		t.Fatalf("创建站点失败: %v", createErr)
	}
	siteCfg := config.Config{
		Security: config.SecurityConfig{
			AllowedCDNProviders: []string{"cloudflare"},
			CDNProviderSettings: map[string]config.CDNProviderSetting{
				"cloudflare": {Endpoint: ts.URL},
			},
		},
		Routing: config.RoutingConfig{DefaultUpstream: "http://127.0.0.1:18080"},
	}
	if _, updateErr := s.UpdateSiteConfig(site.ID, siteCfg, "tester", "manual"); updateErr != nil {
		t.Fatalf("写入站点配置失败: %v", updateErr)
	}

	svc := &SiteOriginService{
		manager:        manager,
		store:          s,
		logger:         zap.NewNop(),
		maxConcurrency: 1,
		tickInterval:   time.Second,
		jitterRatio:    0,
		backoffBase:    time.Second,
		backoffMax:     10 * time.Second,
	}

	ctx := context.Background()

	t.Run("provider not supported", func(t *testing.T) {
		now := time.Now().UTC()
		svc.runJob(ctx, siteOriginJob{
			siteID:     site.ID,
			provider:   "unsupported",
			settings:   config.CDNProviderSetting{},
			hasStatus:  false,
			attemptNow: now,
		})
		got, err := s.GetSiteCDNOriginStatus(site.ID, "unsupported")
		if err != nil {
			t.Fatalf("读取状态失败: %v", err)
		}
		if got.ConsecutiveFailures != 1 || got.LastError == "" {
			t.Fatalf("失败状态不匹配: %+v", got)
		}
	})

	t.Run("success", func(t *testing.T) {
		now := time.Now().UTC()
		svc.runJob(ctx, siteOriginJob{
			siteID:     site.ID,
			provider:   "cloudflare",
			settings:   config.CDNProviderSetting{Endpoint: ts.URL},
			hasStatus:  false,
			attemptNow: now,
		})

		snapshot, err := s.GetLatestSiteCDNOriginSnapshot(site.ID, "cloudflare")
		if err != nil {
			t.Fatalf("读取快照失败: %v", err)
		}
		if len(snapshot.CIDRs) != 1 || snapshot.CIDRs[0] != "1.1.1.1/32" {
			t.Fatalf("快照 CIDRs 不匹配: %+v", snapshot.CIDRs)
		}

		status, err := s.GetSiteCDNOriginStatus(site.ID, "cloudflare")
		if err != nil {
			t.Fatalf("读取状态失败: %v", err)
		}
		if status.ConsecutiveFailures != 0 || status.LastError != "" || status.LastSuccessAt == "" {
			t.Fatalf("成功状态不匹配: %+v", status)
		}
	})

	t.Run("warm cache updates manager snapshot", func(t *testing.T) {
		latest := time.Now().UTC().Add(-2 * time.Second)
		if _, err := s.SaveSiteCDNOriginSnapshot(site.ID, "cloudflare", []string{"1.1.1.1/32"}, latest, "auto"); err != nil {
			t.Fatalf("写入快照失败: %v", err)
		}
		svc.warmCache()

		ip := netip.MustParseAddr("1.1.1.1")
		if !manager.IsIPAllowedBySiteProviders(site.ID, ip, []string{"cloudflare"}, nil) {
			t.Fatalf("warmCache 后应命中 CIDR 放行")
		}
	})
}

// TestSiteOriginService_Tick 验证 tick 可遍历站点并按条件触发同步写入快照。
func TestSiteOriginService_Tick(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/client/v4/ips" {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprintln(w, `{
  "success": true,
  "errors": [],
  "messages": [],
  "result": {
    "ipv4_cidrs": ["2.2.2.2/32"],
    "ipv6_cidrs": []
  }
}`)
	}))
	defer ts.Close()

	manager := buildManagerForTest(t, `{
  "routing": { "defaultUpstream": "http://127.0.0.1:18080", "domains": [] },
  "security": { "allowedCidrs": [], "blockPageHtml": "<html/>", "enableHsts": false }
}`)
	s := buildStoreForTest(t)

	site, createErr := s.CreateSite(store.Site{Name: "S1", Hostname: "s1.example.com", IP: "10.0.0.10"})
	if createErr != nil {
		t.Fatalf("创建站点失败: %v", createErr)
	}
	siteCfg := config.Config{
		Security: config.SecurityConfig{
			AllowedCDNProviders: []string{"cloudflare", "cloudflare"},
			CDNProviderSettings: map[string]config.CDNProviderSetting{
				"cloudflare": {Endpoint: ts.URL},
			},
		},
		Routing: config.RoutingConfig{DefaultUpstream: "http://127.0.0.1:18080"},
	}
	if _, updateErr := s.UpdateSiteConfig(site.ID, siteCfg, "tester", "manual"); updateErr != nil {
		t.Fatalf("写入站点配置失败: %v", updateErr)
	}

	svc := &SiteOriginService{
		manager:        manager,
		store:          s,
		logger:         zap.NewNop(),
		maxConcurrency: 1,
		tickInterval:   time.Second,
		jitterRatio:    0,
		backoffBase:    time.Second,
		backoffMax:     10 * time.Second,
	}

	ctx := context.Background()
	svc.tick(ctx)

	snapshot, err := s.GetLatestSiteCDNOriginSnapshot(site.ID, "cloudflare")
	if err != nil {
		t.Fatalf("读取快照失败: %v", err)
	}
	if len(snapshot.CIDRs) != 1 || snapshot.CIDRs[0] != "2.2.2.2/32" {
		t.Fatalf("快照 CIDRs 不匹配: %+v", snapshot.CIDRs)
	}
}
