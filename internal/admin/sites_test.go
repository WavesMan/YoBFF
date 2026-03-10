package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"YoBFF/internal/logging"
	"YoBFF/internal/store"

	"go.uber.org/zap"
)

func buildSiteHandler(t *testing.T) (http.Handler, *store.Store) {
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
	dbPath := filepath.Join(t.TempDir(), "admin-site.db")
	siteStore, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("初始化站点存储失败: %v", err)
	}
	t.Cleanup(func() {
		_ = siteStore.Close()
	})
	srv := NewServer(manager, logRuntime, pipeline, siteStore)
	return srv.Handler(), siteStore
}

func requestWithToken(t *testing.T, handler http.Handler, method string, target string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "http://example.com"+target, bytes.NewReader(body))
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-Admin-User", "tester")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func decodeMap(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.NewDecoder(rec.Result().Body).Decode(&payload); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	return payload
}

func TestSiteRoutes_CRUDAndGroups(t *testing.T) {
	handler, _ := buildSiteHandler(t)

	createBody := []byte(`{"name":"site-a","hostname":"a.example.com","ip":"10.10.1.1"}`)
	createRec := requestWithToken(t, handler, http.MethodPost, "/api/v1/sites", createBody)
	if createRec.Result().StatusCode != http.StatusOK {
		t.Fatalf("创建站点状态码不匹配: got=%d", createRec.Result().StatusCode)
	}
	createPayload := decodeMap(t, createRec)
	siteID, _ := createPayload["id"].(string)
	if siteID == "" {
		t.Fatalf("站点ID为空: %#v", createPayload)
	}

	listRec := requestWithToken(t, handler, http.MethodGet, "/api/v1/sites", nil)
	if listRec.Result().StatusCode != http.StatusOK {
		t.Fatalf("查询站点列表状态码不匹配: got=%d", listRec.Result().StatusCode)
	}

	groupRec := requestWithToken(t, handler, http.MethodGet, "/api/v1/site-groups?by=hostname", nil)
	if groupRec.Result().StatusCode != http.StatusOK {
		t.Fatalf("按主机分组状态码不匹配: got=%d", groupRec.Result().StatusCode)
	}
	groupPayload := decodeMap(t, groupRec)
	groups, ok := groupPayload["groups"].([]any)
	if !ok || len(groups) == 0 {
		t.Fatalf("分组结果为空: %#v", groupPayload)
	}

	updateBody := []byte(`{"name":"site-a-2","hostname":"a2.example.com","ip":"10.10.1.2"}`)
	updateRec := requestWithToken(t, handler, http.MethodPut, "/api/v1/sites/"+siteID, updateBody)
	if updateRec.Result().StatusCode != http.StatusOK {
		t.Fatalf("更新站点状态码不匹配: got=%d", updateRec.Result().StatusCode)
	}

	detailRec := requestWithToken(t, handler, http.MethodGet, "/api/v1/sites/"+siteID, nil)
	if detailRec.Result().StatusCode != http.StatusOK {
		t.Fatalf("查询站点详情状态码不匹配: got=%d", detailRec.Result().StatusCode)
	}

	groupByIPRec := requestWithToken(t, handler, http.MethodGet, "/api/v1/site-groups?by=ip", nil)
	if groupByIPRec.Result().StatusCode != http.StatusOK {
		t.Fatalf("按IP分组状态码不匹配: got=%d", groupByIPRec.Result().StatusCode)
	}

	deleteRec := requestWithToken(t, handler, http.MethodDelete, "/api/v1/sites/"+siteID, nil)
	if deleteRec.Result().StatusCode != http.StatusOK {
		t.Fatalf("删除站点状态码不匹配: got=%d", deleteRec.Result().StatusCode)
	}
}

func TestSiteRoutes_ConfigAndRollback(t *testing.T) {
	handler, siteStore := buildSiteHandler(t)
	site, err := siteStore.CreateSite(store.Site{
		Name:     "cfg",
		Hostname: "cfg.example.com",
		IP:       "10.0.0.10",
	})
	if err != nil {
		t.Fatalf("创建站点失败: %v", err)
	}

	getConfigRec := requestWithToken(t, handler, http.MethodGet, "/api/v1/sites/"+site.ID+"/config", nil)
	if getConfigRec.Result().StatusCode != http.StatusOK {
		t.Fatalf("读取站点配置状态码不匹配: got=%d", getConfigRec.Result().StatusCode)
	}

	validateRec := requestWithToken(t, handler, http.MethodPost, "/api/v1/sites/"+site.ID+"/config/validate", []byte(`{}`))
	if validateRec.Result().StatusCode != http.StatusOK {
		t.Fatalf("配置预检状态码不匹配: got=%d", validateRec.Result().StatusCode)
	}

	diffRec := requestWithToken(t, handler, http.MethodPost, "/api/v1/sites/"+site.ID+"/config/diff", []byte(`{}`))
	if diffRec.Result().StatusCode != http.StatusOK {
		t.Fatalf("配置差异状态码不匹配: got=%d", diffRec.Result().StatusCode)
	}

	putConfigRec := requestWithToken(t, handler, http.MethodPut, "/api/v1/sites/"+site.ID+"/config", []byte(`{}`))
	if putConfigRec.Result().StatusCode != http.StatusOK {
		t.Fatalf("更新站点配置状态码不匹配: got=%d", putConfigRec.Result().StatusCode)
	}

	versionsRec := requestWithToken(t, handler, http.MethodGet, "/api/v1/sites/"+site.ID+"/config/versions", nil)
	if versionsRec.Result().StatusCode != http.StatusOK {
		t.Fatalf("查询版本状态码不匹配: got=%d", versionsRec.Result().StatusCode)
	}
	versionsPayload := decodeMap(t, versionsRec)
	items, ok := versionsPayload["items"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("版本列表为空: %#v", versionsPayload)
	}
	first, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("版本结构错误: %#v", items[0])
	}
	versionID, _ := first["id"].(string)
	if versionID == "" {
		t.Fatalf("version_id 为空: %#v", first)
	}

	versionDetailRec := requestWithToken(t, handler, http.MethodGet, "/api/v1/sites/"+site.ID+"/config/versions/"+versionID, nil)
	if versionDetailRec.Result().StatusCode != http.StatusOK {
		t.Fatalf("查询版本详情状态码不匹配: got=%d", versionDetailRec.Result().StatusCode)
	}

	rollbackBody := []byte(`{"version_id":"` + versionID + `"}`)
	rollbackRec := requestWithToken(t, handler, http.MethodPost, "/api/v1/sites/"+site.ID+"/config/rollback", rollbackBody)
	if rollbackRec.Result().StatusCode != http.StatusOK {
		t.Fatalf("回滚状态码不匹配: got=%d", rollbackRec.Result().StatusCode)
	}
}

func TestSiteRoutes_LogAndErrorPaths(t *testing.T) {
	handler, siteStore := buildSiteHandler(t)
	site, err := siteStore.CreateSite(store.Site{
		Name:     "log",
		Hostname: "log.example.com",
		IP:       "10.0.0.11",
	})
	if err != nil {
		t.Fatalf("创建站点失败: %v", err)
	}

	notFoundLogRec := requestWithToken(t, handler, http.MethodGet, "/api/v1/sites/"+site.ID+"/log/stream", nil)
	if notFoundLogRec.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("未配置日志流状态码不匹配: got=%d", notFoundLogRec.Result().StatusCode)
	}

	updateLogRec := requestWithToken(t, handler, http.MethodPut, "/api/v1/sites/"+site.ID+"/log/stream", []byte(`{"filter_query":"status>=500"}`))
	if updateLogRec.Result().StatusCode != http.StatusOK {
		t.Fatalf("更新日志流状态码不匹配: got=%d", updateLogRec.Result().StatusCode)
	}

	getLogRec := requestWithToken(t, handler, http.MethodGet, "/api/v1/sites/"+site.ID+"/log/stream", nil)
	if getLogRec.Result().StatusCode != http.StatusOK {
		t.Fatalf("查询日志流状态码不匹配: got=%d", getLogRec.Result().StatusCode)
	}

	badGroupRec := requestWithToken(t, handler, http.MethodGet, "/api/v1/site-groups?by=bad", nil)
	if badGroupRec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("错误分组参数状态码不匹配: got=%d", badGroupRec.Result().StatusCode)
	}

	unknownRouteRec := requestWithToken(t, handler, http.MethodGet, "/api/v1/sites/"+site.ID+"/unknown", nil)
	if unknownRouteRec.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("未知路由状态码不匹配: got=%d", unknownRouteRec.Result().StatusCode)
	}
}
