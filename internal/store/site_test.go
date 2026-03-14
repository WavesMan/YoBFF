package store

import (
	"database/sql"
	"path/filepath"
	"testing"

	"YoBFF/internal/config"
)

// TestStore_SiteLifecycle 验证站点创建、配置更新与按主机名读取完整链路。
func TestStore_SiteLifecycle(t *testing.T) {
	// 1. 初始化临时数据库
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	s, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("初始化存储失败: %v", err)
	}
	defer func(s *Store) {
		err := s.Close()
		if err != nil {

		}
	}(s)

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
	cfgToSave := config.Config{
		Security: config.SecurityConfig{
			AllowedCIDRs: []string{"127.0.0.1/32"},
		},
	}
	version, err := s.UpdateSiteConfig(created.ID, cfgToSave, "admin", "test")
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
	if len(loadedCfg.Security.AllowedCIDRs) != 1 || loadedCfg.Security.AllowedCIDRs[0] != "127.0.0.1/32" {
		t.Errorf("配置未生效: got %v", loadedCfg)
	}
}

// TestStore_GetSiteConfigByHostname_NotFound 验证未知主机名返回站点不存在错误。
func TestStore_GetSiteConfigByHostname_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	s, _ := NewSQLiteStore(filepath.Join(tmpDir, "test.db"))
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {

		}
	}(s.db)

	_, err := s.GetSiteConfigByHostname("unknown.com")
	if err != ErrSiteNotFound {
		t.Errorf("期望 ErrSiteNotFound，实际返回: %v", err)
	}
}

func TestStore_SiteErrorsAndConfigVersion(t *testing.T) {
	var nilStore *Store
	if _, err := nilStore.CreateSite(Site{Name: "x", Hostname: "a", IP: "1"}); err == nil {
		t.Fatalf("nil store 应返回错误")
	}
	if _, err := nilStore.GetSite("id"); err == nil {
		t.Fatalf("nil store 查询应返回错误")
	}

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test2.db")
	s, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("初始化存储失败: %v", err)
	}
	defer func(s *Store) {
		err := s.Close()
		if err != nil {

		}
	}(s)

	if _, err = s.CreateSite(Site{Name: "", Hostname: "a", IP: "1"}); err == nil {
		t.Fatalf("缺失字段创建应失败")
	}
	if _, err = s.UpdateSite("", Site{Name: "x"}); err == nil {
		t.Fatalf("空 siteID 更新应失败")
	}
	if _, err = s.DeleteSite(""); err == nil {
		t.Fatalf("空 siteID 删除应失败")
	}
	if _, err = s.GetSite("missing"); err != ErrSiteNotFound {
		t.Fatalf("缺失站点错误不匹配: %v", err)
	}
	if _, err = s.GetSiteConfig("missing"); err != ErrSiteNotFound {
		t.Fatalf("缺失站点配置错误不匹配: %v", err)
	}
	if _, err = s.GetSiteConfigByHostname("   "); err != ErrSiteNotFound {
		t.Fatalf("空主机名错误不匹配: %v", err)
	}
	if _, err = s.UpdateSiteConfig("", config.Config{}, "op", "src"); err == nil {
		t.Fatalf("空 siteID 更新配置应失败")
	}
	if _, err = s.UpdateSiteConfig("missing", config.Config{}, "op", "src"); err != ErrSiteNotFound {
		t.Fatalf("缺失站点更新配置错误不匹配: %v", err)
	}
	if _, err = s.ListSiteVersions("", 1); err == nil {
		t.Fatalf("空 siteID 查询版本应失败")
	}
	if _, err = s.GetSiteVersionConfig("missing", "v1"); err != ErrSiteVersionMissing {
		t.Fatalf("缺失版本错误不匹配: %v", err)
	}
	if _, err = s.RollbackSiteVersion("missing", "v1", "op"); err != ErrSiteVersionMissing {
		t.Fatalf("回滚缺失版本错误不匹配: %v", err)
	}
	if _, err = s.GetSiteLogStream("missing"); err != ErrSiteLogNotFound {
		t.Fatalf("缺失日志流错误不匹配: %v", err)
	}
	if _, err = s.UpdateSiteLogStream("", "q"); err == nil {
		t.Fatalf("空 siteID 更新日志流应失败")
	}
	if _, err = s.UpdateSiteLogStream("site", ""); err == nil {
		t.Fatalf("空 filter 更新日志流应失败")
	}
}

func TestStore_ListAndLogStreamFlow(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test3.db")
	s, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("初始化存储失败: %v", err)
	}
	defer func(s *Store) {
		err := s.Close()
		if err != nil {

		}
	}(s)

	siteA, err := s.CreateSite(Site{Name: "A", Hostname: "a.example.com", IP: "10.0.0.1"})
	if err != nil {
		t.Fatalf("创建站点A失败: %v", err)
	}
	siteB, err := s.CreateSite(Site{Name: "B", Hostname: "b.example.com", IP: "10.0.0.2"})
	if err != nil {
		t.Fatalf("创建站点B失败: %v", err)
	}

	if _, err = s.UpdateSite(siteA.ID, Site{Name: "A2"}); err != nil {
		t.Fatalf("更新站点失败: %v", err)
	}
	if _, err = s.ListSites(SiteFilter{Hostname: "a.example.com"}); err != nil {
		t.Fatalf("按主机筛选失败: %v", err)
	}
	if _, err = s.ListSites(SiteFilter{IP: "10.0.0.2"}); err != nil {
		t.Fatalf("按IP筛选失败: %v", err)
	}
	if _, err = s.ListSites(SiteFilter{Hostname: "b.example.com", IP: "10.0.0.2"}); err != nil {
		t.Fatalf("组合筛选失败: %v", err)
	}

	ver1, err := s.UpdateSiteConfig(siteA.ID, config.Config{Routing: config.RoutingConfig{DefaultUpstream: "http://127.0.0.1:18080"}}, "admin", "manual")
	if err != nil {
		t.Fatalf("更新配置失败: %v", err)
	}
	if _, err = s.UpdateSiteConfig(siteA.ID, config.Config{Routing: config.RoutingConfig{DefaultUpstream: "http://127.0.0.1:18081"}}, "admin", "manual"); err != nil {
		t.Fatalf("再次更新配置失败: %v", err)
	}
	if _, err = s.GetSiteConfig(siteA.ID); err != nil {
		t.Fatalf("读取站点配置失败: %v", err)
	}
	versions, err := s.ListSiteVersions(siteA.ID, 0)
	if err != nil || len(versions) == 0 {
		t.Fatalf("读取版本列表失败: versions=%d err=%v", len(versions), err)
	}
	if _, err = s.GetSiteVersionConfig(siteA.ID, ver1.ID); err != nil {
		t.Fatalf("读取版本配置失败: %v", err)
	}
	if _, err = s.RollbackSiteVersion(siteA.ID, ver1.ID, "admin"); err != nil {
		t.Fatalf("回滚失败: %v", err)
	}

	if _, err = s.UpdateSiteLogStream(siteA.ID, "status>=500"); err != nil {
		t.Fatalf("更新日志流失败: %v", err)
	}
	if _, err = s.GetSiteLogStream(siteA.ID); err != nil {
		t.Fatalf("读取日志流失败: %v", err)
	}

	if _, err = s.DeleteSite(siteB.ID); err != nil {
		t.Fatalf("删除站点失败: %v", err)
	}
	if _, err = s.GetSite(siteB.ID); err != ErrSiteNotFound {
		t.Fatalf("删除后查询错误不匹配: %v", err)
	}
}

func TestStore_DeleteSiteAndUpdateConfig_SQLFailures(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test4.db")
	s, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("初始化存储失败: %v", err)
	}
	defer func(s *Store) {
		err := s.Close()
		if err != nil {

		}
	}(s)

	site, err := s.CreateSite(Site{Name: "S1", Hostname: "s1.example.com", IP: "10.0.0.10"})
	if err != nil {
		t.Fatalf("创建站点失败: %v", err)
	}

	if _, err = s.db.Exec(`DROP TABLE site_versions`); err != nil {
		t.Fatalf("删除 site_versions 失败: %v", err)
	}
	if _, err = s.UpdateSiteConfig(site.ID, config.Config{}, "tester", "manual"); err == nil {
		t.Fatalf("缺失版本表时更新配置应失败")
	}

	s2, err := NewSQLiteStore(filepath.Join(tmpDir, "test5.db"))
	if err != nil {
		t.Fatalf("初始化第二个存储失败: %v", err)
	}
	defer func(s2 *Store) {
		err := s2.Close()
		if err != nil {

		}
	}(s2)

	site2, err := s2.CreateSite(Site{Name: "S2", Hostname: "s2.example.com", IP: "10.0.0.11"})
	if err != nil {
		t.Fatalf("创建站点失败: %v", err)
	}
	if _, err = s2.db.Exec(`DROP TABLE site_log_streams`); err != nil {
		t.Fatalf("删除 site_log_streams 失败: %v", err)
	}
	if _, err = s2.DeleteSite(site2.ID); err == nil {
		t.Fatalf("缺失日志流表时删除站点应失败")
	}
}

func TestStore_SiteVersionDeleteAndSiteLogs(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test6.db")
	s, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("初始化存储失败: %v", err)
	}
	defer func(s *Store) {
		err := s.Close()
		if err != nil {

		}
	}(s)

	site, err := s.CreateSite(Site{Name: "logs", Hostname: "logs.example.com", IP: "10.0.0.66"})
	if err != nil {
		t.Fatalf("创建站点失败: %v", err)
	}
	if _, err = s.GetSiteIDByHostname("LOGS.EXAMPLE.COM"); err != nil {
		t.Fatalf("按域名查询站点ID失败: %v", err)
	}

	version, err := s.UpdateSiteConfig(site.ID, config.Config{
		Security: config.SecurityConfig{AllowedCIDRs: []string{"127.0.0.1/32"}},
	}, "tester", "manual")
	if err != nil {
		t.Fatalf("更新配置失败: %v", err)
	}
	if err = s.DeleteSiteVersion(site.ID, version.ID); err != nil {
		t.Fatalf("删除站点版本失败: %v", err)
	}
	if err = s.DeleteSiteVersion(site.ID, version.ID); err != ErrSiteVersionMissing {
		t.Fatalf("重复删除版本错误不匹配: %v", err)
	}

	if err = s.SaveSiteTrafficLog(site.ID, SiteTrafficLogWrite{
		Level:      "error",
		EventType:  "proxy",
		Message:    "upstream timeout",
		RequestID:  "r-1",
		ClientIP:   "127.0.0.1",
		Host:       "logs.example.com",
		Method:     "GET",
		Path:       "/healthz",
		StatusCode: 502,
		LatencyMS:  88,
	}); err != nil {
		t.Fatalf("写入流量日志失败: %v", err)
	}
	if err = s.SaveAudit("site_config_version_delete", site.ID, "tester", map[string]any{"version_id": version.ID}); err != nil {
		t.Fatalf("写入系统日志失败: %v", err)
	}

	logs, err := s.ListSiteLogs(site.ID, SiteLogQuery{
		Kind:  "all",
		Level: "",
		Limit: 20,
	})
	if err != nil {
		t.Fatalf("查询日志失败: %v", err)
	}
	if len(logs) < 2 {
		t.Fatalf("日志数量不足: got=%d", len(logs))
	}
}
