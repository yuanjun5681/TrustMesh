import type { QueryClient } from '@tanstack/react-query'
import type { ConversationDetail, ConversationListItem } from '@/types'

function toConversationListItem(conversation: ConversationDetail): ConversationListItem | null {
  const lastMessage = conversation.messages[conversation.messages.length - 1]
  if (!lastMessage) {
    return null
  }

  return {
    id: conversation.id,
    project_id: conversation.project_id,
    status: conversation.status,
    last_message: lastMessage,
    linked_task: conversation.linked_task,
    created_at: conversation.created_at,
    updated_at: conversation.updated_at,
  }
}

function sortConversationsDescending(items: ConversationListItem[]) {
  return items.slice().sort((left, right) => right.updated_at.localeCompare(left.updated_at))
}

export function applyConversationUpdated(
  queryClient: QueryClient,
  payload: { conversation: ConversationDetail }
) {
  const { conversation } = payload
  const listItem = toConversationListItem(conversation)

  // Cancel any in-flight refetch so it doesn't overwrite this SSE-delivered data
  void queryClient.cancelQueries({ queryKey: ['conversations', 'detail', conversation.id] })
  queryClient.setQueryData(['conversations', 'detail', conversation.id], conversation)

  if (!listItem) {
    return
  }

  queryClient.setQueryData<ConversationListItem[] | undefined>(['conversations', conversation.project_id], (items) => {
    if (!items || items.length === 0) {
      return [listItem]
    }

    const index = items.findIndex((item) => item.id === listItem.id)
    const next = items.slice()
    if (index < 0) {
      next.unshift(listItem)
    } else {
      next[index] = listItem
    }
    return sortConversationsDescending(next)
  })
}
