import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { MessageSquare, MoreHorizontal, Pencil, Archive, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger, DropdownMenuSeparator } from '@/components/ui/dropdown-menu'
import { AgentStatusDot } from '@/components/shared/StatusBadge'
import { TaskListView } from '@/components/task/TaskListView'
import { TaskDetailPanel } from '@/components/task/TaskDetailPanel'
import { useProject } from '@/hooks/useProjects'
import { useTasks } from '@/hooks/useTasks'

export function ProjectBoardPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const { data: project } = useProject(projectId)
  const { data: tasks, isLoading } = useTasks(projectId)
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null)

  return (
    <div className="flex h-full flex-col">
      {/* Project Header */}
      <div className="flex items-center justify-between border-b px-6 py-3">
        <div className="flex items-center gap-3 min-w-0">
          <div>
            <h1 className="text-lg font-semibold truncate">{project?.name ?? '...'}</h1>
            {project && (
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <AgentStatusDot status={project.pm_agent.status} />
                <span>PM: {project.pm_agent.name}</span>
              </div>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2">
          {projectId && (
            <Link to={`/projects/${projectId}/chat`}>
              <Button variant="outline" size="sm">
                <MessageSquare className="h-4 w-4 mr-1.5" />
                对话
              </Button>
            </Link>
          )}
          <DropdownMenu>
            <DropdownMenuTrigger className="p-2 rounded-md hover:bg-muted">
              <MoreHorizontal className="h-4 w-4" />
            </DropdownMenuTrigger>
            <DropdownMenuContent>
              <DropdownMenuItem>
                <Pencil className="h-3.5 w-3.5 mr-2" />
                编辑项目
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem className="text-destructive">
                <Archive className="h-3.5 w-3.5 mr-2" />
                归档项目
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      {/* Split layout: Task List + Detail Panel (Asana style) */}
      {isLoading ? (
        <div className="flex flex-1 items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      ) : (
        <div className="flex flex-1 min-h-0">
          {/* Left: Task List */}
          <div className={`px-6 ${selectedTaskId ? 'w-1/2 border-r' : 'w-full'}`}>
            <TaskListView
              tasks={tasks ?? []}
              selectedTaskId={selectedTaskId}
              onTaskClick={(id) => setSelectedTaskId(id === selectedTaskId ? null : id)}
            />
          </div>

          {/* Right: Detail Panel */}
          {selectedTaskId && (
            <div className="w-1/2">
              <TaskDetailPanel
                key={selectedTaskId}
                taskId={selectedTaskId}
                onClose={() => setSelectedTaskId(null)}
              />
            </div>
          )}
        </div>
      )}
    </div>
  )
}
