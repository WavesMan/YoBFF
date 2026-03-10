package syncer

import (
	"context"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"YoBFF/internal/config"
	"YoBFF/internal/plugins/cdn"
	"YoBFF/internal/plugins/cdn/cloudflare"
)

// Service 负责定期同步 CDN 回源 IP 并更新到配置管理器。
type Service struct {
	manager *config.Manager
	logger  *zap.Logger
}

// NewService 创建同步服务实例。
// 参数：manager 为配置管理器，logger 为日志实例。
// 返回：同步服务实例。
func NewService(manager *config.Manager, logger *zap.Logger) *Service {
	return &Service{
		manager: manager,
		logger:  logger,
	}
}

// Start 启动后台同步任务。
// 参数：ctx 为生命周期上下文。
// 返回：无。
func (s *Service) Start(ctx context.Context) {
	go s.run(ctx)
}

// run 执行后台循环并按计划触发同步任务。
// 参数：ctx 为生命周期上下文。
// 返回：无。
// 异常：无。
func (s *Service) run(ctx context.Context) {
	// 初始执行一次
	s.trySync(ctx)

	// 使用 1 分钟作为基础检测周期，支持配置变更感知
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	var lastSync time.Time = time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if s.shouldSync(lastSync) {
				s.trySync(ctx)
				lastSync = time.Now()
			}
		}
	}
}

// shouldSync 根据配置周期判断是否需要触发下一次同步。
// 参数：lastSync 为最近一次同步时间。
// 返回：true 表示应立即执行同步。
// 异常：计划解析失败时按一小时兜底。
func (s *Service) shouldSync(lastSync time.Time) bool {
	cfg := s.manager.CurrentConfig()
	if !cfg.CDNSync.Enabled {
		return false
	}

	schedule := cfg.CDNSync.Schedule
	if schedule == "" {
		schedule = "1h"
	}
	// 兼容 @every 语法
	schedule = strings.TrimPrefix(schedule, "@every ")
	// 兼容空格
	schedule = strings.TrimSpace(schedule)

	duration, err := time.ParseDuration(schedule)
	if err != nil {
		// 解析失败默认 1 小时
		duration = time.Hour
	}

	return time.Since(lastSync) >= duration
}

// trySync 并发执行所有配置提供商的同步流程。
// 参数：ctx 为同步上下文。
// 返回：无。
// 异常：无。
func (s *Service) trySync(ctx context.Context) {
	cfg := s.manager.CurrentConfig()
	if !cfg.CDNSync.Enabled {
		return
	}

	var wg sync.WaitGroup
	for _, name := range cfg.CDNSync.Providers {
		wg.Add(1)
		go func(n string) {
			defer wg.Done()
			s.syncProvider(ctx, n, cfg)
		}(name)
	}
	wg.Wait()
}

// syncProvider 同步单个 CDN 提供商并写回状态快照。
// 参数：ctx 为同步上下文，name 为提供商名称，cfg 为当前配置。
// 返回：无。
// 异常：提供商不支持或抓取失败时记录错误并回写状态。
func (s *Service) syncProvider(ctx context.Context, name string, cfg config.Config) {
	var provider cdn.Provider
	switch strings.ToLower(name) {
	case "cloudflare":
		provider = cloudflare.NewProvider(cloudflare.Config{
			IPv4URL: cfg.CDNSync.Cloudflare.IPv4URL,
			IPv6URL: cfg.CDNSync.Cloudflare.IPv6URL,
		})
	default:
		s.logger.Warn("跳过未知 CDN 提供商", zap.String("provider", name))
		return
	}

	s.logger.Info("开始同步 CDN IP", zap.String("provider", provider.Name()))
	start := time.Now()

	syncCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cidrs, err := provider.FetchCIDRs(syncCtx)
	if err != nil {
		s.logger.Error("CDN IP 同步失败", zap.String("provider", name), zap.Error(err))
		_ = s.manager.UpdateProviderStatus(name, nil, err)
		return
	}

	if err := s.manager.UpdateProviderStatus(name, cidrs, nil); err != nil {
		s.logger.Error("应用 CDN IP 失败", zap.String("provider", name), zap.Error(err))
		return
	}

	s.logger.Info("CDN IP 同步完成",
		zap.String("provider", provider.Name()),
		zap.Int("count", len(cidrs)),
		zap.Duration("duration", time.Since(start)),
	)
}
