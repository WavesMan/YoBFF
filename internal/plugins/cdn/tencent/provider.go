package tencent

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"YoBFF/internal/plugins/cdn"
)

var _ cdn.Provider = (*Provider)(nil)

// Config 表示腾讯云 TEO 回源 IP 拉取所需配置。
// 说明：SecretKey 属于敏感信息，仅用于请求签名，需由上层安全存储并避免回显。
type Config struct {
	SecretID  string
	SecretKey string
	ZoneID    string
	Endpoint  string
}

// Provider 实现腾讯云 TEO 回源 IP 拉取逻辑。
// 说明：通过官方 OpenAPI 获取回源白名单 CIDR，用于网关侧按站点进行来源校验。
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

// Name 返回提供商标识，用于路由与审计记录。
func (p *Provider) Name() string {
	return "tencent"
}

// FetchCIDRs 拉取腾讯云 TEO 回源白名单 CIDR 列表。
// 返回：CIDR 列表（IPv4/IPv6 合并）。
// 异常：配置缺失、签名计算失败、请求失败或响应解析失败时返回错误。
func (p *Provider) FetchCIDRs(ctx context.Context) ([]string, error) {
	secretID := strings.TrimSpace(p.config.SecretID)
	secretKey := strings.TrimSpace(p.config.SecretKey)
	zoneID := strings.TrimSpace(p.config.ZoneID)
	if secretID == "" || secretKey == "" || zoneID == "" {
		return nil, fmt.Errorf("缺少必要配置: apiKey/secretKey/zoneId")
	}

	endpoint, err := normalizeEndpoint(p.config.Endpoint)
	if err != nil {
		return nil, err
	}

	action := "DescribeOriginACL"
	version := "2022-09-01"
	service := "teo"
	method := http.MethodPost
	contentType := "application/json; charset=utf-8"

	bodyPayload, err := json.Marshal(map[string]string{"ZoneId": zoneID})
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	timestamp := now.Unix()
	date := now.Format("2006-01-02")
	payloadHash := sha256Hex(bodyPayload)
	canonicalHeaders := "content-type:" + contentType + "\n" + "host:" + endpoint.Host + "\n"
	signedHeaders := "content-type;host"
	canonicalRequest := strings.Join([]string{
		method,
		"/",
		"",
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	algorithm := "TC3-HMAC-SHA256"
	credentialScope := date + "/" + service + "/tc3_request"
	stringToSign := strings.Join([]string{
		algorithm,
		fmt.Sprintf("%d", timestamp),
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	secretDate := hmacSHA256([]byte("TC3"+secretKey), []byte(date))
	secretService := hmacSHA256(secretDate, []byte(service))
	secretSigning := hmacSHA256(secretService, []byte("tc3_request"))
	signature := hex.EncodeToString(hmacSHA256(secretSigning, []byte(stringToSign)))

	authorization := fmt.Sprintf(
		"%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm,
		secretID,
		credentialScope,
		signedHeaders,
		signature,
	)

	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), bytes.NewReader(bodyPayload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Host", endpoint.Host)
	req.Header.Set("X-TC-Action", action)
	req.Header.Set("X-TC-Version", version)
	req.Header.Set("X-TC-Timestamp", fmt.Sprintf("%d", timestamp))
	req.Header.Set("Authorization", authorization)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var body struct {
			Response struct {
				Error struct {
					Code    string `json:"Code"`
					Message string `json:"Message"`
				} `json:"Error"`
			} `json:"Response"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&body)
		code := strings.TrimSpace(body.Response.Error.Code)
		message := strings.TrimSpace(body.Response.Error.Message)
		if code != "" || message != "" {
			return nil, fmt.Errorf("请求失败: status=%d code=%s message=%s", resp.StatusCode, code, message)
		}
		return nil, fmt.Errorf("请求失败: status=%d", resp.StatusCode)
	}

	var result struct {
		Response struct {
			OriginACLInfo struct {
				CurrentOriginACL struct {
					EntireAddresses struct {
						IPv4 []string `json:"IPv4"`
						IPv6 []string `json:"IPv6"`
					} `json:"EntireAddresses"`
				} `json:"CurrentOriginACL"`
				NextOriginACL struct {
					EntireAddresses struct {
						IPv4 []string `json:"IPv4"`
						IPv6 []string `json:"IPv6"`
					} `json:"EntireAddresses"`
				} `json:"NextOriginACL"`
			} `json:"OriginACLInfo"`
		} `json:"Response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	ipv4 := filterCIDRs(result.Response.OriginACLInfo.NextOriginACL.EntireAddresses.IPv4)
	ipv6 := filterCIDRs(result.Response.OriginACLInfo.NextOriginACL.EntireAddresses.IPv6)
	if len(ipv4) == 0 && len(ipv6) == 0 {
		ipv4 = filterCIDRs(result.Response.OriginACLInfo.CurrentOriginACL.EntireAddresses.IPv4)
		ipv6 = filterCIDRs(result.Response.OriginACLInfo.CurrentOriginACL.EntireAddresses.IPv6)
	}
	return append(ipv4, ipv6...), nil
}

// normalizeEndpoint 规范化并校验腾讯云 API endpoint。
// 规则：为空时使用官方默认地址；仅保留 scheme/host，固定 path 为 "/"。
func normalizeEndpoint(raw string) (*url.URL, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		value = "https://teo.tencentcloudapi.com"
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("endpoint 非法: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("endpoint 非法")
	}
	parsed.Path = "/"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed, nil
}

// sha256Hex 计算 payload 的 SHA256 十六进制哈希，用于签名与防篡改校验。
func sha256Hex(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

// hmacSHA256 计算 HMAC-SHA256 摘要，用于 TC3 请求签名派生。
func hmacSHA256(key []byte, msg []byte) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(msg)
	return mac.Sum(nil)
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
