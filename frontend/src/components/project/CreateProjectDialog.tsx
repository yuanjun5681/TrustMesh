import { useState } from 'react'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select'
import { useCreateProject } from '@/hooks/useProjects'
import { useAgents } from '@/hooks/useAgents'
import { ApiRequestError } from '@/api/client'
import { toast } from 'sonner'

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function CreateProjectDialog({ open, onOpenChange }: Props) {
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [pmAgentId, setPmAgentId] = useState('')
  const [error, setError] = useState('')
  const { data: agents } = useAgents()
  const createProject = useCreateProject()

  const pmAgents = agents?.filter((a) => a.role === 'pm') ?? []

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    try {
      await createProject.mutateAsync({
        name,
        description,
        pm_agent_id: pmAgentId,
      })
      toast.success('项目已创建')
      setName('')
      setDescription('')
      setPmAgentId('')
      onOpenChange(false)
    } catch (err) {
      const message = err instanceof ApiRequestError ? err.message : '创建失败'
      toast.error(message)
      setError(message)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>创建项目</DialogTitle>
          <DialogDescription>新建一个 AI Agent 协作项目</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="flex flex-col gap-4 mt-4">
          <div className="flex flex-col gap-2">
            <label className="text-sm font-medium">项目名称</label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="例如：TrustMesh MVP"
              required
            />
          </div>
          <div className="flex flex-col gap-2">
            <label className="text-sm font-medium">项目描述</label>
            <Textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="描述项目的目标和范围"
              rows={3}
              required
            />
          </div>
          <div className="flex flex-col gap-2">
            <label className="text-sm font-medium">PM Agent</label>
            <Select value={pmAgentId} onValueChange={(val) => setPmAgentId(val ?? '')}>
              <SelectTrigger className="w-full">
                <SelectValue placeholder="选择 PM Agent..." />
              </SelectTrigger>
              <SelectContent>
                {pmAgents.map((agent) => (
                  <SelectItem key={agent.id} value={agent.id}>
                    {agent.name} ({agent.node_id}) - {agent.status === 'online' ? '在线' : '离线'}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {pmAgents.length === 0 && (
              <p className="text-xs text-muted-foreground">
                暂无可用的 PM Agent，请先在 Agent 管理中添加
              </p>
            )}
          </div>
          {error && <p className="text-sm text-destructive">{error}</p>}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              取消
            </Button>
            <Button type="submit" disabled={createProject.isPending || !pmAgentId}>
              {createProject.isPending ? '创建中...' : '创建项目'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
