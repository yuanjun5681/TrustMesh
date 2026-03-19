import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { NotificationGroup } from '@/components/inbox/NotificationGroup'
import { groupNotificationsByDate } from '@/lib/notifications'
import { useNotifications, useMarkNotificationRead, useMarkAllRead } from '@/hooks/useNotifications'

const filters = [
  { label: '最近', value: 'recent' },
  { label: '未读', value: 'unread' },
  { label: '全部', value: 'all' },
] as const

export function InboxPage() {
  const [filter, setFilter] = useState<string>('recent')
  const { data: notifications, isLoading } = useNotifications(filter)
  const markRead = useMarkNotificationRead()
  const markAllRead = useMarkAllRead()

  const groups = notifications ? groupNotificationsByDate(notifications) : []
  const hasUnread = notifications?.some((n) => !n.is_read) ?? false

  return (
    <div className="p-6 space-y-4 max-w-3xl mx-auto">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">收件箱</h1>
        {hasUnread && (
          <Button
            variant="ghost"
            size="sm"
            onClick={() => markAllRead.mutate()}
            disabled={markAllRead.isPending}
          >
            全部已读
          </Button>
        )}
      </div>

      <div className="flex gap-1">
        {filters.map((f) => (
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

      {isLoading && (
        <div className="py-12 text-center text-sm text-muted-foreground">加载中...</div>
      )}

      {!isLoading && groups.length === 0 && (
        <div className="py-12 text-center text-sm text-muted-foreground">
          {filter === 'unread' ? '没有未读通知' : '暂无通知'}
        </div>
      )}

      {!isLoading && groups.length > 0 && (
        <ScrollArea className="h-[calc(100vh-180px)]">
          <div className="space-y-4">
            {groups.map((group) => (
              <NotificationGroup
                key={group.label}
                label={group.label}
                notifications={group.items}
                onMarkRead={(id) => markRead.mutate(id)}
              />
            ))}
          </div>
        </ScrollArea>
      )}
    </div>
  )
}
