import { useState } from 'react'
import { Link } from 'react-router-dom'
import { ExternalLink } from 'lucide-react'
import { cn } from '@/lib/utils'
import { formatRelativeTime } from '@/lib/utils'
import type { Notification } from '@/types'

const priorityColors: Record<string, string> = {
  high: 'text-destructive',
  medium: 'text-warning',
  low: 'text-muted-foreground',
}

const priorityDotColors: Record<string, string> = {
  high: 'bg-destructive',
  medium: 'bg-warning',
  low: 'bg-muted-foreground',
}

const categoryLabels: Record<string, { text: string; path: (n: Notification) => string | null }> = {
  task: {
    text: '查看任务',
    path: (n) => n.task_id ? `/projects/${n.project_id}` : null,
  },
  todo: {
    text: '查看任务',
    path: (n) => n.task_id ? `/projects/${n.project_id}` : null,
  },
  conversation: {
    text: '查看对话',
    path: (n) => `/projects/${n.project_id}/chat`,
  },
  system: {
    text: '',
    path: () => null,
  },
}

interface NotificationItemProps {
  notification: Notification
  onMarkRead?: (id: string) => void
}

export function NotificationItem({ notification, onMarkRead }: NotificationItemProps) {
  const [expanded, setExpanded] = useState(false)

  const handleClick = () => {
    if (!notification.is_read && onMarkRead) {
      onMarkRead(notification.id)
    }
    setExpanded(!expanded)
  }

  const linkInfo = categoryLabels[notification.category]
  const linkPath = linkInfo?.path(notification)

  return (
    <div
      className={cn(
        'rounded-lg transition-colors cursor-pointer hover:bg-accent/50',
        !notification.is_read && 'bg-accent/30'
      )}
      onClick={handleClick}
    >
      {/* 折叠态：标题 + 截断 body */}
      <div className="flex gap-3 p-3">
        <div className="flex items-start gap-2 pt-0.5">
          {!notification.is_read ? (
            <span className="inline-block h-2 w-2 rounded-full bg-primary mt-1.5 shrink-0" />
          ) : (
            <span className="inline-block h-2 w-2 shrink-0" />
          )}
          <span
            className={cn(
              'inline-block h-1.5 w-1.5 rounded-full mt-2 shrink-0',
              priorityDotColors[notification.priority]
            )}
          />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className={cn('text-sm font-medium', priorityColors[notification.priority])}>
              {notification.title}
            </span>
            <span className="text-xs text-muted-foreground ml-auto shrink-0">
              {formatRelativeTime(notification.created_at)}
            </span>
          </div>
          <p className={cn('text-sm text-muted-foreground mt-0.5', !expanded && 'truncate')}>
            {notification.body}
          </p>
        </div>
      </div>

      {/* 展开态：完整内容 + 跳转链接 */}
      {expanded && linkPath && (
        <div className="px-3 pb-3 ml-[26px]">
          <Link
            to={linkPath}
            className="inline-flex items-center gap-1 text-xs text-primary hover:underline"
            onClick={(e) => e.stopPropagation()}
          >
            <ExternalLink className="h-3 w-3" />
            {linkInfo.text}
          </Link>
        </div>
      )}
    </div>
  )
}
