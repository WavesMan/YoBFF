import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import type { WeaverDAG, WeaverNodeContract } from '../../../admin/types'
import { WeaverGraphSkeleton } from '../WeaverGraphSkeleton'

describe('WeaverGraphSkeleton', () => {
  it('添加节点时应用契约模板端口', () => {
    const handleChange = vi.fn()
    const dag: WeaverDAG = {
      nodes: [],
      edges: [],
      output_node_id: '',
    }
    const nodeContracts: WeaverNodeContract[] = [
      {
        node_id: 'template-source',
        type: 'source',
        inputs: [],
        outputs: ['payload'],
      },
    ]
    render(<WeaverGraphSkeleton dag={dag} nodeContracts={nodeContracts} onChange={handleChange} />)

    fireEvent.click(screen.getByRole('button', { name: /添加节点/i }))

    expect(handleChange).toHaveBeenCalledTimes(1)
    const payload = handleChange.mock.calls[0][0] as WeaverDAG
    expect(payload.nodes).toHaveLength(1)
    expect(payload.nodes[0].type).toBe('source')
    expect(payload.nodes[0].outputs).toEqual(['payload'])
  })

  it('节点改名后同步更新连线与输出节点', () => {
    const handleChange = vi.fn()
    const dag: WeaverDAG = {
      nodes: [
        { id: 'node-a', type: 'source', inputs: [], outputs: ['payload'], config: {} },
        { id: 'node-b', type: 'transform', inputs: ['payload'], outputs: ['payload'], config: {} },
      ],
      edges: [{ from: 'node-a', to: 'node-b' }],
      output_node_id: 'node-a',
    }
    render(<WeaverGraphSkeleton dag={dag} onChange={handleChange} />)

    fireEvent.change(screen.getAllByDisplayValue('node-a')[0], { target: { value: 'node-a-1' } })

    expect(handleChange).toHaveBeenCalled()
    const payload = handleChange.mock.calls[0][0] as WeaverDAG
    expect(payload.edges[0].from).toBe('node-a-1')
    expect(payload.output_node_id).toBe('node-a-1')
  })

  it('切换节点类型时同步刷新输入输出端口', () => {
    const handleChange = vi.fn()
    const dag: WeaverDAG = {
      nodes: [{ id: 'node-1', type: 'transform', inputs: ['raw'], outputs: ['payload'], config: {} }],
      edges: [],
      output_node_id: '',
    }
    const nodeContracts: WeaverNodeContract[] = [
      {
        node_id: 'template-transform',
        type: 'transform',
        inputs: ['raw'],
        outputs: ['payload'],
      },
      {
        node_id: 'template-source',
        type: 'source',
        inputs: [],
        outputs: ['payload'],
      },
    ]
    render(<WeaverGraphSkeleton dag={dag} nodeContracts={nodeContracts} onChange={handleChange} />)

    const typeSelect = screen.getAllByRole('combobox')[0]
    fireEvent.change(typeSelect, { target: { value: 'source' } })

    expect(handleChange).toHaveBeenCalled()
    const payload = handleChange.mock.calls[0][0] as WeaverDAG
    expect(payload.nodes[0].type).toBe('source')
    expect(payload.nodes[0].inputs).toEqual([])
    expect(payload.nodes[0].outputs).toEqual(['payload'])
  })
})
