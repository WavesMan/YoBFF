package config

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type routeItem struct {
	Domain     string
	ForceHTTPS bool
	Target     *url.URL
}

type wildcardItem struct {
	Suffix string
	Route  routeItem
}

type snapshot struct {
	cfg            Config
	allowAll       bool
	allowedCIDRs   []netip.Prefix
	exactRoutes    map[string]routeItem
	wildcardRoutes []wildcardItem
	defaultRoute   *routeItem
	certificates   map[string]*tls.Certificate
	defaultCert    *tls.Certificate
}

// RouteMatch 表示路由解析成功后的目标结果。
type RouteMatch struct {
	Domain     string
	ForceHTTPS bool
	Target     *url.URL
}

// CDNStatus 记录单个 CDN 提供商的同步状态
type CDNStatus struct {
	Provider string    `json:"provider"`
	CIDRs    []string  `json:"cidrs"`
	LastSync time.Time `json:"last_sync"`
	Error    string    `json:"error,omitempty"`
}

// Manager 负责管理运行时配置快照与原子切换。
type Manager struct {
	path         string
	data         atomic.Pointer[snapshot]
	mu           sync.Mutex
	dynamicCIDRs []string
	cdnStatuses  map[string]CDNStatus
}

// NewManager 从配置文件初始化配置管理器。
// 参数：path 为配置文件路径。
// 返回：可用于查询与更新的管理器实例。
// 异常：文件读取、解析或构建快照失败时返回错误。
func NewManager(path string) (*Manager, error) {
	manager := &Manager{path: path}
	cfg, err := loadConfig(path)
	if err != nil {
		return nil, err
	}
	cfg = ApplyEnvOverrides(cfg)
	if err = manager.Apply(cfg); err != nil {
		return nil, err
	}
	return manager, nil
}

// ConfigPath 返回当前管理器绑定的配置文件路径。
// 参数：无。
// 返回：配置文件绝对或相对路径。
// 异常：无。
func (m *Manager) ConfigPath() string {
	return m.path
}

// CurrentConfig 获取当前生效配置副本。
// 参数：无。
// 返回：配置对象，未初始化时返回默认配置。
// 异常：无。
func (m *Manager) CurrentConfig() Config {
	data := m.data.Load()
	if data == nil {
		return defaultConfig()
	}
	return data.cfg
}

// Apply 对输入配置执行校验并原子替换运行快照。
// 参数：cfg 为待应用配置。
// 返回：应用失败错误。
// 异常：当路由、CIDR 或证书非法时返回错误。
func (m *Manager) Apply(cfg Config) error {
	m.mu.Lock()
	dynamic := m.dynamicCIDRs
	m.mu.Unlock()

	filled := fillDefaults(cfg)
	next, err := buildSnapshot(filled, m.path, dynamic)
	if err != nil {
		return err
	}
	m.data.Store(next)
	return nil
}

// UpdateProviderStatus 更新特定 CDN 提供商的 IP 列表并重构快照。
// 参数：provider 为提供商名称，cidrs 为新的 IP 列表，syncErr 为同步错误信息（如果有）。
// 返回：更新失败错误。
// 异常：无。
func (m *Manager) UpdateProviderStatus(provider string, cidrs []string, syncErr error) error {
	m.mu.Lock()
	if m.cdnStatuses == nil {
		m.cdnStatuses = make(map[string]CDNStatus)
	}

	errMsg := ""
	if syncErr != nil {
		errMsg = syncErr.Error()
	}

	m.cdnStatuses[provider] = CDNStatus{
		Provider: provider,
		CIDRs:    cidrs,
		LastSync: time.Now(),
		Error:    errMsg,
	}

	var allDynamic []string
	for _, status := range m.cdnStatuses {
		if len(status.CIDRs) > 0 {
			allDynamic = append(allDynamic, status.CIDRs...)
		}
	}
	m.dynamicCIDRs = allDynamic
	m.mu.Unlock()

	return m.Apply(m.CurrentConfig())
}

// GetCDNStatus 获取所有 CDN 提供商的同步状态副本。
// 参数：无。
// 返回：以提供商名称为键的状态映射。
// 异常：无。
func (m *Manager) GetCDNStatus() map[string]CDNStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	copyStatus := make(map[string]CDNStatus, len(m.cdnStatuses))
	for k, v := range m.cdnStatuses {
		copyStatus[k] = v
	}
	return copyStatus
}

// Reload 从磁盘重新加载配置并应用到运行时。
// 参数：无。
// 返回：重载失败错误。
// 异常：读取配置文件失败或配置非法时返回错误。
func (m *Manager) Reload() error {
	cfg, err := loadConfig(m.path)
	if err != nil {
		return err
	}
	return m.Apply(cfg)
}

// ResolveRoute 按域名解析上游路由规则。
// 参数：host 为请求 Host。
// 返回：匹配结果与是否命中。
// 异常：无。
func (m *Manager) ResolveRoute(host string) (RouteMatch, bool) {
	data := m.data.Load()
	if data == nil {
		return RouteMatch{}, false
	}

	normalizedHost := normalizeHost(host)
	if route, ok := data.exactRoutes[normalizedHost]; ok {
		return RouteMatch{Domain: route.Domain, ForceHTTPS: route.ForceHTTPS, Target: route.Target}, true
	}

	for _, item := range data.wildcardRoutes {
		if strings.HasSuffix(normalizedHost, item.Suffix) {
			return RouteMatch{
				Domain:     item.Route.Domain,
				ForceHTTPS: item.Route.ForceHTTPS,
				Target:     item.Route.Target,
			}, true
		}
	}

	if data.defaultRoute != nil {
		return RouteMatch{
			Domain:     data.defaultRoute.Domain,
			ForceHTTPS: data.defaultRoute.ForceHTTPS,
			Target:     data.defaultRoute.Target,
		}, true
	}
	return RouteMatch{}, false
}

// IsHostAuthorized 判断请求域名是否在授权路由集合内。
// 参数：host 为请求域名。
// 返回：true 表示允许继续处理。
// 异常：无。
func (m *Manager) IsHostAuthorized(host string) bool {
	_, ok := m.ResolveRoute(host)
	return ok
}

// IsIPAllowed 判断来源 IP 是否命中白名单。
// 参数：ip 为客户端地址。
// 返回：true 表示来源 IP 合法。
// 异常：无。
func (m *Manager) IsIPAllowed(ip netip.Addr) bool {
	data := m.data.Load()
	if data == nil {
		return false
	}
	if data.allowAll {
		return true
	}
	for _, cidr := range data.allowedCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// GetCertificate 在 TLS 握手阶段按 SNI 返回匹配证书。
// 参数：hello 为握手元信息。
// 返回：匹配证书与错误信息。
// 异常：配置未就绪或证书不存在时返回错误。
func (m *Manager) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	data := m.data.Load()
	if data == nil {
		return nil, errors.New("配置未就绪")
	}
	serverName := normalizeHost(hello.ServerName)
	if cert, ok := data.certificates[serverName]; ok {
		return cert, nil
	}
	if data.defaultCert != nil {
		return data.defaultCert, nil
	}
	return nil, errors.New("未找到可用证书")
}

// HasCertificates 判断当前快照是否已加载证书。
// 参数：无。
// 返回：true 表示可启动 HTTPS 监听。
// 异常：无。
func (m *Manager) HasCertificates() bool {
	data := m.data.Load()
	return data != nil && len(data.certificates) > 0
}

// fillDefaults 补齐用户配置中的默认字段。
// 参数：cfg 为待处理配置。
// 返回：补齐后的配置对象。
// 异常：无。
func fillDefaults(cfg Config) Config {
	base := defaultConfig()
	if cfg.DataPlane.HTTPListenAddr == "" {
		cfg.DataPlane.HTTPListenAddr = base.DataPlane.HTTPListenAddr
	}
	if cfg.DataPlane.HTTPSListenAddr == "" {
		cfg.DataPlane.HTTPSListenAddr = base.DataPlane.HTTPSListenAddr
	}
	if cfg.ControlPlane.AdminListenAddr == "" {
		cfg.ControlPlane.AdminListenAddr = base.ControlPlane.AdminListenAddr
	}
	if cfg.Security.BlockPageHTML == "" {
		cfg.Security.BlockPageHTML = base.Security.BlockPageHTML
	}
	return cfg
}

// buildSnapshot 将配置编译为只读快照结构。
// 参数：cfg 为已补齐配置，configPath 用于定位证书相对路径，dynamicCIDRs 为动态 IP 白名单。
// 返回：可原子替换的快照对象。
// 异常：CIDR、路由或证书加载失败时返回错误。
func buildSnapshot(cfg Config, configPath string, dynamicCIDRs []string) (*snapshot, error) {
	data := &snapshot{
		cfg:            cfg,
		exactRoutes:    make(map[string]routeItem),
		wildcardRoutes: make([]wildcardItem, 0),
		certificates:   make(map[string]*tls.Certificate),
	}

	allCIDRs := make([]string, 0, len(cfg.Security.AllowedCIDRs)+len(dynamicCIDRs))
	allCIDRs = append(allCIDRs, cfg.Security.AllowedCIDRs...)
	allCIDRs = append(allCIDRs, dynamicCIDRs...)

	if len(allCIDRs) == 0 {
		data.allowAll = true
	} else {
		for _, cidrText := range allCIDRs {
			prefix, err := netip.ParsePrefix(strings.TrimSpace(cidrText))
			if err != nil {
				return nil, fmt.Errorf("非法 CIDR %q: %w", cidrText, err)
			}
			data.allowedCIDRs = append(data.allowedCIDRs, prefix)
		}
	}

	for _, rule := range cfg.Routing.Domains {
		upstreamURL, err := url.Parse(rule.Upstream)
		if err != nil || upstreamURL.Host == "" || upstreamURL.Scheme == "" {
			return nil, fmt.Errorf("域名 %q 对应 Upstream 非法: %s", rule.Domain, rule.Upstream)
		}
		domain := normalizeHost(rule.Domain)
		item := routeItem{
			Domain:     domain,
			ForceHTTPS: rule.ForceHTTPS,
			Target:     upstreamURL,
		}
		if strings.HasPrefix(domain, "*.") {
			suffix := domain[1:]
			data.wildcardRoutes = append(data.wildcardRoutes, wildcardItem{
				Suffix: suffix,
				Route:  item,
			})
			continue
		}
		data.exactRoutes[domain] = item
	}

	if cfg.Routing.DefaultUpstream != "" {
		upstreamURL, err := url.Parse(cfg.Routing.DefaultUpstream)
		if err != nil || upstreamURL.Host == "" || upstreamURL.Scheme == "" {
			return nil, fmt.Errorf("默认 Upstream 非法: %s", cfg.Routing.DefaultUpstream)
		}
		data.defaultRoute = &routeItem{
			Domain:     "_default",
			ForceHTTPS: false,
			Target:     upstreamURL,
		}
	}

	if err := loadCertificates(data, configPath); err != nil {
		return nil, err
	}
	return data, nil
}

// loadCertificates 按配置加载证书并注册到快照索引。
// 参数：data 为待填充快照，configPath 为主配置路径。
// 返回：加载失败错误。
// 异常：文件读取失败或证书解析失败时返回错误。
func loadCertificates(data *snapshot, configPath string) error {
	if len(data.cfg.Certificates) == 0 {
		return nil
	}
	baseDir := filepath.Dir(configPath)
	for idx, item := range data.cfg.Certificates {
		domain := normalizeHost(item.Domain)
		if domain == "" {
			return fmt.Errorf("证书 domain 不能为空")
		}
		certPEM := []byte(item.CertPEM)
		keyPEM := []byte(item.KeyPEM)
		var err error
		if item.CertFile != "" {
			certPEM, err = os.ReadFile(filepath.Join(baseDir, item.CertFile))
			if err != nil {
				return fmt.Errorf("读取证书文件失败 %s: %w", item.CertFile, err)
			}
		}
		if item.KeyFile != "" {
			keyPEM, err = os.ReadFile(filepath.Join(baseDir, item.KeyFile))
			if err != nil {
				return fmt.Errorf("读取私钥文件失败 %s: %w", item.KeyFile, err)
			}
		}
		cert, err := tls.X509KeyPair(certPEM, keyPEM)
		if err != nil {
			return fmt.Errorf("加载证书失败 domain=%s: %w", domain, err)
		}
		data.certificates[domain] = &cert
		if idx == 0 {
			data.defaultCert = &cert
		}
	}
	return nil
}

// loadConfig 从磁盘读取并反序列化配置。
// 参数：path 为配置文件路径。
// 返回：配置对象与错误。
// 异常：读取失败或 JSON 非法时返回错误。
func loadConfig(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("读取配置失败: %w", err)
	}
	cfg := defaultConfig()
	if err = json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("解析配置失败: %w", err)
	}
	return cfg, nil
}

// normalizeHost 对 Host 值进行标准化处理。
// 参数：host 为原始 Host 字符串。
// 返回：去协议、去端口并转小写后的域名。
// 异常：无。
func normalizeHost(host string) string {
	value := strings.TrimSpace(strings.ToLower(host))
	value = strings.TrimPrefix(value, "http://")
	value = strings.TrimPrefix(value, "https://")
	if idx := strings.IndexByte(value, ':'); idx >= 0 {
		value = value[:idx]
	}
	return value
}
