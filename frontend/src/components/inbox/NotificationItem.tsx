import { cn } from '@/lib/utils'
import { formatRelativeTime } from '@/lib/utils'
import type { Notification } from '@/types'

const priorityColors: Record<string, string> = {
  high: 'text-destructive',
  medium: 'text-warning',
  low: 'text-muted-foreground',
}

interface NotificationItemProps {
  notification: Notification
  onMarkRead?: (id: string) => void
}

export function NotificationItem({ notification, onMarkRead }: NotificationItemProps) {
  return (
    <div
      className={cn(
        'flex gap-3 p-3 rounded-lg transition-colors cursor-pointer hover:bg-accent/50',
        !notification.is_read && 'bg-accent/30'
      )}
      onClick={() => {
        if (!notification.is_read && onMarkRead) {
          onMarkRead(notification.id)
        }
      }}
    >
      <div className="flex items-start gap-2 pt-0.5">
        {!notification.is_read && (
          <span className="inline-block h-2 w-2 rounded-full bg-primary mt-1.5 shrink-0" />
        )}
        {notification.is_read && <span className="inline-block h-2 w-2 shrink-0" />}
        <span
          className={cn(
            'inline-block h-1.5 w-1.5 rounded-full mt-2 shrink-0',
            priorityColors[notification.priority] === 'text-destructive' && 'bg-destructive',
            priorityColors[notification.priority] === 'text-warning' && 'bg-warning',
            priorityColors[notification.priority] === 'text-muted-foreground' && 'bg-muted-foreground'
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
        <p className="text-sm text-muted-foreground mt-0.5 truncate">{notification.body}</p>
      </div>
    </div>
  )
}
