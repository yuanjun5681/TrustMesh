import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as tasksApi from '@/api/tasks'
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
    refetchInterval: 5000,
  })
}

export function useTask(id: string | undefined) {
  return useQuery({
    queryKey: ['tasks', 'detail', id],
    queryFn: async () => {
      const res = await tasksApi.getTask(id!)
      return normalizeTaskDetail(res.data)
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
