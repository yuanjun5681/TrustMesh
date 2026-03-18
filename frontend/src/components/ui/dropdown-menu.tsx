import * as React from 'react'
import { cn } from '@/lib/utils'

interface DropdownMenuProps {
  children: React.ReactNode
}

interface DropdownContextValue {
  open: boolean
  setOpen: (open: boolean) => void
}

const DropdownContext = React.createContext<DropdownContextValue>({ open: false, setOpen: () => {} })

function DropdownMenu({ children }: DropdownMenuProps) {
  const [open, setOpen] = React.useState(false)

  React.useEffect(() => {
    if (!open) return
    const handler = (e: MouseEvent) => {
      const target = e.target as HTMLElement
      if (!target.closest('[data-dropdown]')) setOpen(false)
    }
    document.addEventListener('click', handler)
    return () => document.removeEventListener('click', handler)
  }, [open])

  return (
    <DropdownContext.Provider value={{ open, setOpen }}>
      <div className="relative inline-block" data-dropdown>
        {children}
      </div>
    </DropdownContext.Provider>
  )
}

function DropdownMenuTrigger({ children, className, ...props }: React.ButtonHTMLAttributes<HTMLButtonElement>) {
  const { open, setOpen } = React.useContext(DropdownContext)
  return (
    <button
      className={cn('cursor-pointer', className)}
      onClick={() => setOpen(!open)}
      {...props}
    >
      {children}
    </button>
  )
}

function DropdownMenuContent({ children, className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  const { open } = React.useContext(DropdownContext)
  if (!open) return null
  return (
    <div
      className={cn(
        'absolute right-0 top-full z-50 mt-1 min-w-[8rem] overflow-hidden rounded-md border bg-popover p-1 text-popover-foreground shadow-md',
        className
      )}
      {...props}
    >
      {children}
    </div>
  )
}

function DropdownMenuItem({ className, ...props }: React.ButtonHTMLAttributes<HTMLButtonElement>) {
  const { setOpen } = React.useContext(DropdownContext)
  const { disabled, onClick, ...rest } = props
  return (
    <button
      className={cn(
        'relative flex w-full select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none transition-colors',
        disabled
          ? 'cursor-not-allowed opacity-50'
          : 'cursor-pointer hover:bg-accent hover:text-accent-foreground',
        className
      )}
      disabled={disabled}
      onClick={(e) => {
        if (disabled) {
          return
        }
        onClick?.(e)
        setOpen(false)
      }}
      {...rest}
    />
  )
}

function DropdownMenuSeparator({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('-mx-1 my-1 h-px bg-border', className)} {...props} />
}

export { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator }
