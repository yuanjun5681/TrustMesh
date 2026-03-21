import { Bot, UserCircle } from 'lucide-react'
import { cn } from '@/lib/utils'
import { formatDateTime } from '@/lib/utils'
import { useTaskComments } from '@/hooks/useTasks'
import type { Comment } from '@/types'

interface TaskCommentsProps {
  taskId: string
}

export function TaskComments({ taskId }: TaskCommentsProps) {
  const { data: comments, isLoading } = useTaskComments(taskId)

  if (isLoading) {
    return <div className="py-8 text-center text-sm text-muted-foreground">加载中...</div>
  }

  if (!comments || comments.length === 0) {
    return <div className="py-8 text-center text-sm text-muted-foreground">暂无讨论</div>
  }

  return (
    <div className="flex flex-col gap-3">
      {comments.map((comment) => (
        <CommentItem key={comment.id} comment={comment} />
      ))}
    </div>
  )
}

function CommentItem({ comment }: { comment: Comment }) {
  const isAgent = comment.actor_type === 'agent'
  const ActorIcon = isAgent ? Bot : UserCircle

  return (
    <div className="flex gap-2.5">
      <div
        className={cn(
          'flex size-7 shrink-0 items-center justify-center rounded-full',
          isAgent ? 'bg-muted' : 'bg-primary/10',
        )}
      >
        <ActorIcon className="size-3.5 text-muted-foreground" />
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-1.5">
          <span className="text-sm font-medium">{comment.actor_name}</span>
          <span className="text-xs text-muted-foreground">{formatDateTime(comment.created_at)}</span>
        </div>
        <div
          className={cn(
            'mt-1 rounded-md px-3 py-2 text-sm whitespace-pre-wrap',
            isAgent ? 'bg-muted' : 'bg-primary/5',
          )}
        >
          {comment.content}
        </div>
      </div>
    </div>
  )
}
