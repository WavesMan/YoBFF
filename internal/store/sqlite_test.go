package store

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"YoBFF/internal/config"
)

func buildTestStore(t *testing.T) *Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "store.db")
	s, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("初始化存储失败: %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})
	return s
}

func createSiteForTest(t *testing.T, s *Store, name string, hostname string, ip string) Site {
	t.Helper()
	site, err := s.CreateSite(Site{
		Name:     name,
		Hostname: hostname,
		IP:       ip,
	})
	if err != nil {
		t.Fatalf("创建站点失败: %v", err)
	}
	return site
}

func TestStore_GlobalConfigVersionAuditAndCertificates(t *testing.T) {
	s := buildTestStore(t)
	cfg1 := config.Config{
		Security: config.SecurityConfig{
			BlockPageHTML: "v1",
		},
	}
	cfg2 := config.Config{
		Security: config.SecurityConfig{
			BlockPageHTML: "v2",
		},
	}
	v1, err := s.SaveVersion(cfg1, "alice", "update")
	if err != nil {
		t.Fatalf("保存版本失败: %v", err)
	}
	time.Sleep(5 * time.Millisecond)
	v2, err := s.SaveVersion(cfg2, "bob", "rollback")
	if err != nil {
		t.Fatalf("保存版本失败: %v", err)
	}

	list, err := s.ListVersions(0)
	if err != nil {
		t.Fatalf("查询版本列表失败: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("版本数量不匹配: got=%d", len(list))
	}
	foundV2 := false
	for _, item := range list {
		if item.ID == v2.ID {
			foundV2 = true
		}
	}
	if !foundV2 {
		t.Fatalf("未找到最新版本")
	}

	gotCfg, err := s.GetVersionConfig(v1.ID)
	if err != nil {
		t.Fatalf("读取版本配置失败: %v", err)
	}
	if gotCfg.Security.BlockPageHTML != "v1" {
		t.Fatalf("配置不匹配: got=%q", gotCfg.Security.BlockPageHTML)
	}

	if err = s.SaveAudit("config_update", v1.ID, "alice", map[string]string{"k": "v"}); err != nil {
		t.Fatalf("写入审计失败: %v", err)
	}
	if err = s.SaveAudit("config_read", v2.ID, "bob", nil); err != nil {
		t.Fatalf("写入审计失败: %v", err)
	}
	var auditCount int
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM audit_logs`).Scan(&auditCount); err != nil {
		t.Fatalf("统计审计失败: %v", err)
	}
	if auditCount != 2 {
		t.Fatalf("审计数量不匹配: got=%d", auditCount)
	}

	cert := &config.SSLCertificate{
		ID:        "cert-1",
		Name:      "test",
		Domains:   []string{"example.com", "*.example.com"},
		NotAfter:  time.Now().UTC().Add(24 * time.Hour),
		Issuer:    "ca",
		CertPEM:   "cert",
		KeyPEM:    "key",
		CreatedAt: time.Now().UTC(),
	}
	if err = s.CreateCertificate(cert); err != nil {
		t.Fatalf("创建证书失败: %v", err)
	}

	gotCert, err := s.GetCertificate("cert-1")
	if err != nil {
		t.Fatalf("查询证书失败: %v", err)
	}
	if gotCert.Name != cert.Name || len(gotCert.Domains) != 2 {
		t.Fatalf("证书内容不匹配: got=%+v", *gotCert)
	}

	certs, total, err := s.ListCertificates(1, 10)
	if err != nil {
		t.Fatalf("证书列表失败: %v", err)
	}
	if total != 1 || len(certs) != 1 {
		t.Fatalf("证书数量不匹配: total=%d len=%d", total, len(certs))
	}

	if err = s.DeleteCertificate("cert-1"); err != nil {
		t.Fatalf("删除证书失败: %v", err)
	}
	if _, err = s.GetCertificate("cert-1"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("删除后查询错误不匹配: %v", err)
	}
}

func TestStore_SiteOperationsCoverAllPaths(t *testing.T) {
	s := buildTestStore(t)
	site1 := createSiteForTest(t, s, "A", "a.example.com", "10.0.0.1")
	site2 := createSiteForTest(t, s, "B", "b.example.com", "10.0.0.2")

	updated, err := s.UpdateSite(site1.ID, Site{
		Name:     "A2",
		Hostname: "A.EXAMPLE.COM",
	})
	if err != nil {
		t.Fatalf("更新站点失败: %v", err)
	}
	if updated.Name != "A2" || updated.Hostname != "A.EXAMPLE.COM" {
		t.Fatalf("更新结果不匹配: %+v", updated)
	}

	gotSite, err := s.GetSite(site1.ID)
	if err != nil {
		t.Fatalf("查询站点失败: %v", err)
	}
	if gotSite.ID != site1.ID {
		t.Fatalf("站点ID不匹配: got=%q", gotSite.ID)
	}

	allSites, err := s.ListSites(SiteFilter{})
	if err != nil {
		t.Fatalf("查询站点列表失败: %v", err)
	}
	if len(allSites) != 2 {
		t.Fatalf("站点数量不匹配: got=%d", len(allSites))
	}

	byHost, err := s.ListSites(SiteFilter{Hostname: "b.example.com"})
	if err != nil {
		t.Fatalf("按主机名筛选失败: %v", err)
	}
	if len(byHost) != 1 || byHost[0].ID != site2.ID {
		t.Fatalf("按主机名筛选结果错误: %+v", byHost)
	}

	byIP, err := s.ListSites(SiteFilter{IP: "10.0.0.2"})
	if err != nil {
		t.Fatalf("按IP筛选失败: %v", err)
	}
	if len(byIP) != 1 || byIP[0].ID != site2.ID {
		t.Fatalf("按IP筛选结果错误: %+v", byIP)
	}

	cfgEmpty, err := s.GetSiteConfig(site1.ID)
	if err != nil {
		t.Fatalf("读取站点空配置失败: %v", err)
	}
	if cfgEmpty.Security.BlockPageHTML != "" {
		t.Fatalf("空配置应返回零值: %+v", cfgEmpty)
	}

	if _, err = s.GetSiteConfigByHostname(""); !errors.Is(err, ErrSiteNotFound) {
		t.Fatalf("空主机名错误不匹配: %v", err)
	}

	cfg := config.Config{
		Security: config.SecurityConfig{
			BlockPageHTML: "<html>blocked</html>",
		},
	}
	for i := 0; i < 12; i++ {
		nextCfg := cfg
		nextCfg.Security.BlockPageHTML = cfg.Security.BlockPageHTML + "-" + time.Now().UTC().Format(time.RFC3339Nano)
		version, saveErr := s.UpdateSiteConfig(site1.ID, nextCfg, "tester", "update")
		if saveErr != nil {
			t.Fatalf("更新站点配置失败: %v", saveErr)
		}
		if version.ID == "" {
			t.Fatalf("版本ID为空")
		}
	}

	currentCfg, err := s.GetSiteConfigByHostname("a.example.com")
	if err != nil {
		t.Fatalf("按主机名读取配置失败: %v", err)
	}
	if currentCfg.Security.BlockPageHTML == "" {
		t.Fatalf("站点配置未生效")
	}

	versions, err := s.ListSiteVersions(site1.ID, 0)
	if err != nil {
		t.Fatalf("查询站点版本失败: %v", err)
	}
	if len(versions) != 10 {
		t.Fatalf("版本保留数量不匹配: got=%d", len(versions))
	}
	rollbackVersionID := versions[len(versions)-1].ID

	if _, err = s.GetSiteVersionConfig(site1.ID, "missing-version"); !errors.Is(err, ErrSiteVersionMissing) {
		t.Fatalf("缺失版本错误不匹配: %v", err)
	}

	rollback, err := s.RollbackSiteVersion(site1.ID, rollbackVersionID, "tester")
	if err != nil {
		t.Fatalf("回滚站点配置失败: %v", err)
	}
	if rollback.ID == "" {
		t.Fatalf("回滚版本ID为空")
	}

	if _, err = s.GetSiteLogStream(site1.ID); !errors.Is(err, ErrSiteLogNotFound) {
		t.Fatalf("未配置日志流错误不匹配: %v", err)
	}
	stream, err := s.UpdateSiteLogStream(site1.ID, "status>=500")
	if err != nil {
		t.Fatalf("更新日志流失败: %v", err)
	}
	if stream.FilterQuery != "status>=500" {
		t.Fatalf("日志流过滤条件不匹配: %+v", stream)
	}
	gotStream, err := s.GetSiteLogStream(site1.ID)
	if err != nil {
		t.Fatalf("查询日志流失败: %v", err)
	}
	if gotStream.SiteID != site1.ID {
		t.Fatalf("日志流站点ID不匹配: %+v", gotStream)
	}

	deleted, err := s.DeleteSite(site2.ID)
	if err != nil {
		t.Fatalf("删除站点失败: %v", err)
	}
	if deleted.ID != site2.ID {
		t.Fatalf("删除返回ID不匹配: %+v", deleted)
	}
	if _, err = s.GetSite(site2.ID); !errors.Is(err, ErrSiteNotFound) {
		t.Fatalf("删除后查询错误不匹配: %v", err)
	}
}

func TestStore_GuardErrors(t *testing.T) {
	var nilStore *Store

	if _, err := nilStore.SaveVersion(config.Config{}, "u", "s"); err == nil {
		t.Fatalf("SaveVersion 应返回错误")
	}
	if _, err := nilStore.ListVersions(1); err == nil {
		t.Fatalf("ListVersions 应返回错误")
	}
	if _, err := nilStore.GetVersionConfig("id"); err == nil {
		t.Fatalf("GetVersionConfig 应返回错误")
	}
	if err := nilStore.SaveAudit("a", "b", "c", nil); err == nil {
		t.Fatalf("SaveAudit 应返回错误")
	}
	if _, err := nilStore.CreateSite(Site{}); err == nil {
		t.Fatalf("CreateSite 应返回错误")
	}
	if _, err := nilStore.UpdateSite("id", Site{}); err == nil {
		t.Fatalf("UpdateSite 应返回错误")
	}
	if _, err := nilStore.DeleteSite("id"); err == nil {
		t.Fatalf("DeleteSite 应返回错误")
	}
	if _, err := nilStore.GetSite("id"); err == nil {
		t.Fatalf("GetSite 应返回错误")
	}
	if _, err := nilStore.GetSiteConfigByHostname("a.com"); err == nil {
		t.Fatalf("GetSiteConfigByHostname 应返回错误")
	}
	if _, err := nilStore.ListSites(SiteFilter{}); err == nil {
		t.Fatalf("ListSites 应返回错误")
	}
	if _, err := nilStore.GetSiteConfig("id"); err == nil {
		t.Fatalf("GetSiteConfig 应返回错误")
	}
	if _, err := nilStore.UpdateSiteConfig("id", config.Config{}, "op", "src"); err == nil {
		t.Fatalf("UpdateSiteConfig 应返回错误")
	}
	if _, err := nilStore.ListSiteVersions("id", 1); err == nil {
		t.Fatalf("ListSiteVersions 应返回错误")
	}
	if _, err := nilStore.GetSiteVersionConfig("id", "v1"); err == nil {
		t.Fatalf("GetSiteVersionConfig 应返回错误")
	}
	if _, err := nilStore.GetSiteLogStream("id"); err == nil {
		t.Fatalf("GetSiteLogStream 应返回错误")
	}
	if _, err := nilStore.UpdateSiteLogStream("id", "q"); err == nil {
		t.Fatalf("UpdateSiteLogStream 应返回错误")
	}
}
