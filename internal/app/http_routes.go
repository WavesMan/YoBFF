package app

import (
	"net/http"
	"strings"
)

// BuildRootHandler 组合根路由处理器，将 /admin 前缀请求交给控制面，其余交给数据面。
// 参数：dataHandler 为数据面处理器，adminHandler 为控制面处理器。
// 返回：可直接挂载到 http.Server 的处理器。
// 异常：无。
func BuildRootHandler(dataHandler http.Handler, adminHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/admin":
			http.Redirect(w, r, "/admin/", http.StatusMovedPermanently)
			return
		case "/":
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok","service":"YoBFF Gateway"}`))
			return
		case "/favicon.ico":
			w.WriteHeader(http.StatusNoContent)
			return
		case "/healthz":
			adminHandler.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/admin/") {
			http.StripPrefix("/admin", adminHandler).ServeHTTP(w, r)
			return
		}
		dataHandler.ServeHTTP(w, r)
	})
}
