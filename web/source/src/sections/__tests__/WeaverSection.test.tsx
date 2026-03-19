import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { WeaverSection } from '../WeaverSection'

const toastSpy = {
  success: vi.fn(),
  error: vi.fn(),
}

vi.mock('../../components/ui/Toast', () => ({
  useToast: () => toastSpy,
}))

vi.mock('../../components/ui/CodeEditor', () => ({
  CodeEditor: ({ value, onChange, label }: { value: string; onChange?: (value: string) => void; label?: string }) => (
    <div>
      <span>{label}</span>
      <textarea
        aria-label={label || 'editor'}
        value={value}
        onChange={(event) => onChange?.(event.target.value)}
      />
    </div>
  ),
}))

vi.mock('../weaver/WeaverGraphSkeleton', () => ({
  WeaverGraphSkeleton: () => <div>Graph Skeleton</div>,
}))

vi.mock('../../admin/api', () => {
  class MockRequestError extends Error {
    error_code?: string
    request_id?: string

    constructor(message: string, error_code?: string, request_id?: string) {
      super(message)
      this.name = 'RequestError'
      this.error_code = error_code
      this.request_id = request_id
    }
  }

  return {
    RequestError: MockRequestError,
    createWeaverDraft: vi.fn(),
    deleteWeaverDraft: vi.fn(),
    fetchWeaverDraft: vi.fn(),
    fetchWeaverDrafts: vi.fn(),
    fetchWeaverDraftVersions: vi.fn(),
    fetchWeaverVersion: vi.fn(),
    publishWeaverDraft: vi.fn(),
    runWeaverDraft: vi.fn(),
    runWeaverVersion: vi.fn(),
    updateWeaverDraft: vi.fn(),
  }
})

describe('WeaverSection', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('在版本不存在时展示明确失败态', async () => {
    const api = await import('../../admin/api')
    const draft = {
      id: 'draft-1',
      name: '测试草稿',
      inputs: [],
      dag: { nodes: [], edges: [], output_node_id: '' },
      mapping: {},
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
      operator: 'tester',
      source: 'manual',
    }
    const version = {
      id: 'version-1',
      draft_id: 'draft-1',
      version: 1,
      name: 'v1',
      inputs: [],
      mapping: {},
      dag: { nodes: [], edges: [], output_node_id: '' },
      node_contracts: [],
      created_at: '2026-01-01T00:00:00Z',
    }
    vi.mocked(api.fetchWeaverDrafts).mockResolvedValue({ items: [draft] })
    vi.mocked(api.fetchWeaverDraft).mockResolvedValue(draft)
    vi.mocked(api.fetchWeaverDraftVersions).mockResolvedValue({ items: [version] })
    vi.mocked(api.fetchWeaverVersion).mockRejectedValue(new api.RequestError('not found', 'version_not_found'))

    render(<WeaverSection token="token-x" operator="tester" />)
    fireEvent.click(await screen.findByText('测试草稿'))

    await waitFor(() => {
      expect(screen.getByText('版本不存在，请刷新版本列表后重试')).toBeInTheDocument()
    })
  })

  it('在版本运行失败时输出友好错误提示', async () => {
    const api = await import('../../admin/api')
    const draft = {
      id: 'draft-2',
      name: '运行草稿',
      inputs: [],
      dag: { nodes: [], edges: [], output_node_id: '' },
      mapping: {},
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
      operator: 'tester',
      source: 'manual',
    }
    const version = {
      id: 'version-2',
      draft_id: 'draft-2',
      version: 2,
      name: 'v2',
      inputs: [],
      mapping: {},
      dag: { nodes: [], edges: [], output_node_id: '' },
      node_contracts: [],
      created_at: '2026-01-01T00:00:00Z',
    }
    vi.mocked(api.fetchWeaverDrafts).mockResolvedValue({ items: [draft] })
    vi.mocked(api.fetchWeaverDraft).mockResolvedValue(draft)
    vi.mocked(api.fetchWeaverDraftVersions).mockResolvedValue({ items: [version] })
    vi.mocked(api.fetchWeaverVersion).mockResolvedValue(version)
    vi.mocked(api.runWeaverVersion).mockRejectedValue(new api.RequestError('run failed', 'weaver_run_failed'))

    render(<WeaverSection token="token-y" operator="tester" />)
    fireEvent.click(await screen.findByText('运行草稿'))
    fireEvent.click(await screen.findByRole('button', { name: /运行版本/i }))

    await waitFor(() => {
      expect(toastSpy.error).toHaveBeenCalledWith('运行失败，请检查输入数据、映射规则与DAG配置')
    })
  })
})
