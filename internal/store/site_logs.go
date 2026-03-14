package store

import (
	"database/sql"
	"errors"
	"sort"
	"time"
)

type SiteLogQuery struct {
	Kind      string
	Level     string
	StartTime string
	EndTime   string
	Limit     int
}

type SiteLogEntry struct {
	ID         string `json:"id"`
	SiteID     string `json:"site_id"`
	Kind       string `json:"kind"`
	Level      string `json:"level"`
	Message    string `json:"message"`
	RequestID  string `json:"request_id,omitempty"`
	ClientIP   string `json:"client_ip,omitempty"`
	Host       string `json:"host,omitempty"`
	Method     string `json:"method,omitempty"`
	Path       string `json:"path,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
	LatencyMS  int64  `json:"latency_ms,omitempty"`
	Operator   string `json:"operator,omitempty"`
	Action     string `json:"action,omitempty"`
	Detail     string `json:"detail,omitempty"`
	CreatedAt  string `json:"created_at"`
}

type SiteTrafficLogWrite struct {
	Level      string
	EventType  string
	Message    string
	RequestID  string
	ClientIP   string
	Host       string
	Method     string
	Path       string
	StatusCode int
	LatencyMS  int64
}

func (s *Store) SaveSiteTrafficLog(siteID string, entry SiteTrafficLogWrite) error {
	if s == nil || s.db == nil {
		return errors.New("db not ready")
	}
	if siteID == "" {
		return errors.New("site id is empty")
	}
	logID, err := randomID()
	if err != nil {
		return err
	}
	createdAt := time.Now().UTC().Format(time.RFC3339Nano)
	level := entry.Level
	if level == "" {
		level = "info"
	}
	eventType := entry.EventType
	if eventType == "" {
		eventType = "proxy"
	}
	_, err = s.db.Exec(
		`INSERT INTO site_traffic_logs (id, site_id, level, event_type, message, request_id, client_ip, host, method, path, status_code, latency_ms, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		logID,
		siteID,
		level,
		eventType,
		entry.Message,
		entry.RequestID,
		entry.ClientIP,
		entry.Host,
		entry.Method,
		entry.Path,
		entry.StatusCode,
		entry.LatencyMS,
		createdAt,
	)
	return err
}

func (s *Store) ListSiteLogs(siteID string, query SiteLogQuery) ([]SiteLogEntry, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("db not ready")
	}
	if siteID == "" {
		return nil, errors.New("site id is empty")
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	kind := query.Kind
	if kind == "" {
		kind = "all"
	}
	var items []SiteLogEntry
	if kind == "all" || kind == "traffic" {
		trafficItems, err := s.listSiteTrafficLogs(siteID, query, limit)
		if err != nil {
			return nil, err
		}
		items = append(items, trafficItems...)
	}
	if kind == "all" || kind == "system" {
		systemItems, err := s.listSiteSystemLogs(siteID, query, limit)
		if err != nil {
			return nil, err
		}
		items = append(items, systemItems...)
	}
	sort.Slice(items, func(i int, j int) bool {
		return items[i].CreatedAt > items[j].CreatedAt
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (s *Store) listSiteTrafficLogs(siteID string, query SiteLogQuery, limit int) ([]SiteLogEntry, error) {
	sqlText := `SELECT id, site_id, level, event_type, message, request_id, client_ip, host, method, path, status_code, latency_ms, created_at FROM site_traffic_logs WHERE site_id = ?`
	args := []any{siteID}
	if query.Level != "" {
		sqlText += ` AND level = ?`
		args = append(args, query.Level)
	}
	if query.StartTime != "" {
		sqlText += ` AND created_at >= ?`
		args = append(args, query.StartTime)
	}
	if query.EndTime != "" {
		sqlText += ` AND created_at <= ?`
		args = append(args, query.EndTime)
	}
	sqlText += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.Query(sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {

		}
	}(rows)
	var items []SiteLogEntry
	for rows.Next() {
		var item SiteLogEntry
		if err := rows.Scan(
			&item.ID,
			&item.SiteID,
			&item.Level,
			&item.Action,
			&item.Message,
			&item.RequestID,
			&item.ClientIP,
			&item.Host,
			&item.Method,
			&item.Path,
			&item.StatusCode,
			&item.LatencyMS,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.Kind = "traffic"
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Store) listSiteSystemLogs(siteID string, query SiteLogQuery, limit int) ([]SiteLogEntry, error) {
	sqlText := `SELECT id, site_id, level, action, message, operator, detail_json, created_at FROM site_system_logs WHERE site_id = ?`
	args := []any{siteID}
	if query.Level != "" {
		sqlText += ` AND level = ?`
		args = append(args, query.Level)
	}
	if query.StartTime != "" {
		sqlText += ` AND created_at >= ?`
		args = append(args, query.StartTime)
	}
	if query.EndTime != "" {
		sqlText += ` AND created_at <= ?`
		args = append(args, query.EndTime)
	}
	sqlText += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.Query(sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {

		}
	}(rows)
	var items []SiteLogEntry
	for rows.Next() {
		var item SiteLogEntry
		if err := rows.Scan(
			&item.ID,
			&item.SiteID,
			&item.Level,
			&item.Action,
			&item.Message,
			&item.Operator,
			&item.Detail,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.Kind = "system"
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
