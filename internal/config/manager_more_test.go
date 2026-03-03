package config

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/netip"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestManager_ConfigPathAndCurrentConfig 验证配置路径读取与默认快照行为。
func TestManager_ConfigPathAndCurrentConfig(t *testing.T) {
	manager := &Manager{path: "config.json"}
	if got := manager.ConfigPath(); got != "config.json" {
		t.Fatalf("ConfigPath 不匹配: got=%q", got)
	}

	cfg := manager.CurrentConfig()
	if cfg.DataPlane.HTTPListenAddr == "" {
		t.Fatalf("默认配置未返回")
	}
}

// TestManager_IsHostAuthorized 验证域名授权判断逻辑。
func TestManager_IsHostAuthorized(t *testing.T) {
	manager := buildManagerFromJSON(t, `{
  "routing": { "defaultUpstream": "http://127.0.0.1:18080", "domains": [] },
  "security": { "allowedCidrs": [], "blockPageHtml": "<html/>", "enableHsts": false }
}`)

	if !manager.IsHostAuthorized("any.host") {
		t.Fatalf("默认上游存在时应授权任意域名")
	}

	empty := buildManagerFromJSON(t, `{
  "routing": { "defaultUpstream": "", "domains": [] },
  "security": { "allowedCidrs": [], "blockPageHtml": "<html/>", "enableHsts": false }
}`)
	if empty.IsHostAuthorized("any.host") {
		t.Fatalf("无路由规则时不应授权域名")
	}
}

// TestManager_IsIPAllowed 验证来源 IP 白名单判断逻辑。
func TestManager_IsIPAllowed(t *testing.T) {
	manager := buildManagerFromJSON(t, `{
  "routing": { "defaultUpstream": "http://127.0.0.1:18080", "domains": [] },
  "security": { "allowedCidrs": [], "blockPageHtml": "<html/>", "enableHsts": false }
}`)
	if !manager.IsIPAllowed(netip.MustParseAddr("192.168.1.1")) {
		t.Fatalf("AllowedCIDRs 为空时应放行所有 IP")
	}

	restricted := buildManagerFromJSON(t, `{
  "routing": { "defaultUpstream": "http://127.0.0.1:18080", "domains": [] },
  "security": { "allowedCidrs": ["127.0.0.1/32"], "blockPageHtml": "<html/>", "enableHsts": false }
}`)
	if !restricted.IsIPAllowed(netip.MustParseAddr("127.0.0.1")) {
		t.Fatalf("应放行 127.0.0.1")
	}
	if restricted.IsIPAllowed(netip.MustParseAddr("192.168.1.1")) {
		t.Fatalf("不应放行 192.168.1.1")
	}
}

// TestManager_Reload 验证从磁盘重新加载配置可生效且非法配置会返回错误。
func TestManager_Reload(t *testing.T) {
	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, "config.json")

	v1 := `{
  "dataPlane": { "httpListenAddr": ":8080", "httpsListenAddr": ":8443", "enableHttps": false },
  "security": { "allowedCidrs": [], "blockPageHtml": "<html/>", "enableHsts": false },
  "routing": { "defaultUpstream": "http://127.0.0.1:18080", "domains": [] }
}`
	if err := os.WriteFile(configPath, []byte(v1), 0o600); err != nil {
		t.Fatalf("写入配置失败: %v", err)
	}

	manager, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("初始化 Manager 失败: %v", err)
	}

	v2 := `{
  "dataPlane": { "httpListenAddr": ":18080", "httpsListenAddr": ":8443", "enableHttps": false },
  "security": { "allowedCidrs": [], "blockPageHtml": "<html/>", "enableHsts": false },
  "routing": { "defaultUpstream": "http://127.0.0.1:18080", "domains": [] }
}`
	if err = os.WriteFile(configPath, []byte(v2), 0o600); err != nil {
		t.Fatalf("写入配置失败: %v", err)
	}
	if err = manager.Reload(); err != nil {
		t.Fatalf("Reload 失败: %v", err)
	}
	if manager.CurrentConfig().DataPlane.HTTPListenAddr != ":18080" {
		t.Fatalf("Reload 未生效: got=%q", manager.CurrentConfig().DataPlane.HTTPListenAddr)
	}

	if err = os.WriteFile(configPath, []byte(`{invalid`), 0o600); err != nil {
		t.Fatalf("写入配置失败: %v", err)
	}
	if err = manager.Reload(); err == nil {
		t.Fatalf("非法配置应返回错误")
	}
}

// TestManager_Certificates 验证证书加载、HasCertificates 与 GetCertificate 行为。
func TestManager_Certificates(t *testing.T) {
	certPEM, keyPEM := buildSelfSignedPEM(t, "example.local")
	manager := buildManagerFromJSON(t, `{
  "routing": { "defaultUpstream": "http://127.0.0.1:18080", "domains": [] },
  "security": { "allowedCidrs": [], "blockPageHtml": "<html/>", "enableHsts": false },
  "certificates": []
}`)

	cfg := manager.CurrentConfig()
	cfg.Certificates = []Certificate{
		{
			Domain:  "example.local",
			CertPEM: string(certPEM),
			KeyPEM:  string(keyPEM),
		},
	}
	if err := manager.Apply(cfg); err != nil {
		t.Fatalf("Apply 失败: %v", err)
	}
	if !manager.HasCertificates() {
		t.Fatalf("HasCertificates 应返回 true")
	}

	cert, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: "example.local"})
	if err != nil || cert == nil {
		t.Fatalf("GetCertificate 失败: cert=%v err=%v", cert, err)
	}

	defaultCert, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: "unknown.local"})
	if err != nil || defaultCert == nil {
		t.Fatalf("GetCertificate 默认证书失败: cert=%v err=%v", defaultCert, err)
	}

	empty := &Manager{}
	if _, err = empty.GetCertificate(&tls.ClientHelloInfo{ServerName: "example.local"}); err == nil {
		t.Fatalf("未就绪配置应返回错误")
	}
}

// TestManager_LoadCertificatesFromFiles 验证从文件加载证书与私钥。
func TestManager_LoadCertificatesFromFiles(t *testing.T) {
	certPEM, keyPEM := buildSelfSignedPEM(t, "example.local")
	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, "config.json")
	certPath := filepath.Join(baseDir, "cert.pem")
	keyPath := filepath.Join(baseDir, "key.pem")
	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		t.Fatalf("写入证书失败: %v", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		t.Fatalf("写入私钥失败: %v", err)
	}

	content := `{
  "routing": { "defaultUpstream": "http://127.0.0.1:18080", "domains": [] },
  "security": { "allowedCidrs": [], "blockPageHtml": "<html/>", "enableHsts": false },
  "certificates": [
    { "domain": "example.local", "certFile": "cert.pem", "keyFile": "key.pem" }
  ]
}`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("写入配置失败: %v", err)
	}
	manager, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("初始化 Manager 失败: %v", err)
	}
	if !manager.HasCertificates() {
		t.Fatalf("HasCertificates 应返回 true")
	}

	cert, err := manager.GetCertificate(&tls.ClientHelloInfo{ServerName: "example.local"})
	if err != nil || cert == nil {
		t.Fatalf("GetCertificate 失败: cert=%v err=%v", cert, err)
	}
}

// TestManager_LoadCertificatesFromFiles_InvalidDomain 验证证书域名为空时 Apply 返回错误。
func TestManager_LoadCertificatesFromFiles_InvalidDomain(t *testing.T) {
	manager := buildManagerFromJSON(t, `{
  "routing": { "defaultUpstream": "http://127.0.0.1:18080", "domains": [] },
  "security": { "allowedCidrs": [], "blockPageHtml": "<html/>", "enableHsts": false }
}`)
	cfg := manager.CurrentConfig()
	cfg.Certificates = []Certificate{
		{
			Domain:  "",
			CertPEM: "x",
			KeyPEM:  "y",
		},
	}
	if err := manager.Apply(cfg); err == nil {
		t.Fatalf("Apply 应返回错误")
	}
}

// buildManagerFromJSON 构建用于单元测试的配置管理器并写入指定 JSON。
// 参数：t 为测试上下文，content 为配置 JSON 文本。
// 返回：初始化完成的配置管理器实例。
// 异常：写入或初始化失败时终止当前测试。
func buildManagerFromJSON(t *testing.T, content string) *Manager {
	t.Helper()

	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, "config.json")
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("写入配置失败: %v", err)
	}
	manager, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("初始化 Manager 失败: %v", err)
	}
	return manager
}

// buildSelfSignedPEM 生成用于测试的自签名证书与私钥 PEM。
// 参数：t 为测试上下文，domain 为证书 DNSNames。
// 返回：证书 PEM 与私钥 PEM 字节。
// 异常：生成失败时终止当前测试。
func buildSelfSignedPEM(t *testing.T, domain string) ([]byte, []byte) {
	t.Helper()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("生成私钥失败: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber:          bigSerialForTest(t),
		Subject:               pkix.Name{CommonName: "test"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{domain},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, privateKey.Public(), privateKey)
	if err != nil {
		t.Fatalf("签发证书失败: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		t.Fatalf("编码私钥失败: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return certPEM, keyPEM
}

// bigSerialForTest 生成用于测试的证书序列号。
// 参数：t 为测试上下文。
// 返回：随机序列号。
// 异常：生成失败时终止当前测试。
func bigSerialForTest(t *testing.T) *big.Int {
	t.Helper()

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 64))
	if err != nil {
		t.Fatalf("生成序列号失败: %v", err)
	}
	return serial
}
