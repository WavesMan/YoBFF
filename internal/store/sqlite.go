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
		CREATE INDEX IF NOT EXISTS idx_config_versions_created_at ON config_versions(created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);
	`)
	return err
}

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
		payload, err := json.Marshal(detail)
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

func randomID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}
