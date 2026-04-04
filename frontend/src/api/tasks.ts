import { api } from './client'
import type {
  AddTaskCommentResult,
  AppendTaskMessageRequest,
  ApiResponse,
  ApiListResponse,
  CreatePlanningTaskRequest,
  TaskListItem,
  TaskDetail,
  TaskPriority,
  Event,
  Comment,
  ListProjectTasksQuery,
} from '@/types'

export interface CreateTaskInput {
  title: string
  description: string
  priority?: TaskPriority
  assignee_agent_id: string
}

export async function createTask(projectId: string, input: CreateTaskInput) {
  return api.post(`projects/${projectId}/tasks`, { json: input }).json<ApiResponse<TaskDetail>>()
}

export async function createPlanningTask(projectId: string, input: CreatePlanningTaskRequest) {
  return api.post(`projects/${projectId}/tasks/planning`, { json: input }).json<ApiResponse<TaskDetail>>()
}

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

export async function appendTaskMessage(taskId: string, input: AppendTaskMessageRequest) {
  return api.post(`tasks/${taskId}/messages`, { json: input }).json<ApiResponse<TaskDetail>>()
}

export async function listTaskEvents(id: string) {
  return api.get(`tasks/${id}/events`).json<ApiListResponse<Event>>()
}

export async function dispatchTodo(taskId: string, todoId: string) {
  return api.post(`tasks/${taskId}/todos/${todoId}/dispatch`).json<ApiResponse<TaskDetail>>()
}

export async function cancelTask(taskId: string, reason: string) {
  return api.post(`tasks/${taskId}/cancel`, { json: { reason } }).json<ApiResponse<TaskDetail>>()
}

export async function listTaskComments(taskId: string) {
  return api.get(`tasks/${taskId}/comments`).json<ApiListResponse<Comment>>()
}

export interface AddTaskCommentInput {
  content: string
  todo_id?: string
  mentions?: Array<{ agent_id: string }>
}

export async function addTaskComment(taskId: string, input: AddTaskCommentInput) {
  return api
    .post(`tasks/${taskId}/comments`, {
      json: input,
    })
    .json<ApiResponse<AddTaskCommentResult>>()
}

export async function getTaskArtifactContent(taskId: string, transferId: string) {
  return api.get(`tasks/${taskId}/artifacts/${transferId}/content`).blob()
}
