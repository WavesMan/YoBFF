import { useCallback, useEffect, useState } from 'react'
import { FiGlobe, FiServer, FiSettings, FiTrash2, FiX } from 'react-icons/fi'
import { createSite, deleteSite, fetchSiteGroups } from '../../admin/api'
import type { Site, SiteCreateRequest, SiteGroup } from '../../admin/types'

type SiteListProps = {
  token: string
  onSelectSite: (siteId: string) => void
}

export function SiteList({ token, onSelectSite }: SiteListProps) {
  const [grouping, setGrouping] = useState<'hostname' | 'ip'>('hostname')
  const [groups, setGroups] = useState<SiteGroup[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<Site | null>(null)
  const [deleting, setDeleting] = useState(false)

  const loadSites = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const data = await fetchSiteGroups(token, grouping)
      setGroups(data.groups || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载站点列表失败')
    } finally {
      setLoading(false)
    }
  }, [token, grouping])

  useEffect(() => {
    loadSites()
  }, [loadSites])

  const handleDelete = async () => {
    if (!deleteTarget) return
    setDeleting(true)
    try {
      await deleteSite(token, deleteTarget.id)
      setDeleteTarget(null)
      loadSites()
    } catch (err) {
      setError(err instanceof Error ? err.message : '删除失败')
    } finally {
      setDeleting(false)
    }
  }

  return (
    <div className="card">
      <div className="card-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h3 className="card-title">站点列表</h3>
        <div className="actions">
          <select
            className="select"
            value={grouping}
            onChange={(e) => setGrouping(e.target.value as 'hostname' | 'ip')}
            style={{ width: 'auto' }}
          >
            <option value="hostname">按域名分组</option>
            <option value="ip">按 IP 分组</option>
          </select>
          <button className="button primary" onClick={() => setShowCreateModal(true)}>
            <svg stroke="currentColor" fill="none" strokeWidth="2" viewBox="0 0 24 24" strokeLinecap="round" strokeLinejoin="round" height="1em" width="1em" xmlns="http://www.w3.org/2000/svg">
              <line x1="12" y1="5" x2="12" y2="19"></line>
              <line x1="5" y1="12" x2="19" y2="12"></line>
            </svg>
            添加站点
          </button>
        </div>
      </div>
      
      {error && <div className="error-message">{error}</div>}
      
      <div className="site-list-content">
        {loading ? (
          <div className="loading">加载中...</div>
        ) : groups.length > 0 ? (
          groups.map((group) => (
            <div key={group.group_key} className="site-group">
              <div className="group-header">
                <span className="group-title">
                  {grouping === 'hostname' ? <FiGlobe /> : <FiServer />}
                  {group.group_key || '未分类'}
                </span>
                <span className="site-badge">{group.count}</span>
              </div>
              <table className="table">
                <thead>
                  <tr>
                    <th>站点名称</th>
                    <th>域名</th>
                    <th>IP</th>
                    <th style={{ textAlign: 'right' }}>操作</th>
                  </tr>
                </thead>
                <tbody>
                  {group.sites.map((site) => (
                    <tr key={site.id}>
                      <td>{site.name}</td>
                      <td>{site.hostname}</td>
                      <td>{site.ip}</td>
                      <td style={{ textAlign: 'right' }}>
                        <div className="action-buttons">
                          <button
                            className="button secondary"
                            onClick={() => onSelectSite(site.id)}
                            title="配置"
                            style={{ padding: '6px 16px' }}
                          >
                            <FiSettings style={{ marginRight: 6 }} /> 配置
                          </button>
                          <button
                            className="button small danger"
                            onClick={() => setDeleteTarget({
                              id: site.id,
                              name: site.name,
                              hostname: site.hostname,
                              ip: site.ip,
                            })}
                            title="删除"
                          >
                            <FiTrash2 />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ))
        ) : (
          <div className="empty-state">暂无站点</div>
        )}
      </div>

      {showCreateModal && (
        <CreateSiteDrawer
          token={token}
          onClose={() => setShowCreateModal(false)}
          onCreated={() => {
            setShowCreateModal(false)
            loadSites()
          }}
        />
      )}

      {deleteTarget && (
        <div className="confirm-overlay" onClick={() => !deleting && setDeleteTarget(null)}>
          <div className="confirm-dialog" role="dialog" aria-modal="true" onClick={(event) => event.stopPropagation()}>
            <div className="confirm-title">确认删除</div>
            <div className="confirm-message">
              确定要删除站点「{deleteTarget.name || deleteTarget.hostname}」吗？此操作不可恢复。
            </div>
            <div className="confirm-actions">
              <button className="button secondary" onClick={() => setDeleteTarget(null)} disabled={deleting}>
                取消
              </button>
              <button className="button danger" onClick={handleDelete} disabled={deleting}>
                {deleting ? '删除中...' : '确定删除'}
              </button>
            </div>
          </div>
        </div>
      )}
      
      <style>{`
        .site-group {
          margin-bottom: 20px;
          border: 1px solid var(--border);
          border-radius: 4px;
          overflow: hidden;
        }
        .site-list-content {
          padding: 12px 16px;
        }
        .group-header {
          background: var(--bg-hover);
          padding: 10px 15px;
          display: flex;
          align-items: center;
          gap: 10px;
          font-weight: 500;
        }
        .table th,
        .table td {
          padding: 10px 14px;
          height: 42px;
          vertical-align: middle;
        }
        .table thead th {
          padding-top: 8px;
          padding-bottom: 8px;
        }
        .group-title {
          display: flex;
          align-items: center;
          gap: 8px;
        }
        .site-badge {
          background: var(--primary);
          color: white;
          padding: 2px 8px;
          border-radius: 10px;
          font-size: 12px;
        }
        .actions {
          display: flex;
          gap: 10px;
        }
        .action-buttons {
          display: flex;
          justify-content: flex-end;
          align-items: center;
          gap: 8px;
        }
        .error-message {
          color: var(--error);
          padding: 10px;
          background: rgba(244, 67, 54, 0.1);
          border-radius: 4px;
          margin-bottom: 10px;
        }
        .icon-btn {
          background: none;
          border: none;
          color: var(--text-sub);
          cursor: pointer;
          font-size: 20px;
          padding: 5px;
        }
        .icon-btn:hover {
          color: var(--text-main);
        }
        .site-edit-overlay {
          position: fixed;
          top: 0;
          left: 0;
          width: 100%;
          height: 100%;
          background: rgba(0, 0, 0, 0.5);
          z-index: 999;
          opacity: 0;
          visibility: hidden;
          transition: all 0.3s;
        }
        .site-edit-overlay.open {
          opacity: 1;
          visibility: visible;
        }
        .site-edit-drawer {
          position: fixed;
          top: 0;
          right: -100%;
          width: 420px;
          max-width: 92vw;
          height: 100vh;
          background: var(--bg-surface);
          z-index: 1000;
          transition: right 0.3s ease;
          display: flex;
          flex-direction: column;
          box-shadow: -5px 0 15px rgba(0,0,0,0.5);
        }
        .site-edit-drawer.open {
          right: 0;
        }
        .site-edit-header {
          padding: 15px 20px;
          border-bottom: 1px solid var(--border);
          display: flex;
          justify-content: space-between;
          align-items: center;
          background: var(--bg-surface);
        }
        .site-edit-title {
          display: flex;
          align-items: center;
          gap: 12px;
        }
        .site-edit-body {
          padding: 20px;
          overflow-y: auto;
          background: var(--bg-surface);
        }
        .confirm-overlay {
          position: fixed;
          inset: 0;
          display: flex;
          align-items: center;
          justify-content: center;
          background: rgba(0, 0, 0, 0.55);
          z-index: 1100;
          padding: 24px;
        }
        .confirm-dialog {
          width: min(420px, 92vw);
          background: var(--bg-surface);
          border: 1px solid var(--border);
          border-radius: 8px;
          box-shadow: 0 20px 50px rgba(0, 0, 0, 0.55);
          padding: 20px;
          display: flex;
          flex-direction: column;
          gap: 12px;
        }
        .confirm-title {
          font-size: 16px;
          font-weight: 600;
        }
        .confirm-message {
          color: var(--text-secondary);
          line-height: 1.6;
        }
        .confirm-actions {
          display: flex;
          justify-content: flex-end;
          gap: 10px;
        }
      `}</style>
    </div>
  )
}

function CreateSiteDrawer({ token, onClose, onCreated }: { token: string, onClose: () => void, onCreated: () => void }) {
  const [form, setForm] = useState<SiteCreateRequest>({
    name: '',
    hostname: '',
    ip: '',
  })
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async () => {
    if (!form.name || !form.hostname) {
      setError('名称和域名为必填项')
      return
    }
    setLoading(true)
    setError('')
    try {
      await createSite(token, form)
      onCreated()
    } catch (err) {
      setError(err instanceof Error ? err.message : '创建站点失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <>
      <div className="site-edit-overlay open" onClick={onClose} />
      <div className="site-edit-drawer open">
        <div className="site-edit-header">
          <div className="site-edit-title">
            <button className="icon-btn" onClick={onClose}><FiX /></button>
            <h3>添加新站点</h3>
          </div>
          <button className="button primary" onClick={handleSubmit} disabled={loading}>
            {loading ? '创建中...' : '创建'}
          </button>
        </div>
        <div className="site-edit-body">
          {error && <div className="error-message">{error}</div>}
          <div className="form-field">
            <label>站点名称</label>
            <input
              className="input"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              placeholder="Example Site"
            />
          </div>
          <div className="form-field">
            <label>域名</label>
            <input
              className="input"
              value={form.hostname}
              onChange={(e) => setForm({ ...form, hostname: e.target.value })}
              placeholder="example.com"
            />
          </div>
          <div className="form-field">
            <label>IP 地址 (可选)</label>
            <input
              className="input"
              value={form.ip || ''}
              onChange={(e) => setForm({ ...form, ip: e.target.value })}
              placeholder="192.168.1.100"
            />
          </div>
        </div>
      </div>
    </>
  )
}


