package admin

import (
	"encoding/json"
	"net/http"
	"strings"

	"YoBFF/internal/config"
)

// lbPools 提供流量池列表查询与创建能力。
// 参数：w 为响应写入器，r 为请求对象。
// 返回：GET 返回池列表，POST 返回创建结果。
// 异常：请求非法或配置应用失败时返回错误。
func (s *Server) lbPools(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg := s.manager.CurrentConfig()
		writeJSON(w, http.StatusOK, map[string]any{
			"items": cfg.LoadBalancer.Pools,
		})
	case http.MethodPost:
		var payload config.LBPool
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", err.Error(), r)
			return
		}
		next := s.manager.CurrentConfig()
		payload.ID = normalizePoolID(payload.ID)
		if payload.ID == "" {
			payload.ID = "pool_" + newRequestID()[:12]
		}
		payload.Strategy = normalizeStrategy(payload.Strategy)
		next.LoadBalancer.Pools = append(next.LoadBalancer.Pools, payload)
		if !s.applyLBConfig(w, r, next, "lb_pool_create") {
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "created",
			"id":     payload.ID,
		})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
	}
}

// lbPoolDetail 提供流量池详情查询、更新与删除能力。
// 参数：w 为响应写入器，r 为请求对象。
// 返回：GET 返回详情，PUT 返回更新结果，DELETE 返回删除结果。
// 异常：请求非法或资源不存在时返回错误。
func (s *Server) lbPoolDetail(w http.ResponseWriter, r *http.Request) {
	poolID := extractTail(r.URL.Path, "/api/v1/lb/pools/")
	if poolID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "pool id is required", r)
		return
	}
	cfg := s.manager.CurrentConfig()
	index := findPoolIndex(cfg.LoadBalancer.Pools, poolID)
	if index < 0 {
		writeError(w, http.StatusNotFound, "pool_not_found", "pool not found", r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, cfg.LoadBalancer.Pools[index])
	case http.MethodPut:
		var payload config.LBPool
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", err.Error(), r)
			return
		}
		payload.ID = poolID
		payload.Strategy = normalizeStrategy(payload.Strategy)
		cfg.LoadBalancer.Pools[index] = payload
		if !s.applyLBConfig(w, r, cfg, "lb_pool_update") {
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "updated",
			"pool_id": poolID,
		})
	case http.MethodDelete:
		cfg.LoadBalancer.Pools = append(cfg.LoadBalancer.Pools[:index], cfg.LoadBalancer.Pools[index+1:]...)
		cfg.LoadBalancer.Routes = removeRoutesByPoolID(cfg.LoadBalancer.Routes, poolID)
		if strings.EqualFold(strings.TrimSpace(cfg.LoadBalancer.DefaultPoolID), poolID) {
			cfg.LoadBalancer.DefaultPoolID = ""
		}
		if !s.applyLBConfig(w, r, cfg, "lb_pool_delete") {
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "deleted",
			"pool_id": poolID,
		})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
	}
}

// lbRoutes 提供域名绑定规则列表查询。
// 参数：w 为响应写入器，r 为请求对象。
// 返回：GET 返回规则列表。
// 异常：方法非法时返回错误。
func (s *Server) lbRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
		return
	}
	cfg := s.manager.CurrentConfig()
	writeJSON(w, http.StatusOK, map[string]any{
		"items":           cfg.LoadBalancer.Routes,
		"default_pool_id": cfg.LoadBalancer.DefaultPoolID,
	})
}

// lbRouteDetail 提供域名绑定规则查询、更新与删除能力。
// 参数：w 为响应写入器，r 为请求对象。
// 返回：GET 返回绑定详情，PUT 返回更新结果，DELETE 返回删除结果。
// 异常：请求非法或规则不存在时返回错误。
func (s *Server) lbRouteDetail(w http.ResponseWriter, r *http.Request) {
	domain := normalizeDomain(extractTail(r.URL.Path, "/api/v1/lb/routes/"))
	if domain == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "domain is required", r)
		return
	}
	cfg := s.manager.CurrentConfig()
	index := findRouteIndex(cfg.LoadBalancer.Routes, domain)
	switch r.Method {
	case http.MethodGet:
		if index < 0 {
			writeError(w, http.StatusNotFound, "route_not_found", "route not found", r)
			return
		}
		writeJSON(w, http.StatusOK, cfg.LoadBalancer.Routes[index])
	case http.MethodPut:
		var payload config.LBRouteRule
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", err.Error(), r)
			return
		}
		payload.Domain = domain
		payload.PoolID = normalizePoolID(payload.PoolID)
		payload.FallbackPoolID = normalizePoolID(payload.FallbackPoolID)
		if index >= 0 {
			cfg.LoadBalancer.Routes[index] = payload
		} else {
			cfg.LoadBalancer.Routes = append(cfg.LoadBalancer.Routes, payload)
		}
		if !s.applyLBConfig(w, r, cfg, "lb_route_upsert") {
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "updated",
			"domain": domain,
		})
	case http.MethodDelete:
		if index < 0 {
			writeError(w, http.StatusNotFound, "route_not_found", "route not found", r)
			return
		}
		cfg.LoadBalancer.Routes = append(cfg.LoadBalancer.Routes[:index], cfg.LoadBalancer.Routes[index+1:]...)
		if !s.applyLBConfig(w, r, cfg, "lb_route_delete") {
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "deleted",
			"domain": domain,
		})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
	}
}

// applyLBConfig 复用统一校验与应用逻辑并记录版本审计。
// 参数：w 为响应写入器，r 为请求对象，cfg 为待应用配置，source 为审计来源。
// 返回：true 表示应用成功，false 表示已写出错误响应。
// 异常：无。
func (s *Server) applyLBConfig(w http.ResponseWriter, r *http.Request, cfg config.Config, source string) bool {
	issues := s.manager.Validate(cfg)
	if len(issues) > 0 {
		writeValidationError(w, http.StatusBadRequest, "config_invalid", "config validation failed", issues, r)
		return false
	}
	if err := s.manager.Apply(cfg); err != nil {
		writeError(w, http.StatusBadRequest, "config_apply_failed", err.Error(), r)
		return false
	}
	if s.store != nil {
		version, err := s.store.SaveVersion(cfg, operatorFromRequest(r), source)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "config_version_failed", err.Error(), r)
			return false
		}
		_ = s.store.SaveAudit(source, version.ID, operatorFromRequest(r), map[string]any{
			"source": source,
		})
	}
	return true
}

// findPoolIndex 在池列表中查找指定池标识。
// 参数：items 为池列表，poolID 为目标标识。
// 返回：命中索引，未命中返回 -1。
// 异常：无。
func findPoolIndex(items []config.LBPool, poolID string) int {
	target := normalizePoolID(poolID)
	for idx, item := range items {
		if strings.EqualFold(normalizePoolID(item.ID), target) {
			return idx
		}
	}
	return -1
}

// findRouteIndex 在域名规则列表中查找目标域名。
// 参数：items 为规则列表，domain 为目标域名。
// 返回：命中索引，未命中返回 -1。
// 异常：无。
func findRouteIndex(items []config.LBRouteRule, domain string) int {
	target := normalizeDomain(domain)
	for idx, item := range items {
		if normalizeDomain(item.Domain) == target {
			return idx
		}
	}
	return -1
}

// removeRoutesByPoolID 删除绑定了指定池的所有路由规则。
// 参数：items 为规则列表，poolID 为目标池标识。
// 返回：过滤后的规则列表。
// 异常：无。
func removeRoutesByPoolID(items []config.LBRouteRule, poolID string) []config.LBRouteRule {
	filtered := make([]config.LBRouteRule, 0, len(items))
	target := normalizePoolID(poolID)
	for _, item := range items {
		primary := normalizePoolID(item.PoolID)
		fallback := normalizePoolID(item.FallbackPoolID)
		if primary == target || fallback == target {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

// extractTail 从固定前缀路径中提取尾部参数值。
// 参数：path 为完整路径，prefix 为固定前缀。
// 返回：去除首尾空白后的尾部值。
// 异常：无。
func extractTail(path string, prefix string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.Trim(path[len(prefix):], "/"))
}

// normalizeDomain 归一化域名文本，统一为小写去空白格式。
// 参数：value 为原始域名文本。
// 返回：归一化后的域名。
// 异常：无。
func normalizeDomain(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// normalizePoolID 归一化池标识，统一为去空白文本。
// 参数：value 为原始池标识。
// 返回：归一化后的池标识。
// 异常：无。
func normalizePoolID(value string) string {
	return strings.TrimSpace(value)
}

// normalizeStrategy 归一化选路策略，空值时回退到 weighted_rr。
// 参数：value 为原始策略值。
// 返回：归一化后的策略值。
// 异常：无。
func normalizeStrategy(value string) string {
	if strings.TrimSpace(value) == "" {
		return "weighted_rr"
	}
	return strings.TrimSpace(strings.ToLower(value))
}
