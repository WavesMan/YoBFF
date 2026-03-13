import type { Config, Site, SiteUpdateRequest } from '../../../../admin/types'
import { Button } from '../../../../components/ui/Button'
import { Input } from '../../../../components/ui/Input'
import { Card, CardContent, CardHeader, CardTitle } from '../../../../components/ui/Card'

type BasicSettingsTabProps = {
  site: Site | null
  siteId: string
  siteForm: SiteUpdateRequest
  setSiteForm: (form: SiteUpdateRequest) => void
  config: Config
  updateConfig: (updater: (prev: Config) => Config) => void
  handleSaveSiteInfo: () => Promise<void>
  updatingSite: boolean
  loading: boolean
}

export function BasicSettingsTab({
  site,
  siteId,
  siteForm,
  setSiteForm,
  config,
  updateConfig,
  handleSaveSiteInfo,
  updatingSite,
  loading,
}: BasicSettingsTabProps) {
  return (
    <div className="space-y-6">
      <Card className="border-0 shadow-none">
        <CardHeader className="flex flex-row items-center justify-between px-0 pt-0">
          <CardTitle className="text-lg">基础设置</CardTitle>
          <Button 
            variant="secondary"
            size="sm"
            onClick={handleSaveSiteInfo}
            disabled={updatingSite || loading}
          >
            {updatingSite ? '更新中...' : '更新基本信息'}
          </Button>
        </CardHeader>
        
        <CardContent className="px-0 space-y-4">
          <Input 
            label="站点 ID" 
            value={siteId} 
            disabled 
            readOnly
          />
          
          {site && (
            <>
              <Input 
                label="站点名称" 
                value={siteForm.name || ''} 
                onChange={e => setSiteForm({ ...siteForm, name: e.target.value })}
              />
              <Input 
                label="主域名" 
                value={siteForm.hostname || ''} 
                onChange={e => setSiteForm({ ...siteForm, hostname: e.target.value })}
              />
              <Input 
                label="绑定 IP" 
                value={siteForm.ip || ''} 
                onChange={e => setSiteForm({ ...siteForm, ip: e.target.value })}
              />
            </>
          )}
        </CardContent>
      </Card>

      <div className="border-t my-6" />

      <Card className="border-0 shadow-none">
        <CardHeader className="px-0 pt-0">
          <CardTitle className="text-lg">数据面配置</CardTitle>
        </CardHeader>
        <CardContent className="px-0">
          <Input 
            label="HTTPS 监听地址"
            value={config.dataPlane?.httpsListenAddr || ''}
            onChange={e => updateConfig(prev => ({
              ...prev,
              dataPlane: { ...prev.dataPlane, httpsListenAddr: e.target.value }
            }))}
            placeholder=":443"
          />
        </CardContent>
      </Card>
    </div>
  )
}
