package config

import "time"

// SSLCertificate 表示 SSL 证书元数据及内容。
type SSLCertificate struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`      // 用户备注名
	Domains   []string  `json:"domains"`   // 证书包含的域名 (CN + SANs)
	NotAfter  time.Time `json:"notAfter"`  // 过期时间
	Issuer    string    `json:"issuer"`    // 颁发机构
	CertPEM   string    `json:"certPem"`   // 证书内容
	KeyPEM    string    `json:"keyPem"`    // 私钥内容
	CreatedAt time.Time `json:"createdAt"` // 创建时间
}

// Config 表示网关完整运行配置。
type Config struct {
	DataPlane       DataPlaneConfig    `json:"dataPlane"`
	ControlPlane    ControlPlaneConfig `json:"controlPlane"`
	Security        SecurityConfig     `json:"security"`
	Routing         RoutingConfig      `json:"routing"`
	LoadBalancer    LoadBalancerConfig `json:"loadBalancer"`
	CDNSync         CDNSyncConfig      `json:"cdnSync"`
	Certificates    []Certificate      `json:"certificates"`
	SSLCertificates []SSLCertificate   `json:"sslCertificates"`
}

// DataPlaneConfig 表示数据平面监听参数。
type DataPlaneConfig struct {
	HTTPListenAddr  string `json:"httpListenAddr"`
	HTTPSListenAddr string `json:"httpsListenAddr"`
	EnableHTTPS     bool   `json:"enableHttps"`
}

// ControlPlaneConfig 表示控制平面监听参数。
type ControlPlaneConfig struct {
	AdminListenAddr string     `json:"adminListenAddr"`
	Auth            AuthConfig `json:"auth"`
}

// AuthConfig 定义控制面认证配置。
type AuthConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Token    string `json:"token"`
}

// CDNSyncConfig 控制 CDN IP 同步行为。
type CDNSyncConfig struct {
	Enabled    bool             `json:"enabled"`
	Providers  []string         `json:"providers"` // 支持多个提供商并行启用
	Schedule   string           `json:"schedule"`
	Cloudflare CloudflareConfig `json:"cloudflare"`
}

// CloudflareConfig 定义 Cloudflare 特定配置。
type CloudflareConfig struct {
	IPv4URL string `json:"ipv4_url"`
	IPv6URL string `json:"ipv6_url"`
	// 以下字段为未来 API 管理预留
	Endpoint string `json:"endpoint,omitempty"`
	APIToken string `json:"api_token,omitempty"`
	ZoneID   string `json:"zone_id,omitempty"`
}

// SecurityConfig 表示回源安全策略。
type SecurityConfig struct {
	AllowedCIDRs        []string                      `json:"allowedCidrs"`
	AllowedCDNProviders []string                      `json:"allowedCdnProviders"`
	CDNProviderSettings map[string]CDNProviderSetting `json:"cdnProviderSettings,omitempty"`
	BlockPageHTML       string                        `json:"blockPageHtml"`
	EnableHSTS          bool                          `json:"enableHsts"`
}

// CDNProviderSetting 定义单个 CDN 厂商的特定配置。
type CDNProviderSetting struct {
	APIKey    string `json:"apiKey,omitempty"`
	SecretKey string `json:"secretKey,omitempty"`
	Option    string `json:"option,omitempty"`
}

// RoutingConfig 表示域名到上游的路由规则集合。
type RoutingConfig struct {
	DefaultUpstream string       `json:"defaultUpstream"`
	Domains         []DomainRule `json:"domains"`
}

// DomainRule 表示单条域名分发规则。
type DomainRule struct {
	Domain     string `json:"domain"`
	Upstream   string `json:"upstream"`
	ForceHTTPS bool   `json:"forceHttps"`
}

// LoadBalancerConfig 表示流量池与路由绑定配置。
type LoadBalancerConfig struct {
	DefaultPoolID string        `json:"defaultPoolId"`
	Pools         []LBPool      `json:"pools"`
	Routes        []LBRouteRule `json:"routes"`
}

// LBPool 表示可被域名路由绑定的流量池。
type LBPool struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Strategy string   `json:"strategy"`
	Nodes    []LBNode `json:"nodes"`
}

// LBNode 表示流量池中的单个上游节点。
type LBNode struct {
	ID       string `json:"id"`
	Upstream string `json:"upstream"`
	Weight   int    `json:"weight"`
	Enabled  bool   `json:"enabled"`
}

// LBRouteRule 表示域名到流量池的绑定规则。
type LBRouteRule struct {
	Domain         string `json:"domain"`
	PoolID         string `json:"poolId"`
	FallbackPoolID string `json:"fallbackPoolId"`
	ForceHTTPS     bool   `json:"forceHttps"`
}

// Certificate 表示证书加载来源与域名绑定关系。
type Certificate struct {
	Domain   string `json:"domain"`
	CertPEM  string `json:"certPem"`
	KeyPEM   string `json:"keyPem"`
	CertFile string `json:"certFile"`
	KeyFile  string `json:"keyFile"`
}

type ValidationIssue struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// defaultConfig 返回系统级默认配置。
// 参数：无。
// 返回：可直接运行的默认配置对象。
// 异常：无。
func defaultConfig() Config {
	return Config{
		DataPlane: DataPlaneConfig{
			HTTPListenAddr:  ":8080",
			HTTPSListenAddr: ":8443",
			EnableHTTPS:     true,
		},
		ControlPlane: ControlPlaneConfig{
			AdminListenAddr: ":9090",
			Auth: AuthConfig{
				Username: "admin",
				Password: "change_me", // 默认密码，强烈建议修改
				Token:    "",
			},
		},
		Security: SecurityConfig{
			BlockPageHTML: "<html><body><h1>403 Forbidden</h1></body></html>",
			EnableHSTS:    false,
		},
		CDNSync: CDNSyncConfig{
			Enabled:   false,
			Providers: []string{"cloudflare"},
			Schedule:  "@every 1h",
		},
	}
}
