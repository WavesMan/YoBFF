import { FiLink2, FiPlus, FiTrash2 } from 'react-icons/fi'
import type { WeaverDAG, WeaverDAGEdge, WeaverDAGNode } from '../../admin/types'
import { Button } from '../../components/ui/Button'
import { Input } from '../../components/ui/Input'
import './WeaverGraphSkeleton.css'

type WeaverGraphSkeletonProps = {
  dag: WeaverDAG
  onChange: (dag: WeaverDAG) => void
}

/**
 *
 * WeaverGraphSkeleton 提供可视化编排阶段A骨架，用于节点与连线的基础编辑。
 *
 */
export function WeaverGraphSkeleton({ dag, onChange }: WeaverGraphSkeletonProps) {
  /**
   *
   * buildNodeId 生成唯一节点标识，用于新增节点时避免重复。
   *
   */
  const buildNodeId = () => `node-${Date.now()}`

  /**
   *
   * addNode 新增一个默认节点，用于快速搭建编排骨架。
   *
   */
  const addNode = () => {
    const node: WeaverDAGNode = {
      id: buildNodeId(),
      type: 'transform',
      inputs: [],
      outputs: [],
      config: {},
    }
    onChange({
      ...dag,
      nodes: [...(dag.nodes || []), node],
    })
  }

  /**
   *
   * removeNode 删除目标节点，并同步移除相关连线与输出节点引用。
   *
   */
  const removeNode = (nodeId: string) => {
    const nextNodes = (dag.nodes || []).filter((item) => item.id !== nodeId)
    const nextEdges = (dag.edges || []).filter((item) => item.from !== nodeId && item.to !== nodeId)
    const nextOutputNodeID = dag.output_node_id === nodeId ? '' : dag.output_node_id
    onChange({
      ...dag,
      nodes: nextNodes,
      edges: nextEdges,
      output_node_id: nextOutputNodeID,
    })
  }

  /**
   *
   * updateNode 更新节点字段，用于基础属性编辑。
   *
   */
  const updateNode = (nodeId: string, patch: Partial<WeaverDAGNode>) => {
    const nextNodes = (dag.nodes || []).map((item) => {
      if (item.id !== nodeId) {
        return item
      }
      return {
        ...item,
        ...patch,
      }
    })
    onChange({
      ...dag,
      nodes: nextNodes,
    })
  }

  /**
   *
   * addEdge 新增默认连线，用于建立节点依赖关系。
   *
   */
  const addEdge = () => {
    const firstNode = dag.nodes?.[0]?.id || ''
    const secondNode = dag.nodes?.[1]?.id || ''
    const edge: WeaverDAGEdge = {
      from: firstNode,
      to: secondNode,
    }
    onChange({
      ...dag,
      edges: [...(dag.edges || []), edge],
    })
  }

  /**
   *
   * updateEdge 更新连线字段，用于调整依赖方向。
   *
   */
  const updateEdge = (index: number, patch: Partial<WeaverDAGEdge>) => {
    const nextEdges = (dag.edges || []).map((item, itemIndex) => {
      if (itemIndex !== index) {
        return item
      }
      return {
        ...item,
        ...patch,
      }
    })
    onChange({
      ...dag,
      edges: nextEdges,
    })
  }

  /**
   *
   * removeEdge 删除目标连线，用于清理无效依赖关系。
   *
   */
  const removeEdge = (index: number) => {
    const nextEdges = (dag.edges || []).filter((_, itemIndex) => itemIndex !== index)
    onChange({
      ...dag,
      edges: nextEdges,
    })
  }

  return (
    <div className="weaver-graph-card">
      <div className="weaver-graph-header">
        <span className="weaver-graph-title">可视化编排（阶段A骨架）</span>
        <div className="weaver-graph-actions">
          <Button size="sm" variant="secondary" onClick={addNode}>
            <FiPlus className="mr-1" /> 添加节点
          </Button>
          <Button size="sm" variant="secondary" onClick={addEdge} disabled={(dag.nodes || []).length < 2}>
            <FiLink2 className="mr-1" /> 添加连线
          </Button>
        </div>
      </div>

      <div className="weaver-graph-body">
        <div className="weaver-graph-column">
          <div className="weaver-graph-subtitle">节点</div>
          {(dag.nodes || []).length === 0 ? (
            <div className="weaver-graph-empty">暂无节点，请先添加节点</div>
          ) : (
            (dag.nodes || []).map((node) => (
              <div key={node.id} className="weaver-graph-item">
                <Input
                  value={node.id}
                  onChange={(event) => updateNode(node.id, { id: event.target.value })}
                  placeholder="节点ID"
                />
                <Input
                  value={node.type}
                  onChange={(event) => updateNode(node.id, { type: event.target.value })}
                  placeholder="节点类型"
                />
                <Button size="sm" variant="danger" onClick={() => removeNode(node.id)}>
                  <FiTrash2 size={14} />
                </Button>
              </div>
            ))
          )}
        </div>

        <div className="weaver-graph-column">
          <div className="weaver-graph-subtitle">连线</div>
          {(dag.edges || []).length === 0 ? (
            <div className="weaver-graph-empty">暂无连线，请先添加连线</div>
          ) : (
            (dag.edges || []).map((edge, index) => (
              <div key={`${edge.from}-${edge.to}-${index}`} className="weaver-graph-item">
                <Input
                  value={edge.from}
                  onChange={(event) => updateEdge(index, { from: event.target.value })}
                  placeholder="from 节点ID"
                />
                <Input
                  value={edge.to}
                  onChange={(event) => updateEdge(index, { to: event.target.value })}
                  placeholder="to 节点ID"
                />
                <Button size="sm" variant="danger" onClick={() => removeEdge(index)}>
                  <FiTrash2 size={14} />
                </Button>
              </div>
            ))
          )}
        </div>
      </div>

      <div className="weaver-graph-footer">
        <Input
          value={dag.output_node_id || ''}
          onChange={(event) => onChange({ ...dag, output_node_id: event.target.value })}
          placeholder="输出节点ID（可选）"
        />
      </div>
    </div>
  )
}
