import { X, MessageSquare } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { TaskStatusBadge, PriorityBadge } from '@/components/shared/StatusBadge'
import { TaskFeed } from './TaskFeed'
import { TaskResultView } from './TaskResult'
import { TaskDescription } from './TaskDescription'
import { TaskCommentComposer, type TaskCommentSubmitInput, type TaskMentionCandidate } from './TaskCommentComposer'
import { CancelTaskDialog } from './CancelTaskDialog'
import { ConversationSheet } from '@/components/conversation/ConversationSheet'
import { useTask, useAddTaskComment } from '@/hooks/useTasks'
import { ScrollArea } from '@/components/ui/scroll-area'
import { ApiRequestError } from '@/api/client'
import { useState } from 'react'
import type { TaskDetail } from '@/types'

interface TaskDetailPanelProps {
  taskId: string
  onClose: () => void
}

function buildTaskMentionCandidates(task: TaskDetail | undefined): TaskMentionCandidate[] {
  if (!task) {
    return []
  }

  const seen = new Set<string>()
  const candidates: TaskMentionCandidate[] = []

  if (task.pm_agent.id && !seen.has(task.pm_agent.id)) {
    candidates.push({
      id: task.pm_agent.id,
      name: task.pm_agent.name,
      roleLabel: 'PM Agent',
    })
    seen.add(task.pm_agent.id)
  }

  for (const todo of task.todos) {
    if (!todo.assignee.agent_id || seen.has(todo.assignee.agent_id)) {
      continue
    }
    candidates.push({
      id: todo.assignee.agent_id,
      name: todo.assignee.name,
      roleLabel: '执行 Agent',
    })
    seen.add(todo.assignee.agent_id)
  }

  return candidates
}

export function TaskDetailPanel({ taskId, onClose }: TaskDetailPanelProps) {
  const { data: task } = useTask(taskId)
  const [tab, setTab] = useState('feed')
  const [chatOpen, setChatOpen] = useState(false)
  const [cancelDialogOpen, setCancelDialogOpen] = useState(false)
  const addComment = useAddTaskComment()
  const canCancelTask = task?.status === 'pending' || task?.status === 'in_progress'
  const mentionCandidates = buildTaskMentionCandidates(task)

  const handleSubmitComment = async ({ content, mentionAgentIds }: TaskCommentSubmitInput) => {
    try {
      const response = await addComment.mutateAsync({ taskId, content, mentionAgentIds })
      const failedDeliveries = response.data.mention_deliveries?.filter((item) => item.status !== 'sent') ?? []

      if (failedDeliveries.length === 1) {
        toast.warning(`评论已发布，但 @${failedDeliveries[0].agent_name} 发送失败`)
      } else if (failedDeliveries.length > 1) {
        toast.warning(`评论已发布，但有 ${failedDeliveries.length} 个 Agent 未收到 mention`)
      }

      return true
    } catch (error) {
      const message = error instanceof ApiRequestError ? error.message : '发表评论失败'
      toast.error(message)
      return false
    }
  }

  if (!task) {
    return (
      <div className="flex items-center justify-center h-full text-sm text-muted-foreground">
        加载中...
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center justify-between px-5 py-3 border-b shrink-0">
        <div className="flex items-center gap-2">
          <TaskStatusBadge status={task.status} />
          <PriorityBadge priority={task.priority} />
        </div>
        <div className="flex items-center gap-1">
          <Button
            variant="outline"
            size="sm"
            disabled={!canCancelTask}
            onClick={() => setCancelDialogOpen(true)}
          >
            终止任务
          </Button>
          <Button variant="ghost" size="icon" className="size-7" onClick={() => setChatOpen(true)} title="查看关联对话">
            <MessageSquare className="size-4" />
          </Button>
          <Button variant="ghost" size="icon" className="size-7" onClick={onClose}>
            <X className="size-4" />
          </Button>
        </div>
      </div>

      {/* Task info */}
      <div className="px-5 py-4 shrink-0 border-b">
        <h2 className="text-lg font-semibold">{task.title}</h2>
        {task.description && (
          <TaskDescription description={task.description} />
        )}
        {task.cancel_reason && (
          <p className="mt-2 rounded-md bg-muted px-3 py-2 text-xs text-muted-foreground">
            终止原因：{task.cancel_reason}
          </p>
        )}
      </div>

      {/* Tabs */}
      <Tabs value={tab} onValueChange={setTab} className="flex flex-col flex-1 min-h-0">
        <TabsList className="mx-5 mt-3 shrink-0">
          <TabsTrigger value="feed">动态</TabsTrigger>
          <TabsTrigger value="result">交付成果</TabsTrigger>
        </TabsList>

        <TabsContent value="feed" className="flex-1 min-h-0 mt-0">
          <TaskFeed taskId={task.id} />
        </TabsContent>

        <TabsContent value="result" className="flex-1 min-h-0 mt-0">
          <ScrollArea className="h-full">
            <div className="px-5 py-3">
              <TaskResultView taskId={task.id} result={task.result} artifacts={task.artifacts} />
            </div>
          </ScrollArea>
        </TabsContent>
      </Tabs>

      {/* Comment input */}
      <div className="border-t px-4 py-3 shrink-0">
        <TaskCommentComposer
          candidates={mentionCandidates}
          disabled={addComment.isPending}
          onSubmit={handleSubmitComment}
        />
      </div>

      <ConversationSheet
        projectId={task.project_id}
        initialConversationId={task.conversation_id}
        open={chatOpen}
        onOpenChange={setChatOpen}
      />
      <CancelTaskDialog
        open={cancelDialogOpen}
        onOpenChange={setCancelDialogOpen}
        task={task}
      />
    </div>
  )
}
