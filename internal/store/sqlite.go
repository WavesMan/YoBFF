package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"YoBFF/internal/config"
)

type Store struct {
	db *sql.DB
}

type ConfigVersion struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	Operator  string `json:"operator"`
	Source    string `json:"source"`
}

// NewSQLiteStore 初始化 SQLite 存储并创建必要数据表。
// 参数：path 为数据库文件路径。
// 返回：Store 实例。
// 异常：路径非法、数据库打开或初始化失败时返回错误。
func NewSQLiteStore(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("db path is empty")
	}
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return nil, err
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	store := &Store{db: db}
	if err := store.init(); err != nil {
		return nil, err
	}
	return store, nil
}

// init 初始化存储层表结构与索引。
// 参数：无。
// 返回：初始化错误信息。
// 异常：数据库不可用时返回错误。
func (s *Store) init() error {
	if s == nil || s.db == nil {
		return errors.New("db not ready")
	}
	_, err := s.db.Exec(`
		PRAGMA journal_mode=WAL;
		CREATE TABLE IF NOT EXISTS config_versions (
			id TEXT PRIMARY KEY,
			config_json TEXT NOT NULL,
			created_at TEXT NOT NULL,
			operator TEXT,
			source TEXT
		);
		CREATE TABLE IF NOT EXISTS audit_logs (
			id TEXT PRIMARY KEY,
			action TEXT NOT NULL,
			target TEXT,
			created_at TEXT NOT NULL,
			operator TEXT,
			detail_json TEXT
		);
		CREATE TABLE IF NOT EXISTS sites (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			hostname TEXT NOT NULL,
			ip TEXT NOT NULL,
			config_json TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS site_versions (
			id TEXT PRIMARY KEY,
			site_id TEXT NOT NULL,
			config_json TEXT NOT NULL,
			created_at TEXT NOT NULL,
			operator TEXT,
			source TEXT
		);
		CREATE TABLE IF NOT EXISTS site_log_streams (
			site_id TEXT PRIMARY KEY,
			filter_query TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_config_versions_created_at ON config_versions(created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_sites_hostname ON sites(hostname);
		CREATE INDEX IF NOT EXISTS idx_sites_ip ON sites(ip);
		CREATE INDEX IF NOT EXISTS idx_site_versions_site_id ON site_versions(site_id);
		CREATE INDEX IF NOT EXISTS idx_site_versions_created_at ON site_versions(created_at DESC);
	`)
	return err
}

// Close 关闭数据库连接。
func (s *Store) Close() error {
	if s != nil && s.db != nil {
		return s.db.Close()
	}
	return nil
}

// SaveVersion 写入全局配置版本，用于配置变更回溯。
// 参数：cfg 为配置对象，operator 为操作人，source 为变更来源。
// 返回：版本记录。
// 异常：数据库不可用或写入失败时返回错误。
func (s *Store) SaveVersion(cfg config.Config, operator string, source string) (ConfigVersion, error) {
	if s == nil || s.db == nil {
		return ConfigVersion{}, errors.New("db not ready")
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
	_, err = s.db.Exec(
		`INSERT INTO config_versions (id, config_json, created_at, operator, source) VALUES (?, ?, ?, ?, ?)`,
		versionID,
		string(payload),
		createdAt,
		operator,
		source,
	)
	if err != nil {
		return ConfigVersion{}, err
	}
	return ConfigVersion{
		ID:        versionID,
		CreatedAt: createdAt,
		Operator:  operator,
		Source:    source,
	}, nil
}

// ListVersions 查询全局配置版本列表，按时间倒序返回。
// 参数：limit 为返回数量上限。
// 返回：版本列表。
// 异常：数据库不可用或查询失败时返回错误。
func (s *Store) ListVersions(limit int) ([]ConfigVersion, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("db not ready")
	}
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Query(
		`SELECT id, created_at, operator, source FROM config_versions ORDER BY created_at DESC LIMIT ?`,
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

// GetVersionConfig 获取指定版本的完整配置。
// 参数：versionID 为版本标识。
// 返回：配置对象。
// 异常：版本不存在或解析失败时返回错误。
func (s *Store) GetVersionConfig(versionID string) (config.Config, error) {
	if s == nil || s.db == nil {
		return config.Config{}, errors.New("db not ready")
	}
	var payload string
	err := s.db.QueryRow(
		`SELECT config_json FROM config_versions WHERE id = ?`,
		versionID,
	).Scan(&payload)
	if err != nil {
		return config.Config{}, err
	}
	var cfg config.Config
	if err := json.Unmarshal([]byte(payload), &cfg); err != nil {
		return config.Config{}, err
	}
	return cfg, nil
}

// SaveAudit 写入审计日志，用于记录管理端操作轨迹。
// 参数：action 为动作类型，target 为目标标识，operator 为操作人，detail 为附加信息。
// 返回：错误信息。
// 异常：数据库不可用或写入失败时返回错误。
func (s *Store) SaveAudit(action string, target string, operator string, detail any) error {
	if s == nil || s.db == nil {
		return errors.New("db not ready")
	}
	auditID, err := randomID()
	if err != nil {
		return err
	}
	createdAt := time.Now().UTC().Format(time.RFC3339)
	detailText := ""
	if detail != nil {
		var payload []byte
		payload, err = json.Marshal(detail)
		if err != nil {
			return err
		}
		detailText = string(payload)
	}
	_, err = s.db.Exec(
		`INSERT INTO audit_logs (id, action, target, created_at, operator, detail_json) VALUES (?, ?, ?, ?, ?, ?)`,
		auditID,
		action,
		target,
		createdAt,
		operator,
		detailText,
	)
	return err
}

// randomID 生成随机标识，用于数据库主键。
// 参数：无。
// 返回：随机字符串。
// 异常：随机源失败时返回错误。
func randomID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}
