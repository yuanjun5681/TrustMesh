import type { ComponentPropsWithoutRef } from 'react'
import { cn } from '@/lib/utils'

type ScrollAreaProps = ComponentPropsWithoutRef<'div'>

function ScrollArea({ className, children, ...props }: ScrollAreaProps) {
  return (
    <div className={cn('overflow-auto', className)} {...props}>
      {children}
    </div>
  )
}

function ScrollBar() {
  return null
}

export { ScrollArea, ScrollBar }
