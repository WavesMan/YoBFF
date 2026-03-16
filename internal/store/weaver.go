package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var ErrWeaverDraftNotFound = errors.New("weaver draft not found")
var ErrWeaverVersionNotFound = errors.New("weaver version not found")

type WeaverDraft struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Inputs    json.RawMessage `json:"inputs"`
	Mapping   json.RawMessage `json:"mapping"`
	DAG       json.RawMessage `json:"dag"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
	Operator  string          `json:"operator,omitempty"`
}

type WeaverNode struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Inputs  []string        `json:"inputs"`
	Outputs []string        `json:"outputs"`
	Config  json.RawMessage `json:"config,omitempty"`
}

type WeaverEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type WeaverDAG struct {
	Nodes        []WeaverNode `json:"nodes"`
	Edges        []WeaverEdge `json:"edges"`
	OutputNodeID string       `json:"output_node_id,omitempty"`
}

type WeaverNodeContract struct {
	NodeID  string   `json:"node_id"`
	Type    string   `json:"type"`
	Inputs  []string `json:"inputs"`
	Outputs []string `json:"outputs"`
}

type WeaverVersion struct {
	ID            string          `json:"id"`
	DraftID       string          `json:"draft_id"`
	Version       int             `json:"version"`
	Name          string          `json:"name"`
	Inputs        json.RawMessage `json:"inputs"`
	Mapping       json.RawMessage `json:"mapping"`
	DAG           json.RawMessage `json:"dag"`
	NodeContracts json.RawMessage `json:"node_contracts"`
	CreatedAt     string          `json:"created_at"`
	Operator      string          `json:"operator,omitempty"`
	Source        string          `json:"source,omitempty"`
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
	draft.DAG = normalizeJSONText(draft.DAG, `{}`)
	_, err = s.db.Exec(
		`INSERT INTO weaver_drafts (id, name, inputs_json, mapping_json, dag_json, created_at, updated_at, operator) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		draft.ID,
		draft.Name,
		string(draft.Inputs),
		string(draft.Mapping),
		string(draft.DAG),
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
	if len(draft.DAG) > 0 {
		current.DAG = normalizeJSONText(draft.DAG, `{}`)
	}
	if draft.Operator != "" {
		current.Operator = draft.Operator
	}
	current.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.Exec(
		`UPDATE weaver_drafts SET name = ?, inputs_json = ?, mapping_json = ?, dag_json = ?, updated_at = ?, operator = ? WHERE id = ?`,
		current.Name,
		string(current.Inputs),
		string(current.Mapping),
		string(current.DAG),
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
	var dagText string
	err := s.db.QueryRow(
		`SELECT id, name, inputs_json, mapping_json, dag_json, created_at, updated_at, operator FROM weaver_drafts WHERE id = ?`,
		draftID,
	).Scan(
		&item.ID,
		&item.Name,
		&inputsText,
		&mappingText,
		&dagText,
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
	item.DAG = json.RawMessage(normalizeJSONStringValue(dagText, `{}`))
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
		`SELECT id, name, inputs_json, mapping_json, dag_json, created_at, updated_at, operator FROM weaver_drafts ORDER BY created_at DESC LIMIT ?`,
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
		var dagText string
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&inputsText,
			&mappingText,
			&dagText,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.Operator,
		); err != nil {
			return nil, err
		}
		item.Inputs = json.RawMessage(inputsText)
		item.Mapping = json.RawMessage(mappingText)
		item.DAG = json.RawMessage(normalizeJSONStringValue(dagText, `{}`))
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// DeleteWeaverDraft 删除指定草稿，用于清理无效实验配置。
// 参数：draftID 为草稿标识。
// 返回：无。
// 异常：草稿不存在或数据库操作失败时返回错误。
func (s *Store) DeleteWeaverDraft(draftID string) error {
	if s == nil || s.db == nil {
		return errors.New("db not ready")
	}
	if draftID == "" {
		return errors.New("draft id is empty")
	}
	result, err := s.db.Exec(`DELETE FROM weaver_drafts WHERE id = ?`, draftID)
	if err != nil {
		return err
	}
	affectedRows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affectedRows == 0 {
		return ErrWeaverDraftNotFound
	}
	return nil
}

// CreateWeaverVersion 基于草稿创建流程版本快照，并冻结节点接口契约。
// 参数：draftID 为草稿标识，operator 为操作人，source 为发布来源。
// 返回：发布后的流程版本信息。
// 异常：草稿不存在、DAG 不合法或持久化失败时返回错误。
func (s *Store) CreateWeaverVersion(draftID string, operator string, source string) (WeaverVersion, error) {
	if s == nil || s.db == nil {
		return WeaverVersion{}, errors.New("db not ready")
	}
	draftID = strings.TrimSpace(draftID)
	if draftID == "" {
		return WeaverVersion{}, errors.New("draft id is empty")
	}
	draft, err := s.GetWeaverDraft(draftID)
	if err != nil {
		return WeaverVersion{}, err
	}
	dagModel, err := parseWeaverDAG(draft.DAG)
	if err != nil {
		return WeaverVersion{}, err
	}
	contracts, err := freezeWeaverNodeContracts(dagModel)
	if err != nil {
		return WeaverVersion{}, err
	}
	contractsPayload, err := json.Marshal(contracts)
	if err != nil {
		return WeaverVersion{}, err
	}
	versionID, err := randomID()
	if err != nil {
		return WeaverVersion{}, err
	}
	createdAt := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.db.Begin()
	if err != nil {
		return WeaverVersion{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	var versionNo int
	if err = tx.QueryRow(
		`SELECT COALESCE(MAX(version_no), 0) + 1 FROM weaver_versions WHERE draft_id = ?`,
		draftID,
	).Scan(&versionNo); err != nil {
		return WeaverVersion{}, err
	}
	_, err = tx.Exec(
		`INSERT INTO weaver_versions (id, draft_id, version_no, name, inputs_json, mapping_json, dag_json, node_contracts_json, created_at, operator, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		versionID,
		draftID,
		versionNo,
		draft.Name,
		string(normalizeJSONText(draft.Inputs, `[]`)),
		string(normalizeJSONText(draft.Mapping, `{}`)),
		string(normalizeJSONText(draft.DAG, `{}`)),
		string(contractsPayload),
		createdAt,
		operator,
		source,
	)
	if err != nil {
		return WeaverVersion{}, err
	}
	if err = tx.Commit(); err != nil {
		return WeaverVersion{}, err
	}
	return WeaverVersion{
		ID:            versionID,
		DraftID:       draftID,
		Version:       versionNo,
		Name:          draft.Name,
		Inputs:        normalizeJSONText(draft.Inputs, `[]`),
		Mapping:       normalizeJSONText(draft.Mapping, `{}`),
		DAG:           normalizeJSONText(draft.DAG, `{}`),
		NodeContracts: contractsPayload,
		CreatedAt:     createdAt,
		Operator:      operator,
		Source:        source,
	}, nil
}

// GetWeaverVersion 查询指定流程版本详情。
// 参数：versionID 为流程版本标识。
// 返回：流程版本快照。
// 异常：版本不存在或查询失败时返回错误。
func (s *Store) GetWeaverVersion(versionID string) (WeaverVersion, error) {
	if s == nil || s.db == nil {
		return WeaverVersion{}, errors.New("db not ready")
	}
	versionID = strings.TrimSpace(versionID)
	if versionID == "" {
		return WeaverVersion{}, errors.New("version id is empty")
	}
	var item WeaverVersion
	var inputsText string
	var mappingText string
	var dagText string
	var contractsText string
	err := s.db.QueryRow(
		`SELECT id, draft_id, version_no, name, inputs_json, mapping_json, dag_json, node_contracts_json, created_at, operator, source
		FROM weaver_versions WHERE id = ?`,
		versionID,
	).Scan(
		&item.ID,
		&item.DraftID,
		&item.Version,
		&item.Name,
		&inputsText,
		&mappingText,
		&dagText,
		&contractsText,
		&item.CreatedAt,
		&item.Operator,
		&item.Source,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return WeaverVersion{}, ErrWeaverVersionNotFound
	}
	if err != nil {
		return WeaverVersion{}, err
	}
	item.Inputs = json.RawMessage(normalizeJSONStringValue(inputsText, `[]`))
	item.Mapping = json.RawMessage(normalizeJSONStringValue(mappingText, `{}`))
	item.DAG = json.RawMessage(normalizeJSONStringValue(dagText, `{}`))
	item.NodeContracts = json.RawMessage(normalizeJSONStringValue(contractsText, `[]`))
	return item, nil
}

// ListWeaverVersionsByDraft 查询草稿下的发布版本列表。
// 参数：draftID 为草稿标识，limit 为返回数量上限。
// 返回：流程版本列表。
// 异常：参数非法或查询失败时返回错误。
func (s *Store) ListWeaverVersionsByDraft(draftID string, limit int) ([]WeaverVersion, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("db not ready")
	}
	draftID = strings.TrimSpace(draftID)
	if draftID == "" {
		return nil, errors.New("draft id is empty")
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := s.db.Query(
		`SELECT id, draft_id, version_no, name, inputs_json, mapping_json, dag_json, node_contracts_json, created_at, operator, source
		FROM weaver_versions WHERE draft_id = ? ORDER BY version_no DESC LIMIT ?`,
		draftID,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)
	items := make([]WeaverVersion, 0)
	for rows.Next() {
		var item WeaverVersion
		var inputsText string
		var mappingText string
		var dagText string
		var contractsText string
		if err := rows.Scan(
			&item.ID,
			&item.DraftID,
			&item.Version,
			&item.Name,
			&inputsText,
			&mappingText,
			&dagText,
			&contractsText,
			&item.CreatedAt,
			&item.Operator,
			&item.Source,
		); err != nil {
			return nil, err
		}
		item.Inputs = json.RawMessage(normalizeJSONStringValue(inputsText, `[]`))
		item.Mapping = json.RawMessage(normalizeJSONStringValue(mappingText, `{}`))
		item.DAG = json.RawMessage(normalizeJSONStringValue(dagText, `{}`))
		item.NodeContracts = json.RawMessage(normalizeJSONStringValue(contractsText, `[]`))
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// parseWeaverDAG 解析并校验发布所需的 DAG 原始数据。
// 参数：raw 为 DAG JSON 原文。
// 返回：解析后的 DAG 模型。
// 异常：DAG 缺失或 JSON 格式非法时返回错误。
func parseWeaverDAG(raw json.RawMessage) (WeaverDAG, error) {
	var dag WeaverDAG
	text := strings.TrimSpace(string(raw))
	if text == "" || text == "null" {
		return WeaverDAG{}, errors.New("dag is required")
	}
	if err := json.Unmarshal(raw, &dag); err != nil {
		return WeaverDAG{}, err
	}
	return dag, nil
}

// freezeWeaverNodeContracts 对 DAG 执行拓扑校验并冻结节点接口契约。
// 参数：dag 为待校验的有向无环图定义。
// 返回：按拓扑顺序的节点契约列表。
// 异常：节点缺失、边非法或存在环路时返回错误。
func freezeWeaverNodeContracts(dag WeaverDAG) ([]WeaverNodeContract, error) {
	if len(dag.Nodes) == 0 {
		return nil, errors.New("dag.nodes is empty")
	}
	nodeMap := make(map[string]WeaverNode, len(dag.Nodes))
	inDegree := make(map[string]int, len(dag.Nodes))
	graph := make(map[string][]string, len(dag.Nodes))
	for _, node := range dag.Nodes {
		nodeID := strings.TrimSpace(node.ID)
		if nodeID == "" {
			return nil, errors.New("dag.nodes.id is required")
		}
		if strings.TrimSpace(node.Type) == "" {
			return nil, errors.New("dag.nodes.type is required")
		}
		if _, exists := nodeMap[nodeID]; exists {
			return nil, errors.New("dag.nodes.id is duplicated")
		}
		node.ID = nodeID
		nodeMap[nodeID] = node
		inDegree[nodeID] = 0
		graph[nodeID] = make([]string, 0)
	}
	for _, edge := range dag.Edges {
		from := strings.TrimSpace(edge.From)
		to := strings.TrimSpace(edge.To)
		if from == "" || to == "" {
			return nil, errors.New("dag.edges.from or dag.edges.to is empty")
		}
		if _, exists := nodeMap[from]; !exists {
			return nil, errors.New("dag.edges.from node not found")
		}
		if _, exists := nodeMap[to]; !exists {
			return nil, errors.New("dag.edges.to node not found")
		}
		graph[from] = append(graph[from], to)
		inDegree[to]++
	}
	queue := make([]string, 0, len(dag.Nodes))
	for _, node := range dag.Nodes {
		if inDegree[node.ID] == 0 {
			queue = append(queue, node.ID)
		}
	}
	ordered := make([]string, 0, len(dag.Nodes))
	for len(queue) > 0 {
		nodeID := queue[0]
		queue = queue[1:]
		ordered = append(ordered, nodeID)
		for _, nextID := range graph[nodeID] {
			inDegree[nextID]--
			if inDegree[nextID] == 0 {
				queue = append(queue, nextID)
			}
		}
	}
	if len(ordered) != len(dag.Nodes) {
		return nil, errors.New("dag has cycle")
	}
	contracts := make([]WeaverNodeContract, 0, len(ordered))
	for _, nodeID := range ordered {
		node := nodeMap[nodeID]
		contracts = append(contracts, WeaverNodeContract{
			NodeID:  node.ID,
			Type:    node.Type,
			Inputs:  node.Inputs,
			Outputs: node.Outputs,
		})
	}
	return contracts, nil
}

// normalizeJSONText 对 JSON 原文做空值归一，保证落库字段可稳定解析。
// 参数：raw 为原始 JSON，fallback 为空值时替代文本。
// 返回：归一后的 JSON 文本。
// 异常：无。
func normalizeJSONText(raw json.RawMessage, fallback string) json.RawMessage {
	return json.RawMessage(normalizeJSONStringValue(string(raw), fallback))
}

// normalizeJSONStringValue 对字符串形式 JSON 做空值归一。
// 参数：content 为原始文本，fallback 为空值时替代文本。
// 返回：归一后的 JSON 字符串。
// 异常：无。
func normalizeJSONStringValue(content string, fallback string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" || trimmed == "null" {
		return fallback
	}
	return trimmed
}
