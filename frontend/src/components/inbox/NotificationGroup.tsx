import { NotificationItem } from './NotificationItem'
import type { Notification } from '@/types'

interface NotificationGroupProps {
  label: string
  notifications: Notification[]
  onMarkRead?: (id: string) => void
}

export function NotificationGroup({ label, notifications, onMarkRead }: NotificationGroupProps) {
  if (notifications.length === 0) return null

  return (
    <div className="space-y-1">
      <div className="text-xs font-medium text-muted-foreground uppercase tracking-wider px-3 py-1">
        {label}
      </div>
      {notifications.map((n) => (
        <NotificationItem key={n.id} notification={n} onMarkRead={onMarkRead} />
      ))}
    </div>
  )
}
