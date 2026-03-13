import { useEffect, useMemo, useState, type Dispatch, type SetStateAction } from 'react'
import { FiRefreshCw, FiPlay, FiPause, FiTrash2, FiSearch } from 'react-icons/fi'
import type { SiteLogEntry, SiteLogKind, SiteLogStream } from '../../../../admin/types'
import { Card, CardContent, CardHeader, CardTitle } from '../../../../components/ui/Card'
import { Input } from '../../../../components/ui/Input'
import { Select } from '../../../../components/ui/Select'
import { Button } from '../../../../components/ui/Button'
import { Badge, type BadgeVariant } from '../../../../components/ui/Badge'

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

  const getLevelBadgeVariant = (level: string): BadgeVariant => {
    switch (level.toLowerCase()) {
      case 'error': return 'error'
      case 'warn': return 'warning'
      case 'info': return 'primary'
      case 'debug': return 'default' // Changed from secondary to default as per BadgeVariant
      default: return 'default'
    }
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>实时日志</CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="space-y-4">
            <div className="flex flex-col gap-2">
              <label className="text-sm font-medium">日志过滤规则 (Filter Query)</label>
              <div className="flex gap-2">
                <div className="flex-1">
                  <Input 
                    value={logStream?.filter_query || ''}
                    onChange={e => setLogStream(prev => prev ? ({ ...prev, filter_query: e.target.value }) : { site_id: siteId, filter_query: e.target.value })}
                    placeholder="e.g. level=error"
                  />
                </div>
                <Button 
                  variant="secondary"
                  onClick={handleUpdateLogStream}
                  disabled={loading}
                  icon={<FiRefreshCw className={loading ? "animate-spin" : ""} />}
                >
                  更新规则
                </Button>
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-6 gap-4 items-end">
              <div className="lg:col-span-1">
                <Select
                  label="日志类型"
                  value={kind} 
                  onChange={e => setKind(e.target.value as SiteLogKind)}
                  options={[
                    { value: "all", label: "全部类型" },
                    { value: "traffic", label: "流量日志" },
                    { value: "system", label: "系统日志" }
                  ]}
                />
              </div>
              <div className="lg:col-span-1">
                <Select
                  label="日志等级"
                  value={level}
                  onChange={e => setLevel(e.target.value)}
                  options={[
                    { value: "", label: "全部等级" },
                    { value: "debug", label: "debug" },
                    { value: "info", label: "info" },
                    { value: "warn", label: "warn" },
                    { value: "error", label: "error" }
                  ]}
                />
              </div>
              <div className="lg:col-span-1">
                <Input
                  label="条数限制"
                  type="number"
                  min={1}
                  max={500}
                  value={limit}
                  onChange={e => setLimit(Number(e.target.value) || 100)}
                  placeholder="条数"
                />
              </div>
              <div className="lg:col-span-1">
                <Input 
                  label="开始时间"
                  type="datetime-local" 
                  value={startTime} 
                  onChange={e => setStartTime(e.target.value)} 
                />
              </div>
              <div className="lg:col-span-1">
                <Input 
                  label="结束时间"
                  type="datetime-local" 
                  value={endTime} 
                  onChange={e => setEndTime(e.target.value)} 
                />
              </div>
              <div className="lg:col-span-1">
                <Button 
                  className="w-full"
                  variant="secondary" 
                  onClick={handleQuery} 
                  disabled={loading || querying}
                  icon={<FiSearch />}
                >
                  {querying ? '查询中...' : '查询历史'}
                </Button>
              </div>
            </div>

            <div className="flex gap-2 pt-2 border-t">
              <Button 
                variant={isLogConnected ? 'danger' : 'primary'}
                size="sm"
                onClick={() => setIsLogConnected(!isLogConnected)}
                icon={isLogConnected ? <FiPause /> : <FiPlay />}
              >
                {isLogConnected ? '断开连接' : '连接日志流'}
              </Button>
              <Button 
                variant="secondary"
                onClick={() => setLogs([])}
                icon={<FiTrash2 />}
              >
                清空日志
              </Button>
            </div>

            <div className="border rounded-md bg-muted/50 font-mono text-xs h-[500px] overflow-auto p-4 space-y-1">
              {logs.length === 0 ? (
                <div className="text-center text-muted-foreground py-20">等待日志数据...</div>
              ) : (
                logs.map(log => (
                  <div key={log.id} className="hover:bg-muted/50 p-1 rounded flex gap-2 break-all">
                    <span className="text-muted-foreground shrink-0 w-[140px]">{new Date(log.created_at).toLocaleString()}</span>
                    <Badge variant="default" className="h-5 px-1 text-[10px] uppercase shrink-0 w-[60px] justify-center">
                      {log.kind}
                    </Badge>
                    <Badge 
                      variant={getLevelBadgeVariant(log.level)} 
                      className="h-5 px-1 text-[10px] uppercase shrink-0 w-[50px] justify-center"
                    >
                      {log.level}
                    </Badge>
                    <span className="flex-1">
                      {log.message}
                      {log.method && <span className="ml-2 text-primary">{log.method}</span>}
                      {log.path && <span className="ml-1 text-muted-foreground">{log.path}</span>}
                      {log.status_code && (
                        <span className={`ml-2 ${log.status_code >= 400 ? 'text-destructive' : 'text-green-500'}`}>
                          {log.status_code}
                        </span>
                      )}
                      {log.latency_ms && <span className="ml-2 text-muted-foreground">{log.latency_ms}ms</span>}
                    </span>
                  </div>
                ))
              )}
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
