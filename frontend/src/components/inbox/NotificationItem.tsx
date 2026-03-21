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

function truncateBody(body: string, maxLen = 80): string {
  // Strip markdown formatting for preview
  const plain = body.replace(/[*_~`#>\-\[\]()]/g, '').replace(/\n+/g, ' ').trim()
  return plain.length > maxLen ? plain.slice(0, maxLen) + '...' : plain
}

interface NotificationItemProps {
  notification: Notification
  onMarkRead?: (id: string) => void
  onViewConversation?: (projectId: string, conversationId?: string) => void
}

export function NotificationItem({ notification, onMarkRead, onViewConversation }: NotificationItemProps) {
  const handleClick = () => {
    if (!notification.is_read && onMarkRead) {
      onMarkRead(notification.id)
    }
  }

  const getLinkInfo = () => {
    switch (notification.category) {
      case 'task':
      case 'todo':
        return notification.task_id
          ? { text: '查看任务', path: `/projects/${notification.project_id}` }
          : null
      case 'conversation':
        return { text: '查看对话', path: null }
      default:
        return null
    }
  }

  const linkInfo = getLinkInfo()

  const handleConversationClick = (e: React.MouseEvent) => {
    e.stopPropagation()
    onViewConversation?.(notification.project_id, notification.conversation_id)
  }

  return (
    <div
      className={cn(
        'rounded-lg transition-colors cursor-pointer hover:bg-accent/50',
        !notification.is_read && 'bg-accent/30'
      )}
      onClick={handleClick}
    >
      <div className="flex gap-3 p-3">
        <div className="flex items-start gap-2 pt-0.5">
          {!notification.is_read ? (
            <span className="inline-block size-2 rounded-full bg-primary mt-1.5 shrink-0" />
          ) : (
            <span className="inline-block size-2 shrink-0" />
          )}
          <span
            className={cn(
              'inline-block size-1.5 rounded-full mt-2 shrink-0',
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
          <p className="text-sm text-muted-foreground mt-0.5 truncate">
            {truncateBody(notification.body)}
          </p>
          {linkInfo && (
            <div className="mt-1.5">
              {linkInfo.path ? (
                <Link
                  to={linkInfo.path}
                  className="inline-flex items-center gap-1 text-xs text-primary hover:underline"
                  onClick={(e) => e.stopPropagation()}
                >
                  <ExternalLink className="size-3" />
                  {linkInfo.text}
                </Link>
              ) : (
                <button
                  className="inline-flex items-center gap-1 text-xs text-primary hover:underline cursor-pointer"
                  onClick={handleConversationClick}
                >
                  <ExternalLink className="size-3" />
                  {linkInfo.text}
                </button>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
