import React, { useState, useEffect } from 'react'
import ReactDOM from 'react-dom'
import { X, Calendar, Table, Download, Filter } from 'lucide-react'
import { Button } from '../ui/button'
import { Badge } from '../ui/badge'
import { Input } from '../ui/input'
import { useWorkflowApi } from '../../hooks/use-workflow-api'
import { useToast } from '../../contexts/ToastContext'
import * as workflow from '../../workflow/github_com_widmogrod_mkunion_x_workflow'
import * as schemaless from '../../workflow/github_com_widmogrod_mkunion_x_storage_schemaless'

interface ScheduleHistoryDialogProps {
  isOpen: boolean
  onClose: () => void
  schedule: {
    id: string
    flowName: string
    parentRunId: string
    cronExpression: string
    status: 'active' | 'paused'
  }
}

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

type ViewMode = 'table' | 'calendar'

export function ScheduleHistoryDialog({ isOpen, onClose, schedule }: ScheduleHistoryDialogProps) {
  const [executions, setExecutions] = useState<RunExecution[]>([])
  const [loading, setLoading] = useState(false)
  const [viewMode, setViewMode] = useState<ViewMode>('table')
  const [dateFilter, setDateFilter] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('')
  
  const { listStates } = useWorkflowApi()
  const toast = useToast()

  // Load run history when dialog opens
  useEffect(() => {
    if (isOpen && schedule.parentRunId) {
      loadRunHistory()
    }
  }, [isOpen, schedule.parentRunId])

  const loadRunHistory = async () => {
    try {
      setLoading(true)
      console.log('Loading run history for parent:', schedule.parentRunId)

      // Create backend predicate to filter by ParentRunID for different state types
      // This is much more efficient than client-side filtering
      const response = await listStates({
        where: {
          Predicate: {
            "$type": "predicate.Or",
            "predicate.Or": {
              L: [
                // workflow.Done states with matching ParentRunID
                {
                  "$type": "predicate.Compare",
                  "predicate.Compare": {
                    Location: 'Data["workflow.Done"]["BaseState"]["RunOption"]["workflow.ScheduleRun"]["ParentRunID"]',
                    Operation: "==",
                    BindValue: {
                      "$type": "predicate.Literal",
                      "predicate.Literal": {
                        Value: { "$type": "schema.String", "schema.String": schedule.parentRunId }
                      }
                    }
                  }
                },
                // workflow.Scheduled states with matching ParentRunID
                {
                  "$type": "predicate.Compare",
                  "predicate.Compare": {
                    Location: 'Data["workflow.Scheduled"]["BaseState"]["RunOption"]["workflow.ScheduleRun"]["ParentRunID"]',
                    Operation: "==",
                    BindValue: {
                      "$type": "predicate.Literal",
                      "predicate.Literal": {
                        Value: { "$type": "schema.String", "schema.String": schedule.parentRunId }
                      }
                    }
                  }
                },
                // workflow.ScheduleStopped states with matching ParentRunID
                {
                  "$type": "predicate.Compare",
                  "predicate.Compare": {
                    Location: 'Data["workflow.ScheduleStopped"]["BaseState"]["RunOption"]["workflow.ScheduleRun"]["ParentRunID"]',
                    Operation: "==",
                    BindValue: {
                      "$type": "predicate.Literal",
                      "predicate.Literal": {
                        Value: { "$type": "schema.String", "schema.String": schedule.parentRunId }
                      }
                    }
                  }
                },
                // workflow.Error states with matching ParentRunID  
                {
                  "$type": "predicate.Compare",
                  "predicate.Compare": {
                    Location: 'Data["workflow.Error"]["BaseState"]["RunOption"]["workflow.ScheduleRun"]["ParentRunID"]',
                    Operation: "==",
                    BindValue: {
                      "$type": "predicate.Literal",
                      "predicate.Literal": {
                        Value: { "$type": "schema.String", "schema.String": schedule.parentRunId }
                      }
                    }
                  }
                },
                // workflow.Await states with matching ParentRunID
                {
                  "$type": "predicate.Compare",
                  "predicate.Compare": {
                    Location: 'Data["workflow.Await"]["BaseState"]["RunOption"]["workflow.ScheduleRun"]["ParentRunID"]',
                    Operation: "==",
                    BindValue: {
                      "$type": "predicate.Literal",
                      "predicate.Literal": {
                        Value: { "$type": "schema.String", "schema.String": schedule.parentRunId }
                      }
                    }
                  }
                }
              ]
            }
          }
        },
        limit: 100 // Much smaller limit since backend does filtering
      })

      console.log('✅ Backend query response:', response.Items?.length || 0, 'pre-filtered items')
      
      if (response.Items && response.Items.length > 0) {
        console.log('🎯 Backend filtering worked! Found', response.Items.length, 'states for ParentRunID:', schedule.parentRunId)
      } else {
        console.log('❌ Backend filter returned no results for ParentRunID:', schedule.parentRunId)
      }

      // Process the states to group them into execution runs
      const processedExecutions = processRunHistory(response.Items || [])
      console.log('📋 Final result:', processedExecutions.length, 'executions ready for display')
      
      setExecutions(processedExecutions)
    } catch (error) {
      console.error('Failed to load run history:', error)
      toast.error('Failed to load history', 'Unable to retrieve run history')
    } finally {
      setLoading(false)
    }
  }

  const processRunHistory = (states: schemaless.Record<workflow.State>[]): RunExecution[] => {
    console.log('🚀 Processing', states.length, 'pre-filtered states from backend')
    console.log('✅ Backend already filtered for ParentRunID:', schedule.parentRunId)
    
    // All states are already filtered by backend, so we can use them directly
    const relatedStates = states.filter(state => state.Data) // Just filter out any without data
    console.log('📊 Valid states for processing:', relatedStates.length)

    // Group states by RunID to create execution runs
    const runGroups: { [runId: string]: schemaless.Record<workflow.State>[] } = {}
    
    relatedStates.forEach(state => {
      const stateData = state.Data
      if (!stateData) return
      
      const stateType = stateData.$type
      
      // Extract BaseState from the specific state type
      let baseState: any = null
      if (stateType && stateData[stateType as keyof typeof stateData]) {
        baseState = (stateData[stateType as keyof typeof stateData] as any)?.BaseState
      }
      
      const runId = baseState?.RunID
      if (runId) {
        if (!runGroups[runId]) {
          runGroups[runId] = []
        }
        runGroups[runId].push(state)
      }
    })
    
    console.log('📈 Created', Object.keys(runGroups).length, 'execution groups from backend-filtered data')

    // Convert run groups to RunExecution objects
    const executions: RunExecution[] = Object.entries(runGroups).map(([runId, runStates]) => {
      // For now, just take first and last states (could improve sorting later)
      const firstState = runStates[0]
      const lastState = runStates[runStates.length - 1]
      
      // Determine status from the final state
      let status: 'scheduled' | 'running' | 'done' | 'error' = 'scheduled'
      let errorMessage: string | undefined
      
      const finalStateType = lastState.Data?.$type
      if (finalStateType === 'workflow.Done') {
        status = 'done'
      } else if (finalStateType === 'workflow.Error') {
        status = 'error'
        // Try to extract error message
        const errorState = lastState.Data?.['workflow.Error'] as any
        errorMessage = errorState?.Message || 'Unknown error'
      } else if (finalStateType === 'workflow.Await') {
        status = 'running'
      }

      // TODO: Actual execution timestamps are not tracked in the current workflow system
      // The Execution struct with StartTime/EndTime exists but is not populated
      // For now, we'll use the ExpectedRunTimestamp from Scheduled states if available
      const scheduledState = runStates.find(s => s.Data?.$type === 'workflow.Scheduled')
      let scheduledTimestamp: number | undefined
      if (scheduledState?.Data?.$type === 'workflow.Scheduled') {
        const scheduledData = scheduledState.Data['workflow.Scheduled'] as workflow.Scheduled
        scheduledTimestamp = scheduledData?.ExpectedRunTimestamp
      }
      
      // Use scheduled timestamp if available, otherwise use current time minus index for ordering
      const startTime = scheduledTimestamp 
        ? new Date(scheduledTimestamp * 1000) // Convert Unix timestamp to milliseconds
        : new Date(Date.now() - (Object.keys(runGroups).length - Object.keys(runGroups).indexOf(runId)) * 3600000)
      
      // Duration is not available without actual execution timestamps
      const endTime = undefined
      const duration = undefined

      return {
        id: runId,
        startTime,
        endTime,
        status,
        duration,
        errorMessage,
        inputData: firstState.Data,
        outputData: lastState.Data
      }
    })

    // Sort executions by ID (newest first - assuming IDs are chronological)
    const sortedExecutions = executions.sort((a, b) => b.id.localeCompare(a.id))
    
    console.log('🎉 Backend filtering successful:', sortedExecutions.length, 'executions processed')
    return sortedExecutions
  }

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'done':
        return <Badge className="bg-green-500 text-white">Done</Badge>
      case 'error':
        return <Badge className="bg-red-500 text-white">Error</Badge>
      case 'running':
        return <Badge className="bg-blue-500 text-white">Running</Badge>
      case 'scheduled':
        return <Badge className="bg-yellow-500 text-white">Scheduled</Badge>
      default:
        return <Badge variant="outline">{status}</Badge>
    }
  }

  const formatDuration = (ms?: number) => {
    if (!ms) return 'N/A'
    const seconds = Math.floor(ms / 1000)
    if (seconds < 60) return `${seconds}s`
    const minutes = Math.floor(seconds / 60)
    const remainingSeconds = seconds % 60
    return `${minutes}m ${remainingSeconds}s`
  }

  const filteredExecutions = executions.filter(execution => {
    if (statusFilter && execution.status !== statusFilter) return false
    if (dateFilter) {
      const filterDate = new Date(dateFilter)
      const executionDate = new Date(execution.startTime)
      return executionDate.toDateString() === filterDate.toDateString()
    }
    return true
  })

  const summaryStats = {
    total: executions.length,
    successful: executions.filter(e => e.status === 'done').length,
    failed: executions.filter(e => e.status === 'error').length,
    avgDuration: undefined // Duration tracking not implemented in workflow system
  }

  if (!isOpen) return null

  return ReactDOM.createPortal(
    <div className="fixed inset-0 z-[9999] flex items-center justify-center">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} />
      <div className="relative bg-background rounded-lg shadow-2xl w-full max-w-6xl max-h-[90vh] overflow-hidden z-[10000] border m-4">
        
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b bg-muted/30">
          <div className="flex items-center gap-3">
            <Calendar className="h-5 w-5 text-primary" />
            <div>
              <h2 className="text-xl font-semibold">Run History</h2>
              <p className="text-sm text-muted-foreground">
                {schedule.flowName} • {schedule.cronExpression}
              </p>
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

        {/* Summary Stats */}
        <div className="p-6 border-b bg-gradient-to-r from-muted/20 to-muted/10">
          <div className="grid grid-cols-4 gap-6">
            <div className="text-center">
              <div className="text-2xl font-bold text-primary">{summaryStats.total}</div>
              <div className="text-sm text-muted-foreground">Total Runs</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-green-600">{summaryStats.successful}</div>
              <div className="text-sm text-muted-foreground">Successful</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-red-600">{summaryStats.failed}</div>
              <div className="text-sm text-muted-foreground">Failed</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-blue-600">{formatDuration(summaryStats.avgDuration)}</div>
              <div className="text-sm text-muted-foreground">Avg Duration</div>
              <div className="text-xs text-muted-foreground mt-1">(Not tracked)</div>
            </div>
          </div>
        </div>

        {/* Controls */}
        <div className="flex items-center justify-between p-4 border-b bg-muted/10">
          <div className="flex items-center gap-3">
            {/* View Mode Toggle */}
            <div className="flex items-center bg-muted rounded-lg p-1">
              <Button
                variant={viewMode === 'table' ? 'default' : 'ghost'}
                size="sm"
                onClick={() => setViewMode('table')}
                className="h-8"
              >
                <Table className="h-4 w-4 mr-1" />
                Table
              </Button>
              <Button
                variant={viewMode === 'calendar' ? 'default' : 'ghost'}
                size="sm"
                onClick={() => setViewMode('calendar')}
                className="h-8"
              >
                <Calendar className="h-4 w-4 mr-1" />
                Calendar
              </Button>
            </div>

            {/* Filters */}
            <div className="flex items-center gap-2">
              <Input
                type="date"
                value={dateFilter}
                onChange={(e) => setDateFilter(e.target.value)}
                className="w-40"
                placeholder="Filter by date"
              />
              <select
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value)}
                className="h-9 px-3 rounded-md border border-input bg-background text-sm"
              >
                <option value="">All Status</option>
                <option value="done">Done</option>
                <option value="error">Error</option>
                <option value="running">Running</option>
                <option value="scheduled">Scheduled</option>
              </select>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm">
              <Download className="h-4 w-4 mr-1" />
              Export
            </Button>
            <Button variant="outline" size="sm" onClick={loadRunHistory} disabled={loading}>
              <Filter className="h-4 w-4 mr-1" />
              Refresh
            </Button>
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-auto p-6">
          {loading ? (
            <div className="flex items-center justify-center py-12">
              <div className="text-center">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-4"></div>
                <p className="text-muted-foreground">Loading run history...</p>
              </div>
            </div>
          ) : filteredExecutions.length === 0 ? (
            <div className="flex items-center justify-center py-12">
              <div className="text-center">
                <Calendar className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                <p className="text-lg font-medium mb-2">No executions found</p>
                <p className="text-muted-foreground mb-4">
                  {executions.length === 0 
                    ? "This schedule hasn't run yet or the data isn't available" 
                    : "No executions match your filters"
                  }
                </p>
                {executions.length === 0 && (
                  <div className="text-xs text-muted-foreground bg-muted/30 rounded-lg p-3 max-w-md mx-auto">
                    <p className="font-medium mb-1">Looking for ParentRunID:</p>
                    <code className="bg-background px-2 py-1 rounded text-xs">{schedule.parentRunId}</code>
                    <p className="mt-2 text-muted-foreground">
                      Check browser console for available ParentRunIDs in the system.
                    </p>
                  </div>
                )}
              </div>
            </div>
          ) : viewMode === 'table' ? (
            /* Table View */
            <div className="space-y-4">
              {filteredExecutions.map((execution) => (
                <div key={execution.id} className="border rounded-lg p-4 hover:bg-muted/30 transition-colors">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-4">
                      {getStatusBadge(execution.status)}
                      <div>
                        <div className="font-medium">
                          {execution.startTime.toLocaleString()}
                        </div>
                        <div className="text-sm text-muted-foreground">
                          Duration: {formatDuration(execution.duration)}
                        </div>
                      </div>
                    </div>
                    <div className="text-right">
                      <div className="text-sm text-muted-foreground">Run ID</div>
                      <div className="font-mono text-xs">{execution.id}</div>
                    </div>
                  </div>
                  {execution.errorMessage && (
                    <div className="mt-3 p-3 bg-red-50 dark:bg-red-950/20 rounded border border-red-200 dark:border-red-800">
                      <div className="text-sm text-red-600 dark:text-red-400">
                        {execution.errorMessage}
                      </div>
                    </div>
                  )}
                </div>
              ))}
            </div>
          ) : (
            /* Calendar View - Placeholder */
            <div className="flex items-center justify-center py-12">
              <div className="text-center">
                <Calendar className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                <p className="text-lg font-medium mb-2">Calendar View</p>
                <p className="text-muted-foreground">Coming soon...</p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>,
    document.body
  )
}