package admin

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"YoBFF/internal/config"
	"YoBFF/internal/store"
)

type siteCreateRequest struct {
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
	IP       string `json:"ip"`
}

type siteUpdateRequest struct {
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
	IP       string `json:"ip"`
}

type siteSummary struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
	IP       string `json:"ip"`
}

type siteGroup struct {
	GroupKey string        `json:"group_key"`
	Count    int           `json:"count"`
	Sites    []siteSummary `json:"sites"`
}

// registerSiteRoutes 注册站点级管理接口，确保站点入口与全局配置互不干扰。
// 参数：mux 为路由复用器。
// 返回：无。
// 异常：无。
func (s *Server) registerSiteRoutes(mux *http.ServeMux) {
	if mux == nil {
		return
	}
	mux.Handle("/api/v1/site-groups", withAuth(http.HandlerFunc(s.siteGroups), s.manager))
	mux.Handle("/api/v1/sites", withAuth(http.HandlerFunc(s.sites), s.manager))
	mux.Handle("/api/v1/sites/", withAuth(http.HandlerFunc(s.siteRouter), s.manager))
}

// siteGroups 返回按 hostname/IP 分类后的站点分组，便于快速筛选与定位。
// 参数：w 为响应写入器，r 为请求对象。
// 返回：站点分组响应。
// 异常：请求非法或存储异常时返回错误。
func (s *Server) siteGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
		return
	}
	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "store_unavailable", "site store unavailable", r)
		return
	}
	groupBy := strings.TrimSpace(r.URL.Query().Get("by"))
	if groupBy != "hostname" && groupBy != "ip" {
		writeError(w, http.StatusBadRequest, "invalid_request", "by must be hostname or ip", r)
		return
	}
	items, err := s.store.ListSites(store.SiteFilter{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_query_failed", err.Error(), r)
		return
	}
	groupMap := make(map[string][]siteSummary)
	for _, item := range items {
		groupKey := item.Hostname
		if groupBy == "ip" {
			groupKey = item.IP
		}
		groupMap[groupKey] = append(groupMap[groupKey], siteSummary{
			ID:       item.ID,
			Name:     item.Name,
			Hostname: item.Hostname,
			IP:       item.IP,
		})
	}
	var groups []siteGroup
	for key, values := range groupMap {
		groups = append(groups, siteGroup{
			GroupKey: key,
			Count:    len(values),
			Sites:    values,
		})
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].GroupKey < groups[j].GroupKey
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"groups": groups,
	})
}

// sites 提供站点清单与创建入口，用于管理站点基础信息。
// 参数：w 为响应写入器，r 为请求对象。
// 返回：GET 返回站点列表，POST 返回新建站点。
// 异常：请求非法或存储异常时返回错误。
func (s *Server) sites(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "store_unavailable", "site store unavailable", r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		filter := store.SiteFilter{
			Hostname: strings.TrimSpace(r.URL.Query().Get("hostname")),
			IP:       strings.TrimSpace(r.URL.Query().Get("ip")),
		}
		items, err := s.store.ListSites(filter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "store_query_failed", err.Error(), r)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"items": items,
		})
	case http.MethodPost:
		var payload siteCreateRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", err.Error(), r)
			return
		}
		site, err := s.store.CreateSite(store.Site{
			Name:     payload.Name,
			Hostname: payload.Hostname,
			IP:       payload.IP,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, "site_create_failed", err.Error(), r)
			return
		}
		_ = s.store.SaveAudit("site_create", site.ID, operatorFromRequest(r), map[string]any{
			"hostname": site.Hostname,
			"ip":       site.IP,
		})
		writeJSON(w, http.StatusOK, site)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
	}
}

// siteRouter 解析站点级子路径并分发到具体处理器，避免引入外部路由依赖。
// 参数：w 为响应写入器，r 为请求对象。
// 返回：无。
// 异常：路径不匹配时返回 404 错误结构。
func (s *Server) siteRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/sites/")
	if path == "" || path == r.URL.Path {
		writeError(w, http.StatusNotFound, "not_found", "not found", r)
		return
	}
	parts := strings.Split(path, "/")
	siteID := strings.TrimSpace(parts[0])
	if siteID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "site_id is required", r)
		return
	}
	if len(parts) == 1 {
		s.siteDetail(w, r, siteID)
		return
	}
	switch parts[1] {
	case "config":
		s.routeSiteConfig(w, r, siteID, parts[2:])
	case "log":
		s.routeSiteLog(w, r, siteID, parts[2:])
	default:
		writeError(w, http.StatusNotFound, "not_found", "not found", r)
	}
}

// routeSiteConfig 分流站点配置相关路径，保证配置操作入口清晰可控。
// 参数：w 为响应写入器，r 为请求对象，siteID 为站点标识，parts 为剩余路径片段。
// 返回：无。
// 异常：路径不匹配时返回 404 错误结构。
func (s *Server) routeSiteConfig(w http.ResponseWriter, r *http.Request, siteID string, parts []string) {
	if len(parts) == 0 {
		s.siteConfig(w, r, siteID)
		return
	}
	switch parts[0] {
	case "validate":
		s.siteConfigValidate(w, r, siteID)
	case "diff":
		s.siteConfigDiff(w, r, siteID)
	case "versions":
		if len(parts) == 1 {
			s.siteConfigVersions(w, r, siteID)
			return
		}
		if len(parts) == 2 {
			s.siteConfigVersionDetail(w, r, siteID, parts[1])
			return
		}
		writeError(w, http.StatusNotFound, "not_found", "not found", r)
	case "rollback":
		s.siteConfigRollback(w, r, siteID)
	default:
		writeError(w, http.StatusNotFound, "not_found", "not found", r)
	}
}

// routeSiteLog 分流站点日志流路径，避免与配置入口耦合。
// 参数：w 为响应写入器，r 为请求对象，siteID 为站点标识，parts 为剩余路径片段。
// 返回：无。
// 异常：路径不匹配时返回 404 错误结构。
func (s *Server) routeSiteLog(w http.ResponseWriter, r *http.Request, siteID string, parts []string) {
	if len(parts) == 1 && parts[0] == "stream" {
		s.siteLogStream(w, r, siteID)
		return
	}
	writeError(w, http.StatusNotFound, "not_found", "not found", r)
}

// siteDetail 读取或更新站点信息，用于站点元数据维护。
// 参数：w 为响应写入器，r 为请求对象，siteID 为站点标识。
// 返回：GET 返回站点信息，PUT 返回更新后的站点信息。
// 异常：站点不存在或请求非法时返回错误。
func (s *Server) siteDetail(w http.ResponseWriter, r *http.Request, siteID string) {
	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "store_unavailable", "site store unavailable", r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		site, err := s.store.GetSite(siteID)
		if err != nil {
			writeSiteError(w, err, r)
			return
		}
		writeJSON(w, http.StatusOK, site)
	case http.MethodPut:
		var payload siteUpdateRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", err.Error(), r)
			return
		}
		site, err := s.store.UpdateSite(siteID, store.Site{
			Name:     payload.Name,
			Hostname: payload.Hostname,
			IP:       payload.IP,
		})
		if err != nil {
			writeSiteError(w, err, r)
			return
		}
		_ = s.store.SaveAudit("site_update", site.ID, operatorFromRequest(r), map[string]any{
			"hostname": site.Hostname,
			"ip":       site.IP,
		})
		writeJSON(w, http.StatusOK, site)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
	}
}

// siteConfig 获取或更新站点配置，用于站点级配置入口。
// 参数：w 为响应写入器，r 为请求对象，siteID 为站点标识。
// 返回：GET 返回当前配置，PUT 返回更新生成的版本信息。
// 异常：站点不存在或配置非法时返回错误。
func (s *Server) siteConfig(w http.ResponseWriter, r *http.Request, siteID string) {
	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "store_unavailable", "site store unavailable", r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		cfg, err := s.store.GetSiteConfig(siteID)
		if err != nil {
			writeSiteError(w, err, r)
			return
		}
		writeJSON(w, http.StatusOK, cfg)
	case http.MethodPut:
		var cfg config.Config
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&cfg); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", err.Error(), r)
			return
		}
		issues := s.manager.Validate(cfg)
		if len(issues) > 0 {
			writeValidationError(w, http.StatusBadRequest, "config_invalid", "config validation failed", issues, r)
			return
		}
		version, err := s.store.UpdateSiteConfig(siteID, cfg, operatorFromRequest(r), "update")
		if err != nil {
			writeSiteError(w, err, r)
			return
		}
		_ = s.store.SaveAudit("site_config_update", siteID, operatorFromRequest(r), map[string]any{
			"version_id": version.ID,
		})
		writeJSON(w, http.StatusOK, version)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
	}
}

// siteConfigValidate 执行站点配置预检，用于提交前的规则校验。
// 参数：w 为响应写入器，r 为请求对象，siteID 为站点标识。
// 返回：预检结果与错误列表。
// 异常：请求非法时返回错误。
func (s *Server) siteConfigValidate(w http.ResponseWriter, r *http.Request, siteID string) {
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
	var cfg config.Config
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error(), r)
		return
	}
	issues := s.manager.Validate(cfg)
	writeJSON(w, http.StatusOK, map[string]any{
		"valid":  len(issues) == 0,
		"errors": issues,
	})
}

// siteConfigDiff 输出站点配置差异，用于改动审查与对比。
// 参数：w 为响应写入器，r 为请求对象，siteID 为站点标识。
// 返回：配置差异列表。
// 异常：站点不存在或请求非法时返回错误。
func (s *Server) siteConfigDiff(w http.ResponseWriter, r *http.Request, siteID string) {
	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "store_unavailable", "site store unavailable", r)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
		return
	}
	var targetConfig config.Config
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&targetConfig); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error(), r)
		return
	}
	baseConfig, err := s.store.GetSiteConfig(siteID)
	if err != nil {
		writeSiteError(w, err, r)
		return
	}
	changes, err := buildConfigDiff(baseConfig, targetConfig)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "diff_failed", err.Error(), r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"changes": changes,
	})
}

// siteConfigVersions 返回站点配置版本列表，用于回滚与预览入口。
// 参数：w 为响应写入器，r 为请求对象，siteID 为站点标识。
// 返回：版本列表。
// 异常：站点不存在或存储异常时返回错误。
func (s *Server) siteConfigVersions(w http.ResponseWriter, r *http.Request, siteID string) {
	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "store_unavailable", "site store unavailable", r)
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
		return
	}
	limit := parsePositiveInt(r.URL.Query().Get("limit"), 10)
	items, err := s.store.ListSiteVersions(siteID, limit)
	if err != nil {
		writeSiteError(w, err, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
	})
}

// siteConfigVersionDetail 返回指定版本配置，用于版本预览。
// 参数：w 为响应写入器，r 为请求对象，siteID 为站点标识，versionID 为版本标识。
// 返回：版本配置。
// 异常：版本不存在或存储异常时返回错误。
func (s *Server) siteConfigVersionDetail(w http.ResponseWriter, r *http.Request, siteID string, versionID string) {
	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "store_unavailable", "site store unavailable", r)
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
		return
	}
	if versionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "version_id is required", r)
		return
	}
	cfg, err := s.store.GetSiteVersionConfig(siteID, versionID)
	if err != nil {
		writeSiteError(w, err, r)
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

// siteConfigRollback 执行站点配置回滚，用于快速恢复站点配置。
// 参数：w 为响应写入器，r 为请求对象，siteID 为站点标识。
// 返回：回滚结果。
// 异常：请求非法、版本不存在或存储异常时返回错误。
func (s *Server) siteConfigRollback(w http.ResponseWriter, r *http.Request, siteID string) {
	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "store_unavailable", "site store unavailable", r)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
		return
	}
	var payload struct {
		VersionID string `json:"version_id"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error(), r)
		return
	}
	if payload.VersionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "version_id is required", r)
		return
	}
	version, err := s.store.RollbackSiteVersion(siteID, payload.VersionID, operatorFromRequest(r))
	if err != nil {
		writeSiteError(w, err, r)
		return
	}
	_ = s.store.SaveAudit("site_config_rollback", siteID, operatorFromRequest(r), map[string]any{
		"origin_id":  payload.VersionID,
		"version_id": version.ID,
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"status":     "rolled_back",
		"version_id": payload.VersionID,
	})
}
