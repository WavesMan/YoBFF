import { Fragment, useState } from 'react'
import { FiChevronDown, FiChevronRight, FiTrash2 } from 'react-icons/fi'
import type { Config, LBNode, LBPool, LBRouteRule } from '../../../../admin/types'

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
  newPool,
  setNewPool,
  newRoute,
  setNewRoute,
  handleAddPool,
  handleRemovePool,
  handleAddRoute,
  handleRemoveRoute,
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
        id,
        label: pool.name ? `${pool.name} (${id})` : id,
        key: `${id}-${index}`,
      }
    })
    .filter((item): item is { id: string; label: string; key: string } => Boolean(item))

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
  const newRouteDomain = (newRoute.domain || '').trim()
  const newRoutePoolID = (newRoute.poolId || '').trim()
  const newRouteFallbackPoolID = (newRoute.fallbackPoolId || '').trim()
  const newRouteErrors = {
    domain: !newRouteDomain ? '域名不能为空' : '',
    poolId: !newRoutePoolID
      ? '主池不能为空'
      : (poolIDSet.has(newRoutePoolID) ? '' : '主池不存在'),
    fallbackPoolId: !newRouteFallbackPoolID
      ? ''
      : (poolIDSet.has(newRouteFallbackPoolID) ? '' : '回退池不存在'),
  }
  const newRouteTouched = Boolean(newRouteDomain || newRoutePoolID || newRouteFallbackPoolID)
  const canAddRoute = Boolean(
    newRouteDomain &&
    newRoutePoolID &&
    !newRouteErrors.domain &&
    !newRouteErrors.poolId &&
    !newRouteErrors.fallbackPoolId,
  )
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
    <div className="panel">
      {summaryItems.length > 0 && (
        <div className="alert error" style={{ marginBottom: 20 }}>
          <div style={{ marginBottom: 8, fontWeight: 600 }}>当前配置存在以下问题：</div>
          {summaryItems.map((item, index) => (
            <div key={`${item}-${index}`}>- {item}</div>
          ))}
        </div>
      )}
      <h4>默认流量池</h4>
      <div className="form-field">
        <label>Default Pool ID</label>
        <select
          className="select"
          value={config.loadBalancer?.defaultPoolId || ''}
          onChange={(e) => updateConfig(prev => ({
            ...prev,
            loadBalancer: {
              ...prev.loadBalancer,
              defaultPoolId: e.target.value,
            },
          }))}
        >
          <option value="">不设置默认池</option>
          {poolSelectOptions.map(item => (
            <option key={item.key} value={item.id}>
              {item.label}
            </option>
          ))}
        </select>
        {defaultPoolError && (
          <div style={{ marginTop: 6, color: '#ef4444', fontSize: 12 }}>
            {defaultPoolError}
          </div>
        )}
      </div>

      <div className="divider" />

      <h4>流量池</h4>
      <table className="table">
        <thead>
          <tr>
            <th>池ID</th>
            <th>池名称</th>
            <th>策略</th>
            <th>节点数</th>
            <th style={{ textAlign: 'right' }}>操作</th>
          </tr>
        </thead>
        <tbody>
          {pools.map((pool, index) => {
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
              <Fragment key={`group-${poolKey}-${index}`}>
                <tr>
              <td>
                <input
                  className="input small"
                  value={pool.id || ''}
                  onChange={e => {
                    writePool(index, currentPool => ({
                      ...currentPool,
                      id: e.target.value,
                    }))
                  }}
                />
                {poolError?.id && (
                  <div style={{ marginTop: 6, color: '#ef4444', fontSize: 12 }}>
                    {poolError.id}
                  </div>
                )}
              </td>
              <td>
                <input
                  className="input small"
                  value={pool.name || ''}
                  onChange={e => {
                    writePool(index, currentPool => ({
                      ...currentPool,
                      name: e.target.value,
                    }))
                  }}
                />
              </td>
              <td>
                <select
                  className="select small"
                  value={pool.strategy || 'weighted_rr'}
                  onChange={e => {
                    writePool(index, currentPool => ({
                      ...currentPool,
                      strategy: e.target.value,
                    }))
                  }}
                >
                  <option value="weighted_rr">weighted_rr</option>
                </select>
              </td>
              <td>
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 8 }}>
                  <span>{pool.nodes?.length || 0}</span>
                  <button
                    className="button secondary small"
                    onClick={() => togglePoolExpanded(pool, index)}
                  >
                    {expanded ? <FiChevronDown /> : <FiChevronRight />}
                    {expanded ? '收起' : '展开'}
                  </button>
                </div>
              </td>
              <td style={{ textAlign: 'right' }}>
                <button className="button danger small" onClick={() => removePoolWithState(pool, index)}>
                  <FiTrash2 />
                </button>
              </td>
            </tr>
                {expanded && (
                <tr>
                <td colSpan={5} style={{ background: 'var(--bg-page)' }}>
                  <div className="muted" style={{ marginBottom: 10 }}>
                    节点配置
                  </div>
                  <table className="table">
                    <thead>
                      <tr>
                        <th style={{ width: '20%' }}>节点ID</th>
                        <th style={{ width: '40%' }}>上游地址</th>
                        <th style={{ width: '12%' }}>权重</th>
                        <th style={{ width: '12%' }}>启用</th>
                        <th style={{ textAlign: 'right' }}>操作</th>
                      </tr>
                    </thead>
                    <tbody>
                      {nodes.map((node, nodeIndex) => (
                        <tr key={`${buildPoolDraftKey(pool, index)}-${nodeIndex}`}>
                          <td>
                            <input
                              className="input small"
                              value={node.id || ''}
                              onChange={e => {
                                writePool(index, currentPool => {
                                  const nextNodes = [...(currentPool.nodes || [])]
                                  nextNodes[nodeIndex] = { ...node, id: e.target.value }
                                  return { ...currentPool, nodes: nextNodes }
                                })
                              }}
                            />
                            {nodeErrors[nodeIndex]?.id && (
                              <div style={{ marginTop: 6, color: '#ef4444', fontSize: 12 }}>
                                {nodeErrors[nodeIndex].id}
                              </div>
                            )}
                          </td>
                          <td>
                            <input
                              className="input small"
                              value={node.upstream || ''}
                              onChange={e => {
                                writePool(index, currentPool => {
                                  const nextNodes = [...(currentPool.nodes || [])]
                                  nextNodes[nodeIndex] = { ...node, upstream: e.target.value }
                                  return { ...currentPool, nodes: nextNodes }
                                })
                              }}
                            />
                            {nodeErrors[nodeIndex]?.upstream && (
                              <div style={{ marginTop: 6, color: '#ef4444', fontSize: 12 }}>
                                {nodeErrors[nodeIndex].upstream}
                              </div>
                            )}
                          </td>
                          <td>
                            <input
                              className="input small"
                              type="number"
                              min={1}
                              value={node.weight ?? 1}
                              onChange={e => {
                                writePool(index, currentPool => {
                                  const nextNodes = [...(currentPool.nodes || [])]
                                  nextNodes[nodeIndex] = {
                                    ...node,
                                    weight: Number(e.target.value) > 0 ? Number(e.target.value) : 1,
                                  }
                                  return { ...currentPool, nodes: nextNodes }
                                })
                              }}
                            />
                            {nodeErrors[nodeIndex]?.weight && (
                              <div style={{ marginTop: 6, color: '#ef4444', fontSize: 12 }}>
                                {nodeErrors[nodeIndex].weight}
                              </div>
                            )}
                          </td>
                          <td>
                            <select
                              className="select small"
                              value={node.enabled === false ? 'false' : 'true'}
                              onChange={e => {
                                writePool(index, currentPool => {
                                  const nextNodes = [...(currentPool.nodes || [])]
                                  nextNodes[nodeIndex] = {
                                    ...node,
                                    enabled: e.target.value === 'true',
                                  }
                                  return { ...currentPool, nodes: nextNodes }
                                })
                              }}
                            >
                              <option value="true">是</option>
                              <option value="false">否</option>
                            </select>
                          </td>
                          <td style={{ textAlign: 'right' }}>
                            <button
                              className="button danger small"
                              onClick={() => {
                                writePool(index, currentPool => ({
                                  ...currentPool,
                                  nodes: (currentPool.nodes || []).filter((_, i) => i !== nodeIndex),
                                }))
                              }}
                            >
                              <FiTrash2 />
                            </button>
                          </td>
                        </tr>
                      ))}
                      <tr>
                        <td>
                          <input
                            className="input small"
                            placeholder="新增节点ID"
                            value={draft.id || ''}
                            onChange={e => writeNodeDraft(pool, index, prev => ({ ...prev, id: e.target.value }))}
                          />
                          {draftErrors.id && (
                            <div style={{ marginTop: 6, color: '#ef4444', fontSize: 12 }}>
                              {draftErrors.id}
                            </div>
                          )}
                        </td>
                        <td>
                          <input
                            className="input small"
                            placeholder="http://127.0.0.1:8080"
                            value={draft.upstream || ''}
                            onChange={e => writeNodeDraft(pool, index, prev => ({ ...prev, upstream: e.target.value }))}
                          />
                          {draftErrors.upstream && (
                            <div style={{ marginTop: 6, color: '#ef4444', fontSize: 12 }}>
                              {draftErrors.upstream}
                            </div>
                          )}
                        </td>
                        <td>
                          <input
                            className="input small"
                            type="number"
                            min={1}
                            value={draft.weight ?? 1}
                            onChange={e => {
                              const weight = Number(e.target.value)
                              writeNodeDraft(pool, index, prev => ({
                                ...prev,
                                weight: weight > 0 ? weight : 1,
                              }))
                            }}
                          />
                          {draftErrors.weight && (
                            <div style={{ marginTop: 6, color: '#ef4444', fontSize: 12 }}>
                              {draftErrors.weight}
                            </div>
                          )}
                        </td>
                        <td>
                          <select
                            className="select small"
                            value={draft.enabled === false ? 'false' : 'true'}
                            onChange={e => writeNodeDraft(pool, index, prev => ({
                              ...prev,
                              enabled: e.target.value === 'true',
                            }))}
                          >
                            <option value="true">是</option>
                            <option value="false">否</option>
                          </select>
                        </td>
                        <td style={{ textAlign: 'right' }}>
                          <button className="button secondary small" onClick={() => addPoolNode(index)} disabled={!canAddNode}>
                            添加节点
                          </button>
                        </td>
                      </tr>
                      {nodes.length === 0 && (
                        <tr>
                          <td colSpan={5} className="muted" style={{ textAlign: 'center' }}>
                            当前池暂无节点，至少添加一个可用节点后再保存配置
                          </td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </td>
              </tr>
                )}
              </Fragment>
            )
          })}
        </tbody>
      </table>
      
      <div className="form-row" style={{ marginTop: 15 }}>
        <div className="form-field">
          <input
            className="input"
            placeholder="新增池ID"
            value={newPool.id || ''}
            onChange={e => setNewPool({ ...newPool, id: e.target.value })}
          />
        </div>
        <div className="form-field">
          <input
            className="input"
            placeholder="新增池名称"
            value={newPool.name || ''}
            onChange={e => setNewPool({ ...newPool, name: e.target.value })}
          />
        </div>
        <button className="button secondary" onClick={handleAddPool}>添加流量池</button>
      </div>

      <div className="divider" />

      <h4>域名绑定规则</h4>
      <table className="table">
        <thead>
          <tr>
            <th>域名</th>
            <th>主池</th>
            <th>回退池</th>
            <th>HTTPS</th>
            <th style={{ textAlign: 'right' }}>操作</th>
          </tr>
        </thead>
        <tbody>
          {routes.map((rule, index) => {
            const routeError = routeErrors[index]
            return (
            <tr key={index}>
              <td>
                <input
                  className="input small"
                  value={rule.domain || ''}
                  onChange={e => {
                    const next = [...routes]
                    next[index] = { ...rule, domain: e.target.value }
                    updateConfig(prev => ({
                      ...prev,
                      loadBalancer: { ...prev.loadBalancer, routes: next },
                    }))
                  }}
                />
                {routeError?.domain && (
                  <div style={{ marginTop: 6, color: '#ef4444', fontSize: 12 }}>
                    {routeError.domain}
                  </div>
                )}
              </td>
              <td>
                <select
                  className="select small"
                  value={rule.poolId || ''}
                  onChange={e => {
                    const next = [...routes]
                    next[index] = { ...rule, poolId: e.target.value }
                    updateConfig(prev => ({
                      ...prev,
                      loadBalancer: { ...prev.loadBalancer, routes: next },
                    }))
                  }}
                >
                  <option value="">选择主池</option>
                  {poolSelectOptions.map(item => (
                    <option key={`main-${item.key}`} value={item.id}>
                      {item.label}
                    </option>
                  ))}
                </select>
                {routeError?.poolId && (
                  <div style={{ marginTop: 6, color: '#ef4444', fontSize: 12 }}>
                    {routeError.poolId}
                  </div>
                )}
              </td>
              <td>
                <select
                  className="select small"
                  value={rule.fallbackPoolId || ''}
                  onChange={e => {
                    const next = [...routes]
                    next[index] = { ...rule, fallbackPoolId: e.target.value }
                    updateConfig(prev => ({
                      ...prev,
                      loadBalancer: { ...prev.loadBalancer, routes: next },
                    }))
                  }}
                >
                  <option value="">不设置</option>
                  {poolSelectOptions.map(item => (
                    <option key={`fallback-${item.key}`} value={item.id}>
                      {item.label}
                    </option>
                  ))}
                </select>
                {routeError?.fallbackPoolId && (
                  <div style={{ marginTop: 6, color: '#ef4444', fontSize: 12 }}>
                    {routeError.fallbackPoolId}
                  </div>
                )}
              </td>
              <td>
                <select
                  className="select small"
                  value={rule.forceHttps ? 'true' : 'false'}
                  onChange={e => {
                    const next = [...routes]
                    next[index] = { ...rule, forceHttps: e.target.value === 'true' }
                    updateConfig(prev => ({
                      ...prev,
                      loadBalancer: { ...prev.loadBalancer, routes: next },
                    }))
                  }}
                >
                  <option value="true">开启</option>
                  <option value="false">关闭</option>
                </select>
              </td>
              <td style={{ textAlign: 'right' }}>
                <button className="button danger small" onClick={() => handleRemoveRoute(index)}>
                  <FiTrash2 />
                </button>
              </td>
            </tr>
            )
          })}
        </tbody>
      </table>

      <div className="form-row" style={{ marginTop: 15 }}>
        <div className="form-field">
          <input
            className="input"
            placeholder="新增域名"
            value={newRoute.domain || ''}
            onChange={e => setNewRoute({ ...newRoute, domain: e.target.value })}
          />
          {newRouteTouched && newRouteErrors.domain && (
            <div style={{ marginTop: 6, color: '#ef4444', fontSize: 12 }}>
              {newRouteErrors.domain}
            </div>
          )}
        </div>
        <div className="form-field">
          <select
            className="select"
            value={newRoute.poolId || ''}
            onChange={e => setNewRoute({ ...newRoute, poolId: e.target.value })}
          >
            <option value="">选择主池</option>
            {poolSelectOptions.map(item => (
              <option key={`new-main-${item.key}`} value={item.id}>
                {item.label}
              </option>
            ))}
          </select>
          {newRouteTouched && newRouteErrors.poolId && (
            <div style={{ marginTop: 6, color: '#ef4444', fontSize: 12 }}>
              {newRouteErrors.poolId}
            </div>
          )}
        </div>
        <div className="form-field">
          <select
            className="select"
            value={newRoute.fallbackPoolId || ''}
            onChange={e => setNewRoute({ ...newRoute, fallbackPoolId: e.target.value })}
          >
            <option value="">不设置回退池</option>
            {poolSelectOptions.map(item => (
              <option key={`new-fallback-${item.key}`} value={item.id}>
                {item.label}
              </option>
            ))}
          </select>
          {newRouteTouched && newRouteErrors.fallbackPoolId && (
            <div style={{ marginTop: 6, color: '#ef4444', fontSize: 12 }}>
              {newRouteErrors.fallbackPoolId}
            </div>
          )}
        </div>
        <button className="button secondary" onClick={handleAddRoute} disabled={!canAddRoute}>添加绑定</button>
      </div>
    </div>
  )
}
