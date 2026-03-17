import { Link, useLocation, useParams } from 'react-router-dom'
import {
  FolderKanban,
  Bot,
  Plus,
  ChevronLeft,
  Sun,
  Moon,
  LogOut,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { Avatar } from '@/components/ui/avatar'
import { useProjects } from '@/hooks/useProjects'
import { useAuthStore } from '@/stores/authStore'
import { useThemeStore } from '@/stores/themeStore'
import { useState } from 'react'

interface SidebarProps {
  onCreateProject: () => void
}

export function Sidebar({ onCreateProject }: SidebarProps) {
  const location = useLocation()
  const { projectId } = useParams()
  const { data: projects } = useProjects()
  const user = useAuthStore((s) => s.user)
  const logout = useAuthStore((s) => s.logout)
  const { theme, setTheme } = useThemeStore()
  const [collapsed, setCollapsed] = useState(false)

  const toggleTheme = () => {
    if (theme === 'light') setTheme('dark')
    else if (theme === 'dark') setTheme('system')
    else setTheme('light')
  }

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
          <Link to="/projects" className="flex items-center gap-2 font-semibold text-lg">
            <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-primary text-primary-foreground text-xs font-bold">
              T
            </div>
            TrustMesh
          </Link>
        )}
        <Button
          variant="ghost"
          size="icon"
          className={cn('ml-auto h-7 w-7', collapsed && 'mx-auto rotate-180')}
          onClick={() => setCollapsed(!collapsed)}
        >
          <ChevronLeft className="h-4 w-4" />
        </Button>
      </div>

      <Separator />

      {/* Navigation */}
      <ScrollArea className="flex-1 py-2">
        <div className="px-2">
          {!collapsed && (
            <div className="mb-1 flex items-center justify-between px-2">
              <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                项目
              </span>
              <Button variant="ghost" size="icon" className="h-6 w-6" onClick={onCreateProject}>
                <Plus className="h-3.5 w-3.5" />
              </Button>
            </div>
          )}

          {collapsed && (
            <Button variant="ghost" size="icon" className="w-full mb-1" onClick={onCreateProject}>
              <Plus className="h-4 w-4" />
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
              <FolderKanban className="h-4 w-4 shrink-0" />
              {!collapsed && (
                <span className="truncate">{project.name}</span>
              )}
              {!collapsed && (
                <span
                  className={cn(
                    'ml-auto h-2 w-2 rounded-full shrink-0',
                    project.pm_agent.status === 'online' ? 'bg-status-online' :
                    project.pm_agent.status === 'busy' ? 'bg-status-busy' : 'bg-status-offline'
                  )}
                />
              )}
            </Link>
          ))}
        </div>

        <Separator className="my-2" />

        <div className="px-2">
          <Link
            to="/agents"
            className={cn(
              'flex items-center gap-2 rounded-lg px-2 py-2 text-sm transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
              location.pathname === '/agents' && 'bg-sidebar-accent text-sidebar-accent-foreground font-medium'
            )}
          >
            <Bot className="h-4 w-4 shrink-0" />
            {!collapsed && <span>Agent 管理</span>}
          </Link>
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
          <Button variant="ghost" size="icon" className="h-7 w-7 shrink-0" onClick={toggleTheme}>
            {theme === 'dark' ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
          </Button>
          <Button variant="ghost" size="icon" className="h-7 w-7 shrink-0" onClick={logout}>
            <LogOut className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </aside>
  )
}
