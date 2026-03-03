package config

import (
	"errors"
	"net/netip"
	"os"
	"testing"
)

func TestManager_UpdateProviderStatus(t *testing.T) {
	// 1. 创建临时配置文件
	f, err := os.CreateTemp("", "config_*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	content := `{
		"dataPlane": {"httpListenAddr": ":8080"},
		"controlPlane": {"adminListenAddr": ":9090"},
		"security": {"allowedCidrs": ["127.0.0.1/32"]},
		"routing": {"defaultUpstream": "http://localhost:8081"},
		"certificates": []
	}`
	if _, err = f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()

	// 2. 初始化 Manager
	manager, err := NewManager(f.Name())
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// 3. 测试更新状态
	provider := "test_provider"
	cidrs := []string{"192.0.2.0/24", "10.0.0.0/8"}

	// Case 1: 成功更新
	if err = manager.UpdateProviderStatus(provider, cidrs, nil); err != nil {
		t.Fatalf("UpdateProviderStatus failed: %v", err)
	}

	// 验证状态
	status := manager.GetCDNStatus()
	if s, ok := status[provider]; !ok {
		t.Errorf("Provider status not found")
	} else {
		if len(s.CIDRs) != 2 {
			t.Errorf("Expected 2 CIDRs, got %d", len(s.CIDRs))
		}
		if s.Error != "" {
			t.Errorf("Unexpected error: %s", s.Error)
		}
	}

	// 验证 IP 是否生效
	ip := netip.MustParseAddr("192.0.2.1")
	if !manager.IsIPAllowed(ip) {
		t.Error("IP 192.0.2.1 should be allowed")
	}

	// Case 2: 更新带错误
	syncErr := errors.New("sync failed")
	if err = manager.UpdateProviderStatus(provider, nil, syncErr); err != nil {
		t.Fatalf("UpdateProviderStatus with error failed: %v", err)
	}

	status = manager.GetCDNStatus()
	if s, ok := status[provider]; !ok {
		t.Errorf("Provider status not found")
	} else {
		if s.Error != "sync failed" {
			t.Errorf("Expected error 'sync failed', got '%s'", s.Error)
		}
		// 之前的 CIDR 应该被清空 (因为传入了 nil)
		if len(s.CIDRs) != 0 {
			t.Errorf("Expected 0 CIDRs, got %d", len(s.CIDRs))
		}
	}

	// 验证 IP 是否不再生效
	if manager.IsIPAllowed(ip) {
		t.Error("IP 192.0.2.1 should not be allowed after update")
	}
}
