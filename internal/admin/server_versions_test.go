package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"YoBFF/internal/config"
	"YoBFF/internal/logging"
	"YoBFF/internal/store"

	"go.uber.org/zap"
)

func buildVersionHandler(t *testing.T) (http.Handler, *config.Manager, *store.Store) {
	t.Helper()
	t.Setenv("ADMIN_API_TOKEN", "token")
	t.Setenv("ADMIN_RATE_LIMIT_PER_MIN", "60")
	manager := buildManagerForTest(t, `{}`)
	logRuntime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化日志运行时失败: %v", err)
	}
	pipeline := logging.NewPipeline(16, zap.NewNop())
	t.Cleanup(func() {
		pipeline.Close()
	})
	dbPath := filepath.Join(t.TempDir(), "admin-version.db")
	siteStore, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("初始化存储失败: %v", err)
	}
	t.Cleanup(func() {
		_ = siteStore.Close()
	})
	srv := NewServer(manager, logRuntime, pipeline, siteStore)
	return srv.Handler(), manager, siteStore
}

func requestVersionAPI(t *testing.T, handler http.Handler, method string, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "http://example.com"+path, bytes.NewReader(body))
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-Admin-User", "tester")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestServerHandler_ConfigVersionsAndRollback(t *testing.T) {
	handler, manager, siteStore := buildVersionHandler(t)
	cfg := manager.CurrentConfig()
	cfg.Security.BlockPageHTML = "<html>v1</html>"
	saved, err := siteStore.SaveVersion(cfg, "tester", "init")
	if err != nil {
		t.Fatalf("保存配置版本失败: %v", err)
	}

	validateRec := requestVersionAPI(t, handler, http.MethodPost, "/api/v1/config/validate", []byte(`{}`))
	if validateRec.Result().StatusCode != http.StatusOK {
		t.Fatalf("配置预检状态码不匹配: got=%d", validateRec.Result().StatusCode)
	}
	var validatePayload map[string]any
	if err = json.NewDecoder(validateRec.Result().Body).Decode(&validatePayload); err != nil {
		t.Fatalf("解析预检响应失败: %v", err)
	}
	if _, ok := validatePayload["valid"]; !ok {
		t.Fatalf("预检响应缺少 valid 字段: %#v", validatePayload)
	}

	versionsRec := requestVersionAPI(t, handler, http.MethodGet, "/api/v1/config/versions?limit=5", nil)
	if versionsRec.Result().StatusCode != http.StatusOK {
		t.Fatalf("配置版本列表状态码不匹配: got=%d", versionsRec.Result().StatusCode)
	}

	rollbackRec := requestVersionAPI(t, handler, http.MethodPost, "/api/v1/config/rollback", []byte(`{"version_id":"`+saved.ID+`"}`))
	if rollbackRec.Result().StatusCode != http.StatusOK {
		t.Fatalf("配置回滚状态码不匹配: got=%d", rollbackRec.Result().StatusCode)
	}

	invalidRollbackRec := requestVersionAPI(t, handler, http.MethodPost, "/api/v1/config/rollback", []byte(`{"version_id":""}`))
	if invalidRollbackRec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("空版本ID状态码不匹配: got=%d", invalidRollbackRec.Result().StatusCode)
	}
}

func TestServerHandler_ConfigValidateAndVersionsErrorPaths(t *testing.T) {
	handler, _, siteStore := buildVersionHandler(t)

	cases := []struct {
		name string
		want int
		path string
		body []byte
	}{
		{
			name: "validate method not allowed",
			want: http.StatusMethodNotAllowed,
			path: "/api/v1/config/validate",
			body: nil,
		},
		{
			name: "validate invalid body",
			want: http.StatusBadRequest,
			path: "/api/v1/config/validate",
			body: []byte(`{"unknown":1}`),
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			method := http.MethodGet
			if tt.body != nil {
				method = http.MethodPost
			}
			rec := requestVersionAPI(t, handler, method, tt.path, tt.body)
			if rec.Result().StatusCode != tt.want {
				t.Fatalf("状态码不匹配: got=%d want=%d body=%s", rec.Result().StatusCode, tt.want, rec.Body.String())
			}
		})
	}

	methodRec := requestVersionAPI(t, handler, http.MethodPost, "/api/v1/config/versions", nil)
	if methodRec.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("版本列表方法状态码不匹配: got=%d", methodRec.Result().StatusCode)
	}

	if err := siteStore.Close(); err != nil {
		t.Fatalf("关闭存储失败: %v", err)
	}
	storeQueryRec := requestVersionAPI(t, handler, http.MethodGet, "/api/v1/config/versions?limit=5", nil)
	if storeQueryRec.Result().StatusCode != http.StatusInternalServerError {
		t.Fatalf("存储查询失败状态码不匹配: got=%d body=%s", storeQueryRec.Result().StatusCode, storeQueryRec.Body.String())
	}
}

func TestServerHandler_RollbackErrorPaths(t *testing.T) {
	handler, _, siteStore := buildVersionHandler(t)

	methodRec := requestVersionAPI(t, handler, http.MethodGet, "/api/v1/config/rollback", nil)
	if methodRec.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("方法状态码不匹配: got=%d", methodRec.Result().StatusCode)
	}

	invalidBodyRec := requestVersionAPI(t, handler, http.MethodPost, "/api/v1/config/rollback", []byte(`{"unknown":1}`))
	if invalidBodyRec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("请求体状态码不匹配: got=%d", invalidBodyRec.Result().StatusCode)
	}

	notFoundRec := requestVersionAPI(t, handler, http.MethodPost, "/api/v1/config/rollback", []byte(`{"version_id":"not-found"}`))
	if notFoundRec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("版本不存在状态码不匹配: got=%d", notFoundRec.Result().StatusCode)
	}

	cfg := buildManagerForTest(t, `{}`).CurrentConfig()
	cfg.Routing.DefaultUpstream = "://bad"
	version, err := siteStore.SaveVersion(cfg, "tester", "invalid")
	if err != nil {
		t.Fatalf("保存版本失败: %v", err)
	}
	invalidCfgRec := requestVersionAPI(
		t,
		handler,
		http.MethodPost,
		"/api/v1/config/rollback",
		[]byte(`{"version_id":"`+version.ID+`"}`),
	)
	if invalidCfgRec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("非法配置状态码不匹配: got=%d body=%s", invalidCfgRec.Result().StatusCode, invalidCfgRec.Body.String())
	}

	if err = siteStore.Close(); err != nil {
		t.Fatalf("关闭存储失败: %v", err)
	}
	storeUnavailableRec := requestVersionAPI(
		t,
		handler,
		http.MethodPost,
		"/api/v1/config/rollback",
		[]byte(`{"version_id":"`+version.ID+`"}`),
	)
	if storeUnavailableRec.Result().StatusCode != http.StatusBadRequest &&
		storeUnavailableRec.Result().StatusCode != http.StatusServiceUnavailable &&
		storeUnavailableRec.Result().StatusCode != http.StatusInternalServerError {
		t.Fatalf("关闭存储后状态码异常: got=%d body=%s", storeUnavailableRec.Result().StatusCode, storeUnavailableRec.Body.String())
	}
}

func TestServerHandler_ConfigVersionStoreUnavailable(t *testing.T) {
	t.Setenv("ADMIN_API_TOKEN", "token")
	t.Setenv("ADMIN_RATE_LIMIT_PER_MIN", "60")

	manager := buildManagerForTest(t, `{}`)
	logRuntime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化日志运行时失败: %v", err)
	}
	pipeline := logging.NewPipeline(16, zap.NewNop())
	t.Cleanup(func() {
		pipeline.Close()
	})

	srv := NewServer(manager, logRuntime, pipeline, nil)
	handler := srv.Handler()

	versionsRec := requestVersionAPI(t, handler, http.MethodGet, "/api/v1/config/versions", nil)
	if versionsRec.Result().StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("版本列表存储不可用状态码不匹配: got=%d", versionsRec.Result().StatusCode)
	}

	rollbackRec := requestVersionAPI(t, handler, http.MethodPost, "/api/v1/config/rollback", []byte(`{"version_id":"x"}`))
	if rollbackRec.Result().StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("回滚存储不可用状态码不匹配: got=%d", rollbackRec.Result().StatusCode)
	}
}

func TestServerHandler_ConfigPutVersionFailed(t *testing.T) {
	t.Setenv("ADMIN_API_TOKEN", "token")
	t.Setenv("ADMIN_RATE_LIMIT_PER_MIN", "60")

	manager := buildManagerForTest(t, `{}`)
	logRuntime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化日志运行时失败: %v", err)
	}
	pipeline := logging.NewPipeline(16, zap.NewNop())
	t.Cleanup(func() {
		pipeline.Close()
	})

	dbPath := filepath.Join(t.TempDir(), "admin-version-put.db")
	siteStore, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("初始化存储失败: %v", err)
	}
	if err = siteStore.Close(); err != nil {
		t.Fatalf("关闭存储失败: %v", err)
	}

	srv := NewServer(manager, logRuntime, pipeline, siteStore)
	handler := srv.Handler()
	body := []byte(`{"security":{"blockPageHtml":"<html/>"},"routing":{"defaultUpstream":"http://127.0.0.1:18080"}}`)
	rec := requestVersionAPI(t, handler, http.MethodPut, "/api/v1/config", body)
	if rec.Result().StatusCode != http.StatusInternalServerError {
		t.Fatalf("配置版本保存失败状态码不匹配: got=%d body=%s", rec.Result().StatusCode, rec.Body.String())
	}
}

func TestServerHandler_CDNConfigErrorPaths(t *testing.T) {
	handler, _, _ := buildVersionHandler(t)

	methodRec := requestVersionAPI(t, handler, http.MethodPost, "/api/v1/config/cdn", nil)
	if methodRec.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("方法状态码不匹配: got=%d", methodRec.Result().StatusCode)
	}

	badBodyRec := requestVersionAPI(t, handler, http.MethodPut, "/api/v1/config/cdn", []byte(`{"unknown":1}`))
	if badBodyRec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("请求体状态码不匹配: got=%d", badBodyRec.Result().StatusCode)
	}
}

func TestOperatorFromRequest_UsesFallback(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	req.Header.Set("X-Operator", " ")
	if got := operatorFromRequest(req); got != "unknown" {
		t.Fatalf("操作人回退不匹配: got=%q", got)
	}
	req.Header.Set("X-Operator", "alice")
	if got := operatorFromRequest(req); got != "alice" {
		t.Fatalf("操作人读取不匹配: got=%q", got)
	}
}

func TestWriteValidationError_RequestIDPaths(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://example.com", nil)
	ctxReq := req.WithContext(withRequestIDContext(req.Context(), "req-1"))
	writeValidationError(rec, http.StatusBadRequest, "config_invalid", "bad", nil, ctxReq)
	if rec.Header().Get("X-Request-ID") != "" {
		t.Fatalf("已有 request_id 不应覆盖响应头")
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "http://example.com", nil)
	writeValidationError(rec2, http.StatusBadRequest, "config_invalid", "bad", nil, req2)
	if rec2.Header().Get("X-Request-ID") == "" {
		t.Fatalf("缺失 request_id 时应注入响应头")
	}
}

func withRequestIDContext(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}
