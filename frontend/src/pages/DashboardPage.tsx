import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { ScrollArea } from '@/components/ui/scroll-area'
import { AgentStatusCard } from '@/components/dashboard/AgentStatusCard'
import { StatsCard } from '@/components/dashboard/StatsCard'
import { RecentTasksList } from '@/components/dashboard/RecentTasksList'
import { EventTimeline } from '@/components/shared/EventTimeline'
import { useAgents } from '@/hooks/useAgents'
import { useDashboardStats, useDashboardEvents, useDashboardTasks } from '@/hooks/useDashboard'

export function DashboardPage() {
  const { data: agents } = useAgents()
  const { data: stats } = useDashboardStats()
  const { data: events, isLoading: eventsLoading } = useDashboardEvents()
  const { data: tasks, isLoading: tasksLoading } = useDashboardTasks()

  return (
    <div className="p-6 space-y-6 max-w-7xl mx-auto">
      <h1 className="text-2xl font-bold">Dashboard</h1>

      {/* Agent 状态卡片 */}
      {agents && agents.length > 0 && (
        <div>
          <h2 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-3">
            Agents
          </h2>
          <div className="flex gap-3 pb-2 overflow-x-auto">
            {agents.map((agent) => (
              <AgentStatusCard key={agent.id} agent={agent} />
            ))}
          </div>
        </div>
      )}

      {/* 统计卡片 */}
      {stats && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          <StatsCard
            label="Agent"
            value={`${stats.agents_online} / ${stats.agents_total}`}
            sub="在线"
          />
          <StatsCard
            label="进行中任务"
            value={stats.tasks_in_progress}
            sub={`共 ${stats.tasks_total} 个`}
          />
          <StatsCard
            label="已完成"
            value={stats.tasks_done_count}
          />
          <StatsCard
            label="成功率"
            value={stats.success_rate > 0 ? `${stats.success_rate.toFixed(0)}%` : '-'}
          />
        </div>
      )}

      {/* 最近活动 + 最近任务 */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-base">最近活动</CardTitle>
          </CardHeader>
          <CardContent>
            <ScrollArea className="h-[400px]">
              <EventTimeline
                events={events ?? []}
                loading={eventsLoading}
                showActorName
                emptyText="暂无活动记录"
              />
            </ScrollArea>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-base">最近任务</CardTitle>
          </CardHeader>
          <CardContent>
            <ScrollArea className="h-[400px]">
              <RecentTasksList tasks={tasks ?? []} loading={tasksLoading} />
            </ScrollArea>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
