import React from 'react'
import { cn } from '../../lib/utils'

interface SidebarProps extends React.HTMLAttributes<HTMLDivElement> {
  children: React.ReactNode
}

export function Sidebar({ children, className, ...props }: SidebarProps) {
  return (
    <aside
      className={cn(
        "fixed left-0 top-0 z-40 h-screen w-64 -translate-x-full border-r bg-background transition-transform sm:translate-x-0",
        className
      )}
      {...props}
    >
      <div className="flex h-full flex-col overflow-y-auto px-3 py-4">
        {children}
      </div>
    </aside>
  )
}