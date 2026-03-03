package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestEnvOrDefault_UsesFallbackWhenEmpty 验证环境变量为空时回退默认值。
func TestEnvOrDefault_UsesFallbackWhenEmpty(t *testing.T) {
	t.Setenv("ENV_OR_DEFAULT_TEST", "")
	if got := EnvOrDefault("ENV_OR_DEFAULT_TEST", "fallback"); got != "fallback" {
		t.Fatalf("返回值不匹配: got=%q", got)
	}
}

// TestEnvOrDefault_UsesValueWhenPresent 验证环境变量有值时返回该值。
func TestEnvOrDefault_UsesValueWhenPresent(t *testing.T) {
	t.Setenv("ENV_OR_DEFAULT_TEST", "value")
	if got := EnvOrDefault("ENV_OR_DEFAULT_TEST", "fallback"); got != "value" {
		t.Fatalf("返回值不匹配: got=%q", got)
	}
}

// TestEnvDurationSeconds_FallbackOnInvalid 验证秒数环境变量非法时回退默认值。
func TestEnvDurationSeconds_FallbackOnInvalid(t *testing.T) {
	t.Setenv("ENV_DURATION_TEST", "abc")
	if got := EnvDurationSeconds("ENV_DURATION_TEST", 3); got != 3*time.Second {
		t.Fatalf("返回值不匹配: got=%s", got)
	}
}

// TestApplyEnvOverrides_SplitsAllowedCIDRs 验证 ALLOWED_CIDRS 会按逗号拆分并去除空项。
func TestApplyEnvOverrides_SplitsAllowedCIDRs(t *testing.T) {
	t.Setenv("HTTP_LISTEN_ADDR", ":18080")
	t.Setenv("HTTPS_LISTEN_ADDR", ":18443")
	t.Setenv("HTTPS_ENABLED", "false")
	t.Setenv("ALLOWED_CIDRS", " 127.0.0.1/32, , ::1/128 ")

	cfg := defaultConfig()
	cfg.Routing.DefaultUpstream = "http://127.0.0.1:18080"
	next := ApplyEnvOverrides(cfg)

	if next.DataPlane.HTTPListenAddr != ":18080" {
		t.Fatalf("HTTP_LISTEN_ADDR 未覆盖: got=%q", next.DataPlane.HTTPListenAddr)
	}
	if next.DataPlane.HTTPSListenAddr != ":18443" {
		t.Fatalf("HTTPS_LISTEN_ADDR 未覆盖: got=%q", next.DataPlane.HTTPSListenAddr)
	}
	if next.DataPlane.EnableHTTPS {
		t.Fatalf("HTTPS_ENABLED 未覆盖: got=true")
	}
	if len(next.Security.AllowedCIDRs) != 2 {
		t.Fatalf("AllowedCIDRs 数量不匹配: got=%d", len(next.Security.AllowedCIDRs))
	}
	if next.Security.AllowedCIDRs[0] != "127.0.0.1/32" || next.Security.AllowedCIDRs[1] != "::1/128" {
		t.Fatalf("AllowedCIDRs 内容不匹配: got=%v", next.Security.AllowedCIDRs)
	}
}

// TestLoadDotEnv_MissingFileReturnsNil 验证 .env 文件不存在时返回 nil。
func TestLoadDotEnv_MissingFileReturnsNil(t *testing.T) {
	baseDir := t.TempDir()
	path := filepath.Join(baseDir, "missing.env")
	if err := LoadDotEnv(path); err != nil {
		t.Fatalf("LoadDotEnv 不应返回错误: %v", err)
	}
}

// TestLoadDotEnv_LoadsFile 验证 .env 文件存在时会加载到进程环境变量。
func TestLoadDotEnv_LoadsFile(t *testing.T) {
	baseDir := t.TempDir()
	path := filepath.Join(baseDir, ".env")
	if err := os.WriteFile(path, []byte("LOAD_DOTENV_TEST=value\n"), 0o600); err != nil {
		t.Fatalf("写入 .env 失败: %v", err)
	}
	_ = os.Unsetenv("LOAD_DOTENV_TEST")
	t.Cleanup(func() {
		_ = os.Unsetenv("LOAD_DOTENV_TEST")
	})

	if err := LoadDotEnv(path); err != nil {
		t.Fatalf("LoadDotEnv 失败: %v", err)
	}
	if got := os.Getenv("LOAD_DOTENV_TEST"); got != "value" {
		t.Fatalf("环境变量未加载: got=%q", got)
	}
}

// TestEnvDurationSeconds_FallbackOnEmptyOrNonPositive 验证为空或非正数时回退默认值。
func TestEnvDurationSeconds_FallbackOnEmptyOrNonPositive(t *testing.T) {
	t.Setenv("ENV_DURATION_TEST", "")
	if got := EnvDurationSeconds("ENV_DURATION_TEST", 3); got != 3*time.Second {
		t.Fatalf("返回值不匹配: got=%s", got)
	}

	t.Setenv("ENV_DURATION_TEST", "0")
	if got := EnvDurationSeconds("ENV_DURATION_TEST", 3); got != 3*time.Second {
		t.Fatalf("返回值不匹配: got=%s", got)
	}

	t.Setenv("ENV_DURATION_TEST", "-1")
	if got := EnvDurationSeconds("ENV_DURATION_TEST", 3); got != 3*time.Second {
		t.Fatalf("返回值不匹配: got=%s", got)
	}
}

// TestApplyEnvOverrides_BoolParsing 验证布尔环境变量多种取值的解析行为。
func TestApplyEnvOverrides_BoolParsing(t *testing.T) {
	t.Setenv("HTTPS_ENABLED", "on")
	t.Setenv("HSTS_ENABLED", "unknown")
	cfg := defaultConfig()
	cfg.DataPlane.EnableHTTPS = false
	cfg.Security.EnableHSTS = true
	next := ApplyEnvOverrides(cfg)

	if !next.DataPlane.EnableHTTPS {
		t.Fatalf("HTTPS_ENABLED 解析失败: got=false")
	}
	if !next.Security.EnableHSTS {
		t.Fatalf("HSTS_ENABLED 非法值应回退: got=false")
	}
}
