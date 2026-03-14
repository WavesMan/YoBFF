import './SiteConfigDrawer.css'
import { useState } from 'react'
import {
  FiActivity,
  FiClock,
  FiGlobe,
  FiLock,
  FiSave,
  FiSettings,
  FiShield,
  FiZap,
} from 'react-icons/fi'
import { Drawer } from '../../../components/ui/Drawer'
import { Modal } from '../../../components/ui/Modal'
import { Button } from '../../../components/ui/Button'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '../../../components/ui/Tabs'

import { BasicSettingsTab } from './tabs/BasicSettingsTab'
import { ProxyRulesTab } from './tabs/ProxyRulesTab'
import { HttpsCertTab } from './tabs/HttpsCertTab'
import { SecurityTab } from './tabs/SecurityTab'
import { RateLimitTab } from './tabs/RateLimitTab'
import { VersionHistoryTab } from './tabs/VersionHistoryTab'
import { RealtimeLogTab } from './tabs/RealtimeLogTab'
import { useSiteConfig } from './useSiteConfig'

type SiteConfigDrawerProps = {
  token: string
  operator: string
  siteId: string
  onClose: () => void
}

export function SiteConfigDrawer({ token, operator, siteId, onClose }: SiteConfigDrawerProps) {
  const [activeTab, setActiveTab] = useState<'basic' | 'proxy' | 'security' | 'https' | 'limit' | 'version' | 'log'>('basic')
  
  const {
    site,
    config,
    versions,
    logStream,
    setLogStream,
    logs,
    setLogs,
    isLogConnected,
    setIsLogConnected,
    loading,
    saving,
    updatingSite,
    error,
    success,
    deleteVersionTarget,
    setDeleteVersionTarget,
    deletingVersion,
    rollbackVersionTarget,
    setRollbackVersionTarget,
    rollingBackVersion,
    newPool,
    setNewPool,
    newRoute,
    setNewRoute,
    siteForm,
    setSiteForm,
    cdnExpanded,
    setCdnExpanded,
    filteredCerts,
    showCertSelector,
    setShowCertSelector,
    handleSave,
    handleRollback,
    confirmRollbackVersion,
    handleDeleteVersion,
    confirmDeleteVersion,
    handleSaveSiteInfo,
    handleUpdateLogStream,
    updateConfig,
    handleAddPool,
    handleRemovePool,
    handleAddRoute,
    handleRemoveRoute,
    handleRemoveCert,
    loadLogHistory
  } = useSiteConfig(token, siteId, operator, activeTab)

  if (!siteId) return null

  return (
    <>
      <Drawer
        isOpen={!!siteId}
        onClose={onClose}
        title={`配置中心: ${site?.name || siteId}`}
        width="85%"
        bodyClassName="site-config-drawer-body"
        headerExtra={
          <Button 
            variant="primary"
            onClick={handleSave} 
            disabled={saving || loading}
            icon={<FiSave />}
          >
            {saving ? '保存中...' : '保存配置'}
          </Button>
        }
      >
        <div className="drawer-body">
          <Tabs 
            value={activeTab}
            defaultValue="basic"
            onValueChange={(v) => setActiveTab(v as typeof activeTab)} 
            className="flex-1 flex overflow-hidden site-config-tabs"
          >
            <div className="config-sidebar">
              <TabsList className="flex flex-col h-full w-full bg-transparent p-0 justify-start space-y-1">
                <TabsTrigger value="basic" className="config-menu-item">
                  <FiSettings /> 基础设置
                </TabsTrigger>
                <TabsTrigger value="proxy" className="config-menu-item">
                  <FiGlobe /> 流量池管理
                </TabsTrigger>
                <TabsTrigger value="https" className="config-menu-item">
                  <FiLock /> HTTPS 证书
                </TabsTrigger>
                <TabsTrigger value="security" className="config-menu-item">
                  <FiShield /> 安全防护
                </TabsTrigger>
                <TabsTrigger value="limit" className="config-menu-item">
                  <FiZap /> 流量限制
                </TabsTrigger>
                <TabsTrigger value="version" className="config-menu-item">
                  <FiClock /> 版本控制
                </TabsTrigger>
                <TabsTrigger value="log" className="config-menu-item">
                  <FiActivity /> 实时日志
                </TabsTrigger>
              </TabsList>
            </div>
          
            <div className="config-content">
              {loading && <div className="loading">加载配置中...</div>}
              {error && <div className="alert error">{error}</div>}
              {success && <div className="alert success">{success}</div>}
              
              {!loading && config && (
                <>
                  <TabsContent value="basic">
                    <BasicSettingsTab 
                      site={site}
                      siteId={siteId}
                      siteForm={siteForm}
                      setSiteForm={setSiteForm}
                      config={config}
                      updateConfig={updateConfig}
                      handleSaveSiteInfo={handleSaveSiteInfo}
                      updatingSite={updatingSite}
                      loading={loading}
                    />
                  </TabsContent>

                  <TabsContent value="proxy">
                    <ProxyRulesTab 
                      config={config}
                      updateConfig={updateConfig}
                      newPool={newPool}
                      setNewPool={setNewPool}
                      newRoute={newRoute}
                      setNewRoute={setNewRoute}
                      handleAddPool={handleAddPool}
                      handleRemovePool={handleRemovePool}
                      handleAddRoute={handleAddRoute}
                      handleRemoveRoute={handleRemoveRoute}
                    />
                  </TabsContent>
                  
                  <TabsContent value="https">
                    <HttpsCertTab 
                      site={site}
                      config={config}
                      updateConfig={updateConfig}
                      filteredCerts={filteredCerts}
                      handleRemoveCert={handleRemoveCert}
                    />
                  </TabsContent>

                  <TabsContent value="security">
                    <SecurityTab 
                      token={token}
                      siteId={siteId}
                      site={site}
                      config={config}
                      updateConfig={updateConfig}
                      cdnExpanded={cdnExpanded}
                      setCdnExpanded={setCdnExpanded}
                      filteredCerts={filteredCerts}
                      showCertSelector={showCertSelector}
                      setShowCertSelector={setShowCertSelector}
                    />
                  </TabsContent>

                  <TabsContent value="limit">
                    <RateLimitTab 
                      config={config}
                      updateConfig={updateConfig}
                    />
                  </TabsContent>

                  <TabsContent value="version">
                    <VersionHistoryTab 
                      versions={versions}
                      handleRollback={handleRollback}
                      handleDeleteVersion={handleDeleteVersion}
                    />
                  </TabsContent>
                  
                  <TabsContent value="log">
                    <RealtimeLogTab 
                      siteId={siteId}
                      logStream={logStream}
                      setLogStream={setLogStream}
                      logs={logs}
                      setLogs={setLogs}
                      isLogConnected={isLogConnected}
                      setIsLogConnected={setIsLogConnected}
                      loading={loading}
                      handleUpdateLogStream={handleUpdateLogStream}
                      loadLogHistory={loadLogHistory}
                    />
                  </TabsContent>
                </>
              )}
            </div>
          </Tabs>
        </div>
      </Drawer>

      <Modal
        isOpen={!!deleteVersionTarget}
        onClose={() => !deletingVersion && setDeleteVersionTarget(null)}
        title="确认删除"
        footer={
          <div className="flex justify-end gap-2">
            <Button
              variant="secondary"
              onClick={() => setDeleteVersionTarget(null)}
              disabled={deletingVersion}
            >
              取消
            </Button>
            <Button
              variant="danger"
              onClick={confirmDeleteVersion}
              disabled={deletingVersion}
            >
              {deletingVersion ? '删除中...' : '确定删除'}
            </Button>
          </div>
        }
      >
        <div className="p-4">
          确定要删除版本「{deleteVersionTarget?.shortID}」吗？此操作不可恢复。
        </div>
      </Modal>

      <Modal
        isOpen={!!rollbackVersionTarget}
        onClose={() => !rollingBackVersion && setRollbackVersionTarget(null)}
        title="确认回滚"
        footer={
          <div className="flex justify-end gap-2">
            <Button
              variant="secondary"
              onClick={() => setRollbackVersionTarget(null)}
              disabled={rollingBackVersion}
            >
              取消
            </Button>
            <Button
              variant="danger"
              onClick={confirmRollbackVersion}
              disabled={rollingBackVersion}
            >
              {rollingBackVersion ? '回滚中...' : '确定回滚'}
            </Button>
          </div>
        }
      >
        <div className="p-4">
          确定要回滚到版本「{rollbackVersionTarget?.shortID}」吗？当前未保存的修改将会丢失。
        </div>
      </Modal>
    </>
  )
}