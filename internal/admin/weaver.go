package admin

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"YoBFF/internal/store"
)

type weaverInputPayload struct {
	Name    string          `json:"name"`
	Payload json.RawMessage `json:"payload"`
}

type weaverDraftPayload struct {
	Name    string               `json:"name"`
	Inputs  []weaverInputPayload `json:"inputs"`
	Mapping json.RawMessage      `json:"mapping"`
	DAG     json.RawMessage      `json:"dag"`
}

type weaverRunPayload struct {
	Inputs  []weaverInputPayload `json:"inputs"`
	Mapping json.RawMessage      `json:"mapping"`
	DAG     json.RawMessage      `json:"dag"`
}

type weaverRunSource struct {
	Name  string `json:"name"`
	Ok    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type weaverRunResponse struct {
	Output     any               `json:"output"`
	DurationMs int64             `json:"duration_ms"`
	Sources    []weaverRunSource `json:"sources"`
}

// weaverDrafts 提供 Weaver 草稿的列表与创建能力。
func (s *Server) weaverDrafts(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "store_unavailable", "weaver store unavailable", r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		limit := parsePositiveInt(r.URL.Query().Get("limit"), 50)
		items, err := s.store.ListWeaverDrafts(limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "weaver_list_failed", err.Error(), r)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"items": items,
		})
	case http.MethodPost:
		draft, err := decodeWeaverDraftPayload(w, r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", err.Error(), r)
			return
		}
		draft.Operator = operatorFromRequest(r)
		result, err := s.store.CreateWeaverDraft(draft)
		if err != nil {
			writeError(w, http.StatusBadRequest, "weaver_create_failed", err.Error(), r)
			return
		}
		_ = s.store.SaveAudit("weaver_create", result.ID, operatorFromRequest(r), map[string]any{
			"name": result.Name,
		})
		writeJSON(w, http.StatusOK, result)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
	}
}

// weaverDraftRouter 解析 Weaver 草稿子路由并分发到对应处理器。
func (s *Server) weaverDraftRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/weaver/drafts/")
	if path == "" || path == r.URL.Path {
		writeError(w, http.StatusNotFound, "not_found", "not found", r)
		return
	}
	parts := strings.Split(path, "/")
	draftID := strings.TrimSpace(parts[0])
	if draftID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "draft_id is required", r)
		return
	}
	if len(parts) == 1 {
		s.weaverDraftDetail(w, r, draftID)
		return
	}
	if len(parts) == 2 && parts[1] == "run" {
		s.runWeaverDraft(w, r, draftID)
		return
	}
	if len(parts) == 2 && parts[1] == "publish" {
		s.publishWeaverDraft(w, r, draftID)
		return
	}
	if len(parts) == 2 && parts[1] == "versions" {
		s.weaverDraftVersions(w, r, draftID)
		return
	}
	writeError(w, http.StatusNotFound, "not_found", "not found", r)
}

// weaverVersionRouter 解析 Weaver 发布版本子路由并分发到对应处理器。
func (s *Server) weaverVersionRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/weaver/versions/")
	if path == "" || path == r.URL.Path {
		writeError(w, http.StatusNotFound, "not_found", "not found", r)
		return
	}
	parts := strings.Split(path, "/")
	versionID := strings.TrimSpace(parts[0])
	if versionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "version_id is required", r)
		return
	}
	if len(parts) == 1 {
		s.weaverVersionDetail(w, r, versionID)
		return
	}
	if len(parts) == 2 && parts[1] == "run" {
		s.runWeaverVersion(w, r, versionID)
		return
	}
	writeError(w, http.StatusNotFound, "not_found", "not found", r)
}

// weaverDraftDetail 处理 Weaver 草稿详情读写与删除。
func (s *Server) weaverDraftDetail(w http.ResponseWriter, r *http.Request, draftID string) {
	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "store_unavailable", "weaver store unavailable", r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		item, err := s.store.GetWeaverDraft(draftID)
		if err != nil {
			if errors.Is(err, store.ErrWeaverDraftNotFound) {
				writeError(w, http.StatusNotFound, "draft_not_found", "weaver draft not found", r)
				return
			}
			writeError(w, http.StatusInternalServerError, "weaver_query_failed", err.Error(), r)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodPut:
		draft, err := decodeWeaverDraftPayload(w, r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", err.Error(), r)
			return
		}
		draft.ID = draftID
		draft.Operator = operatorFromRequest(r)
		result, err := s.store.UpdateWeaverDraft(draft)
		if err != nil {
			if errors.Is(err, store.ErrWeaverDraftNotFound) {
				writeError(w, http.StatusNotFound, "draft_not_found", "weaver draft not found", r)
				return
			}
			writeError(w, http.StatusBadRequest, "weaver_update_failed", err.Error(), r)
			return
		}
		_ = s.store.SaveAudit("weaver_update", result.ID, operatorFromRequest(r), map[string]any{
			"name": result.Name,
		})
		writeJSON(w, http.StatusOK, result)
	case http.MethodDelete:
		item, err := s.store.GetWeaverDraft(draftID)
		if err != nil {
			if errors.Is(err, store.ErrWeaverDraftNotFound) {
				writeError(w, http.StatusNotFound, "draft_not_found", "weaver draft not found", r)
				return
			}
			writeError(w, http.StatusInternalServerError, "weaver_query_failed", err.Error(), r)
			return
		}
		if err := s.store.DeleteWeaverDraft(draftID); err != nil {
			if errors.Is(err, store.ErrWeaverDraftNotFound) {
				writeError(w, http.StatusNotFound, "draft_not_found", "weaver draft not found", r)
				return
			}
			writeError(w, http.StatusInternalServerError, "weaver_delete_failed", err.Error(), r)
			return
		}
		_ = s.store.SaveAudit("weaver_delete", draftID, operatorFromRequest(r), map[string]any{
			"name": item.Name,
		})
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "deleted",
			"id":     draftID,
		})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
	}
}

// runWeaverDraft 运行草稿编排，支持在请求中覆盖 inputs、mapping 与 dag。
func (s *Server) runWeaverDraft(w http.ResponseWriter, r *http.Request, draftID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
		return
	}
	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "store_unavailable", "weaver store unavailable", r)
		return
	}
	startedAt := time.Now()
	payload, err := decodeOptionalWeaverRunPayload(w, r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error(), r)
		return
	}
	draft, err := s.store.GetWeaverDraft(draftID)
	if err != nil {
		if errors.Is(err, store.ErrWeaverDraftNotFound) {
			writeError(w, http.StatusNotFound, "draft_not_found", "weaver draft not found", r)
			return
		}
		writeError(w, http.StatusInternalServerError, "weaver_query_failed", err.Error(), r)
		return
	}
	inputs, mapping, dag, sources, err := resolveWeaverDraftRunContext(draft, payload)
	if err != nil {
		writeError(w, http.StatusBadRequest, "weaver_run_failed", err.Error(), r)
		return
	}
	output, err := executeWeaverRun(inputs, mapping, dag)
	if err != nil {
		writeError(w, http.StatusBadRequest, "weaver_run_failed", err.Error(), r)
		return
	}
	_ = s.store.SaveAudit("weaver_run", draftID, operatorFromRequest(r), map[string]any{
		"name": draft.Name,
	})
	writeJSON(w, http.StatusOK, weaverRunResponse{
		Output:     output,
		DurationMs: time.Since(startedAt).Milliseconds(),
		Sources:    sources,
	})
}

// publishWeaverDraft 将草稿发布为流程版本并冻结节点接口契约。
func (s *Server) publishWeaverDraft(w http.ResponseWriter, r *http.Request, draftID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
		return
	}
	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "store_unavailable", "weaver store unavailable", r)
		return
	}
	version, err := s.store.CreateWeaverVersion(draftID, operatorFromRequest(r), "publish")
	if err != nil {
		if errors.Is(err, store.ErrWeaverDraftNotFound) {
			writeError(w, http.StatusNotFound, "draft_not_found", "weaver draft not found", r)
			return
		}
		writeError(w, http.StatusBadRequest, "weaver_publish_failed", err.Error(), r)
		return
	}
	_ = s.store.SaveAudit("weaver_publish", draftID, operatorFromRequest(r), map[string]any{
		"version_id": version.ID,
		"version":    version.Version,
	})
	writeJSON(w, http.StatusOK, version)
}

// weaverDraftVersions 返回指定草稿的版本列表。
func (s *Server) weaverDraftVersions(w http.ResponseWriter, r *http.Request, draftID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
		return
	}
	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "store_unavailable", "weaver store unavailable", r)
		return
	}
	limit := parsePositiveInt(r.URL.Query().Get("limit"), 20)
	items, err := s.store.ListWeaverVersionsByDraft(draftID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "weaver_version_list_failed", err.Error(), r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
	})
}

// weaverVersionDetail 返回指定版本的冻结快照详情。
func (s *Server) weaverVersionDetail(w http.ResponseWriter, r *http.Request, versionID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
		return
	}
	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "store_unavailable", "weaver store unavailable", r)
		return
	}
	item, err := s.store.GetWeaverVersion(versionID)
	if err != nil {
		if errors.Is(err, store.ErrWeaverVersionNotFound) {
			writeError(w, http.StatusNotFound, "version_not_found", "weaver version not found", r)
			return
		}
		writeError(w, http.StatusInternalServerError, "weaver_query_failed", err.Error(), r)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// runWeaverVersion 运行已发布版本，仅允许覆盖运行输入。
func (s *Server) runWeaverVersion(w http.ResponseWriter, r *http.Request, versionID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
		return
	}
	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "store_unavailable", "weaver store unavailable", r)
		return
	}
	startedAt := time.Now()
	payload, err := decodeOptionalWeaverRunPayload(w, r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error(), r)
		return
	}
	item, err := s.store.GetWeaverVersion(versionID)
	if err != nil {
		if errors.Is(err, store.ErrWeaverVersionNotFound) {
			writeError(w, http.StatusNotFound, "version_not_found", "weaver version not found", r)
			return
		}
		writeError(w, http.StatusInternalServerError, "weaver_query_failed", err.Error(), r)
		return
	}
	inputs, mapping, dag, sources, err := resolveWeaverVersionRunContext(item, payload)
	if err != nil {
		writeError(w, http.StatusBadRequest, "weaver_run_failed", err.Error(), r)
		return
	}
	output, err := executeWeaverRun(inputs, mapping, dag)
	if err != nil {
		writeError(w, http.StatusBadRequest, "weaver_run_failed", err.Error(), r)
		return
	}
	_ = s.store.SaveAudit("weaver_version_run", versionID, operatorFromRequest(r), map[string]any{
		"draft_id": item.DraftID,
		"version":  item.Version,
	})
	writeJSON(w, http.StatusOK, weaverRunResponse{
		Output:     output,
		DurationMs: time.Since(startedAt).Milliseconds(),
		Sources:    sources,
	})
}

// decodeWeaverDraftPayload 解析草稿写请求并转换为存储层对象。
func decodeWeaverDraftPayload(w http.ResponseWriter, r *http.Request) (store.WeaverDraft, error) {
	var payload weaverDraftPayload
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return store.WeaverDraft{}, err
	}
	if payload.Name == "" {
		return store.WeaverDraft{}, errMissingField("name")
	}
	if len(payload.Inputs) == 0 {
		return store.WeaverDraft{}, errMissingField("inputs")
	}
	if len(payload.Mapping) == 0 {
		return store.WeaverDraft{}, errMissingField("mapping")
	}
	for _, item := range payload.Inputs {
		if strings.TrimSpace(item.Name) == "" {
			return store.WeaverDraft{}, errMissingField("inputs.name")
		}
		if len(item.Payload) == 0 {
			return store.WeaverDraft{}, errMissingField("inputs.payload")
		}
	}
	inputsBytes, err := json.Marshal(payload.Inputs)
	if err != nil {
		return store.WeaverDraft{}, err
	}
	return store.WeaverDraft{
		Name:    payload.Name,
		Inputs:  inputsBytes,
		Mapping: payload.Mapping,
		DAG:     payload.DAG,
	}, nil
}

// decodeOptionalWeaverRunPayload 解析可选运行请求体。
func decodeOptionalWeaverRunPayload(w http.ResponseWriter, r *http.Request) (*weaverRunPayload, error) {
	if r.ContentLength == 0 {
		return nil, nil
	}
	var payload weaverRunPayload
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, err
	}
	return &payload, nil
}

// resolveWeaverDraftRunContext 解析草稿运行所需的输入、映射、DAG 与来源状态。
func resolveWeaverDraftRunContext(
	draft store.WeaverDraft,
	payload *weaverRunPayload,
) (map[string]any, any, store.WeaverDAG, []weaverRunSource, error) {
	var inputs []weaverInputPayload
	var mapping any
	var dag store.WeaverDAG
	if payload == nil || len(payload.Inputs) == 0 {
		if err := json.Unmarshal(draft.Inputs, &inputs); err != nil {
			return nil, nil, store.WeaverDAG{}, nil, err
		}
	} else {
		inputs = payload.Inputs
	}
	if payload == nil || len(payload.Mapping) == 0 {
		if err := json.Unmarshal(draft.Mapping, &mapping); err != nil {
			return nil, nil, store.WeaverDAG{}, nil, err
		}
	} else {
		if err := json.Unmarshal(payload.Mapping, &mapping); err != nil {
			return nil, nil, store.WeaverDAG{}, nil, err
		}
	}
	dagRaw := draft.DAG
	if payload != nil && len(payload.DAG) > 0 {
		dagRaw = payload.DAG
	}
	dag, err := parseOptionalWeaverDAG(dagRaw)
	if err != nil {
		return nil, nil, store.WeaverDAG{}, nil, err
	}
	inputMap, sources, err := buildWeaverInputMap(inputs)
	if err != nil {
		return nil, nil, store.WeaverDAG{}, nil, err
	}
	return inputMap, mapping, dag, sources, nil
}

// resolveWeaverVersionRunContext 解析发布版本运行所需的输入、映射、DAG 与来源状态。
func resolveWeaverVersionRunContext(
	item store.WeaverVersion,
	payload *weaverRunPayload,
) (map[string]any, any, store.WeaverDAG, []weaverRunSource, error) {
	var inputs []weaverInputPayload
	var mapping any
	var dag store.WeaverDAG
	if payload == nil || len(payload.Inputs) == 0 {
		if err := json.Unmarshal(item.Inputs, &inputs); err != nil {
			return nil, nil, store.WeaverDAG{}, nil, err
		}
	} else {
		inputs = payload.Inputs
	}
	if err := json.Unmarshal(item.Mapping, &mapping); err != nil {
		return nil, nil, store.WeaverDAG{}, nil, err
	}
	dag, err := parseOptionalWeaverDAG(item.DAG)
	if err != nil {
		return nil, nil, store.WeaverDAG{}, nil, err
	}
	inputMap, sources, err := buildWeaverInputMap(inputs)
	if err != nil {
		return nil, nil, store.WeaverDAG{}, nil, err
	}
	return inputMap, mapping, dag, sources, nil
}

// buildWeaverInputMap 将输入列表转换为可执行输入映射并记录来源状态。
func buildWeaverInputMap(inputs []weaverInputPayload) (map[string]any, []weaverRunSource, error) {
	inputMap := make(map[string]any)
	var sources []weaverRunSource
	for _, item := range inputs {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			return nil, nil, errMissingField("inputs.name")
		}
		if _, exists := inputMap[name]; exists {
			return nil, nil, errDuplicateField("inputs.name")
		}
		var value any
		if len(item.Payload) == 0 {
			sources = append(sources, weaverRunSource{
				Name:  name,
				Ok:    false,
				Error: "payload is empty",
			})
			continue
		}
		if err := json.Unmarshal(item.Payload, &value); err != nil {
			sources = append(sources, weaverRunSource{
				Name:  name,
				Ok:    false,
				Error: "payload is invalid",
			})
			continue
		}
		inputMap[name] = value
		sources = append(sources, weaverRunSource{
			Name: name,
			Ok:   true,
		})
	}
	return inputMap, sources, nil
}

// parseOptionalWeaverDAG 解析可选 DAG 配置，空配置时返回零值 DAG。
func parseOptionalWeaverDAG(raw json.RawMessage) (store.WeaverDAG, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" || trimmed == "{}" {
		return store.WeaverDAG{}, nil
	}
	var dag store.WeaverDAG
	if err := json.Unmarshal(raw, &dag); err != nil {
		return store.WeaverDAG{}, err
	}
	return dag, nil
}

// executeWeaverRun 根据运行输入、映射与 DAG 生成最小可执行结果。
func executeWeaverRun(inputs map[string]any, mapping any, dag store.WeaverDAG) (map[string]any, error) {
	if len(dag.Nodes) == 0 {
		return map[string]any{
			"inputs":  inputs,
			"mapping": mapping,
		}, nil
	}
	nodeResults, finalOutput, err := executeWeaverDAG(dag, inputs)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"inputs":      inputs,
		"mapping":     mapping,
		"dag_output":  finalOutput,
		"node_result": nodeResults,
	}, nil
}

// executeWeaverDAG 执行 DAG 的最小执行语义并返回节点输出与最终输出。
func executeWeaverDAG(dag store.WeaverDAG, inputs map[string]any) (map[string]any, any, error) {
	orderedNodes, dependencyMap, err := topologicalWeaverNodes(dag)
	if err != nil {
		return nil, nil, err
	}
	nodeOutputs := make(map[string]any, len(orderedNodes))
	for _, node := range orderedNodes {
		dependencyOutput := make(map[string]any)
		for _, parent := range dependencyMap[node.ID] {
			dependencyOutput[parent] = nodeOutputs[parent]
		}
		nodeInput := make(map[string]any)
		for _, inputKey := range node.Inputs {
			if value, ok := inputs[inputKey]; ok {
				nodeInput[inputKey] = value
			}
		}
		config := map[string]any{}
		if len(node.Config) > 0 {
			_ = json.Unmarshal(node.Config, &config)
		}
		nodeOutputs[node.ID] = map[string]any{
			"id":      node.ID,
			"type":    node.Type,
			"inputs":  nodeInput,
			"deps":    dependencyOutput,
			"config":  config,
			"outputs": node.Outputs,
		}
	}
	finalNodeID := strings.TrimSpace(dag.OutputNodeID)
	if finalNodeID == "" {
		finalNodeID = orderedNodes[len(orderedNodes)-1].ID
	}
	finalOutput, ok := nodeOutputs[finalNodeID]
	if !ok {
		return nil, nil, errors.New("dag.output_node_id not found")
	}
	return nodeOutputs, finalOutput, nil
}

// topologicalWeaverNodes 计算 DAG 拓扑序并返回依赖关系映射。
func topologicalWeaverNodes(dag store.WeaverDAG) ([]store.WeaverNode, map[string][]string, error) {
	if len(dag.Nodes) == 0 {
		return nil, nil, errors.New("dag.nodes is empty")
	}
	nodeMap := make(map[string]store.WeaverNode, len(dag.Nodes))
	inDegree := make(map[string]int, len(dag.Nodes))
	adjacent := make(map[string][]string, len(dag.Nodes))
	dependencyMap := make(map[string][]string, len(dag.Nodes))
	for _, node := range dag.Nodes {
		nodeID := strings.TrimSpace(node.ID)
		if nodeID == "" {
			return nil, nil, errors.New("dag.nodes.id is required")
		}
		if strings.TrimSpace(node.Type) == "" {
			return nil, nil, errors.New("dag.nodes.type is required")
		}
		if _, exists := nodeMap[nodeID]; exists {
			return nil, nil, errors.New("dag.nodes.id is duplicated")
		}
		node.ID = nodeID
		nodeMap[nodeID] = node
		inDegree[nodeID] = 0
		adjacent[nodeID] = make([]string, 0)
		dependencyMap[nodeID] = make([]string, 0)
	}
	for _, edge := range dag.Edges {
		from := strings.TrimSpace(edge.From)
		to := strings.TrimSpace(edge.To)
		if from == "" || to == "" {
			return nil, nil, errors.New("dag.edges.from or dag.edges.to is empty")
		}
		if _, exists := nodeMap[from]; !exists {
			return nil, nil, errors.New("dag.edges.from node not found")
		}
		if _, exists := nodeMap[to]; !exists {
			return nil, nil, errors.New("dag.edges.to node not found")
		}
		adjacent[from] = append(adjacent[from], to)
		dependencyMap[to] = append(dependencyMap[to], from)
		inDegree[to]++
	}
	queue := make([]string, 0)
	for _, node := range dag.Nodes {
		if inDegree[node.ID] == 0 {
			queue = append(queue, node.ID)
		}
	}
	ordered := make([]store.WeaverNode, 0, len(dag.Nodes))
	for len(queue) > 0 {
		nodeID := queue[0]
		queue = queue[1:]
		ordered = append(ordered, nodeMap[nodeID])
		for _, nextID := range adjacent[nodeID] {
			inDegree[nextID]--
			if inDegree[nextID] == 0 {
				queue = append(queue, nextID)
			}
		}
	}
	if len(ordered) != len(dag.Nodes) {
		return nil, nil, errors.New("dag has cycle")
	}
	return ordered, dependencyMap, nil
}

type requestFieldError struct {
	field string
}

// Error 返回字段校验错误信息。
func (e requestFieldError) Error() string {
	return e.field
}

// errMissingField 构造缺失字段错误。
func errMissingField(field string) error {
	return requestFieldError{field: field + " is required"}
}

// errDuplicateField 构造重复字段错误。
func errDuplicateField(field string) error {
	return requestFieldError{field: field + " is duplicated"}
}
