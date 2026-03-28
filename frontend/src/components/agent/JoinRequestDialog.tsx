import { useState } from 'react'
import { Check, X, Network } from 'lucide-react'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Avatar } from '@/components/ui/avatar'
import { Input } from '@/components/ui/input'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select'
import { useJoinRequests, useApproveJoinRequest, useRejectJoinRequest } from '@/hooks/useJoinRequests'
import { ApiRequestError } from '@/api/client'
import { toast } from 'sonner'
import type { JoinRequest, AgentRole } from '@/types'

const ROLE_LABELS: Record<string, string> = {
  pm: 'PM',
  developer: '开发者',
  reviewer: '审核者',
  custom: '自定义',
}

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function JoinRequestDialog({ open, onOpenChange }: Props) {
  const { data: requests } = useJoinRequests('pending')
  const approveRequest = useApproveJoinRequest()
  const rejectRequest = useRejectJoinRequest()
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editName, setEditName] = useState('')
  const [editRole, setEditRole] = useState<AgentRole>('developer')
  const [editDescription, setEditDescription] = useState('')

  const startEdit = (jr: JoinRequest) => {
    setEditingId(jr.id)
    setEditName(jr.name)
    setEditRole(jr.role)
    setEditDescription(jr.description)
  }

  const handleApprove = async (jr: JoinRequest) => {
    try {
      const overrides = editingId === jr.id
        ? { name: editName, role: editRole, description: editDescription }
        : undefined
      await approveRequest.mutateAsync({ id: jr.id, overrides })
      toast.success(`已批准 Agent「${overrides?.name || jr.name}」加入`)
      setEditingId(null)
    } catch (err) {
      toast.error(err instanceof ApiRequestError ? err.message : '批准失败')
    }
  }

  const handleReject = async (jr: JoinRequest) => {
    try {
      await rejectRequest.mutateAsync(jr.id)
      toast.success(`已拒绝 Agent「${jr.name}」的申请`)
    } catch (err) {
      toast.error(err instanceof ApiRequestError ? err.message : '拒绝失败')
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:!max-w-lg max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>审批请求</DialogTitle>
        </DialogHeader>

        {(!requests || requests.length === 0) ? (
          <p className="text-sm text-muted-foreground py-4 text-center">暂无待审批的请求</p>
        ) : (
          <div className="space-y-3">
            {requests.map((jr) => (
              <div key={jr.id} className="rounded-lg border p-4 space-y-3">
                {editingId === jr.id ? (
                  <div className="space-y-2">
                    <Input
                      value={editName}
                      onChange={(e) => setEditName(e.target.value)}
                      placeholder="Agent 名称"
                    />
                    <Select value={editRole} onValueChange={(v) => setEditRole(v as AgentRole)}>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="pm">PM</SelectItem>
                        <SelectItem value="developer">开发者</SelectItem>
                        <SelectItem value="reviewer">审核者</SelectItem>
                        <SelectItem value="custom">自定义</SelectItem>
                      </SelectContent>
                    </Select>
                    <Input
                      value={editDescription}
                      onChange={(e) => setEditDescription(e.target.value)}
                      placeholder="描述"
                    />
                  </div>
                ) : (
                  <div className="flex gap-3">
                    <Avatar
                      fallback={jr.name}
                      seed={jr.node_id}
                      kind="agent"
                      role={jr.role}
                      size="lg"
                      className="shrink-0"
                    />
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="font-semibold text-base">{jr.name}</span>
                        <Badge variant="outline">{ROLE_LABELS[jr.role] ?? jr.role}</Badge>
                        {jr.agent_product && (
                          <Badge variant="secondary">{jr.agent_product}</Badge>
                        )}
                      </div>
                      <p className="text-sm text-muted-foreground mt-1.5 line-clamp-2">
                        {jr.description || '无描述'}
                      </p>
                      <div className="flex items-center gap-3 mt-2 text-xs text-muted-foreground">
                        <span className="inline-flex items-center gap-1">
                          <Network className="size-3" />
                          {jr.node_id}
                        </span>
                        {jr.capabilities.length > 0 && (
                          <span>能力: {jr.capabilities.join(', ')}</span>
                        )}
                      </div>
                    </div>
                  </div>
                )}

                <div className="flex items-center justify-end gap-2 pt-1 border-t">
                  {editingId === jr.id ? (
                    <Button size="sm" variant="ghost" onClick={() => setEditingId(null)}>
                      取消
                    </Button>
                  ) : (
                    <Button size="sm" variant="ghost" onClick={() => startEdit(jr)}>
                      编辑
                    </Button>
                  )}
                  <Button
                    size="sm"
                    variant="outline"
                    className="text-red-600 hover:text-red-700"
                    onClick={() => handleReject(jr)}
                    disabled={rejectRequest.isPending}
                  >
                    <X className="size-4 mr-1" />
                    拒绝
                  </Button>
                  <Button
                    size="sm"
                    onClick={() => handleApprove(jr)}
                    disabled={approveRequest.isPending}
                  >
                    <Check className="size-4 mr-1" />
                    批准
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}
