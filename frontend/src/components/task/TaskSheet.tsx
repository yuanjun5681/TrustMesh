import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { TaskStatusBadge, PriorityBadge } from '@/components/shared/StatusBadge'
import { TaskFeed } from './TaskFeed'
import { TaskResultView } from './TaskResult'
import { TaskDescription } from './TaskDescription'
import { TaskTodoSection } from './TaskTodoSection'
import { useTask } from '@/hooks/useTasks'
import { Separator } from '@/components/ui/separator'
import { ScrollArea } from '@/components/ui/scroll-area'
import { useState } from 'react'
import type { TaskDetail } from '@/types'

interface TaskSheetProps {
  taskId: string | null
  onClose: () => void
}

export function TaskSheet({ taskId, onClose }: TaskSheetProps) {
  const { data: task } = useTask(taskId ?? undefined)

  return (
    <Sheet open={!!taskId} onOpenChange={() => onClose()}>
      <SheetContent className="max-w-2xl">
        {task && (
          <TaskSheetBody key={task.id} task={task} />
        )}
      </SheetContent>
    </Sheet>
  )
}

function TaskSheetBody({ task }: { task: TaskDetail }) {
  const [tab, setTab] = useState('feed')

  return (
    <>
      <SheetHeader>
        <div className="flex items-center gap-2 flex-wrap pr-8">
          <TaskStatusBadge status={task.status} />
          <PriorityBadge priority={task.priority} />
          {task.external_ref?.platform === 'clawhire' && (
            <span className="inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-xs font-medium bg-amber-500/10 text-amber-600 border border-amber-500/20">
              <span className="font-bold">CH</span> ClawHire 任务
            </span>
          )}
        </div>
        <SheetTitle className="text-lg mt-1">{task.title}</SheetTitle>
        {task.description && (
          <TaskDescription description={task.description} />
        )}
      </SheetHeader>

      <Separator className="my-4" />

      <div className="flex flex-1 flex-col gap-4 px-6 pb-6">
        <TaskTodoSection todos={task.todos} artifacts={task.artifacts} />

        <Tabs value={tab} onValueChange={setTab}>
          <TabsList>
            <TabsTrigger value="feed">动态</TabsTrigger>
            <TabsTrigger value="result">结果</TabsTrigger>
          </TabsList>

          <TabsContent value="feed">
            <div className="h-[calc(100vh-280px)]">
              <TaskFeed taskId={task.id} />
            </div>
          </TabsContent>

          <TabsContent value="result">
            <ScrollArea className="max-h-[calc(100vh-280px)]">
              <TaskResultView taskId={task.id} result={task.result} artifacts={task.artifacts} />
            </ScrollArea>
          </TabsContent>
        </Tabs>
      </div>
    </>
  )
}
