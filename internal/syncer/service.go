package syncer

import (
	"context"
	"database/sql"
	"errors"
	"math/rand"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"YoBFF/internal/config"
	"YoBFF/internal/plugins/cdn"
	"YoBFF/internal/plugins/cdn/aliyun"
	"YoBFF/internal/plugins/cdn/cloudflare"
	"YoBFF/internal/plugins/cdn/tencent"
	"YoBFF/internal/store"
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

// SiteOriginService 负责按站点配置定期拉取 CDN 回源 IP 快照并写入存储与内存快照。
type SiteOriginService struct {
	manager *config.Manager
	store   *store.Store
	logger  *zap.Logger

	maxConcurrency int
	tickInterval   time.Duration
	jitterRatio    float64
	backoffBase    time.Duration
	backoffMax     time.Duration
}

type siteOriginJob struct {
	siteID     string
	provider   string
	settings   config.CDNProviderSetting
	status     store.SiteCDNOriginStatus
	hasStatus  bool
	attemptNow time.Time
}

// NewSiteOriginService 创建站点回源 IP 同步服务实例。
// 参数：manager 为配置管理器，store 为站点配置存储，logger 为日志实例。
// 返回：同步服务实例。
func NewSiteOriginService(manager *config.Manager, store *store.Store, logger *zap.Logger) *SiteOriginService {
	if logger == nil {
		logger = zap.NewNop()
	}
	maxConcurrency := parsePositiveInt(config.EnvOrDefault("SITE_CDN_ORIGIN_SYNC_MAX_CONCURRENCY", "6"), 6)
	tickSeconds := parsePositiveInt(config.EnvOrDefault("SITE_CDN_ORIGIN_SYNC_TICK_SECONDS", "15"), 15)
	baseBackoffSeconds := parsePositiveInt(config.EnvOrDefault("SITE_CDN_ORIGIN_SYNC_BACKOFF_BASE_SECONDS", "30"), 30)
	maxBackoffSeconds := parsePositiveInt(config.EnvOrDefault("SITE_CDN_ORIGIN_SYNC_BACKOFF_MAX_SECONDS", "1800"), 1800)
	jitterPermille := parsePositiveInt(config.EnvOrDefault("SITE_CDN_ORIGIN_SYNC_JITTER_PERMILLE", "200"), 200)
	if jitterPermille < 0 {
		jitterPermille = 0
	}
	if jitterPermille > 900 {
		jitterPermille = 900
	}
	return &SiteOriginService{
		manager:        manager,
		store:          store,
		logger:         logger,
		maxConcurrency: maxConcurrency,
		tickInterval:   time.Duration(tickSeconds) * time.Second,
		jitterRatio:    float64(jitterPermille) / 1000.0,
		backoffBase:    time.Duration(baseBackoffSeconds) * time.Second,
		backoffMax:     time.Duration(maxBackoffSeconds) * time.Second,
	}
}

// Start 启动站点回源 IP 后台同步任务。
// 参数：ctx 为生命周期上下文。
// 返回：无。
func (s *SiteOriginService) Start(ctx context.Context) {
	go s.run(ctx)
}

// run 循环调度站点回源 IP 拉取任务。
// 参数：ctx 为生命周期上下文。
// 返回：无。
// 异常：无。
func (s *SiteOriginService) run(ctx context.Context) {
	if s == nil || s.store == nil || s.manager == nil {
		return
	}

	s.warmCache(ctx)

	ticker := time.NewTicker(s.tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *SiteOriginService) tick(ctx context.Context) {
	sites, err := s.store.ListSites(store.SiteFilter{})
	if err != nil {
		s.logger.Warn("拉取站点列表失败", zap.Error(err))
		return
	}

	now := time.Now().UTC()
	random := rand.New(rand.NewSource(now.UnixNano()))
	jobs := make([]siteOriginJob, 0, 16)

	for _, site := range sites {
		siteCfg, cfgErr := s.store.GetSiteConfig(site.ID)
		if cfgErr != nil {
			continue
		}
		providers := normalizeProviders(siteCfg.Security.AllowedCDNProviders)
		if len(providers) == 0 {
			continue
		}
		statuses, statusErr := s.store.ListSiteCDNOriginStatus(site.ID)
		if statusErr != nil {
			continue
		}
		statusMap := make(map[string]store.SiteCDNOriginStatus, len(statuses))
		for _, item := range statuses {
			statusMap[strings.ToLower(strings.TrimSpace(item.Provider))] = item
		}

		for _, provider := range providers {
			settings := config.CDNProviderSetting{}
			if siteCfg.Security.CDNProviderSettings != nil {
				if value, ok := siteCfg.Security.CDNProviderSettings[provider]; ok {
					settings = value
				}
			}

			itemStatus, ok := statusMap[provider]
			due := s.isDue(now, provider, settings, itemStatus, ok, random)
			if !due {
				continue
			}
			jobs = append(jobs, siteOriginJob{
				siteID:     site.ID,
				provider:   provider,
				settings:   settings,
				status:     itemStatus,
				hasStatus:  ok,
				attemptNow: now,
			})
		}
	}

	if len(jobs) == 0 {
		return
	}

	limit := s.maxConcurrency
	if limit <= 0 {
		limit = 1
	}
	if limit > len(jobs) {
		limit = len(jobs)
	}
	sem := make(chan struct{}, limit)

	var wg sync.WaitGroup
	for _, item := range jobs {
		sem <- struct{}{}
		wg.Add(1)
		go func(j siteOriginJob) {
			defer wg.Done()
			defer func() {
				<-sem
			}()
			s.runJob(ctx, j)
		}(item)
	}
	wg.Wait()
}

func (s *SiteOriginService) isDue(now time.Time, provider string, settings config.CDNProviderSetting, status store.SiteCDNOriginStatus, hasStatus bool, random *rand.Rand) bool {
	refreshInterval := time.Hour
	if settings.RefreshIntervalSeconds > 0 {
		refreshInterval = time.Duration(settings.RefreshIntervalSeconds) * time.Second
	}
	refreshInterval = jitterDuration(refreshInterval, s.jitterRatio, random)

	if !hasStatus || strings.TrimSpace(status.LastAttemptAt) == "" {
		return true
	}
	lastAttempt, err := time.Parse(time.RFC3339, status.LastAttemptAt)
	if err != nil {
		return true
	}

	backoff := time.Duration(0)
	if status.ConsecutiveFailures > 0 {
		backoff = s.backoffBase
		for i := 1; i < status.ConsecutiveFailures; i++ {
			backoff *= 2
			if backoff >= s.backoffMax {
				backoff = s.backoffMax
				break
			}
		}
		backoff = jitterDuration(backoff, s.jitterRatio, random)
	}

	nextDelay := refreshInterval
	if backoff > 0 && backoff < nextDelay {
		nextDelay = backoff
	}

	return now.Sub(lastAttempt) >= nextDelay
}

func (s *SiteOriginService) runJob(ctx context.Context, j siteOriginJob) {
	prevFailures := 0
	if j.hasStatus {
		prevFailures = j.status.ConsecutiveFailures
	}
	attempt := j.attemptNow
	_, _ = s.store.UpsertSiteCDNOriginStatus(j.siteID, j.provider, store.SiteCDNOriginStatusUpdate{
		LastAttemptAt:       &attempt,
		ConsecutiveFailures: prevFailures,
		LastError:           "",
	})

	provider, providerErr := buildOriginProvider(j.provider, j.settings)
	if providerErr != nil {
		failures := prevFailures + 1
		_, _ = s.store.UpsertSiteCDNOriginStatus(j.siteID, j.provider, store.SiteCDNOriginStatusUpdate{
			LastAttemptAt:       &attempt,
			ConsecutiveFailures: failures,
			LastError:           providerErr.Error(),
		})
		return
	}

	fetchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	cidrs, err := provider.FetchCIDRs(fetchCtx)
	cancel()
	if err != nil {
		failures := prevFailures + 1
		_, _ = s.store.UpsertSiteCDNOriginStatus(j.siteID, j.provider, store.SiteCDNOriginStatusUpdate{
			LastAttemptAt:       &attempt,
			ConsecutiveFailures: failures,
			LastError:           err.Error(),
		})
		return
	}

	success := j.attemptNow
	if _, err := s.store.SaveSiteCDNOriginSnapshot(j.siteID, j.provider, cidrs, success, "auto"); err != nil {
		failures := prevFailures + 1
		_, _ = s.store.UpsertSiteCDNOriginStatus(j.siteID, j.provider, store.SiteCDNOriginStatusUpdate{
			LastAttemptAt:       &attempt,
			ConsecutiveFailures: failures,
			LastError:           err.Error(),
		})
		return
	}

	_, _ = s.store.UpsertSiteCDNOriginStatus(j.siteID, j.provider, store.SiteCDNOriginStatusUpdate{
		LastAttemptAt:       &attempt,
		LastSuccessAt:       &success,
		ConsecutiveFailures: 0,
		LastError:           "",
	})
	_ = s.manager.UpdateSiteCDNProviderSnapshot(j.siteID, j.provider, cidrs, success)
}

func (s *SiteOriginService) warmCache(ctx context.Context) {
	sites, err := s.store.ListSites(store.SiteFilter{})
	if err != nil {
		return
	}
	for _, site := range sites {
		siteCfg, cfgErr := s.store.GetSiteConfig(site.ID)
		if cfgErr != nil {
			continue
		}
		providers := normalizeProviders(siteCfg.Security.AllowedCDNProviders)
		for _, provider := range providers {
			snapshot, snapErr := s.store.GetLatestSiteCDNOriginSnapshot(site.ID, provider)
			if snapErr != nil {
				if errors.Is(snapErr, sql.ErrNoRows) {
					continue
				}
				continue
			}
			fetchedAt := parseRFC3339(snapshot.FetchedAt)
			if fetchedAt.IsZero() {
				continue
			}
			_ = s.manager.UpdateSiteCDNProviderSnapshot(site.ID, provider, snapshot.CIDRs, fetchedAt)
		}
	}
}

func parseRFC3339(value string) time.Time {
	text := strings.TrimSpace(value)
	if text == "" {
		return time.Time{}
	}
	tm, err := time.Parse(time.RFC3339, text)
	if err != nil {
		return time.Time{}
	}
	return tm
}

func jitterDuration(base time.Duration, ratio float64, random *rand.Rand) time.Duration {
	if base <= 0 || ratio <= 0 || random == nil {
		return base
	}
	offset := (random.Float64()*2 - 1) * float64(base) * ratio
	next := time.Duration(float64(base) + offset)
	if next < 0 {
		return base
	}
	if next < time.Second {
		return time.Second
	}
	return next
}

func buildOriginProvider(provider string, settings config.CDNProviderSetting) (cdn.Provider, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "cloudflare":
		return cloudflare.NewProvider(cloudflare.Config{
			IPv4URL: strings.TrimSpace(settings.IPv4URL),
			IPv6URL: strings.TrimSpace(settings.IPv6URL),
		}), nil
	case "aliyun":
		return aliyun.NewProvider(aliyun.Config{
			IPv4URL: strings.TrimSpace(settings.IPv4URL),
			IPv6URL: strings.TrimSpace(settings.IPv6URL),
		}), nil
	case "tencent":
		return tencent.NewProvider(tencent.Config{
			IPv4URL: strings.TrimSpace(settings.IPv4URL),
			IPv6URL: strings.TrimSpace(settings.IPv6URL),
		}), nil
	default:
		return nil, errors.New("cdn provider is not supported")
	}
}

func normalizeProviders(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	var providers []string
	for _, item := range items {
		value := strings.ToLower(strings.TrimSpace(item))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		providers = append(providers, value)
	}
	return providers
}

func parsePositiveInt(text string, fallback int) int {
	value := strings.TrimSpace(text)
	if value == "" {
		return fallback
	}
	number := 0
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return fallback
		}
		number = number*10 + int(ch-'0')
	}
	if number <= 0 {
		return fallback
	}
	return number
}
