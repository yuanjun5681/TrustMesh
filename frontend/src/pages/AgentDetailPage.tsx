import { useParams, Link } from 'react-router-dom'
import { ChevronLeft, Plus } from 'lucide-react'
import { PageContainer } from '@/components/layout/PageContainer'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { AgentStatusBadge } from '@/components/shared/StatusBadge'
import { EventTimeline } from '@/components/shared/EventTimeline'
import { useAgent, useAgentStats } from '@/hooks/useAgents'
import { useAgentInsights } from '@/hooks/useAgentInsights'
import { useAgentEvents } from '@/hooks/useDashboard'
import { formatRelativeTime } from '@/lib/utils'
import { useCallback, useRef, useState } from 'react'
import {
  AgentMetricCards,
  AgentDailyChart,
  AgentInsightPanels,
  AgentWorkload,
} from '@/components/agent/AgentStatsPanel'
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
  const [filter, setFilter] = useState<EventType | 'all'>('all')
  const [createTaskOpen, setCreateTaskOpen] = useState(false)
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

  const filteredEvents = filter === 'all'
    ? (events ?? [])
    : (events ?? []).filter((e) => e.event_type === filter)

  return (
    <PageContainer ref={containerRef} className="h-full overflow-y-auto overflow-x-hidden p-0 [--agent-header-h:4.5rem]">
      <div className="min-h-full bg-background">
        <div ref={headerRef} className="sticky top-0 z-20 border-b bg-background/95 px-6 py-4 supports-backdrop-filter:backdrop-blur-xs">
          <div className="flex items-center gap-2">
            <Link to="/dashboard">
              <Button variant="ghost" size="icon">
                <ChevronLeft className="size-4" />
              </Button>
            </Link>
            <h1 className="text-2xl font-bold">{agent.name}</h1>
            <AgentStatusBadge status={agent.status} />
            <div className="ml-auto">
              {agent.role !== 'pm' && !agent.archived && (
                <Button size="sm" variant="outline" onClick={() => setCreateTaskOpen(true)}>
                  <Plus className="size-4 mr-1.5" />
                  创建任务
                </Button>
              )}
            </div>
          </div>
        </div>

        <div className="px-6 py-6">
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-[minmax(0,1fr)_480px] lg:items-start">
            {/* 左侧：概要 + 指标 + 图表 + 工作负载 */}
            <div className="flex min-w-0 flex-col gap-4 lg:pr-2">
              {/* Agent 概要 */}
              <Card>
                <CardContent className="p-4">
                  <div className="grid grid-cols-2 gap-4 text-sm md:grid-cols-4">
                    <div>
                      <div className="text-xs text-muted-foreground">角色</div>
                      <div className="font-medium capitalize">{agent.role}</div>
                    </div>
                    <div>
                      <div className="text-xs text-muted-foreground">Node ID</div>
                      <div className="font-mono text-xs">{agent.node_id}</div>
                    </div>
                    <div>
                      <div className="text-xs text-muted-foreground">最后活跃</div>
                      <div>{agent.last_seen_at ? formatRelativeTime(agent.last_seen_at) : '-'}</div>
                    </div>
                    <div>
                      <div className="text-xs text-muted-foreground">使用统计</div>
                      <div>
                        {agent.usage.project_count} 项目 · {agent.usage.task_count} 任务 · {agent.usage.todo_count} Todo
                      </div>
                    </div>
                  </div>
                  {agent.capabilities.length > 0 && (
                    <div className="mt-3 flex flex-wrap gap-1.5">
                      {agent.capabilities.map((cap) => (
                        <Badge key={cap} variant="secondary" className="text-xs">
                          {cap}
                        </Badge>
                      ))}
                    </div>
                  )}
                </CardContent>
              </Card>

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
    </PageContainer>
  )
}
