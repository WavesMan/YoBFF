package config

import (
	"crypto/tls"
	"fmt"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// ValidateConfig 校验配置结构、路由、证书与 CDN 提供商合法性。
// 参数：cfg 为待校验配置，configPath 为配置文件路径用于解析相对证书路径。
// 返回：配置问题列表，空切片表示无错误。
// 异常：无。
func ValidateConfig(cfg Config, configPath string) []ValidationIssue {
	cfg = fillDefaults(cfg)
	var issues []ValidationIssue

	for idx, cidrText := range cfg.Security.AllowedCIDRs {
		value := strings.TrimSpace(cidrText)
		if value == "" {
			continue
		}
		if _, err := netip.ParsePrefix(value); err != nil {
			issues = append(issues, ValidationIssue{
				Path:    fmt.Sprintf("security.allowedCidrs[%d]", idx),
				Message: err.Error(),
			})
			continue
		}
	}
	for idx, cidrText := range cfg.Security.TrustedProxyCIDRs {
		value := strings.TrimSpace(cidrText)
		if value == "" {
			continue
		}
		if _, err := netip.ParsePrefix(value); err != nil {
			issues = append(issues, ValidationIssue{
				Path:    fmt.Sprintf("security.trustedProxyCidrs[%d]", idx),
				Message: err.Error(),
			})
			continue
		}
	}

	supportedCDNProviders := map[string]struct{}{
		"aliyun":     {},
		"cloudflare": {},
		"tencent":    {},
	}
	seenProviders := make(map[string]struct{}, len(cfg.Security.AllowedCDNProviders))
	for idx, provider := range cfg.Security.AllowedCDNProviders {
		value := strings.ToLower(strings.TrimSpace(provider))
		if value == "" {
			continue
		}
		if _, ok := supportedCDNProviders[value]; !ok {
			issues = append(issues, ValidationIssue{
				Path:    fmt.Sprintf("security.allowedCdnProviders[%d]", idx),
				Message: "cdn provider is not supported",
			})
			continue
		}
		if _, exists := seenProviders[value]; exists {
			issues = append(issues, ValidationIssue{
				Path:    fmt.Sprintf("security.allowedCdnProviders[%d]", idx),
				Message: "cdn provider is duplicated",
			})
			continue
		}
		seenProviders[value] = struct{}{}
	}

	if cfg.Security.OriginProtectionMode != "" {
		mode := strings.ToLower(strings.TrimSpace(cfg.Security.OriginProtectionMode))
		if mode != "disabled" && mode != "enforced" {
			issues = append(issues, ValidationIssue{
				Path:    "security.originProtectionMode",
				Message: "origin protection mode is invalid",
			})
		}
	}

	for provider := range seenProviders {
		settings := CDNProviderSetting{}
		if cfg.Security.CDNProviderSettings != nil {
			if value, ok := cfg.Security.CDNProviderSettings[provider]; ok {
				settings = value
			}
		}

		if settings.RefreshIntervalSeconds != 0 {
			if settings.RefreshIntervalSeconds < 60 || settings.RefreshIntervalSeconds > 86400 {
				issues = append(issues, ValidationIssue{
					Path:    fmt.Sprintf("security.cdnProviderSettings.%s.refreshIntervalSeconds", provider),
					Message: "refresh interval seconds is out of range",
				})
			}
		}

		if settings.MaxStalenessSeconds != 0 {
			if settings.MaxStalenessSeconds < 60 || settings.MaxStalenessSeconds > 604800 {
				issues = append(issues, ValidationIssue{
					Path:    fmt.Sprintf("security.cdnProviderSettings.%s.maxStalenessSeconds", provider),
					Message: "max staleness seconds is out of range",
				})
			}
			if settings.RefreshIntervalSeconds > 0 && settings.MaxStalenessSeconds > 0 && settings.MaxStalenessSeconds < settings.RefreshIntervalSeconds {
				issues = append(issues, ValidationIssue{
					Path:    fmt.Sprintf("security.cdnProviderSettings.%s.maxStalenessSeconds", provider),
					Message: "max staleness seconds must be >= refresh interval seconds",
				})
			}
		}

		switch provider {
		case "aliyun":
			if strings.TrimSpace(settings.APIKey) == "" {
				issues = append(issues, ValidationIssue{
					Path:    fmt.Sprintf("security.cdnProviderSettings.%s.apiKey", provider),
					Message: "apiKey is required",
				})
			}
			if strings.TrimSpace(settings.Option) == "" {
				issues = append(issues, ValidationIssue{
					Path:    fmt.Sprintf("security.cdnProviderSettings.%s.option", provider),
					Message: "option(siteId) is required",
				})
			}
			if strings.TrimSpace(settings.Endpoint) != "" {
				if parsed, err := url.Parse(strings.TrimSpace(settings.Endpoint)); err != nil || parsed.Scheme == "" || parsed.Host == "" {
					issues = append(issues, ValidationIssue{
						Path:    fmt.Sprintf("security.cdnProviderSettings.%s.endpoint", provider),
						Message: "endpoint is invalid",
					})
				}
			}
		case "tencent":
			if strings.TrimSpace(settings.APIKey) == "" {
				issues = append(issues, ValidationIssue{
					Path:    fmt.Sprintf("security.cdnProviderSettings.%s.apiKey", provider),
					Message: "apiKey is required",
				})
			}
			if strings.TrimSpace(settings.ZoneID) == "" {
				issues = append(issues, ValidationIssue{
					Path:    fmt.Sprintf("security.cdnProviderSettings.%s.zoneId", provider),
					Message: "zoneId is required",
				})
			}
			if strings.TrimSpace(settings.Endpoint) != "" {
				if parsed, err := url.Parse(strings.TrimSpace(settings.Endpoint)); err != nil || parsed.Scheme == "" || parsed.Host == "" {
					issues = append(issues, ValidationIssue{
						Path:    fmt.Sprintf("security.cdnProviderSettings.%s.endpoint", provider),
						Message: "endpoint is invalid",
					})
				}
			}
		}
	}

	for idx, rule := range cfg.Routing.Domains {
		domain := strings.TrimSpace(rule.Domain)
		if domain == "" {
			issues = append(issues, ValidationIssue{
				Path:    fmt.Sprintf("routing.domains[%d].domain", idx),
				Message: "domain is required",
			})
			continue
		}
		upstream := strings.TrimSpace(rule.Upstream)
		if upstream == "" {
			issues = append(issues, ValidationIssue{
				Path:    fmt.Sprintf("routing.domains[%d].upstream", idx),
				Message: "upstream is required",
			})
			continue
		}
		parsed, err := url.Parse(upstream)
		if err != nil || parsed.Host == "" || parsed.Scheme == "" {
			issues = append(issues, ValidationIssue{
				Path:    fmt.Sprintf("routing.domains[%d].upstream", idx),
				Message: "upstream url is invalid",
			})
			continue
		}
	}

	if cfg.Routing.DefaultUpstream != "" {
		parsed, err := url.Parse(cfg.Routing.DefaultUpstream)
		if err != nil || parsed.Host == "" || parsed.Scheme == "" {
			issues = append(issues, ValidationIssue{
				Path:    "routing.defaultUpstream",
				Message: "default upstream url is invalid",
			})
		}
	}

	validateLoadBalancerConfig(cfg.LoadBalancer, &issues)

	if len(cfg.Certificates) > 0 {
		baseDir := filepath.Dir(configPath)
		for idx, item := range cfg.Certificates {
			domain := strings.TrimSpace(item.Domain)
			if domain == "" {
				issues = append(issues, ValidationIssue{
					Path:    fmt.Sprintf("certificates[%d].domain", idx),
					Message: "domain is required",
				})
				continue
			}
			certPEM := []byte(item.CertPEM)
			keyPEM := []byte(item.KeyPEM)
			if item.CertFile != "" {
				payload, err := os.ReadFile(filepath.Join(baseDir, item.CertFile))
				if err != nil {
					issues = append(issues, ValidationIssue{
						Path:    fmt.Sprintf("certificates[%d].certFile", idx),
						Message: err.Error(),
					})
					continue
				}
				certPEM = payload
			}
			if item.KeyFile != "" {
				payload, err := os.ReadFile(filepath.Join(baseDir, item.KeyFile))
				if err != nil {
					issues = append(issues, ValidationIssue{
						Path:    fmt.Sprintf("certificates[%d].keyFile", idx),
						Message: err.Error(),
					})
					continue
				}
				keyPEM = payload
			}
			if len(certPEM) == 0 {
				issues = append(issues, ValidationIssue{
					Path:    fmt.Sprintf("certificates[%d].certPem", idx),
					Message: "certPem is required",
				})
				continue
			}
			if len(keyPEM) == 0 {
				issues = append(issues, ValidationIssue{
					Path:    fmt.Sprintf("certificates[%d].keyPem", idx),
					Message: "keyPem is required",
				})
				continue
			}
			if _, err := tls.X509KeyPair(certPEM, keyPEM); err != nil {
				issues = append(issues, ValidationIssue{
					Path:    fmt.Sprintf("certificates[%d]", idx),
					Message: err.Error(),
				})
				continue
			}
		}
	}

	return issues
}

// validateLoadBalancerConfig 校验流量池与域名绑定配置。
// 参数：lb 为负载均衡配置，issues 为问题列表写入目标。
// 返回：无。
// 异常：无。
func validateLoadBalancerConfig(lb LoadBalancerConfig, issues *[]ValidationIssue) {
	poolIndex := make(map[string]int, len(lb.Pools))
	for idx, pool := range lb.Pools {
		poolID := strings.TrimSpace(pool.ID)
		if poolID == "" {
			*issues = append(*issues, ValidationIssue{
				Path:    fmt.Sprintf("loadBalancer.pools[%d].id", idx),
				Message: "pool id is required",
			})
			continue
		}
		if prev, exists := poolIndex[poolID]; exists {
			*issues = append(*issues, ValidationIssue{
				Path:    fmt.Sprintf("loadBalancer.pools[%d].id", idx),
				Message: fmt.Sprintf("pool id is duplicated with index %d", prev),
			})
		}
		poolIndex[poolID] = idx
		validateLoadBalancerPoolNodes(idx, pool, issues)
	}

	if lb.DefaultPoolID != "" {
		if _, ok := poolIndex[strings.TrimSpace(lb.DefaultPoolID)]; !ok {
			*issues = append(*issues, ValidationIssue{
				Path:    "loadBalancer.defaultPoolId",
				Message: "default pool id not found in pools",
			})
		}
	}

	for idx, route := range lb.Routes {
		domain := strings.TrimSpace(route.Domain)
		if domain == "" {
			*issues = append(*issues, ValidationIssue{
				Path:    fmt.Sprintf("loadBalancer.routes[%d].domain", idx),
				Message: "domain is required",
			})
		}
		poolID := strings.TrimSpace(route.PoolID)
		if poolID == "" {
			*issues = append(*issues, ValidationIssue{
				Path:    fmt.Sprintf("loadBalancer.routes[%d].poolId", idx),
				Message: "pool id is required",
			})
		} else if _, ok := poolIndex[poolID]; !ok {
			*issues = append(*issues, ValidationIssue{
				Path:    fmt.Sprintf("loadBalancer.routes[%d].poolId", idx),
				Message: "pool id not found in pools",
			})
		}
		fallbackID := strings.TrimSpace(route.FallbackPoolID)
		if fallbackID != "" {
			if _, ok := poolIndex[fallbackID]; !ok {
				*issues = append(*issues, ValidationIssue{
					Path:    fmt.Sprintf("loadBalancer.routes[%d].fallbackPoolId", idx),
					Message: "fallback pool id not found in pools",
				})
			}
		}
	}
}

// validateLoadBalancerPoolNodes 校验流量池节点配置。
// 参数：poolIndex 为池索引，pool 为待校验流量池，issues 为问题列表写入目标。
// 返回：无。
// 异常：无。
func validateLoadBalancerPoolNodes(poolIndexValue int, pool LBPool, issues *[]ValidationIssue) {
	nodeIndex := make(map[string]int, len(pool.Nodes))
	for nodeIdx, node := range pool.Nodes {
		nodeID := strings.TrimSpace(node.ID)
		if nodeID == "" {
			*issues = append(*issues, ValidationIssue{
				Path:    fmt.Sprintf("loadBalancer.pools[%d].nodes[%d].id", poolIndexValue, nodeIdx),
				Message: "node id is required",
			})
		} else if prev, exists := nodeIndex[nodeID]; exists {
			*issues = append(*issues, ValidationIssue{
				Path:    fmt.Sprintf("loadBalancer.pools[%d].nodes[%d].id", poolIndexValue, nodeIdx),
				Message: fmt.Sprintf("node id is duplicated with index %d", prev),
			})
		}
		nodeIndex[nodeID] = nodeIdx

		upstream := strings.TrimSpace(node.Upstream)
		if upstream == "" {
			*issues = append(*issues, ValidationIssue{
				Path:    fmt.Sprintf("loadBalancer.pools[%d].nodes[%d].upstream", poolIndexValue, nodeIdx),
				Message: "upstream is required",
			})
		} else {
			parsed, err := url.Parse(upstream)
			if err != nil || parsed.Host == "" || parsed.Scheme == "" {
				*issues = append(*issues, ValidationIssue{
					Path:    fmt.Sprintf("loadBalancer.pools[%d].nodes[%d].upstream", poolIndexValue, nodeIdx),
					Message: "upstream url is invalid",
				})
			}
		}

		if node.Weight <= 0 {
			*issues = append(*issues, ValidationIssue{
				Path:    fmt.Sprintf("loadBalancer.pools[%d].nodes[%d].weight", poolIndexValue, nodeIdx),
				Message: "weight must be positive",
			})
		}
	}
}
