import { FiRotateCcw, FiTrash2, FiClock, FiUser } from 'react-icons/fi'
import type { ConfigVersion } from '../../../../admin/types'
import { Button } from '../../../../components/ui/Button'
import { Table } from '../../../../components/ui/Table'
import { Badge } from '../../../../components/ui/Badge'

type VersionHistoryTabProps = {
  versions: ConfigVersion[]
  handleRollback: (versionId: string) => Promise<void>
  handleDeleteVersion: (versionId: string) => Promise<void>
}

export function VersionHistoryTab({ versions, handleRollback, handleDeleteVersion }: VersionHistoryTabProps) {
  const columns = [
    {
      key: 'id',
      title: '版本 ID',
      render: (v: ConfigVersion) => (
        <span className="font-mono text-sm">{v.id.substring(0, 8)}</span>
      )
    },
    {
      key: 'created_at',
      title: '创建时间',
      render: (v: ConfigVersion) => (
        <div className="flex items-center gap-2 text-muted-foreground">
          <FiClock size={14} />
          <span>{new Date(v.created_at).toLocaleString()}</span>
        </div>
      )
    },
    {
      key: 'operator',
      title: '操作人',
      render: (v: ConfigVersion) => (
        <div className="flex items-center gap-2">
            <FiUser size={14} className="text-muted-foreground" />
            <Badge variant="default">{v.operator || 'unknown'}</Badge>
          </div>
      )
    },
    {
      key: 'actions',
      title: '操作',
      align: 'right' as const,
      render: (v: ConfigVersion) => (
        <div className="flex justify-end gap-2">
          <Button
            variant="secondary"
            size="sm"
            onClick={() => handleRollback(v.id)}
            icon={<FiRotateCcw />}
            title="回滚到此版本"
          >
            回滚
          </Button>
          <Button
            variant="danger"
            size="sm"
            onClick={() => handleDeleteVersion(v.id)}
            icon={<FiTrash2 />}
            title="删除此版本记录"
          />
        </div>
      )
    }
  ]

  return (
    <div className="space-y-6">
      <div className="space-y-6">
        <h3 className="text-lg font-medium">历史版本</h3>
        <div>
          {versions.length === 0 ? (
            <div className="p-8 text-center text-muted-foreground border rounded-lg border-dashed">
              暂无历史版本记录
            </div>
          ) : (
            <div className="border rounded-md overflow-hidden">
              <Table
                columns={columns}
                data={versions}
                rowKey="id"
              />
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
