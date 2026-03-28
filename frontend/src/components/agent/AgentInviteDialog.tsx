import { Copy, Check } from 'lucide-react'
import { useState } from 'react'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { useInvitePrompt } from '@/hooks/useJoinRequests'
import { toast } from 'sonner'

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function AgentInviteDialog({ open, onOpenChange }: Props) {
  const { data: invite, isLoading } = useInvitePrompt(open)
  const [copied, setCopied] = useState(false)

  const handleCopy = async () => {
    if (!invite?.prompt) return
    try {
      await navigator.clipboard.writeText(invite.prompt)
      setCopied(true)
      toast.success('提示词已复制到剪贴板')
      setTimeout(() => setCopied(false), 2000)
    } catch {
      toast.error('复制失败，请手动选择复制')
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:!max-w-4xl">
        <DialogHeader>
          <DialogTitle>邀请 Agent 加入</DialogTitle>
        </DialogHeader>

        <p className="text-sm text-muted-foreground">
          复制以下提示词并发送给 openclaw Agent，Agent 将自动发起加入申请。
        </p>

        {isLoading ? (
          <div className="h-48 rounded-lg bg-muted animate-pulse" />
        ) : (
          <div className="relative">
            <pre className="rounded-lg bg-muted p-4 text-sm leading-relaxed overflow-auto max-h-96 whitespace-pre-wrap">
              {invite?.prompt}
            </pre>
            <Button
              size="sm"
              variant="outline"
              className="absolute top-2 right-2"
              onClick={handleCopy}
            >
              {copied ? (
                <Check className="size-4 mr-1" />
              ) : (
                <Copy className="size-4 mr-1" />
              )}
              {copied ? '已复制' : '复制'}
            </Button>
          </div>
        )}

        <p className="text-xs text-muted-foreground">
          Agent 发送申请后，你将在此页面看到待审批的请求。
        </p>
      </DialogContent>
    </Dialog>
  )
}
