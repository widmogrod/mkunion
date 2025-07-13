import React from 'react'
import { Badge } from '../ui/badge'
import * as workflow from '../../workflow/github_com_widmogrod_mkunion_x_workflow'
import { assertNever } from '../../utils/type-helpers'

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
      return <Badge className="bg-green-500">Done</Badge>
    
    case 'workflow.Error':
      return <Badge variant="destructive">Error</Badge>
    
    case 'workflow.Await':
      return <Badge className="bg-blue-500">Await</Badge>
    
    case 'workflow.Scheduled':
      return <Badge className="bg-yellow-500">Scheduled</Badge>
    
    case 'workflow.ScheduleStopped':
      return <Badge variant="outline">Stopped</Badge>
    
    case 'workflow.NextOperation':
      return <Badge className="bg-purple-500">Next Operation</Badge>
    
    // This will cause a compile error if a new state type is added
    // and not handled above
    default:
      return assertNever(stateType)
  }
}