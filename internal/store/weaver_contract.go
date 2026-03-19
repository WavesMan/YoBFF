package store

import (
	"errors"
	"strings"
)

var weaverNodeContractCatalog = []WeaverNodeContract{
	{
		NodeID:  "template-source",
		Type:    "source",
		Inputs:  []string{},
		Outputs: []string{"payload"},
	},
	{
		NodeID:  "template-transform",
		Type:    "transform",
		Inputs:  []string{"payload"},
		Outputs: []string{"payload"},
	},
	{
		NodeID:  "template-merge",
		Type:    "merge",
		Inputs:  []string{"left", "right"},
		Outputs: []string{"payload"},
	},
	{
		NodeID:  "template-output",
		Type:    "output",
		Inputs:  []string{"payload"},
		Outputs: []string{"result"},
	},
}

// ListWeaverNodeContractCatalog 返回可视化编排可选节点契约目录。
// 参数：无。
// 返回：节点契约模板列表。
// 异常：无。
func (s *Store) ListWeaverNodeContractCatalog() []WeaverNodeContract {
	items := make([]WeaverNodeContract, 0, len(weaverNodeContractCatalog))
	items = append(items, weaverNodeContractCatalog...)
	return items
}

// ValidateWeaverDAG 校验 DAG 结构并生成节点契约快照。
// 参数：dag 为待校验的有向无环图。
// 返回：按拓扑顺序的节点契约列表。
// 异常：节点缺失、边非法、输出节点非法或存在环路时返回错误。
func (s *Store) ValidateWeaverDAG(dag WeaverDAG) ([]WeaverNodeContract, error) {
	contracts, err := freezeWeaverNodeContracts(dag)
	if err != nil {
		return nil, err
	}
	outputNodeID := strings.TrimSpace(dag.OutputNodeID)
	if outputNodeID == "" {
		return contracts, nil
	}
	for _, item := range contracts {
		if item.NodeID == outputNodeID {
			return contracts, nil
		}
	}
	return nil, errors.New("dag.output_node_id not found")
}
