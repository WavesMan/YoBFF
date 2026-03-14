import type { Config, Site, SiteUpdateRequest } from '../../../../admin/types'
import { Button } from '../../../../components/ui/Button'
import { Input } from '../../../../components/ui/Input'

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
      <div>
        <div className="flex flex-row items-center justify-between px-0 pt-0 mb-4">
          <h3 className="text-lg font-medium">基础设置</h3>
          <Button 
            variant="secondary"
            size="sm"
            onClick={handleSaveSiteInfo}
            disabled={updatingSite || loading}
          >
            {updatingSite ? '更新中...' : '更新基本信息'}
          </Button>
        </div>
        
        <div className="space-y-4">
          <Input 
            label="站点 ID" 
            value={siteId} 
            disabled 
            readOnly
            layout="horizontal"
          />
          
          {site && (
            <>
              <Input 
                label="站点名称" 
                value={siteForm.name || ''} 
                onChange={e => setSiteForm({ ...siteForm, name: e.target.value })}
                layout="horizontal"
              />
              <Input 
                label="主域名" 
                value={siteForm.hostname || ''} 
                onChange={e => setSiteForm({ ...siteForm, hostname: e.target.value })}
                layout="horizontal"
              />
              <Input 
                label="绑定 IP" 
                value={siteForm.ip || ''} 
                onChange={e => setSiteForm({ ...siteForm, ip: e.target.value })}
                layout="horizontal"
              />
            </>
          )}
        </div>
      </div>

      <div className="border-t my-6" />

      <div>
        <div className="px-0 pt-0 mb-4">
          <h3 className="text-lg font-medium">数据面配置</h3>
        </div>
        <div className="space-y-4">
          <Input 
            label="HTTPS 监听地址"
            value={config.dataPlane?.httpsListenAddr || ''}
            onChange={e => updateConfig(prev => ({
              ...prev,
              dataPlane: { ...prev.dataPlane, httpsListenAddr: e.target.value }
            }))}
            placeholder=":443"
            layout="horizontal"
          />
        </div>
      </div>
    </div>
  )
}
