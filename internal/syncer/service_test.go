package syncer

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"YoBFF/internal/config"
	"YoBFF/internal/logging"
)

// TestService_trySync 验证同步任务可抓取并写入 Cloudflare 状态。
func TestService_trySync(t *testing.T) {
	// 1. Mock Cloudflare Server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "1.1.1.1/32")
	}))
	defer ts.Close()

	// 2. Setup Config
	f, err := os.CreateTemp("", "syncer_test_*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	content := fmt.Sprintf(`{
		"dataPlane": {"httpListenAddr": ":8080"},
		"controlPlane": {"adminListenAddr": ":9090"},
		"security": {"allowedCidrs": []},
		"routing": {"defaultUpstream": "http://localhost:8081"},
		"certificates": [],
		"cdnSync": {
			"enabled": true,
			"providers": ["cloudflare"],
			"schedule": "@every 1h",
			"cloudflare": {
				"ipv4_url": "%s/ipv4",
				"ipv6_url": "%s/ipv6"
			}
		}
	}`, ts.URL, ts.URL)
	f.WriteString(content)
	f.Close()

	manager, err := config.NewManager(f.Name())
	if err != nil {
		t.Fatal(err)
	}

	// 3. Setup Service
	runtime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	logger := runtime.Logger()
	service := NewService(manager, logger)

	// 4. Run trySync
	service.trySync(context.Background())

	// 5. Verify
	status := manager.GetCDNStatus()
	if s, ok := status["cloudflare"]; !ok {
		t.Error("Cloudflare status not found")
	} else {
		if len(s.CIDRs) == 0 {
			t.Error("Expected CIDRs, got empty")
		} else if s.CIDRs[0] != "1.1.1.1/32" {
			t.Errorf("Unexpected CIDR: %v", s.CIDRs)
		}
		if s.Error != "" {
			t.Errorf("Unexpected error: %s", s.Error)
		}
	}
}

// TestService_shouldSync 验证调度周期判断在不同时间点的行为。
func TestService_shouldSync(t *testing.T) {
	// 1. Setup minimal Manager
	f, err := os.CreateTemp("", "syncer_shouldSync_*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	content := `{
		"dataPlane": {"httpListenAddr": ":8080"},
		"controlPlane": {"adminListenAddr": ":9090"},
		"security": {"allowedCidrs": []},
		"routing": {"defaultUpstream": "http://localhost:8081"},
		"certificates": [],
		"cdnSync": {
			"enabled": true,
			"providers": ["cloudflare"],
			"schedule": "@every 1m"
		}
	}`
	f.WriteString(content)
	f.Close()

	manager, err := config.NewManager(f.Name())
	if err != nil {
		t.Fatal(err)
	}

	service := &Service{manager: manager}

	// Case 1: Just synced -> False
	if service.shouldSync(time.Now()) {
		t.Error("Should not sync immediately after sync")
	}

	// Case 2: Synced long ago -> True
	if !service.shouldSync(time.Now().Add(-2 * time.Minute)) {
		t.Error("Should sync after duration")
	}
}

func TestService_RunAndStart(t *testing.T) {
	f, err := os.CreateTemp("", "syncer_run_*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	content := `{
		"dataPlane": {"httpListenAddr": ":8080"},
		"controlPlane": {"adminListenAddr": ":9090"},
		"security": {"allowedCidrs": []},
		"routing": {"defaultUpstream": "http://localhost:8081"},
		"certificates": [],
		"cdnSync": {
			"enabled": false,
			"providers": ["cloudflare"],
			"schedule": "1h"
		}
	}`
	_, _ = f.WriteString(content)
	_ = f.Close()

	manager, err := config.NewManager(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(manager, runtime.Logger())

	ctx1, cancel1 := context.WithCancel(context.Background())
	cancel1()
	service.run(ctx1)

	ctx2, cancel2 := context.WithCancel(context.Background())
	service.Start(ctx2)
	time.Sleep(20 * time.Millisecond)
	cancel2()
	time.Sleep(20 * time.Millisecond)
}

func TestService_shouldSyncTable(t *testing.T) {
	f, err := os.CreateTemp("", "syncer_schedule_*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	content := `{
		"dataPlane": {"httpListenAddr": ":8080"},
		"controlPlane": {"adminListenAddr": ":9090"},
		"security": {"allowedCidrs": []},
		"routing": {"defaultUpstream": "http://localhost:8081"},
		"certificates": [],
		"cdnSync": {
			"enabled": true,
			"providers": ["cloudflare"],
			"schedule": "1h"
		}
	}`
	_, _ = f.WriteString(content)
	_ = f.Close()
	manager, err := config.NewManager(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	service := &Service{manager: manager}

	cases := []struct {
		name     string
		enabled  bool
		schedule string
		lastSync time.Time
		want     bool
	}{
		{
			name:     "disabled",
			enabled:  false,
			schedule: "1m",
			lastSync: time.Now().Add(-10 * time.Hour),
			want:     false,
		},
		{
			name:     "empty schedule default 1h not reached",
			enabled:  true,
			schedule: "",
			lastSync: time.Now().Add(-10 * time.Minute),
			want:     false,
		},
		{
			name:     "invalid schedule fallback 1h reached",
			enabled:  true,
			schedule: "bad schedule",
			lastSync: time.Now().Add(-2 * time.Hour),
			want:     true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			cfg := manager.CurrentConfig()
			cfg.CDNSync.Enabled = tt.enabled
			cfg.CDNSync.Schedule = tt.schedule
			if err = manager.Apply(cfg); err != nil {
				t.Fatalf("应用配置失败: %v", err)
			}
			got := service.shouldSync(tt.lastSync)
			if got != tt.want {
				t.Fatalf("shouldSync 结果错误: got=%v want=%v", got, tt.want)
			}
		})
	}
}

func TestService_SyncProviderErrorAndUnknown(t *testing.T) {
	f, err := os.CreateTemp("", "syncer_provider_*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	content := `{
		"dataPlane": {"httpListenAddr": ":8080"},
		"routing": {"defaultUpstream": "http://localhost:8081"},
		"cdnSync": {
			"enabled": true,
			"providers": ["cloudflare", "unknown-provider"],
			"schedule": "1h",
			"cloudflare": {
				"ipv4_url": "http://127.0.0.1:1/ipv4",
				"ipv6_url": "http://127.0.0.1:1/ipv6"
			}
		}
	}`
	_, _ = f.WriteString(content)
	_ = f.Close()

	manager, err := config.NewManager(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(manager, runtime.Logger())
	cfg := manager.CurrentConfig()

	service.syncProvider(context.Background(), "unknown-provider", cfg)
	service.syncProvider(context.Background(), "cloudflare", cfg)

	status := manager.GetCDNStatus()
	if _, ok := status["unknown-provider"]; ok {
		t.Fatalf("未知提供商不应写入状态")
	}
	if s, ok := status["cloudflare"]; !ok {
		t.Fatalf("cloudflare 状态缺失")
	} else if s.Error == "" {
		t.Fatalf("cloudflare 失败时应写入错误状态")
	}
}
