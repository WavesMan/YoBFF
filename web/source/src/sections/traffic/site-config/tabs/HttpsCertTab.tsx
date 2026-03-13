import { FiCheck, FiTrash2 } from 'react-icons/fi'
import type { Config, Site, SSLCertificate, Certificate } from '../../../../admin/types'
import { Card, CardContent, CardHeader, CardTitle } from '../../../../components/ui/Card'
import { Button } from '../../../../components/ui/Button'
import { Table, type Column } from '../../../../components/ui/Table'
import { Badge } from '../../../../components/ui/Badge'

type HttpsCertTabProps = {
  site: Site | null
  config: Config
  updateConfig: (updater: (prev: Config) => Config) => void
  filteredCerts: SSLCertificate[]
  handleRemoveCert: (index: number) => void
}

export function HttpsCertTab({
  site,
  config,
  updateConfig,
  filteredCerts,
  handleRemoveCert,
}: HttpsCertTabProps) {
  const selectedCert = filteredCerts.find(cert => cert.id === config.dataPlane?.certId)

  const uploadedCertColumns = [
    {
      key: 'name',
      title: '备注名',
      render: (cert: SSLCertificate) => {
        const isSelected = config.dataPlane?.certId === cert.id
        return (
          <span className={isSelected ? 'font-semibold' : ''}>
            {cert.name} {isSelected && <Badge variant="default" className="ml-2">已选择</Badge>}
          </span>
        )
      }
    },
    {
      key: 'domains',
      title: '包含域名',
      render: (cert: SSLCertificate) => {
        const domains = cert.domains || []
        const display = domains.slice(0, 3).join(', ')
        const more = domains.length > 3 ? ` +${domains.length - 3}` : ''
        return <span className="text-muted-foreground">{display}{more}</span>
      }
    },
    {
      key: 'issuer',
      title: '颁发机构',
      render: (cert: SSLCertificate) => <span className="text-muted-foreground">{cert.issuer || '-'}</span>
    },
    {
      key: 'notAfter',
      title: '过期时间',
      render: (cert: SSLCertificate) => <span className="text-muted-foreground">{new Date(cert.notAfter).toLocaleDateString()}</span>
    },
    {
      key: 'actions',
      title: '操作',
      render: (cert: SSLCertificate) => {
        const isSelected = config.dataPlane?.certId === cert.id
        return (
          <div className="flex justify-end">
            <Button
              variant={isSelected ? 'secondary' : 'primary'}
              size="sm"
              disabled={isSelected}
              onClick={() => {
                updateConfig(prev => ({
                  ...prev,
                  dataPlane: {
                    ...prev.dataPlane,
                    enableHttps: true,
                    certId: cert.id,
                  },
                }))
              }}
              title={isSelected ? '已选择' : '选择此证书'}
              icon={isSelected ? <FiCheck /> : undefined}
            >
              {isSelected ? '已选择' : '选择'}
            </Button>
          </div>
        )
      }
    }
  ]

  const autoCertColumns: Column<Certificate>[] = [
    {
      key: 'domain',
      title: '域名',
      width: '80%'
    },
    {
      key: 'actions',
      title: '操作',
      width: '20%',
      render: (_: Certificate) => (
        <div className="flex justify-end">
          <Button
            variant="danger"
            size="sm"
            onClick={() => {
              // Note: We need the index but the Table component only passes the record.
              // This is a limitation of the current Table component implementation.
              // For now we will rely on finding the index by domain which should be unique here.
              const idx = config.certificates?.findIndex(c => c.domain === _.domain);
              if (idx !== undefined && idx !== -1) {
                handleRemoveCert(idx);
              }
            }}
            icon={<FiTrash2 />}
          />
        </div>
      )
    }
  ]

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>HTTPS 证书</CardTitle>
          <div className="text-sm text-muted-foreground mt-2">
            为站点 {site?.hostname || '-'} 选择已上传的 SSL 证书，并可配置自动申请证书的域名列表。
          </div>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="space-y-4">
            <h3 className="text-lg font-medium">已上传证书（匹配站点域名）</h3>
            {filteredCerts.length === 0 ? (
              <div className="p-8 text-center text-muted-foreground border rounded-lg border-dashed">
                没有找到匹配的证书，请先在证书管理中上传。
              </div>
            ) : (
              <Table
                columns={uploadedCertColumns}
                data={filteredCerts}
                rowKey="id"
              />
            )}
            <div className="text-sm text-muted-foreground bg-muted/30 p-3 rounded-md">
              当前状态：
              <span className={config.dataPlane?.enableHttps ? 'text-green-500 font-medium ml-2' : 'text-yellow-500 font-medium ml-2'}>
                {config.dataPlane?.enableHttps ? 'HTTPS 已启用' : 'HTTPS 未启用'}
              </span>
              {selectedCert 
                ? <span className="ml-2">，使用证书「{selectedCert.name}」</span>
                : config.dataPlane?.certId ? <span className="ml-2 text-red-400">，证书未匹配到站点域名</span> : ''}
            </div>
          </div>

          <div className="space-y-4 pt-4 border-t">
            <h3 className="text-lg font-medium">自动申请证书域名</h3>
            <Table
              columns={autoCertColumns}
              data={config.certificates || []}
              rowKey={(row: Certificate) => row.domain || ''}
              emptyText="暂无证书配置"
            />
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
