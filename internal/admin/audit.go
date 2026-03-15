package admin

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"YoBFF/internal/store"
)

// auditLogs 读取审计日志列表，用于管理台操作追溯与筛选。
// 参数：w 为响应写入器，r 为请求对象。
// 返回：审计日志列表。
// 异常：请求参数非法或存储查询失败时返回错误。
func (s *Server) auditLogs(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "store_unavailable", "audit store unavailable", r)
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
		return
	}
	query := store.AuditLogQuery{
		Action:   strings.TrimSpace(r.URL.Query().Get("action")),
		Operator: strings.TrimSpace(r.URL.Query().Get("operator")),
		Target:   strings.TrimSpace(r.URL.Query().Get("target")),
	}
	startTime := strings.TrimSpace(r.URL.Query().Get("start_time"))
	endTime := strings.TrimSpace(r.URL.Query().Get("end_time"))
	if startTime != "" {
		if _, err := time.Parse(time.RFC3339, startTime); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "start_time must be RFC3339", r)
			return
		}
		query.StartTime = startTime
	}
	if endTime != "" {
		if _, err := time.Parse(time.RFC3339, endTime); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "end_time must be RFC3339", r)
			return
		}
		query.EndTime = endTime
	}
	limitRaw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if limitRaw != "" {
		limit, err := strconv.Atoi(limitRaw)
		if err != nil || limit <= 0 {
			writeError(w, http.StatusBadRequest, "invalid_request", "limit must be positive", r)
			return
		}
		query.Limit = limit
	}
	pageRaw := strings.TrimSpace(r.URL.Query().Get("page"))
	if pageRaw != "" {
		page, err := strconv.Atoi(pageRaw)
		if err != nil || page <= 0 {
			writeError(w, http.StatusBadRequest, "invalid_request", "page must be positive", r)
			return
		}
		query.Page = page
	}
	pageSizeRaw := strings.TrimSpace(r.URL.Query().Get("pageSize"))
	if pageSizeRaw != "" {
		pageSize, err := strconv.Atoi(pageSizeRaw)
		if err != nil || pageSize <= 0 {
			writeError(w, http.StatusBadRequest, "invalid_request", "pageSize must be positive", r)
			return
		}
		query.PageSize = pageSize
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		if query.Limit > 0 {
			query.PageSize = query.Limit
		} else {
			query.PageSize = 15
		}
	}
	items, total, err := s.store.ListAuditLogs(query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "audit_query_failed", err.Error(), r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":    items,
		"total":    total,
		"page":     query.Page,
		"pageSize": query.PageSize,
	})
}
