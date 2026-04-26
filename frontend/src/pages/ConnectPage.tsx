import { useEffect, useMemo, useState } from 'react'
import { useSearchParams, useNavigate, Link } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { useAgents } from '@/hooks/useAgents'
import { usePlatformConnections, useUpsertPlatformConnection, useDeletePlatformConnection } from '@/hooks/usePlatformConnections'
import { useAuthStore } from '@/stores/authStore'
import { TrustMeshLogo } from '@/components/shared/TrustMeshLogo'
import { ApiRequestError } from '@/api/client'
import { toast } from 'sonner'
import { CheckCircle2, AlertCircle, ArrowLeft, Link2Off, Unplug } from 'lucide-react'

const PLATFORM_META: Record<string, { label: string; color: string; abbr: string }> = {
  clawhire: { label: 'ClawHire', color: 'bg-amber-500/10 text-amber-600', abbr: 'CH' },
}

function getPlatformMeta(platform: string) {
  return PLATFORM_META[platform] ?? {
    label: platform.charAt(0).toUpperCase() + platform.slice(1),
    color: 'bg-blue-500/10 text-blue-600',
    abbr: platform.slice(0, 2).toUpperCase(),
  }
}

type Stage = 'confirm' | 'success' | 'disconnected'

export function ConnectPage() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)

  const platform = searchParams.get('platform') ?? ''
  const platformNodeId = searchParams.get('platform_node_id') ?? ''
  const remoteUserId = searchParams.get('remote_user_id') ?? ''

  // undefined = user hasn't made an explicit choice yet
  const [selectedPmAgentId, setSelectedPmAgentId] = useState<string | undefined>(undefined)
  const [stage, setStage] = useState<Stage>('confirm')

  const { data: agents } = useAgents()
  const { data: connections } = usePlatformConnections()
  const upsert = useUpsertPlatformConnection()
  const deleteConn = useDeletePlatformConnection()

  const pmAgents = useMemo(
    () => agents?.filter((a) => a.role === 'pm' && !a.archived) ?? [],
    [agents],
  )
  const existingConn = connections?.find(
    (c) => c.platform === platform && c.platform_node_id === platformNodeId,
  )

  // Derive effective pmAgentId: explicit selection → existing connection → first available
  const pmAgentId = selectedPmAgentId ?? existingConn?.pm_agent_id ?? pmAgents[0]?.id ?? ''

  // Redirect to login if not authenticated, preserving the full connect URL
  useEffect(() => {
    if (!isAuthenticated()) {
      const next = encodeURIComponent(window.location.pathname + window.location.search)
      navigate(`/login?next=${next}`, { replace: true })
    }
  }, [isAuthenticated, navigate])

  const paramsMissing = !platform || !platformNodeId || !remoteUserId
  const meta = getPlatformMeta(platform)

  const handleConnect = async () => {
    try {
      await upsert.mutateAsync({ platform, platform_node_id: platformNodeId, remote_user_id: remoteUserId, pm_agent_id: pmAgentId })
      setStage('success')
    } catch (err) {
      const message = err instanceof ApiRequestError ? err.message : '连接失败，请重试'
      toast.error(message)
    }
  }

  const handleDisconnect = async () => {
    if (!existingConn) return
    try {
      await deleteConn.mutateAsync({ platform: existingConn.platform, platformNodeId: existingConn.platform_node_id })
      setStage('disconnected')
    } catch {
      toast.error('断开失败，请重试')
    }
  }

  if (!isAuthenticated()) return null

  return (
    <div className="min-h-screen bg-background flex flex-col items-center justify-center px-4 py-12">
      {/* Header */}
      <div className="mb-8 flex flex-col items-center gap-2">
        <TrustMeshLogo size={40} />
        <span className="text-sm font-semibold text-foreground tracking-wide">TrustMesh</span>
      </div>

      <div className="w-full max-w-md">
        {/* Stage: success */}
        {stage === 'success' && (
          <div className="rounded-xl border bg-card p-8 flex flex-col items-center gap-4 text-center shadow-sm">
            <CheckCircle2 className="size-12 text-emerald-500" />
            <div>
              <h2 className="text-lg font-semibold">连接成功</h2>
              <p className="text-sm text-muted-foreground mt-1">
                {meta.label} 已与 TrustMesh 建立连接，任务将自动同步。
              </p>
            </div>
            <Button className="mt-2 w-full" onClick={() => navigate('/settings')}>
              查看连接设置
            </Button>
          </div>
        )}

        {/* Stage: disconnected */}
        {stage === 'disconnected' && (
          <div className="rounded-xl border bg-card p-8 flex flex-col items-center gap-4 text-center shadow-sm">
            <Unplug className="size-12 text-muted-foreground" />
            <div>
              <h2 className="text-lg font-semibold">连接已断开</h2>
              <p className="text-sm text-muted-foreground mt-1">
                {meta.label} 与 TrustMesh 的连接已成功断开。
              </p>
            </div>
            <Button variant="outline" className="mt-2 w-full" onClick={() => navigate('/settings')}>
              返回设置
            </Button>
          </div>
        )}

        {/* Stage: confirm */}
        {stage === 'confirm' && (
          <div className="rounded-xl border bg-card shadow-sm overflow-hidden">
            {/* Card header */}
            <div className="px-6 pt-6 pb-4">
              <div className="flex items-center gap-3 mb-4">
                <div className={`flex size-10 items-center justify-center rounded-md font-bold text-sm select-none ${meta.color}`}>
                  {meta.abbr}
                </div>
                <div className="flex items-center gap-2 text-muted-foreground text-sm">
                  <span className="font-medium text-foreground">{meta.label}</span>
                  <span>想要连接到</span>
                  <span className="font-medium text-foreground">TrustMesh</span>
                </div>
              </div>

              {paramsMissing ? (
                <div className="flex items-start gap-2 rounded-lg bg-destructive/10 text-destructive px-3 py-2.5 text-sm">
                  <AlertCircle className="size-4 mt-0.5 shrink-0" />
                  <span>链接参数不完整，请从外部平台重新发起连接。</span>
                </div>
              ) : (
                <>
                  <p className="text-sm text-muted-foreground">
                    授权后，{meta.label} 可以向 TrustMesh 发送任务，并接收任务进度更新。
                  </p>

                  {existingConn && (
                    <div className="mt-3 flex items-center gap-1.5 text-xs text-amber-600 bg-amber-500/10 rounded-md px-2.5 py-1.5">
                      <AlertCircle className="size-3.5 shrink-0" />
                      该节点已存在连接，确认后将更新配置。
                    </div>
                  )}
                </>
              )}
            </div>

            {!paramsMissing && (
              <>
                <Separator />

                {/* Connection details */}
                <div className="px-6 py-4 space-y-3">
                  <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">连接信息</h3>
                  <dl className="space-y-2 text-sm">
                    <div className="flex justify-between gap-4">
                      <dt className="text-muted-foreground shrink-0">平台节点 ID</dt>
                      <dd className="font-mono text-xs truncate max-w-[200px]" title={platformNodeId}>{platformNodeId}</dd>
                    </div>
                    <div className="flex justify-between gap-4">
                      <dt className="text-muted-foreground shrink-0">账号 ID</dt>
                      <dd className="font-mono text-xs truncate max-w-[200px]" title={remoteUserId}>{remoteUserId}</dd>
                    </div>
                  </dl>
                </div>

                <Separator />

                {/* PM Agent selector */}
                <div className="px-6 py-4 space-y-2">
                  <label className="text-sm font-medium">负责 PM Agent</label>
                  <Select value={pmAgentId} onValueChange={(v) => v && setSelectedPmAgentId(v)}>
                    <SelectTrigger className="w-full">
                      <SelectValue placeholder="选择 PM Agent" />
                    </SelectTrigger>
                    <SelectContent>
                      {pmAgents.length === 0 && (
                        <SelectItem value="__none__" disabled>暂无可用 PM Agent</SelectItem>
                      )}
                      {pmAgents.map((a) => (
                        <SelectItem key={a.id} value={a.id}>
                          <span>{a.name}</span>
                          <Badge variant="secondary" className="ml-2 text-xs">{a.status}</Badge>
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <p className="text-xs text-muted-foreground">收到来自 {meta.label} 的任务时，由此 Agent 负责规划执行。</p>
                </div>

                <Separator />

                {/* Actions */}
                <div className="px-6 py-4 flex flex-col gap-2">
                  <Button
                    className="w-full"
                    disabled={!pmAgentId || pmAgents.length === 0 || upsert.isPending}
                    onClick={handleConnect}
                  >
                    {upsert.isPending ? '连接中…' : existingConn ? '更新连接' : '授权连接'}
                  </Button>

                  {existingConn && (
                    <Button
                      variant="outline"
                      className="w-full text-destructive hover:text-destructive"
                      disabled={deleteConn.isPending}
                      onClick={handleDisconnect}
                    >
                      <Link2Off className="size-4 mr-1.5" />
                      {deleteConn.isPending ? '断开中…' : '断开连接'}
                    </Button>
                  )}

                  <Button
                    variant="ghost"
                    size="sm"
                    className="w-full text-muted-foreground"
                    onClick={() => navigate('/settings')}
                  >
                    <ArrowLeft className="size-3.5 mr-1" />
                    取消，返回设置
                  </Button>
                </div>
              </>
            )}

            {paramsMissing && (
              <div className="px-6 pb-6">
                <Button variant="outline" size="sm" className="w-full mt-2" onClick={() => navigate('/settings')}>
                  返回设置
                </Button>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Footer */}
      <p className="mt-8 text-xs text-muted-foreground">
        <Link to="/settings" className="hover:underline">TrustMesh 设置</Link>
        {' · '}
        <Link to="/dashboard" className="hover:underline">返回工作台</Link>
      </p>
    </div>
  )
}
