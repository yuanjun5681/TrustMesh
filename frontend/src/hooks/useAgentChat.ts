import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as agentChatApi from '@/api/agentChat'

export function useAgentChat(agentId: string | undefined) {
  return useQuery({
    queryKey: ['agents', agentId, 'chat'],
    queryFn: async () => {
      const res = await agentChatApi.getAgentChat(agentId!)
      return res.data
    },
    enabled: !!agentId,
    staleTime: 15_000,
  })
}

export function useSendAgentChatMessage() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ agentId, content }: { agentId: string; content: string }) =>
      agentChatApi.sendAgentChatMessage(agentId, content),
    onSuccess: (data, variables) => {
      qc.setQueryData(['agents', variables.agentId, 'chat'], data.data)
    },
    onSettled: (_data, _error, variables) => {
      if (!variables) return
      void qc.invalidateQueries({ queryKey: ['agents', variables.agentId, 'chat'] })
    },
  })
}

export function useResetAgentChat() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (agentId: string) => agentChatApi.resetAgentChat(agentId),
    onSuccess: (data, agentId) => {
      qc.setQueryData(['agents', agentId, 'chat'], data.data)
    },
  })
}
