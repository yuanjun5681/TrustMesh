import { PageContainer } from '@/components/layout/PageContainer'
import { ClawHireConnectionCard } from '@/components/settings/ClawHireConnectionCard'
import { usePlatformConnections } from '@/hooks/usePlatformConnections'
import { Skeleton } from '@/components/ui/skeleton'

export function SettingsPage() {
  const { data: connections, isLoading } = usePlatformConnections()
  const clawhireConn = connections?.find((c) => c.platform === 'clawhire')

  return (
    <PageContainer>
      <div className="max-w-2xl space-y-8">
        <h1 className="text-xl font-semibold">设置</h1>

        <section>
          <div className="mb-4">
            <h2 className="text-base font-semibold">平台集成</h2>
            <p className="text-sm text-muted-foreground mt-0.5">
              连接外部平台，自动同步任务与进度
            </p>
          </div>

          {isLoading ? (
            <Skeleton className="h-28 w-full rounded-lg" />
          ) : (
            <ClawHireConnectionCard connection={clawhireConn} />
          )}
        </section>
      </div>
    </PageContainer>
  )
}
