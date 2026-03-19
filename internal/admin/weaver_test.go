package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestWeaverDraftRoutes_CRUDAndRun 验证 Weaver 草稿的创建、查询、更新、运行与列表链路。
func TestWeaverDraftRoutes_CRUDAndRun(t *testing.T) {
	handler, _ := buildSiteHandler(t)
	createBody := []byte(`{
		"name":"order-flow",
		"inputs":[{"name":"order","payload":{"id":"o-1001","amount":99}}],
		"mapping":{"result":"$.order.id"}
	}`)

	createRec := requestWeaverWithToken(t, handler, http.MethodPost, "/api/v1/weaver/drafts", createBody)
	if createRec.Code != http.StatusOK {
		t.Fatalf("创建草稿失败: status=%d body=%s", createRec.Code, createRec.Body.String())
	}
	created := decodeWeaverPayload(t, createRec)
	draftID, _ := created["id"].(string)
	if draftID == "" {
		t.Fatalf("草稿ID为空: %#v", created)
	}
	if operator, _ := created["operator"].(string); operator != "admin" {
		t.Fatalf("操作人不匹配: got=%q", operator)
	}

	detailRec := requestWeaverWithToken(t, handler, http.MethodGet, "/api/v1/weaver/drafts/"+draftID, nil)
	if detailRec.Code != http.StatusOK {
		t.Fatalf("读取草稿失败: status=%d body=%s", detailRec.Code, detailRec.Body.String())
	}

	updateBody := []byte(`{
		"name":"order-flow-v2",
		"inputs":[{"name":"order","payload":{"id":"o-1002","amount":188}}],
		"mapping":{"result":"$.order.amount"}
	}`)
	updateRec := requestWeaverWithToken(t, handler, http.MethodPut, "/api/v1/weaver/drafts/"+draftID, updateBody)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("更新草稿失败: status=%d body=%s", updateRec.Code, updateRec.Body.String())
	}

	runBody := []byte(`{
		"inputs":[{"name":"order","payload":{"id":"o-1003","amount":288}}],
		"mapping":{"result":"$.order.amount"}
	}`)
	runRec := requestWeaverWithToken(t, handler, http.MethodPost, "/api/v1/weaver/drafts/"+draftID+"/run", runBody)
	if runRec.Code != http.StatusOK {
		t.Fatalf("运行草稿失败: status=%d body=%s", runRec.Code, runRec.Body.String())
	}
	runPayload := decodeWeaverPayload(t, runRec)
	if runID, _ := runPayload["run_id"].(string); runID == "" {
		t.Fatalf("运行结果缺少 run_id: %#v", runPayload)
	}
	if status, _ := runPayload["status"].(string); status != "succeeded" {
		t.Fatalf("运行结果状态异常: %#v", runPayload)
	}
	if _, ok := runPayload["duration_ms"].(float64); !ok {
		t.Fatalf("运行结果缺少 duration_ms: %#v", runPayload)
	}
	sources, ok := runPayload["sources"].([]any)
	if !ok || len(sources) == 0 {
		t.Fatalf("运行结果 sources 为空: %#v", runPayload)
	}
	attempts, ok := runPayload["attempts"].([]any)
	if !ok || len(attempts) != 1 {
		t.Fatalf("运行结果 attempts 异常: %#v", runPayload)
	}
	retry, ok := runPayload["retry"].(map[string]any)
	if !ok {
		t.Fatalf("运行结果 retry 缺失: %#v", runPayload)
	}
	if used, _ := retry["used"].(float64); used != 1 {
		t.Fatalf("运行结果 retry.used 异常: %#v", runPayload)
	}

	listRec := requestWeaverWithToken(t, handler, http.MethodGet, "/api/v1/weaver/drafts?limit=10", nil)
	if listRec.Code != http.StatusOK {
		t.Fatalf("读取草稿列表失败: status=%d body=%s", listRec.Code, listRec.Body.String())
	}
	listPayload := decodeWeaverPayload(t, listRec)
	items, ok := listPayload["items"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("草稿列表为空: %#v", listPayload)
	}

	deleteRec := requestWeaverWithToken(t, handler, http.MethodDelete, "/api/v1/weaver/drafts/"+draftID, nil)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("删除草稿失败: status=%d body=%s", deleteRec.Code, deleteRec.Body.String())
	}
	deletePayload := decodeWeaverPayload(t, deleteRec)
	if status, _ := deletePayload["status"].(string); status != "deleted" {
		t.Fatalf("删除结果状态异常: %#v", deletePayload)
	}

	detailAfterDelete := requestWeaverWithToken(t, handler, http.MethodGet, "/api/v1/weaver/drafts/"+draftID, nil)
	if detailAfterDelete.Code != http.StatusNotFound {
		t.Fatalf("删除后读取应为404: status=%d body=%s", detailAfterDelete.Code, detailAfterDelete.Body.String())
	}
}

// TestWeaverDraftRoutes_PublishAndRunVersion 验证草稿发布、版本查询与版本运行链路。
func TestWeaverDraftRoutes_PublishAndRunVersion(t *testing.T) {
	handler, _ := buildSiteHandler(t)
	createBody := []byte(`{
		"name":"order-dag",
		"inputs":[{"name":"order","payload":{"id":"o-2001","amount":188}}],
		"mapping":{"result":"$.order.id"},
		"dag":{
			"nodes":[
				{"id":"fetch-order","type":"source","inputs":["order"],"outputs":["order"]},
				{"id":"render","type":"transform","inputs":["order"],"outputs":["result"]}
			],
			"edges":[{"from":"fetch-order","to":"render"}],
			"output_node_id":"render"
		}
	}`)
	createRec := requestWeaverWithToken(t, handler, http.MethodPost, "/api/v1/weaver/drafts", createBody)
	if createRec.Code != http.StatusOK {
		t.Fatalf("创建草稿失败: status=%d body=%s", createRec.Code, createRec.Body.String())
	}
	createPayload := decodeWeaverPayload(t, createRec)
	draftID, _ := createPayload["id"].(string)
	if draftID == "" {
		t.Fatalf("草稿ID为空: %#v", createPayload)
	}

	publishRec := requestWeaverWithToken(t, handler, http.MethodPost, "/api/v1/weaver/drafts/"+draftID+"/publish", nil)
	if publishRec.Code != http.StatusOK {
		t.Fatalf("发布草稿失败: status=%d body=%s", publishRec.Code, publishRec.Body.String())
	}
	publishPayload := decodeWeaverPayload(t, publishRec)
	versionID, _ := publishPayload["id"].(string)
	if versionID == "" {
		t.Fatalf("版本ID为空: %#v", publishPayload)
	}
	if versionNo, ok := publishPayload["version"].(float64); !ok || versionNo < 1 {
		t.Fatalf("版本号异常: %#v", publishPayload)
	}
	nodeContracts, ok := publishPayload["node_contracts"].([]any)
	if !ok || len(nodeContracts) != 2 {
		t.Fatalf("节点契约冻结结果异常: %#v", publishPayload)
	}

	listVersionRec := requestWeaverWithToken(t, handler, http.MethodGet, "/api/v1/weaver/drafts/"+draftID+"/versions?limit=10", nil)
	if listVersionRec.Code != http.StatusOK {
		t.Fatalf("读取版本列表失败: status=%d body=%s", listVersionRec.Code, listVersionRec.Body.String())
	}
	listVersionPayload := decodeWeaverPayload(t, listVersionRec)
	items, ok := listVersionPayload["items"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("版本列表为空: %#v", listVersionPayload)
	}

	versionDetailRec := requestWeaverWithToken(t, handler, http.MethodGet, "/api/v1/weaver/versions/"+versionID, nil)
	if versionDetailRec.Code != http.StatusOK {
		t.Fatalf("读取版本详情失败: status=%d body=%s", versionDetailRec.Code, versionDetailRec.Body.String())
	}

	runVersionRec := requestWeaverWithToken(t, handler, http.MethodPost, "/api/v1/weaver/versions/"+versionID+"/run", []byte(`{}`))
	if runVersionRec.Code != http.StatusOK {
		t.Fatalf("运行版本失败: status=%d body=%s", runVersionRec.Code, runVersionRec.Body.String())
	}
	runVersionPayload := decodeWeaverPayload(t, runVersionRec)
	if status, _ := runVersionPayload["status"].(string); status != "succeeded" {
		t.Fatalf("运行版本状态异常: %#v", runVersionPayload)
	}
	output, ok := runVersionPayload["output"].(map[string]any)
	if !ok {
		t.Fatalf("运行版本输出异常: %#v", runVersionPayload)
	}
	if _, ok := output["dag_output"]; !ok {
		t.Fatalf("运行版本缺少 dag_output: %#v", runVersionPayload)
	}
}

func TestWeaverDraftRoutes_RunRetryAndFailureLocate(t *testing.T) {
	handler, _ := buildSiteHandler(t)
	createBody := []byte(`{
		"name":"retry-observe",
		"inputs":[{"name":"order","payload":{"id":"o-5001","amount":188}}],
		"mapping":{"result":"$.order.id"},
		"dag":{
			"nodes":[
				{"id":"source-1","type":"source","inputs":["order"],"outputs":["payload"]},
				{"id":"transform-1","type":"transform","inputs":["payload"],"outputs":["result"],"config":{"fail_until_attempt":1}},
				{"id":"output-1","type":"output","inputs":["result"],"outputs":["final"]}
			],
			"edges":[
				{"from":"source-1","to":"transform-1"},
				{"from":"transform-1","to":"output-1"}
			],
			"output_node_id":"output-1"
		}
	}`)
	createRec := requestWeaverWithToken(t, handler, http.MethodPost, "/api/v1/weaver/drafts", createBody)
	if createRec.Code != http.StatusOK {
		t.Fatalf("创建草稿失败: status=%d body=%s", createRec.Code, createRec.Body.String())
	}
	created := decodeWeaverPayload(t, createRec)
	draftID, _ := created["id"].(string)
	if draftID == "" {
		t.Fatalf("草稿ID为空: %#v", created)
	}

	runBody := []byte(`{
		"retry":{
			"max_attempts":2,
			"retry_on_node_error":true,
			"retryable_codes":["node_attempt_guard"],
			"backoff_initial_ms":0,
			"backoff_multiplier":2,
			"backoff_max_ms":500
		}
	}`)
	runRec := requestWeaverWithToken(t, handler, http.MethodPost, "/api/v1/weaver/drafts/"+draftID+"/run", runBody)
	if runRec.Code != http.StatusOK {
		t.Fatalf("运行草稿失败: status=%d body=%s", runRec.Code, runRec.Body.String())
	}
	runPayload := decodeWeaverPayload(t, runRec)
	if status, _ := runPayload["status"].(string); status != "succeeded" {
		t.Fatalf("运行状态异常: %#v", runPayload)
	}
	attempts, ok := runPayload["attempts"].([]any)
	if !ok || len(attempts) != 2 {
		t.Fatalf("attempts 数量异常: %#v", runPayload)
	}
	failures, ok := runPayload["failures"].([]any)
	if !ok || len(failures) != 1 {
		t.Fatalf("failures 数量异常: %#v", runPayload)
	}
	firstFailure, ok := failures[0].(map[string]any)
	if !ok {
		t.Fatalf("failure 结构异常: %#v", runPayload)
	}
	if scope, _ := firstFailure["scope"].(string); scope != "node" {
		t.Fatalf("failure scope 异常: %#v", runPayload)
	}
	if nodeID, _ := firstFailure["node_id"].(string); nodeID != "transform-1" {
		t.Fatalf("failure node_id 异常: %#v", runPayload)
	}
	if code, _ := firstFailure["code"].(string); code != "node_attempt_guard" {
		t.Fatalf("failure code 异常: %#v", runPayload)
	}
	retrySnapshot, ok := runPayload["retry"].(map[string]any)
	if !ok {
		t.Fatalf("retry 结构异常: %#v", runPayload)
	}
	if used, _ := retrySnapshot["used"].(float64); used != 2 {
		t.Fatalf("retry.used 异常: %#v", runPayload)
	}
	if triggered, _ := retrySnapshot["triggered"].(bool); !triggered {
		t.Fatalf("retry.triggered 异常: %#v", runPayload)
	}
	if initial, _ := retrySnapshot["backoff_initial_ms"].(float64); initial != 0 {
		t.Fatalf("retry.backoff_initial_ms 异常: %#v", runPayload)
	}
	codes, ok := retrySnapshot["retryable_codes"].([]any)
	if !ok || len(codes) != 1 {
		t.Fatalf("retry.retryable_codes 异常: %#v", runPayload)
	}
	if code, _ := codes[0].(string); code != "node_attempt_guard" {
		t.Fatalf("retry.retryable_codes 值异常: %#v", runPayload)
	}
	delays, ok := retrySnapshot["delays_ms"].([]any)
	if !ok || len(delays) != 1 {
		t.Fatalf("retry.delays_ms 异常: %#v", runPayload)
	}
	if delay, _ := delays[0].(float64); delay != 0 {
		t.Fatalf("retry.delays_ms 值异常: %#v", runPayload)
	}
}

// TestWeaverRunStatsRoutes_GroupAndTrend 验证最近运行统计接口的错误码分组与趋势输出。
func TestWeaverRunStatsRoutes_GroupAndTrend(t *testing.T) {
	handler, _ := buildSiteHandler(t)
	createBody := []byte(`{
		"name":"stats-observe",
		"inputs":[{"name":"order","payload":{"id":"o-7001","amount":88}}],
		"mapping":{"result":"$.order.id"},
		"dag":{
			"nodes":[{"id":"output","type":"transform","inputs":["result"],"outputs":["result"]}],
			"edges":[],
			"output_node_id":"output"
		}
	}`)
	createRec := requestWeaverWithToken(t, handler, http.MethodPost, "/api/v1/weaver/drafts", createBody)
	if createRec.Code != http.StatusOK {
		t.Fatalf("创建草稿失败: status=%d body=%s", createRec.Code, createRec.Body.String())
	}
	created := decodeWeaverPayload(t, createRec)
	draftID, _ := created["id"].(string)
	if draftID == "" {
		t.Fatalf("草稿ID为空: %#v", created)
	}

	successRun := requestWeaverWithToken(t, handler, http.MethodPost, "/api/v1/weaver/drafts/"+draftID+"/run", []byte(`{}`))
	if successRun.Code != http.StatusOK {
		t.Fatalf("运行成功样本失败: status=%d body=%s", successRun.Code, successRun.Body.String())
	}
	failedRun := requestWeaverWithToken(t, handler, http.MethodPost, "/api/v1/weaver/drafts/"+draftID+"/run", []byte(`{
		"inputs":[{"name":"order"}]
	}`))
	if failedRun.Code != http.StatusOK {
		t.Fatalf("运行失败样本失败: status=%d body=%s", failedRun.Code, failedRun.Body.String())
	}
	publishRec := requestWeaverWithToken(t, handler, http.MethodPost, "/api/v1/weaver/drafts/"+draftID+"/publish", nil)
	if publishRec.Code != http.StatusOK {
		t.Fatalf("发布草稿失败: status=%d body=%s", publishRec.Code, publishRec.Body.String())
	}
	publishPayload := decodeWeaverPayload(t, publishRec)
	versionID, _ := publishPayload["id"].(string)
	if versionID == "" {
		t.Fatalf("版本ID为空: %#v", publishPayload)
	}
	versionRun := requestWeaverWithToken(t, handler, http.MethodPost, "/api/v1/weaver/versions/"+versionID+"/run", []byte(`{}`))
	if versionRun.Code != http.StatusOK {
		t.Fatalf("运行版本样本失败: status=%d body=%s", versionRun.Code, versionRun.Body.String())
	}

	statsRec := requestWeaverWithToken(t, handler, http.MethodGet, "/api/v1/weaver/runs/stats?limit=10", nil)
	if statsRec.Code != http.StatusOK {
		t.Fatalf("读取运行统计失败: status=%d body=%s", statsRec.Code, statsRec.Body.String())
	}
	statsPayload := decodeWeaverPayload(t, statsRec)
	if totalRuns, _ := statsPayload["total_runs"].(float64); totalRuns < 2 {
		t.Fatalf("total_runs 异常: %#v", statsPayload)
	}
	if successRuns, _ := statsPayload["success_runs"].(float64); successRuns < 1 {
		t.Fatalf("success_runs 异常: %#v", statsPayload)
	}
	if failedRuns, _ := statsPayload["failed_runs"].(float64); failedRuns < 1 {
		t.Fatalf("failed_runs 异常: %#v", statsPayload)
	}
	trends, ok := statsPayload["trends"].([]any)
	if !ok || len(trends) < 2 {
		t.Fatalf("trends 异常: %#v", statsPayload)
	}
	errorGroups, ok := statsPayload["error_groups"].([]any)
	if !ok || len(errorGroups) == 0 {
		t.Fatalf("error_groups 为空: %#v", statsPayload)
	}
	hasSourcePayloadEmpty := false
	for _, item := range errorGroups {
		group, groupOK := item.(map[string]any)
		if !groupOK {
			continue
		}
		if code, _ := group["code"].(string); code == "source_payload_empty" {
			hasSourcePayloadEmpty = true
			break
		}
	}
	if !hasSourcePayloadEmpty {
		t.Fatalf("未命中 source_payload_empty 分组: %#v", statsPayload)
	}
	draftStatsRec := requestWeaverWithToken(t, handler, http.MethodGet, "/api/v1/weaver/runs/stats?limit=10&scope=draft&target_id="+draftID, nil)
	if draftStatsRec.Code != http.StatusOK {
		t.Fatalf("按草稿过滤读取运行统计失败: status=%d body=%s", draftStatsRec.Code, draftStatsRec.Body.String())
	}
	draftStatsPayload := decodeWeaverPayload(t, draftStatsRec)
	if scope, _ := draftStatsPayload["scope"].(string); scope != "draft" {
		t.Fatalf("草稿过滤 scope 异常: %#v", draftStatsPayload)
	}
	if targetID, _ := draftStatsPayload["target_id"].(string); targetID != draftID {
		t.Fatalf("草稿过滤 target_id 异常: %#v", draftStatsPayload)
	}
	if totalRuns, _ := draftStatsPayload["total_runs"].(float64); totalRuns < 2 {
		t.Fatalf("草稿过滤 total_runs 异常: %#v", draftStatsPayload)
	}
	versionStatsRec := requestWeaverWithToken(t, handler, http.MethodGet, "/api/v1/weaver/runs/stats?limit=10&scope=version&target_id="+versionID, nil)
	if versionStatsRec.Code != http.StatusOK {
		t.Fatalf("按版本过滤读取运行统计失败: status=%d body=%s", versionStatsRec.Code, versionStatsRec.Body.String())
	}
	versionStatsPayload := decodeWeaverPayload(t, versionStatsRec)
	if scope, _ := versionStatsPayload["scope"].(string); scope != "version" {
		t.Fatalf("版本过滤 scope 异常: %#v", versionStatsPayload)
	}
	if targetID, _ := versionStatsPayload["target_id"].(string); targetID != versionID {
		t.Fatalf("版本过滤 target_id 异常: %#v", versionStatsPayload)
	}
	if totalRuns, _ := versionStatsPayload["total_runs"].(float64); totalRuns < 1 {
		t.Fatalf("版本过滤 total_runs 异常: %#v", versionStatsPayload)
	}
}

// TestWeaverDraftRoutes_NodeContractsAndValidate 验证节点契约目录与草稿DAG校验接口。
func TestWeaverDraftRoutes_NodeContractsAndValidate(t *testing.T) {
	handler, _ := buildSiteHandler(t)
	contractRec := requestWeaverWithToken(t, handler, http.MethodGet, "/api/v1/weaver/node-contracts", nil)
	if contractRec.Code != http.StatusOK {
		t.Fatalf("读取节点契约目录失败: status=%d body=%s", contractRec.Code, contractRec.Body.String())
	}
	contractPayload := decodeWeaverPayload(t, contractRec)
	catalogItems, ok := contractPayload["items"].([]any)
	if !ok || len(catalogItems) == 0 {
		t.Fatalf("节点契约目录为空: %#v", contractPayload)
	}

	createBody := []byte(`{
		"name":"validate-dag",
		"inputs":[{"name":"order","payload":{"id":"o-3001","amount":256}}],
		"mapping":{"result":"$.order.id"},
		"dag":{
			"nodes":[
				{"id":"source-1","type":"source","inputs":["order"],"outputs":["payload"]},
				{"id":"output-1","type":"output","inputs":["payload"],"outputs":["result"]}
			],
			"edges":[{"from":"source-1","to":"output-1"}],
			"output_node_id":"output-1"
		}
	}`)
	createRec := requestWeaverWithToken(t, handler, http.MethodPost, "/api/v1/weaver/drafts", createBody)
	if createRec.Code != http.StatusOK {
		t.Fatalf("创建草稿失败: status=%d body=%s", createRec.Code, createRec.Body.String())
	}
	createPayload := decodeWeaverPayload(t, createRec)
	draftID, _ := createPayload["id"].(string)
	if draftID == "" {
		t.Fatalf("草稿ID为空: %#v", createPayload)
	}

	validateRec := requestWeaverWithToken(t, handler, http.MethodPost, "/api/v1/weaver/drafts/"+draftID+"/validate", []byte(`{}`))
	if validateRec.Code != http.StatusOK {
		t.Fatalf("校验草稿DAG失败: status=%d body=%s", validateRec.Code, validateRec.Body.String())
	}
	validatePayload := decodeWeaverPayload(t, validateRec)
	if status, _ := validatePayload["status"].(string); status != "valid" {
		t.Fatalf("草稿校验状态异常: %#v", validatePayload)
	}
	frozenContracts, ok := validatePayload["node_contracts"].([]any)
	if !ok || len(frozenContracts) != 2 {
		t.Fatalf("草稿校验契约异常: %#v", validatePayload)
	}

	cycleBody := []byte(`{
		"dag":{
			"nodes":[
				{"id":"a","type":"transform","inputs":["x"],"outputs":["y"]},
				{"id":"b","type":"transform","inputs":["y"],"outputs":["z"]}
			],
			"edges":[
				{"from":"a","to":"b"},
				{"from":"b","to":"a"}
			],
			"output_node_id":"b"
		}
	}`)
	invalidRec := requestWeaverWithToken(t, handler, http.MethodPost, "/api/v1/weaver/drafts/"+draftID+"/validate", cycleBody)
	if invalidRec.Code != http.StatusBadRequest {
		t.Fatalf("循环DAG校验应失败: status=%d body=%s", invalidRec.Code, invalidRec.Body.String())
	}
	invalidPayload := decodeWeaverPayload(t, invalidRec)
	if errorCode, _ := invalidPayload["error_code"].(string); errorCode != "weaver_invalid_dag" {
		t.Fatalf("循环DAG错误码异常: %#v", invalidPayload)
	}
}

// TestWeaverDraftRoutes_InvalidPayloads 验证 Weaver 草稿接口的异常路径与错误码行为。
func TestWeaverDraftRoutes_InvalidPayloads(t *testing.T) {
	handler, _ := buildSiteHandler(t)

	testCases := []struct {
		name       string
		method     string
		target     string
		body       []byte
		statusCode int
		errorCode  string
	}{
		{
			name:       "创建草稿缺少name",
			method:     http.MethodPost,
			target:     "/api/v1/weaver/drafts",
			body:       []byte(`{"inputs":[{"name":"a","payload":{}}],"mapping":{}}`),
			statusCode: http.StatusBadRequest,
			errorCode:  "invalid_request",
		},
		{
			name:       "运行草稿ID不存在",
			method:     http.MethodPost,
			target:     "/api/v1/weaver/drafts/not-exist/run",
			body:       []byte(`{}`),
			statusCode: http.StatusNotFound,
			errorCode:  "draft_not_found",
		},
		{
			name:       "删除草稿ID不存在",
			method:     http.MethodDelete,
			target:     "/api/v1/weaver/drafts/not-exist",
			body:       nil,
			statusCode: http.StatusNotFound,
			errorCode:  "draft_not_found",
		},
		{
			name:       "读取流程版本不存在",
			method:     http.MethodGet,
			target:     "/api/v1/weaver/versions/not-exist",
			body:       nil,
			statusCode: http.StatusNotFound,
			errorCode:  "version_not_found",
		},
		{
			name:       "运行统计过滤缺少target_id",
			method:     http.MethodGet,
			target:     "/api/v1/weaver/runs/stats?scope=draft",
			body:       nil,
			statusCode: http.StatusBadRequest,
			errorCode:  "invalid_request",
		},
		{
			name:       "校验草稿ID不存在",
			method:     http.MethodPost,
			target:     "/api/v1/weaver/drafts/not-exist/validate",
			body:       []byte(`{}`),
			statusCode: http.StatusNotFound,
			errorCode:  "draft_not_found",
		},
		{
			name:       "节点契约目录方法不允许",
			method:     http.MethodPost,
			target:     "/api/v1/weaver/node-contracts",
			body:       []byte(`{}`),
			statusCode: http.StatusMethodNotAllowed,
			errorCode:  "method_not_allowed",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			rec := requestWeaverWithToken(t, handler, tc.method, tc.target, tc.body)
			if rec.Code != tc.statusCode {
				t.Fatalf("状态码不匹配: got=%d want=%d body=%s", rec.Code, tc.statusCode, rec.Body.String())
			}
			payload := decodeWeaverPayload(t, rec)
			got, _ := payload["error_code"].(string)
			if got != tc.errorCode {
				t.Fatalf("错误码不匹配: got=%q want=%q", got, tc.errorCode)
			}
		})
	}
}

// requestWeaverWithToken 构造带鉴权信息的 Weaver 请求并返回响应记录器。
func requestWeaverWithToken(
	t *testing.T,
	handler http.Handler,
	method string,
	target string,
	body []byte,
) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, "http://example.com"+target, bytes.NewReader(body))
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-Operator", "tester")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// decodeWeaverPayload 解析 Weaver 接口响应为 map 结构，便于断言字段。
func decodeWeaverPayload(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var payload map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	return payload
}
