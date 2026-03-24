import { Link } from 'react-router-dom'
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
  Legend,
} from 'recharts'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { formatRelativeTime } from '@/lib/utils'
import type { AgentStats } from '@/types'
import {
  CheckCircle2,
  Clock,
  Zap,
  ListTodo,
  FolderKanban,
  ClipboardList,
  MessageSquare,
} from 'lucide-react'

function formatDuration(ms: number | null): string {
  if (ms === null) return '-'
  if (ms < 1000) return `${Math.round(ms)} ms`
  const sec = ms / 1000
  if (sec < 60) return `${sec.toFixed(1)} 秒`
  const min = sec / 60
  if (min < 60) return `${min.toFixed(1)} 分钟`
  const hr = min / 60
  return `${hr.toFixed(1)} 小时`
}

function formatDateLabel(dateStr: string): string {
  const parts = dateStr.split('-')
  return `${parts[1]}/${parts[2]}`
}

/** 指标卡片 — 根据角色显示不同内容 */
export function AgentMetricCards({ stats }: { stats: AgentStats }) {
  if (stats.role === 'pm') {
    return <PMMetricCards stats={stats} />
  }
  return <ExecutorMetricCards stats={stats} />
}

function PMMetricCards({ stats }: { stats: AgentStats }) {
  const hasTask = stats.tasks_created > 0
  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-2 text-muted-foreground mb-1">
            <FolderKanban className="size-3.5" />
            <span className="text-xs">管理项目</span>
          </div>
          <div className="text-2xl font-bold">{stats.projects_managed}</div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-2 text-muted-foreground mb-1">
            <ClipboardList className="size-3.5" />
            <span className="text-xs">创建任务</span>
          </div>
          <div className="text-2xl font-bold">{stats.tasks_created}</div>
          <div className="flex gap-1.5 mt-1 flex-wrap">
            {stats.tasks_in_progress > 0 && (
              <Badge variant="secondary" className="text-xs px-1.5 py-0">
                {stats.tasks_in_progress} 进行中
              </Badge>
            )}
            {stats.tasks_pending > 0 && (
              <Badge variant="outline" className="text-xs px-1.5 py-0">
                {stats.tasks_pending} 待处理
              </Badge>
            )}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-2 text-muted-foreground mb-1">
            <CheckCircle2 className="size-3.5" />
            <span className="text-xs">任务完成率</span>
          </div>
          <div className="text-2xl font-bold">
            {hasTask ? `${stats.task_success_rate.toFixed(1)}%` : '-'}
          </div>
          <div className="text-xs text-muted-foreground mt-0.5">
            {stats.tasks_done} 完成 / {stats.tasks_failed} 失败
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-2 text-muted-foreground mb-1">
            <MessageSquare className="size-3.5" />
            <span className="text-xs">对话回复</span>
          </div>
          <div className="text-2xl font-bold">{stats.conversation_replies}</div>
        </CardContent>
      </Card>
    </div>
  )
}

function ExecutorMetricCards({ stats }: { stats: AgentStats }) {
  const hasActivity = stats.todos_total > 0
  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-2 text-muted-foreground mb-1">
            <CheckCircle2 className="size-3.5" />
            <span className="text-xs">完成率</span>
          </div>
          <div className="text-2xl font-bold">
            {hasActivity ? `${stats.success_rate.toFixed(1)}%` : '-'}
          </div>
          <div className="text-xs text-muted-foreground mt-0.5">
            {stats.todos_done} 完成 / {stats.todos_failed} 失败
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-2 text-muted-foreground mb-1">
            <ListTodo className="size-3.5" />
            <span className="text-xs">Todo 总览</span>
          </div>
          <div className="text-2xl font-bold">{stats.todos_total}</div>
          <div className="flex gap-1.5 mt-1 flex-wrap">
            {stats.todos_in_progress > 0 && (
              <Badge variant="secondary" className="text-xs px-1.5 py-0">
                {stats.todos_in_progress} 进行中
              </Badge>
            )}
            {stats.todos_pending > 0 && (
              <Badge variant="outline" className="text-xs px-1.5 py-0">
                {stats.todos_pending} 待处理
              </Badge>
            )}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-2 text-muted-foreground mb-1">
            <Zap className="size-3.5" />
            <span className="text-xs">平均响应时间</span>
          </div>
          <div className="text-2xl font-bold">
            {formatDuration(stats.avg_response_time_ms)}
          </div>
          <div className="text-xs text-muted-foreground mt-0.5">
            接收到开始执行
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-2 text-muted-foreground mb-1">
            <Clock className="size-3.5" />
            <span className="text-xs">平均完成时间</span>
          </div>
          <div className="text-2xl font-bold">
            {formatDuration(stats.avg_completion_time_ms)}
          </div>
          <div className="text-xs text-muted-foreground mt-0.5">
            开始到完成
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

/** 每日工作量图表 — 根据角色显示不同数据 */
export function AgentDailyChart({ stats }: { stats: AgentStats }) {
  const isPM = stats.role === 'pm'
  const hasActivity = isPM ? stats.tasks_created > 0 : stats.todos_total > 0

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-base">
          {isPM ? '每日任务动态（近 30 天）' : '每日工作量（近 30 天）'}
        </CardTitle>
      </CardHeader>
      <CardContent>
        {hasActivity ? (
          <ResponsiveContainer width="100%" height={220}>
            <BarChart
              data={stats.daily_activity}
              margin={{ top: 4, right: 4, bottom: 0, left: -20 }}
            >
              <CartesianGrid
                strokeDasharray="3 3"
                vertical={false}
                stroke="var(--color-border)"
              />
              <XAxis
                dataKey="date"
                tickFormatter={formatDateLabel}
                tick={{ fontSize: 11, fill: 'var(--color-muted-foreground)' }}
                interval={4}
                axisLine={false}
                tickLine={false}
              />
              <YAxis
                allowDecimals={false}
                tick={{ fontSize: 11, fill: 'var(--color-muted-foreground)' }}
                axisLine={false}
                tickLine={false}
              />
              <Tooltip
                labelFormatter={(label) => `日期: ${String(label)}`}
                contentStyle={{
                  backgroundColor: 'var(--color-popover)',
                  border: '1px solid var(--color-border)',
                  borderRadius: 'var(--radius-md)',
                  fontSize: 12,
                  color: 'var(--color-popover-foreground)',
                }}
              />
              <Legend
                wrapperStyle={{ fontSize: 12 }}
              />
              {isPM && (
                <Bar
                  dataKey="created"
                  name="创建"
                  fill="var(--color-primary)"
                  radius={[2, 2, 0, 0]}
                />
              )}
              <Bar
                dataKey="completed"
                name="完成"
                stackId="result"
                fill="var(--color-success)"
                radius={[0, 0, 0, 0]}
              />
              <Bar
                dataKey="failed"
                name="失败"
                stackId="result"
                fill="var(--color-destructive)"
                radius={[2, 2, 0, 0]}
              />
            </BarChart>
          </ResponsiveContainer>
        ) : (
          <div className="h-[220px] flex items-center justify-center text-sm text-muted-foreground">
            暂无活动数据
          </div>
        )}
      </CardContent>
    </Card>
  )
}

/** 当前工作负载 */
export function AgentWorkload({ stats }: { stats: AgentStats }) {
  if (stats.current_workload.length === 0) return null
  const isPM = stats.role === 'pm'
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-base flex items-center gap-2">
          {isPM ? '进行中的任务' : '当前工作负载'}
          <Badge variant="secondary" className="text-xs">
            {stats.current_workload.length}
          </Badge>
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="divide-y divide-border">
          {stats.current_workload.map((item) => (
            <div
              key={item.todo_id || item.task_id}
              className="py-2.5 first:pt-0 last:pb-0 flex items-center justify-between gap-3"
            >
              <div className="min-w-0 flex-1">
                <div className="text-sm font-medium truncate">
                  {isPM ? item.task_title : item.todo_title}
                </div>
                {!isPM && item.task_title && (
                  <div className="text-xs text-muted-foreground mt-0.5">
                    <Link
                      to={`/projects/${item.project_id}/tasks/${item.task_id}`}
                      className="hover:underline"
                    >
                      {item.task_title}
                    </Link>
                  </div>
                )}
              </div>
              <div className="text-xs text-muted-foreground shrink-0">
                {formatRelativeTime(item.started_at)}
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}
