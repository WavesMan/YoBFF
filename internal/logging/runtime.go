package logging

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewRuntimeFromEnv 用于构建基于环境变量的控制台日志运行时。
// 参数：无，读取 LOG_LEVEL 与 LOG_STACKTRACE_LEVEL 环境变量。
// 返回：日志运行时实例与初始化错误。
// 异常：当日志级别字符串非法时返回错误。
func NewRuntimeFromEnv() (*Runtime, error) {
	levelText := envOrDefault("LOG_LEVEL", "info")
	stacktraceLevelText := envOrDefault("LOG_STACKTRACE_LEVEL", "error")

	level, err := parseLevel(levelText)
	if err != nil {
		return nil, fmt.Errorf("LOG_LEVEL 配置非法: %w", err)
	}
	stacktraceLevel, err := parseLevel(stacktraceLevelText)
	if err != nil {
		return nil, fmt.Errorf("LOG_STACKTRACE_LEVEL 配置非法: %w", err)
	}

	atomicLevel := zap.NewAtomicLevelAt(level)
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.LevelKey = "level"
	encoderConfig.MessageKey = "msg"
	encoderConfig.CallerKey = "caller"
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeDuration = zapcore.StringDurationEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.Lock(os.Stdout),
		atomicLevel,
	)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(stacktraceLevel))

	return &Runtime{
		logger: logger,
		level:  atomicLevel,
	}, nil
}

// Runtime 封装日志实例与动态日志级别控制能力。
type Runtime struct {
	logger *zap.Logger
	level  zap.AtomicLevel
}

// Logger 返回底层 zap.Logger 供业务模块复用。
// 参数：无。
// 返回：可直接使用的日志对象。
// 异常：无。
func (r *Runtime) Logger() *zap.Logger {
	if r == nil || r.logger == nil {
		return zap.NewNop()
	}
	return r.logger
}

// SetLevel 在运行期更新日志输出级别。
// 参数：levelText 为目标级别，支持 debug/info/warn/error/dpanic/panic/fatal。
// 返回：更新失败时返回错误。
// 异常：当级别文本不受支持时返回错误。
func (r *Runtime) SetLevel(levelText string) error {
	nextLevel, err := parseLevel(levelText)
	if err != nil {
		return err
	}
	r.level.SetLevel(nextLevel)
	return nil
}

// Level 返回当前日志级别文本。
// 参数：无。
// 返回：当前运行级别字符串。
// 异常：无。
func (r *Runtime) Level() string {
	return r.level.Level().String()
}

// Sync 用于在进程退出前刷新缓冲日志。
// 参数：无。
// 返回：无。
// 异常：忽略输出目标不支持同步导致的错误。
func (r *Runtime) Sync() {
	if r == nil || r.logger == nil {
		return
	}
	_ = r.logger.Sync()
}

// parseLevel 用于将字符串解析为 zapcore.Level。
// 参数：levelText 为待解析的日志级别。
// 返回：解析后的级别和错误信息。
// 异常：当级别不支持时返回错误。
func parseLevel(levelText string) (zapcore.Level, error) {
	switch strings.ToLower(strings.TrimSpace(levelText)) {
	case "debug":
		return zap.DebugLevel, nil
	case "info":
		return zap.InfoLevel, nil
	case "warn", "warning":
		return zap.WarnLevel, nil
	case "error":
		return zap.ErrorLevel, nil
	case "dpanic":
		return zap.DPanicLevel, nil
	case "panic":
		return zap.PanicLevel, nil
	case "fatal":
		return zap.FatalLevel, nil
	default:
		return zap.InfoLevel, fmt.Errorf("不支持的日志级别: %s", levelText)
	}
}

// envOrDefault 用于读取环境变量并提供兜底值。
// 参数：key 为环境变量名，fallback 为默认值。
// 返回：环境变量值或默认值。
// 异常：无。
func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
