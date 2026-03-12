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
		if r.URL.Path != "/client/v4/ips" {
			http.NotFound(w, r)
			return
		}
		_, _ = fmt.Fprintln(w, `{
  "success": true,
  "errors": [],
  "messages": [],
  "result": {
    "ipv4_cidrs": ["192.0.2.0/24", "198.51.100.0/24"],
    "ipv6_cidrs": ["2001:db8::/32"]
  }
}`)
	}))
	defer ts.Close()

	// 2. 初始化 Provider
	cfg := Config{
		Endpoint: ts.URL,
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
		Endpoint: ts.URL,
	}
	p := NewProvider(cfg)

	_, err := p.FetchCIDRs(context.Background())
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestProvider_DefaultConfigAndName(t *testing.T) {
	p := NewProvider(Config{})
	if p.config.Endpoint != DefaultEndpoint {
		t.Fatalf("默认 endpoint 不匹配: %s", p.config.Endpoint)
	}
	if p.Name() != "cloudflare" {
		t.Fatalf("Provider 名称不匹配: %s", p.Name())
	}
}

func TestProvider_FetchCIDRs_IgnoresCommentsAndBlank(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, `{
  "success": true,
  "errors": [],
  "messages": [],
  "result": {
    "ipv4_cidrs": ["", "#comment", "  203.0.113.0/24  "],
    "ipv6_cidrs": ["2001:db8::/32"]
  }
}`)
	}))
	defer ts.Close()

	p := NewProvider(Config{
		Endpoint: ts.URL,
	})
	cidrs, err := p.FetchCIDRs(context.Background())
	if err != nil {
		t.Fatalf("FetchCIDRs failed: %v", err)
	}
	if len(cidrs) != 2 {
		t.Fatalf("CIDR 数量不匹配: got=%d cidrs=%v", len(cidrs), cidrs)
	}
}
