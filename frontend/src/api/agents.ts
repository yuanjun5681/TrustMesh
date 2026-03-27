import { api } from './client'
import type {
  ApiResponse,
  ApiListResponse,
  Agent,
  AgentInsights,
  AgentStats,
  AgentTaskItem,
  CreateAgentRequest,
  UpdateAgentRequest,
} from '@/types'

export async function createAgent(input: CreateAgentRequest) {
  return api.post('agents', { json: input }).json<ApiResponse<Agent>>()
}

export async function listAgents() {
  return api.get('agents').json<ApiListResponse<Agent>>()
}

export async function getAgent(id: string) {
  return api.get(`agents/${id}`).json<ApiResponse<Agent>>()
}

export async function updateAgent(id: string, input: UpdateAgentRequest) {
  return api.patch(`agents/${id}`, { json: input }).json<ApiResponse<Agent>>()
}

export async function deleteAgent(id: string) {
  await api.delete(`agents/${id}`)
}

export async function getAgentStats(id: string) {
  return api.get(`agents/${id}/stats`).json<ApiResponse<AgentStats>>()
}

export async function getAgentInsights(id: string) {
  return api.get(`agents/${id}/insights`).json<ApiResponse<AgentInsights>>()
}

export async function listAgentTasks(id: string, status?: string) {
  const searchParams: Record<string, string> = {}
  if (status) searchParams.status = status
  return api.get(`agents/${id}/tasks`, { searchParams }).json<ApiListResponse<AgentTaskItem>>()
}
