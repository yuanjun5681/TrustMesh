import { X, MessageSquare, Send } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { TaskStatusBadge, PriorityBadge } from '@/components/shared/StatusBadge'
import { TaskTimeline } from './TaskTimeline'
import { TaskComments } from './TaskComments'
import { TaskResultView } from './TaskResult'
import { CancelTaskDialog } from './CancelTaskDialog'
import { ConversationSheet } from '@/components/conversation/ConversationSheet'
import { useTask, useAddTaskComment } from '@/hooks/useTasks'
import { ScrollArea } from '@/components/ui/scroll-area'
import { useState, useRef, useCallback } from 'react'

interface TaskDetailPanelProps {
  taskId: string
  onClose: () => void
}

export function TaskDetailPanel({ taskId, onClose }: TaskDetailPanelProps) {
  const { data: task } = useTask(taskId)
  const [tab, setTab] = useState('activity')
  const [chatOpen, setChatOpen] = useState(false)
  const [cancelDialogOpen, setCancelDialogOpen] = useState(false)
  const [comment, setComment] = useState('')
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const addComment = useAddTaskComment()
  const canCancelTask = task?.status === 'pending' || task?.status === 'in_progress'

  const handleSendComment = useCallback(() => {
    const text = comment.trim()
    if (!text || addComment.isPending) return
    addComment.mutate({ taskId, content: text }, {
      onSuccess: () => {
        setComment('')
        if (textareaRef.current) {
          textareaRef.current.style.height = 'auto'
        }
      },
    })
  }, [comment, taskId, addComment])

  const handleKeyDown = useCallback((e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSendComment()
    }
  }, [handleSendComment])

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

      {/* Content */}
      <ScrollArea className="flex-1">
        <div className="flex flex-col gap-4 px-5 py-4">
          <div>
            <h2 className="text-lg font-semibold">{task.title}</h2>
            {task.description && (
              <p className="text-sm text-muted-foreground mt-1.5 whitespace-pre-wrap">
                {task.description}
              </p>
            )}
            {task.cancel_reason && (
              <p className="mt-2 rounded-md bg-muted px-3 py-2 text-xs text-muted-foreground">
                终止原因：{task.cancel_reason}
              </p>
            )}
          </div>

          <Tabs value={tab} onValueChange={setTab}>
            <TabsList>
              <TabsTrigger value="activity">全部活动</TabsTrigger>
              <TabsTrigger value="comments">评论讨论</TabsTrigger>
              <TabsTrigger value="result">交付成果</TabsTrigger>
            </TabsList>

            <TabsContent value="activity" className="mt-3">
              <TaskTimeline taskId={task.id} />
            </TabsContent>

            <TabsContent value="comments" className="mt-3">
              <TaskComments taskId={task.id} />
            </TabsContent>

            <TabsContent value="result" className="mt-3">
              <TaskResultView taskId={task.id} result={task.result} artifacts={task.artifacts} />
            </TabsContent>
          </Tabs>
        </div>
      </ScrollArea>

      {/* Comment input */}
      <div className="border-t px-4 py-3 shrink-0">
        <div className="flex items-end gap-2">
          <textarea
            ref={textareaRef}
            className="flex-1 resize-none rounded-md border bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring min-h-[36px] max-h-[120px]"
            placeholder="输入评论... (Enter 发送, Shift+Enter 换行)"
            rows={1}
            value={comment}
            onChange={(e) => {
              setComment(e.target.value)
              e.target.style.height = 'auto'
              e.target.style.height = e.target.scrollHeight + 'px'
            }}
            onKeyDown={handleKeyDown}
          />
          <Button
            size="icon"
            className="size-9 shrink-0"
            disabled={!comment.trim() || addComment.isPending}
            onClick={handleSendComment}
          >
            <Send className="size-4" />
          </Button>
        </div>
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
