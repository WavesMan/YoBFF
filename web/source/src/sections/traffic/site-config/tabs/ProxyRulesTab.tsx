import { useState } from 'react'
import { FiChevronDown, FiChevronRight, FiTrash2, FiPlus, FiAlertCircle } from 'react-icons/fi'
import type { Config, LBNode, LBPool, LBRouteRule } from '../../../../admin/types'
import { Card, CardContent, CardHeader, CardTitle } from '../../../../components/ui/Card'
import { Button } from '../../../../components/ui/Button'
import { Input } from '../../../../components/ui/Input'
import { Select } from '../../../../components/ui/Select'
import { Badge } from '../../../../components/ui/Badge'

type ProxyRulesTabProps = {
  config: Config
  updateConfig: (updater: (prev: Config) => Config) => void
  newPool: LBPool
  setNewPool: (pool: LBPool) => void
  newRoute: LBRouteRule
  setNewRoute: (route: LBRouteRule) => void
  handleAddPool: () => void
  handleRemovePool: (index: number) => void
  handleAddRoute: () => void
  handleRemoveRoute: (index: number) => void
}

export function ProxyRulesTab({
  config,
  updateConfig,
  handleAddPool,
  handleRemovePool,
}: ProxyRulesTabProps) {
  const pools = config.loadBalancer?.pools || []
  const routes = config.loadBalancer?.routes || []
  const [poolNodeDrafts, setPoolNodeDrafts] = useState<Record<string, LBNode>>({})
  const [expandedPoolKeys, setExpandedPoolKeys] = useState<Record<string, boolean>>({})

  const buildPoolDraftKey = (_pool: LBPool, index: number) => {
    return `pool_${index}`
  }

  const isURLValid = (rawValue: string) => {
    try {
      const parsed = new URL(rawValue)
      return Boolean(parsed.protocol && parsed.hostname)
    } catch {
      return false
    }
  }

  const validateNodes = (nodes: LBNode[]) => {
    const nodeErrors = nodes.map(() => ({
      id: '',
      upstream: '',
      weight: '',
    }))
    const nodeIndexByID = new Map<string, number>()

    nodes.forEach((node, index) => {
      const nodeID = (node.id || '').trim()
      const upstream = (node.upstream || '').trim()
      const weight = Number(node.weight ?? 0)

      if (!nodeID) {
        nodeErrors[index].id = '节点ID不能为空'
      } else {
        const previousIndex = nodeIndexByID.get(nodeID)
        if (previousIndex !== undefined) {
          nodeErrors[index].id = '节点ID重复'
          nodeErrors[previousIndex].id = '节点ID重复'
        } else {
          nodeIndexByID.set(nodeID, index)
        }
      }

      if (!upstream) {
        nodeErrors[index].upstream = '上游地址不能为空'
      } else if (!isURLValid(upstream)) {
        nodeErrors[index].upstream = '上游地址格式无效'
      }

      if (weight <= 0 || Number.isNaN(weight)) {
        nodeErrors[index].weight = '权重必须大于0'
      }
    })

    return nodeErrors
  }

  const validatePools = (poolItems: LBPool[]) => {
    const poolErrors = poolItems.map(() => ({
      id: '',
    }))
    const poolIndexByID = new Map<string, number>()

    poolItems.forEach((pool, index) => {
      const poolID = (pool.id || '').trim()
      if (!poolID) {
        poolErrors[index].id = '池ID不能为空'
        return
      }
      const previousIndex = poolIndexByID.get(poolID)
      if (previousIndex !== undefined) {
        poolErrors[index].id = '池ID重复'
        poolErrors[previousIndex].id = '池ID重复'
        return
      }
      poolIndexByID.set(poolID, index)
    })

    return poolErrors
  }

  const validateRoutes = (routeItems: LBRouteRule[], poolIDSet: Set<string>) => {
    return routeItems.map(route => {
      const domain = (route.domain || '').trim()
      const poolID = (route.poolId || '').trim()
      const fallbackPoolID = (route.fallbackPoolId || '').trim()
      return {
        domain: domain ? '' : '域名不能为空',
        poolId: !poolID
          ? '主池不能为空'
          : (poolIDSet.has(poolID) ? '' : '主池不存在'),
        fallbackPoolId: !fallbackPoolID
          ? ''
          : (poolIDSet.has(fallbackPoolID) ? '' : '回退池不存在'),
      }
    })
  }

  const readNodeDraft = (pool: LBPool, index: number): LBNode => {
    const key = buildPoolDraftKey(pool, index)
    const draft = poolNodeDrafts[key]
    if (draft) {
      return draft
    }
    return {
      id: '',
      upstream: '',
      weight: 1,
      enabled: true,
    }
  }

  const writeNodeDraft = (pool: LBPool, index: number, updater: (prev: LBNode) => LBNode) => {
    const key = buildPoolDraftKey(pool, index)
    const current = readNodeDraft(pool, index)
    setPoolNodeDrafts(prev => ({
      ...prev,
      [key]: updater(current),
    }))
  }

  const writePool = (poolIndex: number, updater: (pool: LBPool) => LBPool) => {
    const nextPools = [...pools]
    nextPools[poolIndex] = updater(nextPools[poolIndex] || {})
    updateConfig(prev => ({
      ...prev,
      loadBalancer: {
        ...prev.loadBalancer,
        pools: nextPools,
      },
    }))
  }

  const addPoolNode = (poolIndex: number) => {
    const pool = pools[poolIndex]
    if (!pool) {
      return
    }
    const draft = readNodeDraft(pool, poolIndex)
    const nodeID = (draft.id || '').trim()
    const upstream = (draft.upstream || '').trim()
    const weight = Number(draft.weight) > 0 ? Number(draft.weight) : 1
    if (!nodeID || !upstream) {
      return
    }
    writePool(poolIndex, currentPool => ({
      ...currentPool,
      nodes: [
        ...(currentPool.nodes || []),
        {
          id: nodeID,
          upstream,
          weight,
          enabled: draft.enabled !== false,
        },
      ],
    }))
    writeNodeDraft(pool, poolIndex, () => ({
      id: '',
      upstream: '',
      weight: 1,
      enabled: true,
    }))
  }

  const togglePoolExpanded = (pool: LBPool, index: number) => {
    const key = buildPoolDraftKey(pool, index)
    setExpandedPoolKeys(prev => ({
      ...prev,
      [key]: !prev[key],
    }))
  }

  const removePoolWithState = (pool: LBPool, index: number) => {
    const key = buildPoolDraftKey(pool, index)
    setExpandedPoolKeys(prev => {
      if (!prev[key]) {
        return prev
      }
      const next = { ...prev }
      delete next[key]
      return next
    })
    setPoolNodeDrafts(prev => {
      if (!prev[key]) {
        return prev
      }
      const next = { ...prev }
      delete next[key]
      return next
    })
    handleRemovePool(index)
  }

  const poolSelectOptions = pools
    .map((pool, index) => {
      const id = (pool.id || '').trim()
      if (!id) {
        return null
      }
      return {
        value: id,
        label: pool.name ? `${pool.name} (${id})` : id,
        key: `${id}-${index}`,
      }
    })
    .filter((item): item is { value: string; label: string; key: string } => Boolean(item))

  const poolErrors = validatePools(pools)
  const nodeErrorsByPool = pools.map(pool => validateNodes(pool.nodes || []))
  const poolIDSet = new Set(
    pools
      .map(pool => (pool.id || '').trim())
      .filter(Boolean),
  )
  const routeErrors = validateRoutes(routes, poolIDSet)
  const defaultPoolID = (config.loadBalancer?.defaultPoolId || '').trim()
  const defaultPoolError = defaultPoolID && !poolIDSet.has(defaultPoolID)
    ? '默认流量池不存在'
    : ''

  const summaryItems: string[] = []
  if (defaultPoolError) {
    summaryItems.push(`默认流量池: ${defaultPoolError}`)
  }
  poolErrors.forEach((poolError, index) => {
    if (poolError.id) {
      summaryItems.push(`流量池 #${index + 1}: ${poolError.id}`)
    }
  })
  nodeErrorsByPool.forEach((nodeErrors, poolIndex) => {
    nodeErrors.forEach((nodeError, nodeIndex) => {
      if (nodeError.id) {
        summaryItems.push(`流量池 #${poolIndex + 1} 节点 #${nodeIndex + 1}: ${nodeError.id}`)
      }
      if (nodeError.upstream) {
        summaryItems.push(`流量池 #${poolIndex + 1} 节点 #${nodeIndex + 1}: ${nodeError.upstream}`)
      }
      if (nodeError.weight) {
        summaryItems.push(`流量池 #${poolIndex + 1} 节点 #${nodeIndex + 1}: ${nodeError.weight}`)
      }
    })
  })
  routeErrors.forEach((routeError, index) => {
    if (routeError.domain) {
      summaryItems.push(`路由 #${index + 1}: ${routeError.domain}`)
    }
    if (routeError.poolId) {
      summaryItems.push(`路由 #${index + 1}: ${routeError.poolId}`)
    }
    if (routeError.fallbackPoolId) {
      summaryItems.push(`路由 #${index + 1}: ${routeError.fallbackPoolId}`)
    }
  })

  return (
    <div className="space-y-6">
      {summaryItems.length > 0 && (
        <div className="bg-destructive/10 border border-destructive/20 text-destructive p-4 rounded-md flex items-start gap-3">
          <FiAlertCircle className="mt-0.5 shrink-0" size={18} />
          <div>
            <div className="font-semibold mb-2">当前配置存在以下问题：</div>
            <ul className="list-disc pl-5 space-y-1 text-sm">
              {summaryItems.map((item, index) => (
                <li key={`${item}-${index}`}>{item}</li>
              ))}
            </ul>
          </div>
        </div>
      )}

      <Card>
        <CardHeader>
          <CardTitle>默认流量池</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="max-w-md">
            <Select
              label="Default Pool ID"
              options={[
                { value: "", label: "不设置默认池" },
                ...poolSelectOptions
              ]}
              value={config.loadBalancer?.defaultPoolId || ''}
              onChange={(e) => updateConfig(prev => ({
                ...prev,
                loadBalancer: {
                  ...prev.loadBalancer,
                  defaultPoolId: e.target.value,
                },
              }))}
              error={defaultPoolError}
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>流量池配置</CardTitle>
          <Button onClick={handleAddPool} icon={<FiPlus />}>
            添加流量池
          </Button>
        </CardHeader>
        <CardContent>
          <div className="border rounded-md divide-y">
            {pools.length === 0 ? (
              <div className="p-8 text-center text-muted-foreground">暂无流量池配置</div>
            ) : (
              pools.map((pool, index) => {
                const draft = readNodeDraft(pool, index)
                const poolKey = buildPoolDraftKey(pool, index)
                const expanded = Boolean(expandedPoolKeys[poolKey])
                const nodes = pool.nodes || []
                const nodeErrors = nodeErrorsByPool[index] || validateNodes(nodes)
                const poolError = poolErrors[index]
                const draftNodeID = (draft.id || '').trim()
                const draftUpstream = (draft.upstream || '').trim()
                const draftWeight = Number(draft.weight ?? 0)
                
                const draftErrors = {
                  id: '',
                  upstream: '',
                  weight: '',
                }
                if (draftNodeID) {
                  const duplicated = nodes.some(node => (node.id || '').trim() === draftNodeID)
                  if (duplicated) {
                    draftErrors.id = '节点ID重复'
                  }
                }
                if (draftUpstream && !isURLValid(draftUpstream)) {
                  draftErrors.upstream = '上游地址格式无效'
                }
                if (draft.weight !== undefined && (draftWeight <= 0 || Number.isNaN(draftWeight))) {
                  draftErrors.weight = '权重必须大于0'
                }
                
                const canAddNode = Boolean(
                  draftNodeID &&
                  draftUpstream &&
                  !draftErrors.id &&
                  !draftErrors.upstream &&
                  !draftErrors.weight,
                )

                return (
                  <div key={`group-${poolKey}-${index}`} className="bg-card">
                    <div className="p-4 flex items-center gap-4">
                      <div className="flex-1 grid grid-cols-12 gap-4 items-center">
                        <div className="col-span-3">
                          <Input
                            placeholder="池ID"
                            value={pool.id || ''}
                            onChange={e => writePool(index, currentPool => ({
                              ...currentPool,
                              id: e.target.value,
                            }))}
                            error={poolError?.id}
                          />
                        </div>
                        <div className="col-span-3">
                          <Input
                            placeholder="池名称"
                            value={pool.name || ''}
                            onChange={e => writePool(index, currentPool => ({
                              ...currentPool,
                              name: e.target.value,
                            }))}
                          />
                        </div>
                        <div className="col-span-3">
                          <Select
                            options={[{ value: "weighted_rr", label: "weighted_rr" }]}
                            value={pool.strategy || 'weighted_rr'}
                            onChange={e => writePool(index, currentPool => ({
                              ...currentPool,
                              strategy: e.target.value,
                            }))}
                          />
                        </div>
                        <div className="col-span-3 flex items-center justify-end gap-2">
                        <Badge variant="default">{pool.nodes?.length || 0} 节点</Badge>
                        <Button
                            variant="secondary"
                            size="sm"
                            onClick={() => togglePoolExpanded(pool, index)}
                            icon={expanded ? <FiChevronDown /> : <FiChevronRight />}
                          >
                            {expanded ? '收起' : '展开'}
                          </Button>
                          <Button
                            variant="danger"
                            size="sm"
                            onClick={() => removePoolWithState(pool, index)}
                            icon={<FiTrash2 />}
                          />
                        </div>
                      </div>
                    </div>
                    
                    {expanded && (
                      <div className="bg-muted/30 p-4 border-t">
                        <div className="mb-4 text-sm font-medium text-muted-foreground">节点列表</div>
                        <div className="space-y-2">
                          {nodes.map((node, nodeIndex) => (
                            <div key={`${buildPoolDraftKey(pool, index)}-${nodeIndex}`} className="grid grid-cols-12 gap-2 items-start">
                              <div className="col-span-3">
                                <Input
                                  placeholder="节点ID"
                                  value={node.id || ''}
                                  onChange={e => writePool(index, currentPool => {
                                    const nextNodes = [...(currentPool.nodes || [])]
                                    nextNodes[nodeIndex] = { ...node, id: e.target.value }
                                    return { ...currentPool, nodes: nextNodes }
                                  })}
                                  error={nodeErrors[nodeIndex]?.id}
                                />
                              </div>
                              <div className="col-span-5">
                                <Input
                                  placeholder="上游地址 (http://...)"
                                  value={node.upstream || ''}
                                  onChange={e => writePool(index, currentPool => {
                                    const nextNodes = [...(currentPool.nodes || [])]
                                    nextNodes[nodeIndex] = { ...node, upstream: e.target.value }
                                    return { ...currentPool, nodes: nextNodes }
                                  })}
                                  error={nodeErrors[nodeIndex]?.upstream}
                                />
                              </div>
                              <div className="col-span-2">
                                <Input
                                  type="number"
                                  placeholder="权重"
                                  min={1}
                                  value={node.weight ?? 1}
                                  onChange={e => writePool(index, currentPool => {
                                    const nextNodes = [...(currentPool.nodes || [])]
                                    nextNodes[nodeIndex] = {
                                      ...node,
                                      weight: Number(e.target.value) > 0 ? Number(e.target.value) : 1,
                                    }
                                    return { ...currentPool, nodes: nextNodes }
                                  })}
                                  error={nodeErrors[nodeIndex]?.weight}
                                />
                              </div>
                              <div className="col-span-1">
                                <Select
                                  options={[
                                    { value: "true", label: "是" },
                                    { value: "false", label: "否" }
                                  ]}
                                  value={node.enabled === false ? 'false' : 'true'}
                                  onChange={e => writePool(index, currentPool => {
                                    const nextNodes = [...(currentPool.nodes || [])]
                                    nextNodes[nodeIndex] = {
                                      ...node,
                                      enabled: e.target.value === 'true',
                                    }
                                    return { ...currentPool, nodes: nextNodes }
                                  })}
                                />
                              </div>
                              <div className="col-span-1 flex justify-end">
                                <Button
                                  variant="danger"
                                  size="sm"
                                  onClick={() => {
                                    writePool(index, currentPool => ({
                                      ...currentPool,
                                      nodes: (currentPool.nodes || []).filter((_, i) => i !== nodeIndex),
                                    }))
                                  }}
                                  icon={<FiTrash2 />}
                                />
                              </div>
                            </div>
                          ))}
                          
                          {/* Add New Node Row */}
                          <div className="grid grid-cols-12 gap-2 items-start mt-4 pt-4 border-t border-dashed">
                            <div className="col-span-3">
                              <Input
                                placeholder="新增节点ID"
                                value={draft.id || ''}
                                onChange={e => writeNodeDraft(pool, index, prev => ({ ...prev, id: e.target.value }))}
                                error={draftErrors.id}
                              />
                            </div>
                            <div className="col-span-5">
                              <Input
                                placeholder="上游地址"
                                value={draft.upstream || ''}
                                onChange={e => writeNodeDraft(pool, index, prev => ({ ...prev, upstream: e.target.value }))}
                                error={draftErrors.upstream}
                              />
                            </div>
                            <div className="col-span-2">
                              <Input
                                type="number"
                                placeholder="权重"
                                min={1}
                                value={draft.weight ?? 1}
                                onChange={e => writeNodeDraft(pool, index, prev => ({ ...prev, weight: Number(e.target.value) }))}
                                error={draftErrors.weight}
                              />
                            </div>
                            <div className="col-span-2 flex justify-end">
                              <Button
                                disabled={!canAddNode}
                                onClick={() => addPoolNode(index)}
                                icon={<FiPlus />}
                              >
                                添加节点
                              </Button>
                            </div>
                          </div>
                        </div>
                      </div>
                    )}
                  </div>
                )
              })
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
