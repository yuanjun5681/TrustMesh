import { useState, useEffect, useRef } from 'react'
import { AlertCircle, MessageSquare, ListChecks, Sparkles, ListTodo, Bug, Lightbulb } from 'lucide-react'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Sheet, SheetContent, SheetTitle } from '@/components/ui/sheet'
import { MessageBubble } from '@/components/conversation/MessageBubble'
import { MessageInput } from '@/components/conversation/MessageInput'
import { ThinkingIndicator } from '@/components/conversation/ThinkingIndicator'
import {
  useConversation,
  useCreateConversation,
  useAppendMessage,
} from '@/hooks/useConversations'
import { useConversationStream } from '@/hooks/useLiveStreams'
import { useProject } from '@/hooks/useProjects'
import { ApiRequestError } from '@/api/client'
import { toast } from 'sonner'

interface ConversationSheetProps {
  projectId: string
  open: boolean
  onOpenChange: (open: boolean) => void
  /** 传入已有对话 ID 时进入历史查看模式 */
  initialConversationId?: string
  onTaskCreated?: (taskId: string) => void
}

export function ConversationSheet({ projectId, open, onOpenChange, initialConversationId, onTaskCreated }: ConversationSheetProps) {
  const { data: project } = useProject(projectId)
  const [conversationId, setConversationId] = useState<string | null>(initialConversationId ?? null)
  const [error, setError] = useState<string | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const [inputValue, setInputValue] = useState<string | undefined>(undefined)

  const isHistoryMode = !!initialConversationId
  const pmOffline = project?.pm_agent.status !== 'online'

  const handleOpenChange = (nextOpen: boolean) => {
    if (nextOpen) {
      setConversationId(initialConversationId ?? null)
      setError(null)
      setInputValue(undefined)
    }
    onOpenChange(nextOpen)
  }

  const { data: conversation } = useConversation(
    open && conversationId ? conversationId : undefined,
    !isHistoryMode
  )

  const shouldStream = open && !!conversationId && conversation?.status !== 'resolved' && !isHistoryMode
  useConversationStream(conversationId ?? undefined, shouldStream)

  const createConversation = useCreateConversation()
  const appendMessage = useAppendMessage()

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [conversation?.messages])

  const handleSend = async (content: string) => {
    setError(null)
    try {
      if (conversationId && conversation?.status === 'active') {
        await appendMessage.mutateAsync({ id: conversationId, input: { content } })
      } else {
        const res = await createConversation.mutateAsync({
          projectId,
          input: { content },
        })
        setConversationId(res.data.id)
      }
    } catch (err) {
      if (err instanceof ApiRequestError) {
        const message = err.code === 'PM_AGENT_OFFLINE' ? 'PM Agent 当前离线，无法发送消息' : err.message
        toast.error(message)
        setError(message)
      }
    }
  }

  const handleViewTask = (taskId: string) => {
    onTaskCreated?.(taskId)
    onOpenChange(false)
  }

  const title = isHistoryMode ? '需求对话记录' : '提交新需求'

  return (
    <Sheet open={open} onOpenChange={handleOpenChange}>
      <SheetContent side="right" showCloseButton className="w-full max-w-4xl! gap-0! p-0 overflow-hidden">
        {/* Header */}
        <div className="flex items-center gap-2 px-4 py-3 border-b shrink-0">
          <MessageSquare className="size-4 text-muted-foreground" />
          <SheetTitle className="text-sm font-medium flex-1 p-0! m-0!">{title}</SheetTitle>
        </div>

        {/* Content Area */}
        {conversation ? (
          <ScrollArea className="flex-1 min-h-0 p-4">
            <div className="flex flex-col gap-4">
              {conversation.messages.map((msg) => (
                <MessageBubble key={msg.id} message={msg} />
              ))}

              {/* Thinking Indicator */}
              {!isHistoryMode &&
                conversation.status === 'active' &&
                conversation.messages.length > 0 &&
                conversation.messages[conversation.messages.length - 1].role === 'user' && (
                  <ThinkingIndicator />
                )}

              {/* Task Created Banner */}
              {!isHistoryMode && conversation.linked_task && (
                <div className="flex items-center gap-2 py-2 my-1">
                  <div className="flex-1 border-t" />
                  <button
                    onClick={() => handleViewTask(conversation.linked_task!.id)}
                    className="flex items-center gap-1.5 text-xs text-primary hover:text-primary/80 transition-colors cursor-pointer shrink-0"
                  >
                    <ListChecks className="size-3.5" />
                    <span>任务已创建：{conversation.linked_task.title}</span>
                    <span className="text-muted-foreground">[查看]</span>
                  </button>
                  <div className="flex-1 border-t" />
                </div>
              )}

              <div ref={messagesEndRef} />
            </div>
          </ScrollArea>
        ) : conversationId ? (
          <div className="flex flex-1 items-center justify-center text-sm text-muted-foreground">
            加载中...
          </div>
        ) : (
          /* New conversation - empty state with examples */
          <div className="flex flex-1 flex-col items-center justify-center px-6">
            <div className="w-full max-w-md flex flex-col items-center">
              <div className="flex size-14 items-center justify-center rounded-2xl bg-primary/10 mb-5">
                <Sparkles className="size-7 text-primary" />
              </div>
              <h2 className="text-xl font-semibold mb-2">提交新需求</h2>
              <p className="text-sm text-muted-foreground mb-8 text-center max-w-sm">
                {pmOffline
                  ? 'PM Agent 当前离线，请等待上线后发起对话'
                  : '向 PM Agent 描述你的需求，AI 将帮你分析并规划任务'}
              </p>

              {!pmOffline && (
                <div className="grid grid-cols-1 gap-3 w-full">
                  {[
                    { icon: ListTodo, label: '规划功能', text: '帮我设计一个用户注册登录模块，需要支持邮箱和手机号两种注册方式，包含密码强度校验、邮箱验证码、登录态管理等功能' },
                    { icon: Bug, label: '修复问题', text: '排查并修复订单列表分页异常的问题：当筛选条件变更后切换到第二页，返回的数据仍然是旧筛选条件的结果，疑似缓存未清除' },
                    { icon: Lightbulb, label: '技术方案', text: '设计一套消息通知系统的技术方案，需支持站内信、邮件、WebSocket 实时推送三种渠道，要求支持消息模板、已读未读状态跟踪和批量发送' },
                  ].map((example) => (
                    <button
                      key={example.label}
                      onClick={() => setInputValue(example.text)}
                      disabled={createConversation.isPending}
                      className="group flex flex-col items-start gap-2 rounded-xl border bg-card p-4 text-left transition-all hover:border-primary/30 hover:shadow-sm disabled:opacity-50 cursor-pointer"
                    >
                      <div className="flex items-center gap-2 text-xs font-medium text-muted-foreground group-hover:text-primary transition-colors">
                        <example.icon className="size-3.5" />
                        {example.label}
                      </div>
                      <span className="text-sm leading-snug">{example.text}</span>
                    </button>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}

        {/* Error */}
        {error && (
          <div className="px-4 pb-2">
            <div className="flex items-center gap-2 rounded-lg bg-destructive/10 px-3 py-2 text-sm text-destructive">
              <AlertCircle className="size-4 shrink-0" />
              {error}
            </div>
          </div>
        )}

        {/* Footer */}
        <div className="px-4 py-3 shrink-0 flex justify-center">
          <div className="w-full max-w-2xl">
          {isHistoryMode ? (
            conversation?.status === 'resolved' ? (
              <div className="text-center text-sm text-muted-foreground py-2">
                对话已结束
              </div>
            ) : null
          ) : conversation?.status === 'resolved' ? (
            <div className="text-center text-sm text-muted-foreground py-2">
              对话已结束
            </div>
          ) : (
            <MessageInput
              onSend={handleSend}
              disabled={(conversation ? appendMessage.isPending : createConversation.isPending) || pmOffline}
              placeholder={pmOffline ? 'PM Agent 离线，无法发送消息' : '输入你的需求...'}
              defaultValue={inputValue}
            />
          )}
          </div>
        </div>
      </SheetContent>
    </Sheet>
  )
}
