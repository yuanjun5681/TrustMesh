import { useTaskEvents } from '@/hooks/useTasks'
import { EventTimeline } from '@/components/shared/EventTimeline'

interface TaskTimelineProps {
  taskId: string
}

export function TaskTimeline({ taskId }: TaskTimelineProps) {
  const { data: events, isLoading } = useTaskEvents(taskId)

  return <EventTimeline events={events ?? []} loading={isLoading} />
}
