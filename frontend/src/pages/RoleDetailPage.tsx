import { useState } from 'react'
import { useParams, useNavigate, useLocation } from 'react-router-dom'
import { ArrowLeft, Download, FileText, Brain, BookOpen, Terminal } from 'lucide-react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Separator } from '@/components/ui/separator'
import { ScrollArea } from '@/components/ui/scroll-area'
import { cn } from '@/lib/utils'
import { useMarketRole } from '@/hooks/useMarket'
import { downloadRole } from '@/api/market'
import { toast } from 'sonner'

const deptColorMap: Record<string, string> = {
  engineering:        'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
  marketing:          'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400',
  design:             'bg-pink-100 text-pink-700 dark:bg-pink-900/30 dark:text-pink-400',
  product:            'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400',
  'project-management': 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400',
  testing:            'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
  support:            'bg-teal-100 text-teal-700 dark:bg-teal-900/30 dark:text-teal-400',
  specialized:        'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400',
  'creative-tech':    'bg-cyan-100 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-400',
  finance:            'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400',
  hr:                 'bg-lime-100 text-lime-700 dark:bg-lime-900/30 dark:text-lime-400',
  legal:              'bg-stone-100 text-stone-700 dark:bg-stone-900/30 dark:text-stone-400',
  'sales-marketing':  'bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400',
  'supply-chain':     'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400',
  academic:           'bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-400',
}

function MarkdownBody({ content }: { content: string }) {
  return (
    <div className={cn(
      'prose prose-sm dark:prose-invert max-w-none',
      'prose-headings:font-semibold prose-headings:text-foreground',
      'prose-p:text-muted-foreground prose-p:leading-relaxed',
      'prose-li:text-muted-foreground',
      'prose-code:bg-muted prose-code:px-1 prose-code:py-0.5 prose-code:rounded prose-code:text-xs prose-code:font-mono',
      'prose-pre:bg-muted prose-pre:rounded-lg',
      'prose-strong:text-foreground',
      'prose-a:text-primary',
      '[&>*:first-child]:mt-0',
    )}>
      <ReactMarkdown remarkPlugins={[remarkGfm]}>{content}</ReactMarkdown>
    </div>
  )
}

function InstallStep({ num, children }: { num: number; children: React.ReactNode }) {
  return (
    <div className="flex gap-3">
      <span className="flex size-5 shrink-0 items-center justify-center rounded-full bg-primary/10 text-xs font-semibold text-primary mt-0.5">
        {num}
      </span>
      <div className="flex-1 text-sm text-muted-foreground">{children}</div>
    </div>
  )
}

function RoleDetailSkeleton() {
  return (
    <div className="flex h-full flex-col">
      <div className="border-b px-6 py-4">
        <Skeleton className="h-4 w-24" />
      </div>
      <div className="border-b px-6 py-6">
        <Skeleton className="h-4 w-20 mb-3" />
        <Skeleton className="h-8 w-48 mb-2" />
        <Skeleton className="h-4 w-96" />
      </div>
      <div className="flex flex-1 min-h-0 gap-0">
        <div className="flex-1 p-6 space-y-3">
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-5/6" />
          <Skeleton className="h-4 w-4/5" />
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-3/4" />
        </div>
        <div className="w-72 border-l p-5 space-y-3">
          <Skeleton className="h-4 w-24" />
          <Skeleton className="h-20 w-full rounded-lg" />
        </div>
      </div>
    </div>
  )
}

export function RoleDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const [downloading, setDownloading] = useState(false)

  const { data: role, isLoading } = useMarketRole(id)

  // 返回列表时恢复之前的筛选状态
  function handleBack() {
    const fromSearch = location.state?.fromSearch as string | undefined
    navigate(fromSearch ? `/market?${fromSearch}` : '/market')
  }

  async function handleDownload() {
    if (!id) return
    setDownloading(true)
    try {
      await downloadRole(id)
      toast.success('角色包下载成功')
    } catch {
      toast.error('下载失败，请重试')
    } finally {
      setDownloading(false)
    }
  }

  if (isLoading || !role) {
    return <RoleDetailSkeleton />
  }

  const deptColor = deptColorMap[role.dept_id] ?? 'bg-muted text-muted-foreground'

  return (
    <div className="flex h-full flex-col">

      {/* 顶部导航栏 */}
      <div className="flex items-center justify-between border-b px-6 py-3 shrink-0">
        <button
          onClick={handleBack}
          className="flex items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
        >
          <ArrowLeft className="size-4" />
          岗位市场
        </button>
        <Button onClick={handleDownload} disabled={downloading} size="sm">
          <Download className="size-4 mr-1.5" />
          {downloading ? '下载中...' : '下载角色包'}
        </Button>
      </div>

      {/* Hero：角色名 + 描述 */}
      <div className="border-b px-6 py-5 shrink-0">
        <div className="flex items-start gap-3">
          <div className="flex-1 min-w-0">
            <span className={cn('inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium mb-2', deptColor)}>
              {role.dept_name}
            </span>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">{role.name}</h1>
            <p className="mt-1 text-sm text-muted-foreground leading-relaxed max-w-2xl">{role.description}</p>
          </div>
        </div>
      </div>

      {/* 主体：左侧 Tabs 内容 + 右侧边栏 */}
      <div className="flex flex-1 min-h-0">

        {/* 左侧：Tabs + Markdown */}
        <div className="flex flex-1 flex-col min-w-0 min-h-0">
          <Tabs defaultValue="soul" className="flex flex-col flex-1 min-h-0">
            <div className="border-b px-6 shrink-0">
              <TabsList variant="line" className="h-10 gap-1">
                <TabsTrigger value="identity" className="gap-1.5">
                  <FileText className="size-3.5" />
                  简介
                </TabsTrigger>
                <TabsTrigger value="soul" className="gap-1.5">
                  <Brain className="size-3.5" />
                  人格
                </TabsTrigger>
                <TabsTrigger value="agents" className="gap-1.5">
                  <BookOpen className="size-3.5" />
                  工作规范
                </TabsTrigger>
              </TabsList>
            </div>

            <TabsContent value="identity" className="flex-1 min-h-0 mt-0">
              <ScrollArea className="h-full">
                <div className="px-6 py-6 max-w-3xl">
                  <MarkdownBody content={role.identity_content} />
                </div>
              </ScrollArea>
            </TabsContent>

            <TabsContent value="soul" className="flex-1 min-h-0 mt-0">
              <ScrollArea className="h-full">
                <div className="px-6 py-6 max-w-3xl">
                  <MarkdownBody content={role.soul_content} />
                </div>
              </ScrollArea>
            </TabsContent>

            <TabsContent value="agents" className="flex-1 min-h-0 mt-0">
              <ScrollArea className="h-full">
                <div className="px-6 py-6 max-w-3xl">
                  <MarkdownBody content={role.agents_content} />
                </div>
              </ScrollArea>
            </TabsContent>
          </Tabs>
        </div>

        {/* 右侧边栏：安装说明 + 元信息 */}
        <aside className="w-72 shrink-0 border-l flex flex-col min-h-0">
          <ScrollArea className="flex-1">
            <div className="p-5 space-y-5">

              {/* 安装说明（默认展开） */}
              <div>
                <div className="flex items-center gap-2 mb-3">
                  <Terminal className="size-3.5 text-muted-foreground" />
                  <span className="text-sm font-medium">安装说明</span>
                </div>
                <div className="space-y-3">
                  <InstallStep num={1}>
                    下载角色包{' '}
                    <code className="text-xs bg-muted px-1 py-0.5 rounded font-mono">{role.id}.zip</code>
                  </InstallStep>
                  <InstallStep num={2}>
                    解压后将角色目录复制到本地 OpenClaw：
                    <pre className="mt-2 rounded-md bg-muted px-3 py-2 text-xs font-mono overflow-x-auto whitespace-pre-wrap break-all">
                      {`cp -r ${role.id} ~/.openclaw/agents/`}
                    </pre>
                  </InstallStep>
                  <InstallStep num={3}>
                    重启 OpenClaw 网关，角色自动激活
                  </InstallStep>
                  <InstallStep num={4}>
                    在 TrustMesh 项目团队中邀请该智能体加入，即可参与协作
                  </InstallStep>
                </div>
                <p className="mt-3 text-xs text-muted-foreground">
                  需要安装{' '}
                  <a
                    href="https://docs.openclaw.ai"
                    target="_blank"
                    rel="noreferrer"
                    className="text-primary hover:underline"
                  >
                    OpenClaw
                  </a>{' '}
                  并通过 Clawsynapse 连接到 TrustMesh。
                </p>
              </div>

              <Separator />

              {/* 角色元信息 */}
              <div className="space-y-2">
                <span className="text-sm font-medium">角色信息</span>
                <div className="space-y-1.5 text-xs text-muted-foreground">
                  <div className="flex justify-between">
                    <span>ID</span>
                    <code className="font-mono text-[11px] text-foreground/70 max-w-[160px] truncate" title={role.id}>
                      {role.id}
                    </code>
                  </div>
                  <div className="flex justify-between">
                    <span>部门</span>
                    <span className="text-foreground/70">{role.dept_name}</span>
                  </div>
                  <div className="flex justify-between">
                    <span>格式</span>
                    <span className="text-foreground/70">OpenClaw Agent</span>
                  </div>
                </div>
              </div>

            </div>
          </ScrollArea>
        </aside>

      </div>
    </div>
  )
}
