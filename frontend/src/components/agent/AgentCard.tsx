import { Bot, Pencil, Trash2, MoreHorizontal } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { AgentStatusBadge } from '@/components/shared/StatusBadge'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger, DropdownMenuSeparator } from '@/components/ui/dropdown-menu'
import type { Agent } from '@/types'
import { formatRelativeTime } from '@/lib/utils'

const roleLabels: Record<string, string> = {
  pm: 'PM',
  developer: '开发者',
  reviewer: '审核者',
  custom: '自定义',
}

interface AgentCardProps {
  agent: Agent
  onEdit: () => void
  onDelete: () => void
}

function formatUsage(agent: Agent) {
  const parts: string[] = []
  if (agent.usage.project_count > 0) parts.push(`${agent.usage.project_count} 个项目`)
  if (agent.usage.task_count > 0) parts.push(`${agent.usage.task_count} 个任务`)
  if (agent.usage.todo_count > 0) parts.push(`${agent.usage.todo_count} 个 Todo`)
  return parts.join('、')
}

export function AgentCard({ agent, onEdit, onDelete }: AgentCardProps) {
  const usageText = formatUsage(agent)

  return (
    <Card className="transition-all hover:shadow-md">
      <CardContent className="p-4">
        <div className="flex items-start justify-between gap-3">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary/10 text-primary shrink-0">
              <Bot className="h-5 w-5" />
            </div>
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <h3 className="text-sm font-semibold truncate">{agent.name}</h3>
                <AgentStatusBadge status={agent.status} />
              </div>
              <p className="text-xs text-muted-foreground mt-0.5">
                {roleLabels[agent.role]} &middot; {agent.node_id}
              </p>
            </div>
          </div>

          <DropdownMenu>
            <DropdownMenuTrigger className="p-1 rounded hover:bg-muted shrink-0">
              <MoreHorizontal className="h-4 w-4 text-muted-foreground" />
            </DropdownMenuTrigger>
            <DropdownMenuContent>
              <DropdownMenuItem onClick={onEdit}>
                <Pencil className="h-3.5 w-3.5 mr-2" />
                编辑
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onClick={onDelete}
                disabled={agent.usage.in_use}
                className="text-destructive disabled:text-muted-foreground"
                title={agent.usage.in_use ? `该 Agent 正被 ${usageText} 引用` : '删除 Agent'}
              >
                <Trash2 className="h-3.5 w-3.5 mr-2" />
                {agent.usage.in_use ? '删除前需解除引用' : '删除'}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        {agent.description && (
          <p className="text-sm text-muted-foreground mt-3 line-clamp-2">{agent.description}</p>
        )}

        <div className="flex flex-wrap gap-1 mt-3">
          {agent.capabilities.map((cap) => (
            <Badge key={cap} variant="secondary" className="text-xs">
              {cap}
            </Badge>
          ))}
        </div>

        <div className="mt-3 flex items-center gap-2">
          <Badge variant={agent.usage.in_use ? 'destructive' : 'secondary'} className="text-xs">
            {agent.usage.in_use ? '已被引用' : '可删除'}
          </Badge>
          {agent.usage.in_use && (
            <p className="text-xs text-muted-foreground truncate" title={`被 ${usageText} 引用`}>
              被 {usageText} 引用
            </p>
          )}
        </div>

        {agent.last_seen_at && (
          <p className="text-xs text-muted-foreground mt-3">
            最近在线: {formatRelativeTime(agent.last_seen_at)}
          </p>
        )}
      </CardContent>
    </Card>
  )
}
