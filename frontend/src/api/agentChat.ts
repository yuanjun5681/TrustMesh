import { api } from './client'
import type { AgentChatDetail, ApiResponse } from '@/types'

export async function getAgentChat(agentId: string) {
  return api.get(`agents/${agentId}/chat`).json<ApiResponse<AgentChatDetail | null>>()
}

export async function sendAgentChatMessage(agentId: string, content: string) {
  return api.post(`agents/${agentId}/chat/messages`, { json: { content } }).json<ApiResponse<AgentChatDetail>>()
}

export async function resetAgentChat(agentId: string) {
  return api.post(`agents/${agentId}/chat/reset`).json<ApiResponse<AgentChatDetail>>()
}
