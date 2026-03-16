import { api } from './client'
import type {
  ApiResponse,
  ApiListResponse,
  ConversationDetail,
  ConversationListItem,
  CreateConversationRequest,
  AppendConversationMessageRequest,
} from '@/types'

export async function createConversation(projectId: string, input: CreateConversationRequest) {
  return api
    .post(`projects/${projectId}/conversations`, { json: input })
    .json<ApiResponse<ConversationDetail>>()
}

export async function listProjectConversations(projectId: string) {
  return api
    .get(`projects/${projectId}/conversations`)
    .json<ApiListResponse<ConversationListItem>>()
}

export async function getConversation(id: string) {
  return api.get(`conversations/${id}`).json<ApiResponse<ConversationDetail>>()
}

export async function appendConversationMessage(
  id: string,
  input: AppendConversationMessageRequest
) {
  return api
    .post(`conversations/${id}/messages`, { json: input })
    .json<ApiResponse<ConversationDetail>>()
}
