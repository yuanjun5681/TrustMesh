import { ChevronDown, ChevronRight } from 'lucide-react'
import { useState } from 'react'
import { cn } from '@/lib/utils'
import { normalizeEscapedText } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import type { Todo, TaskArtifact } from '@/types'

const statusIndicatorColors = {
  planning: 'bg-cyan-500',
  review: 'bg-amber-500',
  pending: 'bg-slate-400',
  in_progress: 'bg-sky-500 animate-pulse',
  done: 'bg-emerald-500',
  failed: 'bg-rose-500',
  canceled: 'bg-slate-500/60',
  interrupted: 'bg-amber-500',
}

interface TodoListProps {
  todos: Todo[]
  artifacts: TaskArtifact[]
  variant?: 'card' | 'nested'
}

export function TodoList({ todos, artifacts, variant = 'card' }: TodoListProps) {
  const safeArtifacts = artifacts ?? []

  if (todos.length === 0) {
    return <p className="py-8 text-center text-sm text-muted-foreground">暂无 Todo</p>
  }

  return (
    <div className={cn('flex flex-col', variant === 'card' ? 'gap-1' : 'gap-0')}>
      {todos.map((todo) => (
        <TodoItem
          key={todo.id}
          todo={todo}
          artifacts={safeArtifacts}
          variant={variant}
        />
      ))}
    </div>
  )
}

function TodoItem({
  todo,
  artifacts,
  variant,
}: {
  todo: Todo
  artifacts: TaskArtifact[]
  variant: 'card' | 'nested'
}) {
  const [expanded, setExpanded] = useState(false)
  const relatedArtifacts = (artifacts ?? []).filter((a) => a.todo_id === todo.id)
  const hasDetails = todo.description || todo.error || todo.interrupt_reason || relatedArtifacts.length > 0
  const isCard = variant === 'card'

  return (
    <div
      className={cn(
        isCard
          ? 'rounded-lg border bg-card'
          : 'border-t border-border/60 first:border-t-0'
      )}
    >
      <div className={cn('flex items-start gap-3', isCard ? 'p-3' : 'py-3 pr-1')}>
        <button
          type="button"
          onClick={() => hasDetails && setExpanded(!expanded)}
          className={cn(
            'flex min-w-0 flex-1 items-start gap-3 text-left',
            hasDetails && 'cursor-pointer rounded-md transition-colors hover:bg-muted/40',
          )}
        >
          <span
            aria-hidden="true"
            className={cn(
              'mt-1.5 size-2 shrink-0 rounded-full',
              statusIndicatorColors[todo.status],
            )}
          />
          <div className="flex min-w-0 flex-1 items-start gap-3">
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <span className={cn('truncate font-medium', isCard ? 'text-sm' : 'text-[13px]')}>
                  {todo.title}
                </span>
              </div>
            </div>
            <span className="mt-0.5 shrink-0 text-xs text-muted-foreground">
              {todo.assignee.name}
            </span>
          </div>
          {hasDetails && (
            expanded
              ? <ChevronDown className="size-4 shrink-0 text-muted-foreground mt-0.5" />
              : <ChevronRight className="size-4 shrink-0 text-muted-foreground mt-0.5" />
          )}
        </button>
      </div>

      {expanded && hasDetails && (
        <div
          className={cn(
            'ml-5 flex flex-col gap-2 text-sm',
            isCard ? 'px-3 pb-3 pt-0' : 'mb-3 border-l border-border/70 pl-3',
          )}
        >
          {todo.description && (
            <p className="text-muted-foreground whitespace-pre-wrap">{normalizeEscapedText(todo.description)}</p>
          )}
          {todo.error && (
            <p className="rounded-md bg-destructive/10 p-2 text-xs text-destructive whitespace-pre-wrap">
              {normalizeEscapedText(todo.error)}
            </p>
          )}
          {todo.interrupt_reason && (
            <p className="rounded-md bg-amber-500/10 p-2 text-xs text-amber-700 dark:text-amber-400 whitespace-pre-wrap">
              中断原因：{normalizeEscapedText(todo.interrupt_reason)}
            </p>
          )}
          {relatedArtifacts.length > 0 && (
            <div className="flex flex-wrap gap-1">
              {relatedArtifacts.map((a) => (
                <Badge key={a.transfer_id} variant={isCard ? 'secondary' : 'outline'} className="text-xs">
                  {a.file_name}
                </Badge>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
