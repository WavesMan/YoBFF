import { FiTrash2 } from 'react-icons/fi'
import type { Config, LBPool, LBRouteRule } from '../../../../admin/types'

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

  return (
    <div className="panel">
      <h4>默认流量池</h4>
      <div className="form-field">
        <label>Default Pool ID</label>
        <input
          className="input"
          value={config.loadBalancer?.defaultPoolId || ''}
          onChange={(e) => updateConfig(prev => ({
            ...prev,
            loadBalancer: {
              ...prev.loadBalancer,
              defaultPoolId: e.target.value,
            },
          }))}
          placeholder="pool_default"
        />
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
          {pools.map((pool, index) => (
            <tr key={index}>
              <td>
                <input
                  className="input small"
                  value={pool.id || ''}
                  onChange={e => {
                    const next = [...pools]
                    next[index] = { ...pool, id: e.target.value }
                    updateConfig(prev => ({
                      ...prev,
                      loadBalancer: { ...prev.loadBalancer, pools: next },
                    }))
                  }}
                />
              </td>
              <td>
                <input
                  className="input small"
                  value={pool.name || ''}
                  onChange={e => {
                    const next = [...pools]
                    next[index] = { ...pool, name: e.target.value }
                    updateConfig(prev => ({
                      ...prev,
                      loadBalancer: { ...prev.loadBalancer, pools: next },
                    }))
                  }}
                />
              </td>
              <td>
                <select
                  className="select small"
                  value={pool.strategy || 'weighted_rr'}
                  onChange={e => {
                    const next = [...pools]
                    next[index] = { ...pool, strategy: e.target.value }
                    updateConfig(prev => ({
                      ...prev,
                      loadBalancer: { ...prev.loadBalancer, pools: next },
                    }))
                  }}
                >
                  <option value="weighted_rr">weighted_rr</option>
                </select>
              </td>
              <td>{pool.nodes?.length || 0}</td>
              <td style={{ textAlign: 'right' }}>
                <button className="button danger small" onClick={() => handleRemovePool(index)}>
                  <FiTrash2 />
                </button>
              </td>
            </tr>
          ))}
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
          {routes.map((rule, index) => (
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
              </td>
              <td>
                <input
                  className="input small"
                  value={rule.poolId || ''}
                  onChange={e => {
                    const next = [...routes]
                    next[index] = { ...rule, poolId: e.target.value }
                    updateConfig(prev => ({
                      ...prev,
                      loadBalancer: { ...prev.loadBalancer, routes: next },
                    }))
                  }}
                />
              </td>
              <td>
                <input
                  className="input small"
                  value={rule.fallbackPoolId || ''}
                  onChange={e => {
                    const next = [...routes]
                    next[index] = { ...rule, fallbackPoolId: e.target.value }
                    updateConfig(prev => ({
                      ...prev,
                      loadBalancer: { ...prev.loadBalancer, routes: next },
                    }))
                  }}
                />
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
          ))}
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
        </div>
        <div className="form-field">
          <input
            className="input"
            placeholder="绑定主池ID"
            value={newRoute.poolId || ''}
            onChange={e => setNewRoute({ ...newRoute, poolId: e.target.value })}
          />
        </div>
        <button className="button secondary" onClick={handleAddRoute}>添加绑定</button>
      </div>
    </div>
  )
}
