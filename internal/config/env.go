package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// LoadDotEnv 用于按路径加载 .env 文件到进程环境变量。
// 参数：path 为 .env 文件路径。
// 返回：加载失败错误，文件不存在时返回 nil。
// 异常：当文件存在但格式非法时返回错误。
func LoadDotEnv(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	return godotenv.Load(path)
}

// EnvOrDefault 读取字符串环境变量并提供默认值。
// 参数：key 为变量名，fallback 为默认值。
// 返回：环境变量值或默认值。
// 异常：无。
func EnvOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

// EnvDurationSeconds 读取秒级整数环境变量并转换为 time.Duration。
// 参数：key 为变量名，fallbackSeconds 为兜底秒数。
// 返回：转换后的时长。
// 异常：当值非法时回退默认值。
func EnvDurationSeconds(key string, fallbackSeconds int) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return time.Duration(fallbackSeconds) * time.Second
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return time.Duration(fallbackSeconds) * time.Second
	}
	return time.Duration(seconds) * time.Second
}

// envBoolValue 读取布尔环境变量并兼容常见文本值。
// 参数：key 为变量名，fallback 为兜底值。
// 返回：解析后的布尔结果。
// 异常：值非法时回退默认值。
func envBoolValue(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

// ApplyEnvOverrides 将环境变量覆盖到配置对象中。
// 参数：cfg 为待覆盖的配置副本。
// 返回：覆盖后的配置。
// 异常：无。
func ApplyEnvOverrides(cfg Config) Config {
	cfg.DataPlane.HTTPListenAddr = EnvOrDefault("HTTP_LISTEN_ADDR", cfg.DataPlane.HTTPListenAddr)
	cfg.DataPlane.HTTPSListenAddr = EnvOrDefault("HTTPS_LISTEN_ADDR", cfg.DataPlane.HTTPSListenAddr)
	cfg.DataPlane.EnableHTTPS = envBoolValue("HTTPS_ENABLED", cfg.DataPlane.EnableHTTPS)
	cfg.ControlPlane.AdminListenAddr = EnvOrDefault("ADMIN_LISTEN_ADDR", cfg.ControlPlane.AdminListenAddr)
	cfg.ControlPlane.Auth.Username = EnvOrDefault("ADMIN_USERNAME", cfg.ControlPlane.Auth.Username)
	cfg.ControlPlane.Auth.Password = EnvOrDefault("ADMIN_PASSWORD", cfg.ControlPlane.Auth.Password)
	cfg.ControlPlane.Auth.Token = EnvOrDefault("ADMIN_API_TOKEN", cfg.ControlPlane.Auth.Token)
	cfg.Security.BlockPageHTML = EnvOrDefault("BLOCK_PAGE_HTML", cfg.Security.BlockPageHTML)
	cfg.Security.EnableHSTS = envBoolValue("HSTS_ENABLED", cfg.Security.EnableHSTS)
	cfg.Routing.DefaultUpstream = EnvOrDefault("DEFAULT_UPSTREAM", cfg.Routing.DefaultUpstream)

	allowedCIDRs := strings.TrimSpace(os.Getenv("ALLOWED_CIDRS"))
	if allowedCIDRs != "" {
		parts := strings.Split(allowedCIDRs, ",")
		next := make([]string, 0, len(parts))
		for _, part := range parts {
			value := strings.TrimSpace(part)
			if value != "" {
				next = append(next, value)
			}
		}
		cfg.Security.AllowedCIDRs = next
	}
	return cfg
}
