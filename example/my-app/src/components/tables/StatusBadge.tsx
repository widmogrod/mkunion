import React from 'react'
import { Badge } from '../ui/badge'
import * as workflow from '../../workflow/github_com_widmogrod_mkunion_x_workflow'
import { assertNever } from '../../utils/type-helpers'
import { StatusIcon } from '../ui/icons'
import { STATUS_COLORS, SPACING } from '../../design-system/constants'

interface StatusBadgeProps {
  state: workflow.State
}

export function StatusBadge({ state }: StatusBadgeProps) {
  // Extract the type for cleaner switch statement
  const stateType = state.$type
  
  // Handle the case where $type is undefined
  if (!stateType) {
    console.error('StatusBadge: state.$type is undefined', state)
    return <Badge variant="secondary">Unknown</Badge>
  }
  
  switch (stateType) {
    case 'workflow.Done':
      return (
        <Badge className={`${STATUS_COLORS.success.bg} ${STATUS_COLORS.success.border} ${STATUS_COLORS.success.text} border ${SPACING.xs}`}>
          <StatusIcon status="done" size="xs" />
          Done
        </Badge>
      )
    
    case 'workflow.Error':
      return (
        <Badge className={`${STATUS_COLORS.error.bg} ${STATUS_COLORS.error.border} ${STATUS_COLORS.error.text} border ${SPACING.xs}`}>
          <StatusIcon status="error" size="xs" />
          Error
        </Badge>
      )
    
    case 'workflow.Await':
      return (
        <Badge className={`${STATUS_COLORS.info.bg} ${STATUS_COLORS.info.border} ${STATUS_COLORS.info.text} border ${SPACING.xs}`}>
          <StatusIcon status="info" size="xs" />
          Await
        </Badge>
      )
    
    case 'workflow.Scheduled':
      return (
        <Badge className={`${STATUS_COLORS.warning.bg} ${STATUS_COLORS.warning.border} ${STATUS_COLORS.warning.text} border ${SPACING.xs}`}>
          <StatusIcon status="scheduled" size="xs" />
          Scheduled
        </Badge>
      )
    
    case 'workflow.ScheduleStopped':
      return (
        <Badge className={`${STATUS_COLORS.neutral.bg} ${STATUS_COLORS.neutral.border} ${STATUS_COLORS.neutral.text} border ${SPACING.xs}`}>
          <StatusIcon status="paused" size="xs" />
          Paused
        </Badge>
      )
    
    case 'workflow.NextOperation':
      return (
        <Badge className={`bg-purple-50 dark:bg-purple-900/20 border-purple-200 dark:border-purple-800 text-purple-700 dark:text-purple-300 border ${SPACING.xs}`}>
          <StatusIcon status="info" size="xs" />
          Next Operation
        </Badge>
      )
    
    // This will cause a compile error if a new state type is added
    // and not handled above
    default:
      return assertNever(stateType)
  }
}