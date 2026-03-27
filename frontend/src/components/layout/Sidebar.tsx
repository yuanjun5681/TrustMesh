import { Link, useLocation, useNavigate, useParams } from 'react-router-dom'
import {
  FolderKanban,
  Bot,
  Plus,
  ChevronLeft,
  Sun,
  Moon,
  LogOut,
  LayoutDashboard,
  Inbox,
  BookOpen,
  Loader2,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { Avatar } from '@/components/ui/avatar'
import { AgentStatusIcon, ProjectWorkStatusDot } from '@/components/shared/StatusBadge'
import { TrustMeshLogo } from '@/components/shared/TrustMeshLogo'
import { AgentConfigDialog } from '@/components/agent/AgentConfigDialog'
import { useProjects } from '@/hooks/useProjects'
import { useAgents } from '@/hooks/useAgents'
import { useUnreadCount } from '@/hooks/useNotifications'
import { useAuthStore } from '@/stores/authStore'
import { useThemeStore } from '@/stores/themeStore'
import { useState, useEffect } from 'react'
import type { ProjectWorkStatus } from '@/types'

interface SidebarProps {
  onCreateProject: () => void
}

function ProjectSidebarStatus({ status }: { status: ProjectWorkStatus }) {
  if (status === 'running') {
    return (
      <Badge variant="info" className="ml-auto gap-1.5">
        <Loader2 className="size-3 animate-spin" />
        执行中
      </Badge>
    )
  }

  if (status === 'attention') {
    return (
      <Badge variant="destructive" className="ml-auto">
        需关注
      </Badge>
    )
  }

  if (status === 'queued') {
    return (
      <Badge variant="warning" className="ml-auto">
        待处理
      </Badge>
    )
  }

  return (
    <span className="ml-auto flex items-center gap-1.5 shrink-0 text-xs text-muted-foreground">
      <ProjectWorkStatusDot status={status} />
    </span>
  )
}

export function Sidebar({ onCreateProject }: SidebarProps) {
  const location = useLocation()
  const navigate = useNavigate()
  const { projectId } = useParams()
  const { data: projects } = useProjects()
  const { data: agents } = useAgents()
  const { data: unreadCount } = useUnreadCount()
  const user = useAuthStore((s) => s.user)
  const logout = useAuthStore((s) => s.logout)
  const { setTheme, resolvedTheme } = useThemeStore()
  const isDark = resolvedTheme() === 'dark'
  const [collapsed, setCollapsed] = useState(() => window.matchMedia('(max-width: 1280px)').matches)
  const [addAgentOpen, setAddAgentOpen] = useState(false)

  useEffect(() => {
    const mql = window.matchMedia('(max-width: 1280px)')
    const handler = (e: MediaQueryListEvent) => setCollapsed(e.matches)
    mql.addEventListener('change', handler)
    return () => mql.removeEventListener('change', handler)
  }, [])

  const toggleTheme = () => {
    setTheme(isDark ? 'light' : 'dark')
  }

  const isActive = (path: string) => location.pathname === path

  return (
    <aside
      className={cn(
        'flex h-full flex-col border-r bg-sidebar text-sidebar-foreground transition-all duration-200',
        collapsed ? 'w-16' : 'w-64'
      )}
    >
      {/* Header */}
      <div className="flex h-14 items-center gap-2 px-4">
        {!collapsed && (
          <Link to="/dashboard" className="flex items-center gap-2 font-semibold text-lg">
            <TrustMeshLogo size={28} />
            智能体协作平台
          </Link>
        )}
        <Button
          variant="ghost"
          size="icon"
          className={cn('ml-auto size-7', collapsed && 'mx-auto rotate-180')}
          onClick={() => setCollapsed(!collapsed)}
        >
          <ChevronLeft className="size-4" />
        </Button>
      </div>

      <Separator />

      {/* Navigation */}
      <ScrollArea className="flex-1 py-2">
        {/* Dashboard + Inbox */}
        <div className="flex flex-col gap-0.5 px-2">
          <Link
            to="/dashboard"
            className={cn(
              'flex items-center gap-2 rounded-lg px-2 py-2 text-sm transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
              isActive('/dashboard') && 'bg-sidebar-accent text-sidebar-accent-foreground font-medium'
            )}
          >
            <LayoutDashboard className="size-4 shrink-0" />
            {!collapsed && <span>仪表盘</span>}
          </Link>

          <Link
            to="/inbox"
            className={cn(
              'relative flex items-center gap-2 rounded-lg px-2 py-2 text-sm transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
              isActive('/inbox') && 'bg-sidebar-accent text-sidebar-accent-foreground font-medium'
            )}
          >
            <Inbox className="size-4 shrink-0" />
            {!collapsed && (
              <>
                <span>收件箱</span>
                {unreadCount != null && unreadCount > 0 && (
                  <span className="ml-auto text-xs bg-primary text-primary-foreground rounded-full px-1.5 py-0.5 min-w-[20px] text-center">
                    {unreadCount > 99 ? '99+' : unreadCount}
                  </span>
                )}
              </>
            )}
            {collapsed && unreadCount != null && unreadCount > 0 && (
              <span className="absolute right-1 top-0.5 size-2 rounded-full bg-primary" />
            )}
          </Link>

          <Link
            to="/knowledge"
            className={cn(
              'flex items-center gap-2 rounded-lg px-2 py-2 text-sm transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
              isActive('/knowledge') && 'bg-sidebar-accent text-sidebar-accent-foreground font-medium'
            )}
          >
            <BookOpen className="size-4 shrink-0" />
            {!collapsed && <span>知识库</span>}
          </Link>
        </div>

        <Separator className="my-2" />

        {/* Projects */}
        <div className="px-2">
          {!collapsed && (
            <div className="mb-1 flex items-center justify-between px-2">
              <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                项目
              </span>
              <Button variant="ghost" size="icon" className="size-6" onClick={onCreateProject}>
                <Plus className="size-3.5" />
              </Button>
            </div>
          )}

          {collapsed && (
            <Button variant="ghost" size="icon" className="w-full mb-1" onClick={onCreateProject}>
              <Plus className="size-4" />
            </Button>
          )}

          {projects?.filter(p => p.status === 'active').map((project) => (
            <Link
              key={project.id}
              to={`/projects/${project.id}`}
              className={cn(
                'flex items-center gap-2 rounded-lg px-2 py-2 text-sm transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
                projectId === project.id && 'bg-sidebar-accent text-sidebar-accent-foreground font-medium'
              )}
            >
              <FolderKanban className="size-4 shrink-0" />
              {!collapsed && (
                <>
                  <span className="truncate">{project.name}</span>
                  <ProjectSidebarStatus status={project.task_summary.work_status} />
                </>
              )}
            </Link>
          ))}
        </div>

        <Separator className="my-2" />

        {/* Agents */}
        <div className="px-2">
          {!collapsed && (
            <div className="mb-1 px-2">
              <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                智能体
              </span>
            </div>
          )}

          {agents?.map((agent) => (
            <Link
              key={agent.id}
              to={`/agents/${agent.id}`}
              className={cn(
                'flex items-center gap-2 rounded-lg px-2 py-2 text-sm transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
                location.pathname === `/agents/${agent.id}` && 'bg-sidebar-accent text-sidebar-accent-foreground font-medium'
              )}
            >
              <Bot className="size-4 shrink-0" />
              {!collapsed && (
                <>
                  <span className="truncate">{agent.name}</span>
                  <span className="ml-auto"><AgentStatusIcon status={agent.status} className="size-3.5" /></span>
                </>
              )}
            </Link>
          ))}

          <button
            onClick={() => setAddAgentOpen(true)}
            className="flex w-full items-center gap-2 rounded-lg px-2 py-2 text-sm text-muted-foreground transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
          >
            <Plus className="size-4 shrink-0" />
            {!collapsed && <span>添加智能体</span>}
          </button>
          <AgentConfigDialog
            open={addAgentOpen}
            onOpenChange={setAddAgentOpen}
            onCreated={(id) => navigate(`/agents/${id}`)}
          />
        </div>
      </ScrollArea>

      <Separator />

      {/* Footer */}
      <div className="p-2">
        <div className={cn('flex items-center gap-2', collapsed ? 'flex-col' : 'px-2')}>
          {!collapsed && user && (
            <div className="flex items-center gap-2 flex-1 min-w-0">
              <Avatar fallback={user.name} size="sm" />
              <span className="truncate text-sm">{user.name}</span>
            </div>
          )}
          <Button variant="ghost" size="icon" className="size-7 shrink-0" onClick={toggleTheme}>
            {isDark ? <Sun className="size-4" /> : <Moon className="size-4" />}
          </Button>
          <Button variant="ghost" size="icon" className="size-7 shrink-0" onClick={logout}>
            <LogOut className="size-4" />
          </Button>
        </div>
      </div>
    </aside>
  )
}
