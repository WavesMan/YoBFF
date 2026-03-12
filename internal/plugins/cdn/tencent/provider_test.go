package tencent

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestNormalizeEndpoint 验证 endpoint 默认值与非法输入处理。
func TestNormalizeEndpoint(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		got, err := normalizeEndpoint("")
		if err != nil {
			t.Fatalf("默认 endpoint 不应失败: %v", err)
		}
		if got.Scheme != "https" || got.Host != "teo.tencentcloudapi.com" || got.Path != "/" {
			t.Fatalf("默认 endpoint 不匹配: %v", got.String())
		}
	})

	t.Run("missing scheme", func(t *testing.T) {
		if _, err := normalizeEndpoint("teo.tencentcloudapi.com"); err == nil {
			t.Fatalf("缺少 scheme 的 endpoint 应返回错误")
		}
	})

	t.Run("invalid", func(t *testing.T) {
		if _, err := normalizeEndpoint("://bad"); err == nil {
			t.Fatalf("非法 endpoint 应返回错误")
		}
	})
}

// TestCryptoHelpers 验证签名相关辅助函数输出稳定且符合标准实现。
func TestCryptoHelpers(t *testing.T) {
	if got := sha256Hex([]byte("abc")); got != "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" {
		t.Fatalf("sha256Hex 结果错误: %q", got)
	}

	key := []byte("key")
	msg := []byte("msg")
	wantMac := hmac.New(sha256.New, key)
	_, _ = wantMac.Write(msg)
	want := wantMac.Sum(nil)
	got := hmacSHA256(key, msg)
	if hex.EncodeToString(got) != hex.EncodeToString(want) {
		t.Fatalf("hmacSHA256 结果错误")
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

	t.Run("success with fallback from current", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/" {
				http.NotFound(w, r)
				return
			}
			_, _ = fmt.Fprintln(w, `{
  "Response": {
    "OriginACLInfo": {
      "CurrentOriginACL": {
        "EntireAddresses": { "IPv4": ["1.1.1.1/32"], "IPv6": [] }
      },
      "NextOriginACL": {
        "EntireAddresses": { "IPv4": ["#comment"], "IPv6": [] }
      }
    }
  }
}`)
		}))
		defer ts.Close()

		p := NewProvider(Config{
			SecretID:  "id",
			SecretKey: "key",
			ZoneID:    "zone",
			Endpoint:  ts.URL,
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
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprintln(w, `{
  "Response": {
    "Error": { "Code": "BadRequest", "Message": "nope" }
  }
}`)
		}))
		defer ts.Close()

		p := NewProvider(Config{
			SecretID:  "id",
			SecretKey: "key",
			ZoneID:    "zone",
			Endpoint:  ts.URL,
		})
		p.client = ts.Client()

		_, err := p.FetchCIDRs(context.Background())
		if err == nil {
			t.Fatalf("非 200 应返回错误")
		}
		if !strings.Contains(err.Error(), "BadRequest") || !strings.Contains(err.Error(), "nope") {
			t.Fatalf("错误信息不包含响应详情: %v", err)
		}
	})
}
