import { useState } from 'react'
import { Download, Eye, FileBarChart, FileCode, FileText, Link as LinkIcon, Loader2, type LucideIcon } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ApiRequestError } from '@/api/client'
import { getTaskArtifactContent } from '@/api/tasks'
import { cn } from '@/lib/utils'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { toast } from 'sonner'
import { FileViewer } from '@/components/task/FileViewer'
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

function normalizeResultText(value: string | null | undefined) {
  if (!value) {
    return ''
  }

  return value
    .replace(/\\r\\n/g, '\n')
    .replace(/\\n/g, '\n')
    .replace(/\\r/g, '\r')
    .replace(/\\t/g, '\t')
}

interface TaskResultViewProps {
  taskId: string
  result: TaskResult
  artifacts: TaskArtifact[]
}

export function TaskResultView({ taskId, result, artifacts }: TaskResultViewProps) {
  const safeArtifacts = artifacts ?? []
  const summaryText = normalizeResultText(result.summary)
  const finalOutputText = normalizeResultText(result.final_output)
  const hasResult = summaryText || finalOutputText
  const hasArtifacts = safeArtifacts.length > 0
  const [loadingArtifactId, setLoadingArtifactId] = useState<string | null>(null)
  const [downloadingArtifactId, setDownloadingArtifactId] = useState<string | null>(null)
  const [transferErrors, setTransferErrors] = useState<Record<string, string>>({})
  const [viewerBlob, setViewerBlob] = useState<Blob | null>(null)
  const [viewerFileName, setViewerFileName] = useState('')
  const [viewerOpen, setViewerOpen] = useState(false)

  if (!hasResult && !hasArtifacts) {
    return <p className="py-8 text-center text-sm text-muted-foreground">任务尚未产出结果</p>
  }

  const handlePreview = async (artifact: TaskArtifact) => {
    setLoadingArtifactId(artifact.id)
    setTransferErrors((current) => ({ ...current, [artifact.id]: '' }))
    try {
      const fileBlob = await getTaskArtifactContent(taskId, artifact.id)
      const fileName = artifactFileName(artifact) || artifact.title || artifact.id
      setViewerBlob(fileBlob)
      setViewerFileName(fileName)
      setViewerOpen(true)
    } catch (err) {
      const message = err instanceof ApiRequestError ? err.message : '打开文件失败'
      toast.error(message)
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
      const fileName = artifactFileName(artifact) || `${artifact.title || artifact.id}`
      downloadBlob(fileBlob, fileName)
    } catch (err) {
      const message = err instanceof ApiRequestError ? err.message : '下载文件失败'
      toast.error(message)
      setTransferErrors((current) => ({ ...current, [artifact.id]: message }))
    } finally {
      setDownloadingArtifactId(null)
    }
  }

  const handleViewerDownload = () => {
    if (viewerBlob && viewerFileName) {
      downloadBlob(viewerBlob, viewerFileName)
    }
  }

  return (
    <div className="flex flex-col gap-4">
      {hasResult && (
        <Card>
          <CardContent className="flex flex-col gap-2 p-4">
            {summaryText && (
              <div>
                <h4 className="text-sm font-medium mb-1">摘要</h4>
                <p className="text-sm text-muted-foreground whitespace-pre-wrap">{summaryText}</p>
              </div>
            )}
            {finalOutputText && (
              <div>
                <h4 className="text-sm font-medium mb-1">最终产出</h4>
                <div className="rounded-md bg-muted p-3">
                  <div
                    className={cn(
                      'prose prose-sm max-w-none text-foreground dark:prose-invert',
                      'prose-p:my-2 prose-headings:my-3',
                      'prose-ul:my-2 prose-ol:my-2 prose-li:my-0.5',
                      'prose-pre:overflow-x-auto prose-pre:text-xs',
                      'prose-code:text-xs prose-table:text-xs'
                    )}
                  >
                    <ReactMarkdown remarkPlugins={[remarkGfm]}>
                      {finalOutputText}
                    </ReactMarkdown>
                  </div>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {hasArtifacts && (
        <div>
          <h4 className="text-sm font-medium mb-2">交付物</h4>
          <div className="flex flex-col gap-2">
            {safeArtifacts.map((artifact) => {
              const normalizedKind = normalizeArtifactKind(artifact.kind)
              const Icon = kindIcons[normalizedKind] ?? FileText
              const transferId = transferIdFromArtifact(artifact)
              const transferError = transferErrors[artifact.id]
              const fileName = artifactFileName(artifact)

              return (
                <Card key={artifact.id}>
                  <CardContent className="flex flex-col gap-3 p-3">
                    <div className="flex items-center gap-3">
                      <div className="flex size-8 items-center justify-center rounded-lg bg-muted">
                        <Icon className="size-4 text-muted-foreground" />
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
                          onClick={() => void handlePreview(artifact)}
                        >
                          {loadingArtifactId === artifact.id ? (
                            <>
                              <Loader2 className="size-3.5 animate-spin" />
                              加载中
                            </>
                          ) : (
                            <>
                              <Eye className="size-3.5" />
                              预览
                            </>
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
                              <Loader2 className="size-3.5 animate-spin" />
                              下载中
                            </>
                          ) : (
                            <>
                              <Download className="size-3.5" />
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
                  </CardContent>
                </Card>
              )
            })}
          </div>
        </div>
      )}

      <FileViewer
        open={viewerOpen}
        onOpenChange={setViewerOpen}
        blob={viewerBlob}
        fileName={viewerFileName}
        onDownload={handleViewerDownload}
      />
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
