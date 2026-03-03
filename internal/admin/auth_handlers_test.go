package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"YoBFF/internal/logging"

	"go.uber.org/zap"
)

type fakeCaptchaService struct {
	mu    sync.Mutex
	next  int
	codes map[string]string
}

// Generate 生成测试用验证码，固定答案为 1234。
// 参数：preset 为验证码预设。
// 返回：id 为验证码标识，imageBase64 为占位图片内容。
// 异常：无。
func (f *fakeCaptchaService) Generate(_ string) (string, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.codes == nil {
		f.codes = make(map[string]string)
	}
	f.next++
	id := fmt.Sprintf("id-%d", f.next)
	f.codes[id] = "1234"
	return id, "ZmFrZQ==", nil
}

// Verify 校验测试用验证码并销毁答案。
// 参数：id 为验证码标识，code 为用户输入。
// 返回：true 表示校验通过。
// 异常：无。
func (f *fakeCaptchaService) Verify(id string, code string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	expect, ok := f.codes[id]
	if !ok {
		return false
	}
	if expect != code {
		return false
	}
	delete(f.codes, id)
	return true
}

// TestLogin_CaptchaAfterFirstFailure 验证首次失败后后续登录必须带验证码。
func TestLogin_CaptchaAfterFirstFailure(t *testing.T) {
	t.Setenv("ADMIN_API_TOKEN", "token")
	t.Setenv("ADMIN_RATE_LIMIT_PER_MIN", "60")

	manager := buildManagerForTest(t, `{}`)
	runtime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化日志运行时失败: %v", err)
	}
	pipeline := logging.NewPipeline(16, zap.NewNop())
	defer pipeline.Close()

	srv := NewServer(manager, runtime, pipeline)
	srv.captcha = &fakeCaptchaService{}
	srv.guard = newLoginGuard(time.Hour)
	handler := srv.Handler()

	req1 := httptest.NewRequest(http.MethodPost, "http://example.com/api/v1/login", bytes.NewReader([]byte(`{"username":"admin","password":"bad"}`)))
	req1.RemoteAddr = "127.0.0.1:1234"
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	if rec1.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("状态码不匹配: got=%d", rec1.Result().StatusCode)
	}

	req2 := httptest.NewRequest(http.MethodPost, "http://example.com/api/v1/login", bytes.NewReader([]byte(`{"username":"admin","password":"change_me"}`)))
	req2.RemoteAddr = "127.0.0.1:1234"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("状态码不匹配: got=%d", rec2.Result().StatusCode)
	}
	var got2 errorResponse
	_ = json.NewDecoder(rec2.Result().Body).Decode(&got2)
	if got2.ErrorCode != "captcha_required" {
		t.Fatalf("错误码不匹配: got=%q", got2.ErrorCode)
	}

	req3 := httptest.NewRequest(http.MethodGet, "http://example.com/api/v1/captcha", nil)
	req3.RemoteAddr = "127.0.0.1:1234"
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	if rec3.Result().StatusCode != http.StatusOK {
		t.Fatalf("状态码不匹配: got=%d", rec3.Result().StatusCode)
	}
	var captchaResp map[string]string
	_ = json.NewDecoder(rec3.Result().Body).Decode(&captchaResp)
	captchaID := captchaResp["captcha_id"]
	if captchaID == "" {
		t.Fatalf("captcha_id 为空")
	}

	body4 := fmt.Sprintf(`{"username":"admin","password":"change_me","captcha_id":"%s","captcha_code":"1234"}`, captchaID)
	req4 := httptest.NewRequest(http.MethodPost, "http://example.com/api/v1/login", bytes.NewReader([]byte(body4)))
	req4.RemoteAddr = "127.0.0.1:1234"
	rec4 := httptest.NewRecorder()
	handler.ServeHTTP(rec4, req4)
	if rec4.Result().StatusCode != http.StatusOK {
		t.Fatalf("状态码不匹配: got=%d", rec4.Result().StatusCode)
	}
	var ok4 map[string]string
	_ = json.NewDecoder(rec4.Result().Body).Decode(&ok4)
	if ok4["token"] != "token" {
		t.Fatalf("token 不匹配: got=%q", ok4["token"])
	}

	req5 := httptest.NewRequest(http.MethodPost, "http://example.com/api/v1/login", bytes.NewReader([]byte(`{"username":"admin","password":"change_me"}`)))
	req5.RemoteAddr = "127.0.0.1:1234"
	rec5 := httptest.NewRecorder()
	handler.ServeHTTP(rec5, req5)
	if rec5.Result().StatusCode != http.StatusOK {
		t.Fatalf("状态码不匹配: got=%d", rec5.Result().StatusCode)
	}
}

// TestLogin_InvalidCaptcha 验证验证码错误时返回固定错误码。
func TestLogin_InvalidCaptcha(t *testing.T) {
	t.Setenv("ADMIN_API_TOKEN", "token")
	t.Setenv("ADMIN_RATE_LIMIT_PER_MIN", "60")

	manager := buildManagerForTest(t, `{}`)
	runtime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化日志运行时失败: %v", err)
	}
	pipeline := logging.NewPipeline(16, zap.NewNop())
	defer pipeline.Close()

	srv := NewServer(manager, runtime, pipeline)
	srv.captcha = &fakeCaptchaService{}
	srv.guard = newLoginGuard(time.Hour)
	handler := srv.Handler()

	req1 := httptest.NewRequest(http.MethodPost, "http://example.com/api/v1/login", bytes.NewReader([]byte(`{"username":"admin","password":"bad"}`)))
	req1.RemoteAddr = "127.0.0.1:1234"
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	if rec1.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("状态码不匹配: got=%d", rec1.Result().StatusCode)
	}

	req2 := httptest.NewRequest(http.MethodGet, "http://example.com/api/v1/captcha", nil)
	req2.RemoteAddr = "127.0.0.1:1234"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Result().StatusCode != http.StatusOK {
		t.Fatalf("状态码不匹配: got=%d", rec2.Result().StatusCode)
	}
	var captchaResp map[string]string
	_ = json.NewDecoder(rec2.Result().Body).Decode(&captchaResp)
	captchaID := captchaResp["captcha_id"]
	if captchaID == "" {
		t.Fatalf("captcha_id 为空")
	}

	body3 := fmt.Sprintf(`{"username":"admin","password":"change_me","captcha_id":"%s","captcha_code":"0000"}`, captchaID)
	req3 := httptest.NewRequest(http.MethodPost, "http://example.com/api/v1/login", bytes.NewReader([]byte(body3)))
	req3.RemoteAddr = "127.0.0.1:1234"
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	if rec3.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("状态码不匹配: got=%d", rec3.Result().StatusCode)
	}
	var got3 errorResponse
	_ = json.NewDecoder(rec3.Result().Body).Decode(&got3)
	if got3.ErrorCode != "invalid_captcha" {
		t.Fatalf("错误码不匹配: got=%q", got3.ErrorCode)
	}
}

// TestCaptcha_PresetEcho 验证验证码接口支持预设切换参数。
func TestCaptcha_PresetEcho(t *testing.T) {
	t.Setenv("ADMIN_API_TOKEN", "token")
	t.Setenv("ADMIN_RATE_LIMIT_PER_MIN", "60")

	manager := buildManagerForTest(t, `{}`)
	runtime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化日志运行时失败: %v", err)
	}
	pipeline := logging.NewPipeline(16, zap.NewNop())
	defer pipeline.Close()

	srv := NewServer(manager, runtime, pipeline)
	srv.captcha = &fakeCaptchaService{}
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/v1/captcha?preset=easy", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Result().StatusCode != http.StatusOK {
		t.Fatalf("状态码不匹配: got=%d", rec.Result().StatusCode)
	}
	var got map[string]string
	_ = json.NewDecoder(rec.Result().Body).Decode(&got)
	if got["preset"] != "easy" {
		t.Fatalf("preset 不匹配: got=%q", got["preset"])
	}
	if got["captcha_id"] == "" || got["image_base64"] == "" {
		t.Fatalf("响应字段缺失: %#v", got)
	}
}
