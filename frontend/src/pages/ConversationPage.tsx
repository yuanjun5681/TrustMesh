import { useState, useEffect, useRef } from 'react'
import { useParams, Link } from 'react-router-dom'
import { ArrowLeft, Plus, MessageSquare, AlertCircle, Sparkles, ListTodo, Bug, Lightbulb, ChevronLeft, ChevronRight } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { MessageBubble } from '@/components/conversation/MessageBubble'
import { MessageInput } from '@/components/conversation/MessageInput'
import { PlanPreview } from '@/components/conversation/PlanPreview'
import { EmptyState } from '@/components/shared/EmptyState'
import { useConversations, useConversation, useCreateConversation, useAppendMessage } from '@/hooks/useConversations'
import { useConversationStream } from '@/hooks/useLiveStreams'
import { useProject } from '@/hooks/useProjects'
import { cn, formatRelativeTime } from '@/lib/utils'
import { ApiRequestError } from '@/api/client'
import { toast } from 'sonner'

export function ConversationPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const { data: project } = useProject(projectId)
  const { data: conversations, isLoading } = useConversations(projectId)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [isCreatingNew, setIsCreatingNew] = useState(false)
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  const defaultConversationId =
    conversations?.find((c) => c.status === 'active')?.id ?? conversations?.[0]?.id ?? null
  const activeConversationId = isCreatingNew ? null : selectedId ?? defaultConversationId
  const selectedConversation = conversations?.find((c) => c.id === activeConversationId)
  const { data: conversation } = useConversation(
    activeConversationId ?? undefined,
    selectedConversation?.status === 'active'
  )
  const shouldStreamConversation =
    !!activeConversationId &&
    ((conversation?.status ?? selectedConversation?.status) === 'active' || !conversation)
  useConversationStream(activeConversationId ?? undefined, shouldStreamConversation)
  const createConversation = useCreateConversation()
  const appendMessage = useAppendMessage()

  const pmOffline = project?.pm_agent.status !== 'online'

  // Scroll to bottom on new messages
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [conversation?.messages])

  const handleSend = async (content: string) => {
    setError(null)
    try {
      if (activeConversationId && (conversation?.status ?? selectedConversation?.status) === 'active') {
        await appendMessage.mutateAsync({ id: activeConversationId, input: { content } })
      } else {
        const res = await createConversation.mutateAsync({
          projectId: projectId!,
          input: { content },
        })
        setIsCreatingNew(false)
        setSelectedId(res.data.id)
      }
    } catch (err) {
      if (err instanceof ApiRequestError) {
        const message = err.code === 'PM_AGENT_OFFLINE' ? 'PM Agent 当前离线，无法发送消息' : err.message
        toast.error(message)
        setError(message)
      }
    }
  }

  return (
    <div className="flex h-full">
      {/* Conversation List */}
      <div className={cn(
        'flex flex-col border-r bg-sidebar shrink-0 transition-all duration-200',
        sidebarOpen ? 'w-72' : 'w-0 overflow-hidden border-r-0'
      )}>
        <div className="flex items-center justify-between px-4 py-3 border-b">
          <div className="flex items-center gap-2">
            <Link to={`/projects/${projectId}`}>
              <Button variant="ghost" size="icon" className="size-7">
                <ArrowLeft className="size-4" />
              </Button>
            </Link>
            <span className="text-sm font-medium">对话</span>
          </div>
          <div className="flex items-center gap-1">
            <Button
              variant="ghost"
              size="icon"
              className="size-7"
              onClick={() => {
                setIsCreatingNew(true)
                setSelectedId(null)
              }}
              disabled={pmOffline}
            >
              <Plus className="size-4" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="size-7"
              onClick={() => setSidebarOpen(false)}
            >
              <ChevronLeft className="size-4" />
            </Button>
          </div>
        </div>

        <ScrollArea className="flex-1">
          {isLoading ? (
            <div className="flex flex-col gap-2 p-4">
              {[1, 2, 3].map((i) => (
                <Skeleton key={i} className="h-16 rounded-lg" />
              ))}
            </div>
          ) : !conversations?.length ? (
            <div className="p-4 text-center text-sm text-muted-foreground">
              暂无对话
            </div>
          ) : (
            <div className="flex flex-col gap-0.5 p-2">
              {conversations.map((conv) => (
                <button
                  key={conv.id}
                  onClick={() => {
                    setIsCreatingNew(false)
                    setSelectedId(conv.id)
                  }}
                  className={cn(
                    'w-full rounded-lg p-3 text-left transition-colors hover:bg-sidebar-accent cursor-pointer',
                    activeConversationId === conv.id && 'bg-sidebar-accent'
                  )}
                >
                  <div className="flex items-center justify-between mb-1">
                    <Badge variant={conv.status === 'active' ? 'info' : 'secondary'} className="text-[10px] py-0">
                      {conv.status === 'active' ? '进行中' : '已完成'}
                    </Badge>
                    <span className="text-[10px] text-muted-foreground">
                      {formatRelativeTime(conv.updated_at)}
                    </span>
                  </div>
                  <p className="text-sm line-clamp-2 text-muted-foreground">
                    {conv.last_message.content}
                  </p>
                  {conv.linked_task && (
                    <p className="text-xs text-primary mt-1 truncate">
                      {conv.linked_task.title}
                    </p>
                  )}
                </button>
              ))}
            </div>
          )}
        </ScrollArea>
      </div>

      {/* Message Area */}
      <div className="flex flex-1 flex-col min-w-0">
        {!sidebarOpen && (
          <div className="flex items-center gap-2 px-4 py-2 border-b shrink-0">
            <Button
              variant="ghost"
              size="icon"
              className="size-7"
              onClick={() => setSidebarOpen(true)}
            >
              <ChevronRight className="size-4" />
            </Button>
            <span className="text-sm font-medium truncate">{project?.name}</span>
          </div>
        )}
        {conversation ? (
          <>
            {/* Messages */}
            <ScrollArea className="flex-1 p-4">
              <div className="flex flex-col gap-4 max-w-2xl mx-auto">
                {conversation.messages.map((msg) => (
                  <MessageBubble key={msg.id} message={msg} />
                ))}
                <div ref={messagesEndRef} />
              </div>
            </ScrollArea>

            {/* Linked Task Preview */}
            {conversation.linked_task && (
              <div className="px-4 pb-2 max-w-2xl mx-auto w-full">
                <PlanPreview task={conversation.linked_task} projectId={projectId!} />
              </div>
            )}

            {/* Error */}
            {error && (
              <div className="px-4 pb-2 max-w-2xl mx-auto w-full">
                <div className="flex items-center gap-2 rounded-lg bg-destructive/10 px-3 py-2 text-sm text-destructive">
                  <AlertCircle className="size-4 shrink-0" />
                  {error}
                </div>
              </div>
            )}

            {/* Input */}
            <div className="p-4 max-w-2xl mx-auto w-full">
              {conversation.status === 'resolved' ? (
                <div className="text-center text-sm text-muted-foreground py-2">
                  对话已结束
                </div>
              ) : (
                <MessageInput
                  onSend={handleSend}
                  disabled={appendMessage.isPending || pmOffline}
                  placeholder={pmOffline ? 'PM Agent 离线，无法发送消息' : '输入你的需求...'}
                />
              )}
            </div>
          </>
        ) : activeConversationId ? (
          <div className="flex flex-1 flex-col items-center justify-center">
            <EmptyState
              icon={MessageSquare}
              title="加载中..."
            />
          </div>
        ) : (
          <div className="flex flex-1 flex-col items-center justify-center px-4">
            <div className="w-full max-w-xl flex flex-col items-center">
              <div className="flex size-14 items-center justify-center rounded-2xl bg-primary/10 mb-5">
                <Sparkles className="size-7 text-primary" />
              </div>
              <h2 className="text-xl font-semibold mb-2">开始新对话</h2>
              <p className="text-sm text-muted-foreground mb-8 text-center max-w-sm">
                {pmOffline
                  ? 'PM Agent 当前离线，请等待上线后发起对话'
                  : '向 PM Agent 描述你的需求，AI 将帮你分析并规划任务'}
              </p>

              {!pmOffline && (
                <>
                  <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 w-full mb-6">
                    {[
                      { icon: ListTodo, label: '规划功能', text: '帮我设计一个用户注册登录模块' },
                      { icon: Bug, label: '修复问题', text: '排查并修复订单列表分页异常的问题' },
                      { icon: Lightbulb, label: '技术方案', text: '设计一套消息通知系统的技术方案' },
                    ].map((example) => (
                      <button
                        key={example.label}
                        onClick={() => handleSend(example.text)}
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

                  <div className="w-full">
                    <MessageInput onSend={handleSend} disabled={createConversation.isPending} />
                  </div>
                </>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
