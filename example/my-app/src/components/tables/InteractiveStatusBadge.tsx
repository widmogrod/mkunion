import React, { useState } from 'react'
import { Plus } from 'lucide-react'
import { Badge } from '../ui/badge'
import { cn } from '../../lib/utils'
import * as workflow from '../../workflow/github_com_widmogrod_mkunion_x_workflow'
import { assertNever } from '../../utils/type-helpers'

interface InteractiveStatusBadgeProps {
  state: workflow.State
  onAddFilter?: (stateType: string) => void
  isFilterActive?: boolean
}

export function InteractiveStatusBadge({ state, onAddFilter, isFilterActive }: InteractiveStatusBadgeProps) {
  const [isHovered, setIsHovered] = useState(false)
  
  // Extract the type for cleaner switch statement
  const stateType = state.$type
  
  // Handle the case where $type is undefined
  if (!stateType) {
    console.error('InteractiveStatusBadge: state.$type is undefined', state)
    return <Badge variant="secondary">Unknown</Badge>
  }
  
  const getBadgeProps = () => {
    switch (stateType) {
      case 'workflow.Done':
        return { label: 'Done', className: 'bg-green-500', color: '#10b981' }
      
      case 'workflow.Error':
        return { label: 'Error', className: 'bg-red-500', color: '#ef4444' }
      
      case 'workflow.Await':
        return { label: 'Await', className: 'bg-blue-500', color: '#3b82f6' }
      
      case 'workflow.Scheduled':
        return { label: 'Scheduled', className: 'bg-yellow-500', color: '#eab308' }
      
      case 'workflow.ScheduleStopped':
        return { label: 'Stopped', className: 'bg-gray-500', color: '#6b7280' }
      
      case 'workflow.NextOperation':
        return { label: 'Next', className: 'bg-purple-500', color: '#a855f7' }
      
      default:
        return assertNever(stateType)
    }
  }
  
  const { label, className } = getBadgeProps()
  const isInteractive = onAddFilter && !isFilterActive
  
  return (
    <div 
      className={cn("inline-flex items-center", isInteractive && "group")}
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
    >
      <Badge 
        className={cn(
          className,
          "transition-all duration-200 relative overflow-hidden",
          isInteractive && "cursor-pointer group-hover:pr-7",
          isInteractive && isHovered && "shadow-md"
        )}
        onClick={isInteractive ? () => onAddFilter(stateType) : undefined}
      >
        <span className="relative z-10">{label}</span>
        {isInteractive && (
          <Plus 
            className={cn(
              "ml-1 h-3 w-3 transition-all duration-200 absolute right-1.5 top-1/2 -translate-y-1/2",
              isHovered ? "opacity-100 translate-x-0" : "opacity-0 translate-x-2"
            )}
          />
        )}
      </Badge>
      {isFilterActive && (
        <span className="ml-1 text-xs text-muted-foreground">
          (filtered)
        </span>
      )}
    </div>
  )
}