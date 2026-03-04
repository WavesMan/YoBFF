package admin

import (
	"encoding/json"
	"reflect"

	"YoBFF/internal/config"
)

type configDiffItem struct {
	Path   string `json:"path"`
	Before any    `json:"before"`
	After  any    `json:"after"`
}

// buildConfigDiff 生成配置差异清单，用于审查配置变更范围。
// 参数：base 为基准配置，target 为目标配置。
// 返回：差异项列表。
// 异常：序列化失败时返回错误。
func buildConfigDiff(base config.Config, target config.Config) ([]configDiffItem, error) {
	baseMap, err := convertConfigMap(base)
	if err != nil {
		return nil, err
	}
	targetMap, err := convertConfigMap(target)
	if err != nil {
		return nil, err
	}
	var changes []configDiffItem
	collectDiffs("", baseMap, targetMap, &changes)
	return changes, nil
}

// convertConfigMap 转换配置为通用映射，降低字段遍历成本。
// 参数：cfg 为配置对象。
// 返回：配置映射。
// 异常：序列化或反序列化失败时返回错误。
func convertConfigMap(cfg config.Config) (map[string]any, error) {
	payload, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// collectDiffs 递归收集差异项，避免重复展示未变更字段。
// 参数：path 为当前路径，base 为基准值，target 为目标值，out 为差异收集结果。
// 返回：无。
// 异常：无。
func collectDiffs(path string, base any, target any, out *[]configDiffItem) {
	if reflect.DeepEqual(base, target) {
		return
	}
	baseMap, baseOK := base.(map[string]any)
	targetMap, targetOK := target.(map[string]any)
	if baseOK || targetOK {
		collectMapDiffs(path, baseMap, targetMap, out)
		return
	}
	baseList, baseListOK := base.([]any)
	targetList, targetListOK := target.([]any)
	if baseListOK || targetListOK {
		*out = append(*out, configDiffItem{
			Path:   path,
			Before: baseList,
			After:  targetList,
		})
		return
	}
	*out = append(*out, configDiffItem{
		Path:   path,
		Before: base,
		After:  target,
	})
}

// collectMapDiffs 遍历映射键集合，输出字段级差异。
// 参数：path 为当前路径，base 为基准映射，target 为目标映射，out 为差异收集结果。
// 返回：无。
// 异常：无。
func collectMapDiffs(path string, base map[string]any, target map[string]any, out *[]configDiffItem) {
	keys := make(map[string]struct{})
	for key := range base {
		keys[key] = struct{}{}
	}
	for key := range target {
		keys[key] = struct{}{}
	}
	for key := range keys {
		nextPath := key
		if path != "" {
			nextPath = path + "." + key
		}
		collectDiffs(nextPath, base[key], target[key], out)
	}
}
