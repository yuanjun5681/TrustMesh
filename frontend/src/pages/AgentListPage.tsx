import { useState } from 'react'
import { Bot, Plus } from 'lucide-react'
import { PageContainer } from '@/components/layout/PageContainer'
import { Button } from '@/components/ui/button'
import { AgentCard } from '@/components/agent/AgentCard'
import { AgentConfigDialog } from '@/components/agent/AgentConfigDialog'
import { ArchiveAgentDialog } from '@/components/agent/ArchiveAgentDialog'
import { EmptyState } from '@/components/shared/EmptyState'
import { Skeleton } from '@/components/ui/skeleton'
import { useAgents, useDeleteAgent } from '@/hooks/useAgents'
import { ApiRequestError } from '@/api/client'
import { toast } from 'sonner'
import type { Agent, AgentRole } from '@/types'

const ROLE_FILTERS: { label: string; value: AgentRole | 'all' }[] = [
  { label: '全部', value: 'all' },
  { label: 'PM', value: 'pm' },
  { label: '开发者', value: 'developer' },
  { label: '审核者', value: 'reviewer' },
  { label: '自定义', value: 'custom' },
]

export function AgentListPage() {
  const { data: agents, isLoading } = useAgents()
  const deleteAgent = useDeleteAgent()
  const [showConfig, setShowConfig] = useState(false)
  const [editingAgent, setEditingAgent] = useState<Agent | null>(null)
  const [roleFilter, setRoleFilter] = useState<AgentRole | 'all'>('all')
  const [error, setError] = useState('')
  const [archiveTarget, setArchiveTarget] = useState<Agent | null>(null)

  const filteredAgents = agents?.filter(
    (a) => roleFilter === 'all' || a.role === roleFilter
  ) ?? []

  const handleEdit = (agent: Agent) => {
    setError('')
    setEditingAgent(agent)
    setShowConfig(true)
  }

  const handleDelete = async (agent: Agent) => {
    setError('')
    if (agent.usage.in_use) {
      setArchiveTarget(agent)
      return
    }

    if (confirm(`确定要删除 Agent "${agent.name}" 吗？`)) {
      try {
        await deleteAgent.mutateAsync(agent.id)
        toast.success(`Agent "${agent.name}" 已删除`)
      } catch (err) {
        const message = err instanceof ApiRequestError ? err.message : '删除 Agent 失败'
        toast.error(message)
        setError(message)
      }
    }
  }

  const handleClose = () => {
    setShowConfig(false)
    setEditingAgent(null)
  }

  return (
    <PageContainer>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold">Agent 管理</h1>
          <p className="text-muted-foreground mt-1">管理 AI Agent 的配置和状态</p>
        </div>
        <Button onClick={() => { setEditingAgent(null); setShowConfig(true) }}>
          <Plus className="size-4 mr-2" />
          添加 Agent
        </Button>
      </div>

      {/* Filter Toolbar */}
      <div className="flex gap-1 mb-6 rounded-lg bg-muted p-1 w-fit">
        {ROLE_FILTERS.map((filter) => (
          <button
            key={filter.value}
            onClick={() => setRoleFilter(filter.value)}
            className={`rounded-md px-3 py-1.5 text-sm font-medium transition-colors cursor-pointer ${
              roleFilter === filter.value
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            {filter.label}
          </button>
        ))}
      </div>

      {error && <p className="mb-4 text-sm text-destructive">{error}</p>}

      {isLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-40 rounded-xl" />
          ))}
        </div>
      ) : filteredAgents.length === 0 ? (
        <EmptyState
          icon={Bot}
          title="暂无 Agent"
          description="添加你的第一个 AI Agent 开始协作"
          action={
            <Button onClick={() => { setEditingAgent(null); setShowConfig(true) }}>
              <Plus className="size-4 mr-2" />
              添加 Agent
            </Button>
          }
        />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {filteredAgents.map((agent) => (
            <AgentCard
              key={agent.id}
              agent={agent}
              onEdit={() => handleEdit(agent)}
              onDelete={() => handleDelete(agent)}
            />
          ))}
        </div>
      )}

      <AgentConfigDialog
        open={showConfig}
        onOpenChange={handleClose}
        agent={editingAgent}
      />

      <ArchiveAgentDialog
        open={!!archiveTarget}
        onOpenChange={(open) => { if (!open) setArchiveTarget(null) }}
        agent={archiveTarget ?? undefined}
      />
    </PageContainer>
  )
}
