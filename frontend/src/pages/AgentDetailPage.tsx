import { useParams, Link } from 'react-router-dom'
import { ChevronLeft } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { AgentStatusBadge } from '@/components/shared/StatusBadge'
import { EventTimeline } from '@/components/shared/EventTimeline'
import { useAgent } from '@/hooks/useAgents'
import { useAgentEvents } from '@/hooks/useDashboard'
import { formatRelativeTime } from '@/lib/utils'
import { useState } from 'react'
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
    <div className="p-6 space-y-6 max-w-4xl mx-auto">
      <div className="flex items-center gap-2">
        <Link to="/dashboard">
          <Button variant="ghost" size="icon">
            <ChevronLeft className="h-4 w-4" />
          </Button>
        </Link>
        <h1 className="text-2xl font-bold">{agent.name}</h1>
        <AgentStatusBadge status={agent.status} />
      </div>

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

      {/* 事件历史 */}
      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <CardTitle className="text-base">活动历史</CardTitle>
            <div className="flex gap-1">
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
          </div>
        </CardHeader>
        <CardContent>
          <ScrollArea className="h-[500px]">
            <EventTimeline
              events={filteredEvents}
              loading={eventsLoading}
              emptyText="暂无活动记录"
            />
          </ScrollArea>
        </CardContent>
      </Card>
    </div>
  )
}
