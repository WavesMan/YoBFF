import type { CDNStatus, Config } from '../admin/types'

type SecuritySectionProps = {
  configDraft: Config
  cdnStatus: Record<string, CDNStatus>
  onAllowedCidrsChange: (value: string) => void
  onBlockPageChange: (value: string) => void
}

export function SecuritySection({
  configDraft,
  cdnStatus,
  onAllowedCidrsChange,
  onBlockPageChange,
}: SecuritySectionProps) {
  return (
    <>
      <div className="grid grid-2">
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">回源白名单</h3>
            <span className="pill">Config.security.allowedCidrs</span>
          </div>
          <textarea
            className="textarea"
            value={(configDraft.security?.allowedCidrs || []).join('\n')}
            onChange={(event) => onAllowedCidrsChange(event.target.value)}
            placeholder="192.168.1.0/24"
          />
        </div>
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">拦截页面</h3>
            <span className="pill">Config.security.blockPageHtml</span>
          </div>
          <textarea
            className="textarea"
            value={configDraft.security?.blockPageHtml || ''}
            onChange={(event) => onBlockPageChange(event.target.value)}
            placeholder="<html>...</html>"
          />
        </div>
      </div>
      <div className="card">
        <div className="card-header">
          <h3 className="card-title">CDN 同步状态</h3>
          <span className="pill">GET /admin/api/v1/config/cdn</span>
        </div>
        <div className="stack">
          {Object.keys(cdnStatus).length === 0 && <div className="muted">暂无同步状态</div>}
          {Object.entries(cdnStatus).map(([key, status]) => (
            <div key={key} className="inline">
              <span className="badge">{status.provider || key}</span>
              <span className="muted">CIDR {status.cidrs?.length || 0}</span>
              <span className="muted">最近同步 {status.last_sync || '-'}</span>
              {status.error ? <span className="badge">{status.error}</span> : <span className="badge">正常</span>}
            </div>
          ))}
        </div>
      </div>
    </>
  )
}
