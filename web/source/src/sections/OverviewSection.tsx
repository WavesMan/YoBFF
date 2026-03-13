import { FiActivity, FiServer, FiShield, FiAlertCircle, FiCheckCircle } from 'react-icons/fi'
import type { HealthzResponse, LogStats } from '../admin/types'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/Card'
import { Badge } from '../components/ui/Badge'

type OverviewSectionProps = {
  health: HealthzResponse | null
  logStats: LogStats | null
  logLevel: string
  loadingHealth: boolean
  loadingLogStats: boolean
}

export function OverviewSection({
  health,
  logStats,
  logLevel,
  loadingHealth,
  loadingLogStats,
}: OverviewSectionProps) {
  const healthStatus = health?.status === 'ok' ? 'success' : 'warning'
  
  const alertItems = []
  if (health && health.status !== 'ok') {
    alertItems.push({ level: 'error', text: '健康检查异常' })
  }
  if ((logStats?.blocked ?? 0) > 0) {
    alertItems.push({
      level: 'warning',
      text: `触发限流 ${logStats?.blocked ?? 0} 次`,
    })
  }
  if ((logStats?.dropped ?? 0) > 0) {
    alertItems.push({
      level: 'error',
      text: `请求丢弃 ${logStats?.dropped ?? 0} 次`,
    })
  }
  if (alertItems.length === 0) {
    alertItems.push({ level: 'success', text: '暂无告警' })
  }

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">服务健康度</CardTitle>
            <FiActivity className={`h-4 w-4 ${healthStatus === 'success' ? 'text-green-500' : 'text-yellow-500'}`} />
          </CardHeader>
          <CardContent>
            {loadingHealth ? (
              <div className="h-8 w-24 bg-muted animate-pulse rounded" />
            ) : (
              <div className="text-2xl font-bold">{health?.status || 'unknown'}</div>
            )}
            <p className="text-xs text-muted-foreground mt-1">GET /healthz</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">日志管线统计</CardTitle>
            <FiServer className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            {loadingLogStats ? (
              <div className="space-y-2">
                <div className="h-8 w-16 bg-muted animate-pulse rounded" />
                <div className="h-4 w-full bg-muted animate-pulse rounded" />
              </div>
            ) : (
              <>
                <div className="text-2xl font-bold">{logStats?.total ?? '-'}</div>
                <div className="flex flex-wrap gap-2 mt-2">
                  <Badge variant="default" className="text-[10px] h-5">blocked {logStats?.blocked ?? '-'}</Badge>
                  <Badge variant="default" className="text-[10px] h-5">proxied {logStats?.proxied ?? '-'}</Badge>
                  <Badge variant="default" className="text-[10px] h-5">dropped {logStats?.dropped ?? '-'}</Badge>
                </div>
              </>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">运行策略</CardTitle>
            <FiShield className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium">日志级别:</span>
              <Badge>{logLevel.toUpperCase()}</Badge>
            </div>
            <p className="text-xs text-muted-foreground mt-2">支持 Debug/Info/Warn/Error 即时切换</p>
          </CardContent>
        </Card>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <Card>
          <CardHeader>
            <CardTitle>告警摘要</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {alertItems.map((item, index) => (
                <div key={`${item.level}-${index}`} className="flex items-center gap-2 p-2 rounded-md bg-muted/30">
                  {item.level === 'error' && <FiAlertCircle className="text-destructive" />}
                  {item.level === 'warning' && <FiAlertCircle className="text-yellow-500" />}
                  {item.level === 'success' && <FiCheckCircle className="text-green-500" />}
                  <span className="text-sm">{item.text}</span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>趋势概览</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="h-[120px] flex items-center justify-center border-2 border-dashed rounded-md bg-muted/20">
              <span className="text-muted-foreground text-sm">图表区域已预留，可接入 Recharts 或 ECharts</span>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
