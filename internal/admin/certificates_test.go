package admin

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"YoBFF/internal/config"
	"YoBFF/internal/logging"
	"YoBFF/internal/store"

	"go.uber.org/zap"
)

func buildCertHandler(t *testing.T) http.Handler {
	t.Helper()
	t.Setenv("ADMIN_API_TOKEN", "token")
	t.Setenv("ADMIN_RATE_LIMIT_PER_MIN", "60")
	manager := buildManagerForTest(t, `{}`)
	logRuntime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化日志运行时失败: %v", err)
	}
	pipeline := logging.NewPipeline(16, zap.NewNop())
	t.Cleanup(func() {
		pipeline.Close()
	})
	dbPath := filepath.Join(t.TempDir(), "admin-cert.db")
	siteStore, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("初始化证书存储失败: %v", err)
	}
	t.Cleanup(func() {
		_ = siteStore.Close()
	})
	srv := NewServer(manager, logRuntime, pipeline, siteStore)
	return srv.Handler()
}

func buildCertServer(t *testing.T) (*Server, *store.Store) {
	t.Helper()
	t.Setenv("ADMIN_API_TOKEN", "token")
	t.Setenv("ADMIN_RATE_LIMIT_PER_MIN", "60")
	manager := buildManagerForTest(t, `{}`)
	logRuntime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化日志运行时失败: %v", err)
	}
	pipeline := logging.NewPipeline(16, zap.NewNop())
	t.Cleanup(func() {
		pipeline.Close()
	})
	dbPath := filepath.Join(t.TempDir(), "admin-cert-direct.db")
	siteStore, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("初始化证书存储失败: %v", err)
	}
	t.Cleanup(func() {
		_ = siteStore.Close()
	})
	return NewServer(manager, logRuntime, pipeline, siteStore), siteStore
}

func requestAuthJSON(t *testing.T, handler http.Handler, method string, target string, contentType string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "http://example.com"+target, bytes.NewReader(body))
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-Admin-User", "tester")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func buildCertificatePEM(t *testing.T) ([]byte, []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("生成私钥失败: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "example.com"},
		DNSNames:              []string{"example.com", "www.example.com"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("签发证书失败: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	return certPEM, keyPEM
}

func TestCertificateRoutes_UploadListDelete(t *testing.T) {
	handler := buildCertHandler(t)
	certPEM, keyPEM := buildCertificatePEM(t)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("name", "cert-a"); err != nil {
		t.Fatalf("写入字段失败: %v", err)
	}
	certPart, err := writer.CreateFormFile("cert", "cert.pem")
	if err != nil {
		t.Fatalf("创建证书文件失败: %v", err)
	}
	if _, err = certPart.Write(certPEM); err != nil {
		t.Fatalf("写入证书内容失败: %v", err)
	}
	keyPart, err := writer.CreateFormFile("key", "key.pem")
	if err != nil {
		t.Fatalf("创建私钥文件失败: %v", err)
	}
	if _, err = keyPart.Write(keyPEM); err != nil {
		t.Fatalf("写入私钥内容失败: %v", err)
	}
	if err = writer.Close(); err != nil {
		t.Fatalf("关闭上传体失败: %v", err)
	}

	uploadRec := requestAuthJSON(t, handler, http.MethodPost, "/api/v1/certs", writer.FormDataContentType(), body.Bytes())
	if uploadRec.Result().StatusCode != http.StatusOK {
		t.Fatalf("上传证书状态码不匹配: got=%d body=%s", uploadRec.Result().StatusCode, uploadRec.Body.String())
	}
	var uploadPayload map[string]any
	if err = json.NewDecoder(uploadRec.Result().Body).Decode(&uploadPayload); err != nil {
		t.Fatalf("解析上传响应失败: %v", err)
	}
	certID, _ := uploadPayload["id"].(string)
	if certID == "" {
		t.Fatalf("证书ID为空: %#v", uploadPayload)
	}

	listRec := requestAuthJSON(t, handler, http.MethodGet, "/api/v1/certs?page=1&pageSize=10", "", nil)
	if listRec.Result().StatusCode != http.StatusOK {
		t.Fatalf("查询证书列表状态码不匹配: got=%d", listRec.Result().StatusCode)
	}
	var listPayload map[string]any
	if err = json.NewDecoder(listRec.Result().Body).Decode(&listPayload); err != nil {
		t.Fatalf("解析列表响应失败: %v", err)
	}
	items, ok := listPayload["items"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("证书列表为空: %#v", listPayload)
	}

	deleteRec := requestAuthJSON(t, handler, http.MethodDelete, "/api/v1/certs/"+certID, "", nil)
	if deleteRec.Result().StatusCode != http.StatusNoContent {
		t.Fatalf("删除证书状态码不匹配: got=%d", deleteRec.Result().StatusCode)
	}
}

func TestCertificateRoutes_ErrorsAndParser(t *testing.T) {
	handler := buildCertHandler(t)

	methodRec := requestAuthJSON(t, handler, http.MethodPut, "/api/v1/certs", "", nil)
	if methodRec.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("非法方法状态码不匹配: got=%d", methodRec.Result().StatusCode)
	}

	missingIDRec := requestAuthJSON(t, handler, http.MethodDelete, "/api/v1/certs/", "", nil)
	if missingIDRec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("缺少证书ID状态码不匹配: got=%d", missingIDRec.Result().StatusCode)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("name", "bad")
	_ = writer.Close()
	badUploadRec := requestAuthJSON(t, handler, http.MethodPost, "/api/v1/certs", writer.FormDataContentType(), body.Bytes())
	if badUploadRec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("缺少证书文件状态码不匹配: got=%d", badUploadRec.Result().StatusCode)
	}

	_, _, _, err := parseCertInfo([]byte("bad-pem"))
	if err == nil || !strings.Contains(err.Error(), "decode") {
		t.Fatalf("解析错误不匹配: %v", err)
	}
}

func TestCertificateRoutes_UploadErrorPathsAndDeleteError(t *testing.T) {
	t.Setenv("ADMIN_API_TOKEN", "token")
	t.Setenv("ADMIN_RATE_LIMIT_PER_MIN", "60")
	manager := buildManagerForTest(t, `{}`)
	logRuntime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化日志运行时失败: %v", err)
	}
	pipeline := logging.NewPipeline(16, zap.NewNop())
	t.Cleanup(func() {
		pipeline.Close()
	})
	dbPath := filepath.Join(t.TempDir(), "admin-cert-errors.db")
	siteStore, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("初始化证书存储失败: %v", err)
	}
	t.Cleanup(func() {
		_ = siteStore.Close()
	})
	handler := NewServer(manager, logRuntime, pipeline, siteStore).Handler()

	noNameBody := &bytes.Buffer{}
	noNameWriter := multipart.NewWriter(noNameBody)
	certPart, _ := noNameWriter.CreateFormFile("cert", "cert.pem")
	_, _ = certPart.Write([]byte("dummy"))
	keyPart, _ := noNameWriter.CreateFormFile("key", "key.pem")
	_, _ = keyPart.Write([]byte("dummy"))
	_ = noNameWriter.Close()
	noNameRec := requestAuthJSON(t, handler, http.MethodPost, "/api/v1/certs", noNameWriter.FormDataContentType(), noNameBody.Bytes())
	if noNameRec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("缺少 name 状态码不匹配: got=%d", noNameRec.Result().StatusCode)
	}

	missingKeyBody := &bytes.Buffer{}
	missingKeyWriter := multipart.NewWriter(missingKeyBody)
	_ = missingKeyWriter.WriteField("name", "cert-missing-key")
	onlyCertPart, _ := missingKeyWriter.CreateFormFile("cert", "cert.pem")
	_, _ = onlyCertPart.Write([]byte("dummy"))
	_ = missingKeyWriter.Close()
	missingKeyRec := requestAuthJSON(t, handler, http.MethodPost, "/api/v1/certs", missingKeyWriter.FormDataContentType(), missingKeyBody.Bytes())
	if missingKeyRec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("缺少 key 状态码不匹配: got=%d", missingKeyRec.Result().StatusCode)
	}

	invalidCertBody := &bytes.Buffer{}
	invalidCertWriter := multipart.NewWriter(invalidCertBody)
	_ = invalidCertWriter.WriteField("name", "cert-invalid")
	invalidCertPart, _ := invalidCertWriter.CreateFormFile("cert", "cert.pem")
	_, _ = invalidCertPart.Write([]byte("not-a-pem"))
	invalidKeyPart, _ := invalidCertWriter.CreateFormFile("key", "key.pem")
	_, _ = invalidKeyPart.Write([]byte("not-a-key"))
	_ = invalidCertWriter.Close()
	invalidCertRec := requestAuthJSON(t, handler, http.MethodPost, "/api/v1/certs", invalidCertWriter.FormDataContentType(), invalidCertBody.Bytes())
	if invalidCertRec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("无效证书状态码不匹配: got=%d", invalidCertRec.Result().StatusCode)
	}

	if err = siteStore.Close(); err != nil {
		t.Fatalf("关闭存储失败: %v", err)
	}
	listRec := requestAuthJSON(t, handler, http.MethodGet, "/api/v1/certs?page=1&pageSize=10", "", nil)
	if listRec.Result().StatusCode != http.StatusInternalServerError {
		t.Fatalf("查询失败状态码不匹配: got=%d", listRec.Result().StatusCode)
	}
	deleteRec := requestAuthJSON(t, handler, http.MethodDelete, "/api/v1/certs/any-id", "", nil)
	if deleteRec.Result().StatusCode != http.StatusInternalServerError {
		t.Fatalf("删除失败状态码不匹配: got=%d", deleteRec.Result().StatusCode)
	}
}

func TestCertificateMethods_ListDeleteAndUploadBranches(t *testing.T) {
	srv, siteStore := buildCertServer(t)
	certPEM, keyPEM := buildCertificatePEM(t)

	seedCert := storeConfigCert("seed", certPEM, keyPEM)
	if err := siteStore.CreateCertificate(&seedCert); err != nil {
		t.Fatalf("写入证书失败: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, "http://example.com/api/v1/certs?page=0&pageSize=999", nil)
	listRec := httptest.NewRecorder()
	srv.listCertificates(listRec, listReq)
	if listRec.Result().StatusCode != http.StatusOK {
		t.Fatalf("列表查询状态码不匹配: got=%d", listRec.Result().StatusCode)
	}
	var listPayload map[string]any
	if err := json.NewDecoder(listRec.Result().Body).Decode(&listPayload); err != nil {
		t.Fatalf("解析列表响应失败: %v", err)
	}
	if gotPage, _ := listPayload["page"].(float64); gotPage != 1 {
		t.Fatalf("分页默认值不匹配: got=%v", listPayload["page"])
	}

	listReq2 := httptest.NewRequest(http.MethodGet, "http://example.com/api/v1/certs?page=2&pageSize=1", nil)
	listRec2 := httptest.NewRecorder()
	srv.listCertificates(listRec2, listReq2)
	if listRec2.Result().StatusCode != http.StatusOK {
		t.Fatalf("列表查询状态码不匹配: got=%d", listRec2.Result().StatusCode)
	}
	var listPayload2 struct {
		Items []config.SSLCertificate `json:"items"`
		Page  int                     `json:"page"`
	}
	if err := json.NewDecoder(listRec2.Result().Body).Decode(&listPayload2); err != nil {
		t.Fatalf("解析列表响应失败: %v", err)
	}
	if listPayload2.Page != 2 {
		t.Fatalf("分页参数不匹配: got=%d", listPayload2.Page)
	}
	if len(listPayload2.Items) > 0 && listPayload2.Items[0].KeyPEM != "" {
		t.Fatalf("列表不应返回私钥")
	}

	delReq := httptest.NewRequest(http.MethodDelete, "http://example.com/api/v1/certs/seed", nil)
	delReq.Header.Set("X-Operator", "tester")
	delRec := httptest.NewRecorder()
	srv.deleteCertificate(delRec, delReq, "seed")
	if delRec.Result().StatusCode != http.StatusNoContent {
		t.Fatalf("删除状态码不匹配: got=%d", delRec.Result().StatusCode)
	}

	tooLargeReq := httptest.NewRequest(http.MethodPost, "http://example.com/api/v1/certs", bytes.NewReader(bytes.Repeat([]byte("a"), (1<<20)+1024)))
	tooLargeReq.Header.Set("Content-Type", "multipart/form-data; boundary=test")
	tooLargeRec := httptest.NewRecorder()
	srv.uploadCertificate(tooLargeRec, tooLargeReq)
	if tooLargeRec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("大请求状态码不匹配: got=%d", tooLargeRec.Result().StatusCode)
	}

	missingCertBody := &bytes.Buffer{}
	missingCertWriter := multipart.NewWriter(missingCertBody)
	_ = missingCertWriter.WriteField("name", "missing-cert")
	missingCertKey, _ := missingCertWriter.CreateFormFile("key", "key.pem")
	_, _ = missingCertKey.Write(keyPEM)
	_ = missingCertWriter.Close()
	missingCertReq := httptest.NewRequest(http.MethodPost, "http://example.com/api/v1/certs", bytes.NewReader(missingCertBody.Bytes()))
	missingCertReq.Header.Set("Content-Type", missingCertWriter.FormDataContentType())
	missingCertRec := httptest.NewRecorder()
	srv.uploadCertificate(missingCertRec, missingCertReq)
	if missingCertRec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("缺少 cert 状态码不匹配: got=%d", missingCertRec.Result().StatusCode)
	}

	successBody := &bytes.Buffer{}
	successWriter := multipart.NewWriter(successBody)
	_ = successWriter.WriteField("name", "success-cert")
	successCertPart, _ := successWriter.CreateFormFile("cert", "cert.pem")
	_, _ = successCertPart.Write(certPEM)
	successKeyPart, _ := successWriter.CreateFormFile("key", "key.pem")
	_, _ = successKeyPart.Write(keyPEM)
	_ = successWriter.Close()
	successReq := httptest.NewRequest(http.MethodPost, "http://example.com/api/v1/certs", bytes.NewReader(successBody.Bytes()))
	successReq.Header.Set("Content-Type", successWriter.FormDataContentType())
	successRec := httptest.NewRecorder()
	srv.uploadCertificate(successRec, successReq)
	if successRec.Result().StatusCode != http.StatusOK {
		t.Fatalf("上传成功状态码不匹配: got=%d", successRec.Result().StatusCode)
	}

	if err := siteStore.Close(); err != nil {
		t.Fatalf("关闭存储失败: %v", err)
	}

	listErrRec := httptest.NewRecorder()
	srv.listCertificates(listErrRec, listReq)
	if listErrRec.Result().StatusCode != http.StatusInternalServerError {
		t.Fatalf("列表错误分支状态码不匹配: got=%d", listErrRec.Result().StatusCode)
	}

	delErrRec := httptest.NewRecorder()
	srv.deleteCertificate(delErrRec, delReq, "seed")
	if delErrRec.Result().StatusCode != http.StatusInternalServerError {
		t.Fatalf("删除错误分支状态码不匹配: got=%d", delErrRec.Result().StatusCode)
	}

	validBody := &bytes.Buffer{}
	validWriter := multipart.NewWriter(validBody)
	_ = validWriter.WriteField("name", "valid")
	partCert, _ := validWriter.CreateFormFile("cert", "cert.pem")
	_, _ = partCert.Write(certPEM)
	partKey, _ := validWriter.CreateFormFile("key", "key.pem")
	_, _ = partKey.Write(keyPEM)
	_ = validWriter.Close()

	uploadErrReq := httptest.NewRequest(http.MethodPost, "http://example.com/api/v1/certs", bytes.NewReader(validBody.Bytes()))
	uploadErrReq.Header.Set("Content-Type", validWriter.FormDataContentType())
	uploadErrRec := httptest.NewRecorder()
	srv.uploadCertificate(uploadErrRec, uploadErrReq)
	if uploadErrRec.Result().StatusCode != http.StatusInternalServerError {
		t.Fatalf("上传保存失败状态码不匹配: got=%d", uploadErrRec.Result().StatusCode)
	}
}

func storeConfigCert(id string, certPEM []byte, keyPEM []byte) config.SSLCertificate {
	return config.SSLCertificate{
		ID:        id,
		Name:      id,
		Domains:   []string{"example.com"},
		NotAfter:  time.Now().UTC().Add(time.Hour),
		Issuer:    "test",
		CertPEM:   string(certPEM),
		KeyPEM:    string(keyPEM),
		CreatedAt: time.Now().UTC(),
	}
}
