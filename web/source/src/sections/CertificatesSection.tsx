import { useCallback, useEffect, useState } from 'react'
import {
  FiClock,
  FiLock,
  FiPlus,
  FiTrash2,
  FiFile
} from 'react-icons/fi'
import {
  deleteCertificate,
  fetchCertificates,
  uploadCertificate
} from '../admin/api'
import type { SSLCertificate } from '../admin/types'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/Card'
import { Button } from '../components/ui/Button'
import { Table } from '../components/ui/Table'
import { Badge } from '../components/ui/Badge'
import { Modal } from '../components/ui/Modal'
import { Input } from '../components/ui/Input'
import { useToast } from '../components/ui/Toast'

type CertificatesSectionProps = {
  token: string
}

export function CertificatesSection({ token }: CertificatesSectionProps) {
  const [certs, setCerts] = useState<SSLCertificate[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)
  const [showUpload, setShowUpload] = useState(false)
  const [uploadName, setUploadName] = useState('')
  const [certFile, setCertFile] = useState<File | null>(null)
  const [keyFile, setKeyFile] = useState<File | null>(null)
  const [uploading, setUploading] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<SSLCertificate | null>(null)
  const [deleting, setDeleting] = useState(false)
  const { success, error: toastError } = useToast()

  const loadCertificates = useCallback(async () => {
    setLoading(true)
    try {
      const data = await fetchCertificates(token, page, 15)
      setCerts(data.items || [])
      setTotal(data.total || 0)
    } catch (err) {
      toastError(err instanceof Error ? err.message : '加载证书列表失败')
    } finally {
      setLoading(false)
    }
  }, [token, page, toastError])

  useEffect(() => {
    loadCertificates()
  }, [loadCertificates])

  const confirmDeleteCertificate = async () => {
    if (!deleteTarget) return
    setDeleting(true)
    try {
      await deleteCertificate(token, deleteTarget.id)
      success('证书删除成功')
      setDeleteTarget(null)
      loadCertificates()
    } catch (err) {
      toastError(err instanceof Error ? err.message : '删除证书失败')
    } finally {
      setDeleting(false)
    }
  }

  const closeUploadDialog = () => {
    if (uploading) return
    setShowUpload(false)
    setUploadName('')
    setCertFile(null)
    setKeyFile(null)
  }

  const handleUpload = async () => {
    if (!uploadName || !certFile || !keyFile) {
      toastError('请完整填写信息')
      return
    }

    setUploading(true)
    const formData = new FormData()
    formData.append('name', uploadName)
    formData.append('cert', certFile)
    formData.append('key', keyFile)

    try {
      await uploadCertificate(token, formData)
      success('证书上传成功')
      closeUploadDialog()
      loadCertificates()
    } catch (err) {
      toastError(err instanceof Error ? err.message : '上传失败')
    } finally {
      setUploading(false)
    }
  }

  const columns = [
    {
      key: 'name',
      title: '备注名',
      render: (cert: SSLCertificate) => (
        <div className="flex items-center gap-2">
          <FiLock className="text-muted-foreground" />
          <span className="font-medium">{cert.name}</span>
        </div>
      )
    },
    {
      key: 'domains',
      title: '包含域名',
      render: (cert: SSLCertificate) => (
        <div className="flex flex-wrap gap-1">
          {(cert.domains || []).slice(0, 3).map(d => (
            <Badge key={d} variant="default" className="text-xs">{d}</Badge>
          ))}
          {(cert.domains || []).length > 3 && (
            <Badge variant="default" className="text-xs">+{cert.domains.length - 3}</Badge>
          )}
        </div>
      )
    },
    { key: 'issuer', title: '颁发机构', render: (cert: SSLCertificate) => <span className="text-muted-foreground">{cert.issuer}</span> },
    {
      key: 'expiry',
      title: '过期时间',
      render: (cert: SSLCertificate) => {
        const expiry = new Date(cert.notAfter)
        const isExpired = expiry < new Date()
        const isNearExpiry = expiry < new Date(Date.now() + 30 * 24 * 3600 * 1000)
        
        return (
          <div className={`flex items-center gap-1 ${isExpired ? 'text-red-500' : isNearExpiry ? 'text-yellow-500' : ''}`}>
            <FiClock size={14} />
            {expiry.toLocaleDateString()}
          </div>
        )
      }
    },
    {
      key: 'createdAt',
      title: '创建时间',
      render: (cert: SSLCertificate) => <span className="text-muted-foreground">{new Date(cert.createdAt).toLocaleDateString()}</span>
    },
    {
      key: 'actions',
      title: '操作',
      width: '100px',
      render: (cert: SSLCertificate) => (
        <div className="flex justify-end">
          <Button 
            variant="danger" 
            size="sm" 
            onClick={() => setDeleteTarget(cert)}
            title="删除证书"
            icon={<FiTrash2 />}
          />
        </div>
      )
    }
  ]

  const totalPages = Math.ceil(total / 15)

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <div>
            <CardTitle>HTTPS 证书管理</CardTitle>
            <div className="text-sm text-muted-foreground mt-1">
              统一管理 SSL/TLS 证书，支持自动续期检测
            </div>
          </div>
          <Button onClick={() => setShowUpload(true)} icon={<FiPlus />}>
            上传证书
          </Button>
        </CardHeader>

        <CardContent>
          <Table
            columns={columns}
            data={certs}
            loading={loading}
            emptyText="暂无证书，请点击上方按钮上传"
            rowKey="id"
          />

          {total > 15 && (
            <div className="flex justify-center items-center gap-4 py-4 mt-4 border-t">
              <Button 
                variant="ghost" 
                size="sm" 
                disabled={page === 1}
                onClick={() => setPage(p => p - 1)}
              >
                上一页
              </Button>
              <span className="text-sm text-muted-foreground">第 {page} 页 / 共 {totalPages} 页</span>
              <Button 
                variant="ghost" 
                size="sm"
                disabled={page === totalPages}
                onClick={() => setPage(p => p + 1)}
              >
                下一页
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      <Modal
        isOpen={showUpload}
        onClose={closeUploadDialog}
        title="上传证书"
        footer={
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={closeUploadDialog} disabled={uploading}>
              取消
            </Button>
            <Button onClick={handleUpload} disabled={uploading}>
              {uploading ? '上传中...' : '确认上传'}
            </Button>
          </div>
        }
      >
        <div className="space-y-4 py-2">
          <div className="text-sm text-muted-foreground mb-4">
            填写证书信息后点击“确认上传”。
          </div>
          <Input
            label="备注名称"
            value={uploadName}
            onChange={e => setUploadName(e.target.value)}
            placeholder="例如：example.com 2025"
            autoFocus
            disabled={uploading}
          />
          
          <div className="space-y-2">
            <label className="text-sm font-medium">证书文件 (.pem/.crt/.cer)</label>
            <div className="relative">
              <input
                type="file"
                id="cert-file"
                accept=".pem,.crt,.cer"
                onChange={e => setCertFile(e.target.files?.[0] || null)}
                className="hidden"
                disabled={uploading}
                style={{ display: 'none' }}
              />
              <label 
                htmlFor="cert-file" 
                className="flex items-center gap-2 px-3 py-2 border rounded-md cursor-pointer hover:bg-muted/50 transition-colors"
              >
                <FiFile className="text-muted-foreground" />
                <span className={certFile ? 'text-foreground' : 'text-muted-foreground'}>
                  {certFile ? certFile.name : '选择证书文件...'}
                </span>
              </label>
            </div>
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">私钥文件 (.key/.pem)</label>
            <div className="relative">
              <input
                type="file"
                id="key-file"
                accept=".key,.pem"
                onChange={e => setKeyFile(e.target.files?.[0] || null)}
                className="hidden"
                disabled={uploading}
                style={{ display: 'none' }}
              />
              <label 
                htmlFor="key-file" 
                className="flex items-center gap-2 px-3 py-2 border rounded-md cursor-pointer hover:bg-muted/50 transition-colors"
              >
                <FiFile className="text-muted-foreground" />
                <span className={keyFile ? 'text-foreground' : 'text-muted-foreground'}>
                  {keyFile ? keyFile.name : '选择私钥文件...'}
                </span>
              </label>
            </div>
          </div>
        </div>
      </Modal>

      <Modal
        isOpen={!!deleteTarget}
        onClose={() => !deleting && setDeleteTarget(null)}
        title="确认删除"
        footer={
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setDeleteTarget(null)} disabled={deleting}>
              取消
            </Button>
            <Button variant="danger" onClick={confirmDeleteCertificate} disabled={deleting}>
              {deleting ? '删除中...' : '确定删除'}
            </Button>
          </div>
        }
      >
        <div className="py-2 text-muted-foreground">
          确定要删除证书「{deleteTarget?.name}」吗？此操作不可恢复，且将影响所有使用此证书的站点。
        </div>
      </Modal>
    </div>
  )
}
