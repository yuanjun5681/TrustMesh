import { useParams, Link } from 'react-router-dom'
import { ChevronLeft } from 'lucide-react'
import { PageContainer } from '@/components/layout/PageContainer'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { AgentStatusBadge } from '@/components/shared/StatusBadge'
import { EventTimeline } from '@/components/shared/EventTimeline'
import { useAgent, useAgentStats } from '@/hooks/useAgents'
import { useAgentEvents } from '@/hooks/useDashboard'
import { formatRelativeTime } from '@/lib/utils'
import { useState } from 'react'
import {
  AgentMetricCards,
  AgentDailyChart,
  AgentWorkload,
} from '@/components/agent/AgentStatsPanel'
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
  const { data: events, isLoading: eventsLoading } = useAgentEvents(id)
  const [filter, setFilter] = useState<EventType | 'all'>('all')

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
    <PageContainer className="h-full overflow-auto">
      {/* 标题 */}
      <div className="flex items-center gap-2 mb-6">
        <Link to="/dashboard">
          <Button variant="ghost" size="icon">
            <ChevronLeft className="size-4" />
          </Button>
        </Link>
        <h1 className="text-2xl font-bold">{agent.name}</h1>
        <AgentStatusBadge status={agent.status} />
      </div>

      {/* 左右主体布局 */}
      <div className="grid grid-cols-1 lg:grid-cols-[768px_1fr] gap-6 items-start">
        {/* 左侧：概要 + 指标 + 图表 + 工作负载 */}
        <div className="flex flex-col gap-4 min-w-0">
          {/* Agent 概要 */}
          <Card>
            <CardContent className="p-4">
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                <div>
                  <div className="text-muted-foreground text-xs">角色</div>
                  <div className="font-medium capitalize">{agent.role}</div>
                </div>
                <div>
                  <div className="text-muted-foreground text-xs">Node ID</div>
                  <div className="font-mono text-xs">{agent.node_id}</div>
                </div>
                <div>
                  <div className="text-muted-foreground text-xs">最后活跃</div>
                  <div>{agent.last_seen_at ? formatRelativeTime(agent.last_seen_at) : '-'}</div>
                </div>
                <div>
                  <div className="text-muted-foreground text-xs">使用统计</div>
                  <div>
                    {agent.usage.project_count} 项目 · {agent.usage.task_count} 任务 · {agent.usage.todo_count} Todo
                  </div>
                </div>
              </div>
              {agent.capabilities.length > 0 && (
                <div className="mt-3 flex gap-1.5 flex-wrap">
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
          {stats && <AgentWorkload stats={stats} />}
        </div>

        {/* 右侧：活动历史，视口高度约束 + 内部滚动 */}
        <Card className="flex flex-col overflow-hidden max-h-[calc(100vh-6rem)]">
          <CardHeader className="pb-3 shrink-0">
            <CardTitle className="text-base">活动历史</CardTitle>
            <div className="flex gap-1 flex-wrap mt-2">
              {eventTypeFilters.map((f) => (
                <Button
                  key={f.value}
                  variant={filter === f.value ? 'default' : 'ghost'}
                  size="sm"
                  className="text-xs h-7"
                  onClick={() => setFilter(f.value)}
                >
                  {f.label}
                </Button>
              ))}
            </div>
          </CardHeader>
          <CardContent className="flex-1 min-h-0 overflow-auto">
            <EventTimeline
              events={filteredEvents}
              loading={eventsLoading}
              emptyText="暂无活动记录"
            />
          </CardContent>
        </Card>
      </div>
    </PageContainer>
  )
}
