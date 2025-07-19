import React, { useState } from 'react'
import { ChevronRight, LucideIcon } from 'lucide-react'
import { Button } from '../ui/button'
import { cn } from '../../utils/cn'

interface CollapsibleSectionProps {
  title: string
  icon?: LucideIcon
  children: React.ReactNode
  defaultOpen?: boolean
  className?: string
}

export function CollapsibleSection({ 
  title, 
  icon: Icon, 
  children, 
  defaultOpen = true,
  className 
}: CollapsibleSectionProps) {
  const [isOpen, setIsOpen] = useState(defaultOpen)
  
  return (
    <div className={cn("space-y-2", className)}>
      <Button
        variant="ghost"
        size="sm"
        onClick={() => setIsOpen(!isOpen)}
        className="w-full justify-between px-0 h-8 hover:bg-transparent group"
      >
        <div className="flex items-center gap-2">
          {Icon && <Icon className="h-3.5 w-3.5 text-muted-foreground transition-colors duration-200 group-hover:text-foreground" />}
          <span className="text-sm font-medium text-muted-foreground transition-colors duration-200 group-hover:text-foreground">{title}</span>
        </div>
        <ChevronRight className={cn(
          "h-3.5 w-3.5 text-muted-foreground transition-all duration-200",
          "group-hover:text-foreground",
          isOpen && "rotate-90"
        )} />
      </Button>
      
      {isOpen && (
        <div className={cn(
          "space-y-1 animate-in slide-in-from-top-1 duration-200",
          "ml-0"
        )}>
          {children}
        </div>
      )}
    </div>
  )
}