package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"slices"
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
	Retry   weaverRunRetryPolicy `json:"retry"`
}

type weaverDraftValidatePayload struct {
	DAG json.RawMessage `json:"dag"`
}

type weaverRunSource struct {
	Name  string `json:"name"`
	Ok    bool   `json:"ok"`
	Code  string `json:"code,omitempty"`
	Error string `json:"error,omitempty"`
}

type weaverRunResponse struct {
	RunID      string                 `json:"run_id"`
	Status     string                 `json:"status"`
	Output     any                    `json:"output"`
	DurationMs int64                  `json:"duration_ms"`
	Sources    []weaverRunSource      `json:"sources"`
	Attempts   []weaverRunAttempt     `json:"attempts"`
	Failures   []weaverRunFailure     `json:"failures"`
	Retry      weaverRunRetrySnapshot `json:"retry"`
}

type weaverRunRetryPolicy struct {
	MaxAttempts        int      `json:"max_attempts"`
	RetryOnNodeError   bool     `json:"retry_on_node_error"`
	RetryOnSourceError bool     `json:"retry_on_source_error"`
	RetryableCodes     []string `json:"retryable_codes"`
	BackoffInitialMs   int      `json:"backoff_initial_ms"`
	BackoffMultiplier  float64  `json:"backoff_multiplier"`
	BackoffMaxMs       int      `json:"backoff_max_ms"`
}

type weaverRunRetrySnapshot struct {
	MaxAttempts       int      `json:"max_attempts"`
	Used              int      `json:"used"`
	Triggered         bool     `json:"triggered"`
	RetryableCodes    []string `json:"retryable_codes"`
	BackoffInitialMs  int      `json:"backoff_initial_ms"`
	BackoffMultiplier float64  `json:"backoff_multiplier"`
	BackoffMaxMs      int      `json:"backoff_max_ms"`
	DelaysMs          []int64  `json:"delays_ms"`
}

type weaverRunFailure struct {
	Scope     string `json:"scope"`
	Code      string `json:"code"`
	NodeID    string `json:"node_id,omitempty"`
	Source    string `json:"source,omitempty"`
	Attempt   int    `json:"attempt"`
	Error     string `json:"error"`
	Retryable bool   `json:"retryable"`
}

type weaverRunNodeExecution struct {
	NodeID      string   `json:"node_id"`
	NodeType    string   `json:"node_type"`
	Attempt     int      `json:"attempt"`
	Status      string   `json:"status"`
	DurationMs  int64    `json:"duration_ms"`
	InputKeys   []string `json:"input_keys"`
	ParentNodes []string `json:"parent_nodes"`
	OutputKeys  []string `json:"output_keys"`
	Error       string   `json:"error,omitempty"`
}

type weaverRunAttempt struct {
	Attempt        int                      `json:"attempt"`
	Status         string                   `json:"status"`
	DurationMs     int64                    `json:"duration_ms"`
	NodeExecutions []weaverRunNodeExecution `json:"node_executions"`
	Failure        *weaverRunFailure        `json:"failure,omitempty"`
}

type weaverRunResult struct {
	Output         map[string]any
	AttemptOutputs []weaverRunAttempt
	Failures       []weaverRunFailure
	DelaysMs       []int64
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
	if len(parts) == 2 && parts[1] == "validate" {
		s.validateWeaverDraft(w, r, draftID)
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
	runID := newRequestID()
	runResult := executeWeaverRunWithRetry(inputs, mapping, dag, sources, payload.Retry)
	status := inferRunStatus(runResult.AttemptOutputs)
	durationMs := time.Since(startedAt).Milliseconds()
	retrySnapshot := buildRunRetrySnapshot(payload.Retry, len(runResult.AttemptOutputs), runResult.DelaysMs)
	_ = s.persistWeaverRunHistory(store.WeaverRunHistory{
		RunID:        runID,
		Scope:        "draft",
		TargetID:     draftID,
		DraftID:      draftID,
		Status:       status,
		DurationMs:   durationMs,
		AttemptsUsed: len(runResult.AttemptOutputs),
		Failures:     mustMarshalJSON(runResult.Failures, `[]`),
		Retry:        mustMarshalJSON(retrySnapshot, `{}`),
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
		Operator:     operatorFromRequest(r),
	})
	_ = s.store.SaveAudit("weaver_run", draftID, operatorFromRequest(r), map[string]any{
		"name": draft.Name,
		"run":  runID,
	})
	writeJSON(w, http.StatusOK, weaverRunResponse{
		RunID:      runID,
		Status:     status,
		Output:     runResult.Output,
		DurationMs: durationMs,
		Sources:    sources,
		Attempts:   runResult.AttemptOutputs,
		Failures:   runResult.Failures,
		Retry:      retrySnapshot,
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

// weaverNodeContracts 返回 Weaver 节点契约目录。
func (s *Server) weaverNodeContracts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
		return
	}
	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "store_unavailable", "weaver store unavailable", r)
		return
	}
	items := s.store.ListWeaverNodeContractCatalog()
	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
	})
}

// weaverRunStats 返回最近 N 次 Weaver 运行的错误码分组与趋势统计，支持按草稿或版本过滤。
func (s *Server) weaverRunStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
		return
	}
	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "store_unavailable", "weaver store unavailable", r)
		return
	}
	limit := parsePositiveInt(r.URL.Query().Get("limit"), 20)
	scope := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("scope")))
	targetID := strings.TrimSpace(r.URL.Query().Get("target_id"))
	if (scope == "" && targetID != "") || (scope != "" && targetID == "") {
		writeError(w, http.StatusBadRequest, "invalid_request", "scope and target_id must be provided together", r)
		return
	}
	if scope != "" && scope != "draft" && scope != "version" {
		writeError(w, http.StatusBadRequest, "invalid_request", "scope must be draft or version", r)
		return
	}
	stats, err := s.store.ListWeaverRunStats(limit, scope, targetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "weaver_run_stats_failed", err.Error(), r)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// validateWeaverDraft 校验草稿 DAG 并返回冻结后的节点契约。
func (s *Server) validateWeaverDraft(w http.ResponseWriter, r *http.Request, draftID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
		return
	}
	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "store_unavailable", "weaver store unavailable", r)
		return
	}
	payload, err := decodeOptionalWeaverValidatePayload(w, r)
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
	dagRaw := draft.DAG
	if payload != nil && len(payload.DAG) > 0 {
		dagRaw = payload.DAG
	}
	dag, err := parseOptionalWeaverDAG(dagRaw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "weaver_invalid_dag", err.Error(), r)
		return
	}
	contracts, err := s.store.ValidateWeaverDAG(dag)
	if err != nil {
		writeError(w, http.StatusBadRequest, "weaver_invalid_dag", err.Error(), r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":         "valid",
		"draft_id":       draftID,
		"node_contracts": contracts,
	})
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
	runID := newRequestID()
	runResult := executeWeaverRunWithRetry(inputs, mapping, dag, sources, payload.Retry)
	status := inferRunStatus(runResult.AttemptOutputs)
	durationMs := time.Since(startedAt).Milliseconds()
	retrySnapshot := buildRunRetrySnapshot(payload.Retry, len(runResult.AttemptOutputs), runResult.DelaysMs)
	_ = s.persistWeaverRunHistory(store.WeaverRunHistory{
		RunID:        runID,
		Scope:        "version",
		TargetID:     versionID,
		DraftID:      item.DraftID,
		VersionID:    versionID,
		Status:       status,
		DurationMs:   durationMs,
		AttemptsUsed: len(runResult.AttemptOutputs),
		Failures:     mustMarshalJSON(runResult.Failures, `[]`),
		Retry:        mustMarshalJSON(retrySnapshot, `{}`),
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
		Operator:     operatorFromRequest(r),
	})
	_ = s.store.SaveAudit("weaver_version_run", versionID, operatorFromRequest(r), map[string]any{
		"draft_id": item.DraftID,
		"version":  item.Version,
		"run":      runID,
	})
	writeJSON(w, http.StatusOK, weaverRunResponse{
		RunID:      runID,
		Status:     status,
		Output:     runResult.Output,
		DurationMs: durationMs,
		Sources:    sources,
		Attempts:   runResult.AttemptOutputs,
		Failures:   runResult.Failures,
		Retry:      retrySnapshot,
	})
}

// persistWeaverRunHistory 写入 Weaver 运行历史，供统计接口查询。
func (s *Server) persistWeaverRunHistory(item store.WeaverRunHistory) error {
	if s == nil || s.store == nil {
		return errors.New("store unavailable")
	}
	return s.store.SaveWeaverRunHistory(item)
}

// mustMarshalJSON 序列化任意值，失败时返回默认 JSON 文本。
func mustMarshalJSON(value any, fallback string) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(fallback)
	}
	return encoded
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
		return &weaverRunPayload{}, nil
	}
	var payload weaverRunPayload
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		if err == io.EOF {
			return &weaverRunPayload{}, nil
		}
		return nil, err
	}
	return &payload, nil
}

// decodeOptionalWeaverValidatePayload 解析可选草稿校验请求体。
func decodeOptionalWeaverValidatePayload(w http.ResponseWriter, r *http.Request) (*weaverDraftValidatePayload, error) {
	if r.ContentLength == 0 {
		return nil, nil
	}
	var payload weaverDraftValidatePayload
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
				Code:  "source_payload_empty",
				Error: "payload is empty",
			})
			continue
		}
		if err := json.Unmarshal(item.Payload, &value); err != nil {
			sources = append(sources, weaverRunSource{
				Name:  name,
				Ok:    false,
				Code:  "source_payload_invalid",
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
func executeWeaverRunWithRetry(
	inputs map[string]any,
	mapping any,
	dag store.WeaverDAG,
	sources []weaverRunSource,
	retryPolicy weaverRunRetryPolicy,
) weaverRunResult {
	policy := normalizeRunRetryPolicy(retryPolicy)
	attempts := make([]weaverRunAttempt, 0, policy.MaxAttempts)
	failures := make([]weaverRunFailure, 0, 1)
	delays := make([]int64, 0, policy.MaxAttempts-1)
	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		runStartedAt := time.Now()
		output, executions, failure := executeWeaverRun(inputs, mapping, dag, sources, attempt)
		attemptItem := weaverRunAttempt{
			Attempt:        attempt,
			Status:         "succeeded",
			DurationMs:     time.Since(runStartedAt).Milliseconds(),
			NodeExecutions: executions,
		}
		if failure == nil {
			attempts = append(attempts, attemptItem)
			return weaverRunResult{
				Output:         output,
				AttemptOutputs: attempts,
				Failures:       failures,
				DelaysMs:       delays,
			}
		}
		attemptItem.Status = "failed"
		attemptItem.Failure = failure
		attempts = append(attempts, attemptItem)
		failures = append(failures, *failure)
		if !shouldRetryRun(*failure, policy, attempt) {
			break
		}
		delay := buildBackoffDelayMs(policy, attempt)
		delays = append(delays, delay)
		if delay > 0 {
			time.Sleep(time.Duration(delay) * time.Millisecond)
		}
	}
	return weaverRunResult{
		Output: map[string]any{
			"inputs":  inputs,
			"mapping": mapping,
		},
		AttemptOutputs: attempts,
		Failures:       failures,
		DelaysMs:       delays,
	}
}

func executeWeaverRun(
	inputs map[string]any,
	mapping any,
	dag store.WeaverDAG,
	sources []weaverRunSource,
	attempt int,
) (map[string]any, []weaverRunNodeExecution, *weaverRunFailure) {
	sourcesFailure := buildSourceFailure(sources, attempt)
	if sourcesFailure != nil {
		return nil, nil, sourcesFailure
	}
	if len(dag.Nodes) == 0 {
		return map[string]any{
			"inputs":  inputs,
			"mapping": mapping,
		}, nil, nil
	}
	nodeResults, finalOutput, executions, failure := executeWeaverDAG(dag, inputs, attempt)
	if failure != nil {
		return nil, executions, failure
	}
	return map[string]any{
		"inputs":      inputs,
		"mapping":     mapping,
		"dag_output":  finalOutput,
		"node_result": nodeResults,
	}, executions, nil
}

func executeWeaverDAG(
	dag store.WeaverDAG,
	inputs map[string]any,
	attempt int,
) (map[string]any, any, []weaverRunNodeExecution, *weaverRunFailure) {
	orderedNodes, dependencyMap, err := topologicalWeaverNodes(dag)
	if err != nil {
		return nil, nil, nil, &weaverRunFailure{
			Scope:     "dag",
			Code:      classifyDAGErrorCode(err),
			Attempt:   attempt,
			Error:     err.Error(),
			Retryable: false,
		}
	}
	nodeOutputs := make(map[string]any, len(orderedNodes))
	executions := make([]weaverRunNodeExecution, 0, len(orderedNodes))
	for _, node := range orderedNodes {
		nodeStartedAt := time.Now()
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
		if failureCode, failureText, shouldFail := evaluateNodeFailure(node.ID, config, attempt); shouldFail {
			executions = append(executions, weaverRunNodeExecution{
				NodeID:      node.ID,
				NodeType:    node.Type,
				Attempt:     attempt,
				Status:      "failed",
				DurationMs:  time.Since(nodeStartedAt).Milliseconds(),
				InputKeys:   mapKeys(nodeInput),
				ParentNodes: dependencyMap[node.ID],
				OutputKeys:  node.Outputs,
				Error:       failureText,
			})
			return nil, nil, executions, &weaverRunFailure{
				Scope:     "node",
				Code:      failureCode,
				NodeID:    node.ID,
				Attempt:   attempt,
				Error:     failureText,
				Retryable: true,
			}
		}
		nodeOutputs[node.ID] = map[string]any{
			"id":      node.ID,
			"type":    node.Type,
			"inputs":  nodeInput,
			"deps":    dependencyOutput,
			"config":  config,
			"outputs": node.Outputs,
		}
		executions = append(executions, weaverRunNodeExecution{
			NodeID:      node.ID,
			NodeType:    node.Type,
			Attempt:     attempt,
			Status:      "succeeded",
			DurationMs:  time.Since(nodeStartedAt).Milliseconds(),
			InputKeys:   mapKeys(nodeInput),
			ParentNodes: dependencyMap[node.ID],
			OutputKeys:  node.Outputs,
		})
	}
	finalNodeID := strings.TrimSpace(dag.OutputNodeID)
	if finalNodeID == "" {
		finalNodeID = orderedNodes[len(orderedNodes)-1].ID
	}
	finalOutput, ok := nodeOutputs[finalNodeID]
	if !ok {
		return nil, nil, executions, &weaverRunFailure{
			Scope:     "dag",
			Code:      "dag_output_node_not_found",
			NodeID:    finalNodeID,
			Attempt:   attempt,
			Error:     "dag.output_node_id not found",
			Retryable: false,
		}
	}
	return nodeOutputs, finalOutput, executions, nil
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

func normalizeRunRetryPolicy(policy weaverRunRetryPolicy) weaverRunRetryPolicy {
	maxAttempts := policy.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	if maxAttempts > 5 {
		maxAttempts = 5
	}
	backoffInitial := policy.BackoffInitialMs
	if backoffInitial < 0 {
		backoffInitial = 0
	}
	backoffMultiplier := policy.BackoffMultiplier
	if backoffMultiplier < 1 {
		backoffMultiplier = 2
	}
	backoffMax := policy.BackoffMaxMs
	if backoffMax < backoffInitial {
		backoffMax = backoffInitial
	}
	retryableCodes := normalizeRetryableCodes(policy.RetryableCodes)
	return weaverRunRetryPolicy{
		MaxAttempts:        maxAttempts,
		RetryOnNodeError:   policy.RetryOnNodeError,
		RetryOnSourceError: policy.RetryOnSourceError,
		RetryableCodes:     retryableCodes,
		BackoffInitialMs:   backoffInitial,
		BackoffMultiplier:  backoffMultiplier,
		BackoffMaxMs:       backoffMax,
	}
}

func buildRunRetrySnapshot(policy weaverRunRetryPolicy, used int, delays []int64) weaverRunRetrySnapshot {
	normalized := normalizeRunRetryPolicy(policy)
	return weaverRunRetrySnapshot{
		MaxAttempts:       normalized.MaxAttempts,
		Used:              used,
		Triggered:         used > 1,
		RetryableCodes:    normalized.RetryableCodes,
		BackoffInitialMs:  normalized.BackoffInitialMs,
		BackoffMultiplier: normalized.BackoffMultiplier,
		BackoffMaxMs:      normalized.BackoffMaxMs,
		DelaysMs:          delays,
	}
}

func shouldRetryRun(failure weaverRunFailure, policy weaverRunRetryPolicy, attempt int) bool {
	if attempt >= policy.MaxAttempts {
		return false
	}
	if !failure.Retryable {
		return false
	}
	if len(policy.RetryableCodes) > 0 {
		return slices.Contains(policy.RetryableCodes, strings.ToLower(strings.TrimSpace(failure.Code)))
	}
	if failure.Scope == "node" && policy.RetryOnNodeError {
		return true
	}
	if failure.Scope == "source" && policy.RetryOnSourceError {
		return true
	}
	return false
}

func buildSourceFailure(sources []weaverRunSource, attempt int) *weaverRunFailure {
	for _, item := range sources {
		if item.Ok {
			continue
		}
		return &weaverRunFailure{
			Scope:     "source",
			Code:      sourceFailureCode(item),
			Source:    item.Name,
			Attempt:   attempt,
			Error:     item.Error,
			Retryable: true,
		}
	}
	return nil
}

func evaluateNodeFailure(nodeID string, config map[string]any, attempt int) (string, string, bool) {
	rawText, hasText := config["simulate_error"]
	if hasText {
		if failureText, ok := rawText.(string); ok && strings.TrimSpace(failureText) != "" {
			return "node_simulated_error", failureText, true
		}
	}
	failUntilText, hasFailUntil := config["fail_until_attempt"]
	if !hasFailUntil {
		return "", "", false
	}
	failUntil, ok := numberToInt(failUntilText)
	if !ok || failUntil <= 0 {
		return "", "", false
	}
	if attempt > failUntil {
		return "", "", false
	}
	return "node_attempt_guard", fmt.Sprintf("node %s simulated failure at attempt %d", nodeID, attempt), true
}

func numberToInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int8:
		return int(typed), true
	case int16:
		return int(typed), true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float32:
		return int(typed), true
	case float64:
		return int(typed), true
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil {
			return 0, false
		}
		return int(parsed), true
	default:
		return 0, false
	}
}

func mapKeys(items map[string]any) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	return keys
}

func inferRunStatus(attempts []weaverRunAttempt) string {
	if len(attempts) == 0 {
		return "failed"
	}
	lastAttempt := attempts[len(attempts)-1]
	if lastAttempt.Status == "succeeded" {
		return "succeeded"
	}
	return "failed"
}

func sourceFailureCode(source weaverRunSource) string {
	code := strings.ToLower(strings.TrimSpace(source.Code))
	if code != "" {
		return code
	}
	return "source_invalid"
}

func normalizeRetryableCodes(codes []string) []string {
	used := make(map[string]struct{}, len(codes))
	items := make([]string, 0, len(codes))
	for _, code := range codes {
		normalized := strings.ToLower(strings.TrimSpace(code))
		if normalized == "" {
			continue
		}
		if _, exists := used[normalized]; exists {
			continue
		}
		used[normalized] = struct{}{}
		items = append(items, normalized)
	}
	return items
}

func buildBackoffDelayMs(policy weaverRunRetryPolicy, attempt int) int64 {
	if policy.BackoffInitialMs <= 0 {
		return 0
	}
	delay := float64(policy.BackoffInitialMs)
	for idx := 1; idx < attempt; idx++ {
		delay *= policy.BackoffMultiplier
	}
	if policy.BackoffMaxMs > 0 && delay > float64(policy.BackoffMaxMs) {
		delay = float64(policy.BackoffMaxMs)
	}
	if delay < 0 {
		return 0
	}
	return int64(math.Round(delay))
}

func classifyDAGErrorCode(err error) string {
	if err == nil {
		return "dag_execute_failed"
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	if msg == "dag has cycle" {
		return "dag_cycle"
	}
	if strings.Contains(msg, "node not found") {
		return "dag_edge_node_not_found"
	}
	if strings.Contains(msg, "is required") || strings.Contains(msg, "is duplicated") || strings.Contains(msg, " is empty") {
		return "dag_schema_invalid"
	}
	return "dag_execute_failed"
}
