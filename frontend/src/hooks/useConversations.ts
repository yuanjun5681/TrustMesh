import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import * as conversationsApi from '@/api/conversations'
import { usePageVisibility } from './usePageVisibility'
import type {
  AppendConversationMessageRequest,
  ConversationDetail,
  CreateConversationRequest,
} from '@/types'

export function useConversations(projectId: string | undefined) {
  return useQuery({
    queryKey: ['conversations', projectId],
    queryFn: async () => {
      const res = await conversationsApi.listProjectConversations(projectId!)
      return res.data.items
    },
    enabled: !!projectId,
    staleTime: 30_000,
  })
}

export function useConversation(id: string | undefined, isActiveHint: boolean) {
  const isPageVisible = usePageVisibility()

  return useQuery({
    queryKey: ['conversations', 'detail', id],
    queryFn: async () => {
      const res = await conversationsApi.getConversation(id!)
      return res.data
    },
    enabled: !!id,
    staleTime: 5_000,
    refetchInterval: (currentQuery) => {
      if (!isPageVisible || !id) {
        return false
      }

      const conversation = currentQuery.state.data as ConversationDetail | undefined
      return conversation?.status === 'active' || (!conversation && isActiveHint) ? 3_000 : false
    },
    refetchIntervalInBackground: false,
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
