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
