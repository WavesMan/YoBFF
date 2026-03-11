package config

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"strings"
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

func TestManagerRoutingLoadBalancerStrategy(t *testing.T) {
	cases := []struct {
		name       string
		cfgJSON    string
		expectErr  bool
		errMessage string
		verify     func(t *testing.T, manager *Manager)
	}{
		{
			name: "migrate legacy routing to load balancer",
			cfgJSON: `{
  "security": { "allowedCidrs": [], "blockPageHtml": "<html/>", "enableHsts": false },
  "routing": {
    "defaultUpstream": "http://127.0.0.1:19090",
    "domains": [
      { "domain": "legacy.example.com", "upstream": "http://127.0.0.1:19080", "forceHttps": true }
    ]
  }
}`,
			verify: func(t *testing.T, manager *Manager) {
				cfg := manager.CurrentConfig()
				if len(cfg.Routing.Domains) != 0 || cfg.Routing.DefaultUpstream != "" {
					t.Fatalf("旧路由应被迁移并清空: %#v", cfg.Routing)
				}
				if len(cfg.LoadBalancer.Pools) != 2 {
					t.Fatalf("迁移后流量池数量不匹配: got=%d", len(cfg.LoadBalancer.Pools))
				}
				if len(cfg.LoadBalancer.Routes) != 1 {
					t.Fatalf("迁移后绑定规则数量不匹配: got=%d", len(cfg.LoadBalancer.Routes))
				}
				route, ok := manager.ResolveRoute("legacy.example.com")
				if !ok || route.Target == nil {
					t.Fatalf("应命中迁移后的域名流量池")
				}
				if route.Target.String() != "http://127.0.0.1:19080" {
					t.Fatalf("迁移后域名流量池目标不匹配: got=%v", route.Target)
				}
				defaultRoute, ok := manager.ResolveRoute("unknown.example.com")
				if !ok || defaultRoute.Target == nil {
					t.Fatalf("应命中迁移后的默认流量池")
				}
				if defaultRoute.Target.String() != "http://127.0.0.1:19090" {
					t.Fatalf("迁移后默认流量池目标不匹配: got=%v", defaultRoute.Target)
				}
			},
		},
		{
			name: "reject conflict when routing domain overlaps lb route",
			cfgJSON: `{
  "security": { "allowedCidrs": [], "blockPageHtml": "<html/>", "enableHsts": false },
  "routing": {
    "domains": [
      { "domain": "api.example.com", "upstream": "http://127.0.0.1:18080", "forceHttps": false }
    ]
  },
  "loadBalancer": {
    "pools": [
      {
        "id":"pool_api",
        "name":"API池",
        "strategy":"weighted_rr",
        "nodes":[
          {"id":"n1","upstream":"http://127.0.0.1:28080","weight":1,"enabled":true}
        ]
      }
    ],
    "routes":[
      {"domain":"api.example.com","poolId":"pool_api","forceHttps":true}
    ]
  }
}`,
			expectErr:  true,
			errMessage: "配置冲突",
		},
		{
			name: "allow mixed routing when domains do not overlap",
			cfgJSON: `{
  "security": { "allowedCidrs": [], "blockPageHtml": "<html/>", "enableHsts": false },
  "routing": {
    "domains": [
      { "domain": "legacy.example.com", "upstream": "http://127.0.0.1:18080", "forceHttps": false }
    ]
  },
  "loadBalancer": {
    "pools": [
      {
        "id":"pool_api",
        "name":"API池",
        "strategy":"weighted_rr",
        "nodes":[
          {"id":"n1","upstream":"http://127.0.0.1:28080","weight":1,"enabled":true}
        ]
      }
    ],
    "routes":[
      {"domain":"api.example.com","poolId":"pool_api","forceHttps":true}
    ]
  }
}`,
			verify: func(t *testing.T, manager *Manager) {
				lbRoute, ok := manager.ResolveRoute("api.example.com")
				if !ok || lbRoute.Target == nil || lbRoute.Target.String() != "http://127.0.0.1:28080" {
					t.Fatalf("应命中独立 LB 路由")
				}
				legacyRoute, ok := manager.ResolveRoute("legacy.example.com")
				if !ok || legacyRoute.Target == nil || legacyRoute.Target.String() != "http://127.0.0.1:18080" {
					t.Fatalf("应命中旧路由配置")
				}
			},
		},
	}

	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			baseDir := t.TempDir()
			configPath := filepath.Join(baseDir, "config.json")
			if err := os.WriteFile(configPath, []byte(item.cfgJSON), 0o600); err != nil {
				t.Fatalf("写入配置文件失败: %v", err)
			}
			manager, err := NewManager(configPath)
			if item.expectErr {
				if err == nil {
					t.Fatalf("应返回初始化错误")
				}
				if item.errMessage != "" && !strings.Contains(err.Error(), item.errMessage) {
					t.Fatalf("错误信息不匹配: got=%q", err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("初始化 Manager 失败: %v", err)
			}
			if item.verify != nil {
				item.verify(t, manager)
			}
		})
	}
}

func TestManagerResolveRoute_LoadBalancerFallbackStable(t *testing.T) {
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
      {"domain":"stable.example.com","poolId":"pool_primary","fallbackPoolId":"pool_backup","forceHttps":true}
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
	for index := 0; index < 200; index++ {
		route, ok := manager.ResolveRoute("stable.example.com")
		if !ok || route.Target == nil {
			t.Fatalf("第 %d 次解析未命中回退池", index)
		}
		if route.Target.String() != "http://127.0.0.1:18090" {
			t.Fatalf("第 %d 次回退池目标不匹配: got=%v", index, route.Target)
		}
	}
}

func TestManagerResolveRoute_LoadBalancerConcurrentDistribution(t *testing.T) {
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
          {"id":"n1","upstream":"http://127.0.0.1:28080","weight":2,"enabled":true},
          {"id":"n2","upstream":"http://127.0.0.1:28081","weight":1,"enabled":true}
        ]
      }
    ],
    "routes":[
      {"domain":"concurrent.example.com","poolId":"pool_main","forceHttps":false}
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

	var countA atomic.Int64
	var countB atomic.Int64
	const (
		goroutineCount = 24
		requestPerGo   = 300
	)
	var waitGroup sync.WaitGroup
	waitGroup.Add(goroutineCount)
	for i := 0; i < goroutineCount; i++ {
		go func() {
			defer waitGroup.Done()
			for j := 0; j < requestPerGo; j++ {
				route, ok := manager.ResolveRoute("concurrent.example.com")
				if !ok || route.Target == nil {
					t.Errorf("并发解析未命中")
					return
				}
				switch route.Target.String() {
				case "http://127.0.0.1:28080":
					countA.Add(1)
				case "http://127.0.0.1:28081":
					countB.Add(1)
				default:
					t.Errorf("命中未知目标: %v", route.Target)
					return
				}
			}
		}()
	}
	waitGroup.Wait()

	gotA := countA.Load()
	gotB := countB.Load()
	if gotA != 4800 || gotB != 2400 {
		t.Fatalf("并发分布不匹配: gotA=%d gotB=%d", gotA, gotB)
	}
}
