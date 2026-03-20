import { Link } from 'react-router-dom'
import { Bot } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { AgentStatusDot } from '@/components/shared/StatusBadge'
import { formatRelativeTime } from '@/lib/utils'
import type { Agent } from '@/types'

interface AgentStatusCardProps {
  agent: Agent
}

export function AgentStatusCard({ agent }: AgentStatusCardProps) {
  return (
    <Link to={`/agents/${agent.id}`}>
      <Card className="min-w-[180px] hover:bg-accent/50 transition-colors cursor-pointer">
        <CardContent className="p-4">
          <div className="flex items-center gap-2 mb-2">
            <Bot className="size-4 text-muted-foreground" />
            <span className="text-sm font-medium truncate">{agent.name}</span>
            <AgentStatusDot status={agent.status} />
          </div>
          <div className="flex flex-col gap-0.5 text-xs text-muted-foreground">
            <div className="capitalize">{agent.role}</div>
            {agent.status === 'offline' && agent.last_seen_at && (
              <div>{formatRelativeTime(agent.last_seen_at)}</div>
            )}
            {agent.status !== 'offline' && (
              <div>
                {agent.usage.project_count > 0 && `${agent.usage.project_count} 项目`}
                {agent.usage.todo_count > 0 && ` · ${agent.usage.todo_count} Todo`}
                {agent.usage.total_count === 0 && '空闲'}
              </div>
            )}
          </div>
        </CardContent>
      </Card>
    </Link>
  )
}
