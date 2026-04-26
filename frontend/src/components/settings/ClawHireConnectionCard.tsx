import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { ConnectClawHireDialog } from './ConnectClawHireDialog'
import { useDeletePlatformConnection } from '@/hooks/usePlatformConnections'
import { useAgents } from '@/hooks/useAgents'
import { toast } from 'sonner'
import type { PlatformConnection } from '@/types'

interface Props {
  connection?: PlatformConnection
}

function truncate(str: string, tail = 8) {
  return str.length <= tail ? str : '…' + str.slice(-tail)
}

export function ClawHireConnectionCard({ connection }: Props) {
  const [dialogOpen, setDialogOpen] = useState(false)
  const deleteConn = useDeletePlatformConnection()
  const { data: agents } = useAgents()

  const pmAgent = agents?.find((a) => a.id === connection?.pm_agent_id)

  const handleDisconnect = async () => {
    if (!connection) return
    try {
      await deleteConn.mutateAsync({
        platform: connection.platform,
        platformNodeId: connection.platform_node_id,
      })
      toast.success('已断开 ClawHire 连接')
    } catch {
      toast.error('断开失败，请重试')
    }
  }

  return (
    <div className="rounded-lg border bg-card p-5">
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-center gap-3">
          {/* ClawHire brand mark */}
          <div className="flex size-10 items-center justify-center rounded-md bg-amber-500/10 text-amber-600 font-bold text-sm select-none">
            CH
          </div>
          <div>
            <div className="flex items-center gap-2">
              <span className="font-medium">ClawHire</span>
              {connection ? (
                <Badge variant="default" className="bg-emerald-500/15 text-emerald-600 border-emerald-500/20 text-xs">
                  ● 已连接
                </Badge>
              ) : (
                <Badge variant="secondary" className="text-xs">未连接</Badge>
              )}
            </div>
            <p className="text-sm text-muted-foreground mt-0.5">
              接收来自 ClawHire 的承接任务，自动同步进度与提交验收
            </p>
          </div>
        </div>

        <div className="shrink-0">
          {connection ? (
            <Button
              variant="outline"
              size="sm"
              onClick={handleDisconnect}
              disabled={deleteConn.isPending}
            >
              {deleteConn.isPending ? '断开中…' : '断开连接'}
            </Button>
          ) : (
            <Button size="sm" onClick={() => setDialogOpen(true)}>
              连接
            </Button>
          )}
        </div>
      </div>

      {connection && (
        <>
          <Separator className="my-4" />
          <dl className="grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
            <div>
              <dt className="text-muted-foreground">节点 ID</dt>
              <dd className="font-mono">{truncate(connection.platform_node_id)}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">账号 ID</dt>
              <dd className="font-mono">{truncate(connection.remote_user_id)}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">PM Agent</dt>
              <dd>{pmAgent?.name ?? connection.pm_agent_id}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">绑定时间</dt>
              <dd>{new Date(connection.linked_at).toLocaleDateString('zh-CN')}</dd>
            </div>
          </dl>
          <div className="mt-3 flex justify-end">
            <Button variant="ghost" size="sm" onClick={() => setDialogOpen(true)}>
              修改配置
            </Button>
          </div>
        </>
      )}

      <ConnectClawHireDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        existing={connection}
      />
    </div>
  )
}
