package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"YoBFF/internal/plugins/cdn"
)

var _ cdn.Provider = (*Provider)(nil)

const (
	// DefaultEndpoint 为 Cloudflare 回源 IP 公共接口地址。
	DefaultEndpoint = "https://api.cloudflare.com/client/v4/ips"
)

// Config 定义 Cloudflare 插件配置。
// 说明：Cloudflare 回源 IP 使用公共接口获取，无需账号密钥。
type Config struct {
	Endpoint string
}

// Provider 实现 Cloudflare 回源 IP 拉取逻辑。
type Provider struct {
	config Config
	client *http.Client
}

// NewProvider 创建 Cloudflare 回源 IP 拉取实例。
// 说明：默认使用 Cloudflare 公共接口，可选覆盖 endpoint 以便测试。
func NewProvider(cfg Config) *Provider {
	if strings.TrimSpace(cfg.Endpoint) == "" {
		cfg.Endpoint = DefaultEndpoint
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

// FetchCIDRs 获取 Cloudflare 回源 IP 白名单 CIDR 列表。
// 返回：CIDR 列表（IPv4/IPv6 合并）。
// 异常：请求失败、状态码异常或响应解析失败时返回错误。
func (p *Provider) FetchCIDRs(ctx context.Context) ([]string, error) {
	ipv4, ipv6, err := p.fetch(ctx)
	if err != nil {
		return nil, err
	}
	return append(filterCIDRs(ipv4), filterCIDRs(ipv6)...), nil
}

// fetch 从 Cloudflare 公共接口拉取 IPv4/IPv6 CIDR 列表。
// 参数：ctx 为请求上下文。
// 返回：IPv4 CIDR、IPv6 CIDR。
// 异常：请求失败、状态码异常或响应解析失败时返回错误。
func (p *Provider) fetch(ctx context.Context) ([]string, []string, error) {
	endpoint, err := normalizeEndpoint(p.config.Endpoint)
	if err != nil {
		return nil, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("请求失败: status=%d", resp.StatusCode)
	}

	var result struct {
		Result struct {
			IPv4CIDRs []string `json:"ipv4_cidrs"`
			IPv6CIDRs []string `json:"ipv6_cidrs"`
		} `json:"result"`
		Success bool `json:"success"`
		Errors  []struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil, err
	}
	if !result.Success {
		if len(result.Errors) > 0 {
			return nil, nil, fmt.Errorf("请求失败: code=%d message=%s", result.Errors[0].Code, strings.TrimSpace(result.Errors[0].Message))
		}
		return nil, nil, fmt.Errorf("请求失败")
	}
	return result.Result.IPv4CIDRs, result.Result.IPv6CIDRs, nil
}

// normalizeEndpoint 规范化并校验 Cloudflare API endpoint。
// 规则：必须包含 scheme/host，固定 path 为 "/client/v4/ips"。
func normalizeEndpoint(raw string) (*url.URL, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		value = DefaultEndpoint
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("endpoint 非法: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("endpoint 非法")
	}
	parsed.Path = "/client/v4/ips"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed, nil
}

// filterCIDRs 过滤空行与注释行，避免将无效条目写入放行规则。
func filterCIDRs(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		value := strings.TrimSpace(item)
		if value == "" || strings.HasPrefix(value, "#") {
			continue
		}
		out = append(out, value)
	}
	return out
}
