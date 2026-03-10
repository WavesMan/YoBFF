import type { ConfigVersion } from '../../../../admin/types'

type VersionHistoryTabProps = {
  versions: ConfigVersion[]
  handleRollback: (versionId: string) => Promise<void>
}

export function VersionHistoryTab({ versions, handleRollback }: VersionHistoryTabProps) {
  return (
    <div className="panel">
      <h4>历史版本</h4>
      <div className="version-list">
        {versions.map(v => (
          <div key={v.id} className="version-item">
            <div className="version-info">
              <span className="version-id">{v.id.substring(0, 8)}</span>
              <span className="version-date">{new Date(v.created_at).toLocaleString()}</span>
              <span className="version-operator">{v.operator || 'unknown'}</span>
            </div>
            <button 
              className="button small"
              onClick={() => handleRollback(v.id)}
            >
              回滚
            </button>
          </div>
        ))}
        {versions.length === 0 && <div className="muted">暂无历史版本</div>}
      </div>
    </div>
  )
}
