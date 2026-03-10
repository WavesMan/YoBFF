package cloudflare

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"YoBFF/internal/plugins/cdn"
)

var _ cdn.Provider = (*Provider)(nil)

const (
	// DefaultIPv4URL 默认 IPv4 列表地址
	DefaultIPv4URL = "https://www.cloudflare.com/ips-v4"
	// DefaultIPv6URL 默认 IPv6 列表地址
	DefaultIPv6URL = "https://www.cloudflare.com/ips-v6"
)

// Config 定义 Cloudflare 插件配置。
type Config struct {
	IPv4URL string
	IPv6URL string
}

// Provider 实现 Cloudflare CDN IP 同步。
type Provider struct {
	config Config
	client *http.Client
}

// NewProvider 创建 Cloudflare 插件实例。
// 允许传入自定义 URL，若为空则使用默认值。
func NewProvider(cfg Config) *Provider {
	if cfg.IPv4URL == "" {
		cfg.IPv4URL = DefaultIPv4URL
	}
	if cfg.IPv6URL == "" {
		cfg.IPv6URL = DefaultIPv6URL
	}
	return &Provider{
		config: cfg,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Name 返回插件名称。
func (p *Provider) Name() string {
	return "cloudflare"
}

// FetchCIDRs 获取 Cloudflare 所有回源 IP 段。
func (p *Provider) FetchCIDRs(ctx context.Context) ([]string, error) {
	ipv4, err := p.fetch(ctx, p.config.IPv4URL)
	if err != nil {
		return nil, fmt.Errorf("fetch ipv4 failed: %w", err)
	}

	ipv6, err := p.fetch(ctx, p.config.IPv6URL)
	if err != nil {
		return nil, fmt.Errorf("fetch ipv6 failed: %w", err)
	}

	return append(ipv4, ipv6...), nil
}

// fetch 抓取单个地址列表并解析为 CIDR 切片。
// 参数：ctx 为请求上下文，url 为目标地址。
// 返回：解析后的 CIDR 列表。
// 异常：请求失败、状态码异常或读取失败时返回错误。
func (p *Provider) fetch(ctx context.Context, url string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var cidrs []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			cidrs = append(cidrs, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return cidrs, nil
}
