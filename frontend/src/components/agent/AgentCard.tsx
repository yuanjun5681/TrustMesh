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

export function AgentCard({ agent, onEdit, onDelete }: AgentCardProps) {
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
              <DropdownMenuItem onClick={onDelete} className="text-destructive">
                <Trash2 className="h-3.5 w-3.5 mr-2" />
                删除
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

        {agent.heartbeat_at && (
          <p className="text-xs text-muted-foreground mt-3">
            最近心跳: {formatRelativeTime(agent.heartbeat_at)}
          </p>
        )}
      </CardContent>
    </Card>
  )
}
