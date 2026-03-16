import { useState } from 'react'
import { Link } from 'react-router-dom'
import { FolderKanban, Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { AgentStatusDot } from '@/components/shared/StatusBadge'
import { EmptyState } from '@/components/shared/EmptyState'
import { CreateProjectDialog } from '@/components/project/CreateProjectDialog'
import { useProjects } from '@/hooks/useProjects'
import { formatRelativeTime } from '@/lib/utils'

export function ProjectListPage() {
  const { data: projects, isLoading } = useProjects()
  const [showCreate, setShowCreate] = useState(false)
  const activeProjects = projects?.filter((p) => p.status === 'active') ?? []

  return (
    <div className="p-6 lg:p-8 max-w-6xl mx-auto">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">项目</h1>
          <p className="text-muted-foreground mt-1">管理你的 AI Agent 协作项目</p>
        </div>
        <Button onClick={() => setShowCreate(true)}>
          <Plus className="h-4 w-4 mr-2" />
          新建项目
        </Button>
      </div>

      {isLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {[1, 2, 3].map((i) => (
            <Card key={i} className="animate-pulse">
              <CardHeader>
                <div className="h-5 bg-muted rounded w-32" />
                <div className="h-4 bg-muted rounded w-48 mt-2" />
              </CardHeader>
              <CardContent>
                <div className="h-4 bg-muted rounded w-24" />
              </CardContent>
            </Card>
          ))}
        </div>
      ) : activeProjects.length === 0 ? (
        <EmptyState
          icon={FolderKanban}
          title="还没有项目"
          description="创建你的第一个项目，开始用 AI Agent 协作完成任务"
          action={
            <Button onClick={() => setShowCreate(true)}>
              <Plus className="h-4 w-4 mr-2" />
              创建项目
            </Button>
          }
        />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {activeProjects.map((project) => (
            <Link key={project.id} to={`/projects/${project.id}`}>
              <Card className="transition-all hover:shadow-md hover:border-primary/30 cursor-pointer group">
                <CardHeader>
                  <CardTitle className="flex items-center gap-2 text-base group-hover:text-primary transition-colors">
                    <FolderKanban className="h-4 w-4 shrink-0" />
                    {project.name}
                  </CardTitle>
                  <CardDescription className="line-clamp-2">
                    {project.description}
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="flex items-center justify-between text-xs text-muted-foreground">
                    <div className="flex items-center gap-1.5">
                      <AgentStatusDot status={project.pm_agent.status} />
                      <span>{project.pm_agent.name}</span>
                    </div>
                    <span>{formatRelativeTime(project.updated_at)}</span>
                  </div>
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>
      )}

      <CreateProjectDialog open={showCreate} onOpenChange={setShowCreate} />
    </div>
  )
}
