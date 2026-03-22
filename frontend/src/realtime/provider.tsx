import { useEffect, useState } from 'react'
import type { ReactNode } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { subscribeSSE } from '@/lib/sse'
import { useAuthStore } from '@/stores/authStore'
import type { RealtimeEvent } from './types'
import { RealtimeStatusContext } from './context'
import type { RealtimeStatus } from './context'
import {
  applyNotificationCreated,
  applyNotificationRead,
  applyNotificationsAllRead,
} from './reducers/notifications'
import { applyTaskCommentCreated, applyTaskEventCreated, applyTaskUpdated } from './reducers/tasks'
import { applyAgentStatusChanged } from './reducers/agents'
import { applyConversationUpdated } from './reducers/conversations'

export function RealtimeProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient()
  const isAuthenticated = useAuthStore((state) => !!state.token)
  const [status, setStatus] = useState<RealtimeStatus>('idle')

  useEffect(() => {
    if (!isAuthenticated) {
      return undefined
    }

    const connectingHandle = window.setTimeout(() => {
      setStatus('connecting')
    }, 0)

    const stop = subscribeSSE<RealtimeEvent>({
      path: 'events/stream',
      onOpen: () => {
        setStatus('connected')
        void queryClient.invalidateQueries({ queryKey: ['notifications'] })
        void queryClient.invalidateQueries({ queryKey: ['dashboard'] })
        void queryClient.invalidateQueries({ queryKey: ['projects'] })
        void queryClient.invalidateQueries({ queryKey: ['agents'] })
        void queryClient.invalidateQueries({ queryKey: ['tasks'] })
        void queryClient.invalidateQueries({ queryKey: ['conversations'] })
      },
      onError: () => {
        setStatus((current) => (current === 'connected' ? 'reconnecting' : 'disconnected'))
      },
      onMessage: (event) => {
        switch (event.type) {
          case 'notification.created':
            applyNotificationCreated(queryClient, event.payload)
            break
          case 'notification.read':
            applyNotificationRead(queryClient, event.payload)
            break
          case 'notifications.all_read':
            applyNotificationsAllRead(queryClient, event.payload)
            break
          case 'task.updated':
            applyTaskUpdated(queryClient, event.payload)
            break
          case 'task.event.created':
            applyTaskEventCreated(queryClient, event.payload)
            break
          case 'task.comment.created':
            applyTaskCommentCreated(queryClient, event.payload)
            break
          case 'agent.status.changed':
            applyAgentStatusChanged(queryClient, event.payload)
            break
          case 'conversation.updated':
            applyConversationUpdated(queryClient, event.payload)
            break
        }
      },
    })
    return () => {
      window.clearTimeout(connectingHandle)
      stop()
    }
  }, [isAuthenticated, queryClient])

  return (
    <RealtimeStatusContext.Provider value={isAuthenticated ? status : 'idle'}>
      {children}
    </RealtimeStatusContext.Provider>
  )
}
