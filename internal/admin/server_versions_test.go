package admin

import (
	"bytes"
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
