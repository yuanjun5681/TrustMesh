import { useMemo, useState } from 'react'
import { toast } from 'sonner'
import { ApiRequestError } from '@/api/client'
import { ProjectDialog } from '@/components/project/ProjectDialog'
import { useUpdateProject } from '@/hooks/useProjects'
import type { Project } from '@/types'

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  project: Project | undefined
}

export function EditProjectDialog({ open, onOpenChange, project }: Props) {
  const [error, setError] = useState('')
  const updateProject = useUpdateProject()

  const initialValues = useMemo(
    () => ({
      name: project?.name ?? '',
      description: project?.description ?? '',
    }),
    [project?.description, project?.name],
  )

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen) {
      setError('')
    }
    onOpenChange(nextOpen)
  }

  const handleSubmit = async ({ name, description }: { name: string; description: string }) => {
    if (!project) {
      return
    }

    setError('')
    try {
      await updateProject.mutateAsync({
        id: project.id,
        input: { name, description },
      })
      toast.success('项目已更新')
      handleOpenChange(false)
    } catch (err) {
      const message = err instanceof ApiRequestError ? err.message : '更新失败'
      toast.error(message)
      setError(message)
    }
  }

  return (
    <ProjectDialog
      key={`${project?.id ?? 'project'}-${open ? 'open' : 'closed'}`}
      open={open}
      onOpenChange={handleOpenChange}
      title="编辑项目"
      description="更新项目名称和描述"
      submitLabel="保存修改"
      pending={updateProject.isPending}
      error={error}
      submitDisabled={!project}
      initialName={initialValues.name}
      initialDescription={initialValues.description}
      onSubmit={handleSubmit}
    />
  )
}
