import { FiRefreshCw } from 'react-icons/fi'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/Card'
import { Button } from '../components/ui/Button'
import { Select } from '../components/ui/Select'
import { Badge } from '../components/ui/Badge'

type SystemSectionProps = {
  logLevel: string
  loadingLogLevel: boolean
  onLogLevelChange: (level: string) => void
  loadingReload: boolean
  onReload: () => void
}

const logLevels = ['debug', 'info', 'warn', 'error']

/**
 * 系统管理面板组件
 * 
 * 提供日志级别设置和系统热重载功能。
 * 
 * @param props.logLevel - 当前日志级别
 * @param props.loadingLogLevel - 日志级别同步状态
 * @param props.onLogLevelChange - 日志级别变更回调
 * @param props.loadingReload - 热重载操作状态
 * @param props.onReload - 触发热重载回调
 */
export function SystemSection({
  logLevel,
  loadingLogLevel,
  onLogLevelChange,
  loadingReload,
  onReload,
}: SystemSectionProps) {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
      <Card>
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <CardTitle className="text-lg font-medium">日志级别</CardTitle>
          <Badge variant="default" className="font-mono text-xs">GET/PUT /admin/api/v1/log/level</Badge>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-4">
            <div className="flex-1">
              <Select
                value={logLevel}
                onChange={(event) => onLogLevelChange(event.target.value)}
                options={logLevels.map(level => ({
                  value: level,
                  label: level.toUpperCase()
                }))}
                disabled={loadingLogLevel}
              />
            </div>
            {loadingLogLevel && <span className="text-sm text-muted-foreground animate-pulse">同步中...</span>}
          </div>
          <p className="text-xs text-muted-foreground mt-4">
            设置系统的全局日志级别。Debug 级别会产生大量日志，建议仅在排查问题时开启。
          </p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <CardTitle className="text-lg font-medium">热重载</CardTitle>
          <Badge variant="default" className="font-mono text-xs">POST /admin/api/v1/config/reload</Badge>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-4">
            <Button 
              className="w-full"
              variant="secondary" 
              onClick={onReload} 
              disabled={loadingReload}
              icon={<FiRefreshCw className={loadingReload ? "animate-spin" : ""} />}
            >
              {loadingReload ? '触发中...' : '触发热重载'}
            </Button>
          </div>
          <p className="text-xs text-muted-foreground mt-4">
            强制重新加载配置文件。通常情况下配置更改会自动生效，无需手动触发。
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
