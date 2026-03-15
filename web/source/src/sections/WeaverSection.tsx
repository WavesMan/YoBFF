import { useCallback, useEffect, useState } from 'react'
import { FiLayers, FiPlay, FiSave, FiPlus, FiTrash2, FiBox } from 'react-icons/fi'
import { createWeaverDraft, fetchWeaverDraft, fetchWeaverDrafts, runWeaverDraft, updateWeaverDraft } from '../admin/api'
import type { WeaverDraft, WeaverRunResponse } from '../admin/types'
import { Button } from '../components/ui/Button'
import { Input } from '../components/ui/Input'
import { useToast } from '../components/ui/Toast'
import { CodeEditor } from '../components/ui/CodeEditor'
import './WeaverSection.css'

type WeaverSectionProps = {
  token: string
  operator: string
}

type DraftFormInput = {
  id: string
  name: string
  payloadText: string
}

/**
 *
 * 可视化实验室区域，用于编辑草稿、运行映射与查看结果。
 *
 */
export function WeaverSection({ token, operator }: WeaverSectionProps) {
  const toast = useToast()
  const [draftList, setDraftList] = useState<WeaverDraft[]>([])
  const [draftId, setDraftId] = useState<string | null>(null)
  const [draftName, setDraftName] = useState('')
  const [inputItems, setInputItems] = useState<DraftFormInput[]>([])
  const [mappingText, setMappingText] = useState('')
  const [runResult, setRunResult] = useState<WeaverRunResponse | null>(null)
  const [loadingList, setLoadingList] = useState(false)
  const [savingDraft, setSavingDraft] = useState(false)
  const [runningDraft, setRunningDraft] = useState(false)

  /**
   *
   * 读取草稿列表，用于展示历史配置。
   *
   */
  const loadDraftList = useCallback(async () => {
    if (!token) {
      setDraftList([])
      return
    }
    setLoadingList(true)
    try {
      const response = await fetchWeaverDrafts(token, 50)
      setDraftList(response.items || [])
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '读取草稿失败')
    } finally {
      setLoadingList(false)
    }
  }, [token, toast])

  /**
   *
   * 将草稿信息填充到表单，用于继续编辑。
   *
   */
  const applyDraftToForm = (draft: WeaverDraft) => {
    setDraftId(draft.id)
    setDraftName(draft.name)
    const inputs = (draft.inputs || []).map((item, index) => ({
      id: `${draft.id}-${index}`,
      name: item.name,
      payloadText: JSON.stringify(item.payload ?? {}, null, 2),
    }))
    setInputItems(inputs.length > 0 ? inputs : [])
    setMappingText(JSON.stringify(draft.mapping ?? {}, null, 2))
  }

  /**
   *
   * 选择草稿进行编辑，并同步最新详情。
   *
   */
  const handleSelectDraft = async (targetId: string) => {
    if (!token) {
      return
    }
    try {
      const draft = await fetchWeaverDraft(token, targetId)
      applyDraftToForm(draft)
      setRunResult(null)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '读取草稿失败')
    }
  }

  /**
   *
   * 新建草稿并清空表单，便于重新实验。
   *
   */
  const handleCreateDraft = () => {
    setDraftId(null)
    setDraftName('')
    setInputItems([])
    setMappingText('')
    setRunResult(null)
  }

  /**
   *
   * 添加输入源，用于聚合实验。
   *
   */
  const handleAddInput = () => {
    const item: DraftFormInput = {
      id: String(Date.now()),
      name: '',
      payloadText: '{}',
    }
    setInputItems((prev) => [...prev, item])
  }

  /**
   *
   * 删除指定输入源，用于清理无效数据。
   *
   */
  const handleRemoveInput = (id: string) => {
    setInputItems((prev) => prev.filter((item) => item.id !== id))
  }

  /**
   *
   * 更新输入源名称或内容，用于实时编辑。
   *
   */
  const handleInputChange = (id: string, data: Partial<DraftFormInput>) => {
    setInputItems((prev) => prev.map((item) => (item.id === id ? { ...item, ...data } : item)))
  }

  /**
   *
   * 解析表单内容为草稿载荷，用于保存与运行。
   *
   */
  const buildDraftPayload = () => {
    if (!draftName.trim()) {
      throw new Error('草稿名称不能为空')
    }
    const inputsPayload = inputItems.map((item, index) => {
      if (!item.name.trim()) {
        throw new Error(`第 ${index + 1} 个输入源缺少名称`)
      }
      let payloadValue: unknown
      try {
        payloadValue = JSON.parse(item.payloadText || '{}')
      } catch {
        throw new Error(`输入源 ${item.name} JSON 解析失败`)
      }
      return {
        name: item.name.trim(),
        payload: payloadValue as Record<string, unknown>,
      }
    })
    let mappingValue: unknown
    try {
      mappingValue = JSON.parse(mappingText || '{}')
    } catch {
      throw new Error('映射 JSON 解析失败')
    }
    return {
      name: draftName.trim(),
      inputs: inputsPayload,
      mapping: mappingValue as Record<string, unknown>,
    }
  }

  /**
   *
   * 保存草稿配置，用于持久化当前实验。
   *
   */
  const handleSaveDraft = async () => {
    if (!token) {
      toast.error('请先登录后再保存草稿')
      return
    }
    setSavingDraft(true)
    try {
      const payload = buildDraftPayload()
      if (draftId) {
        await updateWeaverDraft(token, draftId, payload, operator)
        toast.success('草稿更新成功')
      } else {
        const newDraft = await createWeaverDraft(token, payload, operator)
        setDraftId(newDraft.id)
        toast.success('草稿创建成功')
      }
      await loadDraftList()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '保存失败')
    } finally {
      setSavingDraft(false)
    }
  }

  /**
   *
   * 运行当前映射配置，用于验证逻辑正确性。
   *
   */
  const handleRunDraft = async () => {
    if (!token) {
      toast.error('请先登录后再运行')
      return
    }
    if (!draftId) {
      toast.error('请先保存草稿')
      return
    }
    setRunningDraft(true)
    setRunResult(null)
    try {
      const payload = buildDraftPayload()
      const result = await runWeaverDraft(token, draftId, payload, operator)
      setRunResult(result)
      toast.success('运行成功')
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '运行失败')
    } finally {
      setRunningDraft(false)
    }
  }

  useEffect(() => {
    loadDraftList()
  }, [loadDraftList])

  return (
    <div className="weaver-container">
      {/* 左侧草稿列表 */}
      <div className="weaver-sidebar">
        <div className="weaver-sidebar-header">
          <h3 className="weaver-sidebar-title">草稿列表</h3>
          <Button size="sm" onClick={handleCreateDraft} disabled={loadingList}>
            <FiPlus className="mr-1" /> 新建
          </Button>
        </div>
        <div className="weaver-list">
          {loadingList ? (
            <div className="p-4 text-center text-secondary">加载中...</div>
          ) : draftList.length === 0 ? (
            <div className="p-4 text-center text-secondary">暂无草稿</div>
          ) : (
            draftList.map((draft) => (
              <div
                key={draft.id}
                className={`weaver-list-item ${draftId === draft.id ? 'active' : ''}`}
                onClick={() => handleSelectDraft(draft.id)}
              >
                <div className="weaver-item-title">{draft.name}</div>
                <div className="weaver-item-meta">
                  <span>{draft.operator}</span>
                  <span>{new Date(draft.updated_at).toLocaleDateString()}</span>
                </div>
              </div>
            ))
          )}
        </div>
      </div>

      {/* 右侧编辑区域 */}
      <div className="weaver-main">
        <div className="weaver-main-header">
          <div className="flex items-center gap-2">
            <FiLayers className="text-secondary" />
            <Input
              value={draftName}
              onChange={(e) => setDraftName(e.target.value)}
              placeholder="请输入草稿名称"
              className="w-64"
            />
          </div>
          <div className="weaver-actions">
            <Button variant="secondary" onClick={handleSaveDraft} disabled={savingDraft}>
              <FiSave className="mr-1" /> {savingDraft ? '保存中...' : '保存'}
            </Button>
            <Button onClick={handleRunDraft} disabled={runningDraft}>
              <FiPlay className="mr-1" /> {runningDraft ? '运行中...' : '运行'}
            </Button>
          </div>
        </div>

        <div className="weaver-main-content">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 h-full">
            {/* 左半部分：输入源定义 */}
            <div className="flex flex-col gap-4">
              <div className="weaver-form-section-title">
                <span>输入源定义 (Inputs)</span>
                <Button size="sm" variant="secondary" onClick={handleAddInput}>
                  <FiPlus className="mr-1" /> 添加输入源
                </Button>
              </div>
              
              {inputItems.length === 0 && (
                <div className="p-8 rounded-lg text-center bg-placeholder">
                  暂无输入源，请点击上方按钮添加
                </div>
              )}

              {inputItems.map((item, index) => (
                <div key={item.id} className="weaver-input-group">
                  <div className="weaver-input-header">
                    <div className="flex items-center gap-2 flex-1">
                      <span className="text-sm font-medium text-secondary w-8">#{index + 1}</span>
                      <Input
                        value={item.name}
                        onChange={(e) => handleInputChange(item.id, { name: e.target.value })}
                        placeholder="输入源名称 (如 user)"
                        className="flex-1"
                      />
                    </div>
                    <Button
                      size="sm"
                      variant="danger"
                      className="ml-2"
                      onClick={() => handleRemoveInput(item.id)}
                    >
                      <FiTrash2 size={14} />
                    </Button>
                  </div>
                  <CodeEditor
                    label="JSON Payload"
                    value={item.payloadText}
                    onChange={(val) => handleInputChange(item.id, { payloadText: val || '' })}
                    language="json"
                    height={200}
                  />
                </div>
              ))}
            </div>

            {/* 右半部分：映射规则与结果 */}
            <div className="flex flex-col gap-4">
              <div className="flex flex-col h-1/2">
                <div className="weaver-form-section-title">
                  <span>映射规则 (Mapping)</span>
                </div>
                <CodeEditor
                  label="Mapping Configuration"
                  value={mappingText}
                  onChange={(val) => setMappingText(val || '')}
                  language="json"
                  height="100%"
                  className="flex-1 min-h-[300px]"
                />
              </div>

              {runResult && (
                <div className="weaver-result-panel flex flex-col flex-1">
                  <div className="weaver-form-section-title text-success">
                    <FiBox className="mr-2" /> 运行结果
                  </div>
                  <CodeEditor
                    label="Result Output"
                    value={JSON.stringify(runResult.output, null, 2)}
                    language="json"
                    readOnly={true}
                    height="100%"
                    className="flex-1 min-h-[200px]"
                  />
                  {runResult.sources && runResult.sources.some(s => !s.ok) && (
                    <div className="mt-4 p-4 rounded text-sm bg-error-dim">
                      <h4 className="font-bold mb-2">执行错误:</h4>
                      <ul className="list-disc pl-5">
                        {runResult.sources.filter(s => !s.ok).map((source, i) => (
                          <li key={i}>
                            <span className="font-semibold">{source.name}:</span> {source.error}
                          </li>
                        ))}
                      </ul>
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
