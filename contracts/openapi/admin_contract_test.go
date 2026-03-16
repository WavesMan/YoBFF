package openapi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAdminOpenAPIContractConsistency 验证控制面 OpenAPI 契约包含关键路径与核心结构。
func TestAdminOpenAPIContractConsistency(t *testing.T) {
	t.Parallel()

	content := loadAdminContractContent(t)
	requiredPaths := []string{
		"/api/v1/login:",
		"/api/v1/config:",
		"/api/v1/lb/pools:",
		"/api/v1/sites:",
		"/api/v1/log/stats:",
		"/api/v1/audit/logs:",
		"/api/v1/weaver/drafts:",
		"/api/v1/weaver/drafts/{draft_id}:",
		"/api/v1/weaver/drafts/{draft_id}/run:",
	}
	for _, route := range requiredPaths {
		assertContains(t, content, route)
	}

	requiredSchemas := []string{
		"WeaverDraftRequest:",
		"WeaverDraft:",
		"WeaverDraftListResponse:",
		"WeaverRunRequest:",
		"WeaverRunResponse:",
		"WeaverRunSource:",
	}
	for _, schema := range requiredSchemas {
		assertContains(t, content, schema)
	}

	requiredResponseRefs := []string{
		"WeaverDraftNotFound:",
		"error_code: draft_not_found",
	}
	for _, item := range requiredResponseRefs {
		assertContains(t, content, item)
	}
}

// loadAdminContractContent 读取 admin OpenAPI 契约文件并返回文本内容。
func loadAdminContractContent(t *testing.T) string {
	t.Helper()

	filePath := filepath.Join("admin.yaml")
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("读取契约文件失败: %v", err)
	}
	return string(data)
}

// assertContains 断言给定文本包含目标片段。
func assertContains(t *testing.T, content string, expected string) {
	t.Helper()

	if strings.Contains(content, expected) {
		return
	}
	t.Fatalf("契约缺少必要片段: %s", expected)
}
