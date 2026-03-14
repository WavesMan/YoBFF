package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"
)

type SiteCDNOriginSnapshot struct {
	ID        string   `json:"id"`
	SiteID    string   `json:"site_id"`
	Provider  string   `json:"provider"`
	CIDRs     []string `json:"cidrs"`
	FetchedAt string   `json:"fetched_at"`
	Source    string   `json:"source"`
	CreatedAt string   `json:"created_at"`
}

type SiteCDNOriginStatus struct {
	SiteID              string `json:"site_id"`
	Provider            string `json:"provider"`
	LastAttemptAt       string `json:"last_attempt_at"`
	LastSuccessAt       string `json:"last_success_at"`
	ConsecutiveFailures int    `json:"consecutive_failures"`
	LastError           string `json:"last_error"`
	UpdatedAt           string `json:"updated_at"`
}

type SiteCDNOriginStatusUpdate struct {
	LastAttemptAt       *time.Time
	LastSuccessAt       *time.Time
	ConsecutiveFailures int
	LastError           string
}

// SaveSiteCDNOriginSnapshot 写入站点维度的 CDN 回源 CIDR 快照。
// 参数：siteID 为站点标识，provider 为厂商标识，cidrs 为回源 CIDR 列表，fetchedAt 为获取成功时间，source 为来源标识。
// 返回：快照记录。
// 异常：数据库未就绪或写入失败时返回错误。
func (s *Store) SaveSiteCDNOriginSnapshot(siteID string, provider string, cidrs []string, fetchedAt time.Time, source string) (SiteCDNOriginSnapshot, error) {
	if s == nil || s.db == nil {
		return SiteCDNOriginSnapshot{}, errors.New("db not ready")
	}
	siteID = strings.TrimSpace(siteID)
	provider = strings.ToLower(strings.TrimSpace(provider))
	if siteID == "" || provider == "" {
		return SiteCDNOriginSnapshot{}, errors.New("site id or provider is empty")
	}

	payload, err := json.Marshal(cidrs)
	if err != nil {
		return SiteCDNOriginSnapshot{}, err
	}

	snapshotID, err := randomID()
	if err != nil {
		return SiteCDNOriginSnapshot{}, err
	}

	createdAt := time.Now().UTC().Format(time.RFC3339)
	fetchedAtText := fetchedAt.UTC().Format(time.RFC3339)
	_, err = s.db.Exec(
		`INSERT INTO site_cdn_origin_snapshots (id, site_id, provider, cidrs_json, fetched_at, source, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		snapshotID,
		siteID,
		provider,
		string(payload),
		fetchedAtText,
		source,
		createdAt,
	)
	if err != nil {
		return SiteCDNOriginSnapshot{}, err
	}

	return SiteCDNOriginSnapshot{
		ID:        snapshotID,
		SiteID:    siteID,
		Provider:  provider,
		CIDRs:     cidrs,
		FetchedAt: fetchedAtText,
		Source:    source,
		CreatedAt: createdAt,
	}, nil
}

// GetLatestSiteCDNOriginSnapshot 获取站点指定厂商最新的 CDN 回源 CIDR 快照。
// 参数：siteID 为站点标识，provider 为厂商标识。
// 返回：最新快照记录。
// 异常：无记录时返回 sql.ErrNoRows，数据库异常时返回对应错误。
func (s *Store) GetLatestSiteCDNOriginSnapshot(siteID string, provider string) (SiteCDNOriginSnapshot, error) {
	if s == nil || s.db == nil {
		return SiteCDNOriginSnapshot{}, errors.New("db not ready")
	}
	siteID = strings.TrimSpace(siteID)
	provider = strings.ToLower(strings.TrimSpace(provider))
	if siteID == "" || provider == "" {
		return SiteCDNOriginSnapshot{}, errors.New("site id or provider is empty")
	}

	var item SiteCDNOriginSnapshot
	var cidrsJSON string
	err := s.db.QueryRow(
		`SELECT id, site_id, provider, cidrs_json, fetched_at, source, created_at FROM site_cdn_origin_snapshots WHERE site_id = ? AND provider = ? ORDER BY fetched_at DESC LIMIT 1`,
		siteID,
		provider,
	).Scan(&item.ID, &item.SiteID, &item.Provider, &cidrsJSON, &item.FetchedAt, &item.Source, &item.CreatedAt)
	if err != nil {
		return SiteCDNOriginSnapshot{}, err
	}
	if cidrsJSON != "" {
		if err := json.Unmarshal([]byte(cidrsJSON), &item.CIDRs); err != nil {
			return SiteCDNOriginSnapshot{}, err
		}
	}
	return item, nil
}

// UpsertSiteCDNOriginStatus 写入或更新站点维度的 CDN 回源同步状态。
// 参数：siteID 为站点标识，provider 为厂商标识，update 为状态更新内容。
// 返回：更新后的状态记录。
// 异常：数据库未就绪或写入失败时返回错误。
func (s *Store) UpsertSiteCDNOriginStatus(siteID string, provider string, update SiteCDNOriginStatusUpdate) (SiteCDNOriginStatus, error) {
	if s == nil || s.db == nil {
		return SiteCDNOriginStatus{}, errors.New("db not ready")
	}
	siteID = strings.TrimSpace(siteID)
	provider = strings.ToLower(strings.TrimSpace(provider))
	if siteID == "" || provider == "" {
		return SiteCDNOriginStatus{}, errors.New("site id or provider is empty")
	}

	now := time.Now().UTC()
	updatedAt := now.Format(time.RFC3339)
	lastAttemptAt := ""
	lastSuccessAt := ""
	if update.LastAttemptAt != nil {
		lastAttemptAt = update.LastAttemptAt.UTC().Format(time.RFC3339)
	}
	if update.LastSuccessAt != nil {
		lastSuccessAt = update.LastSuccessAt.UTC().Format(time.RFC3339)
	}

	_, err := s.db.Exec(
		`INSERT INTO site_cdn_origin_status (site_id, provider, last_attempt_at, last_success_at, consecutive_failures, last_error, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(site_id, provider) DO UPDATE SET
		 last_attempt_at = excluded.last_attempt_at,
		 last_success_at = CASE WHEN excluded.last_success_at != '' THEN excluded.last_success_at ELSE site_cdn_origin_status.last_success_at END,
		 consecutive_failures = excluded.consecutive_failures,
		 last_error = excluded.last_error,
		 updated_at = excluded.updated_at`,
		siteID,
		provider,
		lastAttemptAt,
		lastSuccessAt,
		update.ConsecutiveFailures,
		update.LastError,
		updatedAt,
	)
	if err != nil {
		return SiteCDNOriginStatus{}, err
	}

	return s.GetSiteCDNOriginStatus(siteID, provider)
}

// GetSiteCDNOriginStatus 获取站点指定厂商的 CDN 回源同步状态。
// 参数：siteID 为站点标识，provider 为厂商标识。
// 返回：状态记录。
// 异常：无记录时返回 sql.ErrNoRows，数据库异常时返回对应错误。
func (s *Store) GetSiteCDNOriginStatus(siteID string, provider string) (SiteCDNOriginStatus, error) {
	if s == nil || s.db == nil {
		return SiteCDNOriginStatus{}, errors.New("db not ready")
	}
	siteID = strings.TrimSpace(siteID)
	provider = strings.ToLower(strings.TrimSpace(provider))
	if siteID == "" || provider == "" {
		return SiteCDNOriginStatus{}, errors.New("site id or provider is empty")
	}

	var item SiteCDNOriginStatus
	err := s.db.QueryRow(
		`SELECT site_id, provider, COALESCE(last_attempt_at, ''), COALESCE(last_success_at, ''), consecutive_failures, COALESCE(last_error, ''), updated_at FROM site_cdn_origin_status WHERE site_id = ? AND provider = ?`,
		siteID,
		provider,
	).Scan(
		&item.SiteID,
		&item.Provider,
		&item.LastAttemptAt,
		&item.LastSuccessAt,
		&item.ConsecutiveFailures,
		&item.LastError,
		&item.UpdatedAt,
	)
	if err != nil {
		return SiteCDNOriginStatus{}, err
	}
	return item, nil
}

// ListSiteCDNOriginStatus 获取站点全部厂商的 CDN 回源同步状态。
// 参数：siteID 为站点标识。
// 返回：状态列表。
// 异常：数据库未就绪或查询失败时返回错误。
func (s *Store) ListSiteCDNOriginStatus(siteID string) ([]SiteCDNOriginStatus, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("db not ready")
	}
	siteID = strings.TrimSpace(siteID)
	if siteID == "" {
		return nil, errors.New("site id is empty")
	}

	rows, err := s.db.Query(
		`SELECT site_id, provider, COALESCE(last_attempt_at, ''), COALESCE(last_success_at, ''), consecutive_failures, COALESCE(last_error, ''), updated_at
		 FROM site_cdn_origin_status WHERE site_id = ? ORDER BY provider ASC`,
		siteID,
	)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Printf("failed to close rows: %v", err)
		}
	}(rows)

	var items []SiteCDNOriginStatus
	for rows.Next() {
		var item SiteCDNOriginStatus
		if err := rows.Scan(
			&item.SiteID,
			&item.Provider,
			&item.LastAttemptAt,
			&item.LastSuccessAt,
			&item.ConsecutiveFailures,
			&item.LastError,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
