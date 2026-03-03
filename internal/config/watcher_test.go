package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"
)

// TestNewWatcher_MissingFile 验证配置文件不存在时 NewWatcher 返回错误。
func TestNewWatcher_MissingFile(t *testing.T) {
	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, "missing.json")
	manager := &Manager{path: configPath}
	if _, err := NewWatcher(manager, zap.NewNop(), time.Millisecond); err == nil {
		t.Fatalf("缺失文件应返回错误")
	}
}

// TestWatcherTick_ReloadsWhenModified 验证 tick 检测到变更后触发 Reload 并更新快照。
func TestWatcherTick_ReloadsWhenModified(t *testing.T) {
	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, "config.json")

	v1 := `{
  "dataPlane": { "httpListenAddr": ":8080", "httpsListenAddr": ":8443", "enableHttps": false },
  "security": { "allowedCidrs": [], "blockPageHtml": "<html/>", "enableHsts": false },
  "routing": { "defaultUpstream": "http://127.0.0.1:18080", "domains": [] }
}`
	if err := os.WriteFile(configPath, []byte(v1), 0o600); err != nil {
		t.Fatalf("写入配置失败: %v", err)
	}
	manager, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("初始化 Manager 失败: %v", err)
	}
	watcher, err := NewWatcher(manager, nil, time.Millisecond)
	if err != nil {
		t.Fatalf("初始化 Watcher 失败: %v", err)
	}

	v2 := `{
  "dataPlane": { "httpListenAddr": ":18080", "httpsListenAddr": ":8443", "enableHttps": false },
  "security": { "allowedCidrs": [], "blockPageHtml": "<html/>", "enableHsts": false },
  "routing": { "defaultUpstream": "http://127.0.0.1:18080", "domains": [] }
}`
	if err = os.WriteFile(configPath, []byte(v2), 0o600); err != nil {
		t.Fatalf("写入配置失败: %v", err)
	}
	if err = os.Chtimes(configPath, time.Now().Add(time.Second), time.Now().Add(time.Second)); err != nil {
		t.Fatalf("更新文件时间失败: %v", err)
	}

	watcher.tick()
	if manager.CurrentConfig().DataPlane.HTTPListenAddr != ":18080" {
		t.Fatalf("配置未更新: got=%q", manager.CurrentConfig().DataPlane.HTTPListenAddr)
	}
}

// TestWatcherStart_StopsOnContextDone 验证 Start 会在上下文结束后退出。
func TestWatcherStart_StopsOnContextDone(t *testing.T) {
	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("写入配置失败: %v", err)
	}
	manager, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("初始化 Manager 失败: %v", err)
	}
	watcher, err := NewWatcher(manager, zap.NewNop(), time.Millisecond)
	if err != nil {
		t.Fatalf("初始化 Watcher 失败: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	watcher.Start(ctx)
	cancel()
}

// TestWatcherTick_StatError 验证 tick 在配置文件无法读取时安全返回。
func TestWatcherTick_StatError(t *testing.T) {
	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("写入配置失败: %v", err)
	}
	manager, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("初始化 Manager 失败: %v", err)
	}
	watcher, err := NewWatcher(manager, zap.NewNop(), time.Millisecond)
	if err != nil {
		t.Fatalf("初始化 Watcher 失败: %v", err)
	}

	if err = os.Remove(configPath); err != nil {
		t.Fatalf("删除配置失败: %v", err)
	}
	watcher.tick()
}
