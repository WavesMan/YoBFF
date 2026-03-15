package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

var ErrWeaverDraftNotFound = errors.New("weaver draft not found")

type WeaverDraft struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Inputs    json.RawMessage `json:"inputs"`
	Mapping   json.RawMessage `json:"mapping"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
	Operator  string          `json:"operator,omitempty"`
}

// CreateWeaverDraft 新建可视化实验草稿，用于管理台实验区保存当前配置。
// 参数：draft 为草稿信息，必须包含 name、inputs、mapping。
// 返回：保存后的草稿记录。
// 异常：数据库不可用或字段缺失时返回错误。
func (s *Store) CreateWeaverDraft(draft WeaverDraft) (WeaverDraft, error) {
	if s == nil || s.db == nil {
		return WeaverDraft{}, errors.New("db not ready")
	}
	if draft.Name == "" {
		return WeaverDraft{}, errors.New("draft name is empty")
	}
	if len(draft.Inputs) == 0 || len(draft.Mapping) == 0 {
		return WeaverDraft{}, errors.New("draft content is empty")
	}
	draftID, err := randomID()
	if err != nil {
		return WeaverDraft{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	draft.ID = draftID
	draft.CreatedAt = now
	draft.UpdatedAt = now
	_, err = s.db.Exec(
		`INSERT INTO weaver_drafts (id, name, inputs_json, mapping_json, created_at, updated_at, operator) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		draft.ID,
		draft.Name,
		string(draft.Inputs),
		string(draft.Mapping),
		draft.CreatedAt,
		draft.UpdatedAt,
		draft.Operator,
	)
	if err != nil {
		return WeaverDraft{}, err
	}
	return draft, nil
}

// UpdateWeaverDraft 更新可视化实验草稿，用于保存配置变更与版本演示。
// 参数：draft 为草稿信息，必须包含 id。
// 返回：更新后的草稿记录。
// 异常：草稿不存在或写入失败时返回错误。
func (s *Store) UpdateWeaverDraft(draft WeaverDraft) (WeaverDraft, error) {
	if s == nil || s.db == nil {
		return WeaverDraft{}, errors.New("db not ready")
	}
	if draft.ID == "" {
		return WeaverDraft{}, errors.New("draft id is empty")
	}
	current, err := s.GetWeaverDraft(draft.ID)
	if err != nil {
		return WeaverDraft{}, err
	}
	if draft.Name != "" {
		current.Name = draft.Name
	}
	if len(draft.Inputs) > 0 {
		current.Inputs = draft.Inputs
	}
	if len(draft.Mapping) > 0 {
		current.Mapping = draft.Mapping
	}
	if draft.Operator != "" {
		current.Operator = draft.Operator
	}
	current.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.Exec(
		`UPDATE weaver_drafts SET name = ?, inputs_json = ?, mapping_json = ?, updated_at = ?, operator = ? WHERE id = ?`,
		current.Name,
		string(current.Inputs),
		string(current.Mapping),
		current.UpdatedAt,
		current.Operator,
		current.ID,
	)
	if err != nil {
		return WeaverDraft{}, err
	}
	return current, nil
}

// GetWeaverDraft 查询指定草稿详情，用于实验配置回放与编辑。
// 参数：draftID 为草稿标识。
// 返回：草稿详情。
// 异常：草稿不存在或查询失败时返回错误。
func (s *Store) GetWeaverDraft(draftID string) (WeaverDraft, error) {
	if s == nil || s.db == nil {
		return WeaverDraft{}, errors.New("db not ready")
	}
	if draftID == "" {
		return WeaverDraft{}, errors.New("draft id is empty")
	}
	var item WeaverDraft
	var inputsText string
	var mappingText string
	err := s.db.QueryRow(
		`SELECT id, name, inputs_json, mapping_json, created_at, updated_at, operator FROM weaver_drafts WHERE id = ?`,
		draftID,
	).Scan(
		&item.ID,
		&item.Name,
		&inputsText,
		&mappingText,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.Operator,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return WeaverDraft{}, ErrWeaverDraftNotFound
	}
	if err != nil {
		return WeaverDraft{}, err
	}
	item.Inputs = json.RawMessage(inputsText)
	item.Mapping = json.RawMessage(mappingText)
	return item, nil
}

// ListWeaverDrafts 分页读取草稿列表，用于管理台展示历史实验记录。
// 参数：limit 为返回条数限制。
// 返回：草稿列表。
// 异常：数据库不可用或查询失败时返回错误。
func (s *Store) ListWeaverDrafts(limit int) ([]WeaverDraft, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("db not ready")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := s.db.Query(
		`SELECT id, name, inputs_json, mapping_json, created_at, updated_at, operator FROM weaver_drafts ORDER BY created_at DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)
	var items []WeaverDraft
	for rows.Next() {
		var item WeaverDraft
		var inputsText string
		var mappingText string
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&inputsText,
			&mappingText,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.Operator,
		); err != nil {
			return nil, err
		}
		item.Inputs = json.RawMessage(inputsText)
		item.Mapping = json.RawMessage(mappingText)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
