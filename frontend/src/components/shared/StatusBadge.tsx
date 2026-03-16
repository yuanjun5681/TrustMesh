import { Badge } from '@/components/ui/badge'
import type { TaskStatus, TaskPriority, AgentStatus } from '@/types'

const taskStatusConfig: Record<TaskStatus, { label: string; variant: 'secondary' | 'info' | 'success' | 'destructive' }> = {
  pending: { label: '待处理', variant: 'secondary' },
  in_progress: { label: '进行中', variant: 'info' },
  done: { label: '已完成', variant: 'success' },
  failed: { label: '失败', variant: 'destructive' },
}

const priorityConfig: Record<TaskPriority, { label: string; className: string }> = {
  low: { label: '低', className: 'bg-priority-low/15 text-priority-low' },
  medium: { label: '中', className: 'bg-priority-medium/15 text-priority-medium' },
  high: { label: '高', className: 'bg-priority-high/15 text-priority-high' },
  urgent: { label: '紧急', className: 'bg-priority-urgent/15 text-priority-urgent' },
}

const agentStatusConfig: Record<AgentStatus, { label: string; className: string }> = {
  online: { label: '在线', className: 'bg-status-online/15 text-status-online' },
  offline: { label: '离线', className: 'bg-status-offline/15 text-status-offline' },
  busy: { label: '忙碌', className: 'bg-status-busy/15 text-status-busy' },
}

export function TaskStatusBadge({ status }: { status: TaskStatus }) {
  const config = taskStatusConfig[status]
  return <Badge variant={config.variant}>{config.label}</Badge>
}

export function PriorityBadge({ priority }: { priority: TaskPriority }) {
  const config = priorityConfig[priority]
  return (
    <Badge className={config.className} variant="outline">
      {config.label}
    </Badge>
  )
}

export function AgentStatusBadge({ status }: { status: AgentStatus }) {
  const config = agentStatusConfig[status]
  return <Badge className={config.className}>{config.label}</Badge>
}

export function AgentStatusDot({ status }: { status: AgentStatus }) {
  const colorClass =
    status === 'online' ? 'bg-status-online' :
    status === 'busy' ? 'bg-status-busy' : 'bg-status-offline'
  return <span className={`inline-block h-2 w-2 rounded-full ${colorClass}`} />
}
