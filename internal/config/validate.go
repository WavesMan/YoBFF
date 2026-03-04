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
