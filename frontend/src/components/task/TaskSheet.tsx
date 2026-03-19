import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { TaskStatusBadge, PriorityBadge } from '@/components/shared/StatusBadge'
import { TodoList } from './TodoList'
import { TaskTimeline } from './TaskTimeline'
import { TaskResultView } from './TaskResult'
import { useTask } from '@/hooks/useTasks'
import { useTaskStream } from '@/hooks/useLiveStreams'
import { Separator } from '@/components/ui/separator'
import { ScrollArea } from '@/components/ui/scroll-area'
import { useState } from 'react'

interface TaskSheetProps {
  taskId: string | null
  onClose: () => void
}

export function TaskSheet({ taskId, onClose }: TaskSheetProps) {
  const { data: task } = useTask(taskId ?? undefined)
  const shouldStream = !!taskId && (!task || task.status === 'pending' || task.status === 'in_progress')
  useTaskStream(taskId ?? undefined, shouldStream)
  const [tab, setTab] = useState('todos')

  return (
    <Sheet open={!!taskId} onOpenChange={() => onClose()}>
      <SheetContent className="max-w-2xl">
        {task && (
          <>
            <SheetHeader>
              <div className="flex items-center gap-2 flex-wrap pr-8">
                <TaskStatusBadge status={task.status} />
                <PriorityBadge priority={task.priority} />
              </div>
              <SheetTitle className="text-lg mt-1">{task.title}</SheetTitle>
              {task.description && (
                <p className="text-sm text-muted-foreground mt-1 whitespace-pre-wrap">
                  {task.description}
                </p>
              )}
            </SheetHeader>

            <Separator className="my-4" />

            <div className="px-6 pb-6 flex-1">
              <Tabs value={tab} onValueChange={setTab}>
                <TabsList>
                  <TabsTrigger value="todos">
                    Todo ({task.todos.length})
                  </TabsTrigger>
                  <TabsTrigger value="timeline">时间线</TabsTrigger>
                  <TabsTrigger value="result">结果</TabsTrigger>
                </TabsList>

                <TabsContent value="todos">
                  <ScrollArea className="max-h-[calc(100vh-280px)]">
                    <TodoList taskId={task.id} todos={task.todos} artifacts={task.artifacts} />
                  </ScrollArea>
                </TabsContent>

                <TabsContent value="timeline">
                  <ScrollArea className="max-h-[calc(100vh-280px)]">
                    <TaskTimeline taskId={task.id} />
                  </ScrollArea>
                </TabsContent>

                <TabsContent value="result">
                  <ScrollArea className="max-h-[calc(100vh-280px)]">
                    <TaskResultView taskId={task.id} result={task.result} artifacts={task.artifacts} />
                  </ScrollArea>
                </TabsContent>
              </Tabs>
            </div>
          </>
        )}
      </SheetContent>
    </Sheet>
  )
}
