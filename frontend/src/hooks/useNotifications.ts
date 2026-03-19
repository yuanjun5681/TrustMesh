import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as notificationsApi from '@/api/notifications'

export function useNotifications(filter = 'recent', limit = 50) {
  return useQuery({
    queryKey: ['notifications', filter, limit],
    queryFn: async () => {
      const res = await notificationsApi.listNotifications(filter, limit)
      return res.data.items
    },
    staleTime: 15_000,
    refetchInterval: 30_000,
  })
}

export function useUnreadCount() {
  return useQuery({
    queryKey: ['notifications', 'unread-count'],
    queryFn: async () => {
      const res = await notificationsApi.getUnreadCount()
      return res.data.count
    },
    staleTime: 15_000,
    refetchInterval: 30_000,
  })
}

export function useMarkNotificationRead() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => notificationsApi.markNotificationRead(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['notifications'] })
    },
  })
}

export function useMarkAllRead() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => notificationsApi.markAllNotificationsRead(),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['notifications'] })
    },
  })
}
