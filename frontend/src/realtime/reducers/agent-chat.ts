import type { QueryClient } from '@tanstack/react-query'
import type { AgentChatDetail } from '@/types'

export function applyAgentChatUpdated(
  queryClient: QueryClient,
  payload: { chat: AgentChatDetail }
) {
  queryClient.setQueryData(['agents', payload.chat.agent_id, 'chat'], payload.chat)
}
