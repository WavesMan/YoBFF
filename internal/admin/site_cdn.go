package admin

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"YoBFF/internal/config"
	"YoBFF/internal/plugins/cdn"
	"YoBFF/internal/plugins/cdn/aliyun"
	"YoBFF/internal/plugins/cdn/cloudflare"
	"YoBFF/internal/plugins/cdn/tencent"
	"YoBFF/internal/store"
)

type siteCDNRefreshRequest struct {
	Providers []string `json:"providers"`
}

type siteCDNOriginStatusItem struct {
	SiteID              string  `json:"site_id"`
	Provider            string  `json:"provider"`
	LastAttemptAt       *string `json:"last_attempt_at,omitempty"`
	LastSuccessAt       *string `json:"last_success_at,omitempty"`
	ConsecutiveFailures int     `json:"consecutive_failures"`
	LastError           string  `json:"last_error,omitempty"`
	UpdatedAt           string  `json:"updated_at"`
}

// routeSiteCDN 分流站点 CDN 相关路径，避免与站点配置入口耦合。
// 参数：w 为响应写入器，r 为请求对象，siteID 为站点标识，parts 为剩余路径片段。
// 返回：无。
// 异常：路径不匹配时返回 404 错误结构。
func (s *Server) routeSiteCDN(w http.ResponseWriter, r *http.Request, siteID string, parts []string) {
	for len(parts) > 0 && strings.TrimSpace(parts[len(parts)-1]) == "" {
		parts = parts[:len(parts)-1]
	}
	if len(parts) == 0 {
		writeError(w, http.StatusNotFound, "not_found", "not found", r)
		return
	}
	switch parts[0] {
	case "origin":
		s.routeSiteCDNOrigin(w, r, siteID, parts[1:])
	default:
		writeError(w, http.StatusNotFound, "not_found", "not found", r)
	}
}

// routeSiteCDNOrigin 分流站点回源 IP 相关路径。
// 参数：w 为响应写入器，r 为请求对象，siteID 为站点标识，parts 为剩余路径片段。
// 返回：无。
// 异常：路径不匹配时返回 404 错误结构。
func (s *Server) routeSiteCDNOrigin(w http.ResponseWriter, r *http.Request, siteID string, parts []string) {
	for len(parts) > 0 && strings.TrimSpace(parts[len(parts)-1]) == "" {
		parts = parts[:len(parts)-1]
	}
	if len(parts) == 1 && parts[0] == "status" {
		s.siteCDNOriginStatus(w, r, siteID)
		return
	}
	if len(parts) == 1 && parts[0] == "refresh" {
		s.siteCDNOriginRefresh(w, r, siteID)
		return
	}
	writeError(w, http.StatusNotFound, "not_found", "not found", r)
}

// siteCDNOriginStatus 返回站点回源 IP 的同步状态。
// 参数：w 为响应写入器，r 为请求对象，siteID 为站点标识。
// 返回：状态列表。
// 异常：站点不存在或存储异常时返回错误。
func (s *Server) siteCDNOriginStatus(w http.ResponseWriter, r *http.Request, siteID string) {
	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "store_unavailable", "site store unavailable", r)
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
		return
	}
	if _, err := s.store.GetSite(siteID); err != nil {
		writeSiteError(w, err, r)
		return
	}

	items, err := s.store.ListSiteCDNOriginStatus(siteID)
	if err != nil {
		writeSiteError(w, err, r)
		return
	}
	resp := make([]siteCDNOriginStatusItem, 0, len(items))
	for _, item := range items {
		siteIDValue := strings.TrimSpace(item.SiteID)
		providerValue := strings.ToLower(strings.TrimSpace(item.Provider))
		lastAttemptText := strings.TrimSpace(item.LastAttemptAt)
		lastSuccessText := strings.TrimSpace(item.LastSuccessAt)
		lastErrorText := strings.TrimSpace(item.LastError)

		var lastAttemptPtr *string
		var lastSuccessPtr *string
		if lastAttemptText != "" {
			lastAttemptPtr = &lastAttemptText
		}
		if lastSuccessText != "" {
			lastSuccessPtr = &lastSuccessText
		}

		resp = append(resp, siteCDNOriginStatusItem{
			SiteID:              siteIDValue,
			Provider:            providerValue,
			LastAttemptAt:       lastAttemptPtr,
			LastSuccessAt:       lastSuccessPtr,
			ConsecutiveFailures: item.ConsecutiveFailures,
			LastError:           lastErrorText,
			UpdatedAt:           strings.TrimSpace(item.UpdatedAt),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": resp,
	})
}

// siteCDNOriginRefresh 触发站点回源 IP 的一次性拉取并写入快照。
// 参数：w 为响应写入器，r 为请求对象，siteID 为站点标识。
// 返回：刷新结果。
// 异常：站点不存在、请求非法或拉取失败时返回错误。
func (s *Server) siteCDNOriginRefresh(w http.ResponseWriter, r *http.Request, siteID string) {
	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "store_unavailable", "site store unavailable", r)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
		return
	}
	if _, err := s.store.GetSite(siteID); err != nil {
		writeSiteError(w, err, r)
		return
	}

	var payload siteCDNRefreshRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error(), r)
		return
	}

	siteCfg, err := s.store.GetSiteConfig(siteID)
	if err != nil {
		writeSiteError(w, err, r)
		return
	}

	providers := normalizeProviders(payload.Providers)
	if len(providers) == 0 {
		providers = normalizeProviders(siteCfg.Security.AllowedCDNProviders)
	}
	if len(providers) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "providers is required", r)
		return
	}

	statuses, err := s.store.ListSiteCDNOriginStatus(siteID)
	if err != nil {
		writeSiteError(w, err, r)
		return
	}
	statusMap := make(map[string]store.SiteCDNOriginStatus, len(statuses))
	for _, item := range statuses {
		statusMap[strings.ToLower(strings.TrimSpace(item.Provider))] = item
	}

	type itemResult struct {
		Provider string `json:"provider"`
		OK       bool   `json:"ok"`
		Count    int    `json:"count,omitempty"`
		Error    string `json:"error,omitempty"`
	}

	now := time.Now().UTC()
	results := make([]itemResult, 0, len(providers))
	for _, providerName := range providers {
		settings := config.CDNProviderSetting{}
		if siteCfg.Security.CDNProviderSettings != nil {
			if value, ok := siteCfg.Security.CDNProviderSettings[providerName]; ok {
				settings = value
			}
		}

		provider, providerErr := buildCDNProvider(providerName, settings)
		if providerErr != nil {
			results = append(results, itemResult{
				Provider: providerName,
				OK:       false,
				Error:    providerErr.Error(),
			})
			continue
		}

		prevFailures := 0
		if previous, ok := statusMap[providerName]; ok {
			prevFailures = previous.ConsecutiveFailures
		}

		attempt := now
		_, _ = s.store.UpsertSiteCDNOriginStatus(siteID, providerName, store.SiteCDNOriginStatusUpdate{
			LastAttemptAt:       &attempt,
			ConsecutiveFailures: prevFailures,
			LastError:           "",
		})

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		cidrs, err := provider.FetchCIDRs(ctx)
		cancel()

		if err != nil {
			failures := prevFailures + 1
			_, _ = s.store.UpsertSiteCDNOriginStatus(siteID, providerName, store.SiteCDNOriginStatusUpdate{
				LastAttemptAt:       &attempt,
				ConsecutiveFailures: failures,
				LastError:           err.Error(),
			})
			results = append(results, itemResult{
				Provider: providerName,
				OK:       false,
				Error:    err.Error(),
			})
			continue
		}

		_, err = s.store.SaveSiteCDNOriginSnapshot(siteID, providerName, cidrs, now, "manual")
		if err != nil {
			failures := prevFailures + 1
			_, _ = s.store.UpsertSiteCDNOriginStatus(siteID, providerName, store.SiteCDNOriginStatusUpdate{
				LastAttemptAt:       &attempt,
				ConsecutiveFailures: failures,
				LastError:           err.Error(),
			})
			results = append(results, itemResult{
				Provider: providerName,
				OK:       false,
				Error:    err.Error(),
			})
			continue
		}

		success := now
		_, _ = s.store.UpsertSiteCDNOriginStatus(siteID, providerName, store.SiteCDNOriginStatusUpdate{
			LastAttemptAt:       &attempt,
			LastSuccessAt:       &success,
			ConsecutiveFailures: 0,
			LastError:           "",
		})
		if s.manager != nil {
			_ = s.manager.UpdateSiteCDNProviderSnapshot(siteID, providerName, cidrs, success)
		}

		results = append(results, itemResult{
			Provider: providerName,
			OK:       true,
			Count:    len(cidrs),
		})
	}

	if s.store != nil {
		_ = s.store.SaveAudit("site_cdn_origin_refresh", siteID, operatorFromRequest(r), map[string]any{
			"providers": providers,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "refreshed",
		"results": results,
	})
}

// buildCDNProvider 根据站点配置构建 CDN 回源 IP 拉取器。
// 说明：阿里云/腾讯云使用官方 API 方式拉取，SecretKey 由存储层解密后传入，不在接口中回显。
func buildCDNProvider(provider string, settings config.CDNProviderSetting) (cdn.Provider, error) {
	switch provider {
	case "cloudflare":
		return cloudflare.NewProvider(cloudflare.Config{
			IPv4URL: strings.TrimSpace(settings.IPv4URL),
			IPv6URL: strings.TrimSpace(settings.IPv6URL),
		}), nil
	case "aliyun":
		return aliyun.NewProvider(aliyun.Config{
			AccessKeyID:     strings.TrimSpace(settings.APIKey),
			AccessKeySecret: strings.TrimSpace(settings.SecretKey),
			Endpoint:        strings.TrimSpace(settings.Endpoint),
			SiteID:          strings.TrimSpace(settings.Option),
		}), nil
	case "tencent":
		return tencent.NewProvider(tencent.Config{
			SecretID:  strings.TrimSpace(settings.APIKey),
			SecretKey: strings.TrimSpace(settings.SecretKey),
			ZoneID:    strings.TrimSpace(settings.ZoneID),
			Endpoint:  strings.TrimSpace(settings.Endpoint),
		}), nil
	default:
		return nil, errors.New("cdn provider is not supported")
	}
}

func normalizeProviders(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	var providers []string
	for _, item := range items {
		value := strings.ToLower(strings.TrimSpace(item))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		providers = append(providers, value)
	}
	return providers
}
