import { useCallback, useEffect, useState } from 'react'
import { FiGlobe, FiServer, FiSettings, FiTrash2, FiPlus } from 'react-icons/fi'
import { createSite, deleteSite, fetchSiteGroups } from '../../admin/api'
import type { Site, SiteCreateRequest, SiteGroup } from '../../admin/types'
import { Card, CardContent, CardHeader, CardTitle } from '../../components/ui/Card'
import { Button } from '../../components/ui/Button'
import { Select } from '../../components/ui/Select'
import { Badge } from '../../components/ui/Badge'
import { Table } from '../../components/ui/Table'
import { Drawer } from '../../components/ui/Drawer'
import { Input } from '../../components/ui/Input'
import { Modal } from '../../components/ui/Modal'
import { useToast } from '../../components/ui/Toast'

type SiteListProps = {
  token: string
  onSelectSite: (siteId: string) => void
}

export function SiteList({ token, onSelectSite }: SiteListProps) {
  const [grouping, setGrouping] = useState<'hostname' | 'ip'>('hostname')
  const [groups, setGroups] = useState<SiteGroup[]>([])
  const [loading, setLoading] = useState(false)
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<Site | null>(null)
  const [deleting, setDeleting] = useState(false)
  const { success, error: toastError } = useToast()

  const loadSites = useCallback(async () => {
    setLoading(true)
    try {
      const data = await fetchSiteGroups(token, grouping)
      setGroups(data.groups || [])
    } catch (err) {
      toastError(err instanceof Error ? err.message : '加载站点列表失败')
    } finally {
      setLoading(false)
    }
  }, [token, grouping, toastError])

  useEffect(() => {
    loadSites()
  }, [loadSites])

  const handleDelete = async () => {
    if (!deleteTarget) return
    setDeleting(true)
    try {
      await deleteSite(token, deleteTarget.id)
      success('站点删除成功')
      setDeleteTarget(null)
      loadSites()
    } catch (err) {
      toastError(err instanceof Error ? err.message : '删除失败')
    } finally {
      setDeleting(false)
    }
  }

  const columns = [
    { key: 'name', title: '站点名称', width: '25%' },
    { key: 'hostname', title: '域名', width: '30%' },
    { key: 'ip', title: 'IP', width: '25%' },
    {
      key: 'actions',
      title: '操作',
      width: '20%',
      render: (site: Site) => (
        <div className="flex justify-end gap-2">
          <Button
            variant="secondary"
            size="sm"
            onClick={() => onSelectSite(site.id)}
            title="配置"
            icon={<FiSettings />}
          >
            配置
          </Button>
          <Button
            variant="danger"
            size="sm"
            onClick={() => setDeleteTarget(site)}
            title="删除"
            icon={<FiTrash2 />}
          />
        </div>
      ),
    },
  ]

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>站点列表</CardTitle>
          <div className="flex gap-4 items-center">
            <div className="w-[200px]">
              <Select
                options={[
                  { label: '按域名分组', value: 'hostname' },
                  { label: '按 IP 分组', value: 'ip' },
                ]}
                value={grouping}
                onChange={(e) => setGrouping(e.target.value as 'hostname' | 'ip')}
              />
            </div>
            <Button onClick={() => setShowCreateModal(true)} icon={<FiPlus />}>
              添加站点
            </Button>
          </div>
        </CardHeader>
        
        <CardContent>
          {loading ? (
            <div className="p-8 text-center text-muted-foreground">加载中...</div>
          ) : groups.length > 0 ? (
            <div className="space-y-6">
              {groups.map((group) => (
                <div key={group.group_key} className="border rounded-md overflow-hidden">
                  <div className="bg-muted/50 px-4 py-3 flex items-center gap-3 border-b">
                    <span className="flex items-center gap-2 font-medium">
                      {grouping === 'hostname' ? <FiGlobe /> : <FiServer />}
                      {group.group_key || '未分类'}
                    </span>
                    <Badge variant="primary">{group.count}</Badge>
                  </div>
                  <Table
                    columns={columns}
                    data={group.sites}
                    rowKey="id"
                    className="border-0"
                  />
                </div>
              ))}
            </div>
          ) : (
            <div className="p-8 text-center text-muted-foreground">暂无站点</div>
          )}
        </CardContent>
      </Card>

      <CreateSiteDrawer
        token={token}
        isOpen={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        onCreated={() => {
          setShowCreateModal(false)
          loadSites()
        }}
      />

      <Modal
        isOpen={!!deleteTarget}
        onClose={() => !deleting && setDeleteTarget(null)}
        title="确认删除"
        footer={
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setDeleteTarget(null)} disabled={deleting}>
              取消
            </Button>
            <Button variant="danger" onClick={handleDelete} disabled={deleting}>
              {deleting ? '删除中...' : '确定删除'}
            </Button>
          </div>
        }
      >
        <div className="py-2 text-muted-foreground">
          确定要删除站点「{deleteTarget?.name || deleteTarget?.hostname}」吗？此操作不可恢复。
        </div>
      </Modal>
    </div>
  )
}

function CreateSiteDrawer({ 
  token, 
  isOpen, 
  onClose, 
  onCreated 
}: { 
  token: string
  isOpen: boolean
  onClose: () => void
  onCreated: () => void 
}) {
  const [form, setForm] = useState<SiteCreateRequest>({
    name: '',
    hostname: '',
    ip: '',
  })
  const [loading, setLoading] = useState(false)
  const { success, error: toastError } = useToast()

  // Reset form when opening
  useEffect(() => {
    if (isOpen) {
      setForm({ name: '', hostname: '', ip: '' })
    }
  }, [isOpen])

  const handleSubmit = async () => {
    if (!form.name || !form.hostname) {
      toastError('名称和域名为必填项')
      return
    }
    setLoading(true)
    try {
      await createSite(token, form)
      success('站点创建成功')
      onCreated()
    } catch (err) {
      toastError(err instanceof Error ? err.message : '创建站点失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Drawer
      isOpen={isOpen}
      onClose={onClose}
      title="添加新站点"
      footer={
        <div className="flex justify-end gap-2 w-full">
          <Button variant="secondary" onClick={onClose} disabled={loading}>
            取消
          </Button>
          <Button onClick={handleSubmit} disabled={loading}>
            {loading ? '创建中...' : '创建'}
          </Button>
        </div>
      }
    >
      <div className="space-y-4 py-4">
        <Input
          label="站点名称"
          value={form.name}
          onChange={(e) => setForm({ ...form, name: e.target.value })}
          placeholder="Example Site"
        />
        <Input
          label="域名"
          value={form.hostname}
          onChange={(e) => setForm({ ...form, hostname: e.target.value })}
          placeholder="example.com"
        />
        <Input
          label="IP 地址 (可选)"
          value={form.ip || ''}
          onChange={(e) => setForm({ ...form, ip: e.target.value })}
          placeholder="192.168.1.100"
        />
      </div>
    </Drawer>
  )
}


