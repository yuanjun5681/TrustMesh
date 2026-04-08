import { useState } from 'react'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'

interface InstallGuideProps {
  roleId: string
}

export function InstallGuide({ roleId }: InstallGuideProps) {
  const [open, setOpen] = useState(false)

  return (
    <div className="rounded-lg border">
      <button
        onClick={() => setOpen(!open)}
        className="flex w-full items-center justify-between px-4 py-3 text-sm font-medium hover:bg-muted/50 transition-colors"
      >
        <span>安装说明</span>
        {open ? (
          <ChevronDown className="size-4 text-muted-foreground" />
        ) : (
          <ChevronRight className="size-4 text-muted-foreground" />
        )}
      </button>
      <div className={cn('border-t px-4 py-3 text-sm text-muted-foreground space-y-3', !open && 'hidden')}>
        <ol className="list-decimal list-inside space-y-2">
          <li>下载角色包（<code className="text-xs bg-muted px-1 py-0.5 rounded">{roleId}.zip</code>）</li>
          <li>
            解压后，将角色目录复制到 OpenClaw 的 agents 目录：
            <pre className="mt-1.5 rounded bg-muted px-3 py-2 text-xs font-mono overflow-x-auto">
              {`cp -r ${roleId} ~/.openclaw/agents/`}
            </pre>
          </li>
          <li>重启 OpenClaw 网关，角色将自动激活</li>
          <li>在 TrustMesh 项目团队中将该智能体添加为成员，即可参与协作</li>
        </ol>
        <p className="text-xs">
          需要安装 <a href="https://docs.openclaw.ai" target="_blank" rel="noreferrer" className="text-primary hover:underline">OpenClaw</a> 并通过 Clawsynapse 连接到 TrustMesh。
        </p>
      </div>
    </div>
  )
}
