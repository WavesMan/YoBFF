package aliyun

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"YoBFF/internal/plugins/cdn"
)

var _ cdn.Provider = (*Provider)(nil)

type Config struct {
	AccessKeyID     string
	AccessKeySecret string
	Endpoint        string
	SiteID          string
}

type Provider struct {
	config Config
	client *http.Client
}

// NewProvider 创建阿里云回源 IP 拉取实例。
func NewProvider(cfg Config) *Provider {
	return &Provider{
		config: cfg,
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (p *Provider) Name() string {
	return "aliyun"
}

func (p *Provider) FetchCIDRs(ctx context.Context) ([]string, error) {
	accessKeyID := strings.TrimSpace(p.config.AccessKeyID)
	accessKeySecret := strings.TrimSpace(p.config.AccessKeySecret)
	siteIDText := strings.TrimSpace(p.config.SiteID)
	if accessKeyID == "" || accessKeySecret == "" || siteIDText == "" {
		return nil, errorsMissingConfig()
	}
	if _, err := strconv.ParseInt(siteIDText, 10, 64); err != nil {
		return nil, fmt.Errorf("siteId 非法: %w", err)
	}

	endpoint, err := normalizeEndpoint(p.config.Endpoint)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Set("SiteId", siteIDText)
	reqURL := endpoint.ResolveReference(&url.URL{Path: "/", RawQuery: query.Encode()})

	method := http.MethodGet
	action := "GetOriginProtection"
	version := "2024-09-10"
	now := time.Now().UTC().Format(time.RFC3339)
	nonce, err := randomHex(16)
	if err != nil {
		return nil, err
	}
	payloadHash := sha256Hex(nil)

	headers := map[string]string{
		"host":                  reqURL.Host,
		"x-acs-action":          action,
		"x-acs-version":         version,
		"x-acs-date":            now,
		"x-acs-signature-nonce": nonce,
		"x-acs-content-sha256":  payloadHash,
	}
	signedHeaders, canonicalHeaders := buildCanonicalHeaders(headers)
	canonicalQuery := buildCanonicalQuery(reqURL.Query())

	canonicalRequest := strings.Join([]string{
		method,
		"/",
		canonicalQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")
	stringToSign := "ACS3-HMAC-SHA256\n" + sha256Hex([]byte(canonicalRequest))
	signature := hmacSHA256Hex([]byte(accessKeySecret), []byte(stringToSign))
	authorization := fmt.Sprintf(
		"ACS3-HMAC-SHA256 Credential=%s,SignedHeaders=%s,Signature=%s",
		accessKeyID,
		signedHeaders,
		signature,
	)

	req, err := http.NewRequestWithContext(ctx, method, reqURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", authorization)
	req.Header.Set("X-Acs-Action", action)
	req.Header.Set("X-Acs-Version", version)
	req.Header.Set("X-Acs-Date", now)
	req.Header.Set("X-Acs-Signature-Nonce", nonce)
	req.Header.Set("X-Acs-Content-Sha256", payloadHash)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var body struct {
			Message string `json:"Message"`
			Code    string `json:"Code"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&body)
		if body.Message != "" || body.Code != "" {
			return nil, fmt.Errorf("请求失败: status=%d code=%s message=%s", resp.StatusCode, strings.TrimSpace(body.Code), strings.TrimSpace(body.Message))
		}
		return nil, fmt.Errorf("请求失败: status=%d", resp.StatusCode)
	}

	var result struct {
		CurrentIPWhitelist struct {
			IPv4 []string `json:"IPv4"`
			IPv6 []string `json:"IPv6"`
		} `json:"CurrentIPWhitelist"`
		LatestIPWhitelist struct {
			IPv4 []string `json:"IPv4"`
			IPv6 []string `json:"IPv6"`
		} `json:"LatestIPWhitelist"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	ipv4 := filterCIDRs(result.LatestIPWhitelist.IPv4)
	ipv6 := filterCIDRs(result.LatestIPWhitelist.IPv6)
	if len(ipv4) == 0 && len(ipv6) == 0 {
		ipv4 = filterCIDRs(result.CurrentIPWhitelist.IPv4)
		ipv6 = filterCIDRs(result.CurrentIPWhitelist.IPv6)
	}
	return append(ipv4, ipv6...), nil
}

func normalizeEndpoint(raw string) (*url.URL, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		value = "https://esa.cn-hangzhou.aliyuncs.com"
	}
	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		value = "https://" + value
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("endpoint 非法: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, errorsMissingEndpoint()
	}
	parsed.Path = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed, nil
}

func buildCanonicalQuery(values url.Values) string {
	if len(values) == 0 {
		return ""
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		vals := values[key]
		if len(vals) == 0 {
			parts = append(parts, url.QueryEscape(key)+"=")
			continue
		}
		sort.Strings(vals)
		for _, val := range vals {
			parts = append(parts, url.QueryEscape(key)+"="+url.QueryEscape(val))
		}
	}
	return strings.Join(parts, "&")
}

func buildCanonicalHeaders(headers map[string]string) (signedHeaders string, canonicalHeaders string) {
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, strings.ToLower(strings.TrimSpace(key)))
	}
	sort.Strings(keys)
	seen := make(map[string]struct{}, len(keys))
	var signed []string
	var canonical strings.Builder
	for _, key := range keys {
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		value := strings.TrimSpace(headers[key])
		signed = append(signed, key)
		canonical.WriteString(key)
		canonical.WriteString(":")
		canonical.WriteString(value)
		canonical.WriteString("\n")
	}
	return strings.Join(signed, ";"), canonical.String()
}

func sha256Hex(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func hmacSHA256Hex(key []byte, msg []byte) string {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(msg)
	return hex.EncodeToString(mac.Sum(nil))
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func errorsMissingConfig() error {
	return fmt.Errorf("缺少必要配置: apiKey/secretKey/option(siteId)")
}

func errorsMissingEndpoint() error {
	return fmt.Errorf("endpoint 不能为空")
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
