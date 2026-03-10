import { useCallback, useEffect, useState } from 'react'
import {
  FiClock,
  FiLock,
  FiPlus,
  FiTrash2,
  FiFile
} from 'react-icons/fi'
import {
  deleteCertificate,
  fetchCertificates,
  uploadCertificate
} from '../admin/api'
import type { SSLCertificate } from '../admin/types'

type CertificatesSectionProps = {
  token: string
}

export function CertificatesSection({ token }: CertificatesSectionProps) {
  const [certs, setCerts] = useState<SSLCertificate[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [showUpload, setShowUpload] = useState(false)
  const [uploadName, setUploadName] = useState('')
  const [certFile, setCertFile] = useState<File | null>(null)
  const [keyFile, setKeyFile] = useState<File | null>(null)
  const [uploading, setUploading] = useState(false)

  const loadCertificates = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const data = await fetchCertificates(token, page, 15)
      setCerts(data.items || [])
      setTotal(data.total || 0)
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载证书列表失败')
    } finally {
      setLoading(false)
    }
  }, [token, page])

  useEffect(() => {
    loadCertificates()
  }, [loadCertificates])

  const handleDelete = async (id: string) => {
    if (!confirm('确定要删除此证书吗？这将影响所有使用此证书的站点。')) return
    
    try {
      await deleteCertificate(token, id)
      loadCertificates()
    } catch (err) {
      alert(err instanceof Error ? err.message : '删除失败')
    }
  }

  const handleUpload = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!uploadName || !certFile || !keyFile) {
      alert('请完整填写信息')
      return
    }

    setUploading(true)
    const formData = new FormData()
    formData.append('name', uploadName)
    formData.append('cert', certFile)
    formData.append('key', keyFile)

    try {
      await uploadCertificate(token, formData)
      setShowUpload(false)
      setUploadName('')
      setCertFile(null)
      setKeyFile(null)
      loadCertificates()
    } catch (err) {
      alert(err instanceof Error ? err.message : '上传失败')
    } finally {
      setUploading(false)
    }
  }

  const totalPages = Math.ceil(total / 15)

  return (
    <div className="card">
      <div className="card-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '15px 20px', borderBottom: '1px solid var(--border)' }}>
        <div>
          <h3 className="card-title" style={{ margin: 0, fontSize: '18px', fontWeight: 600 }}>HTTPS 证书管理</h3>
          <div className="subtitle" style={{ fontSize: '13px', color: 'var(--text-sub)', marginTop: '4px' }}>
            统一管理 SSL/TLS 证书，支持自动续期检测
          </div>
        </div>
        <div className="actions">
          <button className="button primary" onClick={() => setShowUpload(true)}>
            <FiPlus /> 上传证书
          </button>
        </div>
      </div>

      {error && <div className="error-message">{error}</div>}

      <div className="card-content">
        {loading && certs.length === 0 ? (
          <div className="loading">加载中...</div>
        ) : certs.length === 0 ? (
          <div className="empty-state">
            <FiLock size={48} className="icon-muted" />
            <p>暂无证书，请点击上方按钮上传</p>
          </div>
        ) : (
          <div className="table-container">
            <table className="table">
              <thead>
                <tr>
                  <th>备注名</th>
                  <th>包含域名</th>
                  <th>颁发机构</th>
                  <th>过期时间</th>
                  <th>创建时间</th>
                  <th style={{ textAlign: 'right' }}>操作</th>
                </tr>
              </thead>
              <tbody>
                {certs.map(cert => {
                  const expiry = new Date(cert.notAfter)
                  const isExpired = expiry < new Date()
                  const isNearExpiry = expiry < new Date(Date.now() + 30 * 24 * 3600 * 1000)
                  
                  return (
                    <tr key={cert.id}>
                      <td>
                        <div className="cert-name-cell" style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                          <FiLock className="icon-subtle" />
                          <span className="font-medium">{cert.name}</span>
                        </div>
                      </td>
                      <td>
                        <div className="domain-tags" style={{ display: 'flex', flexWrap: 'wrap', gap: '4px' }}>
                          {(cert.domains || []).slice(0, 3).map(d => (
                            <span key={d} className="tag">{d}</span>
                          ))}
                          {(cert.domains || []).length > 3 && (
                            <span className="tag">+{cert.domains.length - 3}</span>
                          )}
                        </div>
                      </td>
                      <td className="text-muted">{cert.issuer}</td>
                      <td>
                        <div className={`expiry-date ${isExpired ? 'text-error' : isNearExpiry ? 'text-warning' : ''}`} style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                          <FiClock size={14} />
                          {expiry.toLocaleDateString()}
                        </div>
                      </td>
                      <td className="text-muted">
                        {new Date(cert.createdAt).toLocaleDateString()}
                      </td>
                      <td style={{ textAlign: 'right' }}>
                        <button 
                          className="button small danger"
                          onClick={() => handleDelete(cert.id)}
                          title="删除证书"
                        >
                          <FiTrash2 />
                        </button>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {total > 15 && (
        <div className="pagination" style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', gap: '15px', padding: '20px', borderTop: '1px solid var(--border)' }}>
          <button 
            className="button ghost small" 
            disabled={page === 1}
            onClick={() => setPage(p => p - 1)}
          >
            上一页
          </button>
          <span className="page-info text-muted">第 {page} 页 / 共 {totalPages} 页</span>
          <button 
            className="button ghost small"
            disabled={page === totalPages}
            onClick={() => setPage(p => p + 1)}
          >
            下一页
          </button>
        </div>
      )}

      {showUpload && (
        <div className="modal-overlay" onClick={() => setShowUpload(false)}>
          <div className="modal-content" style={{ maxWidth: '500px' }} onClick={e => e.stopPropagation()}>
            <div className="modal-header">
              <h4>上传新证书</h4>
              <button className="icon-btn" onClick={() => setShowUpload(false)}>
                <svg stroke="currentColor" fill="none" strokeWidth="2" viewBox="0 0 24 24" strokeLinecap="round" strokeLinejoin="round" height="1em" width="1em" xmlns="http://www.w3.org/2000/svg">
                  <line x1="18" y1="6" x2="6" y2="18"></line>
                  <line x1="6" y1="6" x2="18" y2="18"></line>
                </svg>
              </button>
            </div>
            <div className="modal-body">
              <form onSubmit={handleUpload} style={{ display: 'grid', gap: '15px' }}>
                <div className="form-field">
                  <label>备注名称</label>
                  <input
                    type="text"
                    value={uploadName}
                    onChange={e => setUploadName(e.target.value)}
                    placeholder="例如：example.com 2025"
                    className="input"
                    autoFocus
                  />
                </div>
                <div className="form-field">
                  <label>证书文件 (.pem/.crt)</label>
                  <div className="file-input-wrapper">
                    <input
                      type="file"
                      id="cert-file"
                      accept=".pem,.crt,.cer"
                      onChange={e => setCertFile(e.target.files?.[0] || null)}
                      className="hidden-file-input"
                    />
                    <label htmlFor="cert-file" className="file-input-label">
                      <FiFile />
                      {certFile ? certFile.name : '选择证书文件...'}
                    </label>
                  </div>
                </div>
                <div className="form-field">
                  <label>私钥文件 (.key/.pem)</label>
                  <div className="file-input-wrapper">
                    <input
                      type="file"
                      id="key-file"
                      accept=".key,.pem"
                      onChange={e => setKeyFile(e.target.files?.[0] || null)}
                      className="hidden-file-input"
                    />
                    <label htmlFor="key-file" className="file-input-label">
                      <FiFile />
                      {keyFile ? keyFile.name : '选择私钥文件...'}
                    </label>
                  </div>
                </div>
                <div className="form-actions" style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', marginTop: '10px' }}>
                  <button type="button" className="button ghost" onClick={() => setShowUpload(false)}>
                    取消
                  </button>
                  <button type="submit" className="button primary" disabled={uploading}>
                    {uploading ? '上传中...' : '确认上传'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        </div>
      )}

      <style>{`
        .card {
          background: var(--bg-surface);
          border-radius: 8px;
          box-shadow: 0 1px 3px rgba(0,0,0,0.1);
          margin-bottom: 20px;
        }
        .error-message {
          color: var(--error);
          padding: 10px 20px;
          background: rgba(244, 67, 54, 0.1);
          border-bottom: 1px solid var(--border);
        }
        .table {
          width: 100%;
          border-collapse: collapse;
        }
        .table th, .table td {
          padding: 12px 20px;
          text-align: left;
          border-bottom: 1px solid var(--border);
        }
        .table th {
          background: var(--bg-hover);
          font-weight: 500;
          color: var(--text-sub);
          font-size: 13px;
        }
        .table tr:last-child td {
          border-bottom: none;
        }
        .tag {
          background: var(--bg-hover);
          padding: 2px 6px;
          border-radius: 4px;
          font-size: 12px;
          color: var(--text-sub);
        }
        .text-muted {
          color: var(--text-sub);
        }
        .text-error {
          color: var(--error);
        }
        .text-warning {
          color: #f59e0b;
        }
        .icon-subtle {
          color: var(--text-sub);
          opacity: 0.7;
        }
        .empty-state {
          padding: 40px;
          text-align: center;
          color: var(--text-sub);
          display: flex;
          flex-direction: column;
          align-items: center;
          gap: 10px;
        }
        .icon-muted {
          color: var(--border);
        }
        
        /* Modal Styles */
        .modal-overlay {
          position: fixed;
          top: 0;
          left: 0;
          width: 100%;
          height: 100%;
          background: rgba(0, 0, 0, 0.5);
          z-index: 1000;
          display: flex;
          justify-content: center;
          align-items: center;
        }
        .modal-content {
          background: var(--bg-surface);
          border-radius: 8px;
          width: 90%;
          box-shadow: 0 4px 20px rgba(0,0,0,0.15);
          overflow: hidden;
        }
        .modal-header {
          padding: 15px 20px;
          border-bottom: 1px solid var(--border);
          display: flex;
          justify-content: space-between;
          align-items: center;
        }
        .modal-header h4 {
          margin: 0;
          font-size: 16px;
        }
        .modal-body {
          padding: 20px;
        }
        .file-input-wrapper {
          position: relative;
        }
        .hidden-file-input {
          position: absolute;
          width: 1px;
          height: 1px;
          padding: 0;
          margin: -1px;
          overflow: hidden;
          clip: rect(0, 0, 0, 0);
          border: 0;
        }
        .file-input-label {
          display: flex;
          align-items: center;
          gap: 8px;
          padding: 8px 12px;
          border: 1px dashed var(--border);
          border-radius: 4px;
          cursor: pointer;
          color: var(--text-sub);
          transition: all 0.2s;
        }
        .file-input-label:hover {
          border-color: var(--primary);
          color: var(--primary);
          background: var(--bg-hover);
        }
        .icon-btn {
          background: none;
          border: none;
          color: var(--text-sub);
          cursor: pointer;
          font-size: 20px;
          padding: 5px;
          display: flex;
          align-items: center;
          justify-content: center;
          transition: color 0.2s;
        }
        .icon-btn:hover {
          color: var(--text-main);
        }
      `}</style>
    </div>
  )
}
