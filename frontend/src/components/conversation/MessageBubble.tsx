import { cn } from '@/lib/utils'
import type { ConversationMessage } from '@/types'
import { formatRelativeTime } from '@/lib/utils'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

interface MessageBubbleProps {
  message: ConversationMessage
}

export function MessageBubble({ message }: MessageBubbleProps) {
  const isUser = message.role === 'user'

  return (
    <div className={cn('flex', isUser && 'justify-end')}>
      <div className={cn('flex flex-col gap-1 max-w-[85%]', isUser && 'items-end')}>
        <div
          className={cn(
            'rounded-3xl px-5 py-3.5 text-sm leading-relaxed overflow-hidden',
            isUser
              ? 'bg-primary text-primary-foreground'
              : 'bg-muted'
          )}
        >
          {isUser ? (
            <p className="whitespace-pre-wrap wrap-break-word">{message.content}</p>
          ) : (
            <div className="prose prose-sm dark:prose-invert max-w-none prose-p:my-1 prose-ul:my-1 prose-ol:my-1 prose-li:my-0.5 prose-headings:my-2">
              <ReactMarkdown remarkPlugins={[remarkGfm]}>
                {message.content}
              </ReactMarkdown>
            </div>
          )}
        </div>
        <p className={cn('text-[10px] text-muted-foreground px-1', isUser && 'text-right')}>
          {formatRelativeTime(message.created_at)}
        </p>
      </div>
    </div>
  )
}
