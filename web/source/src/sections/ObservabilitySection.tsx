import { FiFilter, FiList } from 'react-icons/fi'
import type { HealthzResponse, LogStats } from '../admin/types'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/Card'
import { Select } from '../components/ui/Select'
import { Badge } from '../components/ui/Badge'

type ObservabilitySectionProps = {
  health: HealthzResponse | null
  logStats: LogStats | null
  loadingHealth: boolean
  loadingLogStats: boolean
}

export function ObservabilitySection({
  health,
  logStats,
  loadingHealth,
  loadingLogStats,
}: ObservabilitySectionProps) {
  const isLoading = loadingHealth || loadingLogStats
  const logLines = [
    `健康检查 ${health?.status || 'unknown'}`,
    `日志总量 ${logStats?.total ?? '-'}`,
    `阻断请求 ${logStats?.blocked ?? '-'}`,
    `转发请求 ${logStats?.proxied ?? '-'}`,
    `丢弃请求 ${logStats?.dropped ?? '-'}`,
  ]

  return (
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
  )
}
