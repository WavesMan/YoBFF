package store

import (
	"database/sql"
	"errors"
	"time"
)

type AuditLogQuery struct {
	Action    string
	Operator  string
	Target    string
	StartTime string
	EndTime   string
	Limit     int
	Page      int
	PageSize  int
}

type AuditLogEntry struct {
	ID        string `json:"id"`
	Action    string `json:"action"`
	Target    string `json:"target"`
	CreatedAt string `json:"created_at"`
	Operator  string `json:"operator"`
	Detail    string `json:"detail"`
}

// ListAuditLogs 按条件查询审计日志，用于管理台筛选操作轨迹。
// 参数：query 为筛选条件与分页参数。
// 返回：审计日志列表与总记录数。
// 异常：数据库不可用或查询失败时返回错误。
func (s *Store) ListAuditLogs(query AuditLogQuery) ([]AuditLogEntry, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, errors.New("db not ready")
	}
	page := query.Page
	if page <= 0 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = query.Limit
	}
	if pageSize <= 0 {
		pageSize = 15
	}
	if pageSize > 500 {
		pageSize = 500
	}

	offset := (page - 1) * pageSize
	filterSQL := ` FROM audit_logs WHERE 1=1`
	args := make([]any, 0, 6)

	if query.Action != "" {
		filterSQL += ` AND action = ?`
		args = append(args, query.Action)
	}
	if query.Operator != "" {
		filterSQL += ` AND operator = ?`
		args = append(args, query.Operator)
	}
	if query.Target != "" {
		filterSQL += ` AND target = ?`
		args = append(args, query.Target)
	}
	if query.StartTime != "" {
		filterSQL += ` AND created_at >= ?`
		args = append(args, query.StartTime)
	}
	if query.EndTime != "" {
		filterSQL += ` AND created_at <= ?`
		args = append(args, query.EndTime)
	}

	var total int
	countSQL := `SELECT COUNT(*)` + filterSQL
	if err := s.db.QueryRow(countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	sqlText := `SELECT id, action, target, created_at, operator, detail_json FROM audit_logs WHERE 1=1`
	sqlText = `SELECT id, action, target, created_at, operator, detail_json` + filterSQL + ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	listArgs := append(append(make([]any, 0, len(args)+2), args...), pageSize, offset)
	rows, err := s.db.Query(sqlText, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)
	var items []AuditLogEntry
	for rows.Next() {
		var item AuditLogEntry
		if err := rows.Scan(
			&item.ID,
			&item.Action,
			&item.Target,
			&item.CreatedAt,
			&item.Operator,
			&item.Detail,
		); err != nil {
			return nil, 0, err
		}
		if item.CreatedAt == "" {
			item.CreatedAt = time.Now().UTC().Format(time.RFC3339)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
