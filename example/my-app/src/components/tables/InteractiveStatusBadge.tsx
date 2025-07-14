import React, { useState } from 'react'
import { Badge } from '../ui/badge'
import { cn } from '../../lib/utils'
import * as workflow from '../../workflow/github_com_widmogrod_mkunion_x_workflow'
import { assertNever } from '../../utils/type-helpers'
import { StatusIcon, Icon, ActionIcons } from '../ui/icons'
import { STATUS_COLORS, SPACING, TRANSITIONS } from '../../design-system/constants'

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
        return { 
          label: 'Done', 
          status: 'done' as const,
          colorClasses: `${STATUS_COLORS.success.bg} ${STATUS_COLORS.success.border} ${STATUS_COLORS.success.text} border`
        }
      
      case 'workflow.Error':
        return { 
          label: 'Error', 
          status: 'error' as const,
          colorClasses: `${STATUS_COLORS.error.bg} ${STATUS_COLORS.error.border} ${STATUS_COLORS.error.text} border`
        }
      
      case 'workflow.Await':
        return { 
          label: 'Await', 
          status: 'info' as const,
          colorClasses: `${STATUS_COLORS.info.bg} ${STATUS_COLORS.info.border} ${STATUS_COLORS.info.text} border`
        }
      
      case 'workflow.Scheduled':
        return { 
          label: 'Scheduled', 
          status: 'scheduled' as const,
          colorClasses: `${STATUS_COLORS.warning.bg} ${STATUS_COLORS.warning.border} ${STATUS_COLORS.warning.text} border`
        }
      
      case 'workflow.ScheduleStopped':
        return { 
          label: 'Paused', 
          status: 'paused' as const,
          colorClasses: `${STATUS_COLORS.neutral.bg} ${STATUS_COLORS.neutral.border} ${STATUS_COLORS.neutral.text} border`
        }
      
      case 'workflow.NextOperation':
        return { 
          label: 'Next Operation', 
          status: 'info' as const,
          colorClasses: `bg-purple-50 dark:bg-purple-900/20 border-purple-200 dark:border-purple-800 text-purple-700 dark:text-purple-300 border`
        }
      
      default:
        return assertNever(stateType)
    }
  }
  
  const { label, status, colorClasses } = getBadgeProps()
  const isInteractive = onAddFilter && !isFilterActive
  
  return (
    <div 
      className={cn("inline-flex items-center", isInteractive && "group")}
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
    >
      <Badge 
        className={cn(
          colorClasses,
          TRANSITIONS.normal,
          "relative overflow-hidden",
          SPACING.xs,
          isInteractive && "cursor-pointer group-hover:pr-8 hover:brightness-110",
          isInteractive && isHovered && "shadow-md",
          // Ensure text remains visible on hover
          "hover:filter"
        )}
        onClick={isInteractive ? () => onAddFilter(stateType) : undefined}
      >
        <StatusIcon 
          status={status} 
          size="xs" 
          className="relative z-10"
        />
        <span className="relative z-10">{label}</span>
        {isInteractive && (
          <Icon
            icon={ActionIcons.add}
            size="xs"
            className={cn(
              "ml-1 absolute right-1.5 top-1/2 -translate-y-1/2",
              TRANSITIONS.normal,
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