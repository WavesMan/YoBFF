package store

import (
	"path/filepath"
	"testing"

	"YoBFF/internal/config"
)

func TestStore_SiteLifecycle(t *testing.T) {
	// 1. 初始化临时数据库
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	s, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("初始化存储失败: %v", err)
	}
	defer s.Close()

	// 2. 创建站点
	site := Site{
		Name:     "Test Site",
		Hostname: "test.example.com",
		IP:       "127.0.0.1",
	}
	created, err := s.CreateSite(site)
	if err != nil {
		t.Fatalf("创建站点失败: %v", err)
	}
	if created.ID == "" {
		t.Error("创建后 ID 为空")
	}

	// 3. 验证主机名查询（默认配置为空）
	_, err = s.GetSiteConfigByHostname("test.example.com")
	if err != nil {
		t.Fatalf("按主机名查询失败: %v", err)
	}
	// 空配置不应报错，返回零值

	// 4. 更新配置
	newCfg := config.Config{
		Security: config.SecurityConfig{
			BlockPageHTML: "<html>blocked</html>",
		},
	}
	version, err := s.UpdateSiteConfig(created.ID, newCfg, "admin", "test")
	if err != nil {
		t.Fatalf("更新配置失败: %v", err)
	}
	if version.ID == "" {
		t.Error("版本 ID 为空")
	}

	// 5. 验证配置生效
	loadedCfg, err := s.GetSiteConfigByHostname("TEST.EXAMPLE.COM") // 测试大小写不敏感
	if err != nil {
		t.Fatalf("再次查询失败: %v", err)
	}
	if loadedCfg.Security.BlockPageHTML != "<html>blocked</html>" {
		t.Errorf("配置未生效: got %v", loadedCfg)
	}
}

func TestStore_GetSiteConfigByHostname_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	s, _ := NewSQLiteStore(filepath.Join(tmpDir, "test.db"))
	defer s.db.Close()

	_, err := s.GetSiteConfigByHostname("unknown.com")
	if err != ErrSiteNotFound {
		t.Errorf("期望 ErrSiteNotFound，实际返回: %v", err)
	}
}
