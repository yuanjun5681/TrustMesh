import { useQuery } from '@tanstack/react-query'
import * as tasksApi from '@/api/tasks'
import type { ListProjectTasksQuery } from '@/types'

export function useTasks(projectId: string | undefined, query?: ListProjectTasksQuery) {
  return useQuery({
    queryKey: ['tasks', projectId, query],
    queryFn: async () => {
      const res = await tasksApi.listProjectTasks(projectId!, query)
      return res.data.items
    },
    enabled: !!projectId,
    refetchInterval: 5000,
  })
}

export function useTask(id: string | undefined) {
  return useQuery({
    queryKey: ['tasks', 'detail', id],
    queryFn: async () => {
      const res = await tasksApi.getTask(id!)
      return res.data
    },
    enabled: !!id,
    refetchInterval: 3000,
  })
}

export function useTaskEvents(id: string | undefined) {
  return useQuery({
    queryKey: ['tasks', 'events', id],
    queryFn: async () => {
      const res = await tasksApi.listTaskEvents(id!)
      return res.data.items
    },
    enabled: !!id,
  })
}
