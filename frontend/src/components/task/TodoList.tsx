import { CheckCircle2, Circle, Loader2, XCircle, ChevronDown, ChevronRight, CircleSlash2 } from 'lucide-react'
import { useState } from 'react'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import type { Todo, TaskArtifact } from '@/types'

const statusIcons = {
  pending: Circle,
  in_progress: Loader2,
  done: CheckCircle2,
  failed: XCircle,
  canceled: CircleSlash2,
}

const statusColors = {
  pending: 'text-muted-foreground',
  in_progress: 'text-info animate-spin',
  done: 'text-success',
  failed: 'text-destructive',
  canceled: 'text-muted-foreground',
}

interface TodoListProps {
  todos: Todo[]
  artifacts: TaskArtifact[]
}

export function TodoList({ todos, artifacts }: TodoListProps) {
  const safeArtifacts = artifacts ?? []

  if (todos.length === 0) {
    return <p className="py-8 text-center text-sm text-muted-foreground">暂无 Todo</p>
  }

  return (
    <div className="flex flex-col gap-1">
      {todos.map((todo) => (
        <TodoItem
          key={todo.id}
          todo={todo}
          artifacts={safeArtifacts}
        />
      ))}
    </div>
  )
}

function TodoItem({
  todo,
  artifacts,
}: {
  todo: Todo
  artifacts: TaskArtifact[]
}) {
  const [expanded, setExpanded] = useState(false)
  const Icon = statusIcons[todo.status]
  const relatedArtifacts = (artifacts ?? []).filter((a) => a.source_todo_id === todo.id)
  const hasDetails = todo.description || todo.result?.summary || todo.error || relatedArtifacts.length > 0

  return (
    <div className="rounded-lg border bg-card">
      <div className="flex items-start gap-3 p-3">
        <button
          type="button"
          onClick={() => hasDetails && setExpanded(!expanded)}
          className={cn(
            'flex min-w-0 flex-1 items-start gap-3 text-left',
            hasDetails && 'cursor-pointer rounded-md hover:bg-muted/50 transition-colors'
          )}
        >
          <Icon className={cn('size-4 mt-0.5 shrink-0', statusColors[todo.status])} />
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium">{todo.title}</span>
            </div>
            <div className="flex items-center gap-2 mt-0.5 text-xs text-muted-foreground">
              <span>{todo.assignee.name}</span>
            </div>
          </div>
          {hasDetails && (
            expanded
              ? <ChevronDown className="size-4 shrink-0 text-muted-foreground mt-0.5" />
              : <ChevronRight className="size-4 shrink-0 text-muted-foreground mt-0.5" />
          )}
        </button>
      </div>

      {expanded && hasDetails && (
        <div className="flex flex-col gap-2 px-3 pb-3 pt-0 ml-7 text-sm">
          {todo.description && (
            <p className="text-muted-foreground whitespace-pre-wrap">{todo.description}</p>
          )}
          {todo.error && (
            <p className="text-destructive bg-destructive/10 rounded-md p-2 text-xs">
              {todo.error}
            </p>
          )}
          {todo.result?.summary && (
            <div className="bg-muted/50 rounded-md p-2">
              <p className="font-medium text-xs mb-1">结果</p>
              <p className="text-muted-foreground text-xs">{todo.result.summary}</p>
            </div>
          )}
          {relatedArtifacts.length > 0 && (
            <div className="flex flex-wrap gap-1">
              {relatedArtifacts.map((a) => (
                <Badge key={a.id} variant="secondary" className="text-xs">
                  {a.title}
                </Badge>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
