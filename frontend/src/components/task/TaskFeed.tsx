import { useRef, useEffect, useCallback, useState } from 'react'
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
  ArrowRight,
  ArrowDown,
} from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { cn, formatRelativeTime } from '@/lib/utils'
import { TrustMeshLogo } from '@/components/shared/TrustMeshLogo'
import { useTaskEvents } from '@/hooks/useTasks'
import type { Event, EventType } from '@/types'

// ─── 配置 ───

const eventConfig: Record<EventType, { icon: typeof Circle; color: string; label: string }> = {
  task_created: { icon: Circle, color: 'text-info', label: '创建了任务' },
  task_status_changed: { icon: Cog, color: 'text-warning', label: '更新了任务状态' },
  todo_assigned: { icon: UserCircle, color: 'text-info', label: '分配了 Todo' },
  todo_started: { icon: PlayCircle, color: 'text-info', label: '开始执行 Todo' },
  todo_progress: { icon: Cog, color: 'text-muted-foreground', label: '执行进度更新' },
  todo_completed: { icon: CheckCircle2, color: 'text-success', label: '完成了 Todo' },
  todo_failed: { icon: AlertCircle, color: 'text-destructive', label: 'Todo 执行失败' },
  task_comment: { icon: MessageSquare, color: 'text-muted-foreground', label: '评论' },
  conversation_reply: { icon: MessageSquare, color: 'text-info', label: '回复了对话' },
  agent_status_changed: { icon: Radio, color: 'text-warning', label: '状态变更' },
}

const actorIcons: Record<string, typeof UserCircle> = {
  user: UserCircle,
  agent: Bot,
  system: Cog,
}

const actorRoleBadge: Record<string, { label: string; className: string } | null> = {
  user: null,
  agent: { label: 'Agent', className: 'bg-muted text-muted-foreground' },
  system: { label: '系统', className: 'bg-muted text-muted-foreground' },
}

const taskStatusBadge: Record<string, { label: string; variant: 'secondary' | 'info' | 'success' | 'destructive' }> = {
  pending: { label: '待处理', variant: 'secondary' },
  in_progress: { label: '进行中', variant: 'info' },
  done: { label: '已完成', variant: 'success' },
  failed: { label: '失败', variant: 'destructive' },
  canceled: { label: '已取消', variant: 'secondary' },
}

const agentStatusBadge: Record<string, { label: string; className: string }> = {
  online: { label: '在线', className: 'bg-status-online/15 text-status-online' },
  offline: { label: '离线', className: 'bg-status-offline/15 text-status-offline' },
  busy: { label: '忙碌', className: 'bg-status-busy/15 text-status-busy' },
}

// ─── 工具函数 ───

function isSameDay(a: string, b: string): boolean {
  return a.slice(0, 10) === b.slice(0, 10)
}

function formatDate(dateStr: string): string {
  const date = new Date(dateStr)
  const today = new Date()
  const yesterday = new Date(today)
  yesterday.setDate(yesterday.getDate() - 1)

  if (date.toDateString() === today.toDateString()) return '今天'
  if (date.toDateString() === yesterday.toDateString()) return '昨天'
  return date.toLocaleDateString('zh-CN', { month: 'long', day: 'numeric', weekday: 'short' })
}

function shouldGroupWithPrev(current: Event, prev: Event | undefined): boolean {
  if (!prev) return false
  if (current.actor_id !== prev.actor_id) return false
  if (current.event_type !== prev.event_type) return false
  const diffMs = new Date(current.created_at).getTime() - new Date(prev.created_at).getTime()
  return diffMs < 2 * 60 * 1000 // 2 分钟内
}

// ─── 引用块组件 ───

function QuoteBlock({ children, color = 'border-muted-foreground/30' }: { children: React.ReactNode; color?: string }) {
  return (
    <div className={cn('mt-1.5 border-l-2 pl-3 py-1', color)}>
      {children}
    </div>
  )
}

// ─── 状态转换 ───

function StatusTransition({ from, to }: { from: string; to: string }) {
  const fromCfg = taskStatusBadge[from]
  const toCfg = taskStatusBadge[to]
  if (!fromCfg || !toCfg) return null

  return (
    <div className="flex items-center gap-1.5">
      <Badge variant={fromCfg.variant} className="text-[10px] px-1.5 py-0">
        {fromCfg.label}
      </Badge>
      <ArrowRight className="size-3 text-muted-foreground" />
      <Badge variant={toCfg.variant} className="text-[10px] px-1.5 py-0">
        {toCfg.label}
      </Badge>
    </div>
  )
}

function AgentStatusTransition({ from, to }: { from: string; to: string }) {
  const fromCfg = agentStatusBadge[from]
  const toCfg = agentStatusBadge[to]
  if (!fromCfg || !toCfg) return null

  return (
    <div className="flex items-center gap-1.5">
      <Badge className={cn('text-[10px] px-1.5 py-0', fromCfg.className)} variant="outline">
        {fromCfg.label}
      </Badge>
      <ArrowRight className="size-3 text-muted-foreground" />
      <Badge className={cn('text-[10px] px-1.5 py-0', toCfg.className)} variant="outline">
        {toCfg.label}
      </Badge>
    </div>
  )
}

// ─── 事件内容渲染 ───

function EventContent({ event }: { event: Event }) {
  const config = eventConfig[event.event_type] ?? { label: event.event_type }
  const todoTitle = event.metadata.todo_title as string | undefined

  if (event.event_type === 'task_comment') {
    return (
      <div className="mt-1 text-sm whitespace-pre-wrap">
        {event.content}
      </div>
    )
  }

  if (event.event_type === 'task_created') {
    const taskTitle = event.metadata.task_title as string | undefined
    return (
      <div>
        <p className="text-sm">{config.label}</p>
        {taskTitle && (
          <QuoteBlock>
            <p className="text-xs text-muted-foreground">{taskTitle}</p>
          </QuoteBlock>
        )}
      </div>
    )
  }

  if (event.event_type === 'task_status_changed') {
    const from = event.metadata.from as string | undefined
    const to = event.metadata.to as string | undefined
    return (
      <div>
        <p className="text-sm">{config.label}</p>
        {from && to && (
          <QuoteBlock color="border-warning/30">
            <StatusTransition from={from} to={to} />
          </QuoteBlock>
        )}
      </div>
    )
  }

  if (event.event_type === 'todo_assigned') {
    return (
      <div>
        <p className="text-sm">{config.label}</p>
        {todoTitle && (
          <QuoteBlock color="border-info/30">
            <p className="text-xs text-muted-foreground">{todoTitle}</p>
          </QuoteBlock>
        )}
      </div>
    )
  }

  if (event.event_type === 'todo_started') {
    return (
      <div>
        <p className="text-sm">{config.label}</p>
        {todoTitle && (
          <QuoteBlock color="border-info/30">
            <p className="text-xs text-muted-foreground">{todoTitle}</p>
          </QuoteBlock>
        )}
      </div>
    )
  }

  if (event.event_type === 'todo_progress') {
    return (
      <div>
        <p className="text-sm">{config.label}</p>
        {(event.content || todoTitle) && (
          <QuoteBlock>
            {todoTitle && <p className="text-xs font-medium text-muted-foreground">{todoTitle}</p>}
            {event.content && <p className="text-xs text-muted-foreground mt-0.5">{event.content}</p>}
          </QuoteBlock>
        )}
      </div>
    )
  }

  if (event.event_type === 'todo_completed') {
    return (
      <div>
        <p className="text-sm">{config.label}</p>
        {todoTitle && (
          <QuoteBlock color="border-success/30">
            <p className="text-xs text-muted-foreground">{todoTitle}</p>
          </QuoteBlock>
        )}
      </div>
    )
  }

  if (event.event_type === 'todo_failed') {
    const error = event.metadata.error as string | undefined
    return (
      <div>
        <p className="text-sm">{config.label}</p>
        <QuoteBlock color="border-destructive/30">
          {todoTitle && <p className="text-xs font-medium text-muted-foreground">{todoTitle}</p>}
          {error && <p className="text-xs text-destructive mt-0.5">{error}</p>}
        </QuoteBlock>
      </div>
    )
  }

  if (event.event_type === 'conversation_reply') {
    return (
      <div>
        <p className="text-sm">{config.label}</p>
        {event.content && (
          <QuoteBlock color="border-info/30">
            <p className="text-xs text-muted-foreground">{event.content}</p>
          </QuoteBlock>
        )}
      </div>
    )
  }

  if (event.event_type === 'agent_status_changed') {
    const from = event.metadata.prev_status as string | undefined
    const to = event.metadata.new_status as string | undefined
    return (
      <div>
        <p className="text-sm">{config.label}</p>
        {from && to && (
          <QuoteBlock color="border-warning/30">
            <AgentStatusTransition from={from} to={to} />
          </QuoteBlock>
        )}
      </div>
    )
  }

  return <p className="text-sm">{config.label}</p>
}

// ─── 消息组件 ───

function FeedMessage({ event, showHeader }: { event: Event; showHeader: boolean }) {
  const ActorIcon = actorIcons[event.actor_type] ?? UserCircle
  const roleBadge = actorRoleBadge[event.actor_type]
  const isComment = event.event_type === 'task_comment'
  const isSystem = event.actor_type === 'system'

  if (!showHeader) {
    return (
      <div className="flex gap-3 pl-11">
        <div className="flex-1 min-w-0">
          <EventContent event={event} />
        </div>
      </div>
    )
  }

  return (
    <div className="flex items-start gap-3">
      {isSystem ? (
        <TrustMeshLogo size={32} />
      ) : (
        <div className={cn(
          'flex size-8 shrink-0 items-center justify-center rounded-full',
          isComment
            ? (event.actor_type === 'agent' ? 'bg-muted' : 'bg-primary/10')
            : 'bg-muted'
        )}>
          <ActorIcon className="size-4 text-muted-foreground" />
        </div>
      )}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-1.5">
          <span className="text-sm font-semibold">{isSystem ? 'TrustMesh' : event.actor_name}</span>
          {roleBadge && (
            <span className={cn('text-[10px] px-1.5 py-0 rounded-sm font-medium', roleBadge.className)}>
              {roleBadge.label}
            </span>
          )}
          <span className="text-xs text-muted-foreground ml-auto shrink-0">
            {formatRelativeTime(event.created_at)}
          </span>
        </div>
        <EventContent event={event} />
      </div>
    </div>
  )
}

// ─── 日期分隔线 ───

function DateSeparator({ dateStr }: { dateStr: string }) {
  return (
    <div className="flex items-center gap-3 py-2">
      <div className="flex-1 h-px bg-border" />
      <span className="text-[11px] text-muted-foreground font-medium">{formatDate(dateStr)}</span>
      <div className="flex-1 h-px bg-border" />
    </div>
  )
}

// ─── 主组件 ───

interface TaskFeedProps {
  taskId: string
}

export function TaskFeed({ taskId }: TaskFeedProps) {
  const { data: events, isLoading } = useTaskEvents(taskId)
  const scrollRef = useRef<HTMLDivElement>(null)
  const bottomRef = useRef<HTMLDivElement>(null)
  const [isAtBottom, setIsAtBottom] = useState(true)
  const prevCountRef = useRef(0)
  const initialScrollDone = useRef(false)

  const scrollToBottom = useCallback((smooth = true) => {
    bottomRef.current?.scrollIntoView({ behavior: smooth ? 'smooth' : 'instant' })
  }, [])

  const handleScroll = useCallback(() => {
    const el = scrollRef.current
    if (!el) return
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 50
    setIsAtBottom(atBottom)
  }, [])

  // 首次加载跳到底部
  useEffect(() => {
    if (events && events.length > 0 && !initialScrollDone.current) {
      initialScrollDone.current = true
      requestAnimationFrame(() => scrollToBottom(false))
    }
  }, [events, scrollToBottom])

  // 新数据到达时自动滚动
  useEffect(() => {
    if (!events) return
    if (events.length > prevCountRef.current && isAtBottom && initialScrollDone.current) {
      requestAnimationFrame(() => scrollToBottom(true))
    }
    prevCountRef.current = events.length
  }, [events, isAtBottom, scrollToBottom])

  // taskId 变化时重置
  useEffect(() => {
    initialScrollDone.current = false
    prevCountRef.current = 0
    setIsAtBottom(true)
  }, [taskId])

  if (isLoading) {
    return <div className="flex items-center justify-center h-full text-sm text-muted-foreground">加载中...</div>
  }

  if (!events || events.length === 0) {
    return <div className="flex items-center justify-center h-full text-sm text-muted-foreground">暂无动态</div>
  }

  return (
    <div className="relative h-full">
      <div
        ref={scrollRef}
        className="h-full overflow-y-auto px-5 py-4"
        onScroll={handleScroll}
      >
        <div className="flex flex-col gap-1">
          {events.map((event, index) => {
            const prev = index > 0 ? events[index - 1] : undefined
            const showDateSep = !prev || !isSameDay(prev.created_at, event.created_at)
            const showHeader = !shouldGroupWithPrev(event, prev) || showDateSep

            return (
              <div key={event.id}>
                {showDateSep && <DateSeparator dateStr={event.created_at} />}
                <div className={cn('py-1.5 rounded-md hover:bg-accent/30 px-2 -mx-2 transition-colors', !showHeader && 'pt-0')}>
                  <FeedMessage event={event} showHeader={showHeader} />
                </div>
              </div>
            )
          })}
        </div>
        <div ref={bottomRef} />
      </div>

      {!isAtBottom && (
        <Button
          size="sm"
          variant="secondary"
          className="absolute bottom-3 left-1/2 -translate-x-1/2 shadow-md gap-1 text-xs"
          onClick={() => scrollToBottom(true)}
        >
          <ArrowDown className="size-3" />
          新消息
        </Button>
      )}
    </div>
  )
}
