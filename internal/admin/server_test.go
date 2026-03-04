package admin

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"YoBFF/internal/config"
	"YoBFF/internal/logging"

	"go.uber.org/zap"
)

// TestWithAuth_TokenMissingReturns503 验证未配置 Token 时返回 503。
func TestWithAuth_TokenMissingReturns503(t *testing.T) {
	manager := buildManagerForTest(t, `{"controlPlane": {"auth": {"token": ""}}}`)
	handler := withAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("不应放行")
	}), manager)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("状态码不匹配: got=%d", rec.Result().StatusCode)
	}
	var got errorResponse
	_ = json.NewDecoder(rec.Result().Body).Decode(&got)
	if got.ErrorCode != "auth_not_configured" {
		t.Fatalf("错误码不匹配: got=%q", got.ErrorCode)
	}
	if got.RequestID == "" {
		t.Fatalf("request_id 为空")
	}
}

// TestWithAuth_MissingHeaderReturns401 验证缺少 Authorization 时返回 401。
func TestWithAuth_MissingHeaderReturns401(t *testing.T) {
	manager := buildManagerForTest(t, `{"controlPlane": {"auth": {"token": "token"}}}`)
	handler := withAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("不应放行")
	}), manager)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("状态码不匹配: got=%d", rec.Result().StatusCode)
	}
}

// TestWithAuth_InvalidSchemeReturns401 验证非 Bearer 方案时返回 401。
func TestWithAuth_InvalidSchemeReturns401(t *testing.T) {
	manager := buildManagerForTest(t, `{"controlPlane": {"auth": {"token": "token"}}}`)
	handler := withAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("不应放行")
	}), manager)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/healthz", nil)
	req.Header.Set("Authorization", "Basic token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("状态码不匹配: got=%d", rec.Result().StatusCode)
	}
}

// TestWithAuth_WrongTokenReturns401 验证 Token 不匹配时返回 401。
func TestWithAuth_WrongTokenReturns401(t *testing.T) {
	manager := buildManagerForTest(t, `{"controlPlane": {"auth": {"token": "token"}}}`)
	handler := withAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("不应放行")
	}), manager)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/healthz", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("状态码不匹配: got=%d", rec.Result().StatusCode)
	}
}

// TestWithAuth_CorrectTokenPasses 验证 Bearer Token 正确时放行请求。
func TestWithAuth_CorrectTokenPasses(t *testing.T) {
	var hit bool
	manager := buildManagerForTest(t, `{"controlPlane": {"auth": {"token": "token"}}}`)
	handler := withAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusOK)
	}), manager)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/healthz", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusOK {
		t.Fatalf("状态码不匹配: got=%d", rec.Result().StatusCode)
	}
	if !hit {
		t.Fatalf("未命中 next")
	}
}

// TestWithRequestID_PreservesExisting 验证请求携带 request_id 时透传且写回响应头。
func TestWithRequestID_PreservesExisting(t *testing.T) {
	handler := withRequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requestIDFromContext(r.Context()) != "fixed" {
			t.Fatalf("上下文 request_id 不匹配")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.com/healthz", nil)
	req.Header.Set("X-Request-ID", "fixed")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Result().Header.Get("X-Request-ID") != "fixed" {
		t.Fatalf("响应头 request_id 不匹配: got=%q", rec.Result().Header.Get("X-Request-ID"))
	}
}

// TestWriteError_SetsRequestIDHeaderWhenMissing 验证错误响应会补齐 request_id 并写入响应头。
func TestWriteError_SetsRequestIDHeaderWhenMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/healthz", nil)
	rec := httptest.NewRecorder()
	writeError(rec, http.StatusBadRequest, "bad", "bad", req)

	res := rec.Result()
	if res.Header.Get("X-Request-ID") == "" {
		t.Fatalf("未设置 X-Request-ID")
	}
	var got errorResponse
	_ = json.NewDecoder(res.Body).Decode(&got)
	if got.RequestID == "" || got.RequestID != res.Header.Get("X-Request-ID") {
		t.Fatalf("响应体 request_id 不匹配: header=%q body=%q", res.Header.Get("X-Request-ID"), got.RequestID)
	}
}

// TestWithRateLimit_LimitsRequestsPerClient 验证固定窗口限流按客户端维度生效。
func TestWithRateLimit_LimitsRequestsPerClient(t *testing.T) {
	limiter := newRateLimiter(2, time.Minute)
	handler := withRateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), limiter)

	doRequest := func() int {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/healthz", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Result().StatusCode
	}

	if status := doRequest(); status != http.StatusOK {
		t.Fatalf("第 1 次请求状态码不匹配: got=%d", status)
	}
	if status := doRequest(); status != http.StatusOK {
		t.Fatalf("第 2 次请求状态码不匹配: got=%d", status)
	}
	if status := doRequest(); status != http.StatusTooManyRequests {
		t.Fatalf("第 3 次请求状态码不匹配: got=%d", status)
	}
}

// TestClientKey_PrefersForwardedHeaders 验证限流客户端标识优先使用 X-Forwarded-For。
func TestClientKey_PrefersForwardedHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/healthz", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	req.Header.Set("X-Real-IP", "10.0.0.9")

	if key := clientKey(req); key != "10.0.0.1" {
		t.Fatalf("clientKey 不匹配: got=%q", key)
	}
}

// TestParsePositiveInt 验证正整数解析的兜底行为。
func TestParsePositiveInt(t *testing.T) {
	if got := parsePositiveInt("", 60); got != 60 {
		t.Fatalf("解析结果不匹配: got=%d", got)
	}
	if got := parsePositiveInt("abc", 60); got != 60 {
		t.Fatalf("解析结果不匹配: got=%d", got)
	}
	if got := parsePositiveInt("-1", 60); got != 60 {
		t.Fatalf("解析结果不匹配: got=%d", got)
	}
	if got := parsePositiveInt("10", 60); got != 10 {
		t.Fatalf("解析结果不匹配: got=%d", got)
	}
}

// TestServerHandler_HealthzOK 验证 NewServer/Handler 的健康检查端点可用。
func TestServerHandler_HealthzOK(t *testing.T) {
	t.Setenv("ADMIN_API_TOKEN", "token")
	t.Setenv("ADMIN_RATE_LIMIT_PER_MIN", "60")

	manager := buildManagerForTest(t, `{}`)
	logRuntime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化日志运行时失败: %v", err)
	}
	pipeline := logging.NewPipeline(16, zap.NewNop())
	defer pipeline.Close()

	srv := NewServer(manager, logRuntime, pipeline, nil)
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/healthz", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("状态码不匹配: got=%d", res.StatusCode)
	}
	var payload map[string]any
	_ = json.NewDecoder(res.Body).Decode(&payload)
	if payload["status"] != "ok" {
		t.Fatalf("响应体不匹配: got=%v", payload)
	}
}

// TestServerHandler_ConfigGetPut 验证配置查询与在线更新端点可用。
func TestServerHandler_ConfigGetPut(t *testing.T) {
	t.Setenv("ADMIN_API_TOKEN", "token")
	t.Setenv("ADMIN_RATE_LIMIT_PER_MIN", "60")

	manager := buildManagerForTest(t, `{}`)
	logRuntime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化日志运行时失败: %v", err)
	}
	pipeline := logging.NewPipeline(16, zap.NewNop())
	defer pipeline.Close()

	srv := NewServer(manager, logRuntime, pipeline, nil)
	handler := srv.Handler()

	getReq := httptest.NewRequest(http.MethodGet, "http://example.com/api/v1/config", nil)
	getReq.RemoteAddr = "127.0.0.1:1234"
	getReq.Header.Set("Authorization", "Bearer token")
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)
	if getRec.Result().StatusCode != http.StatusOK {
		t.Fatalf("GET 状态码不匹配: got=%d", getRec.Result().StatusCode)
	}

	putBody := []byte(`{"dataPlane":{"httpListenAddr":":18080"},"security":{"blockPageHtml":"<html/>"},"routing":{"defaultUpstream":"http://127.0.0.1:18080"}}`)
	putReq := httptest.NewRequest(http.MethodPut, "http://example.com/api/v1/config", bytes.NewReader(putBody))
	putReq.RemoteAddr = "127.0.0.1:1234"
	putReq.Header.Set("Authorization", "Bearer token")
	putRec := httptest.NewRecorder()
	handler.ServeHTTP(putRec, putReq)
	if putRec.Result().StatusCode != http.StatusOK {
		body, _ := io.ReadAll(putRec.Result().Body)
		t.Fatalf("PUT 状态码不匹配: got=%d body=%s", putRec.Result().StatusCode, string(body))
	}

	updated := manager.CurrentConfig()
	if updated.DataPlane.HTTPListenAddr != ":18080" {
		t.Fatalf("配置未更新: got=%q", updated.DataPlane.HTTPListenAddr)
	}
}

// TestServerHandler_Reload 验证配置热重载端点可以从磁盘重新加载。
func TestServerHandler_Reload(t *testing.T) {
	t.Setenv("ADMIN_API_TOKEN", "token")
	t.Setenv("ADMIN_RATE_LIMIT_PER_MIN", "60")

	cfgV1 := `{
  "dataPlane": { "httpListenAddr": ":8080", "httpsListenAddr": ":8443", "enableHttps": false },
  "controlPlane": { "adminListenAddr": ":9090" },
  "security": { "allowedCidrs": [], "blockPageHtml": "<html/>", "enableHsts": false },
  "routing": { "defaultUpstream": "http://127.0.0.1:18080", "domains": [] },
  "certificates": []
}`
	manager := buildManagerForTest(t, cfgV1)

	logRuntime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化日志运行时失败: %v", err)
	}
	pipeline := logging.NewPipeline(16, zap.NewNop())
	defer pipeline.Close()

	srv := NewServer(manager, logRuntime, pipeline, nil)
	handler := srv.Handler()

	cfgV2 := `{
  "dataPlane": { "httpListenAddr": ":18080", "httpsListenAddr": ":8443", "enableHttps": false },
  "controlPlane": { "adminListenAddr": ":9090" },
  "security": { "allowedCidrs": [], "blockPageHtml": "<html/>", "enableHsts": false },
  "routing": { "defaultUpstream": "http://127.0.0.1:18080", "domains": [] },
  "certificates": []
}`
	if err = os.WriteFile(manager.ConfigPath(), []byte(cfgV2), 0o600); err != nil {
		t.Fatalf("写入配置文件失败: %v", err)
	}

	reloadReq := httptest.NewRequest(http.MethodPost, "http://example.com/api/v1/config/reload", nil)
	reloadReq.RemoteAddr = "127.0.0.1:1234"
	reloadReq.Header.Set("Authorization", "Bearer token")
	reloadRec := httptest.NewRecorder()
	handler.ServeHTTP(reloadRec, reloadReq)
	if reloadRec.Result().StatusCode != http.StatusOK {
		t.Fatalf("状态码不匹配: got=%d", reloadRec.Result().StatusCode)
	}
	if manager.CurrentConfig().DataPlane.HTTPListenAddr != ":18080" {
		t.Fatalf("热重载未生效: got=%q", manager.CurrentConfig().DataPlane.HTTPListenAddr)
	}
}

// TestServerHandler_LogLevelAndStats 验证日志级别与统计端点返回结构正确。
func TestServerHandler_LogLevelAndStats(t *testing.T) {
	t.Setenv("ADMIN_API_TOKEN", "token")
	t.Setenv("ADMIN_RATE_LIMIT_PER_MIN", "60")
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("LOG_STACKTRACE_LEVEL", "error")

	manager := buildManagerForTest(t, `{}`)
	logRuntime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化日志运行时失败: %v", err)
	}
	pipeline := logging.NewPipeline(16, zap.NewNop())
	defer pipeline.Close()
	pipeline.Emit(logging.Event{Level: "info", Type: "proxy"})

	srv := NewServer(manager, logRuntime, pipeline, nil)
	handler := srv.Handler()

	levelReq := httptest.NewRequest(http.MethodGet, "http://example.com/api/v1/log/level", nil)
	levelReq.RemoteAddr = "127.0.0.1:1234"
	levelReq.Header.Set("Authorization", "Bearer token")
	levelRec := httptest.NewRecorder()
	handler.ServeHTTP(levelRec, levelReq)
	if levelRec.Result().StatusCode != http.StatusOK {
		t.Fatalf("状态码不匹配: got=%d", levelRec.Result().StatusCode)
	}

	putReq := httptest.NewRequest(http.MethodPut, "http://example.com/api/v1/log/level", bytes.NewReader([]byte(`{"level":"debug"}`)))
	putReq.RemoteAddr = "127.0.0.1:1234"
	putReq.Header.Set("Authorization", "Bearer token")
	putRec := httptest.NewRecorder()
	handler.ServeHTTP(putRec, putReq)
	if putRec.Result().StatusCode != http.StatusOK {
		t.Fatalf("状态码不匹配: got=%d", putRec.Result().StatusCode)
	}

	statsReq := httptest.NewRequest(http.MethodGet, "http://example.com/api/v1/log/stats", nil)
	statsReq.RemoteAddr = "127.0.0.1:1234"
	statsReq.Header.Set("Authorization", "Bearer token")
	statsRec := httptest.NewRecorder()
	handler.ServeHTTP(statsRec, statsReq)
	if statsRec.Result().StatusCode != http.StatusOK {
		t.Fatalf("状态码不匹配: got=%d", statsRec.Result().StatusCode)
	}
	var stats logging.Stats
	_ = json.NewDecoder(statsRec.Result().Body).Decode(&stats)
	if stats.Total == 0 {
		t.Fatalf("统计未返回: %+v", stats)
	}
}

// TestServerHandler_RateLimitOnHandler 验证路由处理器链路会执行限流。
func TestServerHandler_RateLimitOnHandler(t *testing.T) {
	t.Setenv("ADMIN_API_TOKEN", "token")
	t.Setenv("ADMIN_RATE_LIMIT_PER_MIN", "2")

	manager := buildManagerForTest(t, `{}`)
	logRuntime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化日志运行时失败: %v", err)
	}
	pipeline := logging.NewPipeline(16, zap.NewNop())
	defer pipeline.Close()

	srv := NewServer(manager, logRuntime, pipeline, nil)
	handler := srv.Handler()

	doRequest := func() int {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/healthz", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		req.Header.Set("Authorization", "Bearer token")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Result().StatusCode
	}

	if got := doRequest(); got != http.StatusOK {
		t.Fatalf("第 1 次请求状态码不匹配: got=%d", got)
	}
	if got := doRequest(); got != http.StatusOK {
		t.Fatalf("第 2 次请求状态码不匹配: got=%d", got)
	}
	if got := doRequest(); got != http.StatusTooManyRequests {
		t.Fatalf("第 3 次请求状态码不匹配: got=%d", got)
	}
}

// TestServerHandler_Config_MethodNotAllowed 验证配置接口不支持的方法返回 405。
func TestServerHandler_Config_MethodNotAllowed(t *testing.T) {
	t.Setenv("ADMIN_API_TOKEN", "token")
	t.Setenv("ADMIN_RATE_LIMIT_PER_MIN", "60")

	manager := buildManagerForTest(t, `{}`)
	logRuntime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化日志运行时失败: %v", err)
	}
	pipeline := logging.NewPipeline(16, zap.NewNop())
	defer pipeline.Close()

	srv := NewServer(manager, logRuntime, pipeline, nil)
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/v1/config", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("状态码不匹配: got=%d", rec.Result().StatusCode)
	}
	var payload errorResponse
	_ = json.NewDecoder(rec.Result().Body).Decode(&payload)
	if payload.ErrorCode != "method_not_allowed" {
		t.Fatalf("错误码不匹配: got=%q", payload.ErrorCode)
	}
}

// TestServerHandler_Config_InvalidRequest 验证配置更新请求体非法时返回 400。
func TestServerHandler_Config_InvalidRequest(t *testing.T) {
	t.Setenv("ADMIN_API_TOKEN", "token")
	t.Setenv("ADMIN_RATE_LIMIT_PER_MIN", "60")

	manager := buildManagerForTest(t, `{}`)
	logRuntime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化日志运行时失败: %v", err)
	}
	pipeline := logging.NewPipeline(16, zap.NewNop())
	defer pipeline.Close()

	srv := NewServer(manager, logRuntime, pipeline, nil)
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodPut, "http://example.com/api/v1/config", bytes.NewReader([]byte(`{"unknown":1}`)))
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("状态码不匹配: got=%d", rec.Result().StatusCode)
	}
	var payload errorResponse
	_ = json.NewDecoder(rec.Result().Body).Decode(&payload)
	if payload.ErrorCode != "invalid_request" {
		t.Fatalf("错误码不匹配: got=%q", payload.ErrorCode)
	}
}

// TestServerHandler_Config_Invalid 验证配置预检失败时返回 400。
func TestServerHandler_Config_Invalid(t *testing.T) {
	t.Setenv("ADMIN_API_TOKEN", "token")
	t.Setenv("ADMIN_RATE_LIMIT_PER_MIN", "60")

	manager := buildManagerForTest(t, `{}`)
	logRuntime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化日志运行时失败: %v", err)
	}
	pipeline := logging.NewPipeline(16, zap.NewNop())
	defer pipeline.Close()

	srv := NewServer(manager, logRuntime, pipeline, nil)
	handler := srv.Handler()

	body := []byte(`{"security":{"allowedCidrs":["not-a-cidr"],"blockPageHtml":"<html/>"},"routing":{"defaultUpstream":"http://127.0.0.1:18080"}}`)
	req := httptest.NewRequest(http.MethodPut, "http://example.com/api/v1/config", bytes.NewReader(body))
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("状态码不匹配: got=%d", rec.Result().StatusCode)
	}
	var payload errorResponse
	_ = json.NewDecoder(rec.Result().Body).Decode(&payload)
	if payload.ErrorCode != "config_invalid" {
		t.Fatalf("错误码不匹配: got=%q", payload.ErrorCode)
	}
}

// TestServerHandler_Reload_MethodNotAllowed 验证热重载端点不支持的方法返回 405。
func TestServerHandler_Reload_MethodNotAllowed(t *testing.T) {
	t.Setenv("ADMIN_API_TOKEN", "token")
	t.Setenv("ADMIN_RATE_LIMIT_PER_MIN", "60")

	manager := buildManagerForTest(t, `{}`)
	logRuntime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化日志运行时失败: %v", err)
	}
	pipeline := logging.NewPipeline(16, zap.NewNop())
	defer pipeline.Close()

	srv := NewServer(manager, logRuntime, pipeline, nil)
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/v1/config/reload", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("状态码不匹配: got=%d", rec.Result().StatusCode)
	}
}

// TestServerHandler_Reload_Failed 验证热重载失败时返回 400。
func TestServerHandler_Reload_Failed(t *testing.T) {
	t.Setenv("ADMIN_API_TOKEN", "token")
	t.Setenv("ADMIN_RATE_LIMIT_PER_MIN", "60")

	manager := buildManagerForTest(t, `{}`)
	logRuntime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化日志运行时失败: %v", err)
	}
	pipeline := logging.NewPipeline(16, zap.NewNop())
	defer pipeline.Close()

	srv := NewServer(manager, logRuntime, pipeline, nil)
	handler := srv.Handler()

	if err = os.WriteFile(manager.ConfigPath(), []byte(`{invalid json`), 0o600); err != nil {
		t.Fatalf("写入配置文件失败: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/v1/config/reload", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("状态码不匹配: got=%d", rec.Result().StatusCode)
	}
	var payload errorResponse
	_ = json.NewDecoder(rec.Result().Body).Decode(&payload)
	if payload.ErrorCode != "config_reload_failed" {
		t.Fatalf("错误码不匹配: got=%q", payload.ErrorCode)
	}
}

// TestServerHandler_LogLevel_RuntimeUnavailable 验证日志运行时不可用时返回 503。
func TestServerHandler_LogLevel_RuntimeUnavailable(t *testing.T) {
	t.Setenv("ADMIN_API_TOKEN", "token")
	t.Setenv("ADMIN_RATE_LIMIT_PER_MIN", "60")

	manager := buildManagerForTest(t, `{}`)
	pipeline := logging.NewPipeline(16, zap.NewNop())
	defer pipeline.Close()

	srv := NewServer(manager, nil, pipeline, nil)
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/v1/log/level", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("状态码不匹配: got=%d", rec.Result().StatusCode)
	}
	var payload errorResponse
	_ = json.NewDecoder(rec.Result().Body).Decode(&payload)
	if payload.ErrorCode != "logger_unavailable" {
		t.Fatalf("错误码不匹配: got=%q", payload.ErrorCode)
	}
}

// TestServerHandler_LogLevel_InvalidRequest 验证日志级别更新请求体非法时返回 400。
func TestServerHandler_LogLevel_InvalidRequest(t *testing.T) {
	t.Setenv("ADMIN_API_TOKEN", "token")
	t.Setenv("ADMIN_RATE_LIMIT_PER_MIN", "60")
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("LOG_STACKTRACE_LEVEL", "error")

	manager := buildManagerForTest(t, `{}`)
	logRuntime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化日志运行时失败: %v", err)
	}
	pipeline := logging.NewPipeline(16, zap.NewNop())
	defer pipeline.Close()

	srv := NewServer(manager, logRuntime, pipeline, nil)
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodPut, "http://example.com/api/v1/log/level", bytes.NewReader([]byte(`{"unknown":1}`)))
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("状态码不匹配: got=%d", rec.Result().StatusCode)
	}
	var payload errorResponse
	_ = json.NewDecoder(rec.Result().Body).Decode(&payload)
	if payload.ErrorCode != "invalid_request" {
		t.Fatalf("错误码不匹配: got=%q", payload.ErrorCode)
	}
}

// TestServerHandler_LogLevel_InvalidValue 验证不支持的日志级别返回 400。
func TestServerHandler_LogLevel_InvalidValue(t *testing.T) {
	t.Setenv("ADMIN_API_TOKEN", "token")
	t.Setenv("ADMIN_RATE_LIMIT_PER_MIN", "60")
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("LOG_STACKTRACE_LEVEL", "error")

	manager := buildManagerForTest(t, `{}`)
	logRuntime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化日志运行时失败: %v", err)
	}
	pipeline := logging.NewPipeline(16, zap.NewNop())
	defer pipeline.Close()

	srv := NewServer(manager, logRuntime, pipeline, nil)
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodPut, "http://example.com/api/v1/log/level", bytes.NewReader([]byte(`{"level":"nope"}`)))
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("状态码不匹配: got=%d", rec.Result().StatusCode)
	}
	var payload errorResponse
	_ = json.NewDecoder(rec.Result().Body).Decode(&payload)
	if payload.ErrorCode != "log_level_invalid" {
		t.Fatalf("错误码不匹配: got=%q", payload.ErrorCode)
	}
}

// TestServerHandler_LogLevel_MethodNotAllowed 验证日志级别接口不支持的方法返回 405。
func TestServerHandler_LogLevel_MethodNotAllowed(t *testing.T) {
	t.Setenv("ADMIN_API_TOKEN", "token")
	t.Setenv("ADMIN_RATE_LIMIT_PER_MIN", "60")
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("LOG_STACKTRACE_LEVEL", "error")

	manager := buildManagerForTest(t, `{}`)
	logRuntime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化日志运行时失败: %v", err)
	}
	pipeline := logging.NewPipeline(16, zap.NewNop())
	defer pipeline.Close()

	srv := NewServer(manager, logRuntime, pipeline, nil)
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/v1/log/level", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("状态码不匹配: got=%d", rec.Result().StatusCode)
	}
}

// TestServerHandler_LogStats_MethodNotAllowed 验证日志统计接口不支持的方法返回 405。
func TestServerHandler_LogStats_MethodNotAllowed(t *testing.T) {
	t.Setenv("ADMIN_API_TOKEN", "token")
	t.Setenv("ADMIN_RATE_LIMIT_PER_MIN", "60")

	manager := buildManagerForTest(t, `{}`)
	logRuntime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化日志运行时失败: %v", err)
	}
	pipeline := logging.NewPipeline(16, zap.NewNop())
	defer pipeline.Close()

	srv := NewServer(manager, logRuntime, pipeline, nil)
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodPost, "http://example.com/api/v1/log/stats", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("状态码不匹配: got=%d", rec.Result().StatusCode)
	}
}

// TestServerHandler_LogStats_PipelineUnavailable 验证日志管线不可用时返回 503。
func TestServerHandler_LogStats_PipelineUnavailable(t *testing.T) {
	t.Setenv("ADMIN_API_TOKEN", "token")
	t.Setenv("ADMIN_RATE_LIMIT_PER_MIN", "60")

	manager := buildManagerForTest(t, `{}`)
	logRuntime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化日志运行时失败: %v", err)
	}

	srv := NewServer(manager, logRuntime, nil, nil)
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/v1/log/stats", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("状态码不匹配: got=%d", rec.Result().StatusCode)
	}
	var payload errorResponse
	_ = json.NewDecoder(rec.Result().Body).Decode(&payload)
	if payload.ErrorCode != "log_pipeline_unavailable" {
		t.Fatalf("错误码不匹配: got=%q", payload.ErrorCode)
	}
}

// TestRequestIDFromContext_NonString 验证上下文 request_id 类型非法时返回空串。
func TestRequestIDFromContext_NonString(t *testing.T) {
	ctx := context.WithValue(context.Background(), requestIDKey, 123)
	if got := requestIDFromContext(ctx); got != "" {
		t.Fatalf("requestIDFromContext 不匹配: got=%q", got)
	}
}

type failingReader struct{}

func (f failingReader) Read(_ []byte) (int, error) {
	return 0, errors.New("fail")
}

// TestNewRequestID_Fallback 验证随机数读取失败时 request_id 仍可生成。
func TestNewRequestID_Fallback(t *testing.T) {
	orig := rand.Reader
	rand.Reader = failingReader{}
	defer func() { rand.Reader = orig }()

	if got := newRequestID(); got == "" {
		t.Fatalf("request_id 为空")
	}
}

// TestClientKey_CoversFallback 验证客户端标识的回退路径。
func TestClientKey_CoversFallback(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/healthz", nil)
	req.RemoteAddr = ""
	if got := clientKey(req); got != "unknown" {
		t.Fatalf("clientKey 不匹配: got=%q", got)
	}

	req = httptest.NewRequest(http.MethodGet, "http://example.com/healthz", nil)
	req.RemoteAddr = "127.0.0.1"
	if got := clientKey(req); got != "127.0.0.1" {
		t.Fatalf("clientKey 不匹配: got=%q", got)
	}

	req = httptest.NewRequest(http.MethodGet, "http://example.com/healthz", nil)
	req.RemoteAddr = ""
	req.Header.Set("X-Forwarded-For", ", 10.0.0.2")
	req.Header.Set("X-Real-IP", "10.0.0.9")
	if got := clientKey(req); got != "10.0.0.9" {
		t.Fatalf("clientKey 不匹配: got=%q", got)
	}
}

// buildManagerForTest 构建用于单元测试的配置管理器，并写入初始配置文件。
// 参数：t 为测试上下文，content 为配置 JSON 文本。
// 返回：初始化完成的配置管理器实例。
// 异常：写入或初始化失败时终止当前测试。
func buildManagerForTest(t *testing.T, content string) *config.Manager {
	t.Helper()

	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, "config.json")
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("写入配置文件失败: %v", err)
	}
	manager, err := config.NewManager(configPath)
	if err != nil {
		t.Fatalf("初始化 Manager 失败: %v", err)
	}
	return manager
}
