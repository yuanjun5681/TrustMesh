import { useState } from 'react'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select'
import { useUpsertPlatformConnection } from '@/hooks/usePlatformConnections'
import { useAgents } from '@/hooks/useAgents'
import { ApiRequestError } from '@/api/client'
import { toast } from 'sonner'
import type { PlatformConnection } from '@/types'

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  existing?: PlatformConnection
}

export function ConnectClawHireDialog({ open, onOpenChange, existing }: Props) {
  const [platformNodeId, setPlatformNodeId] = useState(existing?.platform_node_id ?? '')
  const [remoteUserId, setRemoteUserId] = useState(existing?.remote_user_id ?? '')
  const [pmAgentId, setPmAgentId] = useState(existing?.pm_agent_id ?? '')

  const { data: agents } = useAgents()
  const pmAgents = agents?.filter((a) => a.role === 'pm' && !a.archived) ?? []

  const upsert = useUpsertPlatformConnection()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await upsert.mutateAsync({
        platform: 'clawhire',
        platform_node_id: platformNodeId.trim(),
        remote_user_id: remoteUserId.trim(),
        pm_agent_id: pmAgentId,
      })
      toast.success('ClawHire 账号已绑定')
      onOpenChange(false)
    } catch (err) {
      const message = err instanceof ApiRequestError ? err.message : '绑定失败，请检查填写内容'
      toast.error(message)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>连接 ClawHire</DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="flex flex-col gap-4 mt-2">
          <div className="flex flex-col gap-1.5">
            <label className="text-sm font-medium">ClawHire 节点 ID <span className="text-destructive">*</span></label>
            <Input
              placeholder="ClawHire 平台的 clawsynapse nodeId"
              value={platformNodeId}
              onChange={(e) => setPlatformNodeId(e.target.value)}
              required
            />
            <p className="text-xs text-muted-foreground">在 ClawHire 平台设置中查看节点 ID</p>
          </div>

          <div className="flex flex-col gap-1.5">
            <label className="text-sm font-medium">我的 ClawHire 账号 ID <span className="text-destructive">*</span></label>
            <Input
              placeholder="你在 ClawHire 的用户 ID"
              value={remoteUserId}
              onChange={(e) => setRemoteUserId(e.target.value)}
              required
            />
          </div>

          <div className="flex flex-col gap-1.5">
            <label className="text-sm font-medium">负责 PM Agent <span className="text-destructive">*</span></label>
            <Select value={pmAgentId} onValueChange={(v) => setPmAgentId(v ?? '')} required>
              <SelectTrigger>
                <SelectValue placeholder="选择处理 ClawHire 任务的 PM Agent" />
              </SelectTrigger>
              <SelectContent>
                {pmAgents.length === 0 && (
                  <SelectItem value="__none__" disabled>暂无在线 PM Agent</SelectItem>
                )}
                {pmAgents.map((a) => (
                  <SelectItem key={a.id} value={a.id}>
                    {a.name}
                    <span className="ml-2 text-xs text-muted-foreground">{a.status}</span>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className="text-xs text-muted-foreground">收到 ClawHire 任务时，由此 Agent 负责规划执行</p>
          </div>

          <DialogFooter className="mt-2">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              取消
            </Button>
            <Button type="submit" disabled={upsert.isPending || !platformNodeId || !remoteUserId || !pmAgentId}>
              {upsert.isPending ? '绑定中…' : '确认绑定'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
