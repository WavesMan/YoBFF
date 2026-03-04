package app

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildRootHandler_AdminRedirect(t *testing.T) {
	handler := BuildRootHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatalf("不应走数据面: %s", r.URL.Path)
		}),
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatalf("不应走控制面: %s", r.URL.Path)
		}),
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/admin", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusMovedPermanently {
		t.Fatalf("状态码不匹配: got=%d", res.StatusCode)
	}
	if location := res.Header.Get("Location"); location != "/admin/" {
		t.Fatalf("Location 不匹配: got=%q", location)
	}
}

func TestBuildRootHandler_RouteToAdminWithStripPrefix(t *testing.T) {
	var gotPath string
	adminHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})
	dataHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("不应走数据面: %s", r.URL.Path)
	})
	handler := BuildRootHandler(dataHandler, adminHandler, nil)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/admin/api/v1/config", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusOK {
		t.Fatalf("状态码不匹配: got=%d", rec.Result().StatusCode)
	}
	if gotPath != "/api/v1/config" {
		t.Fatalf("路径去前缀失败: got=%q", gotPath)
	}
}

func TestBuildRootHandler_HealthzGoesToAdminWithoutStripPrefix(t *testing.T) {
	var gotPath string
	adminHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})
	handler := BuildRootHandler(http.NotFoundHandler(), adminHandler, nil)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusOK {
		t.Fatalf("状态码不匹配: got=%d", rec.Result().StatusCode)
	}
	if gotPath != "/healthz" {
		t.Fatalf("healthz 路径不应去前缀: got=%q", gotPath)
	}
}

func TestBuildRootHandler_RootReturnsOKJSON(t *testing.T) {
	handler := BuildRootHandler(http.NotFoundHandler(), http.NotFoundHandler(), nil)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	res := rec.Result()
	body, _ := io.ReadAll(res.Body)

	if res.StatusCode != http.StatusOK {
		t.Fatalf("状态码不匹配: got=%d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type 不匹配: got=%q", ct)
	}
	if len(body) == 0 {
		t.Fatalf("响应体为空")
	}
}

func TestBuildRootHandler_FaviconReturnsNoContent(t *testing.T) {
	handler := BuildRootHandler(http.NotFoundHandler(), http.NotFoundHandler(), nil)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/favicon.ico", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusNoContent {
		t.Fatalf("状态码不匹配: got=%d", rec.Result().StatusCode)
	}
}

func TestBuildRootHandler_DefaultGoesToDataPlane(t *testing.T) {
	var dataHit bool
	dataHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dataHit = true
		w.WriteHeader(http.StatusOK)
	})
	uiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("不应走 UI: %s", r.URL.Path)
	})
	adminHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("不应走控制面: %s", r.URL.Path)
	})
	handler := BuildRootHandler(dataHandler, adminHandler, uiHandler)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/some-path", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusOK {
		t.Fatalf("状态码不匹配: got=%d", rec.Result().StatusCode)
	}
	if !dataHit {
		t.Fatalf("数据面未命中")
	}
}

func TestBuildRootHandler_RootGoesToUI(t *testing.T) {
	var uiHit bool
	uiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uiHit = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ui"))
	})
	handler := BuildRootHandler(http.NotFoundHandler(), http.NotFoundHandler(), uiHandler)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusOK {
		t.Fatalf("状态码不匹配: got=%d", rec.Result().StatusCode)
	}
	if !uiHit {
		t.Fatalf("UI 未命中")
	}
}
