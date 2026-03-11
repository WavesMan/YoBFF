package cloudflare

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestProvider_FetchCIDRs 验证成功抓取 IPv4 与 IPv6 段并合并返回。
func TestProvider_FetchCIDRs(t *testing.T) {
	// 1. 创建 Mock Server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ipv4":
			fmt.Fprintln(w, "192.0.2.0/24")
			fmt.Fprintln(w, "198.51.100.0/24")
		case "/ipv6":
			fmt.Fprintln(w, "2001:db8::/32")
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	// 2. 初始化 Provider
	cfg := Config{
		IPv4URL: ts.URL + "/ipv4",
		IPv6URL: ts.URL + "/ipv6",
	}
	p := NewProvider(cfg)

	// 3. 执行 FetchCIDRs
	cidrs, err := p.FetchCIDRs(context.Background())
	if err != nil {
		t.Fatalf("FetchCIDRs failed: %v", err)
	}

	// 4. 验证结果
	expected := map[string]bool{
		"192.0.2.0/24":    true,
		"198.51.100.0/24": true,
		"2001:db8::/32":   true,
	}

	if len(cidrs) != len(expected) {
		t.Errorf("Expected %d CIDRs, got %d", len(expected), len(cidrs))
	}

	for _, cidr := range cidrs {
		if !expected[cidr] {
			t.Errorf("Unexpected CIDR: %s", cidr)
		}
	}
}

// TestProvider_FetchCIDRs_Error 验证远端异常状态码会触发错误返回。
func TestProvider_FetchCIDRs_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	cfg := Config{
		IPv4URL: ts.URL + "/ipv4",
		IPv6URL: ts.URL + "/ipv6",
	}
	p := NewProvider(cfg)

	_, err := p.FetchCIDRs(context.Background())
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestProvider_DefaultConfigAndName(t *testing.T) {
	p := NewProvider(Config{})
	if p.config.IPv4URL != DefaultIPv4URL {
		t.Fatalf("默认 IPv4URL 不匹配: %s", p.config.IPv4URL)
	}
	if p.config.IPv6URL != DefaultIPv6URL {
		t.Fatalf("默认 IPv6URL 不匹配: %s", p.config.IPv6URL)
	}
	if p.Name() != "cloudflare" {
		t.Fatalf("Provider 名称不匹配: %s", p.Name())
	}
}

func TestProvider_FetchCIDRs_IgnoresCommentsAndBlank(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "#comment")
		fmt.Fprintln(w, "  203.0.113.0/24  ")
	}))
	defer ts.Close()

	p := NewProvider(Config{
		IPv4URL: ts.URL + "/v4",
		IPv6URL: ts.URL + "/v6",
	})
	cidrs, err := p.FetchCIDRs(context.Background())
	if err != nil {
		t.Fatalf("FetchCIDRs failed: %v", err)
	}
	if len(cidrs) != 2 {
		t.Fatalf("CIDR 数量不匹配: got=%d cidrs=%v", len(cidrs), cidrs)
	}
	for _, cidr := range cidrs {
		if strings.HasPrefix(cidr, "#") || strings.TrimSpace(cidr) == "" {
			t.Fatalf("CIDR 过滤失败: %q", cidr)
		}
	}
}
