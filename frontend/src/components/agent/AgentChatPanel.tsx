import { useEffect, useRef } from 'react'
import { MessageSquareText, RotateCcw } from 'lucide-react'
import { toast } from 'sonner'
import { Avatar } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { MessageInput } from '@/components/conversation/MessageInput'
import { useAgentChat, useResetAgentChat, useSendAgentChatMessage } from '@/hooks/useAgentChat'
import { formatDateTime, formatRelativeTime } from '@/lib/utils'
import { ApiRequestError } from '@/api/client'
import type { Agent } from '@/types'

interface AgentChatPanelProps {
  agent: Agent
}

const messageStatusText: Record<string, string> = {
  pending: '发送中',
  sent: '已发送',
  failed: '发送失败',
}

export function AgentChatPanel({ agent }: AgentChatPanelProps) {
  const { data: chat, isLoading } = useAgentChat(agent.id)
  const sendMessage = useSendAgentChatMessage()
  const resetChat = useResetAgentChat()
  const messagesEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [chat?.messages])

  const disabled = agent.archived || agent.status !== 'online'

  const handleSend = async (content: string) => {
    try {
      await sendMessage.mutateAsync({ agentId: agent.id, content })
    } catch (err) {
      toast.error(err instanceof ApiRequestError ? err.message : '消息发送失败')
    }
  }

  const handleReset = async () => {
    try {
      await resetChat.mutateAsync(agent.id)
      toast.success('已开始新会话')
    } catch (err) {
      toast.error(err instanceof ApiRequestError ? err.message : '新建会话失败')
    }
  }

  return (
    <div className="flex min-h-[calc(100vh-var(--agent-header-h)-10rem)] flex-col rounded-2xl border bg-card">
      <div className="flex items-center justify-between gap-3 border-b px-4 py-3">
        <div className="min-w-0">
          <div className="flex items-center gap-2 text-sm font-medium">
            <MessageSquareText className="size-4 text-muted-foreground" />
            一对一对话
          </div>
          <p className="mt-0.5 text-xs text-muted-foreground">
            通过 ClawSynapse 向远程 Agent 发送默认 `chat.message`
          </p>
        </div>
        <Button size="sm" variant="outline" onClick={handleReset} disabled={resetChat.isPending}>
          <RotateCcw className="mr-1.5 size-4" />
          新会话
        </Button>
      </div>

      <ScrollArea className="flex-1 p-4">
        {isLoading ? (
          <div className="text-sm text-muted-foreground">加载中...</div>
        ) : !chat || chat.messages.length === 0 ? (
          <div className="flex h-full min-h-64 flex-col items-center justify-center gap-3 text-center">
            <div className="flex size-12 items-center justify-center rounded-2xl bg-primary/10">
              <MessageSquareText className="size-6 text-primary" />
            </div>
            <div>
              <p className="text-sm font-medium">开始和 {agent.name} 对话</p>
              <p className="mt-1 text-sm text-muted-foreground">
                历史消息会保存在 TrustMesh，远端上下文由 Agent 自身维护。
              </p>
            </div>
          </div>
        ) : (
          <div className="space-y-4">
            {chat.messages.map((message) => {
              const isUser = message.sender_type === 'user'
              return (
                <div key={message.id} className={`flex gap-3 ${isUser ? 'justify-end' : 'justify-start'}`}>
                  {!isUser && (
                    <Avatar
                      fallback={agent.name}
                      seed={agent.id}
                      kind="agent"
                      role={agent.role}
                      size="sm"
                    />
                  )}
                  <div className={`max-w-[80%] ${isUser ? 'items-end' : 'items-start'} flex flex-col gap-1`}>
                    <div
                      className={[
                        'rounded-2xl px-4 py-3 text-sm leading-relaxed shadow-xs',
                        isUser ? 'bg-primary text-primary-foreground' : 'border bg-background',
                      ].join(' ')}
                    >
                      {message.content}
                    </div>
                    <div className="text-xs text-muted-foreground" title={formatDateTime(message.created_at)}>
                      {formatRelativeTime(message.created_at)}
                      {isUser ? ` · ${messageStatusText[message.status] ?? message.status}` : ''}
                    </div>
                  </div>
                  {isUser && <Avatar fallback="我" size="sm" />}
                </div>
              )
            })}
            <div ref={messagesEndRef} />
          </div>
        )}
      </ScrollArea>

      <div className="border-t px-4 py-3">
        {disabled && (
          <div className="mb-3 text-sm text-muted-foreground">
            {agent.archived ? '该 Agent 已离职，无法继续发送消息。' : 'Agent 离线，暂时无法发送消息。'}
          </div>
        )}
        <MessageInput
          onSend={handleSend}
          disabled={disabled || sendMessage.isPending}
          placeholder={disabled ? '当前不可发送消息' : '输入要发给 Agent 的消息...'}
        />
      </div>
    </div>
  )
}
