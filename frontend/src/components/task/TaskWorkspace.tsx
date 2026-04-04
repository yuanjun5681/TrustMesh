import { X, MessageSquare, PackageCheck } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { TaskStatusBadge, PriorityBadge } from '@/components/shared/StatusBadge'
import { TaskFeed } from './TaskFeed'
import { TaskThreadSheet } from './TaskThreadSheet'
import { TaskResultView } from './TaskResult'
import { TaskDescription } from './TaskDescription'
import { TaskComposer } from './TaskComposer'
import type { TaskCommentSubmitInput, TaskMentionCandidate } from './TaskCommentComposer'
import { CancelTaskDialog } from './CancelTaskDialog'
import { MessageBubble } from '@/components/task-thread/MessageBubble'
import { ThinkingIndicator } from '@/components/task-thread/ThinkingIndicator'
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet'
import { useTask, useAddTaskComment, useAppendTaskMessage, useCreatePlanningTask } from '@/hooks/useTasks'
import { ScrollArea } from '@/components/ui/scroll-area'
import { ApiRequestError } from '@/api/client'
import { useMemo, useState } from 'react'
import type { TaskMessage, TaskDetail, UIResponse } from '@/types'

type TaskWorkspaceProps = {
  onClose: () => void
  onTaskCreated?: (taskId: string) => void
} & (
  | { taskId: string; projectId?: never }
  | { projectId: string; taskId?: never }
)

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

function DraftPlanningState({ onPlanningSubmit, disabled }: { onPlanningSubmit: (content: string) => Promise<void>, disabled: boolean }) {
  return (
    <>
      <div className="px-5 py-4 shrink-0 border-b">
        <h2 className="text-lg font-semibold">新需求</h2>
        <p className="mt-2 text-sm text-muted-foreground">
          描述你想要实现的需求，PM 会先帮你澄清并拆解任务。
        </p>
      </div>

      <div className="flex-1 min-h-0 px-5 py-6">
        <div className="rounded-2xl border border-dashed bg-muted/20 px-5 py-6">
          <p className="text-sm font-medium">从这里开始规划</p>
          <p className="mt-2 text-sm text-muted-foreground">
            先输入第一条需求描述。创建后，这个工作区会自动切换到 planning 模式，继续和 PM 在同一处完成澄清。
          </p>
        </div>
      </div>

      <div className="border-t px-4 py-3 shrink-0">
        <TaskComposer
          mode="planning"
          disabled={disabled}
          planningPlaceholder="描述你想要实现的需求..."
          onPlanningSubmit={(content) => onPlanningSubmit(content)}
        />
      </div>
    </>
  )
}

export function TaskWorkspace(props: TaskWorkspaceProps) {
  const taskId = 'taskId' in props ? props.taskId : undefined
  const projectId = 'projectId' in props ? props.projectId : undefined
  const { data: task } = useTask(taskId)
  const [chatOpen, setChatOpen] = useState(false)
  const [resultOpen, setResultOpen] = useState(false)
  const [cancelDialogOpen, setCancelDialogOpen] = useState(false)
  const addComment = useAddTaskComment()
  const appendTaskMessage = useAppendTaskMessage()
  const createPlanningTask = useCreatePlanningTask()
  const canCancelTask = task?.status === 'planning' || task?.status === 'pending' || task?.status === 'in_progress'
  const mentionCandidates = buildTaskMentionCandidates(task)
  const isPlanning = task?.status === 'planning'
  const hasTaskThread = (task?.messages?.length ?? 0) > 0
  const mode: 'planning' | 'building' = task?.status === 'planning' || !task ? 'planning' : 'building'

  const pendingUIBlocks = useMemo(() => {
    if (!isPlanning || !task?.messages?.length) {
      return null
    }
    const lastMessage = task.messages[task.messages.length - 1]
    if (lastMessage.role === 'pm_agent' && lastMessage.ui_blocks && lastMessage.ui_blocks.length > 0) {
      return lastMessage.ui_blocks
    }
    return null
  }, [isPlanning, task?.messages])

  const findNextUserResponse = (messages: TaskMessage[], index: number): TaskMessage | undefined => {
    if (index + 1 < messages.length && messages[index + 1].role === 'user') {
      return messages[index + 1]
    }
    return undefined
  }

  const handleSubmitComment = async ({ content, mentionAgentIds }: TaskCommentSubmitInput) => {
    if (!taskId) {
      return false
    }
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

  const handleSendPlanningMessage = async (content: string, uiResponse?: UIResponse) => {
    if (!taskId) {
      return
    }
    try {
      await appendTaskMessage.mutateAsync({
        taskId,
        input: { content, ui_response: uiResponse },
      })
    } catch (error) {
      const message = error instanceof ApiRequestError ? error.message : '发送需求消息失败'
      toast.error(message)
    }
  }

  const handleCreatePlanningTask = async (content: string) => {
    if (!projectId) {
      return
    }

    try {
      const res = await createPlanningTask.mutateAsync({
        projectId,
        input: { content },
      })
      props.onTaskCreated?.(res.data.id)
    } catch (error) {
      const message = error instanceof ApiRequestError ? error.message : '创建规划任务失败'
      toast.error(message)
    }
  }

  if (!taskId && projectId) {
    return (
      <div className="flex flex-col h-full">
        <div className="flex items-center justify-between px-5 py-3 border-b shrink-0">
          <div className="flex items-center gap-2">
            <span className="inline-flex items-center rounded-full bg-info/10 px-2.5 py-1 text-xs font-medium text-info">
              Planning
            </span>
          </div>
          <Button variant="ghost" size="icon" className="size-7" onClick={props.onClose}>
            <X className="size-4" />
          </Button>
        </div>
        <DraftPlanningState
          onPlanningSubmit={handleCreatePlanningTask}
          disabled={createPlanningTask.isPending}
        />
      </div>
    )
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
          <Button variant="ghost" size="icon" className="size-7" onClick={() => setResultOpen(true)} title="查看交付成果">
            <PackageCheck className="size-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="size-7"
            onClick={() => setChatOpen(true)}
            title="查看需求对话"
            disabled={!hasTaskThread}
          >
            <MessageSquare className="size-4" />
          </Button>
          <Button variant="ghost" size="icon" className="size-7" onClick={props.onClose}>
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

      {/* Feed */}
      <div className="flex-1 min-h-0">
        {isPlanning ? (
          <ScrollArea className="h-full px-5 py-4">
            <div className="flex flex-col gap-4">
              {(task.messages ?? []).map((message, index, messages) => {
                const isLastMessage = index === messages.length - 1
                const hasPendingBlocks = isLastMessage && !!pendingUIBlocks
                return (
                  <MessageBubble
                    key={message.id}
                    message={message}
                    nextUserResponse={
                      message.role === 'pm_agent' && message.ui_blocks?.length
                        ? findNextUserResponse(messages, index)
                        : undefined
                    }
                    hideUIBlocks={hasPendingBlocks}
                  />
                )
              })}
              {task.messages && task.messages[task.messages.length - 1]?.role === 'user' && <ThinkingIndicator />}
            </div>
          </ScrollArea>
        ) : (
          <TaskFeed taskId={task.id} />
        )}
      </div>

      {/* Comment input */}
      <div className="border-t px-4 py-3 shrink-0">
        <TaskComposer
          mode={mode}
          disabled={mode === 'planning' ? appendTaskMessage.isPending : addComment.isPending}
          pendingUIBlocks={pendingUIBlocks}
          buildingCandidates={mentionCandidates}
          onPlanningSubmit={handleSendPlanningMessage}
          onBuildingSubmit={handleSubmitComment}
        />
      </div>

      <Sheet open={resultOpen} onOpenChange={setResultOpen}>
        <SheetContent className="w-full max-w-2xl! p-0">
          <SheetHeader className="px-4 py-3 border-b">
            <SheetTitle>交付成果</SheetTitle>
          </SheetHeader>
          <ScrollArea className="flex-1 min-h-0">
            <div className="p-4">
              <TaskResultView taskId={task.id} result={task.result} artifacts={task.artifacts} />
            </div>
          </ScrollArea>
        </SheetContent>
      </Sheet>
      {hasTaskThread ? (
        <TaskThreadSheet
          taskId={task.id}
          open={chatOpen}
          onOpenChange={setChatOpen}
        />
      ) : null}
      <CancelTaskDialog
        open={cancelDialogOpen}
        onOpenChange={setCancelDialogOpen}
        task={task}
      />
    </div>
  )
}
