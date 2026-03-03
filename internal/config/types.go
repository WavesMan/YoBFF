package config

// Config 表示网关完整运行配置。
type Config struct {
	DataPlane    DataPlaneConfig    `json:"dataPlane"`
	ControlPlane ControlPlaneConfig `json:"controlPlane"`
	Security     SecurityConfig     `json:"security"`
	Routing      RoutingConfig      `json:"routing"`
	Certificates []Certificate      `json:"certificates"`
}

// DataPlaneConfig 表示数据平面监听参数。
type DataPlaneConfig struct {
	HTTPListenAddr  string `json:"httpListenAddr"`
	HTTPSListenAddr string `json:"httpsListenAddr"`
	EnableHTTPS     bool   `json:"enableHttps"`
}

// ControlPlaneConfig 表示控制平面监听参数。
type ControlPlaneConfig struct {
	AdminListenAddr string `json:"adminListenAddr"`
}

// SecurityConfig 表示回源安全策略。
type SecurityConfig struct {
	AllowedCIDRs  []string `json:"allowedCidrs"`
	BlockPageHTML string   `json:"blockPageHtml"`
	EnableHSTS    bool     `json:"enableHsts"`
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

// Certificate 表示证书加载来源与域名绑定关系。
type Certificate struct {
	Domain   string `json:"domain"`
	CertPEM  string `json:"certPem"`
	KeyPEM   string `json:"keyPem"`
	CertFile string `json:"certFile"`
	KeyFile  string `json:"keyFile"`
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
		},
		Security: SecurityConfig{
			BlockPageHTML: "<html><body><h1>403 Forbidden</h1></body></html>",
			EnableHSTS:    false,
		},
	}
}
