import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as tasksApi from '@/api/tasks'
import { usePageVisibility } from './usePageVisibility'
import { useRealtimeStatus } from '@/realtime/hooks/useRealtimeStatus'
import type { ListProjectTasksQuery, TaskDetail } from '@/types'

function normalizeTaskDetail(task: TaskDetail): TaskDetail {
  return {
    ...task,
    todos: task.todos ?? [],
    artifacts: task.artifacts ?? [],
  }
}

export function useTasks(projectId: string | undefined, query?: ListProjectTasksQuery) {
  return useQuery({
    queryKey: ['tasks', projectId, query],
    queryFn: async () => {
      const res = await tasksApi.listProjectTasks(projectId!, query)
      return res.data.items
    },
    enabled: !!projectId,
    staleTime: 30_000,
  })
}

export function useTask(id: string | undefined) {
  const isPageVisible = usePageVisibility()
  const realtimeStatus = useRealtimeStatus()

  return useQuery({
    queryKey: ['tasks', 'detail', id],
    queryFn: async () => {
      const res = await tasksApi.getTask(id!)
      return normalizeTaskDetail(res.data)
    },
    enabled: !!id,
    staleTime: 5_000,
    refetchInterval: (currentQuery) => {
      if (!isPageVisible || !id) {
        return false
      }
      if (realtimeStatus !== 'reconnecting' && realtimeStatus !== 'disconnected') {
        return false
      }

      const task = currentQuery.state.data as TaskDetail | undefined
      return !task || task.status === 'pending' || task.status === 'in_progress' ? 3_000 : false
    },
    refetchIntervalInBackground: false,
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

export function useTaskComments(taskId: string | undefined) {
  return useQuery({
    queryKey: ['tasks', 'comments', taskId],
    queryFn: async () => {
      const res = await tasksApi.listTaskComments(taskId!)
      return res.data.items
    },
    enabled: !!taskId,
  })
}

export function useAddTaskComment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ taskId, content, todoId }: { taskId: string; content: string; todoId?: string }) =>
      tasksApi.addTaskComment(taskId, content, todoId),
    onSuccess: (_res, { taskId }) => {
      qc.invalidateQueries({ queryKey: ['tasks', 'comments', taskId] })
      qc.invalidateQueries({ queryKey: ['tasks', 'events', taskId] })
    },
  })
}

export function useDispatchTodo() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ taskId, todoId }: { taskId: string; todoId: string }) =>
      tasksApi.dispatchTodo(taskId, todoId),
    onSuccess: (_res, { taskId }) => {
      qc.invalidateQueries({ queryKey: ['tasks'] })
      qc.invalidateQueries({ queryKey: ['tasks', 'detail', taskId] })
      qc.invalidateQueries({ queryKey: ['tasks', 'events', taskId] })
    },
  })
}

export function useCancelTask() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ taskId, reason }: { taskId: string; reason: string }) =>
      tasksApi.cancelTask(taskId, reason),
    onSuccess: (res, { taskId }) => {
      qc.setQueryData(['tasks', 'detail', taskId], res.data)
      qc.invalidateQueries({ queryKey: ['tasks'] })
      qc.invalidateQueries({ queryKey: ['tasks', 'detail', taskId] })
      qc.invalidateQueries({ queryKey: ['tasks', 'events', taskId] })
      qc.invalidateQueries({ queryKey: ['projects'] })
      qc.invalidateQueries({ queryKey: ['dashboard', 'tasks'] })
      qc.invalidateQueries({ queryKey: ['dashboard', 'stats'] })
    },
  })
}
