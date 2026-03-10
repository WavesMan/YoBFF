import type { Dispatch, SetStateAction } from 'react'
import type { SiteLogStream } from '../../../../admin/types'

type RealtimeLogTabProps = {
  siteId: string
  logStream: SiteLogStream | null
  setLogStream: Dispatch<SetStateAction<SiteLogStream | null>>
  logs: string[]
  setLogs: Dispatch<SetStateAction<string[]>>
  isLogConnected: boolean
  setIsLogConnected: (connected: boolean) => void
  loading: boolean
  handleUpdateLogStream: () => Promise<void>
}

export function RealtimeLogTab({
  siteId,
  logStream,
  setLogStream,
  logs,
  setLogs,
  isLogConnected,
  setIsLogConnected,
  loading,
  handleUpdateLogStream,
}: RealtimeLogTabProps) {
  return (
    <div className="panel">
      <h4>实时日志</h4>
      <div className="form-field">
        <label>日志过滤规则 (Filter Query)</label>
        <div style={{ display: 'flex', gap: '10px' }}>
          <input 
            className="input" 
            value={logStream?.filter_query || ''}
            onChange={e => setLogStream(prev => prev ? ({ ...prev, filter_query: e.target.value }) : { site_id: siteId, filter_query: e.target.value })}
            placeholder="e.g. level=error"
          />
          <button 
            className="button secondary"
            onClick={handleUpdateLogStream}
            disabled={loading}
          >
            更新规则
          </button>
        </div>
      </div>
      
      <div className="log-controls" style={{ margin: '15px 0', display: 'flex', gap: '10px' }}>
        <button 
          className={`button ${isLogConnected ? 'danger' : 'primary'}`}
          onClick={() => setIsLogConnected(!isLogConnected)}
        >
          {isLogConnected ? '断开连接' : '连接日志流'}
        </button>
        <button 
          className="button secondary"
          onClick={() => setLogs([])}
        >
          清空日志
        </button>
      </div>

      <div className="log-viewer">
        {logs.length === 0 ? (
          <div className="log-empty">等待日志数据...</div>
        ) : (
          logs.map((log, i) => (
            <div key={i} className="log-line">{log}</div>
          ))
        )}
      </div>
    </div>
  )
}
