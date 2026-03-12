package config

import (
	"fmt"
	"net/netip"
	"strings"
	"time"
)

type siteCDNSnapshot struct {
	prefixes    []netip.Prefix
	lastSuccess time.Time
}

// UpdateSiteCDNProviderSnapshot 更新站点维度的 CDN 回源 CIDR 快照。
// 参数：siteID 为站点 ID，provider 为厂商标识，cidrs 为 CIDR 列表，lastSuccess 为本次快照的成功时间。
// 返回：更新失败错误。
// 异常：CIDR 非法时返回错误。
func (m *Manager) UpdateSiteCDNProviderSnapshot(siteID string, provider string, cidrs []string, lastSuccess time.Time) error {
	siteID = strings.TrimSpace(siteID)
	provider = strings.ToLower(strings.TrimSpace(provider))
	if siteID == "" || provider == "" {
		return fmt.Errorf("siteID/provider 不能为空")
	}

	prefixes := make([]netip.Prefix, 0, len(cidrs))
	for _, item := range cidrs {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			return fmt.Errorf("非法 CIDR %q: %w", value, err)
		}
		prefixes = append(prefixes, prefix)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.siteCDNProviderSnapshots == nil {
		m.siteCDNProviderSnapshots = make(map[string]map[string]siteCDNSnapshot)
	}
	snapshots, ok := m.siteCDNProviderSnapshots[siteID]
	if !ok {
		snapshots = make(map[string]siteCDNSnapshot)
		m.siteCDNProviderSnapshots[siteID] = snapshots
	}
	snapshots[provider] = siteCDNSnapshot{
		prefixes:    prefixes,
		lastSuccess: lastSuccess,
	}
	return nil
}

// IsIPAllowedBySiteProviders 判断 IP 是否属于站点启用的 CDN 回源 CIDR。
// 参数：siteID 为站点 ID，ip 为来源地址，providers 为启用厂商列表，maxStaleness 返回指定厂商允许的最大陈旧时长。
// 返回：命中则返回 true，否则返回 false。
// 异常：无。
func (m *Manager) IsIPAllowedBySiteProviders(siteID string, ip netip.Addr, providers []string, maxStaleness func(provider string) time.Duration) bool {
	if !ip.IsValid() || len(providers) == 0 {
		return false
	}

	now := time.Now()

	m.mu.Lock()
	snapshots := m.siteCDNProviderSnapshots[siteID]
	localSnapshots := make(map[string]siteCDNSnapshot, len(snapshots))
	for key, value := range snapshots {
		localSnapshots[key] = value
	}
	m.mu.Unlock()
	if len(localSnapshots) == 0 {
		return false
	}

	for _, name := range providers {
		provider := strings.ToLower(strings.TrimSpace(name))
		if provider == "" {
			continue
		}

		snapshot, ok := localSnapshots[provider]
		if !ok || snapshot.lastSuccess.IsZero() || len(snapshot.prefixes) == 0 {
			continue
		}

		staleness := time.Duration(0)
		if maxStaleness != nil {
			staleness = maxStaleness(provider)
		}
		if staleness > 0 && now.Sub(snapshot.lastSuccess) > staleness {
			continue
		}

		for _, prefix := range snapshot.prefixes {
			if prefix.Contains(ip) {
				return true
			}
		}
	}
	return false
}
