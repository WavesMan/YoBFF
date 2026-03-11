package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"YoBFF/internal/logging"

	"go.uber.org/zap"
)

// TestServerLBPoolsAndRoutes 验证流量池与路由绑定接口的基本读写流程。
func TestServerLBPoolsAndRoutes(t *testing.T) {
	t.Setenv("ADMIN_API_TOKEN", "token")
	t.Setenv("ADMIN_RATE_LIMIT_PER_MIN", "60")

	manager := buildManagerForTest(t, `{
  "controlPlane": { "auth": { "token": "token" } },
  "security": { "allowedCidrs": [], "blockPageHtml": "<html/>", "enableHsts": false },
  "routing": { "domains": [] },
  "loadBalancer": { "pools": [], "routes": [] }
}`)
	logRuntime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化日志运行时失败: %v", err)
	}
	pipeline := logging.NewPipeline(16, zap.NewNop())
	defer pipeline.Close()
	server := NewServer(manager, logRuntime, pipeline, nil)
	handler := server.Handler()

	poolBody := []byte(`{
  "id": "pool_order_prod",
  "name": "订单主池",
  "strategy": "weighted_rr",
  "nodes": [
    {"id":"n1","upstream":"http://127.0.0.1:18080","weight":100,"enabled":true}
  ]
}`)
	poolReq := httptest.NewRequest(http.MethodPost, "/api/v1/lb/pools", bytes.NewReader(poolBody))
	poolReq.Header.Set("Authorization", "Bearer token")
	poolRec := httptest.NewRecorder()
	handler.ServeHTTP(poolRec, poolReq)
	if poolRec.Code != http.StatusOK {
		t.Fatalf("创建流量池失败: status=%d body=%s", poolRec.Code, poolRec.Body.String())
	}

	routeBody := []byte(`{"poolId":"pool_order_prod","forceHttps":true}`)
	routeReq := httptest.NewRequest(http.MethodPut, "/api/v1/lb/routes/api.example.com", bytes.NewReader(routeBody))
	routeReq.Header.Set("Authorization", "Bearer token")
	routeRec := httptest.NewRecorder()
	handler.ServeHTTP(routeRec, routeReq)
	if routeRec.Code != http.StatusOK {
		t.Fatalf("更新路由绑定失败: status=%d body=%s", routeRec.Code, routeRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/lb/routes/api.example.com", nil)
	getReq.Header.Set("Authorization", "Bearer token")
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("获取路由绑定失败: status=%d body=%s", getRec.Code, getRec.Body.String())
	}
	var payload struct {
		Domain     string `json:"domain"`
		PoolID     string `json:"poolId"`
		ForceHTTPS bool   `json:"forceHttps"`
	}
	if err = json.NewDecoder(getRec.Body).Decode(&payload); err != nil {
		t.Fatalf("解析路由响应失败: %v", err)
	}
	if payload.Domain != "api.example.com" || payload.PoolID != "pool_order_prod" || !payload.ForceHTTPS {
		t.Fatalf("路由绑定不匹配: %#v", payload)
	}
}

// TestServerLBPoolDeleteCleansRoutes 验证删除流量池时会清理引用该池的路由规则。
func TestServerLBPoolDeleteCleansRoutes(t *testing.T) {
	t.Setenv("ADMIN_API_TOKEN", "token")
	t.Setenv("ADMIN_RATE_LIMIT_PER_MIN", "60")

	manager := buildManagerForTest(t, `{
  "controlPlane": { "auth": { "token": "token" } },
  "security": { "allowedCidrs": [], "blockPageHtml": "<html/>", "enableHsts": false },
  "routing": { "domains": [] },
  "loadBalancer": {
    "pools": [
      {
        "id":"pool_main",
        "name":"主池",
        "strategy":"weighted_rr",
        "nodes":[{"id":"n1","upstream":"http://127.0.0.1:18080","weight":1,"enabled":true}]
      }
    ],
    "routes": [
      {"domain":"a.example.com","poolId":"pool_main","forceHttps":false}
    ]
  }
}`)
	logRuntime, err := logging.NewRuntimeFromEnv()
	if err != nil {
		t.Fatalf("初始化日志运行时失败: %v", err)
	}
	pipeline := logging.NewPipeline(16, zap.NewNop())
	defer pipeline.Close()
	server := NewServer(manager, logRuntime, pipeline, nil)
	handler := server.Handler()

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/lb/pools/pool_main", nil)
	deleteReq.Header.Set("Authorization", "Bearer token")
	deleteRec := httptest.NewRecorder()
	handler.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("删除流量池失败: status=%d body=%s", deleteRec.Code, deleteRec.Body.String())
	}

	getRouteReq := httptest.NewRequest(http.MethodGet, "/api/v1/lb/routes/a.example.com", nil)
	getRouteReq.Header.Set("Authorization", "Bearer token")
	getRouteRec := httptest.NewRecorder()
	handler.ServeHTTP(getRouteRec, getRouteReq)
	if getRouteRec.Code != http.StatusNotFound {
		t.Fatalf("路由应被清理: status=%d body=%s", getRouteRec.Code, getRouteRec.Body.String())
	}
}
