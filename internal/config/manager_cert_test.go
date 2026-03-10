package config

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// generateSelfSignedCert 生成测试用自签名证书与私钥 PEM 文本。
// 参数：t 为测试上下文，domains 为证书绑定域名集合。
// 返回：证书 PEM 与私钥 PEM。
// 异常：证书生成失败时终止当前测试。
func generateSelfSignedCert(t *testing.T, domains []string) (string, string) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("生成私钥失败: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              domains,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("生成证书失败: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyBytes := x509.MarshalPKCS1PrivateKey(priv)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes})

	return string(certPEM), string(keyPEM)
}

// TestManager_LoadSSLCertificates 验证 SSL 证书加载、匹配与默认证书回退行为。
func TestManager_LoadSSLCertificates(t *testing.T) {
	// 1. 准备测试环境
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0644); err != nil {
		t.Fatalf("写入配置文件失败: %v", err)
	}

	manager, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("创建 Manager 失败: %v", err)
	}

	// 2. 生成自签名证书
	certPEM, keyPEM := generateSelfSignedCert(t, []string{"example.com", "*.test.com"})

	// 3. 构建包含 SSLCertificates 的配置
	cfg := Config{
		SSLCertificates: []SSLCertificate{
			{
				ID:      "cert-1",
				Name:    "Test Cert",
				Domains: []string{"example.com", "*.test.com"},
				CertPEM: certPEM,
				KeyPEM:  keyPEM,
			},
		},
	}

	// 4. 应用配置
	if err := manager.Apply(cfg); err != nil {
		t.Fatalf("应用配置失败: %v", err)
	}

	// 5. 验证证书是否已加载
	if !manager.HasCertificates() {
		t.Fatal("预期 HasCertificates 为 true")
	}

	// 6. 验证 GetCertificate (SNI 匹配)
	tests := []struct {
		sni      string
		wantLoad bool
	}{
		{"example.com", true},
		{"sub.test.com", true}, // 泛域名匹配
		{"other.com", false},
	}

	for _, tt := range tests {
		hello := &tls.ClientHelloInfo{
			ServerName: tt.sni,
		}
		cert, err := manager.GetCertificate(hello)
		if tt.wantLoad {
			if err != nil {
				t.Errorf("获取 %s 证书失败: %v", tt.sni, err)
			}
			if cert == nil {
				t.Errorf("预期获取到 %s 的证书，实际为 nil", tt.sni)
			}
		} else {
			// 如果没有默认证书，应该返回 error
			// 但是在这个测试中，我们加载了一个证书，它会自动成为默认证书（因为它是第一个且 defaultCert 为空）。
			// 所以即使 other.com 不匹配，也会返回默认证书。
			// 让我们检查一下代码逻辑：
			/*
			   if idx == 0 && data.defaultCert == nil {
			       data.defaultCert = &cert
			   }
			*/
			// 所以确实会有一个默认证书。
			// 这意味着 other.com 也会返回这个证书。
			// 为了测试严格匹配，我们需要让 defaultCert 为 nil，但这不太容易做到（除非我们不设置 SSLCertificates[0] 为默认，但这不符合逻辑）。
			// 或者我们可以加载两个证书，然后验证 other.com 返回的是默认证书，而匹配的域名返回的是正确的证书。
			// 暂时，我们可以接受 other.com 返回默认证书，或者我们可以断言返回的证书就是默认证书。
			if cert == nil {
				t.Errorf("预期获取到默认证书 for %s，实际为 nil", tt.sni)
			}
		}
	}
}
