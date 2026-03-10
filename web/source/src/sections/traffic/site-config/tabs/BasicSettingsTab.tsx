import type { Config, Site, SiteUpdateRequest } from '../../../../admin/types'

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
    <div className="panel">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
        <h4 style={{ margin: 0 }}>基础设置</h4>
        <button 
          className="button secondary small" 
          onClick={handleSaveSiteInfo}
          disabled={updatingSite || loading}
        >
          {updatingSite ? '更新中...' : '更新基本信息'}
        </button>
      </div>
      
      <div className="form-field">
        <label>站点 ID</label>
        <input className="input" defaultValue={siteId} disabled />
      </div>
      {site && (
        <>
          <div className="form-field">
            <label>站点名称</label>
            <input 
              className="input" 
              value={siteForm.name || ''} 
              onChange={e => setSiteForm({ ...siteForm, name: e.target.value })}
            />
          </div>
          <div className="form-field">
            <label>主域名</label>
            <input 
              className="input" 
              value={siteForm.hostname || ''} 
              onChange={e => setSiteForm({ ...siteForm, hostname: e.target.value })}
            />
          </div>
          <div className="form-field">
            <label>绑定 IP</label>
            <input 
              className="input" 
              value={siteForm.ip || ''} 
              onChange={e => setSiteForm({ ...siteForm, ip: e.target.value })}
            />
          </div>
        </>
      )}
      <div className="divider" />
      <h4>数据面配置</h4>
      <div className="form-field">
         <label>HTTPS 监听地址</label>
         <input 
            className="input"
            value={config.dataPlane?.httpsListenAddr || ''}
            onChange={e => updateConfig(prev => ({
              ...prev,
              dataPlane: { ...prev.dataPlane, httpsListenAddr: e.target.value }
            }))}
            placeholder=":443"
         />
      </div>
    </div>
  )
}
