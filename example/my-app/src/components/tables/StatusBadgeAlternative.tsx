import React from 'react'
import { Badge } from '../ui/badge'
import * as workflow from '../../workflow/github_com_widmogrod_mkunion_x_workflow'
import { match } from '../../utils/type-helpers'

interface StatusBadgeProps {
  state: workflow.State
}

/**
 * Alternative implementation of StatusBadge using the match utility.
 * This demonstrates a more functional approach to handling discriminated unions.
 * 
 * Benefits:
 * - More concise and declarative
 * - Automatically exhaustive (TypeScript will error if a case is missing)
 * - Each case is a pure function
 */
export function StatusBadgeAlternative({ state }: StatusBadgeProps) {
  return match(state, {
    'workflow.Done': () => <Badge className="bg-green-500">Done</Badge>,
    'workflow.Error': () => <Badge variant="destructive">Error</Badge>,
    'workflow.Await': () => <Badge className="bg-blue-500">Await</Badge>,
    'workflow.Scheduled': () => <Badge className="bg-yellow-500">Scheduled</Badge>,
    'workflow.ScheduleStopped': () => <Badge variant="outline">Stopped</Badge>,
    'workflow.NextOperation': () => <Badge className="bg-purple-500">Next Operation</Badge>,
  })
}