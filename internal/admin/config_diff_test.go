package admin

import (
	"testing"

	"YoBFF/internal/config"
)

func TestBuildConfigDiff_HasChanges(t *testing.T) {
	base := config.Config{
		Security: config.SecurityConfig{
			AllowedCIDRs: []string{"127.0.0.1/32"},
		},
		Routing: config.RoutingConfig{
			DefaultUpstream: "http://127.0.0.1:18080",
		},
	}
	target := config.Config{
		Security: config.SecurityConfig{
			AllowedCIDRs: []string{"10.0.0.0/8"},
		},
		Routing: config.RoutingConfig{
			DefaultUpstream: "http://127.0.0.1:28080",
		},
	}
	changes, err := buildConfigDiff(base, target)
	if err != nil {
		t.Fatalf("构建配置差异失败: %v", err)
	}
	if len(changes) == 0 {
		t.Fatalf("应存在配置差异")
	}
}

func TestCollectDiffs_Branches(t *testing.T) {
	var out []configDiffItem
	collectDiffs("same", "v", "v", &out)
	if len(out) != 0 {
		t.Fatalf("相同值不应生成差异: %#v", out)
	}

	collectDiffs("list", []any{"a"}, []any{"b"}, &out)
	if len(out) != 1 || out[0].Path != "list" {
		t.Fatalf("列表差异不匹配: %#v", out)
	}

	collectDiffs("value", 1, 2, &out)
	if len(out) != 2 || out[1].Path != "value" {
		t.Fatalf("值差异不匹配: %#v", out)
	}
}

func TestCollectMapDiffs_CoversAddedRemovedAndNested(t *testing.T) {
	base := map[string]any{
		"same":  "x",
		"onlyA": "a",
		"nest": map[string]any{
			"k1": "v1",
		},
	}
	target := map[string]any{
		"same":  "x",
		"onlyB": "b",
		"nest": map[string]any{
			"k1": "v2",
			"k2": "v3",
		},
	}
	var out []configDiffItem
	collectMapDiffs("root", base, target, &out)
	if len(out) == 0 {
		t.Fatalf("应存在映射差异")
	}

	foundOnlyA := false
	foundOnlyB := false
	foundNested := false
	for _, item := range out {
		if item.Path == "root.onlyA" {
			foundOnlyA = true
		}
		if item.Path == "root.onlyB" {
			foundOnlyB = true
		}
		if item.Path == "root.nest.k1" || item.Path == "root.nest.k2" {
			foundNested = true
		}
	}
	if !foundOnlyA || !foundOnlyB || !foundNested {
		t.Fatalf("映射差异路径不完整: %#v", out)
	}
}
