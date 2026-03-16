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
	writeError(w, http.StatusNotFound, "not_found", "not found", r)
}

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
	payload, err := decodeOptionalWeaverPayload(w, r)
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
	inputs, mapping, sources, err := resolveWeaverRunInputs(draft, payload)
	if err != nil {
		writeError(w, http.StatusBadRequest, "weaver_run_failed", err.Error(), r)
		return
	}
	output := map[string]any{
		"inputs":  inputs,
		"mapping": mapping,
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
	}, nil
}

func decodeOptionalWeaverPayload(w http.ResponseWriter, r *http.Request) (*weaverDraftPayload, error) {
	if r.ContentLength == 0 {
		return nil, nil
	}
	var payload weaverDraftPayload
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

func resolveWeaverRunInputs(
	draft store.WeaverDraft,
	payload *weaverDraftPayload,
) (map[string]any, any, []weaverRunSource, error) {
	var inputs []weaverInputPayload
	var mapping any
	if payload == nil || len(payload.Inputs) == 0 {
		if err := json.Unmarshal(draft.Inputs, &inputs); err != nil {
			return nil, nil, nil, err
		}
	} else {
		inputs = payload.Inputs
	}
	if payload == nil || len(payload.Mapping) == 0 {
		if err := json.Unmarshal(draft.Mapping, &mapping); err != nil {
			return nil, nil, nil, err
		}
	} else {
		if err := json.Unmarshal(payload.Mapping, &mapping); err != nil {
			return nil, nil, nil, err
		}
	}
	inputMap := make(map[string]any)
	var sources []weaverRunSource
	for _, item := range inputs {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			return nil, nil, nil, errMissingField("inputs.name")
		}
		if _, exists := inputMap[name]; exists {
			return nil, nil, nil, errDuplicateField("inputs.name")
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
	return inputMap, mapping, sources, nil
}

type requestFieldError struct {
	field string
}

func (e requestFieldError) Error() string {
	return e.field
}

func errMissingField(field string) error {
	return requestFieldError{field: field + " is required"}
}

func errDuplicateField(field string) error {
	return requestFieldError{field: field + " is duplicated"}
}
