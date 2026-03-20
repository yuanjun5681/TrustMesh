import { X, MessageSquare } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { TaskStatusBadge, PriorityBadge } from '@/components/shared/StatusBadge'
import { TodoList } from './TodoList'
import { TaskTimeline } from './TaskTimeline'
import { TaskResultView } from './TaskResult'
import { ConversationSheet } from '@/components/conversation/ConversationSheet'
import { useTask } from '@/hooks/useTasks'
import { useTaskStream } from '@/hooks/useLiveStreams'
import { ScrollArea } from '@/components/ui/scroll-area'
import { useState } from 'react'

interface TaskDetailPanelProps {
  taskId: string
  onClose: () => void
}

export function TaskDetailPanel({ taskId, onClose }: TaskDetailPanelProps) {
  const { data: task } = useTask(taskId)
  const shouldStream = !task || task.status === 'pending' || task.status === 'in_progress'
  useTaskStream(taskId, shouldStream)
  const [tab, setTab] = useState('todos')
  const [chatOpen, setChatOpen] = useState(false)

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
          </div>

          <Tabs value={tab} onValueChange={setTab}>
            <TabsList>
              <TabsTrigger value="todos">
                Todo ({task.todos.length})
              </TabsTrigger>
              <TabsTrigger value="timeline">时间线</TabsTrigger>
              <TabsTrigger value="result">结果</TabsTrigger>
            </TabsList>

            <TabsContent value="todos" className="mt-3">
              <TodoList taskId={task.id} todos={task.todos} artifacts={task.artifacts} />
            </TabsContent>

            <TabsContent value="timeline" className="mt-3">
              <TaskTimeline taskId={task.id} />
            </TabsContent>

            <TabsContent value="result" className="mt-3">
              <TaskResultView taskId={task.id} result={task.result} artifacts={task.artifacts} />
            </TabsContent>
          </Tabs>
        </div>
      </ScrollArea>

      <ConversationSheet
        projectId={task.project_id}
        initialConversationId={task.conversation_id}
        open={chatOpen}
        onOpenChange={setChatOpen}
      />
    </div>
  )
}
