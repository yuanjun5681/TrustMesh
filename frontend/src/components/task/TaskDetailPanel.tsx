import { X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { TaskStatusBadge, PriorityBadge } from '@/components/shared/StatusBadge'
import { TodoList } from './TodoList'
import { TaskTimeline } from './TaskTimeline'
import { TaskResultView } from './TaskResult'
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
        <Button variant="ghost" size="icon" className="h-7 w-7" onClick={onClose}>
          <X className="h-4 w-4" />
        </Button>
      </div>

      {/* Content */}
      <ScrollArea className="flex-1">
        <div className="px-5 py-4 space-y-4">
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
    </div>
  )
}
