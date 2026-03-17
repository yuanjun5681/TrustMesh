import * as React from 'react'
import { cn } from '@/lib/utils'

interface TabsContextValue {
  value: string
  onValueChange: (value: string) => void
}

const TabsContext = React.createContext<TabsContextValue | null>(null)

interface TabsProps {
  value: string
  onValueChange: (value: string) => void
  children: React.ReactNode
  className?: string
}

function Tabs({ value, onValueChange, children, className }: TabsProps) {
  return (
    <TabsContext.Provider value={{ value, onValueChange }}>
      <div className={className} data-value={value}>
        {children}
      </div>
    </TabsContext.Provider>
  )
}

interface TabsListProps extends React.HTMLAttributes<HTMLDivElement> {
  activeValue?: string
  onValueChange?: (value: string) => void
}

function TabsList({ className, children, activeValue, onValueChange, ...props }: TabsListProps) {
  const context = React.useContext(TabsContext)
  const currentValue = activeValue ?? context?.value
  const handleValueChange = onValueChange ?? context?.onValueChange

  return (
    <div
      className={cn(
        'inline-flex h-9 items-center justify-center rounded-lg bg-muted p-1 text-muted-foreground',
        className
      )}
      {...props}
    >
      {React.Children.map(children, (child) => {
        if (React.isValidElement(child)) {
          return React.cloneElement(child as React.ReactElement<{ activeValue?: string; onValueChange?: (v: string) => void }>, {
            activeValue: currentValue,
            onValueChange: handleValueChange,
          })
        }
        return child
      })}
    </div>
  )
}

interface TabsTriggerProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  value: string
  activeValue?: string
  onValueChange?: (value: string) => void
}

function TabsTrigger({ className, value, activeValue, onValueChange, ...props }: TabsTriggerProps) {
  const context = React.useContext(TabsContext)
  const currentValue = activeValue ?? context?.value
  const handleValueChange = onValueChange ?? context?.onValueChange

  return (
    <button
      className={cn(
        'inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-sm font-medium transition-all cursor-pointer',
        currentValue === value
          ? 'bg-background text-foreground shadow-sm'
          : 'hover:text-foreground/80',
        className
      )}
      onClick={() => handleValueChange?.(value)}
      {...props}
    />
  )
}

interface TabsContentProps extends React.HTMLAttributes<HTMLDivElement> {
  value: string
  activeValue?: string
}

function TabsContent({ className, value, activeValue, ...props }: TabsContentProps) {
  const context = React.useContext(TabsContext)
  const currentValue = activeValue ?? context?.value

  if (value !== currentValue) return null
  return <div className={cn('mt-2', className)} {...props} />
}

export { Tabs, TabsList, TabsTrigger, TabsContent }
