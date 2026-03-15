import { useCallback, useEffect, useMemo, useState } from 'react'
import { FiFilter, FiList } from 'react-icons/fi'
import type { AuditLogEntry, HealthzResponse, LogStats } from '../admin/types'
import { fetchAuditLogs } from '../admin/api'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/Card'
import { Select } from '../components/ui/Select'
import { Badge } from '../components/ui/Badge'
import { Table } from '../components/ui/Table'
import { Input } from '../components/ui/Input'
import { Button } from '../components/ui/Button'

type ObservabilitySectionProps = {
  health: HealthzResponse | null
  logStats: LogStats | null
  loadingHealth: boolean
  loadingLogStats: boolean
  token: string
}

/**
 *
 * 可观测性区域，用于展示日志概况与审计轨迹。
 *
 */
export function ObservabilitySection({
  health,
  logStats,
  loadingHealth,
  loadingLogStats,
  token,
}: ObservabilitySectionProps) {
  const isLoading = loadingHealth || loadingLogStats
  const [auditLogs, setAuditLogs] = useState<AuditLogEntry[]>([])
  const [auditLoading, setAuditLoading] = useState(false)
  const [auditAction, setAuditAction] = useState('')
  const [auditOperator, setAuditOperator] = useState('')
  const [auditTarget, setAuditTarget] = useState('')
  const [auditStartTime, setAuditStartTime] = useState('')
  const [auditEndTime, setAuditEndTime] = useState('')
  const [auditLimit, setAuditLimit] = useState(100)
  const [auditPage, setAuditPage] = useState(1)
  const [auditPageSize, setAuditPageSize] = useState(15)
  const [auditTotal, setAuditTotal] = useState(0)
  const logLines = [
    `健康检查 ${health?.status || 'unknown'}`,
    `日志总量 ${logStats?.total ?? '-'}`,
    `阻断请求 ${logStats?.blocked ?? '-'}`,
    `转发请求 ${logStats?.proxied ?? '-'}`,
    `丢弃请求 ${logStats?.dropped ?? '-'}`,
  ]
  const auditColumns = useMemo(() => ([
    { key: 'created_at', title: '时间' },
    { key: 'action', title: '动作' },
    { key: 'target', title: '目标' },
    { key: 'operator', title: '操作人' },
    {
      key: 'detail',
      title: '详情',
      render: (record: AuditLogEntry) => {
        const text = record.detail || ''
        return text.length > 120 ? `${text.slice(0, 120)}...` : text
      },
    },
  ]), [])

  /**
   *
   * 触发审计日志查询，用于筛选管理台操作记录。
   *
   */
  const loadAuditLogs = useCallback(async (page: number, pageSize: number) => {
    setAuditLoading(true)
    try {
      const startText = auditStartTime ? new Date(auditStartTime).toISOString() : ''
      const endText = auditEndTime ? new Date(auditEndTime).toISOString() : ''
      const response = await fetchAuditLogs(token, {
        action: auditAction.trim(),
        operator: auditOperator.trim(),
        target: auditTarget.trim(),
        startTime: startText,
        endTime: endText,
        limit: auditLimit,
        page,
        pageSize,
      })
      setAuditLogs(response.items || [])
      setAuditTotal(response.total || 0)
    } finally {
      setAuditLoading(false)
    }
  }, [auditAction, auditEndTime, auditLimit, auditOperator, auditStartTime, auditTarget, token])

  const handleAuditQuery = async () => {
    if (auditPage !== 1) {
      setAuditPage(1)
      return
    }
    await loadAuditLogs(1, auditPageSize)
  }

  useEffect(() => {
    if (!token) {
      setAuditLogs([])
      setAuditTotal(0)
      return
    }
    void loadAuditLogs(auditPage, auditPageSize)
  }, [token, auditPage, auditPageSize, loadAuditLogs])

  return (
    <div className="flex flex-col gap-6">
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card className="h-full flex flex-col">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="flex items-center gap-2">
              <FiList />
              实时日志
            </CardTitle>
            <Badge variant="default">WebSocket</Badge>
          </CardHeader>
          <CardContent className="flex-1">
            <div className="bg-black/90 text-green-400 font-mono text-xs p-4 rounded-md h-[300px] overflow-auto shadow-inner">
              {isLoading ? (
                <div className="animate-pulse">日志加载中...</div>
              ) : (
                logLines.map((line, index) => (
                  <div key={index} className="mb-1 border-b border-white/10 pb-1 last:border-0">
                    <span className="text-gray-500 mr-2">[{new Date().toLocaleTimeString()}]</span>
                    {line}
                  </div>
                ))
              )}
            </div>
          </CardContent>
        </Card>

        <Card className="h-full flex flex-col">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="flex items-center gap-2">
              <FiFilter />
              日志分析
            </CardTitle>
            <Badge variant="default">筛选器</Badge>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="grid grid-cols-2 gap-4">
              <Select
                label="时间范围"
                options={[
                  { value: "1h", label: "最近 1 小时" },
                  { value: "24h", label: "最近 24 小时" },
                  { value: "7d", label: "最近 7 天" }
                ]}
                defaultValue="1h"
              />
              <Select
                label="状态码"
                options={[
                  { value: "all", label: "全部" },
                  { value: "2xx", label: "2xx" },
                  { value: "4xx", label: "4xx" },
                  { value: "5xx", label: "5xx" }
                ]}
                defaultValue="all"
              />
            </div>

            <div className="border rounded-md p-4 bg-muted/30">
              <h4 className="text-sm font-medium mb-3 text-muted-foreground">统计概览</h4>
              {isLoading ? (
                <div className="space-y-2">
                  <div className="h-4 bg-muted animate-pulse rounded w-3/4" />
                  <div className="h-4 bg-muted animate-pulse rounded w-1/2" />
                </div>
              ) : (
                <div className="grid grid-cols-2 gap-4">
                  <div className="flex flex-col gap-1 p-2 bg-background rounded border">
                    <span className="text-xs text-muted-foreground">Total Requests</span>
                    <span className="text-xl font-bold">{logStats?.total ?? '-'}</span>
                  </div>
                  <div className="flex flex-col gap-1 p-2 bg-background rounded border">
                    <span className="text-xs text-muted-foreground">Blocked</span>
                    <span className="text-xl font-bold text-yellow-500">{logStats?.blocked ?? '-'}</span>
                  </div>
                  <div className="flex flex-col gap-1 p-2 bg-background rounded border">
                    <span className="text-xs text-muted-foreground">Proxied</span>
                    <span className="text-xl font-bold text-green-500">{logStats?.proxied ?? '-'}</span>
                  </div>
                  <div className="flex flex-col gap-1 p-2 bg-background rounded border">
                    <span className="text-xs text-muted-foreground">Dropped</span>
                    <span className="text-xl font-bold text-destructive">{logStats?.dropped ?? '-'}</span>
                  </div>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      </div>

      <Card className="h-full flex flex-col">
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="flex items-center gap-2">
            <FiFilter />
            审计日志
          </CardTitle>
          <Badge variant="default">管理台</Badge>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-7 gap-2 items-end">
            <Input
              label="动作"
              placeholder="如 config_apply"
              value={auditAction}
              onChange={(event) => setAuditAction(event.target.value)}
            />
            <Input
              label="操作人"
              placeholder="操作者名称"
              value={auditOperator}
              onChange={(event) => setAuditOperator(event.target.value)}
            />
            <Input
              label="目标"
              placeholder="站点或资源 ID"
              value={auditTarget}
              onChange={(event) => setAuditTarget(event.target.value)}
            />
            <Input
              label="返回数量"
              type="number"
              min={1}
              max={500}
              value={auditLimit}
              onChange={(event) => setAuditLimit(Number(event.target.value))}
            />
            <Input
              label="开始时间"
              type="datetime-local"
              value={auditStartTime}
              onChange={(event) => setAuditStartTime(event.target.value)}
            />
            <Input
              label="结束时间"
              type="datetime-local"
              value={auditEndTime}
              onChange={(event) => setAuditEndTime(event.target.value)}
            />
            <Button variant="primary" className="w-full" onClick={handleAuditQuery} disabled={auditLoading}>
              {auditLoading ? '加载中...' : '查询'}
            </Button>
          </div>
          <Table
            columns={auditColumns}
            data={auditLogs}
            loading={auditLoading}
            emptyText="暂无审计记录"
            rowKey="id"
            pagination={{
              total: auditTotal,
              page: auditPage,
              pageSize: auditPageSize,
              onPageChange: (page) => setAuditPage(page),
              onPageSizeChange: (pageSize) => {
                setAuditPageSize(pageSize)
                setAuditPage(1)
              },
            }}
          />
        </CardContent>
      </Card>
    </div>
  )
}
