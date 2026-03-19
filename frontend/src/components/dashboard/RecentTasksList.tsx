import { Link } from 'react-router-dom'
import { TaskStatusBadge } from '@/components/shared/StatusBadge'
import { formatRelativeTime } from '@/lib/utils'
import type { TaskListItem } from '@/types'

interface RecentTasksListProps {
  tasks: TaskListItem[]
  loading?: boolean
}

export function RecentTasksList({ tasks, loading }: RecentTasksListProps) {
  if (loading) {
    return <div className="py-8 text-center text-sm text-muted-foreground">加载中...</div>
  }

  if (tasks.length === 0) {
    return <div className="py-8 text-center text-sm text-muted-foreground">暂无任务</div>
  }

  return (
    <div className="space-y-2">
      {tasks.map((task) => (
        <Link
          key={task.id}
          to={`/projects/${task.project_id}`}
          className="flex items-start gap-3 rounded-lg p-2 hover:bg-accent/50 transition-colors"
        >
          <TaskStatusBadge status={task.status} />
          <div className="flex-1 min-w-0">
            <div className="text-sm font-medium truncate">{task.title}</div>
            <div className="text-xs text-muted-foreground mt-0.5">
              {task.pm_agent.name} · {formatRelativeTime(task.updated_at)}
              {task.todo_count > 0 && (
                <span className="ml-1">
                  · {task.completed_todo_count}/{task.todo_count} Todo
                </span>
              )}
            </div>
          </div>
        </Link>
      ))}
    </div>
  )
}
