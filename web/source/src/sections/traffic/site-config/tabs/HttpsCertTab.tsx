import { FiTrash2 } from 'react-icons/fi'
import type { Config } from '../../../../admin/types'

type HttpsCertTabProps = {
  config: Config
  handleRemoveCert: (index: number) => void
}

export function HttpsCertTab({ config, handleRemoveCert }: HttpsCertTabProps) {
  return (
    <div className="panel">
      <h4>HTTPS 证书管理</h4>
      <div className="muted" style={{ marginBottom: 15 }}>
        配置需要自动申请证书的域名列表。
      </div>
      
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
