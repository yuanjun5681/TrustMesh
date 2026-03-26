import { useState } from 'react'
import { toast } from 'sonner'
import { ApiRequestError } from '@/api/client'
import { useCreateTask } from '@/hooks/useTasks'
import { useAgents } from '@/hooks/useAgents'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Select, SelectTrigger, SelectContent, SelectItem } from '@/components/ui/select'
import type { TaskPriority } from '@/types'

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectId: string
  onCreated?: (taskId: string) => void
}

const priorities: { value: TaskPriority; label: string }[] = [
  { value: 'low', label: '低' },
  { value: 'medium', label: '中' },
  { value: 'high', label: '高' },
  { value: 'urgent', label: '紧急' },
]

export function CreateTaskDialog({ open, onOpenChange, projectId, onCreated }: Props) {
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [priority, setPriority] = useState<TaskPriority>('medium')
  const [agentId, setAgentId] = useState('')
  const [error, setError] = useState('')

  const createTask = useCreateTask()
  const { data: agents } = useAgents()

  const executorAgents = agents?.filter((a) => !a.archived && a.role !== 'pm') ?? []

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen) {
      setTitle('')
      setDescription('')
      setPriority('medium')
      setAgentId('')
      setError('')
    }
    onOpenChange(nextOpen)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    try {
      const res = await createTask.mutateAsync({
        projectId,
        input: {
          title: title.trim(),
          description: description.trim(),
          priority,
          assignee_agent_id: agentId,
        },
      })
      toast.success('任务已创建')
      const taskId = res.data.id
      handleOpenChange(false)
      onCreated?.(taskId)
    } catch (err) {
      const message = err instanceof ApiRequestError ? err.message : '创建任务失败'
      toast.error(message)
      setError(message)
    }
  }

  const isSubmitDisabled = createTask.isPending || !title.trim() || !description.trim() || !agentId

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>创建任务</DialogTitle>
          <DialogDescription>手动创建一个任务并指派给 Agent 执行</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="mt-4 flex flex-col gap-4">
          <div className="flex flex-col gap-2">
            <label className="text-sm font-medium">任务标题</label>
            <Input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="例如：实现用户登录功能"
              required
            />
          </div>
          <div className="flex flex-col gap-2">
            <label className="text-sm font-medium">任务描述</label>
            <Textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="描述任务的目标和要求"
              rows={3}
              required
            />
          </div>
          <div className="flex flex-col gap-2">
            <label className="text-sm font-medium">优先级</label>
            <Select value={priority} onValueChange={(val) => setPriority(val as TaskPriority)}>
              <SelectTrigger className="w-full">
                <span>{priorities.find((p) => p.value === priority)?.label ?? '中'}</span>
              </SelectTrigger>
              <SelectContent>
                {priorities.map((p) => (
                  <SelectItem key={p.value} value={p.value}>
                    {p.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="flex flex-col gap-2">
            <label className="text-sm font-medium">执行 Agent</label>
            <Select value={agentId} onValueChange={(val) => setAgentId(val ?? '')}>
              <SelectTrigger className="w-full">
                <span>
                  {executorAgents.find((a) => a.id === agentId)
                    ? `${executorAgents.find((a) => a.id === agentId)!.name} - ${executorAgents.find((a) => a.id === agentId)!.status === 'online' ? '在线' : '离线'}`
                    : '选择执行 Agent...'}
                </span>
              </SelectTrigger>
              <SelectContent>
                {executorAgents.map((agent) => (
                  <SelectItem key={agent.id} value={agent.id}>
                    {agent.name} ({agent.role}) - {agent.status === 'online' ? '在线' : '离线'}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {executorAgents.length === 0 && (
              <p className="text-xs text-muted-foreground">
                暂无可用的执行 Agent，请先在 Agent 管理中添加
              </p>
            )}
          </div>
          {error && <p className="text-sm text-destructive">{error}</p>}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => handleOpenChange(false)}>
              取消
            </Button>
            <Button type="submit" disabled={isSubmitDisabled}>
              {createTask.isPending ? '创建中...' : '创建任务'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
