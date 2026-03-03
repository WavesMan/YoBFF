package logging

import (
	"os"
	"testing"
)

// TestNewRuntimeFromEnv_InvalidLevel 验证非法日志级别会返回错误。
func TestNewRuntimeFromEnv_InvalidLevel(t *testing.T) {
	t.Setenv("LOG_LEVEL", "not-supported")
	t.Setenv("LOG_STACKTRACE_LEVEL", "error")

	if _, err := NewRuntimeFromEnv(); err == nil {
		t.Fatalf("非法 LOG_LEVEL 应返回错误")
	}
}

// TestRuntime_LevelAndSetLevel 验证日志级别读取与动态更新行为。
func TestRuntime_LevelAndSetLevel(t *testing.T) {
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("LOG_STACKTRACE_LEVEL", "error")

	runtime, err := NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化 Runtime 失败: %v", err)
	}
	if runtime.Level() == "" {
		t.Fatalf("Level 为空")
	}
	if err = runtime.SetLevel("debug"); err != nil {
		t.Fatalf("SetLevel 失败: %v", err)
	}
	if got := runtime.Level(); got != "debug" {
		t.Fatalf("Level 不匹配: got=%q", got)
	}
	if err = runtime.SetLevel("not-supported"); err == nil {
		t.Fatalf("SetLevel 应返回错误")
	}
}

// TestRuntime_LoggerAndSyncNilReceiver 验证 nil 接收者下的安全行为。
func TestRuntime_LoggerAndSyncNilReceiver(t *testing.T) {
	var runtime *Runtime
	_ = runtime.Logger()
	runtime.Sync()
}

// TestParseLevel_CoversBranches 验证 parseLevel 支持的分支。
func TestParseLevel_CoversBranches(t *testing.T) {
	cases := []struct {
		input string
		ok    bool
	}{
		{input: "debug", ok: true},
		{input: "info", ok: true},
		{input: "warn", ok: true},
		{input: "warning", ok: true},
		{input: "error", ok: true},
		{input: "dpanic", ok: true},
		{input: "panic", ok: true},
		{input: "fatal", ok: true},
		{input: "unknown", ok: false},
	}
	for _, tc := range cases {
		_, err := parseLevel(tc.input)
		if tc.ok && err != nil {
			t.Fatalf("parseLevel 应成功: input=%q err=%v", tc.input, err)
		}
		if !tc.ok && err == nil {
			t.Fatalf("parseLevel 应失败: input=%q", tc.input)
		}
	}
}

// TestEnvOrDefault_CoversBranches 验证 envOrDefault 的兜底逻辑。
func TestEnvOrDefault_CoversBranches(t *testing.T) {
	_ = os.Unsetenv("LOGGING_ENV_TEST")
	if got := envOrDefault("LOGGING_ENV_TEST", "fallback"); got != "fallback" {
		t.Fatalf("返回值不匹配: got=%q", got)
	}
	t.Setenv("LOGGING_ENV_TEST", "value")
	if got := envOrDefault("LOGGING_ENV_TEST", "fallback"); got != "value" {
		t.Fatalf("返回值不匹配: got=%q", got)
	}
}
