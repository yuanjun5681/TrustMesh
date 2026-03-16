import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import * as conversationsApi from '@/api/conversations'
import type { CreateConversationRequest, AppendConversationMessageRequest } from '@/types'

export function useConversations(projectId: string | undefined) {
  return useQuery({
    queryKey: ['conversations', projectId],
    queryFn: async () => {
      const res = await conversationsApi.listProjectConversations(projectId!)
      return res.data.items
    },
    enabled: !!projectId,
  })
}

export function useConversation(id: string | undefined, isActive: boolean) {
  return useQuery({
    queryKey: ['conversations', 'detail', id],
    queryFn: async () => {
      const res = await conversationsApi.getConversation(id!)
      return res.data
    },
    enabled: !!id,
    refetchInterval: isActive ? 3000 : false,
  })
}

export function useCreateConversation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ projectId, input }: { projectId: string; input: CreateConversationRequest }) =>
      conversationsApi.createConversation(projectId, input),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: ['conversations', variables.projectId] })
    },
  })
}

export function useAppendMessage() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: AppendConversationMessageRequest }) =>
      conversationsApi.appendConversationMessage(id, input),
    onSuccess: (data) => {
      qc.setQueryData(['conversations', 'detail', data.data.id], data.data)
    },
  })
}
