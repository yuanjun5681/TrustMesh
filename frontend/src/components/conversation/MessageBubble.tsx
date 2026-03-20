import { cn } from '@/lib/utils'
import { Avatar } from '@/components/ui/avatar'
import { Bot } from 'lucide-react'
import type { ConversationMessage } from '@/types'
import { formatRelativeTime } from '@/lib/utils'

interface MessageBubbleProps {
  message: ConversationMessage
}

export function MessageBubble({ message }: MessageBubbleProps) {
  const isUser = message.role === 'user'

  return (
    <div className={cn('flex gap-3', isUser && 'flex-row-reverse')}>
      {isUser ? (
        <Avatar fallback="我" size="sm" className="mt-0.5" />
      ) : (
        <div className="flex size-7 items-center justify-center rounded-full bg-primary/10 text-primary mt-0.5 shrink-0">
          <Bot className="size-4" />
        </div>
      )}
      <div className={cn('flex flex-col gap-1 max-w-[75%]', isUser && 'items-end')}>
        <div
          className={cn(
            'rounded-2xl px-4 py-2.5 text-sm leading-relaxed',
            isUser
              ? 'bg-primary text-primary-foreground rounded-tr-md'
              : 'bg-muted rounded-tl-md'
          )}
        >
          <p className="whitespace-pre-wrap">{message.content}</p>
        </div>
        <p className={cn('text-[10px] text-muted-foreground px-1', isUser && 'text-right')}>
          {formatRelativeTime(message.created_at)}
        </p>
      </div>
    </div>
  )
}
