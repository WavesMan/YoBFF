package cloudflare

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
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
