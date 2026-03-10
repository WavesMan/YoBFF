package app

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestNewUIHandler_ErrorPaths(t *testing.T) {
	if _, err := NewUIHandler(nil); err == nil {
		t.Fatalf("nil 文件系统应返回错误")
	}
	fsys := fstest.MapFS{
		"assets/app.js": &fstest.MapFile{Data: []byte("console.log(1)")},
	}
	if _, err := NewUIHandler(fsys); err == nil {
		t.Fatalf("缺少 index.html 应返回错误")
	}
}

func TestNewUIHandler_ServeIndexAndAssets(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html":    &fstest.MapFile{Data: []byte("<html>index</html>")},
		"assets/app.js": &fstest.MapFile{Data: []byte("console.log('ok')")},
	}
	handler, err := NewUIHandler(fsys)
	if err != nil {
		t.Fatalf("初始化 UI 处理器失败: %v", err)
	}

	cases := []struct {
		name       string
		method     string
		target     string
		statusCode int
		bodyLike   string
	}{
		{
			name:       "root serves index",
			method:     http.MethodGet,
			target:     "/",
			statusCode: http.StatusOK,
			bodyLike:   "index",
		},
		{
			name:       "directory serves index",
			method:     http.MethodGet,
			target:     "/admin/",
			statusCode: http.StatusOK,
			bodyLike:   "index",
		},
		{
			name:       "asset serves file",
			method:     http.MethodGet,
			target:     "/assets/app.js",
			statusCode: http.StatusOK,
			bodyLike:   "console.log",
		},
		{
			name:       "missing path fallback index",
			method:     http.MethodGet,
			target:     "/dashboard",
			statusCode: http.StatusOK,
			bodyLike:   "index",
		},
		{
			name:       "head allowed",
			method:     http.MethodHead,
			target:     "/",
			statusCode: http.StatusOK,
		},
		{
			name:       "post denied",
			method:     http.MethodPost,
			target:     "/",
			statusCode: http.StatusMethodNotAllowed,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "http://example.com"+tt.target, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Result().StatusCode != tt.statusCode {
				t.Fatalf("状态码不匹配: got=%d", rec.Result().StatusCode)
			}
			if tt.bodyLike != "" && !strings.Contains(rec.Body.String(), tt.bodyLike) {
				t.Fatalf("响应体不匹配: body=%q", rec.Body.String())
			}
		})
	}
}

func TestServeIndex_ErrorPaths(t *testing.T) {
	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	serveIndex(rec1, req1, fstest.MapFS{})
	if rec1.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("缺失文件状态码不匹配: got=%d", rec1.Result().StatusCode)
	}

	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "index.html")
	if err := os.WriteFile(indexPath, []byte("x"), 0o600); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}
	realFS := os.DirFS(tmpDir)
	infoErrFS := errorFileStatFS{FS: realFS}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	serveIndex(rec2, req2, infoErrFS)
	if rec2.Result().StatusCode != http.StatusInternalServerError {
		t.Fatalf("文件状态错误码不匹配: got=%d", rec2.Result().StatusCode)
	}
}

func TestFileExists(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("ok")},
		"assets":     &fstest.MapFile{Mode: fs.ModeDir},
	}
	if fileExists(fsys, "") {
		t.Fatalf("空路径不应存在")
	}
	if !fileExists(fsys, "index.html") {
		t.Fatalf("文件应存在")
	}
	if fileExists(fsys, "assets") {
		t.Fatalf("目录不应被视为文件")
	}
	if fileExists(fsys, "missing.txt") {
		t.Fatalf("不存在文件不应返回 true")
	}
}

type errorFileStatFS struct {
	fs.FS
}

func (e errorFileStatFS) Open(name string) (fs.File, error) {
	f, err := e.FS.Open(name)
	if err != nil {
		return nil, err
	}
	return errorStatFile{File: f}, nil
}

type errorStatFile struct {
	fs.File
}

func (e errorStatFile) Stat() (fs.FileInfo, error) {
	return nil, fs.ErrInvalid
}
