import { Switch } from '../../../../components/ui/Switch'
import type { Config } from '../../../../admin/types'
import { Card, CardContent, CardHeader, CardTitle } from '../../../../components/ui/Card'
import { Input } from '../../../../components/ui/Input'

type RateLimitTabProps = {
  config: Config
  updateConfig: (updater: (prev: Config) => Config) => void
}

export function RateLimitTab({ config, updateConfig }: RateLimitTabProps) {
  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>流量限制</CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="border rounded-lg p-4 bg-card">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-3">
                <Switch
                  checked={config.rateLimit?.enabled || false}
                  onCheckedChange={(checked) => updateConfig(prev => ({
                    ...prev,
                    rateLimit: { ...prev.rateLimit, enabled: checked }
                  }))}
                />
                <div>
                  <div className="font-medium">启用全局速率限制 (Rate Limit)</div>
                  <div className="text-sm text-muted-foreground">
                    限制每个 IP 的请求速率，防止滥用。
                  </div>
                </div>
              </div>
            </div>

            {config.rateLimit?.enabled && (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6 pt-4 border-t animate-in fade-in slide-in-from-top-2 duration-200">
                <Input 
                  label="每秒请求数 (Requests/Second)"
                  type="number"
                  value={config.rateLimit?.requestsPerSecond || 0}
                  onChange={e => updateConfig(prev => ({
                    ...prev,
                    rateLimit: { ...prev.rateLimit, requestsPerSecond: parseInt(e.target.value) || 0 }
                  }))}
                  placeholder="例如: 10"
                />
                <Input 
                  label="突发流量 (Burst)"
                  type="number"
                  value={config.rateLimit?.burst || 0}
                  onChange={e => updateConfig(prev => ({
                    ...prev,
                    rateLimit: { ...prev.rateLimit, burst: parseInt(e.target.value) || 0 }
                  }))}
                  placeholder="例如: 20"
                />
              </div>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
