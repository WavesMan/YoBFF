package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"YoBFF/internal/config"
)

var (
	ErrSiteNotFound       = errors.New("site not found")
	ErrSiteVersionMissing = errors.New("site version not found")
	ErrSiteLogNotFound    = errors.New("site log stream not found")
)

type Site struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Hostname  string `json:"hostname"`
	IP        string `json:"ip"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type SiteFilter struct {
	Hostname string
	IP       string
}

type SiteLogStream struct {
	SiteID      string `json:"site_id"`
	FilterQuery string `json:"filter_query"`
	UpdatedAt   string `json:"updated_at"`
}

// CreateSite 新建站点记录，用于站点级配置隔离与分类入口统一入口。
// 参数：site 为站点基础信息，必须包含 name/hostname/ip。
// 返回：写入后的站点记录。
// 异常：数据库未就绪或参数缺失时返回错误。
func (s *Store) CreateSite(site Site) (Site, error) {
	if s == nil || s.db == nil {
		return Site{}, errors.New("db not ready")
	}
	if site.Name == "" || site.Hostname == "" || site.IP == "" {
		return Site{}, errors.New("site fields missing")
	}
	siteID, err := randomID()
	if err != nil {
		return Site{}, err
	}
	currentTime := time.Now().UTC().Format(time.RFC3339)
	site.ID = siteID
	site.CreatedAt = currentTime
	site.UpdatedAt = currentTime
	_, err = s.db.Exec(
		`INSERT INTO sites (id, name, hostname, ip, config_json, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		site.ID,
		site.Name,
		site.Hostname,
		site.IP,
		"",
		site.CreatedAt,
		site.UpdatedAt,
	)
	if err != nil {
		return Site{}, err
	}
	return site, nil
}

// UpdateSite 更新站点基础信息，用于保持分类视图与站点元数据一致。
// 参数：siteID 为站点标识，payload 为待更新字段。
// 返回：更新后的站点记录。
// 异常：站点不存在或数据库异常时返回错误。
func (s *Store) UpdateSite(siteID string, payload Site) (Site, error) {
	if s == nil || s.db == nil {
		return Site{}, errors.New("db not ready")
	}
	if siteID == "" {
		return Site{}, errors.New("site id is empty")
	}
	current, err := s.GetSite(siteID)
	if err != nil {
		return Site{}, err
	}
	if payload.Name != "" {
		current.Name = payload.Name
	}
	if payload.Hostname != "" {
		current.Hostname = payload.Hostname
	}
	if payload.IP != "" {
		current.IP = payload.IP
	}
	current.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.Exec(
		`UPDATE sites SET name = ?, hostname = ?, ip = ?, updated_at = ? WHERE id = ?`,
		current.Name,
		current.Hostname,
		current.IP,
		current.UpdatedAt,
		current.ID,
	)
	if err != nil {
		return Site{}, err
	}
	return current, nil
}

// DeleteSite 删除站点及其关联记录，用于站点下线与清理历史配置。
// 参数：siteID 为站点标识。
// 返回：被删除的站点记录。
// 异常：站点不存在或数据库异常时返回错误。
func (s *Store) DeleteSite(siteID string) (Site, error) {
	if s == nil || s.db == nil {
		return Site{}, errors.New("db not ready")
	}
	if siteID == "" {
		return Site{}, errors.New("site id is empty")
	}
	site, err := s.GetSite(siteID)
	if err != nil {
		return Site{}, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return Site{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if _, err = tx.Exec(`DELETE FROM site_versions WHERE site_id = ?`, siteID); err != nil {
		return Site{}, err
	}
	if _, err = tx.Exec(`DELETE FROM site_log_streams WHERE site_id = ?`, siteID); err != nil {
		return Site{}, err
	}
	if _, err = tx.Exec(`DELETE FROM sites WHERE id = ?`, siteID); err != nil {
		return Site{}, err
	}
	if err := tx.Commit(); err != nil {
		return Site{}, err
	}
	return site, nil
}

// GetSite 查询站点详情，用于站点级管理与配置入口定位。
// 参数：siteID 为站点标识。
// 返回：站点记录。
// 异常：未找到站点或数据库异常时返回错误。
func (s *Store) GetSite(siteID string) (Site, error) {
	if s == nil || s.db == nil {
		return Site{}, errors.New("db not ready")
	}
	var site Site
	err := s.db.QueryRow(
		`SELECT id, name, hostname, ip, created_at, updated_at FROM sites WHERE id = ?`,
		siteID,
	).Scan(&site.ID, &site.Name, &site.Hostname, &site.IP, &site.CreatedAt, &site.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Site{}, ErrSiteNotFound
		}
		return Site{}, err
	}
	return site, nil
}

// ListSites 获取站点列表，用于 hostname/IP 分类视图快速呈现。
// 参数：filter 用于可选的 hostname/ip 精确过滤。
// 返回：站点列表。
// 异常：数据库异常时返回错误。
func (s *Store) ListSites(filter SiteFilter) ([]Site, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("db not ready")
	}
	query := `SELECT id, name, hostname, ip, created_at, updated_at FROM sites`
	var args []any
	if filter.Hostname != "" || filter.IP != "" {
		query += " WHERE"
	}
	if filter.Hostname != "" {
		query += " hostname = ?"
		args = append(args, filter.Hostname)
	}
	if filter.Hostname != "" && filter.IP != "" {
		query += " AND"
	}
	if filter.IP != "" {
		query += " ip = ?"
		args = append(args, filter.IP)
	}
	query += " ORDER BY updated_at DESC"
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Site
	for rows.Next() {
		var site Site
		if err := rows.Scan(&site.ID, &site.Name, &site.Hostname, &site.IP, &site.CreatedAt, &site.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, site)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// GetSiteConfig 获取站点当前配置，用于站点配置预览与差异基准。
// 参数：siteID 为站点标识。
// 返回：站点配置。
// 异常：站点不存在或配置解析失败时返回错误。
func (s *Store) GetSiteConfig(siteID string) (config.Config, error) {
	if s == nil || s.db == nil {
		return config.Config{}, errors.New("db not ready")
	}
	var payload string
	err := s.db.QueryRow(
		`SELECT config_json FROM sites WHERE id = ?`,
		siteID,
	).Scan(&payload)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return config.Config{}, ErrSiteNotFound
		}
		return config.Config{}, err
	}
	if payload == "" {
		return config.Config{}, nil
	}
	var cfg config.Config
	if err := json.Unmarshal([]byte(payload), &cfg); err != nil {
		return config.Config{}, err
	}
	return cfg, nil
}

// UpdateSiteConfig 更新站点配置并生成版本快照，用于回滚与审计追踪。
// 参数：siteID 为站点标识，cfg 为站点新配置，operator 为操作人，source 为变更来源。
// 返回：新增版本信息。
// 异常：站点不存在、持久化失败或事务异常时返回错误。
func (s *Store) UpdateSiteConfig(siteID string, cfg config.Config, operator string, source string) (ConfigVersion, error) {
	if s == nil || s.db == nil {
		return ConfigVersion{}, errors.New("db not ready")
	}
	if siteID == "" {
		return ConfigVersion{}, errors.New("site id is empty")
	}
	checkRow := s.db.QueryRow(`SELECT id FROM sites WHERE id = ?`, siteID)
	var siteKey string
	if err := checkRow.Scan(&siteKey); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ConfigVersion{}, ErrSiteNotFound
		}
		return ConfigVersion{}, err
	}
	payload, err := json.Marshal(cfg)
	if err != nil {
		return ConfigVersion{}, err
	}
	versionID, err := randomID()
	if err != nil {
		return ConfigVersion{}, err
	}
	createdAt := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.db.Begin()
	if err != nil {
		return ConfigVersion{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	_, err = tx.Exec(
		`UPDATE sites SET config_json = ?, updated_at = ? WHERE id = ?`,
		string(payload),
		createdAt,
		siteID,
	)
	if err != nil {
		return ConfigVersion{}, err
	}
	_, err = tx.Exec(
		`INSERT INTO site_versions (id, site_id, config_json, created_at, operator, source) VALUES (?, ?, ?, ?, ?, ?)`,
		versionID,
		siteID,
		string(payload),
		createdAt,
		operator,
		source,
	)
	if err != nil {
		return ConfigVersion{}, err
	}
	_, err = tx.Exec(
		`DELETE FROM site_versions WHERE site_id = ? AND id NOT IN (
			SELECT id FROM site_versions WHERE site_id = ? ORDER BY created_at DESC LIMIT 10
		)`,
		siteID,
		siteID,
	)
	if err != nil {
		return ConfigVersion{}, err
	}
	if err := tx.Commit(); err != nil {
		return ConfigVersion{}, err
	}
	return ConfigVersion{
		ID:        versionID,
		CreatedAt: createdAt,
		Operator:  operator,
		Source:    source,
	}, nil
}

// ListSiteVersions 查询站点配置版本列表，用于展示最近版本与回滚入口。
// 参数：siteID 为站点标识，limit 为数量上限。
// 返回：版本列表。
// 异常：站点不存在或数据库异常时返回错误。
func (s *Store) ListSiteVersions(siteID string, limit int) ([]ConfigVersion, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("db not ready")
	}
	if siteID == "" {
		return nil, errors.New("site id is empty")
	}
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.db.Query(
		`SELECT id, created_at, operator, source FROM site_versions WHERE site_id = ? ORDER BY created_at DESC LIMIT ?`,
		siteID,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var versions []ConfigVersion
	for rows.Next() {
		var item ConfigVersion
		if err := rows.Scan(&item.ID, &item.CreatedAt, &item.Operator, &item.Source); err != nil {
			return nil, err
		}
		versions = append(versions, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return versions, nil
}

// GetSiteVersionConfig 获取站点指定版本配置，用于版本预览。
// 参数：siteID 为站点标识，versionID 为版本标识。
// 返回：版本配置。
// 异常：版本不存在或解析失败时返回错误。
func (s *Store) GetSiteVersionConfig(siteID string, versionID string) (config.Config, error) {
	if s == nil || s.db == nil {
		return config.Config{}, errors.New("db not ready")
	}
	var payload string
	err := s.db.QueryRow(
		`SELECT config_json FROM site_versions WHERE site_id = ? AND id = ?`,
		siteID,
		versionID,
	).Scan(&payload)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return config.Config{}, ErrSiteVersionMissing
		}
		return config.Config{}, err
	}
	var cfg config.Config
	if err := json.Unmarshal([]byte(payload), &cfg); err != nil {
		return config.Config{}, err
	}
	return cfg, nil
}

// RollbackSiteVersion 回滚站点配置版本并生成新的版本快照，用于审计与回退链路闭环。
// 参数：siteID 为站点标识，versionID 为目标版本，operator 为操作人。
// 返回：回滚生成的新版本信息。
// 异常：目标版本不存在或保存失败时返回错误。
func (s *Store) RollbackSiteVersion(siteID string, versionID string, operator string) (ConfigVersion, error) {
	cfg, err := s.GetSiteVersionConfig(siteID, versionID)
	if err != nil {
		return ConfigVersion{}, err
	}
	return s.UpdateSiteConfig(siteID, cfg, operator, "rollback")
}

// GetSiteLogStream 获取站点日志流探查配置，用于排查与观测入口呈现。
// 参数：siteID 为站点标识。
// 返回：日志流配置。
// 异常：站点日志流未配置或数据库异常时返回错误。
func (s *Store) GetSiteLogStream(siteID string) (SiteLogStream, error) {
	if s == nil || s.db == nil {
		return SiteLogStream{}, errors.New("db not ready")
	}
	var stream SiteLogStream
	err := s.db.QueryRow(
		`SELECT site_id, filter_query, updated_at FROM site_log_streams WHERE site_id = ?`,
		siteID,
	).Scan(&stream.SiteID, &stream.FilterQuery, &stream.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return SiteLogStream{}, ErrSiteLogNotFound
		}
		return SiteLogStream{}, err
	}
	return stream, nil
}

// UpdateSiteLogStream 更新站点日志流探查配置，用于保留站点专属观测条件。
// 参数：siteID 为站点标识，filterQuery 为过滤条件。
// 返回：更新后的日志流配置。
// 异常：参数缺失或数据库异常时返回错误。
func (s *Store) UpdateSiteLogStream(siteID string, filterQuery string) (SiteLogStream, error) {
	if s == nil || s.db == nil {
		return SiteLogStream{}, errors.New("db not ready")
	}
	if siteID == "" {
		return SiteLogStream{}, errors.New("site id is empty")
	}
	if filterQuery == "" {
		return SiteLogStream{}, errors.New("filter query is empty")
	}
	updatedAt := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`INSERT INTO site_log_streams (site_id, filter_query, updated_at) VALUES (?, ?, ?)
		 ON CONFLICT(site_id) DO UPDATE SET filter_query = excluded.filter_query, updated_at = excluded.updated_at`,
		siteID,
		filterQuery,
		updatedAt,
	)
	if err != nil {
		return SiteLogStream{}, err
	}
	return SiteLogStream{
		SiteID:      siteID,
		FilterQuery: filterQuery,
		UpdatedAt:   updatedAt,
	}, nil
}
