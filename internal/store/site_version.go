package store

import (
	"database/sql"
	"errors"
)

func (s *Store) DeleteSiteVersion(siteID string, versionID string) error {
	if s == nil || s.db == nil {
		return errors.New("db not ready")
	}
	if siteID == "" {
		return errors.New("site id is empty")
	}
	if versionID == "" {
		return errors.New("version id is empty")
	}
	result, err := s.db.Exec(
		`DELETE FROM site_versions WHERE site_id = ? AND id = ?`,
		siteID,
		versionID,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrSiteVersionMissing
	}
	return nil
}

func (s *Store) GetSiteIDByHostname(hostname string) (string, error) {
	if s == nil || s.db == nil {
		return "", errors.New("db not ready")
	}
	value := normalizeHostname(hostname)
	if value == "" {
		return "", ErrSiteNotFound
	}
	var siteID string
	err := s.db.QueryRow(`SELECT id FROM sites WHERE lower(hostname) = ? LIMIT 1`, value).Scan(&siteID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrSiteNotFound
		}
		return "", err
	}
	return siteID, nil
}
