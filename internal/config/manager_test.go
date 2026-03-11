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

// TestManagerResolveRoute_LoadBalancerWeighted 验证流量池按加权轮询返回上游节点。
func TestManagerResolveRoute_LoadBalancerWeighted(t *testing.T) {
	cfgJSON := `{
  "security": { "allowedCidrs": [], "blockPageHtml": "<html/>", "enableHsts": false },
  "routing": { "domains": [] },
  "loadBalancer": {
    "pools": [
      {
        "id":"pool_main",
        "name":"主池",
        "strategy":"weighted_rr",
        "nodes":[
          {"id":"n1","upstream":"http://127.0.0.1:18080","weight":2,"enabled":true},
          {"id":"n2","upstream":"http://127.0.0.1:18081","weight":1,"enabled":true}
        ]
      }
    ],
    "routes":[
      {"domain":"api.example.com","poolId":"pool_main","forceHttps":false}
    ]
  }
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

	result := map[string]int{}
	for idx := 0; idx < 6; idx++ {
		route, ok := manager.ResolveRoute("api.example.com")
		if !ok || route.Target == nil {
			t.Fatalf("应命中负载均衡路由")
		}
		result[route.Target.String()]++
	}
	if result["http://127.0.0.1:18080"] != 4 || result["http://127.0.0.1:18081"] != 2 {
		t.Fatalf("加权轮询不匹配: %#v", result)
	}
}

// TestManagerResolveRoute_LoadBalancerFallback 验证主池不可用时会切换到回退池。
func TestManagerResolveRoute_LoadBalancerFallback(t *testing.T) {
	cfgJSON := `{
  "security": { "allowedCidrs": [], "blockPageHtml": "<html/>", "enableHsts": false },
  "routing": { "domains": [] },
  "loadBalancer": {
    "pools": [
      {
        "id":"pool_primary",
        "name":"主池",
        "strategy":"weighted_rr",
        "nodes":[
          {"id":"n1","upstream":"http://127.0.0.1:18080","weight":1,"enabled":false}
        ]
      },
      {
        "id":"pool_backup",
        "name":"回退池",
        "strategy":"weighted_rr",
        "nodes":[
          {"id":"b1","upstream":"http://127.0.0.1:18090","weight":1,"enabled":true}
        ]
      }
    ],
    "routes":[
      {"domain":"api.example.com","poolId":"pool_primary","fallbackPoolId":"pool_backup","forceHttps":true}
    ]
  }
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
	route, ok := manager.ResolveRoute("api.example.com")
	if !ok || route.Target == nil {
		t.Fatalf("应命中回退池路由")
	}
	if route.Target.String() != "http://127.0.0.1:18090" {
		t.Fatalf("回退池目标不匹配: got=%v", route.Target)
	}
	if !route.ForceHTTPS {
		t.Fatalf("forceHttps 应保持为 true")
	}
}
