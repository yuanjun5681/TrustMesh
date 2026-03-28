import { useState } from 'react'
import { Check, X, ChevronDown, ChevronUp } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
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

export function JoinRequestList() {
  const { data: requests } = useJoinRequests('pending')
  const approveRequest = useApproveJoinRequest()
  const rejectRequest = useRejectJoinRequest()
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editName, setEditName] = useState('')
  const [editRole, setEditRole] = useState<AgentRole>('developer')
  const [editDescription, setEditDescription] = useState('')
  const [collapsed, setCollapsed] = useState(false)

  if (!requests || requests.length === 0) return null

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
    <div className="mb-6 rounded-xl border border-amber-200 bg-amber-50/50 dark:border-amber-900 dark:bg-amber-950/20">
      <button
        className="flex w-full items-center justify-between p-4 cursor-pointer"
        onClick={() => setCollapsed(!collapsed)}
      >
        <div className="flex items-center gap-2">
          <Badge variant="secondary" className="bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-200">
            {requests.length} 个待审批
          </Badge>
          <span className="text-sm font-medium">Agent 加入申请</span>
        </div>
        {collapsed ? <ChevronDown className="size-4" /> : <ChevronUp className="size-4" />}
      </button>

      {!collapsed && (
        <div className="space-y-3 px-4 pb-4">
          {requests.map((jr) => (
            <div key={jr.id} className="rounded-lg border bg-background p-4">
              <div className="flex items-start justify-between gap-4">
                <div className="flex-1 min-w-0">
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
                    <>
                      <div className="flex items-center gap-2">
                        <span className="font-medium">{jr.name}</span>
                        <Badge variant="outline">{ROLE_LABELS[jr.role] ?? jr.role}</Badge>
                        {jr.agent_product && (
                          <Badge variant="secondary">{jr.agent_product}</Badge>
                        )}
                      </div>
                      <p className="text-sm text-muted-foreground mt-1 truncate">
                        {jr.description || '无描述'}
                      </p>
                      <p className="text-xs text-muted-foreground mt-1">
                        节点: {jr.node_id}
                        {jr.capabilities.length > 0 && (
                          <> · 能力: {jr.capabilities.join(', ')}</>
                        )}
                      </p>
                    </>
                  )}
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  {editingId !== jr.id && (
                    <Button size="sm" variant="ghost" onClick={() => startEdit(jr)}>
                      编辑
                    </Button>
                  )}
                  {editingId === jr.id && (
                    <Button size="sm" variant="ghost" onClick={() => setEditingId(null)}>
                      取消
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
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
