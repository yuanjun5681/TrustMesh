import { useState, useRef } from 'react'
import {
  FileText,
  Upload,
  Trash2,
  RefreshCw,
  Search,
  Tag,
  ChevronDown,
  ChevronUp,
  Loader2,
  CheckCircle2,
  XCircle,
  Clock,
} from 'lucide-react'
import { toast } from 'sonner'
import { PageContainer } from '@/components/layout/PageContainer'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { EmptyState } from '@/components/shared/EmptyState'
import {
  useKnowledgeDocs,
  useUploadDocument,
  useDeleteDocument,
  useReprocessDocument,
  useKnowledgeSearch,
  useKnowledgeChunks,
} from '@/hooks/useKnowledge'
import { cn, formatRelativeTime } from '@/lib/utils'
import type { KnowledgeDocument, KnowledgeSearchResult } from '@/types'

const STATUS_FILTERS = [
  { value: 'all', label: '全部' },
  { value: 'ready', label: '已就绪' },
  { value: 'processing', label: '处理中' },
  { value: 'failed', label: '失败' },
] as const

function StatusIcon({ status }: { status: string }) {
  switch (status) {
    case 'ready':
      return <CheckCircle2 className="size-4 text-green-500" />
    case 'processing':
      return <Loader2 className="size-4 text-blue-500 animate-spin" />
    case 'failed':
      return <XCircle className="size-4 text-red-500" />
    default:
      return <Clock className="size-4 text-muted-foreground" />
  }
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function DocCard({
  doc,
  onDelete,
  onReprocess,
}: {
  doc: KnowledgeDocument
  onDelete: (id: string) => void
  onReprocess: (id: string) => void
}) {
  const [expanded, setExpanded] = useState(false)
  const { data: chunks } = useKnowledgeChunks(expanded ? doc.id : undefined)

  return (
    <Card className="p-4">
      <div className="flex items-start gap-3">
        <div className="rounded-lg bg-muted p-2">
          <FileText className="size-5 text-muted-foreground" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <h3 className="font-medium truncate">{doc.title}</h3>
            <StatusIcon status={doc.status} />
          </div>
          {doc.description && (
            <p className="text-sm text-muted-foreground mt-0.5 line-clamp-1">{doc.description}</p>
          )}
          <div className="flex items-center gap-3 mt-2 text-xs text-muted-foreground">
            <span>{formatFileSize(doc.file_size)}</span>
            <span>{doc.chunk_count} 分块</span>
            <span>{formatRelativeTime(doc.created_at)}</span>
          </div>
          {doc.tags.length > 0 && (
            <div className="flex gap-1 mt-2 flex-wrap">
              {doc.tags.map((tag) => (
                <Badge key={tag} variant="secondary" className="text-xs">
                  <Tag className="size-3 mr-1" />
                  {tag}
                </Badge>
              ))}
            </div>
          )}
        </div>
        <div className="flex items-center gap-1 shrink-0">
          {doc.status === 'failed' && (
            <Button variant="ghost" size="icon" className="size-8" onClick={() => onReprocess(doc.id)} title="重新处理">
              <RefreshCw className="size-4" />
            </Button>
          )}
          <Button
            variant="ghost"
            size="icon"
            className="size-8"
            onClick={() => setExpanded(!expanded)}
            title={expanded ? '收起分块' : '查看分块'}
          >
            {expanded ? <ChevronUp className="size-4" /> : <ChevronDown className="size-4" />}
          </Button>
          <Button variant="ghost" size="icon" className="size-8 text-destructive hover:text-destructive" onClick={() => onDelete(doc.id)} title="删除">
            <Trash2 className="size-4" />
          </Button>
        </div>
      </div>

      {expanded && (
        <div className="mt-3 pt-3 border-t space-y-2">
          {!chunks ? (
            <div className="space-y-2">
              {[1, 2].map((i) => <Skeleton key={i} className="h-12 rounded-lg" />)}
            </div>
          ) : chunks.length === 0 ? (
            <p className="text-sm text-muted-foreground">暂无分块数据</p>
          ) : (
            chunks.map((chunk) => (
              <div key={chunk.id} className="rounded-lg bg-muted/50 p-3 text-sm">
                <div className="flex items-center gap-2 mb-1">
                  <Badge variant="outline" className="text-xs">#{chunk.chunk_index}</Badge>
                  <span className="text-xs text-muted-foreground">{chunk.token_count} tokens</span>
                </div>
                <p className="text-muted-foreground line-clamp-3 whitespace-pre-wrap">{chunk.content}</p>
              </div>
            ))
          )}
        </div>
      )}
    </Card>
  )
}

function SearchPanel() {
  const [query, setQuery] = useState('')
  const searchMutation = useKnowledgeSearch()
  const [results, setResults] = useState<KnowledgeSearchResult[]>([])

  const handleSearch = async () => {
    if (!query.trim()) return
    try {
      const res = await searchMutation.mutateAsync({ query: query.trim(), top_k: 5, min_score: 0.3 })
      setResults(res.data.items)
    } catch {
      toast.error('搜索失败')
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex gap-2">
        <Input
          placeholder="输入语义搜索内容..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
        />
        <Button onClick={handleSearch} disabled={searchMutation.isPending || !query.trim()}>
          {searchMutation.isPending ? <Loader2 className="size-4 animate-spin" /> : <Search className="size-4" />}
          <span className="ml-2">搜索</span>
        </Button>
      </div>

      {results.length > 0 && (
        <div className="space-y-2">
          {results.map((result) => (
            <Card key={result.chunk_id} className="p-3">
              <div className="flex items-center gap-2 mb-1">
                <span className="font-medium text-sm">{result.document_title}</span>
                <Badge variant="outline" className="text-xs">#{result.chunk_index}</Badge>
                <span className="text-xs text-muted-foreground ml-auto">
                  相似度 {(result.score * 100).toFixed(1)}%
                </span>
              </div>
              <p className="text-sm text-muted-foreground whitespace-pre-wrap line-clamp-4">{result.content}</p>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}

export function KnowledgePage() {
  const [statusFilter, setStatusFilter] = useState<string>('all')
  const [activeTab, setActiveTab] = useState<'documents' | 'search'>('documents')
  const fileInputRef = useRef<HTMLInputElement>(null)

  const queryParams = statusFilter === 'all' ? undefined : { status: statusFilter }
  const { data: docs, isLoading } = useKnowledgeDocs(queryParams)
  const uploadMutation = useUploadDocument()
  const deleteMutation = useDeleteDocument()
  const reprocessMutation = useReprocessDocument()

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    const formData = new FormData()
    formData.append('file', file)
    formData.append('title', file.name)

    try {
      await uploadMutation.mutateAsync(formData)
      toast.success('文档上传成功，正在处理中...')
    } catch {
      toast.error('上传失败')
    }
    if (fileInputRef.current) fileInputRef.current.value = ''
  }

  const handleDelete = async (id: string) => {
    try {
      await deleteMutation.mutateAsync(id)
      toast.success('文档已删除')
    } catch {
      toast.error('删除失败')
    }
  }

  const handleReprocess = async (id: string) => {
    try {
      await reprocessMutation.mutateAsync(id)
      toast.success('正在重新处理...')
    } catch {
      toast.error('重新处理失败')
    }
  }

  return (
    <PageContainer>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold">知识库</h1>
          <p className="text-muted-foreground mt-1">管理文档和知识，供 Agent 检索使用</p>
        </div>
        <div>
          <input
            ref={fileInputRef}
            type="file"
            className="hidden"
            accept=".md,.txt,.text"
            onChange={handleUpload}
          />
          <Button onClick={() => fileInputRef.current?.click()} disabled={uploadMutation.isPending}>
            {uploadMutation.isPending ? (
              <Loader2 className="size-4 mr-2 animate-spin" />
            ) : (
              <Upload className="size-4 mr-2" />
            )}
            上传文档
          </Button>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 mb-6 rounded-lg bg-muted p-1 w-fit">
        {[
          { key: 'documents' as const, label: '文档管理' },
          { key: 'search' as const, label: '语义搜索' },
        ].map((tab) => (
          <button
            key={tab.key}
            onClick={() => setActiveTab(tab.key)}
            className={cn(
              'px-3 py-1.5 text-sm rounded-md transition-colors',
              activeTab === tab.key
                ? 'bg-background text-foreground shadow-sm font-medium'
                : 'text-muted-foreground hover:text-foreground'
            )}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {activeTab === 'search' ? (
        <SearchPanel />
      ) : (
        <>
          {/* Status filter */}
          <div className="flex gap-1 mb-4 rounded-lg bg-muted p-1 w-fit">
            {STATUS_FILTERS.map((filter) => (
              <button
                key={filter.value}
                onClick={() => setStatusFilter(filter.value)}
                className={cn(
                  'px-3 py-1.5 text-sm rounded-md transition-colors',
                  statusFilter === filter.value
                    ? 'bg-background text-foreground shadow-sm font-medium'
                    : 'text-muted-foreground hover:text-foreground'
                )}
              >
                {filter.label}
              </button>
            ))}
          </div>

          {/* Document list */}
          {isLoading ? (
            <div className="space-y-3">
              {[1, 2, 3].map((i) => <Skeleton key={i} className="h-24 rounded-xl" />)}
            </div>
          ) : !docs || docs.length === 0 ? (
            <EmptyState
              icon={FileText}
              title="暂无知识文档"
              description="上传文档后，Agent 在执行任务时可以检索相关知识"
              action={
                <Button onClick={() => fileInputRef.current?.click()}>
                  <Upload className="size-4 mr-2" />
                  上传文档
                </Button>
              }
            />
          ) : (
            <div className="space-y-3">
              {docs.map((doc) => (
                <DocCard
                  key={doc.id}
                  doc={doc}
                  onDelete={handleDelete}
                  onReprocess={handleReprocess}
                />
              ))}
            </div>
          )}
        </>
      )}
    </PageContainer>
  )
}
