import { Outlet } from 'react-router-dom'
import { Sidebar } from './Sidebar'
import { useState } from 'react'
import { CreateProjectDialog } from '@/components/project/CreateProjectDialog'

export function MainLayout() {
  const [showCreateProject, setShowCreateProject] = useState(false)

  return (
    <div className="flex h-screen overflow-hidden">
      <Sidebar onCreateProject={() => setShowCreateProject(true)} />
      <main className="flex-1 overflow-auto bg-background">
        <Outlet />
      </main>
      <CreateProjectDialog
        open={showCreateProject}
        onOpenChange={setShowCreateProject}
      />
    </div>
  )
}
