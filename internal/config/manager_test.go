package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestManagerResolveRoute_ExactAndWildcard 验证精确域名与通配域名路由匹配逻辑。
func TestManagerResolveRoute_ExactAndWildcard(t *testing.T) {
	cfgJSON := `{
  "dataPlane": { "httpListenAddr": ":8080", "httpsListenAddr": ":8443", "enableHttps": true },
  "controlPlane": { "adminListenAddr": ":9090" },
  "security": { "allowedCidrs": [], "blockPageHtml": "<html/>", "enableHsts": false },
  "routing": {
    "defaultUpstream": "http://127.0.0.1:18080",
    "domains": [
      { "domain": "example.local", "upstream": "http://127.0.0.1:18080", "forceHttps": false },
      { "domain": "*.wild.local", "upstream": "http://127.0.0.1:18081", "forceHttps": true }
    ]
  },
  "certificates": []
}`

	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, "config.json")
	if err := os.WriteFile(configPath, []byte(cfgJSON), 0o600); err != nil {
		t.Fatalf("写入配置文件失败: %v", err)
	}

	manager, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("初始化 Manager 失败: %v", err)
	}

	route, ok := manager.ResolveRoute("HTTP://Example.Local:443")
	if !ok {
		t.Fatalf("未命中精确路由")
	}
	if route.Domain != "example.local" {
		t.Fatalf("域名不匹配: got=%q", route.Domain)
	}
	if route.ForceHTTPS {
		t.Fatalf("forceHttps 不匹配: got=true")
	}
	if route.Target == nil || route.Target.String() != "http://127.0.0.1:18080" {
		t.Fatalf("Target 不匹配: got=%v", route.Target)
	}

	route, ok = manager.ResolveRoute("a.wild.local:8080")
	if !ok {
		t.Fatalf("未命中通配路由")
	}
	if route.Domain != "*.wild.local" {
		t.Fatalf("域名不匹配: got=%q", route.Domain)
	}
	if !route.ForceHTTPS {
		t.Fatalf("forceHttps 不匹配: got=false")
	}
	if route.Target == nil || route.Target.String() != "http://127.0.0.1:18081" {
		t.Fatalf("Target 不匹配: got=%v", route.Target)
	}
}
