import { X, MessageSquare, PackageCheck } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { TaskStatusBadge, PriorityBadge } from '@/components/shared/StatusBadge'
import { TaskFeed } from './TaskFeed'
import { TaskThreadSheet } from './TaskThreadSheet'
import { TaskResultView } from './TaskResult'
import { TaskDescription } from './TaskDescription'
import { TaskComposer } from './TaskComposer'
import { TaskCommentComposer, type TaskCommentSubmitInput, type TaskMentionCandidate } from './TaskCommentComposer'
import { CancelTaskDialog } from './CancelTaskDialog'
import { MessageBubble } from '@/components/task-thread/MessageBubble'
import { ThinkingIndicator } from '@/components/task-thread/ThinkingIndicator'
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet'
import { useTask, useAddTaskComment, useAppendTaskMessage, useCreateTaskFromText, useApprovePlan, useRejectPlan } from '@/hooks/useTasks'
import { useAgents } from '@/hooks/useAgents'
import { ScrollArea } from '@/components/ui/scroll-area'
import { ApiRequestError } from '@/api/client'
import { useMemo, useState } from 'react'
import type { TaskMessage, TaskDetail, UIResponse, Todo } from '@/types'

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

function PlanReviewPanel({
  todos,
  onApprove,
  onReject,
  isApproving,
  isRejecting,
}: {
  todos: Todo[]
  onApprove: () => void
  onReject: (feedback: string) => void
  isApproving: boolean
  isRejecting: boolean
}) {
  const [showRejectInput, setShowRejectInput] = useState(false)
  const [feedback, setFeedback] = useState('')

  const handleReject = () => {
    if (!feedback.trim()) return
    onReject(feedback.trim())
  }

  return (
    <div className="border rounded-xl bg-muted/20 p-4 flex flex-col gap-3">
      <div>
        <p className="text-sm font-medium">PM 已完成规划，请确认后开始执行</p>
        <p className="text-xs text-muted-foreground mt-1">共 {todos.length} 个子任务</p>
      </div>

      <div className="flex flex-col gap-1.5">
        {todos.map((todo, idx) => (
          <div key={todo.id} className="flex items-start gap-2 rounded-lg bg-background border px-3 py-2 text-sm">
            <span className="shrink-0 text-xs text-muted-foreground w-5 pt-0.5">{idx + 1}.</span>
            <div className="min-w-0">
              <p className="font-medium truncate">{todo.title}</p>
              <p className="text-xs text-muted-foreground truncate">
                {todo.assignee.name}
              </p>
            </div>
          </div>
        ))}
      </div>

      {showRejectInput ? (
        <div className="flex flex-col gap-2">
          <textarea
            className="w-full rounded-lg border bg-background px-3 py-2 text-sm resize-none focus:outline-none focus:ring-1 focus:ring-ring"
            rows={3}
            placeholder="说明需要调整的地方..."
            value={feedback}
            onChange={(e) => setFeedback(e.target.value)}
            autoFocus
          />
          <div className="flex gap-2">
            <Button
              size="sm"
              variant="destructive"
              disabled={!feedback.trim() || isRejecting}
              onClick={handleReject}
            >
              {isRejecting ? '提交中...' : '提交修改意见'}
            </Button>
            <Button
              size="sm"
              variant="ghost"
              onClick={() => { setShowRejectInput(false); setFeedback('') }}
            >
              取消
            </Button>
          </div>
        </div>
      ) : (
        <div className="flex gap-2">
          <Button size="sm" disabled={isApproving} onClick={onApprove}>
            {isApproving ? '确认中...' : '确认执行'}
          </Button>
          <Button size="sm" variant="outline" onClick={() => setShowRejectInput(true)}>
            修改规划
          </Button>
        </div>
      )}
    </div>
  )
}

function DraftPlanningState({
  onSubmit,
  disabled,
  candidates,
}: {
  onSubmit: (content: string, agentId?: string) => Promise<void>
  disabled: boolean
  candidates: TaskMentionCandidate[]
}) {
  const handleSubmit = async ({ content, mentionAgentIds }: TaskCommentSubmitInput) => {
    await onSubmit(content, mentionAgentIds[0])
    return true
  }

  return (
    <>
      <div className="px-5 py-4 shrink-0 border-b">
        <h2 className="text-lg font-semibold">新任务</h2>
        <p className="mt-2 text-sm text-muted-foreground">
          描述需求由 PM 规划；输入 @ 直接指派给执行 Agent。
        </p>
      </div>

      <div className="flex-1 min-h-0 px-5 py-6">
        <div className="rounded-2xl border border-dashed bg-muted/20 px-5 py-6">
          <p className="text-sm font-medium">从这里开始</p>
          <p className="mt-2 text-sm text-muted-foreground">
            提交后系统会自动判断：有指定执行 Agent 时直接创建并派发任务；否则进入 planning 模式由 PM 澄清需求。
          </p>
        </div>
      </div>

      <div className="border-t px-4 py-3 shrink-0">
        <TaskCommentComposer
          candidates={candidates}
          disabled={disabled}
          placeholder="描述需求，或 @ 执行 Agent 直接指派任务... (Enter 发送，Shift+Enter 换行)"
          onSubmit={handleSubmit}
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
  const createTaskFromText = useCreateTaskFromText()
  const approvePlan = useApprovePlan()
  const rejectPlan = useRejectPlan()
  const { data: allAgents } = useAgents()
  const canCancelTask = task?.status === 'planning' || task?.status === 'review' || task?.status === 'pending' || task?.status === 'in_progress'
  const mentionCandidates = buildTaskMentionCandidates(task)
  const isPlanning = task?.status === 'planning'
  const isReview = task?.status === 'review'
  const hasTaskThread = (task?.messages?.length ?? 0) > 0
  const mode: 'planning' | 'building' = task?.status === 'planning' || task?.status === 'review' || !task ? 'planning' : 'building'

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

  const handleApprovePlan = async () => {
    if (!taskId) return
    try {
      await approvePlan.mutateAsync({ taskId })
    } catch (error) {
      const message = error instanceof ApiRequestError ? error.message : '确认规划失败'
      toast.error(message)
    }
  }

  const handleRejectPlan = async (feedback: string) => {
    if (!taskId) return
    try {
      await rejectPlan.mutateAsync({ taskId, input: { feedback } })
    } catch (error) {
      const message = error instanceof ApiRequestError ? error.message : '提交修改意见失败'
      toast.error(message)
    }
  }

  const handleCreateTaskFromText = async (content: string, agentId?: string) => {
    if (!projectId) {
      return
    }
    try {
      const res = await createTaskFromText.mutateAsync({ projectId, content, agentId })
      props.onTaskCreated?.(res.data.id)
    } catch (error) {
      const message = error instanceof ApiRequestError ? error.message : '创建任务失败'
      toast.error(message)
    }
  }

  if (!taskId && projectId) {
    const executorCandidates: TaskMentionCandidate[] = (allAgents ?? [])
      .filter((a) => a.role !== 'pm')
      .map((a) => ({ id: a.id, name: a.name, roleLabel: '执行 Agent' }))

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
          onSubmit={handleCreateTaskFromText}
          disabled={createTaskFromText.isPending}
          candidates={executorCandidates}
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
        {isPlanning || isReview ? (
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
              {isPlanning && task.messages && task.messages[task.messages.length - 1]?.role === 'user' && <ThinkingIndicator />}
              {isReview && (
                <PlanReviewPanel
                  todos={task.todos}
                  onApprove={handleApprovePlan}
                  onReject={handleRejectPlan}
                  isApproving={approvePlan.isPending}
                  isRejecting={rejectPlan.isPending}
                />
              )}
            </div>
          </ScrollArea>
        ) : (
          <TaskFeed taskId={task.id} />
        )}
      </div>

      {/* Comment input */}
      <div className="border-t px-4 py-3 shrink-0">
        {isReview ? null : (
          <TaskComposer
            mode={mode}
            disabled={mode === 'planning' ? appendTaskMessage.isPending : addComment.isPending}
            pendingUIBlocks={pendingUIBlocks}
            buildingCandidates={mentionCandidates}
            onPlanningSubmit={handleSendPlanningMessage}
            onBuildingSubmit={handleSubmitComment}
          />
        )}
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
