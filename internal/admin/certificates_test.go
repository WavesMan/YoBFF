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
