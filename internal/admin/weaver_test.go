package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestWeaverDraftRoutes_CRUDAndRun 验证 Weaver 草稿的创建、查询、更新、运行与列表链路。
func TestWeaverDraftRoutes_CRUDAndRun(t *testing.T) {
	handler, _ := buildSiteHandler(t)
	createBody := []byte(`{
		"name":"order-flow",
		"inputs":[{"name":"order","payload":{"id":"o-1001","amount":99}}],
		"mapping":{"result":"$.order.id"}
	}`)

	createRec := requestWeaverWithToken(t, handler, http.MethodPost, "/api/v1/weaver/drafts", createBody)
	if createRec.Code != http.StatusOK {
		t.Fatalf("创建草稿失败: status=%d body=%s", createRec.Code, createRec.Body.String())
	}
	created := decodeWeaverPayload(t, createRec)
	draftID, _ := created["id"].(string)
	if draftID == "" {
		t.Fatalf("草稿ID为空: %#v", created)
	}
	if operator, _ := created["operator"].(string); operator != "admin" {
		t.Fatalf("操作人不匹配: got=%q", operator)
	}

	detailRec := requestWeaverWithToken(t, handler, http.MethodGet, "/api/v1/weaver/drafts/"+draftID, nil)
	if detailRec.Code != http.StatusOK {
		t.Fatalf("读取草稿失败: status=%d body=%s", detailRec.Code, detailRec.Body.String())
	}

	updateBody := []byte(`{
		"name":"order-flow-v2",
		"inputs":[{"name":"order","payload":{"id":"o-1002","amount":188}}],
		"mapping":{"result":"$.order.amount"}
	}`)
	updateRec := requestWeaverWithToken(t, handler, http.MethodPut, "/api/v1/weaver/drafts/"+draftID, updateBody)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("更新草稿失败: status=%d body=%s", updateRec.Code, updateRec.Body.String())
	}

	runBody := []byte(`{
		"inputs":[{"name":"order","payload":{"id":"o-1003","amount":288}}],
		"mapping":{"result":"$.order.amount"}
	}`)
	runRec := requestWeaverWithToken(t, handler, http.MethodPost, "/api/v1/weaver/drafts/"+draftID+"/run", runBody)
	if runRec.Code != http.StatusOK {
		t.Fatalf("运行草稿失败: status=%d body=%s", runRec.Code, runRec.Body.String())
	}
	runPayload := decodeWeaverPayload(t, runRec)
	if _, ok := runPayload["duration_ms"].(float64); !ok {
		t.Fatalf("运行结果缺少 duration_ms: %#v", runPayload)
	}
	sources, ok := runPayload["sources"].([]any)
	if !ok || len(sources) == 0 {
		t.Fatalf("运行结果 sources 为空: %#v", runPayload)
	}

	listRec := requestWeaverWithToken(t, handler, http.MethodGet, "/api/v1/weaver/drafts?limit=10", nil)
	if listRec.Code != http.StatusOK {
		t.Fatalf("读取草稿列表失败: status=%d body=%s", listRec.Code, listRec.Body.String())
	}
	listPayload := decodeWeaverPayload(t, listRec)
	items, ok := listPayload["items"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("草稿列表为空: %#v", listPayload)
	}

	deleteRec := requestWeaverWithToken(t, handler, http.MethodDelete, "/api/v1/weaver/drafts/"+draftID, nil)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("删除草稿失败: status=%d body=%s", deleteRec.Code, deleteRec.Body.String())
	}
	deletePayload := decodeWeaverPayload(t, deleteRec)
	if status, _ := deletePayload["status"].(string); status != "deleted" {
		t.Fatalf("删除结果状态异常: %#v", deletePayload)
	}

	detailAfterDelete := requestWeaverWithToken(t, handler, http.MethodGet, "/api/v1/weaver/drafts/"+draftID, nil)
	if detailAfterDelete.Code != http.StatusNotFound {
		t.Fatalf("删除后读取应为404: status=%d body=%s", detailAfterDelete.Code, detailAfterDelete.Body.String())
	}
}

// TestWeaverDraftRoutes_InvalidPayloads 验证 Weaver 草稿接口的异常路径与错误码行为。
func TestWeaverDraftRoutes_InvalidPayloads(t *testing.T) {
	handler, _ := buildSiteHandler(t)

	testCases := []struct {
		name       string
		method     string
		target     string
		body       []byte
		statusCode int
		errorCode  string
	}{
		{
			name:       "创建草稿缺少name",
			method:     http.MethodPost,
			target:     "/api/v1/weaver/drafts",
			body:       []byte(`{"inputs":[{"name":"a","payload":{}}],"mapping":{}}`),
			statusCode: http.StatusBadRequest,
			errorCode:  "invalid_request",
		},
		{
			name:       "运行草稿ID不存在",
			method:     http.MethodPost,
			target:     "/api/v1/weaver/drafts/not-exist/run",
			body:       []byte(`{}`),
			statusCode: http.StatusNotFound,
			errorCode:  "draft_not_found",
		},
		{
			name:       "删除草稿ID不存在",
			method:     http.MethodDelete,
			target:     "/api/v1/weaver/drafts/not-exist",
			body:       nil,
			statusCode: http.StatusNotFound,
			errorCode:  "draft_not_found",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			rec := requestWeaverWithToken(t, handler, tc.method, tc.target, tc.body)
			if rec.Code != tc.statusCode {
				t.Fatalf("状态码不匹配: got=%d want=%d body=%s", rec.Code, tc.statusCode, rec.Body.String())
			}
			payload := decodeWeaverPayload(t, rec)
			got, _ := payload["error_code"].(string)
			if got != tc.errorCode {
				t.Fatalf("错误码不匹配: got=%q want=%q", got, tc.errorCode)
			}
		})
	}
}

// requestWeaverWithToken 构造带鉴权信息的 Weaver 请求并返回响应记录器。
func requestWeaverWithToken(
	t *testing.T,
	handler http.Handler,
	method string,
	target string,
	body []byte,
) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, "http://example.com"+target, bytes.NewReader(body))
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-Operator", "tester")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// decodeWeaverPayload 解析 Weaver 接口响应为 map 结构，便于断言字段。
func decodeWeaverPayload(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var payload map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	return payload
}
