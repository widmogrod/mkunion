import React from 'react'
import { LucideIcon } from 'lucide-react'
import { cn } from '../../lib/utils'
import { Breadcrumb } from '../navigation/Breadcrumb'

interface PageHeaderProps {
  icon: LucideIcon
  title: string
  description: string
  actions?: React.ReactNode
  className?: string
}

export function PageHeader({ 
  icon: Icon, 
  title, 
  description, 
  actions, 
  className 
}: PageHeaderProps) {
  return (
    <div className={cn("p-6 border-b", className)}>
      <Breadcrumb />
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Icon className="h-6 w-6 text-primary" />
          <div>
            <h1 className="text-2xl font-bold">{title}</h1>
            <p className="text-sm text-muted-foreground">{description}</p>
          </div>
        </div>
        {actions && (
          <div className="flex items-center gap-2">
            {actions}
          </div>
        )}
      </div>
    </div>
  )
}