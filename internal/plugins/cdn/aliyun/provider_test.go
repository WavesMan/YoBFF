package aliyun

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestNormalizeEndpoint 验证 endpoint 默认值、scheme 补齐与非法输入处理。
func TestNormalizeEndpoint(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		got, err := normalizeEndpoint("")
		if err != nil {
			t.Fatalf("默认 endpoint 不应失败: %v", err)
		}
		if got.Scheme != "https" || got.Host != "esa.cn-hangzhou.aliyuncs.com" {
			t.Fatalf("默认 endpoint 不匹配: %v", got.String())
		}
	})

	t.Run("no scheme", func(t *testing.T) {
		got, err := normalizeEndpoint("esa.cn-hangzhou.aliyuncs.com")
		if err != nil {
			t.Fatalf("补齐 scheme 不应失败: %v", err)
		}
		if got.Scheme != "https" || got.Host != "esa.cn-hangzhou.aliyuncs.com" {
			t.Fatalf("endpoint 不匹配: %v", got.String())
		}
	})

	t.Run("missing host", func(t *testing.T) {
		if _, err := normalizeEndpoint("https://"); err == nil {
			t.Fatalf("缺失 host 应返回错误")
		}
	})
}

// TestSigningHelpers 验证签名辅助逻辑的排序与加密输出符合预期。
func TestSigningHelpers(t *testing.T) {
	values := url.Values{}
	values.Add("b", "2")
	values.Add("a", "2")
	values.Add("a", "1")
	if got := buildCanonicalQuery(values); got != "a=1&a=2&b=2" {
		t.Fatalf("canonical query 不匹配: %q", got)
	}

	signed, canonical := buildCanonicalHeaders(map[string]string{
		"Host":         "example.com",
		"X-Acs-Action": "GetOriginProtection",
	})
	if signed != "host;x-acs-action" {
		t.Fatalf("signed headers 不匹配: %q", signed)
	}
	if canonical != "host:\n"+"x-acs-action:\n" {
		t.Fatalf("canonical headers 不匹配: %q", canonical)
	}

	if got := sha256Hex(nil); got != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Fatalf("sha256Hex(empty) 不匹配: %q", got)
	}

	key := []byte("key")
	msg := []byte("msg")
	wantMac := hmac.New(sha256.New, key)
	_, _ = wantMac.Write(msg)
	want := wantMac.Sum(nil)
	got := hmacSHA256Hex(key, msg)
	if got != hex.EncodeToString(want) {
		t.Fatalf("hmacSHA256Hex 结果错误")
	}

	nonce, err := randomHex(16)
	if err != nil {
		t.Fatalf("randomHex 失败: %v", err)
	}
	if len(nonce) != 32 {
		t.Fatalf("randomHex 长度错误: got=%d", len(nonce))
	}
	if _, err := hex.DecodeString(nonce); err != nil {
		t.Fatalf("randomHex 应返回 hex: %v", err)
	}
}

// TestFilterCIDRs 验证 CIDR 列表过滤规则会剔除空行与注释行。
func TestFilterCIDRs(t *testing.T) {
	got := filterCIDRs([]string{
		"",
		"  ",
		"# comment",
		"  # comment2",
		"1.1.1.1/32",
		" 2.2.2.2/32 ",
	})
	if len(got) != 2 || got[0] != "1.1.1.1/32" || got[1] != "2.2.2.2/32" {
		t.Fatalf("过滤结果错误: %+v", got)
	}
}

// TestProviderFetchCIDRs 使用本地 HTTP 服务覆盖成功与失败分支，避免真实请求依赖。
func TestProviderFetchCIDRs(t *testing.T) {
	t.Run("missing config", func(t *testing.T) {
		p := NewProvider(Config{})
		if _, err := p.FetchCIDRs(context.Background()); err == nil {
			t.Fatalf("缺少必要配置应失败")
		}
	})

	t.Run("invalid site id", func(t *testing.T) {
		p := NewProvider(Config{
			AccessKeyID:     "id",
			AccessKeySecret: "secret",
			SiteID:          "not-number",
			Endpoint:        "https://esa.cn-hangzhou.aliyuncs.com",
		})
		if _, err := p.FetchCIDRs(context.Background()); err == nil {
			t.Fatalf("非法 siteId 应失败")
		}
	})

	t.Run("success with fallback from current", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.URL.Path != "/" {
				http.NotFound(w, r)
				return
			}
			if r.URL.Query().Get("SiteId") != "123" {
				http.Error(w, "bad query", http.StatusBadRequest)
				return
			}
			_, _ = fmt.Fprintln(w, `{
  "CurrentIPWhitelist": { "IPv4": ["1.1.1.1/32"], "IPv6": [] },
  "LatestIPWhitelist": { "IPv4": ["#comment"], "IPv6": [] }
}`)
		}))
		defer ts.Close()

		p := NewProvider(Config{
			AccessKeyID:     "id",
			AccessKeySecret: "secret",
			SiteID:          "123",
			Endpoint:        ts.URL,
		})
		p.client = ts.Client()

		got, err := p.FetchCIDRs(context.Background())
		if err != nil {
			t.Fatalf("FetchCIDRs 失败: %v", err)
		}
		if len(got) != 1 || got[0] != "1.1.1.1/32" {
			t.Fatalf("CIDR 不匹配: %+v", got)
		}
	})

	t.Run("non-200 with error body", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = fmt.Fprintln(w, `{"Message":"deny","Code":"Denied"}`)
		}))
		defer ts.Close()

		p := NewProvider(Config{
			AccessKeyID:     "id",
			AccessKeySecret: "secret",
			SiteID:          "123",
			Endpoint:        ts.URL,
		})
		p.client = ts.Client()

		_, err := p.FetchCIDRs(context.Background())
		if err == nil {
			t.Fatalf("非 200 应返回错误")
		}
		if !strings.Contains(err.Error(), "Denied") || !strings.Contains(err.Error(), "deny") {
			t.Fatalf("错误信息不包含响应详情: %v", err)
		}
	})
}
