package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"YoBFF/internal/store"
)

// siteLogStream 读取或更新站点日志流探查配置，用于站点独立观测。
// 参数：w 为响应写入器，r 为请求对象，siteID 为站点标识。
// 返回：日志流配置。
// 异常：站点未配置日志流或请求非法时返回错误。
func (s *Server) siteLogStream(w http.ResponseWriter, r *http.Request, siteID string) {
	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "store_unavailable", "site store unavailable", r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		stream, err := s.store.GetSiteLogStream(siteID)
		if err != nil {
			writeSiteError(w, err, r)
			return
		}
		writeJSON(w, http.StatusOK, stream)
	case http.MethodPut:
		var payload store.SiteLogStream
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", err.Error(), r)
			return
		}
		stream, err := s.store.UpdateSiteLogStream(siteID, strings.TrimSpace(payload.FilterQuery))
		if err != nil {
			writeSiteError(w, err, r)
			return
		}
		_ = s.store.SaveAudit("site_log_stream_update", siteID, operatorFromRequest(r), map[string]any{
			"filter_query": stream.FilterQuery,
		})
		writeJSON(w, http.StatusOK, stream)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
	}
}

func (s *Server) siteLogHistory(w http.ResponseWriter, r *http.Request, siteID string) {
	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "store_unavailable", "site store unavailable", r)
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
		return
	}
	query := store.SiteLogQuery{
		Kind:  strings.TrimSpace(r.URL.Query().Get("kind")),
		Level: strings.TrimSpace(r.URL.Query().Get("level")),
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
	items, err := s.store.ListSiteLogs(siteID, query)
	if err != nil {
		writeSiteError(w, err, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
	})
}

// writeSiteError 将站点相关错误映射为统一响应结构，便于前端处理。
// 参数：w 为响应写入器，err 为错误对象，r 为请求对象。
// 返回：无。
// 异常：无。
func writeSiteError(w http.ResponseWriter, err error, r *http.Request) {
	if err == nil {
		return
	}
	switch {
	case errors.Is(err, store.ErrSiteNotFound):
		writeError(w, http.StatusNotFound, "site_not_found", err.Error(), r)
	case errors.Is(err, store.ErrSiteVersionMissing):
		writeError(w, http.StatusNotFound, "version_not_found", err.Error(), r)
	case errors.Is(err, store.ErrSiteLogNotFound):
		writeError(w, http.StatusNotFound, "log_stream_not_found", err.Error(), r)
	default:
		writeError(w, http.StatusInternalServerError, "store_failed", err.Error(), r)
	}
}
