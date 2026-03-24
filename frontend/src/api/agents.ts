import { api } from './client'
import type {
  ApiResponse,
  ApiListResponse,
  Agent,
  AgentStats,
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
