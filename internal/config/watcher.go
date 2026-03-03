package config

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
)

// Watcher 通过轮询文件修改时间触发配置热重载。
type Watcher struct {
	manager  *Manager
	logger   *zap.Logger
	interval time.Duration
	lastMod  time.Time
}

// NewWatcher 构建配置文件监听器。
// 参数：manager 为配置管理器，logger 为日志实例，interval 为轮询周期。
// 返回：可启动的监听器对象与初始化错误。
// 异常：当配置文件不存在或不可访问时返回错误。
func NewWatcher(manager *Manager, logger *zap.Logger, interval time.Duration) (*Watcher, error) {
	if logger == nil {
		logger = zap.NewNop()
	}
	stat, err := os.Stat(manager.ConfigPath())
	if err != nil {
		return nil, fmt.Errorf("读取配置文件状态失败: %w", err)
	}
	return &Watcher{
		manager:  manager,
		logger:   logger,
		interval: interval,
		lastMod:  stat.ModTime(),
	}, nil
}

// Start 在后台协程中持续执行配置轮询。
// 参数：ctx 为生命周期控制上下文。
// 返回：无。
// 异常：无。
func (w *Watcher) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.tick()
			}
		}
	}()
}

// tick 执行单次文件变更检测与热重载。
// 参数：无。
// 返回：无。
// 异常：读取文件状态或重载失败时仅记录日志。
func (w *Watcher) tick() {
	stat, err := os.Stat(w.manager.ConfigPath())
	if err != nil {
		w.logger.Warn("读取配置文件状态失败", zap.Error(err))
		return
	}
	if !stat.ModTime().After(w.lastMod) {
		return
	}
	if err = w.manager.Reload(); err != nil {
		w.logger.Error("配置热重载失败", zap.Error(err))
		return
	}
	w.lastMod = stat.ModTime()
	w.logger.Info("配置热重载成功", zap.Time("updated_at", w.lastMod))
}
