import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { MessageSquarePlus, Plus, MoreHorizontal, Pencil, Archive, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger, DropdownMenuSeparator } from '@/components/ui/dropdown-menu'
import { AgentStatusDot, ProjectStatusBadge, ProjectWorkStatusBadge } from '@/components/shared/StatusBadge'
import { TaskListView } from '@/components/task/TaskListView'
import { TaskDetailPanel } from '@/components/task/TaskDetailPanel'
import { ConversationSheet } from '@/components/conversation/ConversationSheet'
import { EditProjectDialog } from '@/components/project/EditProjectDialog'
import { ArchiveProjectDialog } from '@/components/project/ArchiveProjectDialog'
import { CreateTaskDialog } from '@/components/task/CreateTaskDialog'
import { useProject } from '@/hooks/useProjects'
import { useTasks } from '@/hooks/useTasks'
import { formatDateTime, formatRelativeTime } from '@/lib/utils'

export function ProjectBoardPage() {
  const navigate = useNavigate()
  const { projectId } = useParams<{ projectId: string }>()
  const { data: project } = useProject(projectId)
  const { data: tasks, isLoading } = useTasks(projectId)
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null)
  const [sheetOpen, setSheetOpen] = useState(false)
  const [editOpen, setEditOpen] = useState(false)
  const [archiveOpen, setArchiveOpen] = useState(false)
  const [createTaskOpen, setCreateTaskOpen] = useState(false)
  const projectArchived = project?.status === 'archived'

  return (
    <div className="flex h-full flex-col">
      {/* Project Header */}
      <div className="flex items-start justify-between gap-4 border-b px-6 py-4">
        <div className="min-w-0 flex-1">
          <div className="flex min-w-0 items-center gap-3">
            <h1 className="truncate text-lg font-semibold">{project?.name ?? '...'}</h1>
            {project && (
              <>
                <ProjectStatusBadge status={project.status} />
                <ProjectWorkStatusBadge status={project.task_summary.work_status} />
              </>
            )}
          </div>
          {project?.description && (
            <p className="mt-1 line-clamp-2 text-sm text-muted-foreground">
              {project.description}
            </p>
          )}
          {project && (
            <div className="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
              <div className="flex items-center gap-1.5">
                <AgentStatusDot status={project.pm_agent.status} />
                <span>PM: {project.pm_agent.name}</span>
              </div>
              <span>
                任务: {project.task_summary.task_total}
                {project.task_summary.in_progress_count > 0 && ` · ${project.task_summary.in_progress_count} 执行中`}
                {project.task_summary.pending_count > 0 && ` · ${project.task_summary.pending_count} 待处理`}
                {project.task_summary.failed_count > 0 && ` · ${project.task_summary.failed_count} 失败`}
                {project.task_summary.canceled_count > 0 && ` · ${project.task_summary.canceled_count} 已取消`}
              </span>
              <span title={formatDateTime(project.updated_at)}>
                最后更新: {formatRelativeTime(project.updated_at)}
              </span>
              {project.task_summary.latest_task_at && (
                <span title={formatDateTime(project.task_summary.latest_task_at)}>
                  最近任务: {formatRelativeTime(project.task_summary.latest_task_at)}
                </span>
              )}
              <span title={formatDateTime(project.created_at)}>
                创建于: {formatDateTime(project.created_at)}
              </span>
            </div>
          )}
          </div>
        <div className="flex items-center gap-2">
          <Button size="sm" variant="outline" disabled={projectArchived} onClick={() => setCreateTaskOpen(true)}>
            <Plus className="size-4 mr-1.5" />
            创建任务
          </Button>
          <Button size="sm" disabled={projectArchived} onClick={() => setSheetOpen(true)}>
            <MessageSquarePlus className="size-4 mr-1.5" />
            {projectArchived ? '项目已归档' : '提交新需求'}
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger className="p-2 rounded-md hover:bg-muted">
              <MoreHorizontal className="size-4" />
            </DropdownMenuTrigger>
            <DropdownMenuContent>
              <DropdownMenuItem disabled={!project} onClick={() => setEditOpen(true)}>
                <Pencil className="size-3.5 mr-2" />
                编辑项目
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                className="text-destructive"
                disabled={!project || projectArchived}
                onClick={() => setArchiveOpen(true)}
              >
                <Archive className="size-3.5 mr-2" />
                归档项目
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      {/* Split layout: Task List + Detail Panel (Asana style) */}
      {isLoading ? (
        <div className="flex flex-1 items-center justify-center">
          <Loader2 className="size-8 animate-spin text-muted-foreground" />
        </div>
      ) : (
        <div className="flex flex-1 min-h-0">
          {/* Left: Task List */}
          <div className={`px-6 h-full ${selectedTaskId ? 'w-1/2 border-r' : 'w-full'}`}>
            <TaskListView
              tasks={tasks ?? []}
              selectedTaskId={selectedTaskId}
              onTaskClick={(id) => setSelectedTaskId(id === selectedTaskId ? null : id)}
            />
          </div>

          {/* Right: Detail Panel */}
          {selectedTaskId && (
            <div className="w-1/2 h-full">
              <TaskDetailPanel
                key={selectedTaskId}
                taskId={selectedTaskId}
                onClose={() => setSelectedTaskId(null)}
              />
            </div>
          )}
        </div>
      )}

      {/* Conversation Sheet */}
      {projectId && (
        <ConversationSheet
          projectId={projectId}
          open={sheetOpen}
          onOpenChange={setSheetOpen}
          onTaskCreated={(taskId) => {
            setSheetOpen(false)
            setSelectedTaskId(taskId)
          }}
        />
      )}

      {projectId && (
        <CreateTaskDialog
          open={createTaskOpen}
          onOpenChange={setCreateTaskOpen}
          projectId={projectId}
          onCreated={(taskId) => setSelectedTaskId(taskId)}
        />
      )}
      <EditProjectDialog open={editOpen} onOpenChange={setEditOpen} project={project} />
      <ArchiveProjectDialog
        open={archiveOpen}
        onOpenChange={setArchiveOpen}
        project={project}
        onArchived={() => navigate('/projects', { replace: true })}
      />
    </div>
  )
}
