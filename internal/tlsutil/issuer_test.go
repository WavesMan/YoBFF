package tlsutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"YoBFF/internal/config"
)

// TestCollectDomains_DedupAndDefault 验证域名归集去重与默认值逻辑。
func TestCollectDomains_DedupAndDefault(t *testing.T) {
	cfg := config.Config{
		Routing: config.RoutingConfig{
			Domains: []config.DomainRule{
				{Domain: "Example.Local"},
				{Domain: " example.local "},
				{Domain: ""},
				{Domain: "Other.Local"},
			},
		},
	}
	domains := collectDomains(cfg)
	if len(domains) != 2 {
		t.Fatalf("域名数量不匹配: got=%d domains=%v", len(domains), domains)
	}

	empty := config.Config{}
	domains = collectDomains(empty)
	if len(domains) != 1 || domains[0] != "localhost" {
		t.Fatalf("默认域名不匹配: got=%v", domains)
	}
}

// TestBuildCertificate_SelfSigned 验证在未配置 CA 文件时生成自签证书。
func TestBuildCertificate_SelfSigned(t *testing.T) {
	t.Setenv("TLS_CA_CERT_FILE", "")
	t.Setenv("TLS_CA_KEY_FILE", "")

	cfg := config.Config{
		Routing: config.RoutingConfig{
			Domains: []config.DomainRule{
				{Domain: "example.local"},
			},
		},
	}
	cert, mode, err := BuildCertificate(cfg)
	if err != nil {
		t.Fatalf("BuildCertificate 失败: %v", err)
	}
	if cert == nil {
		t.Fatalf("证书为空")
	}
	if mode != "self-signed" {
		t.Fatalf("mode 不匹配: got=%q", mode)
	}
}

// TestBuildCertificate_CASigned 验证配置 CA 文件时生成 CA 签发证书。
func TestBuildCertificate_CASigned(t *testing.T) {
	caCertPEM, caKeyPEM := buildCAForTest(t)
	baseDir := t.TempDir()
	certFile := filepath.Join(baseDir, "ca.crt")
	keyFile := filepath.Join(baseDir, "ca.key")
	if err := os.WriteFile(certFile, caCertPEM, 0o600); err != nil {
		t.Fatalf("写入 CA 证书失败: %v", err)
	}
	if err := os.WriteFile(keyFile, caKeyPEM, 0o600); err != nil {
		t.Fatalf("写入 CA 私钥失败: %v", err)
	}

	t.Setenv("TLS_CA_CERT_FILE", certFile)
	t.Setenv("TLS_CA_KEY_FILE", keyFile)

	cfg := config.Config{
		Routing: config.RoutingConfig{
			Domains: []config.DomainRule{
				{Domain: "example.local"},
			},
		},
	}
	cert, mode, err := BuildCertificate(cfg)
	if err != nil {
		t.Fatalf("BuildCertificate 失败: %v", err)
	}
	if cert == nil {
		t.Fatalf("证书为空")
	}
	if mode != "ca-signed" {
		t.Fatalf("mode 不匹配: got=%q", mode)
	}
}

// TestParseCertificatePEM_Error 验证证书 PEM 解析失败分支。
func TestParseCertificatePEM_Error(t *testing.T) {
	if _, err := parseCertificatePEM([]byte("not pem")); err == nil {
		t.Fatalf("应返回错误")
	}
}

// TestParsePrivateKeyPEM_Error 验证私钥 PEM 解析失败分支。
func TestParsePrivateKeyPEM_Error(t *testing.T) {
	if _, err := parsePrivateKeyPEM([]byte("not pem")); err == nil {
		t.Fatalf("应返回错误")
	}
}

// TestIssueFromCA_MissingFiles 验证 CA 文件缺失时会返回错误。
func TestIssueFromCA_MissingFiles(t *testing.T) {
	if _, err := issueFromCA([]string{"example.local"}, "missing.crt", "missing.key"); err == nil {
		t.Fatalf("缺失文件应返回错误")
	}
}

func TestIssueFromCA_ErrorBranches(t *testing.T) {
	baseDir := t.TempDir()
	certPath := filepath.Join(baseDir, "ca.crt")
	keyPath := filepath.Join(baseDir, "ca.key")

	if err := os.WriteFile(certPath, []byte("bad cert"), 0o600); err != nil {
		t.Fatalf("写入证书失败: %v", err)
	}
	if err := os.WriteFile(keyPath, []byte("bad key"), 0o600); err != nil {
		t.Fatalf("写入私钥失败: %v", err)
	}
	if _, err := issueFromCA([]string{"example.local"}, certPath, keyPath); err == nil {
		t.Fatalf("非法 CA 证书应返回错误")
	}

	caCertPEM, _ := buildCAForTest(t)
	if err := os.WriteFile(certPath, caCertPEM, 0o600); err != nil {
		t.Fatalf("写入证书失败: %v", err)
	}
	if _, err := issueFromCA([]string{"example.local"}, certPath, keyPath); err == nil {
		t.Fatalf("非法 CA 私钥应返回错误")
	}

	if err := os.Remove(keyPath); err != nil {
		t.Fatalf("删除私钥失败: %v", err)
	}
	if _, err := issueFromCA([]string{"example.local"}, certPath, keyPath); err == nil {
		t.Fatalf("缺失 CA 私钥应返回错误")
	}
}

// TestParsePrivateKeyPEM_CoversFormats 验证 PKCS8/EC/PKCS1 等私钥格式解析路径。
func TestParsePrivateKeyPEM_CoversFormats(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("生成 RSA 私钥失败: %v", err)
	}
	pkcs8, err := x509.MarshalPKCS8PrivateKey(rsaKey)
	if err != nil {
		t.Fatalf("编码 PKCS8 失败: %v", err)
	}
	pkcs8PEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8})
	if _, err = parsePrivateKeyPEM(pkcs8PEM); err != nil {
		t.Fatalf("解析 PKCS8 失败: %v", err)
	}

	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("生成 EC 私钥失败: %v", err)
	}
	ecDER, err := x509.MarshalECPrivateKey(ecKey)
	if err != nil {
		t.Fatalf("编码 EC 私钥失败: %v", err)
	}
	ecPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: ecDER})
	if _, err = parsePrivateKeyPEM(ecPEM); err != nil {
		t.Fatalf("解析 EC 私钥失败: %v", err)
	}

	pkcs1 := x509.MarshalPKCS1PrivateKey(rsaKey)
	pkcs1PEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: pkcs1})
	if _, err = parsePrivateKeyPEM(pkcs1PEM); err != nil {
		t.Fatalf("解析 PKCS1 失败: %v", err)
	}

	unknownPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("bad")})
	if _, err = parsePrivateKeyPEM(unknownPEM); err == nil {
		t.Fatalf("未知私钥内容应返回错误")
	}
}

// TestEncodeCertificate_InvalidDER 验证无效证书 DER 会导致构造 TLS 证书失败。
func TestEncodeCertificate_InvalidDER(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("生成私钥失败: %v", err)
	}
	if _, err = encodeCertificate(privateKey, []byte("bad")); err == nil {
		t.Fatalf("无效 DER 应返回错误")
	}
}

// TestEncodeCertificate_InvalidKey 验证无效私钥会导致 PEM 编码失败。
func TestEncodeCertificate_InvalidKey(t *testing.T) {
	validDER := buildCertificateDERForTest(t)
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("生成私钥失败: %v", err)
	}
	privateKey.D = big.NewInt(0)
	if _, err := encodeCertificate(privateKey, validDER); err == nil {
		t.Fatalf("无效私钥应返回错误")
	}
}

// buildCertificateDERForTest 生成用于测试的证书 DER。
// 参数：t 为测试上下文。
// 返回：证书 DER 字节。
// 异常：生成失败时终止当前测试。
func buildCertificateDERForTest(t *testing.T) []byte {
	t.Helper()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("生成私钥失败: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"example.local"},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, privateKey.Public(), privateKey)
	if err != nil {
		t.Fatalf("签发证书失败: %v", err)
	}
	return der
}

// buildCAForTest 生成用于测试的自签名 CA 证书与私钥 PEM。
// 参数：t 为测试上下文。
// 返回：CA 证书 PEM 与私钥 PEM。
// 异常：生成失败时终止当前测试。
func buildCAForTest(t *testing.T) ([]byte, []byte) {
	t.Helper()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("生成私钥失败: %v", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 64))
	if err != nil {
		t.Fatalf("生成序列号失败: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: "YoBFF Test CA",
		},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, privateKey.Public(), privateKey)
	if err != nil {
		t.Fatalf("签发 CA 证书失败: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		t.Fatalf("编码私钥失败: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return certPEM, keyPEM
}
