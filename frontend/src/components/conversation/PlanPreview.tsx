import { Link } from 'react-router-dom'
import { ExternalLink } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { TaskStatusBadge, PriorityBadge } from '@/components/shared/StatusBadge'
import type { TaskSummary } from '@/types'

interface PlanPreviewProps {
  task: TaskSummary
  projectId: string
}

export function PlanPreview({ task, projectId }: PlanPreviewProps) {
  const progress = task.todo_count > 0
    ? Math.round((task.completed_todo_count / task.todo_count) * 100)
    : 0

  return (
    <Card className="border-primary/20 bg-primary/5">
      <CardContent className="p-3 space-y-2">
        <div className="flex items-center justify-between gap-2">
          <div className="flex items-center gap-2">
            <TaskStatusBadge status={task.status} />
            <PriorityBadge priority={task.priority} />
          </div>
          <Link
            to={`/projects/${projectId}`}
            className="text-xs text-primary hover:underline flex items-center gap-1"
          >
            查看看板 <ExternalLink className="h-3 w-3" />
          </Link>
        </div>
        <p className="text-sm font-medium">{task.title}</p>
        {task.todo_count > 0 && (
          <div className="space-y-1">
            <div className="h-1.5 rounded-full bg-background overflow-hidden">
              <div
                className="h-full rounded-full bg-primary transition-all"
                style={{ width: `${progress}%` }}
              />
            </div>
            <p className="text-xs text-muted-foreground">
              {task.completed_todo_count}/{task.todo_count} Todo 完成
            </p>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
