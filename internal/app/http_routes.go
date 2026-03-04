package app

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"net/http"
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
		serveIndex(w, r, fsys)
	}), nil
}

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

func serveIndex(w http.ResponseWriter, r *http.Request, fsys fs.FS) {
	file, err := fsys.Open("index.html")
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer file.Close()
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
