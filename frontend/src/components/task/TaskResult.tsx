import { FileText, Link as LinkIcon, FileCode, FileBarChart, type LucideIcon } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import type { TaskResult, TaskArtifact } from '@/types'

const kindIcons: Record<string, LucideIcon> = {
  file: FileText,
  link: LinkIcon,
  log: FileCode,
  report: FileBarChart,
}

function normalizeArtifactKind(kind: string | null | undefined) {
  const normalizedKind = kind?.trim().toLowerCase()
  if (!normalizedKind) {
    return 'unknown'
  }
  return normalizedKind
}

interface TaskResultViewProps {
  result: TaskResult
  artifacts: TaskArtifact[]
}

export function TaskResultView({ result, artifacts }: TaskResultViewProps) {
  const safeArtifacts = artifacts ?? []
  const hasResult = result.summary || result.final_output
  const hasArtifacts = safeArtifacts.length > 0

  if (!hasResult && !hasArtifacts) {
    return <p className="py-8 text-center text-sm text-muted-foreground">任务尚未产出结果</p>
  }

  return (
    <div className="space-y-4">
      {hasResult && (
        <Card>
          <CardContent className="p-4 space-y-2">
            {result.summary && (
              <div>
                <h4 className="text-sm font-medium mb-1">摘要</h4>
                <p className="text-sm text-muted-foreground whitespace-pre-wrap">{result.summary}</p>
              </div>
            )}
            {result.final_output && (
              <div>
                <h4 className="text-sm font-medium mb-1">最终产出</h4>
                <div className="rounded-md bg-muted p-3 text-sm whitespace-pre-wrap font-mono text-xs">
                  {result.final_output}
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {hasArtifacts && (
        <div>
          <h4 className="text-sm font-medium mb-2">交付物</h4>
          <div className="space-y-2">
            {safeArtifacts.map((artifact) => {
              const normalizedKind = normalizeArtifactKind(artifact.kind)
              const Icon = kindIcons[normalizedKind] ?? FileText
              return (
                <Card key={artifact.id}>
                  <CardContent className="flex items-center gap-3 p-3">
                    <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-muted">
                      <Icon className="h-4 w-4 text-muted-foreground" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium truncate">{artifact.title}</p>
                      <p className="text-xs text-muted-foreground truncate">{artifact.uri}</p>
                    </div>
                    <Badge variant="secondary" className="text-xs shrink-0">
                      {normalizedKind}
                    </Badge>
                  </CardContent>
                </Card>
              )
            })}
          </div>
        </div>
      )}
    </div>
  )
}
