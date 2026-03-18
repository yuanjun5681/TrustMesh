import { useState } from 'react'
import { Download, ExternalLink, FileBarChart, FileCode, FileText, Link as LinkIcon, Loader2, type LucideIcon } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ApiRequestError } from '@/api/client'
import { getTaskArtifactContent, getTaskArtifactTransfer } from '@/api/tasks'
import type { TaskResult, TaskArtifact, TransferDetail } from '@/types'

const kindIcons: Record<string, LucideIcon> = {
  file: FileText,
  link: LinkIcon,
  log: FileCode,
  report: FileBarChart,
}

function normalizeArtifactKind(kind: string | null | undefined) {
  const normalizedKind = kind?.trim().toLowerCase()
  if (!normalizedKind) {
    return 'unknown'
  }
  return normalizedKind
}

interface TaskResultViewProps {
  taskId: string
  result: TaskResult
  artifacts: TaskArtifact[]
}

export function TaskResultView({ taskId, result, artifacts }: TaskResultViewProps) {
  const safeArtifacts = artifacts ?? []
  const hasResult = result.summary || result.final_output
  const hasArtifacts = safeArtifacts.length > 0
  const [loadingArtifactId, setLoadingArtifactId] = useState<string | null>(null)
  const [downloadingArtifactId, setDownloadingArtifactId] = useState<string | null>(null)
  const [transferDetails, setTransferDetails] = useState<Record<string, TransferDetail>>({})
  const [transferErrors, setTransferErrors] = useState<Record<string, string>>({})

  if (!hasResult && !hasArtifacts) {
    return <p className="py-8 text-center text-sm text-muted-foreground">任务尚未产出结果</p>
  }

  const handleOpenTransfer = async (artifact: TaskArtifact) => {
    setLoadingArtifactId(artifact.id)
    setTransferErrors((current) => ({ ...current, [artifact.id]: '' }))
    try {
      const res = await getTaskArtifactTransfer(taskId, artifact.id)
      const detail = res.data
      setTransferDetails((current) => ({ ...current, [artifact.id]: detail }))
      const fileBlob = await getTaskArtifactContent(taskId, artifact.id)
      const objectURL = URL.createObjectURL(fileBlob)
      window.open(objectURL, '_blank', 'noopener,noreferrer')
      window.setTimeout(() => URL.revokeObjectURL(objectURL), 60_000)
    } catch (err) {
      const message = err instanceof ApiRequestError ? err.message : '加载传输详情失败'
      setTransferErrors((current) => ({ ...current, [artifact.id]: message }))
    } finally {
      setLoadingArtifactId(null)
    }
  }

  const handleDownloadArtifact = async (artifact: TaskArtifact) => {
    setDownloadingArtifactId(artifact.id)
    setTransferErrors((current) => ({ ...current, [artifact.id]: '' }))
    try {
      const fileBlob = await getTaskArtifactContent(taskId, artifact.id)
      const fileName = artifactFileName(artifact, transferDetails[artifact.id]) || `${artifact.title || artifact.id}`
      downloadBlob(fileBlob, fileName)
    } catch (err) {
      const message = err instanceof ApiRequestError ? err.message : '下载文件失败'
      setTransferErrors((current) => ({ ...current, [artifact.id]: message }))
    } finally {
      setDownloadingArtifactId(null)
    }
  }

  return (
    <div className="space-y-4">
      {hasResult && (
        <Card>
          <CardContent className="p-4 space-y-2">
            {result.summary && (
              <div>
                <h4 className="text-sm font-medium mb-1">摘要</h4>
                <p className="text-sm text-muted-foreground whitespace-pre-wrap">{result.summary}</p>
              </div>
            )}
            {result.final_output && (
              <div>
                <h4 className="text-sm font-medium mb-1">最终产出</h4>
                <div className="rounded-md bg-muted p-3 text-sm whitespace-pre-wrap font-mono text-xs">
                  {result.final_output}
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {hasArtifacts && (
        <div>
          <h4 className="text-sm font-medium mb-2">交付物</h4>
          <div className="space-y-2">
            {safeArtifacts.map((artifact) => {
              const normalizedKind = normalizeArtifactKind(artifact.kind)
              const Icon = kindIcons[normalizedKind] ?? FileText
              const transferId = transferIdFromArtifact(artifact)
              const transferDetail = transferDetails[artifact.id]
              const transferError = transferErrors[artifact.id]
              const fileName = artifactFileName(artifact, transferDetail)

              return (
                <Card key={artifact.id}>
                  <CardContent className="p-3 space-y-3">
                    <div className="flex items-center gap-3">
                      <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-muted">
                        <Icon className="h-4 w-4 text-muted-foreground" />
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium truncate">{artifact.title}</p>
                        <p className="text-xs text-muted-foreground truncate">{fileName || artifact.uri || transferId || '无直接 URI'}</p>
                      </div>
                      <Badge variant="secondary" className="text-xs shrink-0">
                        {normalizedKind}
                      </Badge>
                    </div>

                    {transferId && (
                      <div className="flex items-center gap-2">
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          disabled={loadingArtifactId === artifact.id}
                          onClick={() => void handleOpenTransfer(artifact)}
                        >
                          {loadingArtifactId === artifact.id ? (
                            <>
                              <Loader2 className="h-3.5 w-3.5 animate-spin" />
                              加载中
                            </>
                          ) : transferDetail ? (
                            <>
                              <ExternalLink className="h-3.5 w-3.5" />
                              打开文件
                            </>
                          ) : (
                            '查看传输'
                          )}
                        </Button>
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          disabled={downloadingArtifactId === artifact.id}
                          onClick={() => void handleDownloadArtifact(artifact)}
                        >
                          {downloadingArtifactId === artifact.id ? (
                            <>
                              <Loader2 className="h-3.5 w-3.5 animate-spin" />
                              下载中
                            </>
                          ) : (
                            <>
                              <Download className="h-3.5 w-3.5" />
                              下载
                            </>
                          )}
                        </Button>
                        <span className="text-xs text-muted-foreground">transfer: {transferId}</span>
                      </div>
                    )}

                    {transferError && (
                      <p className="text-xs text-destructive">{transferError}</p>
                    )}

                    {transferDetail && (
                      <div className="rounded-md bg-muted/50 p-3 space-y-1 text-xs text-muted-foreground">
                        {renderTransferLine('大小', firstString(transferDetail, ['size']))}
                        {renderTransferLine('校验', firstString(transferDetail, ['checksum']))}
                        {renderTransferLine('Bucket', firstString(transferDetail, ['bucket']))}
                        {renderTransferLine('状态', firstString(transferDetail, ['status']))}
                        {!firstString(transferDetail, ['size', 'checksum', 'bucket', 'status']) && (
                          <pre className="whitespace-pre-wrap break-all text-[11px] text-foreground/80">
                            {JSON.stringify(transferDetail, null, 2)}
                          </pre>
                        )}
                      </div>
                    )}
                  </CardContent>
                </Card>
              )
            })}
          </div>
        </div>
      )}
    </div>
  )
}

function transferIdFromArtifact(artifact: TaskArtifact) {
  const directTransferID = firstString(artifact.metadata as TransferDetail, ['transfer_id'])
  if (directTransferID) {
    return directTransferID
  }
  const nestedTransfer = artifact.metadata?.transfer
  if (nestedTransfer && typeof nestedTransfer === 'object' && nestedTransfer !== null) {
    return firstString(nestedTransfer as TransferDetail, ['transfer_id', 'transferId'])
  }
  if (artifact.uri.startsWith('transfer://')) {
    return artifact.uri.replace('transfer://', '')
  }
  return ''
}

function artifactFileName(artifact: TaskArtifact, transferDetail?: TransferDetail) {
  return (
    firstString(artifact.metadata as TransferDetail, ['file_name']) ||
    firstString(artifact.metadata?.transfer as TransferDetail | undefined, ['fileName', 'file_name']) ||
    firstString(transferDetail, ['fileName', 'file_name'])
  )
}

function firstString(obj: TransferDetail | undefined, keys: string[]) {
  if (!obj) return ''
  for (const key of keys) {
    const value = obj[key]
    if (typeof value === 'string' && value.trim()) {
      return value.trim()
    }
    if (typeof value === 'number') {
      return String(value)
    }
  }
  return ''
}

function renderTransferLine(label: string, value: string) {
  if (!value) return null
  return (
    <p>
      <span className="font-medium text-foreground/80">{label}:</span> {value}
    </p>
  )
}

function downloadBlob(blob: Blob, fileName: string) {
  const objectURL = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = objectURL
  anchor.download = fileName
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
  window.setTimeout(() => URL.revokeObjectURL(objectURL), 60_000)
}
