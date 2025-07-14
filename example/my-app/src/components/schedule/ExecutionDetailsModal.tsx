import React from 'react'
import ReactDOM from 'react-dom'
import { X, CheckCircle, XCircle, Clock, AlertCircle } from 'lucide-react'
import { Button } from '../ui/button'
import { Badge } from '../ui/badge'

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
}

export function ExecutionDetailsModal({ isOpen, onClose, execution }: ExecutionDetailsModalProps) {
  if (!isOpen || !execution) return null

  const getStatusIcon = () => {
    switch (execution.status) {
      case 'done':
        return <CheckCircle className="h-5 w-5 text-green-500" />
      case 'error':
        return <XCircle className="h-5 w-5 text-red-500" />
      case 'running':
        return <Clock className="h-5 w-5 text-blue-500 animate-pulse" />
      case 'scheduled':
        return <AlertCircle className="h-5 w-5 text-yellow-500" />
    }
  }

  const getStatusBadge = () => {
    switch (execution.status) {
      case 'done':
        return <Badge className="bg-green-500 text-white">Done</Badge>
      case 'error':
        return <Badge className="bg-red-500 text-white">Error</Badge>
      case 'running':
        return <Badge className="bg-blue-500 text-white">Running</Badge>
      case 'scheduled':
        return <Badge className="bg-yellow-500 text-white">Scheduled</Badge>
    }
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
        <div className="flex items-center justify-end gap-2 p-4 border-t bg-muted/10">
          <Button variant="outline" size="sm" onClick={onClose}>
            Close
          </Button>
        </div>
      </div>
    </div>,
    document.body
  )
}