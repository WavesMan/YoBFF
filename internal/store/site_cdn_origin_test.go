package store

import (
	"database/sql"
	"errors"
	"testing"
	"time"
)

// TestStore_SiteCDNOriginSnapshotFlow 验证站点回源快照写入与按时间读取最新记录。
func TestStore_SiteCDNOriginSnapshotFlow(t *testing.T) {
	s := buildTestStore(t)
	site := createSiteForTest(t, s, "S1", "s1.example.com", "10.0.0.10")

	tm1 := time.Now().Add(-2 * time.Minute).UTC()
	_, err := s.SaveSiteCDNOriginSnapshot(site.ID, "cloudflare", []string{"1.1.1.1/32"}, tm1, "auto")
	if err != nil {
		t.Fatalf("写入快照失败: %v", err)
	}

	tm2 := time.Now().Add(-1 * time.Minute).UTC()
	_, err = s.SaveSiteCDNOriginSnapshot(site.ID, "cloudflare", []string{"2.2.2.2/32"}, tm2, "manual")
	if err != nil {
		t.Fatalf("写入快照失败: %v", err)
	}

	got, err := s.GetLatestSiteCDNOriginSnapshot(site.ID, "cloudflare")
	if err != nil {
		t.Fatalf("读取最新快照失败: %v", err)
	}
	if got.Provider != "cloudflare" {
		t.Fatalf("provider 不匹配: %q", got.Provider)
	}
	if len(got.CIDRs) != 1 || got.CIDRs[0] != "2.2.2.2/32" {
		t.Fatalf("CIDRs 不匹配: %+v", got.CIDRs)
	}
	if got.Source != "manual" {
		t.Fatalf("source 不匹配: %q", got.Source)
	}
	if got.FetchedAt == "" || got.CreatedAt == "" || got.ID == "" {
		t.Fatalf("快照字段缺失: %+v", got)
	}
}

// TestStore_SiteCDNOriginStatusUpsertAndList 验证状态 upsert、成功时间保留与排序行为。
func TestStore_SiteCDNOriginStatusUpsertAndList(t *testing.T) {
	s := buildTestStore(t)
	site := createSiteForTest(t, s, "S1", "s1.example.com", "10.0.0.10")

	now := time.Now().UTC()
	status, err := s.UpsertSiteCDNOriginStatus(site.ID, "cloudflare", SiteCDNOriginStatusUpdate{
		LastAttemptAt:       &now,
		ConsecutiveFailures: 1,
		LastError:           "failed",
	})
	if err != nil {
		t.Fatalf("写入状态失败: %v", err)
	}
	if status.Provider != "cloudflare" {
		t.Fatalf("provider 不匹配: %q", status.Provider)
	}
	if status.ConsecutiveFailures != 1 || status.LastError != "failed" {
		t.Fatalf("状态字段不匹配: %+v", status)
	}
	if status.LastAttemptAt == "" || status.UpdatedAt == "" {
		t.Fatalf("时间字段缺失: %+v", status)
	}

	success := now.Add(-10 * time.Second).UTC()
	status, err = s.UpsertSiteCDNOriginStatus(site.ID, "cloudflare", SiteCDNOriginStatusUpdate{
		LastAttemptAt:       &now,
		LastSuccessAt:       &success,
		ConsecutiveFailures: 0,
		LastError:           "",
	})
	if err != nil {
		t.Fatalf("更新状态失败: %v", err)
	}
	if status.LastSuccessAt == "" {
		t.Fatalf("应写入 lastSuccessAt: %+v", status)
	}

	// LastSuccessAt 为空时不应覆盖已存在成功时间
	status, err = s.UpsertSiteCDNOriginStatus(site.ID, "cloudflare", SiteCDNOriginStatusUpdate{
		LastAttemptAt:       &now,
		LastSuccessAt:       nil,
		ConsecutiveFailures: 2,
		LastError:           "oops",
	})
	if err != nil {
		t.Fatalf("更新状态失败: %v", err)
	}
	if status.LastSuccessAt == "" {
		t.Fatalf("不应被空值覆盖: %+v", status)
	}
	if status.ConsecutiveFailures != 2 || status.LastError != "oops" {
		t.Fatalf("更新字段不匹配: %+v", status)
	}

	_, err = s.UpsertSiteCDNOriginStatus(site.ID, "tencent", SiteCDNOriginStatusUpdate{
		ConsecutiveFailures: 3,
		LastError:           "bad",
	})
	if err != nil {
		t.Fatalf("写入第二条状态失败: %v", err)
	}

	list, err := s.ListSiteCDNOriginStatus(site.ID)
	if err != nil {
		t.Fatalf("查询状态列表失败: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("状态数量不匹配: got=%d", len(list))
	}
	if list[0].Provider != "cloudflare" || list[1].Provider != "tencent" {
		t.Fatalf("排序不符合预期: %+v", list)
	}
}

// TestStore_SiteCDNOriginGuards 覆盖空接收者、参数校验与无记录等边界分支。
func TestStore_SiteCDNOriginGuards(t *testing.T) {
	var nilStore *Store
	if _, err := nilStore.SaveSiteCDNOriginSnapshot("s", "p", nil, time.Now(), "auto"); err == nil {
		t.Fatalf("nil store 写入快照应失败")
	}
	if _, err := nilStore.GetLatestSiteCDNOriginSnapshot("s", "p"); err == nil {
		t.Fatalf("nil store 读取快照应失败")
	}
	if _, err := nilStore.UpsertSiteCDNOriginStatus("s", "p", SiteCDNOriginStatusUpdate{}); err == nil {
		t.Fatalf("nil store 写入状态应失败")
	}
	if _, err := nilStore.GetSiteCDNOriginStatus("s", "p"); err == nil {
		t.Fatalf("nil store 读取状态应失败")
	}
	if _, err := nilStore.ListSiteCDNOriginStatus("s"); err == nil {
		t.Fatalf("nil store 列表查询应失败")
	}

	s := buildTestStore(t)
	if _, err := s.GetLatestSiteCDNOriginSnapshot("missing", "cloudflare"); err == nil {
		t.Fatalf("无记录应返回错误")
	} else if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("错误类型不匹配: %v", err)
	}

	if _, err := s.SaveSiteCDNOriginSnapshot("", "cloudflare", nil, time.Now(), "auto"); err == nil {
		t.Fatalf("空 siteID 应失败")
	}
	if _, err := s.SaveSiteCDNOriginSnapshot("site", "", nil, time.Now(), "auto"); err == nil {
		t.Fatalf("空 provider 应失败")
	}
	if _, err := s.UpsertSiteCDNOriginStatus("", "cloudflare", SiteCDNOriginStatusUpdate{}); err == nil {
		t.Fatalf("空 siteID 应失败")
	}
	if _, err := s.UpsertSiteCDNOriginStatus("site", "", SiteCDNOriginStatusUpdate{}); err == nil {
		t.Fatalf("空 provider 应失败")
	}
	if _, err := s.ListSiteCDNOriginStatus(""); err == nil {
		t.Fatalf("空 siteID 列表应失败")
	}
}
