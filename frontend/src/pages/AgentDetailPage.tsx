import { useParams, useNavigate } from 'react-router-dom'
import { Bot, Plus, Pencil, MoreHorizontal, Trash2, Archive } from 'lucide-react'
import { toast } from 'sonner'
import { PageContainer } from '@/components/layout/PageContainer'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger, DropdownMenuSeparator } from '@/components/ui/dropdown-menu'
import { AgentStatusBadge } from '@/components/shared/StatusBadge'
import { EventTimeline } from '@/components/shared/EventTimeline'
import { useAgent, useAgentStats, useDeleteAgent } from '@/hooks/useAgents'
import { useAgentInsights } from '@/hooks/useAgentInsights'
import { useAgentEvents } from '@/hooks/useDashboard'
import { ApiRequestError } from '@/api/client'
import { formatRelativeTime } from '@/lib/utils'
import { useCallback, useRef, useState } from 'react'
import {
  AgentMetricCards,
  AgentDailyChart,
  AgentInsightPanels,
  AgentWorkload,
} from '@/components/agent/AgentStatsPanel'
import { AgentConfigDialog } from '@/components/agent/AgentConfigDialog'
import { ArchiveAgentDialog } from '@/components/agent/ArchiveAgentDialog'
import { CreateTaskDialog } from '@/components/task/CreateTaskDialog'
import type { EventType } from '@/types'

const eventTypeFilters: { label: string; value: EventType | 'all' }[] = [
  { label: '全部', value: 'all' },
  { label: '任务', value: 'task_created' },
  { label: 'Todo', value: 'todo_completed' },
  { label: '失败', value: 'todo_failed' },
  { label: '进度', value: 'todo_progress' },
]

export function AgentDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { data: agent, isLoading: agentLoading } = useAgent(id)
  const { data: stats } = useAgentStats(id)
  const { data: insights, isLoading: insightsLoading } = useAgentInsights(id)
  const { data: events, isLoading: eventsLoading } = useAgentEvents(id)
  const navigate = useNavigate()
  const deleteAgent = useDeleteAgent()
  const [filter, setFilter] = useState<EventType | 'all'>('all')
  const [createTaskOpen, setCreateTaskOpen] = useState(false)
  const [editOpen, setEditOpen] = useState(false)
  const [archiveOpen, setArchiveOpen] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)
  const headerRef = useCallback((node: HTMLDivElement | null) => {
    if (!node) return
    const observer = new ResizeObserver(() => {
      containerRef.current?.style.setProperty('--agent-header-h', `${node.offsetHeight}px`)
    })
    observer.observe(node)
  }, [])

  if (agentLoading) {
    return <div className="p-6 text-sm text-muted-foreground">加载中...</div>
  }

  if (!agent) {
    return <div className="p-6 text-sm text-muted-foreground">Agent 未找到</div>
  }

  const handleDelete = async () => {
    if (agent.usage.in_use) {
      setArchiveOpen(true)
      return
    }
    if (confirm(`确定要删除 Agent "${agent.name}" 吗？`)) {
      try {
        await deleteAgent.mutateAsync(agent.id)
        toast.success(`Agent "${agent.name}" 已删除`)
        navigate('/agents')
      } catch (err) {
        const message = err instanceof ApiRequestError ? err.message : '删除失败'
        toast.error(message)
      }
    }
  }

  const filteredEvents = filter === 'all'
    ? (events ?? [])
    : (events ?? []).filter((e) => e.event_type === filter)

  return (
    <PageContainer ref={containerRef} className="h-full overflow-y-auto overflow-x-hidden p-0 [--agent-header-h:4.5rem]">
      <div className="min-h-full bg-background">
        <div ref={headerRef} className="sticky top-0 z-20 border-b bg-background/95 px-6 py-3 supports-backdrop-filter:backdrop-blur-xs">
          <div className="flex items-center gap-3">
            <div className="flex size-11 items-center justify-center rounded-xl bg-primary/10 text-primary shrink-0">
              <Bot className="size-6" />
            </div>
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <h1 className="text-lg font-bold truncate">{agent.name}</h1>
                <AgentStatusBadge status={agent.status} />
                {agent.archived && (
                  <Badge variant="secondary" className="text-xs">已归档</Badge>
                )}
                <span className="text-xs text-muted-foreground">
                  <span className="capitalize">{agent.role}</span>
                  <span> · </span>
                  <span className="font-mono">{agent.node_id}</span>
                  {agent.last_seen_at && (
                    <span> · {formatRelativeTime(agent.last_seen_at)}</span>
                  )}
                </span>
              </div>
              <div className="flex items-center gap-2 mt-0.5">
                {agent.description && (
                  <p className="text-xs text-muted-foreground truncate">{agent.description}</p>
                )}
                {agent.description && agent.capabilities.length > 0 && (
                  <span className="text-xs text-muted-foreground shrink-0">·</span>
                )}
                {agent.capabilities.length > 0 && (
                  <div className="flex items-center gap-1 shrink-0">
                    {agent.capabilities.map((cap) => (
                      <Badge key={cap} variant="secondary" className="text-[11px] px-1.5 py-0">
                        {cap}
                      </Badge>
                    ))}
                  </div>
                )}
              </div>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              {agent.role !== 'pm' && !agent.archived && (
                <Button size="sm" variant="outline" onClick={() => setCreateTaskOpen(true)}>
                  <Plus className="size-4 mr-1.5" />
                  创建任务
                </Button>
              )}
              {!agent.archived && (
                <DropdownMenu>
                  <DropdownMenuTrigger className="inline-flex items-center justify-center rounded-md hover:bg-muted size-8">
                    <MoreHorizontal className="size-4" />
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem onClick={() => setEditOpen(true)}>
                      <Pencil className="size-3.5 mr-2" />
                      编辑
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem onClick={handleDelete} className="text-destructive">
                      {agent.usage.in_use ? (
                        <><Archive className="size-3.5 mr-2" />归档</>
                      ) : (
                        <><Trash2 className="size-3.5 mr-2" />删除</>
                      )}
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              )}
            </div>
          </div>
        </div>

        {agent.archived && (
          <div className="mx-6 mt-4 flex items-center gap-3 rounded-lg border border-amber-500/20 bg-amber-500/5 px-4 py-3">
            <Archive className="size-4 text-amber-600 shrink-0" />
            <p className="text-sm text-amber-700 dark:text-amber-400">
              该 Agent 已归档，不再接收新任务
            </p>
          </div>
        )}

        <div className="px-6 py-6">
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-[minmax(0,1fr)_480px] lg:items-start">
            {/* 左侧：指标 + 图表 + 工作负载 */}
            <div className="flex min-w-0 flex-col gap-4 lg:pr-2">

              {stats && <AgentMetricCards stats={stats} />}
              {stats && <AgentDailyChart stats={stats} />}
              {stats && (
              <AgentInsightPanels
                stats={stats}
                insights={insights ?? null}
                loading={insightsLoading}
              />
              )}
              {stats && <AgentWorkload stats={stats} />}
            </div>

            {/* 右侧：活动历史 */}
            <Card className="flex flex-col overflow-hidden lg:sticky lg:top-[calc(var(--agent-header-h)+1.5rem)] lg:max-h-[calc(100dvh-var(--agent-header-h)-3rem)]">
              <CardHeader className="shrink-0 pb-3">
                <CardTitle className="text-base">活动历史</CardTitle>
                <div className="mt-2 flex flex-wrap gap-1">
                  {eventTypeFilters.map((f) => (
                    <Button
                      key={f.value}
                      variant={filter === f.value ? 'default' : 'ghost'}
                      size="sm"
                      className="h-7 text-xs"
                      onClick={() => setFilter(f.value)}
                    >
                      {f.label}
                    </Button>
                  ))}
                </div>
              </CardHeader>
              <CardContent className="lg:flex-1 lg:min-h-0 lg:overflow-y-auto">
                <EventTimeline
                  events={filteredEvents}
                  loading={eventsLoading}
                  emptyText="暂无活动记录"
                />
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
      {id && (
        <CreateTaskDialog
          open={createTaskOpen}
          onOpenChange={setCreateTaskOpen}
          defaultAgentId={id}
        />
      )}
      <AgentConfigDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        agent={agent}
      />
      <ArchiveAgentDialog
        open={archiveOpen}
        onOpenChange={setArchiveOpen}
        agent={agent}
        onArchived={() => navigate('/agents')}
      />
    </PageContainer>
  )
}
