package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"YoBFF/internal/config"
	"YoBFF/internal/logging"

	"go.uber.org/zap"
)

type fakeServerStarter struct {
	httpErr error
	tlsErr  error
}

type closeErrorListener struct {
	net.Listener
}

func (l *closeErrorListener) Close() error {
	_ = l.Listener.Close()
	return errors.New("listener close failed")
}

// ListenAndServe 用于替代真实 HTTP 监听启动逻辑。
// 参数：srv 为 HTTP 服务对象。
// 返回：模拟的启动结果错误。
// 异常：无。
func (s *fakeServerStarter) ListenAndServe(_ *http.Server) error {
	return s.httpErr
}

// ListenAndServeTLS 用于替代真实 HTTPS 监听启动逻辑。
// 参数：srv 为 HTTPS 服务对象。
// 返回：模拟的启动结果错误。
// 异常：无。
func (s *fakeServerStarter) ListenAndServeTLS(_ *http.Server) error {
	return s.tlsErr
}

// TestShutdownServer_Nil 验证 srv 为 nil 时安全返回。
func TestShutdownServer_Nil(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	shutdownServer(ctx, zap.NewNop(), "nil", nil)
}

// TestShutdownServer_Success 验证正常 Shutdown 可以优雅退出服务。
func TestShutdownServer_Success(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("监听失败: %v", err)
	}
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {

		}
	}(listener)

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}

	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- srv.Serve(listener)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	shutdownServer(ctx, zap.NewNop(), "http", srv)

	select {
	case err = <-serveErrCh:
		if err != nil && err != http.ErrServerClosed {
			t.Fatalf("Serve 退出错误: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("服务未退出")
	}
}

// TestShutdownServer_CanceledContext 验证关闭上下文已取消时不会 panic。
func TestShutdownServer_CanceledContext(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("监听失败: %v", err)
	}
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {

		}
	}(listener)

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}

	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- srv.Serve(listener)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	shutdownServer(ctx, zap.NewNop(), "http", srv)

	_ = srv.Close()
	select {
	case <-serveErrCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("服务未退出")
	}
}

// TestRun_InvalidArgs 验证 run 的入参校验分支。
func TestRun_InvalidArgs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := run(ctx, zap.NewNop(), nil, "config.json", &fakeServerStarter{}); err == nil {
		t.Fatalf("logRuntime 为空应返回错误")
	}
	runtime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化 Runtime 失败: %v", err)
	}
	if err = run(ctx, zap.NewNop(), runtime, "", &fakeServerStarter{}); err == nil {
		t.Fatalf("configPath 为空应返回错误")
	}
}

// TestRun_ContextDone 验证 ctx.Done 分支可退出并执行关闭逻辑。
func TestRun_ContextDone(t *testing.T) {
	configPath := writeConfigFile(t, config.Config{})
	runtime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化 Runtime 失败: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err = run(ctx, zap.NewNop(), runtime, configPath, &fakeServerStarter{
		httpErr: http.ErrServerClosed,
		tlsErr:  http.ErrServerClosed,
	}); err != nil {
		t.Fatalf("run 失败: %v", err)
	}
}

// TestRun_HTTPServeError 验证服务异常退出分支可触发并安全收敛。
func TestRun_HTTPServeError(t *testing.T) {
	configPath := writeConfigFile(t, config.Config{})
	runtime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化 Runtime 失败: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err = run(ctx, zap.NewNop(), runtime, configPath, &fakeServerStarter{
		httpErr: errors.New("boom"),
		tlsErr:  http.ErrServerClosed,
	})
	if err != nil {
		t.Fatalf("run 失败: %v", err)
	}
}

// TestRun_HTTPSDisabled 验证 HTTPS 禁用分支。
func TestRun_HTTPSDisabled(t *testing.T) {
	cfg := config.Config{
		DataPlane: config.DataPlaneConfig{
			HTTPListenAddr:  ":8080",
			HTTPSListenAddr: ":8443",
			EnableHTTPS:     false,
		},
	}
	configPath := writeConfigFile(t, cfg)
	runtime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化 Runtime 失败: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err = run(ctx, zap.NewNop(), runtime, configPath, &fakeServerStarter{
		httpErr: http.ErrServerClosed,
	}); err != nil {
		t.Fatalf("run 失败: %v", err)
	}
}

// TestRun_HTTPSAddrEmpty 验证 HTTPS 监听地址为空分支。
func TestRun_HTTPSAddrEmpty(t *testing.T) {
	cfg := config.Config{
		DataPlane: config.DataPlaneConfig{
			HTTPListenAddr:  ":8080",
			HTTPSListenAddr: "",
			EnableHTTPS:     true,
		},
	}
	configPath := writeConfigFile(t, cfg)
	runtime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化 Runtime 失败: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err = run(ctx, zap.NewNop(), runtime, configPath, &fakeServerStarter{
		httpErr: http.ErrServerClosed,
	}); err != nil {
		t.Fatalf("run 失败: %v", err)
	}
}

// TestRun_AutoCertError 验证自动证书生成失败分支与跳过 HTTPS 分支。
func TestRun_AutoCertError(t *testing.T) {
	cfg := config.Config{
		DataPlane: config.DataPlaneConfig{
			HTTPListenAddr:  ":8080",
			HTTPSListenAddr: ":8443",
			EnableHTTPS:     true,
		},
	}
	configPath := writeConfigFile(t, cfg)
	runtime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化 Runtime 失败: %v", err)
	}

	t.Setenv("TLS_CA_CERT_FILE", filepath.Join(t.TempDir(), "missing.crt"))
	t.Setenv("TLS_CA_KEY_FILE", filepath.Join(t.TempDir(), "missing.key"))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err = run(ctx, zap.NewNop(), runtime, configPath, &fakeServerStarter{
		httpErr: http.ErrServerClosed,
		tlsErr:  http.ErrServerClosed,
	}); err != nil {
		t.Fatalf("run 失败: %v", err)
	}
}

// TestRun_ConfigCertificates 验证配置证书分支会走 GetCertificate 逻辑并启动 TLS 服务。
func TestRun_ConfigCertificates(t *testing.T) {
	certPEM, keyPEM := buildSelfSignedPEM(t, "example.local")
	cfg := config.Config{
		DataPlane: config.DataPlaneConfig{
			HTTPListenAddr:  ":8080",
			HTTPSListenAddr: ":8443",
			EnableHTTPS:     true,
		},
		Certificates: []config.Certificate{
			{
				Domain:  "example.local",
				CertPEM: string(certPEM),
				KeyPEM:  string(keyPEM),
			},
		},
	}
	configPath := writeConfigFile(t, cfg)
	runtime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化 Runtime 失败: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err = run(ctx, zap.NewNop(), runtime, configPath, &fakeServerStarter{
		httpErr: http.ErrServerClosed,
		tlsErr:  http.ErrServerClosed,
	}); err != nil {
		t.Fatalf("run 失败: %v", err)
	}
}

func TestDefaultServerStarter_ListenAndServe(t *testing.T) {
	srv := &http.Server{
		Addr: "127.0.0.1:0",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}
	done := make(chan error, 1)
	go func() {
		done <- defaultServerStarter{}.ListenAndServe(srv)
	}()
	time.Sleep(120 * time.Millisecond)
	if err := srv.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("关闭服务失败: %v", err)
	}
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("ListenAndServe 返回错误: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("ListenAndServe 未退出")
	}
}

func TestDefaultServerStarter_ListenAndServeTLS(t *testing.T) {
	certPEM, keyPEM := buildSelfSignedPEM(t, "localhost")
	pair, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("解析证书失败: %v", err)
	}
	srv := &http.Server{
		Addr: "127.0.0.1:0",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{pair},
		},
	}
	done := make(chan error, 1)
	go func() {
		done <- defaultServerStarter{}.ListenAndServeTLS(srv)
	}()
	time.Sleep(120 * time.Millisecond)
	if err = srv.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("关闭服务失败: %v", err)
	}
	select {
	case err = <-done:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("ListenAndServeTLS 返回错误: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("ListenAndServeTLS 未退出")
	}
}

func TestBuildUIHandler_UseEnvDirectory(t *testing.T) {
	uiDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(uiDir, "index.html"), []byte("<html>ok</html>"), 0o600); err != nil {
		t.Fatalf("写入 index 失败: %v", err)
	}
	t.Setenv("ADMIN_UI_DIR", uiDir)
	handler, err := buildUIHandler()
	if err != nil {
		t.Fatalf("buildUIHandler 失败: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	handler.ServeHTTP(rec, req)
	if rec.Result().StatusCode != http.StatusOK {
		t.Fatalf("状态码不匹配: got=%d", rec.Result().StatusCode)
	}
}

func TestBuildUIHandler_UseEmbeddedFS(t *testing.T) {
	t.Setenv("ADMIN_UI_DIR", "")
	handler, err := buildUIHandler()
	if err != nil {
		t.Fatalf("buildUIHandler 失败: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	handler.ServeHTTP(rec, req)
	if rec.Result().StatusCode != http.StatusOK {
		t.Fatalf("状态码不匹配: got=%d", rec.Result().StatusCode)
	}
}

func TestShutdownServer_ErrorBranch(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("监听失败: %v", err)
	}
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {

		}
	}(listener)
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}
	done := make(chan error, 1)
	go func() {
		done <- srv.Serve(listener)
	}()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	shutdownServer(ctx, zap.NewNop(), "http", srv)
	_ = srv.Close()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("服务未退出")
	}
}

func TestShutdownServer_DeadlineExceeded(t *testing.T) {
	startedCh := make(chan struct{}, 1)
	releaseCh := make(chan struct{})
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case startedCh <- struct{}{}:
			default:
			}
			<-releaseCh
			w.WriteHeader(http.StatusOK)
		}),
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("监听失败: %v", err)
	}
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {

		}
	}(listener)

	serveDone := make(chan error, 1)
	go func() {
		serveDone <- srv.Serve(listener)
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = http.Get("http://" + listener.Addr().String() + "/slow")
	}()

	select {
	case <-startedCh:
	case <-time.After(time.Second):
		t.Fatalf("请求未进入处理器")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	shutdownServer(ctx, zap.NewNop(), "http", srv)

	close(releaseCh)
	wg.Wait()
	_ = srv.Close()
	select {
	case serveErr := <-serveDone:
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			t.Fatalf("Serve 返回错误: %v", serveErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("服务未退出")
	}
}

func TestShutdownServer_ListenerCloseError(t *testing.T) {
	baseListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("监听失败: %v", err)
	}
	listener := &closeErrorListener{Listener: baseListener}
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}
	done := make(chan error, 1)
	go func() {
		done <- srv.Serve(listener)
	}()
	ready := false
	client := &http.Client{Timeout: 80 * time.Millisecond}
	for i := 0; i < 20; i++ {
		resp, reqErr := client.Get("http://" + baseListener.Addr().String() + "/healthz")
		if reqErr == nil {
			_ = resp.Body.Close()
			ready = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !ready {
		t.Fatalf("服务未就绪")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	shutdownServer(ctx, zap.NewNop(), "http", srv)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("服务未退出")
	}
}

func TestRun_NilLoggerAndNilStarter(t *testing.T) {
	cfg := config.Config{
		DataPlane: config.DataPlaneConfig{
			HTTPListenAddr:  "127.0.0.1:0",
			HTTPSListenAddr: "",
			EnableHTTPS:     false,
		},
		ControlPlane: config.ControlPlaneConfig{
			AdminListenAddr: "127.0.0.1:9000",
		},
	}
	configPath := writeConfigFile(t, cfg)
	runtime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化 Runtime 失败: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err = run(ctx, nil, runtime, configPath, nil); err != nil {
		t.Fatalf("run 失败: %v", err)
	}
}

func TestMain_PanicOnInvalidLogLevel(t *testing.T) {
	t.Setenv("LOG_LEVEL", "invalid-level")
	defer func() {
		if recover() == nil {
			t.Fatalf("main 应触发 panic")
		}
	}()
	main()
}

// writeConfigFile 写入测试配置文件并返回路径。
// 参数：t 为测试上下文，cfg 为待写入配置对象。
// 返回：配置文件路径。
// 异常：写入失败时终止当前测试。
func writeConfigFile(t *testing.T, cfg config.Config) string {
	t.Helper()

	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, "config.json")
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("序列化配置失败: %v", err)
	}
	if err = os.WriteFile(configPath, raw, 0o600); err != nil {
		t.Fatalf("写入配置失败: %v", err)
	}
	return configPath
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
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 64))
	if err != nil {
		t.Fatalf("生成序列号失败: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{domain},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, privateKey.Public(), privateKey)
	if err != nil {
		t.Fatalf("签发证书失败: %v", err)
	}
	certPEM := pemEncode("CERTIFICATE", der)

	keyDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		t.Fatalf("编码私钥失败: %v", err)
	}
	keyPEM := pemEncode("EC PRIVATE KEY", keyDER)
	return certPEM, keyPEM
}

// pemEncode 将 DER 编码转换为 PEM 编码字节。
// 参数：blockType 为 PEM 块类型，der 为 DER 编码内容。
// 返回：PEM 编码字节。
// 异常：无。
func pemEncode(blockType string, der []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der})
}
