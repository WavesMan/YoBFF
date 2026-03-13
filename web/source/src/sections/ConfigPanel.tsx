import { FiCheck, FiClock, FiCpu, FiPlay, FiRefreshCw, FiSearch } from 'react-icons/fi'
import type { Config, ConfigVersion, ValidationIssue } from '../admin/types'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/Card'
import { Button } from '../components/ui/Button'
import { Input } from '../components/ui/Input'
import { Select } from '../components/ui/Select'
import { Badge } from '../components/ui/Badge'
import { Table } from '../components/ui/Table'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '../components/ui/Tabs'

type ConfigView = 'apply' | 'dry-run' | 'rollback'

type ConfigPanelProps = {
  configDraft: Config
  configSnapshot: Config | null
  advancedMode: boolean
  jsonDraft: string
  loadingApplying: boolean
  loadingValidating: boolean
  loadingVersions: boolean
  view: ConfigView
  operator: string
  validationIssues: ValidationIssue[]
  validationTime: string
  configVersions: ConfigVersion[]
  diffCurrentJson: string
  diffDraftJson: string
  diffChanged: boolean
  onViewChange: (view: ConfigView) => void
  onOperatorChange: (value: string) => void
  onToggleMode: (isAdvanced: boolean) => void
  onJsonChange: (value: string) => void
  onConfigChange: (next: Config) => void
  onApply: () => void
  onReset: () => void
  onValidate: () => void
  onRefreshVersions: () => void
  onRollback: (versionID: string) => void
}

/**
 * 配置管理面板组件
 * 
 * 提供配置的应用、预检（Dry-run）和版本回滚功能。
 * 
 * @param props.configDraft - 当前编辑中的配置草稿
 * @param props.configSnapshot - 上一次成功应用的配置快照，用于对比和恢复
 * @param props.advancedMode - 是否处于高级 JSON 编辑模式
 * @param props.jsonDraft - 高级模式下的 JSON 字符串
 * @param props.loadingApplying - 应用配置操作的加载状态
 * @param props.loadingValidating - 预检操作的加载状态
 * @param props.loadingVersions - 版本列表加载状态
 * @param props.view - 当前激活的视图标签 ('apply' | 'dry-run' | 'rollback')
 * @param props.operator - 当前操作人名称
 * @param props.validationIssues - 预检发现的问题列表
 * @param props.validationTime - 上次预检的时间戳
 * @param props.configVersions - 历史配置版本列表
 * @param props.diffCurrentJson - 差异对比：当前配置 JSON
 * @param props.diffDraftJson - 差异对比：草稿配置 JSON
 * @param props.diffChanged - 是否存在配置差异
 * @param props.onViewChange - 切换视图回调
 * @param props.onOperatorChange - 修改操作人回调
 * @param props.onToggleMode - 切换基础/高级模式回调
 * @param props.onJsonChange - 修改 JSON 内容回调
 * @param props.onConfigChange - 修改配置对象回调
 * @param props.onApply - 执行应用配置回调
 * @param props.onReset - 执行恢复默认回调
 * @param props.onValidate - 执行预检回调
 * @param props.onRefreshVersions - 刷新版本列表回调
 * @param props.onRollback - 执行回滚回调
 */
export function ConfigPanel({
  configDraft,
  configSnapshot,
  advancedMode,
  jsonDraft,
  loadingApplying,
  loadingValidating,
  loadingVersions,
  view,
  operator,
  validationIssues,
  validationTime,
  configVersions,
  diffCurrentJson,
  diffDraftJson,
  diffChanged,
  onViewChange,
  onOperatorChange,
  onToggleMode,
  onJsonChange,
  onConfigChange,
  onApply,
  onReset,
  onValidate,
  onRefreshVersions,
  onRollback,
}: ConfigPanelProps) {
  return (
    <Card className="w-full">
      <CardHeader className="flex flex-row items-center justify-between pb-4 border-b border-white/5">
        <div>
          <CardTitle className="text-xl font-bold">配置管理</CardTitle>
          <p className="text-sm text-muted-foreground mt-1">配置预检、差异比对与版本回滚入口</p>
        </div>
        <Badge variant="default" className="font-mono">/admin/api/v1/config</Badge>
      </CardHeader>

      <CardContent className="pt-6">
        <div className="space-y-6">
          {/* 操作人设置 */}
          <div className="max-w-xs">
            <Input
              label="操作人"
              value={operator}
              onChange={(event) => onOperatorChange(event.target.value)}
              placeholder="请输入操作人名称"
              icon={<FiSearch />}
            />
          </div>

          {/* 视图切换 */}
          <Tabs
            value={view}
            onValueChange={(v) => onViewChange(v as ConfigView)}
            defaultValue="apply"
            className="w-full"
          >
            <TabsList className="grid w-full grid-cols-3 mb-8">
              <TabsTrigger value="apply" className="flex items-center justify-center">
                <FiPlay className="mr-2" /> 配置应用
              </TabsTrigger>
              <TabsTrigger value="dry-run" className="flex items-center justify-center">
                <FiCpu className="mr-2" /> 预检与差异
              </TabsTrigger>
              <TabsTrigger value="rollback" className="flex items-center justify-center">
                <FiClock className="mr-2" /> 版本回滚
              </TabsTrigger>
            </TabsList>

            <TabsContent value="apply">
              <div className="space-y-6 mt-6">
                <div className="flex items-center justify-between">
                  <div className="flex gap-2">
                    <Button
                      variant={advancedMode ? 'secondary' : 'primary'}
                      onClick={() => onToggleMode(false)}
                      size="sm"
                    >
                      基础模式
                    </Button>
                    <Button
                      variant={advancedMode ? 'primary' : 'secondary'}
                      onClick={() => onToggleMode(true)}
                      size="sm"
                    >
                      高级 JSON
                    </Button>
                  </div>
                  <Badge variant="default">PUT /admin/api/v1/config</Badge>
                </div>

                {advancedMode ? (
                  <div className="relative">
                    <textarea 
                      className="w-full h-96 bg-black/50 border border-white/10 rounded-lg p-4 font-mono text-sm focus:outline-none focus:border-primary/50 transition-colors"
                      value={jsonDraft} 
                      onChange={(event) => onJsonChange(event.target.value)} 
                    />
                  </div>
                ) : (
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-6 p-6 border border-white/10 rounded-lg bg-white/5">
                    <Input
                      label="HTTP 监听地址"
                      value={configDraft.dataPlane?.httpListenAddr || ''}
                      onChange={(e) => onConfigChange({
                        ...configDraft,
                        dataPlane: { ...configDraft.dataPlane, httpListenAddr: e.target.value }
                      })}
                    />
                    <Input
                      label="HTTPS 监听地址"
                      value={configDraft.dataPlane?.httpsListenAddr || ''}
                      onChange={(e) => onConfigChange({
                        ...configDraft,
                        dataPlane: { ...configDraft.dataPlane, httpsListenAddr: e.target.value }
                      })}
                    />
                    <Input
                      label="控制面地址"
                      value={configDraft.controlPlane?.adminListenAddr || ''}
                      onChange={(e) => onConfigChange({
                        ...configDraft,
                        controlPlane: { ...configDraft.controlPlane, adminListenAddr: e.target.value }
                      })}
                    />
                    <Select
                      label="HSTS"
                      value={configDraft.security?.enableHsts ? 'true' : 'false'}
                      onChange={(e) => onConfigChange({
                        ...configDraft,
                        security: { ...configDraft.security, enableHsts: e.target.value === 'true' }
                      })}
                      options={[
                        { label: '启用', value: 'true' },
                        { label: '关闭', value: 'false' },
                      ]}
                    />
                  </div>
                )}

                <div className="flex gap-3">
                  <Button variant="primary" onClick={onApply} icon={<FiCheck />} loading={loadingApplying}>
                    应用配置
                  </Button>
                  <Button variant="secondary" onClick={onReset} disabled={!configSnapshot}>
                    恢复默认
                  </Button>
                </div>
              </div>
            </TabsContent>

            <TabsContent value="dry-run">
              <div className="space-y-6 mt-6">
                <div className="flex items-center justify-between p-4 bg-white/5 border border-white/10 rounded-lg">
                  <div className="flex flex-col gap-1">
                    <div className="flex items-center gap-2">
                      <Badge variant="default">POST /admin/api/v1/config/validate</Badge>
                      {validationTime && (
                        <span className="text-xs text-muted-foreground">最近预检：{validationTime}</span>
                      )}
                    </div>
                    <p className="text-sm text-muted-foreground mt-1">
                      {diffChanged ? '检测到配置差异' : '当前与待应用配置一致'}
                    </p>
                  </div>
                  <Button variant="primary" onClick={onValidate} loading={loadingValidating}>
                    执行预检
                  </Button>
                </div>

                {validationIssues.length > 0 ? (
                  <div className="space-y-4">
                    <div className="p-3 bg-destructive/10 border border-destructive/20 rounded-md text-destructive text-sm font-medium">
                      预检失败，共 {validationIssues.length} 项问题
                    </div>
                    <Table
                      columns={[
                        { key: 'path', title: '路径', render: (issue: ValidationIssue) => issue.path },
                        { key: 'message', title: '说明', render: (issue: ValidationIssue) => issue.message },
                      ]}
                      data={validationIssues}
                    />
                  </div>
                ) : (
                  validationTime && (
                    <div className="p-3 bg-green-500/10 border border-green-500/20 rounded-md text-green-500 text-sm font-medium">
                      预检通过，配置格式正确
                    </div>
                  )
                )}

                <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                  <div className="space-y-2">
                    <Badge variant="default">当前配置</Badge>
                    <pre className="bg-black/90 text-green-400 font-mono text-xs p-4 rounded-lg h-80 overflow-auto border border-white/5">
                      {diffCurrentJson}
                    </pre>
                  </div>
                  <div className="space-y-2">
                    <Badge variant="default">待应用配置</Badge>
                    <pre className="bg-black/90 text-blue-400 font-mono text-xs p-4 rounded-lg h-80 overflow-auto border border-white/5">
                      {diffDraftJson}
                    </pre>
                  </div>
                </div>
              </div>
            </TabsContent>

            <TabsContent value="rollback">
              <div className="space-y-6 mt-6">
                <div className="flex justify-between items-center">
                  <div className="flex gap-2">
                    <Badge variant="default">GET /admin/api/v1/config/versions</Badge>
                    <Badge variant="default">POST /admin/api/v1/config/rollback</Badge>
                  </div>
                  <Button variant="secondary" size="sm" onClick={onRefreshVersions} icon={<FiRefreshCw className={loadingVersions ? 'animate-spin' : ''} />}>
                    刷新列表
                  </Button>
                </div>

                <Table
                  columns={[
                    { key: 'created_at', title: '时间', render: (item: ConfigVersion) => item.created_at },
                    { key: 'operator', title: '操作人', render: (item: ConfigVersion) => item.operator || '-' },
                    { key: 'source', title: '来源', render: (item: ConfigVersion) => item.source || '-' },
                    { key: 'id', title: '版本 ID', render: (item: ConfigVersion) => <span className="font-mono text-xs">{item.id}</span> },
                    { 
                      key: 'actions',
                      title: '操作', 
                      render: (item: ConfigVersion) => (
                        <Button variant="secondary" size="sm" onClick={() => onRollback(item.id)}>
                          回滚
                        </Button>
                      ),
                      // align: 'right' // Table component doesn't support align prop on Column based on interface read
                    },
                  ]}
                  data={configVersions}
                />
                
                {configVersions.length === 0 && !loadingVersions && (
                  <div className="text-center py-12 border border-dashed border-white/10 rounded-lg text-muted-foreground">
                    暂无历史版本记录
                  </div>
                )}
              </div>
            </TabsContent>
          </Tabs>
        </div>
      </CardContent>
    </Card>
  )
}
