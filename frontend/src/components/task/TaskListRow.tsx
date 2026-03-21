import { cn } from '@/lib/utils'
import { formatRelativeTime } from '@/lib/utils'
import { PriorityBadge } from '@/components/shared/StatusBadge'
import { Circle, CircleDot, CheckCircle2, XCircle } from 'lucide-react'
import type { TaskListItem, TaskStatus } from '@/types'

const statusIcon: Record<TaskStatus, { icon: typeof Circle; className: string }> = {
  pending: { icon: Circle, className: 'text-muted-foreground' },
  in_progress: { icon: CircleDot, className: 'text-info animate-pulse' },
  done: { icon: CheckCircle2, className: 'text-success' },
  failed: { icon: XCircle, className: 'text-destructive' },
}

interface TaskListRowProps {
  task: TaskListItem
  isSelected: boolean
  onClick: () => void
}

export function TaskListRow({ task, isSelected, onClick }: TaskListRowProps) {
  const progress = task.todo_count > 0
    ? Math.round((task.completed_todo_count / task.todo_count) * 100)
    : 0
  const Icon = statusIcon[task.status].icon
  const iconClass = statusIcon[task.status].className

  return (
    <div
      className={cn(
        'group grid grid-cols-[1fr_80px_140px_80px] items-center gap-3 px-3 py-2.5 cursor-pointer rounded-md transition-colors',
        'hover:bg-muted/50',
        isSelected && 'bg-muted',
      )}
      onClick={onClick}
    >
      {/* Name */}
      <div className="flex items-center gap-2.5 min-w-0">
        <Icon className={cn('size-4 shrink-0', iconClass)} />
        <div className="min-w-0">
          <div className="text-sm font-medium truncate">{task.title}</div>
          {task.description && (
            <div className="text-xs text-muted-foreground truncate mt-0.5">{task.description}</div>
          )}
        </div>
      </div>

      {/* Priority */}
      <div className="flex justify-center">
        <PriorityBadge priority={task.priority} />
      </div>

      {/* Progress */}
      <div className="flex items-center gap-2">
        {task.todo_count > 0 ? (
          <>
            <div className="flex-1 h-1.5 rounded-full bg-muted overflow-hidden">
              <div
                className={cn(
                  'h-full rounded-full transition-all duration-300',
                  task.failed_todo_count > 0 ? 'bg-destructive' : 'bg-primary',
                )}
                style={{ width: `${progress}%` }}
              />
            </div>
            <span className="text-xs text-muted-foreground whitespace-nowrap">
              {task.completed_todo_count}/{task.todo_count}
            </span>
            {task.failed_todo_count > 0 && (
              <span className="text-xs text-destructive whitespace-nowrap">
                {task.failed_todo_count}!
              </span>
            )}
          </>
        ) : (
          <span className="text-xs text-muted-foreground">—</span>
        )}
      </div>

      {/* Time */}
      <div className="text-xs text-muted-foreground text-right whitespace-nowrap">
        {formatRelativeTime(task.updated_at)}
      </div>
    </div>
  )
}
