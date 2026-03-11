package config

import (
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkManagerResolveRouteLoadBalancerWeightedParallel(b *testing.B) {
	manager := benchmarkManagerFromJSON(b, `{
  "security": { "allowedCidrs": [], "blockPageHtml": "<html/>", "enableHsts": false },
  "routing": { "domains": [] },
  "loadBalancer": {
    "pools": [
      {
        "id":"pool_main",
        "name":"主池",
        "strategy":"weighted_rr",
        "nodes":[
          {"id":"n1","upstream":"http://127.0.0.1:38080","weight":3,"enabled":true},
          {"id":"n2","upstream":"http://127.0.0.1:38081","weight":2,"enabled":true},
          {"id":"n3","upstream":"http://127.0.0.1:38082","weight":1,"enabled":true}
        ]
      }
    ],
    "routes":[
      {"domain":"bench.example.com","poolId":"pool_main","forceHttps":false}
    ]
  }
}`)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			route, ok := manager.ResolveRoute("bench.example.com")
			if !ok || route.Target == nil {
				b.Fatalf("应命中负载均衡路由")
			}
		}
	})
}

func BenchmarkManagerResolveRouteLoadBalancerFallbackParallel(b *testing.B) {
	manager := benchmarkManagerFromJSON(b, `{
  "security": { "allowedCidrs": [], "blockPageHtml": "<html/>", "enableHsts": false },
  "routing": { "domains": [] },
  "loadBalancer": {
    "pools": [
      {
        "id":"pool_primary",
        "name":"主池",
        "strategy":"weighted_rr",
        "nodes":[
          {"id":"p1","upstream":"http://127.0.0.1:48080","weight":1,"enabled":false}
        ]
      },
      {
        "id":"pool_backup",
        "name":"回退池",
        "strategy":"weighted_rr",
        "nodes":[
          {"id":"b1","upstream":"http://127.0.0.1:48090","weight":1,"enabled":true}
        ]
      }
    ],
    "routes":[
      {"domain":"bench-fallback.example.com","poolId":"pool_primary","fallbackPoolId":"pool_backup","forceHttps":false}
    ]
  }
}`)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			route, ok := manager.ResolveRoute("bench-fallback.example.com")
			if !ok || route.Target == nil {
				b.Fatalf("应命中回退池路由")
			}
		}
	})
}

func benchmarkManagerFromJSON(b *testing.B, cfgJSON string) *Manager {
	b.Helper()
	baseDir := b.TempDir()
	configPath := filepath.Join(baseDir, "config.json")
	if err := os.WriteFile(configPath, []byte(cfgJSON), 0o600); err != nil {
		b.Fatalf("写入配置文件失败: %v", err)
	}
	manager, err := NewManager(configPath)
	if err != nil {
		b.Fatalf("初始化 Manager 失败: %v", err)
	}
	return manager
}
