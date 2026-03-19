import {
  CheckCircle2,
  Circle,
  AlertCircle,
  PlayCircle,
  UserCircle,
  Bot,
  Cog,
  MessageSquare,
  Radio,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { formatDateTime } from '@/lib/utils'
import type { Event, EventType } from '@/types'

const eventConfig: Record<EventType, { icon: typeof Circle; color: string; label: string }> = {
  task_created: { icon: Circle, color: 'text-info', label: '任务创建' },
  task_status_changed: { icon: Cog, color: 'text-warning', label: '状态变更' },
  todo_assigned: { icon: UserCircle, color: 'text-info', label: '分配 Todo' },
  todo_started: { icon: PlayCircle, color: 'text-info', label: '开始执行' },
  todo_progress: { icon: Cog, color: 'text-muted-foreground', label: '执行中' },
  todo_completed: { icon: CheckCircle2, color: 'text-success', label: 'Todo 完成' },
  todo_failed: { icon: AlertCircle, color: 'text-destructive', label: 'Todo 失败' },
  task_comment: { icon: MessageSquare, color: 'text-muted-foreground', label: '评论' },
  conversation_reply: { icon: MessageSquare, color: 'text-info', label: 'PM 回复' },
  agent_status_changed: { icon: Radio, color: 'text-warning', label: 'Agent 状态' },
}

const actorIcons = {
  user: UserCircle,
  agent: Bot,
  system: Cog,
}

interface EventTimelineProps {
  events: Event[]
  loading?: boolean
  emptyText?: string
  showActorName?: boolean
}

export function EventTimeline({
  events,
  loading,
  emptyText = '暂无事件',
  showActorName = false,
}: EventTimelineProps) {
  if (loading) {
    return <div className="py-8 text-center text-sm text-muted-foreground">加载中...</div>
  }

  if (!events || events.length === 0) {
    return <div className="py-8 text-center text-sm text-muted-foreground">{emptyText}</div>
  }

  return (
    <div className="relative space-y-0">
      {events.map((event, index) => {
        const config = eventConfig[event.event_type] ?? {
          icon: Circle,
          color: 'text-muted-foreground',
          label: event.event_type,
        }
        const Icon = config.icon
        const ActorIcon = actorIcons[event.actor_type]
        const isLast = index === events.length - 1

        return (
          <div key={event.id} className="flex gap-3">
            <div className="flex flex-col items-center">
              <div
                className={cn(
                  'flex h-7 w-7 items-center justify-center rounded-full bg-muted',
                  config.color
                )}
              >
                <Icon className="h-3.5 w-3.5" />
              </div>
              {!isLast && <div className="w-px flex-1 bg-border" />}
            </div>
            <div className={cn('pb-4 flex-1', isLast && 'pb-0')}>
              <div className="flex items-center gap-1.5">
                <ActorIcon className="h-3.5 w-3.5 text-muted-foreground" />
                {showActorName && event.actor_name && (
                  <span className="text-xs text-muted-foreground">{event.actor_name}</span>
                )}
                <span className="text-sm font-medium">{config.label}</span>
                <span className="text-xs text-muted-foreground ml-auto">
                  {formatDateTime(event.created_at)}
                </span>
              </div>
              {event.content && (
                <p className="text-sm text-muted-foreground mt-0.5">{event.content}</p>
              )}
            </div>
          </div>
        )
      })}
    </div>
  )
}
