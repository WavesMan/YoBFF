import { useEffect, useMemo, useState, type Dispatch, type SetStateAction } from 'react'
import type { SiteLogEntry, SiteLogKind, SiteLogStream } from '../../../../admin/types'

type RealtimeLogTabProps = {
  siteId: string
  logStream: SiteLogStream | null
  setLogStream: Dispatch<SetStateAction<SiteLogStream | null>>
  logs: SiteLogEntry[]
  setLogs: Dispatch<SetStateAction<SiteLogEntry[]>>
  loadLogHistory: (params: {
    kind: SiteLogKind
    level: string
    startTime: string
    endTime: string
    limit: number
  }) => Promise<void>
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
  loadLogHistory,
  isLogConnected,
  setIsLogConnected,
  loading,
  handleUpdateLogStream,
}: RealtimeLogTabProps) {
  const [kind, setKind] = useState<SiteLogKind>('all')
  const [level, setLevel] = useState('')
  const [startTime, setStartTime] = useState('')
  const [endTime, setEndTime] = useState('')
  const [limit, setLimit] = useState(100)
  const [querying, setQuerying] = useState(false)

  const queryParams = useMemo(() => ({
    kind,
    level,
    startTime: startTime ? new Date(startTime).toISOString() : '',
    endTime: endTime ? new Date(endTime).toISOString() : '',
    limit,
  }), [kind, level, startTime, endTime, limit])

  const handleQuery = async () => {
    setQuerying(true)
    try {
      await loadLogHistory(queryParams)
    } finally {
      setQuerying(false)
    }
  }

  useEffect(() => {
    if (!siteId) return
    void loadLogHistory({
      kind: 'all',
      level: '',
      startTime: '',
      endTime: '',
      limit: 100,
    })
  }, [siteId, loadLogHistory])

  useEffect(() => {
    if (!isLogConnected) return
    const timer = setInterval(() => {
      void loadLogHistory(queryParams)
    }, 2000)
    return () => clearInterval(timer)
  }, [isLogConnected, loadLogHistory, queryParams])

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

      <div className="form-field">
        <label>日志筛选</label>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', gap: '10px' }}>
          <select className="input" value={kind} onChange={e => setKind(e.target.value as SiteLogKind)}>
            <option value="all">全部类型</option>
            <option value="traffic">流量日志</option>
            <option value="system">系统日志</option>
          </select>
          <select className="input" value={level} onChange={e => setLevel(e.target.value)}>
            <option value="">全部等级</option>
            <option value="debug">debug</option>
            <option value="info">info</option>
            <option value="warn">warn</option>
            <option value="error">error</option>
          </select>
          <input
            className="input"
            type="number"
            min={1}
            max={500}
            value={limit}
            onChange={e => setLimit(Number(e.target.value) || 100)}
            placeholder="条数"
          />
          <input className="input" type="datetime-local" value={startTime} onChange={e => setStartTime(e.target.value)} />
          <input className="input" type="datetime-local" value={endTime} onChange={e => setEndTime(e.target.value)} />
          <button className="button secondary" onClick={handleQuery} disabled={loading || querying}>
            {querying ? '查询中...' : '查询历史'}
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
          logs.map(log => (
            <div key={log.id} className="log-line">
              [{log.created_at}] [{log.kind}] [{log.level}] {log.message}
              {log.method ? ` ${log.method}` : ''}
              {log.path ? ` ${log.path}` : ''}
              {log.status_code ? ` ${log.status_code}` : ''}
              {log.latency_ms ? ` ${log.latency_ms}ms` : ''}
            </div>
          ))
        )}
      </div>
    </div>
  )
}
