import { useEffect } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { subscribeSSE } from '@/lib/sse'
import { usePageVisibility } from './usePageVisibility'
import type {
  ConversationListItem,
  ConversationStreamSnapshot,
  TaskListItem,
  TaskStreamSnapshot,
} from '@/types'

function toConversationListItem(snapshot: ConversationStreamSnapshot): ConversationListItem {
  const { conversation } = snapshot
  const lastMessage = conversation.messages[conversation.messages.length - 1]

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

function toTaskListItem(snapshot: TaskStreamSnapshot): TaskListItem {
  const { task } = snapshot
  const completedTodoCount = task.todos.filter((todo) => todo.status === 'done').length
  const failedTodoCount = task.todos.filter((todo) => todo.status === 'failed').length

  return {
    id: task.id,
    project_id: task.project_id,
    conversation_id: task.conversation_id,
    title: task.title,
    description: task.description,
    status: task.status,
    priority: task.priority,
    pm_agent: task.pm_agent,
    todo_count: task.todos.length,
    completed_todo_count: completedTodoCount,
    failed_todo_count: failedTodoCount,
    created_at: task.created_at,
    updated_at: task.updated_at,
  }
}

function upsertById<T extends { id: string }>(items: T[] | undefined, nextItem: T) {
  if (!items || items.length === 0) {
    return [nextItem]
  }

  const index = items.findIndex((item) => item.id === nextItem.id)
  if (index < 0) {
    return [nextItem, ...items]
  }

  const next = items.slice()
  next[index] = nextItem
  return next
}

export function useConversationStream(conversationId: string | undefined, enabled = true) {
  const queryClient = useQueryClient()
  const isPageVisible = usePageVisibility()

  useEffect(() => {
    if (!conversationId || !enabled || !isPageVisible) {
      return undefined
    }

    return subscribeSSE<ConversationStreamSnapshot>({
      path: `conversations/${conversationId}/stream`,
      onMessage: (snapshot) => {
        queryClient.setQueryData(['conversations', 'detail', snapshot.conversation.id], snapshot.conversation)
        queryClient.setQueriesData<ConversationListItem[] | undefined>(
          { queryKey: ['conversations', snapshot.conversation.project_id] },
          (items) => upsertById(items, toConversationListItem(snapshot))
        )
      },
    })
  }, [conversationId, enabled, isPageVisible, queryClient])
}

export function useTaskStream(taskId: string | undefined, enabled = true) {
  const queryClient = useQueryClient()
  const isPageVisible = usePageVisibility()

  useEffect(() => {
    if (!taskId || !enabled || !isPageVisible) {
      return undefined
    }

    return subscribeSSE<TaskStreamSnapshot>({
      path: `tasks/${taskId}/stream`,
      onMessage: (snapshot) => {
        queryClient.setQueryData(['tasks', 'detail', snapshot.task.id], snapshot.task)
        queryClient.setQueryData(['tasks', 'events', snapshot.task.id], snapshot.events)
        queryClient.setQueriesData<TaskListItem[] | undefined>(
          { queryKey: ['tasks', snapshot.task.project_id] },
          (items) => upsertById(items, toTaskListItem(snapshot))
        )
      },
    })
  }, [enabled, isPageVisible, queryClient, taskId])
}
