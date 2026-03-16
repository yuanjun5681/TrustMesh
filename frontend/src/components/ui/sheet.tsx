import * as React from 'react'
import { cn } from '@/lib/utils'
import { X } from 'lucide-react'

interface SheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  children: React.ReactNode
}

function Sheet({ open, onOpenChange, children }: SheetProps) {
  if (!open) return null
  return (
    <div className="fixed inset-0 z-50">
      <div className="fixed inset-0 bg-black/30 backdrop-blur-sm" onClick={() => onOpenChange(false)} />
      {children}
    </div>
  )
}

const SheetContent = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement> & { onClose?: () => void }>(
  ({ className, children, onClose, ...props }, ref) => (
    <div
      ref={ref}
      className={cn(
        'fixed inset-y-0 right-0 z-50 w-full max-w-xl border-l bg-background shadow-xl transition-transform duration-300 ease-in-out',
        className
      )}
      {...props}
    >
      <div className="flex h-full flex-col overflow-y-auto">
        {children}
      </div>
      {onClose && (
        <button
          onClick={onClose}
          className="absolute right-4 top-4 rounded-sm opacity-70 transition-opacity hover:opacity-100 cursor-pointer"
        >
          <X className="h-4 w-4" />
        </button>
      )}
    </div>
  )
)
SheetContent.displayName = 'SheetContent'

function SheetHeader({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('flex flex-col space-y-2 p-6 pb-0', className)} {...props} />
}

function SheetTitle({ className, ...props }: React.HTMLAttributes<HTMLHeadingElement>) {
  return <h2 className={cn('text-lg font-semibold', className)} {...props} />
}

export { Sheet, SheetContent, SheetHeader, SheetTitle }
