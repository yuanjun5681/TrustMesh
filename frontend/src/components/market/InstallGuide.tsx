import { useMemo } from 'react'
import { Check, Copy, Terminal } from 'lucide-react'
import { toast } from 'sonner'
import type { MarketRoleDetail } from '@/types'
import { Button } from '@/components/ui/button'
import { useCopyToClipboard } from '@/hooks/useCopyToClipboard'

interface InstallGuideProps {
  role: MarketRoleDetail
}

function buildDownloadUrl(roleId: string) {
  const origin = window.location.origin.replace(/\/$/, '')
  return `${origin}/api/v1/market/roles/${roleId}/download`
}

function buildLocalZipPath(roleId: string) {
  return `~/Downloads/${roleId}.zip`
}

function escapePromptValue(value: string) {
  return value.replace(/`/g, '\\`')
}

function buildOpenClawPrompt(role: MarketRoleDetail) {
  const agentId = role.id
  const displayName = escapePromptValue(role.name)
  const department = escapePromptValue(role.dept_name)
  const description = escapePromptValue(role.description)
  const jsonDisplayName = JSON.stringify(role.name)
  const jsonDepartment = JSON.stringify(role.dept_name)
  const jsonDescription = JSON.stringify(role.description)
  const zipUrl = buildDownloadUrl(role.id)
  const localZipPath = buildLocalZipPath(role.id)
  const workspaceName = `workspace-${agentId}`
  const workspacePath = `~/.openclaw/${workspaceName}`

  return `## 创建新 Agent（Multi-Agent 模式）

我需要你帮我创建一个新的独立 Agent，并把下面这个角色注册到 Openclaw 中。

### Agent 角色信息

- Agent ID: \`${agentId}\`
- Display Name: \`${displayName}\`
- Department: \`${department}\`
- Description: \`${description}\`
- Workspace: \`${workspacePath}\`
- Agent Dir: \`~/.openclaw/agents/${agentId}/agent\`
- Sessions Dir: \`~/.openclaw/agents/${agentId}/sessions\`

### 第一步：下载并解压 Agent 包

请优先使用下面的角色包来源：

- URL: \`${zipUrl}\`
- 本地路径（如果我已经下载完成）: \`${localZipPath}\`

下载后解压到临时目录：

\`\`\`bash
# 下载（如果需要）
curl -L -o /tmp/${agentId}.zip "${zipUrl}"

# 解压到临时目录
mkdir -p /tmp/agent-package
unzip -o /tmp/${agentId}.zip -d /tmp/agent-package

# 查看解压后的文件
ls -la /tmp/agent-package/
\`\`\`

### 第二步：检查压缩包内容

解压后应该包含以下 3 个文件（可能还有子文件夹 memory/ 等）：

\`\`\`
${agentId}/
├── AGENTS.md
├── IDENTITY.md
└── SOUL.md
\`\`\`

如果文件不在根目录，而是直接平铺在 zip 中，请将它们移动到 \`${agentId}/\` 子文件夹下。

### 第三步：创建 Agent 工作区

\`\`\`bash
# 1. 创建工作区目录
mkdir -p ${workspacePath}

# 2. 将解压的文件复制到工作区
cp /tmp/agent-package/AGENTS.md   ${workspacePath}/
cp /tmp/agent-package/IDENTITY.md ${workspacePath}/
cp /tmp/agent-package/SOUL.md     ${workspacePath}/

# 3. 可选：创建 memory 目录
mkdir -p ${workspacePath}/memory

# 4. 可选：初始化 Git（用于备份）
cd ${workspacePath}
git init 2>/dev/null || true
git add AGENTS.md IDENTITY.md SOUL.md memory/
git commit -m "Add ${agentId} workspace" 2>/dev/null || true
\`\`\`

### 第四步：注册 Agent 到配置

打开 \`~/.openclaw/openclaw.json\`，在 \`agents.list\` 中添加新 Agent：

\`\`\`json5
{
  agents: {
    list: [
      // ... 你现有的 agents ...

      {
        id: "${agentId}",
        name: ${jsonDisplayName},
        workspace: "${workspacePath}",
        agentDir: "~/.openclaw/agents/${agentId}/agent",
        role: ${jsonDepartment},
        description: ${jsonDescription},
        // 可选：指定模型
        // model: "anthropic/claude-sonnet-4-6",
        // 可选：工具限制
        // tools: { allow: ["read", "exec"], deny: ["write", "edit"] },
      },
    ],
  },
}
\`\`\`

### 第五步：创建 Agent 状态目录

\`\`\`bash
mkdir -p ~/.openclaw/agents/${agentId}/agent
mkdir -p ~/.openclaw/agents/${agentId}/sessions
\`\`\`

### 第六步：重启 Gateway

\`\`\`bash
openclaw gateway restart
\`\`\`

### 第七步：验证

\`\`\`bash
openclaw agents list --bindings
\`\`\`

### 注意事项

1. AGENTS.md 每次会话都会加载，是最重要的运营文件
2. SOUL.md 定义人格，必须保留
3. IDENTITY.md 定义名字、定位和基础身份，建议保留完整内容
4. 如需长期记忆，在 \`memory/\` 目录下创建 MEMORY.md

如果 URL 无法直接访问，请改用我本地已下载的 zip 文件继续安装。完成后告诉我修改了哪些目录和配置项。`
}

export function InstallGuide({ role }: InstallGuideProps) {
  const { copiedKey, copy } = useCopyToClipboard(2000)

  const prompt = useMemo(() => buildOpenClawPrompt(role), [role])

  async function handleCopyPrompt() {
    const ok = await copy(prompt, role.id)
    if (ok) {
      toast.success('Openclaw 提示词已复制')
    } else {
      toast.error('复制失败，请手动选择复制')
    }
  }

  return (
    <div className="flex h-full min-h-0 flex-col rounded-lg border">
      <div className="flex items-center gap-2 border-b px-4 py-3 text-sm font-medium">
        <span className="flex items-center gap-2">
          <Terminal className="size-4 text-muted-foreground" />
          安装说明
        </span>
      </div>
      <div className="flex min-h-0 flex-1 flex-col px-4 py-4">
        <div className="space-y-2">
          <p className="text-sm text-foreground">复制下面这段提示词到 Openclaw。</p>
          <p className="text-xs text-muted-foreground">
            提示词里已经带上当前角色的 ID、名称、部门、描述和角色包地址，Openclaw 可直接继续安装与注册。
          </p>
        </div>

        <div className="mt-4 flex items-center justify-between gap-3 rounded-md border bg-muted/30 px-3 py-2">
          <div className="min-w-0 text-xs text-muted-foreground">
            当前角色：<span className="font-mono text-foreground">{role.id}</span>
          </div>
          <Button onClick={handleCopyPrompt} size="sm" variant="outline">
            {copiedKey === role.id ? <Check className="size-3.5" /> : <Copy className="size-3.5" />}
            {copiedKey === role.id ? '已复制' : '复制提示词'}
          </Button>
        </div>

        <div className="mt-4 min-h-0 flex-1 overflow-auto rounded-lg bg-muted">
          <pre className="px-3 py-3 text-xs leading-5 text-foreground whitespace-pre-wrap break-words">
            {prompt}
          </pre>
        </div>

        <p className="mt-4 text-xs text-muted-foreground">
          如果你已经通过当前页面下载了角色包，也可以把本地 zip 路径改成实际下载位置后再发给 Openclaw。
        </p>
      </div>
    </div>
  )
}
