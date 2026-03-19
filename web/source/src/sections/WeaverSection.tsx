import { useCallback, useEffect, useState } from 'react'
import { FiLayers, FiPlay, FiSave, FiPlus, FiTrash2, FiBox, FiGitBranch } from 'react-icons/fi'
import {
  createWeaverDraft,
  deleteWeaverDraft,
  fetchWeaverDraft,
  fetchWeaverDrafts,
  fetchWeaverNodeContracts,
  fetchWeaverDraftVersions,
  fetchWeaverVersion,
  publishWeaverDraft,
  runWeaverDraft,
  fetchWeaverRunStats,
  runWeaverVersion,
  updateWeaverDraft,
  validateWeaverDraft
} from '../admin/api'
import type {
  WeaverDAG,
  WeaverDraft,
  WeaverNodeContract,
  WeaverRunResponse,
  WeaverRunRetryPolicy,
  WeaverRunStatsResponse,
  WeaverVersion
} from '../admin/types'
import { Button } from '../components/ui/Button'
import { Input } from '../components/ui/Input'
import { useToast } from '../components/ui/Toast'
import { CodeEditor } from '../components/ui/CodeEditor'
import { WeaverGraphSkeleton } from './weaver/WeaverGraphSkeleton'
import { buildEmptyDag, buildVersionSnapshot, resolveWeaverRequestError } from './weaver/utils'
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

type RunStatsScopeMode = 'all' | 'draft' | 'version'

const defaultWeaverRunRetryPolicy: WeaverRunRetryPolicy = {
  max_attempts: 3,
  retry_on_node_error: true,
  retry_on_source_error: false,
  retryable_codes: ['node_attempt_guard', 'node_simulated_error'],
  backoff_initial_ms: 150,
  backoff_multiplier: 2,
  backoff_max_ms: 1200,
}

/**
 *
 * 可视化实验室区域，用于编辑草稿、运行映射与查看结果。
 *
 */
export function WeaverSection({ token, operator }: WeaverSectionProps) {
  const toast = useToast()
  const [draftList, setDraftList] = useState<WeaverDraft[]>([])
  const [versionList, setVersionList] = useState<WeaverVersion[]>([])
  const [draftId, setDraftId] = useState<string | null>(null)
  const [selectedVersionId, setSelectedVersionId] = useState<string | null>(null)
  const [draftName, setDraftName] = useState('')
  const [inputItems, setInputItems] = useState<DraftFormInput[]>([])
  const [dagValue, setDagValue] = useState<WeaverDAG>(buildEmptyDag())
  const [mappingText, setMappingText] = useState('')
  const [runResult, setRunResult] = useState<WeaverRunResponse | null>(null)
  const [dagJsonMode, setDagJsonMode] = useState<'visual' | 'advanced'>('visual')
  const [dagText, setDagText] = useState(JSON.stringify(buildEmptyDag(), null, 2))
  const [dagJsonError, setDagJsonError] = useState('')
  const [retryPolicy, setRetryPolicy] = useState<WeaverRunRetryPolicy>(defaultWeaverRunRetryPolicy)
  const [retryCodeText, setRetryCodeText] = useState((defaultWeaverRunRetryPolicy.retryable_codes || []).join(','))
  const [nodeContractCatalog, setNodeContractCatalog] = useState<WeaverNodeContract[]>([])
  const [validatedContracts, setValidatedContracts] = useState<WeaverNodeContract[]>([])
  const [runStats, setRunStats] = useState<WeaverRunStatsResponse | null>(null)
  const [runStatsScope, setRunStatsScope] = useState<RunStatsScopeMode>('all')
  const [selectedVersionDetail, setSelectedVersionDetail] = useState<WeaverVersion | null>(null)
  const [versionDetailError, setVersionDetailError] = useState('')
  const [loadingList, setLoadingList] = useState(false)
  const [loadingVersions, setLoadingVersions] = useState(false)
  const [loadingVersionDetail, setLoadingVersionDetail] = useState(false)
  const [savingDraft, setSavingDraft] = useState(false)
  const [publishingDraft, setPublishingDraft] = useState(false)
  const [runningDraft, setRunningDraft] = useState(false)
  const [runningVersion, setRunningVersion] = useState(false)
  const [validatingDag, setValidatingDag] = useState(false)
  const [loadingRunStats, setLoadingRunStats] = useState(false)

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
   * 读取草稿下的版本列表，用于选择与运行发布版本。
   *
   */
  const loadVersionList = useCallback(async (targetDraftId: string) => {
    if (!token || !targetDraftId) {
      setVersionList([])
      setSelectedVersionId(null)
      return
    }
    setLoadingVersions(true)
    try {
      const response = await fetchWeaverDraftVersions(token, targetDraftId, 20)
      const items = response.items || []
      setVersionList(items)
      setSelectedVersionId((prev) => {
        if (prev && items.some((item) => item.id === prev)) {
          return prev
        }
        return items.length > 0 ? items[0].id : null
      })
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '读取版本列表失败')
      setVersionList([])
      setSelectedVersionId(null)
    } finally {
      setLoadingVersions(false)
    }
  }, [token, toast])

  /**
   *
   * 读取节点契约目录，用于辅助编排节点配置。
   *
   */
  const loadNodeContractCatalog = useCallback(async () => {
    if (!token) {
      setNodeContractCatalog([])
      return
    }
    try {
      const response = await fetchWeaverNodeContracts(token)
      setNodeContractCatalog(response.items || [])
    } catch (error) {
      toast.error(resolveWeaverRequestError(error, '读取节点契约目录失败'))
      setNodeContractCatalog([])
    }
  }, [token, toast])

  /**
   *
   * 读取最近运行统计，用于展示错误码分组与趋势变化。
   *
   */
  const loadRunStats = useCallback(async (
    scopeMode: RunStatsScopeMode,
    currentDraftID: string | null,
    currentVersionID: string | null
  ) => {
    if (!token) {
      setRunStats(null)
      return
    }
    let scope: 'draft' | 'version' | undefined
    let targetID: string | undefined
    if (scopeMode === 'draft') {
      if (!currentDraftID) {
        setRunStats(null)
        return
      }
      scope = 'draft'
      targetID = currentDraftID
    }
    if (scopeMode === 'version') {
      if (!currentVersionID) {
        setRunStats(null)
        return
      }
      scope = 'version'
      targetID = currentVersionID
    }
    setLoadingRunStats(true)
    try {
      const response = await fetchWeaverRunStats(token, {
        limit: 20,
        scope,
        target_id: targetID,
      })
      setRunStats(response)
    } catch (error) {
      toast.error(resolveWeaverRequestError(error, '读取运行统计失败'))
      setRunStats(null)
    } finally {
      setLoadingRunStats(false)
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
    setDagValue(draft.dag ?? buildEmptyDag())
    setDagText(JSON.stringify(draft.dag ?? buildEmptyDag(), null, 2))
    setDagJsonError('')
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
      await loadVersionList(draft.id)
      setSelectedVersionDetail(null)
      setVersionDetailError('')
      setRunResult(null)
      setValidatedContracts([])
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
    setVersionList([])
    setSelectedVersionId(null)
    setDraftName('')
    setInputItems([])
    setDagValue(buildEmptyDag())
    setDagText(JSON.stringify(buildEmptyDag(), null, 2))
    setDagJsonError('')
    setMappingText('')
    setSelectedVersionDetail(null)
    setVersionDetailError('')
    setRunResult(null)
    setValidatedContracts([])
  }

  /**
   *
   * 删除当前草稿，用于清理无效实验记录。
   *
   */
  const handleDeleteDraft = async () => {
    if (!token) {
      toast.error('请先登录后再删除')
      return
    }
    if (!draftId) {
      toast.error('请先选择草稿')
      return
    }
    try {
      await deleteWeaverDraft(token, draftId, operator)
      toast.success('草稿删除成功')
      handleCreateDraft()
      await loadDraftList()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '删除失败')
    }
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
    if (dagJsonError) {
      throw new Error(dagJsonError)
    }
    return {
      name: draftName.trim(),
      inputs: inputsPayload,
      dag: dagValue,
      mapping: mappingValue as Record<string, unknown>,
    }
  }

  /**
   *
   * 解析输入源覆盖参数，用于版本运行时替换冻结输入。
   *
   */
  const buildRunInputsPayload = () => {
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
    const retryableCodes = retryCodeText
      .split(',')
      .map((item) => item.trim().toLowerCase())
      .filter((item, index, list) => item.length > 0 && list.indexOf(item) === index)
    const retryPayload: WeaverRunRetryPolicy = {
      max_attempts: retryPolicy.max_attempts,
      retry_on_node_error: retryPolicy.retry_on_node_error,
      retry_on_source_error: retryPolicy.retry_on_source_error,
      retryable_codes: retryableCodes,
      backoff_initial_ms: retryPolicy.backoff_initial_ms,
      backoff_multiplier: retryPolicy.backoff_multiplier,
      backoff_max_ms: retryPolicy.backoff_max_ms,
    }
    return {
      inputs: inputsPayload,
      retry: retryPayload,
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
        await loadVersionList(draftId)
      } else {
        const createdDraft = await createWeaverDraft(token, payload, operator)
        setDraftId(createdDraft.id)
        setVersionList([])
        setSelectedVersionId(null)
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
      const retryPayload = buildRunInputsPayload().retry
      const result = await runWeaverDraft(
        token,
        draftId,
        { ...payload, retry: retryPayload },
        operator
      )
      setRunResult(result)
      await loadRunStats(runStatsScope, draftId, selectedVersionId)
      if (result.status === 'failed') {
        toast.error(`运行失败，run_id=${result.run_id}`)
      } else {
        toast.success(`运行成功，run_id=${result.run_id}`)
      }
    } catch (error) {
      toast.error(resolveWeaverRequestError(error, '运行失败'))
    } finally {
      setRunningDraft(false)
    }
  }

  /**
   *
   * 发布当前草稿，用于生成可追踪版本快照。
   *
   */
  const handlePublishDraft = async () => {
    if (!token) {
      toast.error('请先登录后再发布')
      return
    }
    if (!draftId) {
      toast.error('请先保存草稿')
      return
    }
    setPublishingDraft(true)
    try {
      const version = await publishWeaverDraft(token, draftId, operator)
      toast.success(`发布成功，版本号 v${version.version}`)
      await loadVersionList(draftId)
      setSelectedVersionId(version.id)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '发布失败')
    } finally {
      setPublishingDraft(false)
    }
  }

  /**
   *
   * 运行已发布版本，用于验证发布快照执行结果。
   *
   */
  const handleRunVersion = async () => {
    if (!token) {
      toast.error('请先登录后再运行')
      return
    }
    if (!selectedVersionId) {
      toast.error('请先选择版本')
      return
    }
    setRunningVersion(true)
    setRunResult(null)
    try {
      const payload = buildRunInputsPayload()
      const result = await runWeaverVersion(token, selectedVersionId, payload, operator)
      setRunResult(result)
      await loadRunStats(runStatsScope, draftId, selectedVersionId)
      if (result.status === 'failed') {
        toast.error(`版本运行失败，run_id=${result.run_id}`)
      } else {
        toast.success(`版本运行成功，run_id=${result.run_id}`)
      }
    } catch (error) {
      toast.error(resolveWeaverRequestError(error, '版本运行失败'))
    } finally {
      setRunningVersion(false)
    }
  }

  /**
   *
   * 校验当前草稿DAG，用于发布前识别环路与输出节点配置问题。
   *
   */
  const handleValidateDraft = async () => {
    if (!token) {
      toast.error('请先登录后再校验')
      return
    }
    if (!draftId) {
      toast.error('请先保存草稿')
      return
    }
    setValidatingDag(true)
    try {
      const result = await validateWeaverDraft(token, draftId, { dag: dagValue as Record<string, unknown> })
      setValidatedContracts(result.node_contracts || [])
      toast.success('DAG校验通过')
    } catch (error) {
      setValidatedContracts([])
      toast.error(resolveWeaverRequestError(error, 'DAG校验失败'))
    } finally {
      setValidatingDag(false)
    }
  }

  useEffect(() => {
    loadDraftList()
    loadNodeContractCatalog()
  }, [loadDraftList, loadNodeContractCatalog])

  useEffect(() => {
    loadRunStats(runStatsScope, draftId, selectedVersionId)
  }, [loadRunStats, runStatsScope, draftId, selectedVersionId])

  useEffect(() => {
    setDagText(JSON.stringify(dagValue, null, 2))
    setDagJsonError('')
    setValidatedContracts([])
  }, [dagValue])

  /**
   *
   * 读取当前选中版本详情，用于展示冻结输入、映射与DAG快照。
   *
   */
  useEffect(() => {
    if (!token || !selectedVersionId) {
      setSelectedVersionDetail(null)
      setVersionDetailError('')
      return
    }
    let cancelled = false
    const loadVersionDetail = async () => {
      setLoadingVersionDetail(true)
      setVersionDetailError('')
      try {
        const detail = await fetchWeaverVersion(token, selectedVersionId)
        if (cancelled) {
          return
        }
        setSelectedVersionDetail(detail)
      } catch (error) {
        if (cancelled) {
          return
        }
        setSelectedVersionDetail(null)
        setVersionDetailError(resolveWeaverRequestError(error, '读取版本详情失败'))
      } finally {
        if (!cancelled) {
          setLoadingVersionDetail(false)
        }
      }
    }
    loadVersionDetail()
    return () => {
      cancelled = true
    }
  }, [token, selectedVersionId])

  const statsScopeLabel = runStatsScope === 'draft' ? '当前草稿' : runStatsScope === 'version' ? '当前版本' : '全局'
  const statsScopeMissing = (runStatsScope === 'draft' && !draftId) || (runStatsScope === 'version' && !selectedVersionId)

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
        <div className="weaver-sidebar-header">
          <h3 className="weaver-sidebar-title">版本列表</h3>
        </div>
        <div className="weaver-list">
          {!draftId ? (
            <div className="p-4 text-center text-secondary">请先选择草稿</div>
          ) : loadingVersions ? (
            <div className="p-4 text-center text-secondary">加载中...</div>
          ) : versionList.length === 0 ? (
            <div className="p-4 text-center text-secondary">暂无发布版本</div>
          ) : (
            versionList.map((version) => (
              <div
                key={version.id}
                className={`weaver-list-item ${selectedVersionId === version.id ? 'active' : ''}`}
                onClick={() => {
                  setSelectedVersionId(version.id)
                  setRunResult(null)
                }}
              >
                <div className="weaver-item-title">v{version.version} · {version.name}</div>
                <div className="weaver-item-meta">
                  <span>{version.operator || '-'}</span>
                  <span>{new Date(version.created_at).toLocaleDateString()}</span>
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
            <Button variant="secondary" onClick={handleDeleteDraft} disabled={!draftId || savingDraft || runningDraft}>
              <FiTrash2 className="mr-1" /> 删除
            </Button>
            <Button variant="secondary" onClick={handleSaveDraft} disabled={savingDraft}>
              <FiSave className="mr-1" /> {savingDraft ? '保存中...' : '保存'}
            </Button>
            <Button
              variant="secondary"
              onClick={handlePublishDraft}
              disabled={!draftId || publishingDraft || savingDraft || runningDraft || runningVersion}
            >
              <FiGitBranch className="mr-1" /> {publishingDraft ? '发布中...' : '发布'}
            </Button>
            <Button
              variant="secondary"
              onClick={handleValidateDraft}
              disabled={!draftId || validatingDag || savingDraft || runningDraft || runningVersion}
            >
              <FiGitBranch className="mr-1" /> {validatingDag ? '校验中...' : '校验DAG'}
            </Button>
            <Button onClick={handleRunDraft} disabled={runningDraft}>
              <FiPlay className="mr-1" /> {runningDraft ? '运行中...' : '运行'}
            </Button>
            <Button
              onClick={handleRunVersion}
              disabled={!selectedVersionId || runningVersion || runningDraft || publishingDraft}
            >
              <FiPlay className="mr-1" /> {runningVersion ? '版本运行中...' : '运行版本'}
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
              <div className="weaver-editor-mode-bar">
                <div className="weaver-form-section-title mb-0">
                  <span>DAG 编排</span>
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    size="sm"
                    variant={dagJsonMode === 'visual' ? 'primary' : 'secondary'}
                    onClick={() => setDagJsonMode('visual')}
                  >
                    可视化模式
                  </Button>
                  <Button
                    size="sm"
                    variant={dagJsonMode === 'advanced' ? 'primary' : 'secondary'}
                    onClick={() => setDagJsonMode('advanced')}
                  >
                    高级 JSON
                  </Button>
                </div>
              </div>
              {dagJsonMode === 'visual' ? (
                <WeaverGraphSkeleton dag={dagValue} nodeContracts={nodeContractCatalog} onChange={setDagValue} />
              ) : (
                <div className="flex flex-col gap-2">
                  <CodeEditor
                    label="DAG Configuration"
                    value={dagText}
                    onChange={(val) => {
                      const nextText = val || ''
                      setDagText(nextText)
                      try {
                        const nextValue = JSON.parse(nextText) as WeaverDAG
                        setDagValue(nextValue)
                        setDagJsonError('')
                      } catch {
                        setDagJsonError('DAG JSON 解析失败')
                      }
                    }}
                    language="json"
                    height={260}
                  />
                  {dagJsonError && (
                    <div className="p-3 rounded text-sm bg-error-dim">{dagJsonError}</div>
                  )}
                </div>
              )}
              <div className="flex flex-col">
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
              <div className="weaver-retry-panel">
                <div className="weaver-form-section-title">
                  <span>重试策略配置</span>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                  <Input
                    label="最大尝试次数"
                    type="number"
                    min={1}
                    max={5}
                    value={String(retryPolicy.max_attempts ?? 3)}
                    onChange={(event) => setRetryPolicy((prev) => ({
                      ...prev,
                      max_attempts: Number(event.target.value) || 1,
                    }))}
                  />
                  <Input
                    label="初始退避(ms)"
                    type="number"
                    min={0}
                    value={String(retryPolicy.backoff_initial_ms ?? 0)}
                    onChange={(event) => setRetryPolicy((prev) => ({
                      ...prev,
                      backoff_initial_ms: Number(event.target.value) || 0,
                    }))}
                  />
                  <Input
                    label="退避倍率"
                    type="number"
                    min={1}
                    step="0.1"
                    value={String(retryPolicy.backoff_multiplier ?? 2)}
                    onChange={(event) => setRetryPolicy((prev) => ({
                      ...prev,
                      backoff_multiplier: Number(event.target.value) || 1,
                    }))}
                  />
                  <Input
                    label="最大退避(ms)"
                    type="number"
                    min={0}
                    value={String(retryPolicy.backoff_max_ms ?? 0)}
                    onChange={(event) => setRetryPolicy((prev) => ({
                      ...prev,
                      backoff_max_ms: Number(event.target.value) || 0,
                    }))}
                  />
                  <Input
                    label="重试错误码(逗号分隔)"
                    value={retryCodeText}
                    onChange={(event) => setRetryCodeText(event.target.value)}
                    className="md:col-span-2"
                  />
                </div>
                <div className="mt-3 flex items-center gap-2">
                  <Button
                    size="sm"
                    variant={retryPolicy.retry_on_node_error ? 'primary' : 'secondary'}
                    onClick={() => setRetryPolicy((prev) => ({
                      ...prev,
                      retry_on_node_error: !prev.retry_on_node_error,
                    }))}
                  >
                    节点错误重试
                  </Button>
                  <Button
                    size="sm"
                    variant={retryPolicy.retry_on_source_error ? 'primary' : 'secondary'}
                    onClick={() => setRetryPolicy((prev) => ({
                      ...prev,
                      retry_on_source_error: !prev.retry_on_source_error,
                    }))}
                  >
                    输入错误重试
                  </Button>
                </div>
              </div>
              <div className="weaver-result-panel flex flex-col">
                <div className="weaver-form-section-title">
                  <FiBox className="mr-2" /> 节点契约目录
                </div>
                {dagJsonMode === 'visual' ? (
                  <div className="weaver-contract-list">
                    {nodeContractCatalog.length === 0 ? (
                      <div className="p-3 rounded text-sm bg-placeholder">暂无节点契约目录</div>
                    ) : (
                      nodeContractCatalog.map((contract) => (
                        <div key={contract.node_id} className="weaver-contract-card">
                          <div className="weaver-contract-head">
                            <span>{contract.node_id}</span>
                            <span>{contract.type}</span>
                          </div>
                          <div className="weaver-contract-meta">
                            <span>inputs: {contract.inputs.join(', ') || '无'}</span>
                            <span>outputs: {contract.outputs.join(', ') || '无'}</span>
                          </div>
                        </div>
                      ))
                    )}
                  </div>
                ) : (
                  <CodeEditor
                    label="Node Contract Catalog"
                    value={JSON.stringify(nodeContractCatalog, null, 2)}
                    language="json"
                    readOnly={true}
                    height={180}
                  />
                )}
              </div>
              <div className="weaver-result-panel flex flex-col">
                <div className="weaver-form-section-title">
                  <FiBox className="mr-2" /> DAG校验结果
                </div>
                {dagJsonMode === 'visual' ? (
                  <div className="weaver-contract-list">
                    {validatedContracts.length === 0 ? (
                      <div className="p-3 rounded text-sm bg-placeholder">暂无校验契约，请先执行 DAG 校验</div>
                    ) : (
                      validatedContracts.map((contract) => (
                        <div key={`${contract.node_id}-${contract.type}`} className="weaver-contract-card">
                          <div className="weaver-contract-head">
                            <span>{contract.node_id}</span>
                            <span>{contract.type}</span>
                          </div>
                          <div className="weaver-contract-meta">
                            <span>inputs: {contract.inputs.join(', ') || '无'}</span>
                            <span>outputs: {contract.outputs.join(', ') || '无'}</span>
                          </div>
                        </div>
                      ))
                    )}
                  </div>
                ) : (
                  <CodeEditor
                    label="Validated Node Contracts"
                    value={JSON.stringify(validatedContracts, null, 2)}
                    language="json"
                    readOnly={true}
                    height={180}
                  />
                )}
              </div>
              <div className="weaver-result-panel">
                <div className="weaver-form-section-title">
                  <FiBox className="mr-2" /> 运行统计
                  <div className="weaver-stats-switch">
                    <Button
                      size="sm"
                      variant={runStatsScope === 'all' ? 'primary' : 'secondary'}
                      onClick={() => setRunStatsScope('all')}
                    >
                      全局
                    </Button>
                    <Button
                      size="sm"
                      variant={runStatsScope === 'draft' ? 'primary' : 'secondary'}
                      disabled={!draftId}
                      onClick={() => setRunStatsScope('draft')}
                    >
                      草稿
                    </Button>
                    <Button
                      size="sm"
                      variant={runStatsScope === 'version' ? 'primary' : 'secondary'}
                      disabled={!selectedVersionId}
                      onClick={() => setRunStatsScope('version')}
                    >
                      版本
                    </Button>
                  </div>
                </div>
                {loadingRunStats ? (
                  <div className="p-4 rounded text-sm bg-placeholder">运行统计加载中...</div>
                ) : statsScopeMissing ? (
                  <div className="p-4 rounded text-sm bg-placeholder">
                    {runStatsScope === 'draft' ? '请先选择草稿后查看草稿统计' : '请先选择版本后查看版本统计'}
                  </div>
                ) : !runStats || runStats.total_runs === 0 ? (
                  <div className="p-4 rounded text-sm bg-placeholder">暂无运行统计数据</div>
                ) : (
                  <div className="weaver-stats-panel">
                    <div className="weaver-stats-scope">统计维度：{statsScopeLabel}</div>
                    <div className="weaver-stats-summary">
                      <div className="weaver-stats-card">
                        <div className="weaver-stats-label">最近样本</div>
                        <div className="weaver-stats-value">{runStats.total_runs}</div>
                      </div>
                      <div className="weaver-stats-card">
                        <div className="weaver-stats-label">成功次数</div>
                        <div className="weaver-stats-value">{runStats.success_runs}</div>
                      </div>
                      <div className="weaver-stats-card">
                        <div className="weaver-stats-label">失败次数</div>
                        <div className="weaver-stats-value">{runStats.failed_runs}</div>
                      </div>
                    </div>
                    <div className="weaver-error-groups">
                      {(runStats.error_groups || []).slice(0, 8).map((group) => (
                        <div key={group.code} className="weaver-error-chip">
                          <span className="weaver-error-code">{group.code}</span>
                          <span className="weaver-error-count">{group.count}</span>
                        </div>
                      ))}
                      {(runStats.error_groups || []).length === 0 && (
                        <div className="text-secondary text-sm">最近样本未产生失败错误码</div>
                      )}
                    </div>
                    <div className="weaver-trend-list">
                      {(runStats.trends || []).map((point, index) => {
                        const durationRatio = Math.min(100, Math.round((point.duration_ms / 3000) * 100))
                        const rowClass = point.status === 'failed' ? 'weaver-trend-row failed' : 'weaver-trend-row'
                        return (
                          <div key={point.run_id} className={rowClass}>
                            <div className="weaver-trend-head">
                              <span>#{index + 1}</span>
                              <span>{point.status}</span>
                              <span>{point.duration_ms}ms</span>
                            </div>
                            <div className="weaver-trend-bar">
                              <div
                                className="weaver-trend-fill"
                                style={{ width: `${Math.max(durationRatio, 3)}%` }}
                              />
                            </div>
                            <div className="weaver-trend-meta">
                              <span>attempts={point.attempts_used}</span>
                              <span>{point.error_codes.join(', ') || '无错误码'}</span>
                            </div>
                          </div>
                        )
                      })}
                    </div>
                  </div>
                )}
              </div>
              {selectedVersionId && (
                <div className="weaver-result-panel flex flex-col">
                  <div className="weaver-form-section-title">
                    <FiBox className="mr-2" /> 版本详情
                  </div>
                  {loadingVersionDetail ? (
                    <div className="p-4 rounded text-sm bg-placeholder">版本详情加载中...</div>
                  ) : versionDetailError ? (
                    <div className="p-4 rounded text-sm bg-error-dim">{versionDetailError}</div>
                  ) : selectedVersionDetail ? (
                    <CodeEditor
                      label="Version Snapshot"
                      value={JSON.stringify(buildVersionSnapshot(selectedVersionDetail), null, 2)}
                      language="json"
                      readOnly={true}
                      height={220}
                    />
                  ) : (
                    <div className="p-4 rounded text-sm bg-placeholder">暂无版本详情</div>
                  )}
                </div>
              )}

              {runResult && (
                <div className="weaver-result-panel flex flex-col flex-1">
                  <div className={`weaver-form-section-title ${runResult.status === 'failed' ? 'text-danger' : 'text-success'}`}>
                    <FiBox className="mr-2" /> 运行结果
                  </div>
                  <CodeEditor
                    label="Run Observability"
                    value={JSON.stringify({
                      run_id: runResult.run_id,
                      status: runResult.status,
                      duration_ms: runResult.duration_ms,
                      retry: runResult.retry,
                      failures: runResult.failures,
                      attempts: runResult.attempts,
                    }, null, 2)}
                    language="json"
                    readOnly={true}
                    height={220}
                  />
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
                  {runResult.failures.length > 0 && (
                    <div className="mt-4 p-4 rounded text-sm bg-error-dim">
                      <h4 className="font-bold mb-2">失败定位:</h4>
                      <ul className="list-disc pl-5">
                        {runResult.failures.map((item, index) => (
                          <li key={`${item.scope}-${index}`}>
                            {`attempt=${item.attempt} scope=${item.scope} code=${item.code}${item.node_id ? ` node=${item.node_id}` : ''}${item.source ? ` source=${item.source}` : ''} error=${item.error}`}
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
