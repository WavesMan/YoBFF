import { FiLink2, FiPlus, FiTrash2 } from 'react-icons/fi'
import type { WeaverDAG, WeaverDAGEdge, WeaverDAGNode, WeaverNodeContract } from '../../admin/types'
import { Button } from '../../components/ui/Button'
import { Input } from '../../components/ui/Input'
import { Select } from '../../components/ui/Select'
import './WeaverGraphSkeleton.css'

type WeaverGraphSkeletonProps = {
  dag: WeaverDAG
  nodeContracts?: WeaverNodeContract[]
  onChange: (dag: WeaverDAG) => void
}

function formatWeaverPorts(items?: string[]) {
  if (!items || items.length === 0) {
    return ''
  }
  return items.join(', ')
}

function parseWeaverPorts(text: string) {
  return text
    .split(',')
    .map((item) => item.trim())
    .filter((item) => item.length > 0)
}

export function WeaverGraphSkeleton({ dag, nodeContracts = [], onChange }: WeaverGraphSkeletonProps) {
  const nodeList = dag.nodes || []
  const edgeList = dag.edges || []

  const buildNodeId = (prefix: string) => {
    const safePrefix = prefix.trim() || 'node'
    let sequence = nodeList.length + 1
    let candidate = `${safePrefix}-${sequence}`
    const exists = (nodeID: string) => nodeList.some((node) => node.id === nodeID)
    while (exists(candidate)) {
      sequence += 1
      candidate = `${safePrefix}-${sequence}`
    }
    return candidate
  }

  const resolveNodeTemplate = (nodeType: string) => {
    return nodeContracts.find((item) => item.type === nodeType)
  }

  const buildNodeTypeOptions = () => {
    const catalogTypes = nodeContracts.map((item) => item.type)
    const nodeTypes = nodeList.map((item) => item.type)
    const merged = Array.from(new Set([...catalogTypes, ...nodeTypes].filter((item) => item.trim() !== '')))
    if (merged.length === 0) {
      return [
        { label: 'transform', value: 'transform' },
      ]
    }
    return merged.map((item) => ({
      label: item,
      value: item,
    }))
  }

  const buildNodeIDOptions = () => {
    if (nodeList.length === 0) {
      return [
        { label: '暂无节点', value: '', disabled: true },
      ]
    }
    return nodeList.map((item) => ({
      label: item.id,
      value: item.id,
    }))
  }

  const addNode = () => {
    const defaultType = nodeContracts[0]?.type || 'transform'
    const template = resolveNodeTemplate(defaultType)
    const node: WeaverDAGNode = {
      id: buildNodeId(template?.type || defaultType),
      type: template?.type || defaultType,
      inputs: template?.inputs || [],
      outputs: template?.outputs || [],
      config: {},
    }
    onChange({
      ...dag,
      nodes: [...nodeList, node],
    })
  }

  const removeNode = (nodeId: string) => {
    const nextNodes = nodeList.filter((item) => item.id !== nodeId)
    const nextEdges = edgeList.filter((item) => item.from !== nodeId && item.to !== nodeId)
    const nextOutputNodeID = dag.output_node_id === nodeId ? '' : dag.output_node_id
    onChange({
      ...dag,
      nodes: nextNodes,
      edges: nextEdges,
      output_node_id: nextOutputNodeID,
    })
  }

  const updateNode = (nodeId: string, patch: Partial<WeaverDAGNode>) => {
    const nextNodes = nodeList.map((item) => {
      if (item.id !== nodeId) {
        return item
      }
      return {
        ...item,
        ...patch,
      }
    })
    const renamedNodeID = patch.id
    if (typeof renamedNodeID !== 'string' || renamedNodeID === nodeId) {
      onChange({
        ...dag,
        nodes: nextNodes,
      })
      return
    }
    const nextEdges = edgeList.map((item) => ({
      from: item.from === nodeId ? renamedNodeID : item.from,
      to: item.to === nodeId ? renamedNodeID : item.to,
    }))
    const nextOutputNodeID = dag.output_node_id === nodeId ? renamedNodeID : dag.output_node_id
    onChange({
      ...dag,
      nodes: nextNodes,
      edges: nextEdges,
      output_node_id: nextOutputNodeID,
    })
  }

  const addEdge = () => {
    const firstNode = nodeList[0]?.id || ''
    const secondNode = nodeList[1]?.id || ''
    const edge: WeaverDAGEdge = {
      from: firstNode,
      to: secondNode,
    }
    onChange({
      ...dag,
      edges: [...edgeList, edge],
    })
  }

  const updateEdge = (index: number, patch: Partial<WeaverDAGEdge>) => {
    const nextEdges = edgeList.map((item, itemIndex) => {
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

  const removeEdge = (index: number) => {
    const nextEdges = edgeList.filter((_, itemIndex) => itemIndex !== index)
    onChange({
      ...dag,
      edges: nextEdges,
    })
  }

  const applyNodeTemplate = (nodeId: string, nodeType: string) => {
    const template = resolveNodeTemplate(nodeType)
    if (!template) {
      updateNode(nodeId, { type: nodeType })
      return
    }
    updateNode(nodeId, {
      type: template.type,
      inputs: template.inputs,
      outputs: template.outputs,
    })
  }

  const nodeTypeOptions = buildNodeTypeOptions()
  const nodeIDOptions = buildNodeIDOptions()
  const outputNodeValue = nodeList.some((item) => item.id === dag.output_node_id) ? (dag.output_node_id || '') : ''

  return (
    <div className="weaver-graph-card">
      <div className="weaver-graph-header">
        <span className="weaver-graph-title">可视化编排（M2编辑器）</span>
        <div className="weaver-graph-actions">
          <Button size="sm" variant="secondary" onClick={addNode}>
            <FiPlus className="mr-1" /> 添加节点
          </Button>
          <Button size="sm" variant="secondary" onClick={addEdge} disabled={nodeList.length < 2}>
            <FiLink2 className="mr-1" /> 添加连线
          </Button>
        </div>
      </div>

      <div className="weaver-graph-body">
        <div className="weaver-graph-column">
          <div className="weaver-graph-subtitle">节点</div>
          {nodeList.length === 0 ? (
            <div className="weaver-graph-empty">暂无节点，请先添加节点</div>
          ) : (
            nodeList.map((node) => (
              <div key={node.id} className="weaver-graph-node-item">
                <Input
                  value={node.id}
                  onChange={(event) => updateNode(node.id, { id: event.target.value })}
                  placeholder="节点ID"
                />
                <Select
                  options={nodeTypeOptions}
                  value={node.type}
                  onChange={(event) => applyNodeTemplate(node.id, event.target.value)}
                />
                <div className="weaver-graph-row">
                  <Input
                    value={formatWeaverPorts(node.inputs)}
                    onChange={(event) => updateNode(node.id, { inputs: parseWeaverPorts(event.target.value) })}
                    placeholder="输入端口（逗号分隔）"
                  />
                  <Input
                    value={formatWeaverPorts(node.outputs)}
                    onChange={(event) => updateNode(node.id, { outputs: parseWeaverPorts(event.target.value) })}
                    placeholder="输出端口（逗号分隔）"
                  />
                </div>
                <Button size="sm" variant="danger" onClick={() => removeNode(node.id)}>
                  <FiTrash2 size={14} />
                </Button>
              </div>
            ))
          )}
        </div>

        <div className="weaver-graph-column">
          <div className="weaver-graph-subtitle">连线</div>
          {edgeList.length === 0 ? (
            <div className="weaver-graph-empty">暂无连线，请先添加连线</div>
          ) : (
            edgeList.map((edge, index) => (
              <div key={`${edge.from}-${edge.to}-${index}`} className="weaver-graph-item">
                <Select
                  options={nodeIDOptions}
                  value={edge.from}
                  onChange={(event) => updateEdge(index, { from: event.target.value })}
                />
                <Select
                  options={nodeIDOptions}
                  value={edge.to}
                  onChange={(event) => updateEdge(index, { to: event.target.value })}
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
        <Select
          options={[
            { label: '请选择输出节点（可选）', value: '' },
            ...nodeIDOptions,
          ]}
          value={outputNodeValue}
          onChange={(event) => onChange({ ...dag, output_node_id: event.target.value })}
        />
      </div>
    </div>
  )
}
