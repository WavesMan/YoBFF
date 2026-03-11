package admin

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"YoBFF/internal/config"

	"go.uber.org/zap"
)

type sslCertificateView struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Domains   []string  `json:"domains"`
	NotAfter  time.Time `json:"notAfter"`
	Issuer    string    `json:"issuer"`
	CreatedAt time.Time `json:"createdAt"`
}

func toSSLCertificateView(cert config.SSLCertificate) sslCertificateView {
	return sslCertificateView{
		ID:        cert.ID,
		Name:      cert.Name,
		Domains:   cert.Domains,
		NotAfter:  cert.NotAfter,
		Issuer:    cert.Issuer,
		CreatedAt: cert.CreatedAt,
	}
}

// handleCertificates 处理证书列表查询与上传。
// GET /api/v1/certs: 分页查询证书。
// POST /api/v1/certs: 上传新证书。
func (s *Server) handleCertificates(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "store_unavailable", "site store unavailable", r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.listCertificates(w, r)
	case http.MethodPost:
		s.uploadCertificate(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
	}
}

// handleCertificateDetail 处理单个证书的查询与删除。
// DELETE /api/v1/certs/{id}: 删除指定证书。
func (s *Server) handleCertificateDetail(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		writeError(w, http.StatusServiceUnavailable, "store_unavailable", "site store unavailable", r)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/certs/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "missing certificate id", r)
		return
	}

	switch r.Method {
	case http.MethodDelete:
		s.deleteCertificate(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", r)
	}
}

// listCertificates 分页返回已上传的 SSL 证书列表。
func (s *Server) listCertificates(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 15
	}

	certs, total, err := s.store.ListCertificates(page, pageSize)
	if err != nil {
		s.runtime.Logger().Error("failed to list certificates", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list certificates", r)
		return
	}

	views := make([]sslCertificateView, 0, len(certs))
	for _, cert := range certs {
		views = append(views, toSSLCertificateView(cert))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": views,
		"total": total,
		"page":  page,
	})
}

// uploadCertificate 接收并解析上传的 SSL 证书文件。
func (s *Server) uploadCertificate(w http.ResponseWriter, r *http.Request) {
	// 限制上传大小 1MB
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "file too large or invalid format", r)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "name is required", r)
		return
	}

	certFile, _, err := r.FormFile("cert")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "cert file is required", r)
		return
	}
	defer certFile.Close()

	keyFile, _, err := r.FormFile("key")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "key file is required", r)
		return
	}
	defer keyFile.Close()

	certBytes, err := io.ReadAll(certFile)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "failed to read cert file", r)
		return
	}
	keyBytes, err := io.ReadAll(keyFile)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "failed to read key file", r)
		return
	}

	certPEM, parsedCert, err := normalizeCertificateBytes(certBytes)
	if err != nil {
		message := fmt.Sprintf("invalid certificate: %v", err)
		if strings.Contains(err.Error(), "failed to decode PEM block") {
			message = "invalid certificate: 证书文件格式不正确，请上传 PEM/DER 编码的 .pem/.crt/.cer"
		}
		writeError(w, http.StatusBadRequest, "invalid_certificate", message, r)
		return
	}
	keyPEM, err := normalizePrivateKeyBytes(keyBytes)
	if err != nil {
		message := fmt.Sprintf("invalid private key: %v", err)
		if strings.Contains(err.Error(), "failed to decode PEM block") {
			message = "invalid private key: 私钥文件格式不正确，请上传 PEM/DER 编码的 .key/.pem"
		}
		writeError(w, http.StatusBadRequest, "invalid_certificate", message, r)
		return
	}
	if _, err := tls.X509KeyPair(certPEM, keyPEM); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_certificate", fmt.Sprintf("invalid certificate: 证书与私钥不匹配或格式不正确: %v", err), r)
		return
	}

	domains, notAfter, issuer := extractCertInfo(parsedCert)

	cert := &config.SSLCertificate{
		ID:        fmt.Sprintf("cert_%d", time.Now().UnixNano()),
		Name:      name,
		Domains:   domains,
		NotAfter:  notAfter,
		Issuer:    issuer,
		CertPEM:   string(certPEM),
		KeyPEM:    string(keyPEM),
		CreatedAt: time.Now(),
	}

	if err := s.store.CreateCertificate(cert); err != nil {
		s.runtime.Logger().Error("failed to save certificate", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to save certificate", r)
		return
	}

	s.store.SaveAudit("create_cert", cert.ID, operatorFromRequest(r), map[string]any{"name": name})
	writeJSON(w, http.StatusOK, toSSLCertificateView(*cert))
}

// deleteCertificate 删除指定的 SSL 证书。
func (s *Server) deleteCertificate(w http.ResponseWriter, r *http.Request, id string) {
	if err := s.store.DeleteCertificate(id); err != nil {
		s.runtime.Logger().Error("failed to delete certificate", zap.Error(err), zap.String("id", id))
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete certificate", r)
		return
	}
	s.store.SaveAudit("delete_cert", id, operatorFromRequest(r), nil)
	w.WriteHeader(http.StatusNoContent)
}

// normalizeCertificateBytes 将证书输入转换为可用于 TLS 解析的 PEM 字节并返回首张证书对象。
// 参数：raw 为证书文件字节，支持 PEM/DER 编码。
// 返回：规范化后的证书 PEM、首张证书对象。
// 异常：格式非法时返回错误。
func normalizeCertificateBytes(raw []byte) ([]byte, *x509.Certificate, error) {
	rest := raw
	var pemCert []byte
	var firstCert *x509.Certificate
	for len(rest) > 0 {
		block, next := pem.Decode(rest)
		if block == nil {
			break
		}
		rest = next
		if block.Type != "CERTIFICATE" {
			continue
		}
		pemCert = append(pemCert, pem.EncodeToMemory(block)...)
		if firstCert == nil {
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, nil, err
			}
			firstCert = cert
		}
	}
	if len(pemCert) > 0 && firstCert != nil {
		return pemCert, firstCert, nil
	}

	cert, err := x509.ParseCertificate(raw)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode PEM block")
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}), cert, nil
}

// normalizePrivateKeyBytes 将私钥输入转换为可用于 TLS 解析的 PEM 字节。
// 参数：raw 为私钥文件字节，支持 PEM/DER 编码。
// 返回：规范化后的私钥 PEM。
// 异常：格式非法时返回错误。
func normalizePrivateKeyBytes(raw []byte) ([]byte, error) {
	block, _ := pem.Decode(raw)
	if block != nil {
		return pem.EncodeToMemory(block), nil
	}

	if _, err := x509.ParsePKCS8PrivateKey(raw); err == nil {
		return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: raw}), nil
	}
	if _, err := x509.ParseECPrivateKey(raw); err == nil {
		return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: raw}), nil
	}
	if _, err := x509.ParsePKCS1PrivateKey(raw); err == nil {
		return pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: raw}), nil
	}
	return nil, fmt.Errorf("failed to decode PEM block")
}

func extractCertInfo(cert *x509.Certificate) (domains []string, notAfter time.Time, issuer string) {
	domains = append(domains, cert.Subject.CommonName)
	domains = append(domains, cert.DNSNames...)
	unique := make(map[string]bool)
	cleanDomains := make([]string, 0, len(domains))
	for _, domain := range domains {
		value := strings.TrimSpace(domain)
		if value == "" {
			continue
		}
		if unique[value] {
			continue
		}
		unique[value] = true
		cleanDomains = append(cleanDomains, value)
	}
	return cleanDomains, cert.NotAfter, cert.Issuer.CommonName
}

func parseCertInfo(certRaw []byte) (domains []string, notAfter time.Time, issuer string, err error) {
	_, cert, err := normalizeCertificateBytes(certRaw)
	if err != nil {
		return nil, time.Time{}, "", err
	}
	domains, notAfter, issuer = extractCertInfo(cert)
	return domains, notAfter, issuer, nil
}
