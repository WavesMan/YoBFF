package config

import (
	"net/netip"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateConfig_Issues(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		wantKey string
	}{
		{
			name: "invalid cidr",
			cfg: Config{
				Security: SecurityConfig{
					AllowedCIDRs: []string{"bad-cidr"},
				},
			},
			wantKey: "security.allowedCidrs[0]",
		},
		{
			name: "invalid trusted proxy cidr",
			cfg: Config{
				Security: SecurityConfig{
					TrustedProxyCIDRs: []string{"bad-proxy-cidr"},
				},
			},
			wantKey: "security.trustedProxyCidrs[0]",
		},
		{
			name: "unsupported cdn provider",
			cfg: Config{
				Security: SecurityConfig{
					AllowedCDNProviders: []string{"bad-provider"},
				},
			},
			wantKey: "security.allowedCdnProviders[0]",
		},
		{
			name: "duplicated cdn provider",
			cfg: Config{
				Security: SecurityConfig{
					AllowedCDNProviders: []string{"cloudflare", "cloudflare"},
				},
			},
			wantKey: "security.allowedCdnProviders[1]",
		},
		{
			name: "aliyun cdn settings api key required",
			cfg: Config{
				Security: SecurityConfig{
					AllowedCDNProviders: []string{"aliyun"},
					CDNProviderSettings: map[string]CDNProviderSetting{
						"aliyun": {Option: "123"},
					},
				},
			},
			wantKey: "security.cdnProviderSettings.aliyun.apiKey",
		},
		{
			name: "aliyun cdn settings option required",
			cfg: Config{
				Security: SecurityConfig{
					AllowedCDNProviders: []string{"aliyun"},
					CDNProviderSettings: map[string]CDNProviderSetting{
						"aliyun": {APIKey: "a"},
					},
				},
			},
			wantKey: "security.cdnProviderSettings.aliyun.option",
		},
		{
			name: "tencent cdn settings zone id required",
			cfg: Config{
				Security: SecurityConfig{
					AllowedCDNProviders: []string{"tencent"},
					CDNProviderSettings: map[string]CDNProviderSetting{
						"tencent": {APIKey: "a"},
					},
				},
			},
			wantKey: "security.cdnProviderSettings.tencent.zoneId",
		},
		{
			name: "domain required",
			cfg: Config{
				Routing: RoutingConfig{
					Domains: []DomainRule{
						{Domain: "", Upstream: "http://127.0.0.1:8080"},
					},
				},
			},
			wantKey: "routing.domains[0].domain",
		},
		{
			name: "upstream required",
			cfg: Config{
				Routing: RoutingConfig{
					Domains: []DomainRule{
						{Domain: "a.example.com", Upstream: ""},
					},
				},
			},
			wantKey: "routing.domains[0].upstream",
		},
		{
			name: "upstream invalid",
			cfg: Config{
				Routing: RoutingConfig{
					Domains: []DomainRule{
						{Domain: "a.example.com", Upstream: "://bad"},
					},
				},
			},
			wantKey: "routing.domains[0].upstream",
		},
		{
			name: "default upstream invalid",
			cfg: Config{
				Routing: RoutingConfig{
					DefaultUpstream: "://bad",
				},
			},
			wantKey: "routing.defaultUpstream",
		},
		{
			name: "certificate domain required",
			cfg: Config{
				Certificates: []Certificate{
					{Domain: "", CertPEM: "a", KeyPEM: "b"},
				},
			},
			wantKey: "certificates[0].domain",
		},
		{
			name: "certificate cert required",
			cfg: Config{
				Certificates: []Certificate{
					{Domain: "a.example.com", CertPEM: "", KeyPEM: "b"},
				},
			},
			wantKey: "certificates[0].certPem",
		},
		{
			name: "certificate key required",
			cfg: Config{
				Certificates: []Certificate{
					{Domain: "a.example.com", CertPEM: "a", KeyPEM: ""},
				},
			},
			wantKey: "certificates[0].keyPem",
		},
		{
			name: "lb pool node upstream invalid",
			cfg: Config{
				LoadBalancer: LoadBalancerConfig{
					Pools: []LBPool{
						{
							ID:       "pool_main",
							Name:     "主池",
							Strategy: "weighted_rr",
							Nodes: []LBNode{
								{ID: "n1", Upstream: "://bad", Weight: 1, Enabled: true},
							},
						},
					},
				},
			},
			wantKey: "loadBalancer.pools[0].nodes[0].upstream",
		},
		{
			name: "lb route pool not found",
			cfg: Config{
				LoadBalancer: LoadBalancerConfig{
					Routes: []LBRouteRule{
						{Domain: "a.example.com", PoolID: "pool_missing"},
					},
				},
			},
			wantKey: "loadBalancer.routes[0].poolId",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			issues := ValidateConfig(tt.cfg, "config.json")
			if len(issues) == 0 {
				t.Fatalf("应返回校验问题")
			}
			found := false
			for _, issue := range issues {
				if issue.Path == tt.wantKey {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("未命中预期字段: want=%s issues=%#v", tt.wantKey, issues)
			}
		})
	}
}

func TestValidateConfig_CertificateFileErrors(t *testing.T) {
	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, "config.json")
	if err := os.WriteFile(configPath, []byte("{}"), 0o600); err != nil {
		t.Fatalf("写入配置失败: %v", err)
	}

	certFileMissing := Config{
		Certificates: []Certificate{
			{
				Domain:   "a.example.com",
				CertFile: "missing-cert.pem",
				KeyPEM:   "x",
			},
		},
	}
	certIssues := ValidateConfig(certFileMissing, configPath)
	if len(certIssues) == 0 || certIssues[0].Path != "certificates[0].certFile" {
		t.Fatalf("证书文件缺失校验失败: %#v", certIssues)
	}

	keyFileMissing := Config{
		Certificates: []Certificate{
			{
				Domain:   "a.example.com",
				CertPEM:  "x",
				KeyFile:  "missing-key.pem",
				CertFile: "",
			},
		},
	}
	keyIssues := ValidateConfig(keyFileMissing, configPath)
	if len(keyIssues) == 0 || keyIssues[0].Path != "certificates[0].keyFile" {
		t.Fatalf("私钥文件缺失校验失败: %#v", keyIssues)
	}
}

func TestManagerValidateAndProviderIP(t *testing.T) {
	manager := buildManagerFromJSON(t, `{
  "routing": { "defaultUpstream": "http://127.0.0.1:18080", "domains": [] },
  "security": {
    "allowedCidrs": ["127.0.0.1/32"],
    "allowedCdnProviders": ["cloudflare"],
    "blockPageHtml": "<html/>",
    "enableHsts": false
  },
  "cdnSync": {
    "enabled": true,
    "providers": ["cloudflare"],
    "cloudflare": {
      "endpoint": "https://api.cloudflare.com"
    }
  }
}`)

	invalidIssues := manager.Validate(Config{
		Routing: RoutingConfig{
			DefaultUpstream: "://bad",
		},
	})
	if len(invalidIssues) == 0 {
		t.Fatalf("Validate 应返回错误")
	}

	if manager.IsIPAllowedByProviders(netip.MustParseAddr("127.0.0.1"), nil) {
		t.Fatalf("空提供商列表不应放行")
	}
	if manager.IsIPAllowedByProviders(netip.MustParseAddr("127.0.0.1"), []string{"unknown"}) {
		t.Fatalf("未知提供商不应放行")
	}
}
