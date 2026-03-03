package tlsutil

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"YoBFF/internal/config"
)

// BuildCertificate 根据环境变量与路由域名构建运行时证书。
// 参数：cfg 为当前配置快照。
// 返回：证书实例、签发模式与错误信息。
// 异常：CA 文件无效或证书签发失败时返回错误。
func BuildCertificate(cfg config.Config) (*tls.Certificate, string, error) {
	domains := collectDomains(cfg)
	caCertFile := strings.TrimSpace(os.Getenv("TLS_CA_CERT_FILE"))
	caKeyFile := strings.TrimSpace(os.Getenv("TLS_CA_KEY_FILE"))

	if caCertFile != "" && caKeyFile != "" {
		cert, err := issueFromCA(domains, caCertFile, caKeyFile)
		if err != nil {
			return nil, "", err
		}
		return cert, "ca-signed", nil
	}

	cert, err := issueSelfSigned(domains)
	if err != nil {
		return nil, "", err
	}
	return cert, "self-signed", nil
}

// collectDomains 归集可用于证书 SAN 的域名列表。
// 参数：cfg 为当前配置。
// 返回：去重后的域名集合，至少包含 localhost。
// 异常：无。
func collectDomains(cfg config.Config) []string {
	seen := make(map[string]struct{})
	domains := make([]string, 0, len(cfg.Routing.Domains)+1)
	for _, rule := range cfg.Routing.Domains {
		domain := strings.TrimSpace(strings.ToLower(rule.Domain))
		if domain == "" {
			continue
		}
		if _, exists := seen[domain]; exists {
			continue
		}
		seen[domain] = struct{}{}
		domains = append(domains, domain)
	}
	if len(domains) == 0 {
		domains = append(domains, "localhost")
	}
	return domains
}

// issueSelfSigned 生成自签名服务器证书。
// 参数：domains 为证书 SAN 域名集合。
// 返回：可直接用于 TLS 握手的证书。
// 异常：随机数生成或证书编码失败时返回错误。
func issueSelfSigned(domains []string) (*tls.Certificate, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("生成私钥失败: %w", err)
	}

	template, err := buildServerTemplate(domains, "YoBFF Self Signed")
	if err != nil {
		return nil, err
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, privateKey.Public(), privateKey)
	if err != nil {
		return nil, fmt.Errorf("生成自签证书失败: %w", err)
	}
	return encodeCertificate(privateKey, der)
}

// issueFromCA 使用外部 CA 证书与私钥签发服务器证书。
// 参数：domains 为证书 SAN 域名集合，caCertFile 与 caKeyFile 为 CA 文件路径。
// 返回：可直接用于 TLS 握手的证书。
// 异常：CA 文件读取、解析或签发失败时返回错误。
func issueFromCA(domains []string, caCertFile string, caKeyFile string) (*tls.Certificate, error) {
	caCertPEM, err := os.ReadFile(caCertFile)
	if err != nil {
		return nil, fmt.Errorf("读取 CA 证书失败: %w", err)
	}
	caKeyPEM, err := os.ReadFile(caKeyFile)
	if err != nil {
		return nil, fmt.Errorf("读取 CA 私钥失败: %w", err)
	}

	caCert, err := parseCertificatePEM(caCertPEM)
	if err != nil {
		return nil, fmt.Errorf("解析 CA 证书失败: %w", err)
	}
	caKey, err := parsePrivateKeyPEM(caKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("解析 CA 私钥失败: %w", err)
	}

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("生成服务器私钥失败: %w", err)
	}
	template, err := buildServerTemplate(domains, "YoBFF CA Signed")
	if err != nil {
		return nil, err
	}
	der, err := x509.CreateCertificate(rand.Reader, template, caCert, privateKey.Public(), caKey)
	if err != nil {
		return nil, fmt.Errorf("CA 签发服务器证书失败: %w", err)
	}
	return encodeCertificate(privateKey, der)
}

// buildServerTemplate 构建服务器证书模板。
// 参数：domains 为 SAN 列表，commonName 为证书主体名称。
// 返回：可用于签发的模板对象。
// 异常：序列号生成失败时返回错误。
func buildServerTemplate(domains []string, commonName string) (*x509.Certificate, error) {
	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return nil, fmt.Errorf("生成证书序列号失败: %w", err)
	}
	return &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: commonName,
		},
		NotBefore:             time.Now().Add(-10 * time.Minute),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              domains,
	}, nil
}

// parseCertificatePEM 将 PEM 证书文本解析为 x509.Certificate。
// 参数：raw 为证书 PEM 数据。
// 返回：解析后的证书对象。
// 异常：格式非法时返回错误。
func parseCertificatePEM(raw []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("PEM 证书为空")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	return cert, nil
}

// parsePrivateKeyPEM 解析 PEM 私钥并返回 crypto.Signer。
// 参数：raw 为私钥 PEM 数据。
// 返回：签名私钥对象。
// 异常：格式不受支持时返回错误。
func parsePrivateKeyPEM(raw []byte) (crypto.Signer, error) {
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("PEM 私钥为空")
	}

	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		signer, ok := key.(crypto.Signer)
		if !ok {
			return nil, fmt.Errorf("PKCS8 私钥类型不支持")
		}
		return signer, nil
	}
	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	return nil, fmt.Errorf("无法识别私钥格式")
}

// encodeCertificate 将证书与私钥编码为 tls.Certificate。
// 参数：privateKey 为私钥对象，certificateDER 为证书 DER 字节。
// 返回：TLS 证书对象。
// 异常：PEM 编码失败时返回错误。
func encodeCertificate(privateKey *ecdsa.PrivateKey, certificateDER []byte) (*tls.Certificate, error) {
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certificateDER,
	})
	keyDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("编码私钥失败: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyDER,
	})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("构造 TLS 证书失败: %w", err)
	}
	return &cert, nil
}
