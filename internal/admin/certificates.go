package admin

import (
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

	// 解析证书信息
	domains, notAfter, issuer, err := parseCertInfo(certBytes)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_certificate", fmt.Sprintf("invalid certificate: %v", err), r)
		return
	}

	cert := &config.SSLCertificate{
		ID:        fmt.Sprintf("cert_%d", time.Now().UnixNano()),
		Name:      name,
		Domains:   domains,
		NotAfter:  notAfter,
		Issuer:    issuer,
		CertPEM:   string(certBytes),
		KeyPEM:    string(keyBytes),
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

// parseCertInfo 解析 PEM 格式证书，提取域名、有效期和颁发者。
func parseCertInfo(certPEM []byte) (domains []string, notAfter time.Time, issuer string, err error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, time.Time{}, "", fmt.Errorf("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, time.Time{}, "", err
	}

	domains = append(domains, cert.Subject.CommonName)
	domains = append(domains, cert.DNSNames...)
	// 去重
	unique := make(map[string]bool)
	var cleanDomains []string
	for _, d := range domains {
		if d != "" && !unique[d] {
			unique[d] = true
			cleanDomains = append(cleanDomains, d)
		}
	}

	return cleanDomains, cert.NotAfter, cert.Issuer.CommonName, nil
}
