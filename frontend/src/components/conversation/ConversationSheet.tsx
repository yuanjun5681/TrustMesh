import { useState, useEffect, useRef, useMemo } from 'react'
import { AlertCircle, MessageSquare, ListChecks, Sparkles, ListTodo, Bug, Lightbulb } from 'lucide-react'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Sheet, SheetContent, SheetTitle } from '@/components/ui/sheet'
import { MessageBubble } from '@/components/conversation/MessageBubble'
import { MessageInput } from '@/components/conversation/MessageInput'
import { ThinkingIndicator } from '@/components/conversation/ThinkingIndicator'
import { UIResponsePanel } from '@/components/conversation/UIResponsePanel'
import {
  useConversation,
  useCreateConversation,
  useAppendMessage,
} from '@/hooks/useConversations'
import { useProject } from '@/hooks/useProjects'
import { ApiRequestError } from '@/api/client'
import { toast } from 'sonner'
import type { UIResponse, ConversationMessage } from '@/types'

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
  const projectArchived = project?.status === 'archived'

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
    {
      isActiveHint: !isHistoryMode,
    }
  )

  const createConversation = useCreateConversation()
  const appendMessage = useAppendMessage()

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [conversation?.messages])

  // 检测最新一条 PM 消息是否含有未回复的 ui_blocks
  const pendingUIBlocks = useMemo(() => {
    if (!conversation || conversation.status !== 'active' || isHistoryMode) return null
    const msgs = conversation.messages
    if (msgs.length === 0) return null
    const lastMsg = msgs[msgs.length - 1]
    // 最新消息是 PM 且带 ui_blocks → 未回复
    if (lastMsg.role === 'pm_agent' && lastMsg.ui_blocks && lastMsg.ui_blocks.length > 0) {
      return lastMsg.ui_blocks
    }
    return null
  }, [conversation, isHistoryMode])

  // 为含 ui_blocks 的 PM 消息查找下一条用户消息（用于回显选择结果）
  const findNextUserResponse = (msgs: ConversationMessage[], index: number): ConversationMessage | undefined => {
    if (index + 1 < msgs.length && msgs[index + 1].role === 'user') {
      return msgs[index + 1]
    }
    return undefined
  }

  const handleSend = async (content: string, uiResponse?: UIResponse) => {
    if (projectArchived) {
      const message = '项目已归档，无法继续发送消息'
      toast.error(message)
      setError(message)
      return
    }

    setError(null)
    try {
      if (conversationId && conversation?.status === 'active') {
        await appendMessage.mutateAsync({
          id: conversationId,
          input: { content, ui_response: uiResponse },
        })
      } else {
        const res = await createConversation.mutateAsync({
          projectId,
          input: { content },
        })
        setConversationId(res.data.id)
      }
      setInputValue(undefined)
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
              {conversation.messages.map((msg, i) => {
                const isLastMsg = i === conversation.messages.length - 1
                const hasPendingBlocks = isLastMsg && !!pendingUIBlocks
                return (
                  <MessageBubble
                    key={msg.id}
                    message={msg}
                    nextUserResponse={
                      msg.role === 'pm_agent' && msg.ui_blocks?.length
                        ? findNextUserResponse(conversation.messages, i)
                        : undefined
                    }
                    hideUIBlocks={hasPendingBlocks}
                  />
                )
              })}

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
                {projectArchived
                  ? '项目已归档，历史内容仍可查看，但不能继续发起协作'
                  : pmOffline
                  ? 'PM Agent 当前离线，请等待上线后发起对话'
                  : '向 PM Agent 描述你的需求，AI 将帮你分析并规划任务'}
              </p>

              {!pmOffline && !projectArchived && (
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
          {conversation?.status === 'resolved' ? (
            <div className="text-center text-sm text-muted-foreground py-2">
              对话已结束
            </div>
          ) : pendingUIBlocks ? (
            <UIResponsePanel
              blocks={pendingUIBlocks}
              onSubmit={(content, uiResponse) => handleSend(content, uiResponse)}
              disabled={appendMessage.isPending || pmOffline || projectArchived}
            />
          ) : (
            <MessageInput
              onSend={handleSend}
              disabled={(conversation ? appendMessage.isPending : createConversation.isPending) || pmOffline || projectArchived}
              placeholder={
                projectArchived
                  ? '项目已归档，无法发送消息'
                  : pmOffline
                  ? 'PM Agent 离线，无法发送消息'
                  : '输入你的需求...'
              }
              defaultValue={inputValue}
            />
          )}
          </div>
        </div>
      </SheetContent>
    </Sheet>
  )
}
