import React from 'react'
import { Database, CalendarDays, ExternalLink } from 'lucide-react'
import { useNavigationWithContext } from '../../hooks/useNavigation'
import { cn } from '../../lib/utils'
import { Button } from '../ui/button'
import * as workflow from '../../workflow/github_com_widmogrod_mkunion_x_workflow'
import * as schemaless from '../../workflow/github_com_widmogrod_mkunion_x_storage_schemaless'

interface ClickableStateRowProps {
  state: schemaless.Record<workflow.State>
  flowName?: string
  className?: string
}

export function ClickableStateRow({ state, flowName, className }: ClickableStateRowProps) {
  const { navigateToWorkflows, navigateToSchedules } = useNavigationWithContext()

  if (!state.Data || !state.Data.$type) return null

  const stateData = state.Data[state.Data.$type as keyof typeof state.Data] as any
  const isScheduled = state.Data.$type === 'workflow.Scheduled' || state.Data.$type === 'workflow.ScheduleStopped'
  const parentRunId = stateData?.BaseState?.RunOption?.['workflow.ScheduleRun']?.ParentRunID

  const handleWorkflowClick = (e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (flowName) {
      navigateToWorkflows(flowName, state.ID)
    }
  }

  const handleScheduleClick = (e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (parentRunId) {
      navigateToSchedules({
        parentRunId,
        focus: state.ID
      })
    }
  }

  return (
    <div className={cn("inline-flex items-center gap-1", className)}>
      {/* Navigate to workflow details */}
      {flowName && (
        <Button
          variant="ghost"
          size="sm"
          onClick={handleWorkflowClick}
          className="h-6 w-6 p-0 hover:bg-primary/10"
          title={`View workflow: ${flowName}`}
        >
          <Database className="h-3 w-3" />
        </Button>
      )}

      {/* Navigate to schedule details for scheduled runs */}
      {isScheduled && parentRunId && (
        <Button
          variant="ghost"
          size="sm"
          onClick={handleScheduleClick}
          className="h-6 w-6 p-0 hover:bg-primary/10"
          title="View schedule history"
        >
          <CalendarDays className="h-3 w-3" />
        </Button>
      )}
    </div>
  )
}

// Component for making run IDs clickable
interface ClickableRunIdProps {
  runId: string
  flowName?: string
  className?: string
}

export function ClickableRunId({ runId, flowName, className }: ClickableRunIdProps) {
  const { navigateToExecutions } = useNavigationWithContext()

  const handleClick = (e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    navigateToExecutions({ 
      runId,
      workflow: flowName 
    })
  }

  return (
    <button
      onClick={handleClick}
      className={cn(
        "inline-flex items-center gap-1 font-mono text-xs",
        "text-muted-foreground hover:text-primary",
        "hover:underline decoration-dotted underline-offset-2",
        "transition-colors duration-200",
        "focus:outline-none focus:ring-2 focus:ring-primary/20 focus:ring-offset-1 rounded-sm",
        className
      )}
      title={`View execution details for run: ${runId}`}
    >
      {runId}
      <ExternalLink className="h-3 w-3 opacity-50" />
    </button>
  )
}