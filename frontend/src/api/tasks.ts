import { api } from './client'
import type {
  ApiResponse,
  ApiListResponse,
  TaskListItem,
  TaskDetail,
  TaskEvent,
  ListProjectTasksQuery,
} from '@/types'

export async function listProjectTasks(projectId: string, query?: ListProjectTasksQuery) {
  const searchParams: Record<string, string> = {}
  if (query?.status) searchParams.status = query.status
  return api
    .get(`projects/${projectId}/tasks`, { searchParams })
    .json<ApiListResponse<TaskListItem>>()
}

export async function getTask(id: string) {
  return api.get(`tasks/${id}`).json<ApiResponse<TaskDetail>>()
}

export async function listTaskEvents(id: string) {
  return api.get(`tasks/${id}/events`).json<ApiListResponse<TaskEvent>>()
}

export async function dispatchTodo(taskId: string, todoId: string) {
  return api.post(`tasks/${taskId}/todos/${todoId}/dispatch`).json<ApiResponse<TaskDetail>>()
}
