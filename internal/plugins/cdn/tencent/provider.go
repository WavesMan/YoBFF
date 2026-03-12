package tencent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"YoBFF/internal/plugins/cdn"
)

var _ cdn.Provider = (*Provider)(nil)

type Config struct {
	IPv4URL string
	IPv6URL string
}

type Provider struct {
	config Config
	client *http.Client
}

// NewProvider 创建腾讯云回源 IP 拉取实例。
func NewProvider(cfg Config) *Provider {
	return &Provider{
		config: cfg,
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (p *Provider) Name() string {
	return "tencent"
}

func (p *Provider) FetchCIDRs(ctx context.Context) ([]string, error) {
	ipv4, err := fetchCIDRsFromURL(ctx, p.client, p.config.IPv4URL)
	if err != nil {
		return nil, fmt.Errorf("fetch ipv4 failed: %w", err)
	}
	ipv6, err := fetchCIDRsFromURL(ctx, p.client, p.config.IPv6URL)
	if err != nil {
		return nil, fmt.Errorf("fetch ipv6 failed: %w", err)
	}
	return append(ipv4, ipv6...), nil
}

func fetchCIDRsFromURL(ctx context.Context, client *http.Client, url string) ([]string, error) {
	value := strings.TrimSpace(url)
	if value == "" {
		return nil, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, value, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var jsonCIDRs []string
	if err := json.Unmarshal(body, &jsonCIDRs); err == nil {
		return filterCIDRs(jsonCIDRs), nil
	}

	var cidrs []string
	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		cidrs = append(cidrs, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return cidrs, nil
}

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

