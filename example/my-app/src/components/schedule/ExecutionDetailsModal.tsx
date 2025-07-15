import React from 'react'
import ReactDOM from 'react-dom'
import { X, Database, CalendarDays } from 'lucide-react'
import { Button } from '../ui/button'
import { Badge } from '../ui/badge'
import { StatusIcon } from '../ui/icons'
import { STATUS_COLORS, SPACING } from '../../design-system/constants'

interface RunExecution {
  id: string
  startTime: Date
  endTime?: Date
  status: 'scheduled' | 'running' | 'done' | 'error'
  duration?: number
  errorMessage?: string
  inputData?: any
  outputData?: any
}

interface ExecutionDetailsModalProps {
  isOpen: boolean
  onClose: () => void
  execution: RunExecution | null
  onNavigateToWorkflow?: (workflowName: string) => void
  onNavigateToSchedule?: (parentRunId: string) => void
  workflowName?: string
  parentRunId?: string
}

export function ExecutionDetailsModal({ 
  isOpen, 
  onClose, 
  execution,
  onNavigateToWorkflow,
  onNavigateToSchedule,
  workflowName,
  parentRunId
}: ExecutionDetailsModalProps) {
  if (!isOpen || !execution) return null

  const getStatusIcon = () => {
    return (
      <StatusIcon 
        status={execution.status} 
        size="md" 
      />
    )
  }

  const getStatusBadge = () => {
    const statusConfig = {
      done: { colors: STATUS_COLORS.success, label: 'Done' },
      error: { colors: STATUS_COLORS.error, label: 'Error' },
      running: { colors: STATUS_COLORS.info, label: 'Running' },
      scheduled: { colors: STATUS_COLORS.warning, label: 'Scheduled' }
    }
    
    const config = statusConfig[execution.status]
    
    return (
      <Badge className={`${config.colors.bg} ${config.colors.border} ${config.colors.text} border ${SPACING.xs}`}>
        <StatusIcon 
          status={execution.status} 
          size="xs" 
        />
        {config.label}
      </Badge>
    )
  }

  return ReactDOM.createPortal(
    <div className="fixed inset-0 z-[10001] flex items-center justify-center">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} />
      <div className="relative bg-background rounded-lg shadow-2xl w-full max-w-2xl max-h-[80vh] overflow-hidden z-[10002] border m-4">
        
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b bg-muted/30">
          <div className="flex items-center gap-3">
            {getStatusIcon()}
            <div>
              <h2 className="text-lg font-semibold">Execution Details</h2>
              <p className="text-sm text-muted-foreground">Run ID: {execution.id}</p>
            </div>
          </div>
          <Button
            variant="ghost"
            size="icon"
            onClick={onClose}
            className="rounded-full"
          >
            <X className="h-4 w-4" />
          </Button>
        </div>

        {/* Content */}
        <div className="p-6 space-y-6 overflow-auto max-h-[calc(80vh-140px)]">
          {/* Status Section */}
          <div className="space-y-2">
            <h3 className="text-sm font-medium text-muted-foreground">Status</h3>
            <div className="flex items-center gap-2">
              {getStatusBadge()}
              {execution.errorMessage && (
                <span className="text-sm text-red-600 dark:text-red-400">
                  {execution.errorMessage}
                </span>
              )}
            </div>
          </div>

          {/* Timing Section */}
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <h3 className="text-sm font-medium text-muted-foreground">Start Time</h3>
              <p className="text-sm font-mono">
                {execution.startTime.toLocaleString()}
              </p>
            </div>
            <div className="space-y-2">
              <h3 className="text-sm font-medium text-muted-foreground">Duration</h3>
              <p className="text-sm font-mono">
                {execution.duration ? `${Math.round(execution.duration / 1000)}s` : 'N/A'}
              </p>
            </div>
          </div>

          {/* Input Data Section */}
          {execution.inputData && (
            <div className="space-y-2">
              <h3 className="text-sm font-medium text-muted-foreground">Input Data</h3>
              <div className="bg-muted/30 p-4 rounded-lg overflow-auto max-h-40">
                <pre className="text-xs font-mono">
                  {JSON.stringify(execution.inputData, null, 2)}
                </pre>
              </div>
            </div>
          )}

          {/* Output Data Section */}
          {execution.outputData && (
            <div className="space-y-2">
              <h3 className="text-sm font-medium text-muted-foreground">Output Data</h3>
              <div className="bg-muted/30 p-4 rounded-lg overflow-auto max-h-40">
                <pre className="text-xs font-mono">
                  {JSON.stringify(execution.outputData, null, 2)}
                </pre>
              </div>
            </div>
          )}

          {/* Error Details */}
          {execution.status === 'error' && execution.errorMessage && (
            <div className="space-y-2">
              <h3 className="text-sm font-medium text-muted-foreground">Error Details</h3>
              <div className="bg-red-50 dark:bg-red-950/20 border border-red-200 dark:border-red-800 p-4 rounded-lg">
                <p className="text-sm text-red-600 dark:text-red-400">
                  {execution.errorMessage}
                </p>
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between gap-2 p-4 border-t bg-muted/10">
          <div className="flex items-center gap-2">
            {/* Navigate to workflow */}
            {workflowName && onNavigateToWorkflow && (
              <Button 
                variant="outline" 
                size="sm" 
                onClick={() => {
                  onNavigateToWorkflow(workflowName)
                  onClose()
                }}
                className="flex items-center gap-2"
              >
                <Database className="h-3 w-3" />
                View Workflow
              </Button>
            )}
            
            {/* Navigate to schedule */}
            {parentRunId && onNavigateToSchedule && (
              <Button 
                variant="outline" 
                size="sm"
                onClick={() => {
                  onNavigateToSchedule(parentRunId)
                  onClose()
                }}
                className="flex items-center gap-2"
              >
                <CalendarDays className="h-3 w-3" />
                View Schedule
              </Button>
            )}
          </div>
          
          <Button variant="outline" size="sm" onClick={onClose}>
            Close
          </Button>
        </div>
      </div>
    </div>,
    document.body
  )
}