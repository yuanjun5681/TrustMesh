import { useState } from 'react'
import { PageContainer } from '@/components/layout/PageContainer'
import { ClawHireConnectionCard } from '@/components/settings/ClawHireConnectionCard'
import { usePlatformConnections } from '@/hooks/usePlatformConnections'
import { Skeleton } from '@/components/ui/skeleton'
import { Button } from '@/components/ui/button'
import { Copy, Check } from 'lucide-react'

function ConnectLinkBox() {
  const [copied, setCopied] = useState(false)
  const base = `${window.location.origin}/connect`
  const example = `${base}?platform=clawhire&platform_node_id=<NODE_ID>&remote_user_id=<USER_ID>`

  const handleCopy = () => {
    navigator.clipboard.writeText(base).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    })
  }

  return (
    <div className="rounded-lg border bg-muted/30 p-4 space-y-2">
      <div className="flex items-center justify-between gap-2">
        <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">连接端点 URL</span>
        <Button variant="ghost" size="sm" className="h-6 px-2 text-xs gap-1" onClick={handleCopy}>
          {copied ? <Check className="size-3" /> : <Copy className="size-3" />}
          {copied ? '已复制' : '复制'}
        </Button>
      </div>
      <code className="block text-xs font-mono text-foreground break-all">{example}</code>
      <p className="text-xs text-muted-foreground">
        外部平台通过上述格式构造链接，用户点击后将跳转至 TrustMesh 授权页完成绑定。
      </p>
    </div>
  )
}

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

        <section>
          <div className="mb-4">
            <h2 className="text-base font-semibold">接入指引</h2>
            <p className="text-sm text-muted-foreground mt-0.5">
              向外部平台提供以下连接端点，无需手动填写参数
            </p>
          </div>
          <ConnectLinkBox />
        </section>
      </div>
    </PageContainer>
  )
}
