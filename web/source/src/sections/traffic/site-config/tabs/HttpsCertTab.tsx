import { FiCheck, FiTrash2 } from 'react-icons/fi'
import type { Config, Site, SSLCertificate } from '../../../../admin/types'

type HttpsCertTabProps = {
  site: Site | null
  config: Config
  updateConfig: (updater: (prev: Config) => Config) => void
  filteredCerts: SSLCertificate[]
  handleRemoveCert: (index: number) => void
}

export function HttpsCertTab({
  site,
  config,
  updateConfig,
  filteredCerts,
  handleRemoveCert,
}: HttpsCertTabProps) {
  const selectedCert = filteredCerts.find(cert => cert.id === config.dataPlane?.certId)
  return (
    <div className="panel">
      <h4>HTTPS 证书</h4>
      <div className="muted" style={{ marginBottom: 15 }}>
        为站点 {site?.hostname || '-'} 选择已上传的 SSL 证书，并可配置自动申请证书的域名列表。
      </div>

      <div className="panel" style={{ marginBottom: 16 }}>
        <h4 style={{ marginTop: 0 }}>已上传证书（匹配站点域名）</h4>
        {filteredCerts.length === 0 ? (
          <div className="muted">没有找到匹配的证书，请先在证书管理中上传。</div>
        ) : (
          <div className="table-container">
            <table className="table">
              <thead>
                <tr>
                  <th>备注名</th>
                  <th>包含域名</th>
                  <th>颁发机构</th>
                  <th>过期时间</th>
                  <th style={{ textAlign: 'right' }}>操作</th>
                </tr>
              </thead>
              <tbody>
                {filteredCerts.map(cert => {
                  const isSelected = config.dataPlane?.certId === cert.id
                  return (
                    <tr key={cert.id}>
                      <td style={{ fontWeight: isSelected ? 600 : 400 }}>
                        {cert.name} {isSelected ? '（已选择）' : ''}
                      </td>
                      <td className="text-muted">
                        {(cert.domains || []).slice(0, 3).join(', ')}
                        {(cert.domains || []).length > 3 ? ` +${cert.domains.length - 3}` : ''}
                      </td>
                      <td className="text-muted">{cert.issuer || '-'}</td>
                      <td className="text-muted">{new Date(cert.notAfter).toLocaleDateString()}</td>
                      <td style={{ textAlign: 'right' }}>
                        <button
                          className={`button small ${isSelected ? 'secondary' : 'primary'}`}
                          disabled={isSelected}
                          onClick={() => {
                            updateConfig(prev => ({
                              ...prev,
                              dataPlane: {
                                ...prev.dataPlane,
                                enableHttps: true,
                                certId: cert.id,
                              },
                            }))
                          }}
                          title={isSelected ? '已选择' : '选择此证书'}
                        >
                          {isSelected ? <FiCheck /> : '选择'}
                        </button>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
        <div className="muted" style={{ marginTop: 10 }}>
          当前状态：{config.dataPlane?.enableHttps ? 'HTTPS 已启用' : 'HTTPS 未启用'}
          {selectedCert ? `，使用证书「${selectedCert.name}」` : config.dataPlane?.certId ? '，证书未匹配到站点域名' : ''}
        </div>
      </div>

      <h4 style={{ marginTop: 0 }}>自动申请证书域名</h4>
      <table className="table">
        <thead>
          <tr>
            <th>域名</th>
            <th style={{ textAlign: 'right' }}>操作</th>
          </tr>
        </thead>
        <tbody>
          {(config.certificates || []).map((cert, index) => (
            <tr key={index}>
              <td>{cert.domain}</td>
              <td style={{ textAlign: 'right' }}>
                <button className="button danger small" onClick={() => handleRemoveCert(index)}>
                  <FiTrash2 />
                </button>
              </td>
            </tr>
          ))}
          {(!config.certificates || config.certificates.length === 0) && (
            <tr>
              <td colSpan={2} className="muted" style={{ textAlign: 'center' }}>
                暂无证书配置
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  )
}
