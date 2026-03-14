import { useState } from 'react'
import { FiChevronDown, FiChevronRight, FiTrash2, FiPlus, FiAlertCircle } from 'react-icons/fi'
import type { Config, LBNode, LBPool, LBRouteRule } from '../../../../admin/types'
import { Button } from '../../../../components/ui/Button'
import { Input } from '../../../../components/ui/Input'
import { Select } from '../../../../components/ui/Select'
import { Switch } from '../../../../components/ui/Switch'
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

  const removePoolWithState = (index: number) => {
    setExpandedPoolKeys(prev => {
      const next: Record<string, boolean> = {}
      Object.keys(prev).forEach(key => {
        const parts = key.split('_')
        if (parts.length !== 2 || parts[0] !== 'pool') {
          next[key] = prev[key]
          return
        }
        const idx = parseInt(parts[1], 10)
        if (Number.isNaN(idx)) {
          next[key] = prev[key]
          return
        }

        if (idx < index) {
          next[key] = prev[key]
        } else if (idx > index) {
          next[`pool_${idx - 1}`] = prev[key]
        }
        // idx == index is dropped
      })
      return next
    })

    setPoolNodeDrafts(prev => {
      const next: Record<string, LBNode> = {}
      Object.keys(prev).forEach(key => {
        const parts = key.split('_')
        if (parts.length !== 2 || parts[0] !== 'pool') {
          next[key] = prev[key]
          return
        }
        const idx = parseInt(parts[1], 10)
        if (Number.isNaN(idx)) {
          next[key] = prev[key]
          return
        }

        if (idx < index) {
          next[key] = prev[key]
        } else if (idx > index) {
          next[`pool_${idx - 1}`] = prev[key]
        }
      })
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

      <div className="space-y-4">
        <h3 className="text-lg font-medium">默认流量池</h3>
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
            layout="horizontal"
          />
        </div>
      </div>

      <div className="border-t my-6" />

      <div className="space-y-4">
        <div className="flex flex-row items-center justify-between">
          <h3 className="text-lg font-medium">流量池配置</h3>
          <Button onClick={handleAddPool} icon={<FiPlus />}>
            添加流量池
          </Button>
        </div>
        
        <div className="space-y-4">
          {pools.length === 0 ? (
            <div className="p-8 text-center text-muted-foreground border rounded-md border-dashed">暂无流量池配置</div>
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
                  <div key={`group-${poolKey}-${index}`} className="border rounded-md bg-card overflow-hidden">
                    <div className="flex items-center justify-between p-3 border-b bg-muted/20">
                      <div className="flex items-center gap-3">
                        <span className="font-medium text-sm">流量池 #{index + 1}</span>
                        <Badge variant="default" className="font-normal text-muted-foreground">{pool.nodes?.length || 0} 节点</Badge>
                      </div>
                      <div className="flex items-center gap-2">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => togglePoolExpanded(pool, index)}
                          icon={expanded ? <FiChevronDown /> : <FiChevronRight />}
                        >
                          {expanded ? '收起配置' : '展开配置'}
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="text-destructive hover:text-destructive hover:bg-destructive/10"
                          onClick={() => removePoolWithState(index)}
                          icon={<FiTrash2 />}
                        />
                      </div>
                    </div>

                    <div className="p-4 space-y-4">
                      <Input
                        label="池 ID (Pool ID)"
                        placeholder="e.g. pool_01"
                        value={pool.id || ''}
                        onChange={e => writePool(index, currentPool => ({
                          ...currentPool,
                          id: e.target.value,
                        }))}
                        error={poolError?.id}
                        layout="horizontal"
                      />
                      <Input
                        label="池名称 (Name)"
                        placeholder="e.g. Primary Backend"
                        value={pool.name || ''}
                        onChange={e => writePool(index, currentPool => ({
                          ...currentPool,
                          name: e.target.value,
                        }))}
                        layout="horizontal"
                      />
                      <Select
                        label="负载均衡策略"
                        options={[{ value: "weighted_rr", label: "加权轮询 (Weighted Round Robin)" }]}
                        value={pool.strategy || 'weighted_rr'}
                        onChange={e => writePool(index, currentPool => ({
                          ...currentPool,
                          strategy: e.target.value,
                        }))}
                        layout="horizontal"
                      />
                      
                    {expanded && (
                        <div className="pt-4 border-t animate-in fade-in slide-in-from-top-2 duration-200">
                          <div className="mb-4 flex items-center justify-between">
                            <span className="text-sm font-medium text-muted-foreground">节点列表</span>
                          </div>
                          
                          <div className="space-y-3">
                            <div className="grid grid-cols-12 gap-2 px-2 text-xs font-medium text-muted-foreground mb-2">
                              <div className="col-span-3">节点 ID</div>
                              <div className="col-span-5">上游地址</div>
                              <div className="col-span-2">权重</div>
                              <div className="col-span-1 text-center">启用</div>
                              <div className="col-span-1"></div>
                            </div>

                            {nodes.map((node, nodeIndex) => (
                              <div key={`${buildPoolDraftKey(pool, index)}-${nodeIndex}`} className="grid grid-cols-12 gap-2 items-center">
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
                                    className="h-9"
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
                                    className="h-9"
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
                                    className="h-9"
                                  />
                                </div>
                                <div className="col-span-1 flex justify-center">
                                  <Switch
                                    checked={node.enabled !== false}
                                    onCheckedChange={checked => writePool(index, currentPool => {
                                      const nextNodes = [...(currentPool.nodes || [])]
                                      nextNodes[nodeIndex] = {
                                        ...node,
                                        enabled: checked,
                                      }
                                      return { ...currentPool, nodes: nextNodes }
                                    })}
                                  />
                                </div>
                                <div className="col-span-1 flex justify-end">
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    className="text-muted-foreground hover:text-destructive h-9 w-9 p-0"
                                    onClick={() => {
                                      writePool(index, currentPool => ({
                                        ...currentPool,
                                        nodes: (currentPool.nodes || []).filter((_, i) => i !== nodeIndex),
                                      }))
                                    }}
                                    icon={<FiTrash2 size={14} />}
                                  />
                                </div>
                              </div>
                            ))}
                            
                            {/* Add New Node Row */}
                            <div className="grid grid-cols-12 gap-2 items-center pt-2 border-t border-dashed mt-2">
                              <div className="col-span-3">
                                <Input
                                  placeholder="新增节点ID"
                                  value={draft.id || ''}
                                  onChange={e => writeNodeDraft(pool, index, prev => ({ ...prev, id: e.target.value }))}
                                  error={draftErrors.id}
                                  className="h-9"
                                />
                              </div>
                              <div className="col-span-5">
                                <Input
                                  placeholder="上游地址"
                                  value={draft.upstream || ''}
                                  onChange={e => writeNodeDraft(pool, index, prev => ({ ...prev, upstream: e.target.value }))}
                                  error={draftErrors.upstream}
                                  className="h-9"
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
                                  className="h-9"
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
                  </div>
                )
              })
            )}
          </div>
        </div>
    </div>
  )
}
