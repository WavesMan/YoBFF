package app

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"strings"
)

// BuildRootHandler 组合根路由处理器，将 /admin 前缀请求交给控制面，其余交给数据面。
// 参数：dataHandler 为数据面处理器，adminHandler 为控制面处理器。
// 返回：可直接挂载到 http.Server 的处理器。
// 异常：无。
func BuildRootHandler(dataHandler http.Handler, adminHandler http.Handler, uiHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/admin":
			http.Redirect(w, r, "/admin/", http.StatusMovedPermanently)
			return
		case "/healthz":
			adminHandler.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/admin/") {
			http.StripPrefix("/admin", adminHandler).ServeHTTP(w, r)
			return
		}
		if uiHandler != nil {
			if r.URL.Path == "/" || isUIAssetPath(r.URL.Path) {
				uiHandler.ServeHTTP(w, r)
				return
			}
		}
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok","service":"YoBFF Gateway"}`))
			return
		}
		if r.URL.Path == "/favicon.ico" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		dataHandler.ServeHTTP(w, r)
	})
}

// NewUIHandler 构建管理端静态资源处理器并注入 SPA 入口兜底。
// 参数：fsys 为 UI 静态资源文件系统。
// 返回：可处理管理端前端资源的处理器。
// 异常：资源缺失或初始化失败时返回错误。
func NewUIHandler(fsys fs.FS) (http.Handler, error) {
	if fsys == nil {
		return nil, errors.New("ui fs is nil")
	}
	if _, err := fs.Stat(fsys, "index.html"); err != nil {
		return nil, err
	}
	fileServer := http.FileServer(http.FS(fsys))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" || strings.HasSuffix(r.URL.Path, "/") {
			serveIndex(w, r, fsys)
			return
		}
		if fileExists(fsys, path) {
			fileServer.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(path, "assets/") {
			altPath := strings.TrimPrefix(path, "assets/")
			if fileExists(fsys, altPath) {
				r2 := r.Clone(r.Context())
				r2.URL = &url.URL{Path: "/" + altPath}
				fileServer.ServeHTTP(w, r2)
				return
			}
		}
		serveIndex(w, r, fsys)
	}), nil
}

// isUIAssetPath 判断路径是否属于前端静态资源。
// 参数：path 为请求路径。
// 返回：true 表示命中静态资源路由。
// 异常：无。
func isUIAssetPath(path string) bool {
	if strings.HasPrefix(path, "/assets/") {
		return true
	}
	switch path {
	case "/vite.svg", "/favicon.ico":
		return true
	default:
		return false
	}
}

// fileExists 判断目标文件是否存在且非目录。
// 参数：fsys 为文件系统，name 为文件路径。
// 返回：true 表示文件存在。
// 异常：无。
func fileExists(fsys fs.FS, name string) bool {
	if name == "" {
		return false
	}
	info, err := fs.Stat(fsys, name)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// serveIndex 返回管理端 SPA 入口文件内容。
// 参数：w 为响应写入器，r 为请求对象，fsys 为静态资源文件系统。
// 返回：无。
// 异常：读取或解析失败时返回对应状态码。
func serveIndex(w http.ResponseWriter, r *http.Request, fsys fs.FS) {
	file, err := fsys.Open("index.html")
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer func(file fs.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)
	info, err := file.Stat()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	content, err := io.ReadAll(file)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	http.ServeContent(w, r, "index.html", info.ModTime(), bytes.NewReader(content))
}
