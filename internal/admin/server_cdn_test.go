package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"YoBFF/internal/config"
	"YoBFF/internal/logging"
)

func TestServer_CDNConfig(t *testing.T) {
	// 1. Setup Env
	os.Setenv("ADMIN_API_TOKEN", "test-token")
	defer os.Unsetenv("ADMIN_API_TOKEN")

	// 2. Setup Manager with temp config
	f, err := os.CreateTemp("", "admin_test_*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	initialConfig := `{
		"dataPlane": {"httpListenAddr": ":8080"},
		"controlPlane": {"adminListenAddr": ":9090"},
		"security": {"allowedCidrs": ["127.0.0.1/32"]},
		"routing": {"defaultUpstream": "http://localhost:8081"},
		"certificates": [],
		"cdnSync": {
			"enabled": false,
			"providers": ["cloudflare"],
			"schedule": "@every 1h"
		}
	}`
	if _, err = f.WriteString(initialConfig); err != nil {
		t.Fatal(err)
	}
	f.Close()

	manager, err := config.NewManager(f.Name())
	if err != nil {
		t.Fatal(err)
	}

	// 3. Setup Server
	logger, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	pipeline := logging.NewPipeline(100, logger.Logger())
	server := NewServer(manager, logger, pipeline, nil)
	handler := server.Handler()

	// 4. Test GET /api/v1/config/cdn
	t.Run("GET CDN Config", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/config/cdn", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		// Verify structure
		if _, ok := resp["config"]; !ok {
			t.Error("Response missing config")
		}
		if _, ok := resp["status"]; !ok {
			t.Error("Response missing status")
		}
	})

	// 5. Test PUT /api/v1/config/cdn
	t.Run("PUT CDN Config", func(t *testing.T) {
		newConfig := config.CDNSyncConfig{
			Enabled:    true,
			Providers:  []string{"cloudflare", "aliyun"},
			Schedule:   "@every 30m",
			Cloudflare: config.CloudflareConfig{IPv4URL: "http://example.com"},
		}
		body, _ := json.Marshal(newConfig)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/config/cdn", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		// Verify update in manager
		current := manager.CurrentConfig().CDNSync
		if !current.Enabled {
			t.Error("Config not updated: Enabled should be true")
		}
		if len(current.Providers) != 2 {
			t.Errorf("Config not updated: Providers length %d", len(current.Providers))
		}
		if current.Cloudflare.IPv4URL != "http://example.com" {
			t.Error("Config not updated: Cloudflare IPv4URL mismatch")
		}
	})
}
