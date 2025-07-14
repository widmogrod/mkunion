import React from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../components/ui/card'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Badge } from '../components/ui/badge'
import { 
  CalendarIcon, 
  PauseCircle, 
  PlayCircle, 
  Search, 
  Plus,
  Clock,
  History,
  Trash2
} from 'lucide-react'
import { useWorkflowApi } from '../hooks/use-workflow-api'
import { useRefreshStore } from '../stores/refresh-store'
import { useToast } from '../contexts/ToastContext'
import { CreateScheduleDialog } from '../components/schedule/CreateScheduleDialog'
import { NextRunDisplay } from '../components/schedule/NextRunDisplay'
import { ScheduleStatusBadge } from '../components/schedule/ScheduleStatusBadge'
import { ScheduleHistoryDialog } from '../components/schedule/ScheduleHistoryDialog'
import * as workflow from '../workflow/github_com_widmogrod_mkunion_x_workflow'
import * as schemaless from '../workflow/github_com_widmogrod_mkunion_x_storage_schemaless'

interface ScheduledWorkflow {
  id: string
  flowName: string
  cronExpression: string
  status: 'active' | 'paused'
  parentRunId: string
  nextRun?: number
  lastRun?: number
  state: workflow.State
  record: schemaless.Record<workflow.State>
}

export function SchedulesPage() {
  const { listStates, stopSchedule, resumeSchedule, deleteStates } = useWorkflowApi()
  const { refreshAll, schedulesRefreshTrigger } = useRefreshStore()
  const toast = useToast()
  
  const [scheduledWorkflows, setScheduledWorkflows] = React.useState<ScheduledWorkflow[]>([])
  const [loading, setLoading] = React.useState(true)
  const [searchTerm, setSearchTerm] = React.useState('')
  const [showCreateDialog, setShowCreateDialog] = React.useState(false)
  const [selectedSchedule, setSelectedSchedule] = React.useState<ScheduledWorkflow | null>(null)
  const [showHistoryDialog, setShowHistoryDialog] = React.useState(false)

  // Load scheduled workflows
  React.useEffect(() => {
    loadScheduledWorkflows()
  }, [])

  // Listen for global refresh events
  React.useEffect(() => {
    const handleRefresh = () => {
      loadScheduledWorkflows()
    }
    
    // Subscribe to refresh events
    window.addEventListener('refresh-schedules', handleRefresh)
    
    return () => {
      window.removeEventListener('refresh-schedules', handleRefresh)
    }
  }, [])

  // Auto-refresh when schedulesRefreshTrigger changes
  React.useEffect(() => {
    if (schedulesRefreshTrigger > 0) {
      loadScheduledWorkflows()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [schedulesRefreshTrigger])

  const loadScheduledWorkflows = async () => {
    try {
      setLoading(true)
      
      console.log('Loading scheduled workflows...')
      
      // Query for both Scheduled and ScheduleStopped states
      const response = await listStates({
        where: {
          Predicate: {
            "$type": "predicate.Or",
            "predicate.Or": {
              L: [
                {
                  "$type": "predicate.Compare",
                  "predicate.Compare": {
                    Location: 'Data["$type"]',
                    Operation: "==",
                    BindValue: {
                      "$type": "predicate.Literal",
                      "predicate.Literal": {
                        Value: { "$type": "schema.String", "schema.String": "workflow.Scheduled" }
                      }
                    }
                  }
                },
                {
                  "$type": "predicate.Compare",
                  "predicate.Compare": {
                    Location: 'Data["$type"]',
                    Operation: "==",
                    BindValue: {
                      "$type": "predicate.Literal",
                      "predicate.Literal": {
                        Value: { "$type": "schema.String", "schema.String": "workflow.ScheduleStopped" }
                      }
                    }
                  }
                }
              ]
            }
          }
        },
        limit: 100
      })

      console.log('Query response:', response)
      console.log('Found items:', response.Items?.length || 0)

      // Transform the states into scheduled workflow objects
      const workflows: ScheduledWorkflow[] = []
      
      if (response.Items) {
        for (const item of response.Items) {
          const state = item.Data
          if (!state) continue
          
          let baseState: any = null
          let status: 'active' | 'paused' = 'active'
          
          if (state.$type === 'workflow.Scheduled') {
            baseState = state['workflow.Scheduled']?.BaseState
            status = 'active'
          } else if (state.$type === 'workflow.ScheduleStopped') {
            baseState = state['workflow.ScheduleStopped']?.BaseState
            status = 'paused'
          }
          
          if (baseState?.RunOption?.['workflow.ScheduleRun'] && item.ID) {
            const scheduleRun = baseState.RunOption['workflow.ScheduleRun']
            const flowName = baseState.Flow?.['workflow.Flow']?.Name || 'Unknown'
            
            workflows.push({
              id: item.ID,
              flowName,
              cronExpression: scheduleRun.Interval || '',
              status,
              parentRunId: scheduleRun.ParentRunID || '',
              nextRun: state.$type === 'workflow.Scheduled' 
                ? state['workflow.Scheduled']?.ExpectedRunTimestamp 
                : undefined,
              state: state,
              record: item
            })
          }
        }
      }
      
      console.log('Processed workflows:', workflows.length)
      setScheduledWorkflows(workflows)
    } catch (error) {
      console.error('Failed to load scheduled workflows:', error)
      toast.error('Failed to load schedules', 'Unable to retrieve scheduled workflows')
    } finally {
      setLoading(false)
    }
  }

  const handlePauseResume = async (schedule: ScheduledWorkflow) => {
    try {
      if (schedule.status === 'active') {
        await stopSchedule(schedule.parentRunId)
        toast.success('Schedule paused', `${schedule.flowName} has been paused`)
      } else {
        await resumeSchedule(schedule.parentRunId)
        toast.success('Schedule resumed', `${schedule.flowName} has been resumed`)
      }
      refreshAll()
      loadScheduledWorkflows()
    } catch (error) {
      console.error('Failed to update schedule:', error)
      toast.error('Failed to update schedule', error instanceof Error ? error.message : 'Unknown error')
    }
  }

  const handleDelete = async (schedule: ScheduledWorkflow) => {
    try {
      await deleteStates([schedule.record])
      toast.success('Schedule deleted', `${schedule.flowName} schedule has been deleted`)
      refreshAll()
      loadScheduledWorkflows()
    } catch (error) {
      console.error('Failed to delete schedule:', error)
      toast.error('Failed to delete schedule', error instanceof Error ? error.message : 'Unknown error')
    }
  }

  const filteredWorkflows = scheduledWorkflows.filter(workflow =>
    workflow.flowName.toLowerCase().includes(searchTerm.toLowerCase()) ||
    workflow.cronExpression.toLowerCase().includes(searchTerm.toLowerCase())
  )

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold">Scheduled Workflows</h1>
          <p className="text-muted-foreground mt-2">
            Manage and monitor your scheduled workflow executions
          </p>
        </div>
        <Button onClick={() => setShowCreateDialog(true)}>
          <Plus className="h-4 w-4 mr-2" />
          Create Schedule
        </Button>
      </div>

      <Card>
        <CardHeader>
          <div className="flex justify-between items-center">
            <CardTitle>Active Schedules</CardTitle>
            <div className="relative w-64">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Search schedules..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="pl-10"
              />
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="text-center py-8 text-muted-foreground">
              Loading schedules...
            </div>
          ) : filteredWorkflows.length === 0 ? (
            <div className="text-center py-8">
              <CalendarIcon className="h-12 w-12 mx-auto mb-4 text-muted-foreground" />
              <p className="text-muted-foreground">
                {searchTerm ? 'No schedules match your search' : 'No scheduled workflows yet'}
              </p>
              {!searchTerm && (
                <Button 
                  variant="outline" 
                  className="mt-4"
                  onClick={() => setShowCreateDialog(true)}
                >
                  Create your first schedule
                </Button>
              )}
            </div>
          ) : (
            <div className="space-y-4">
              {filteredWorkflows.map((schedule) => (
                <div
                  key={schedule.id}
                  className="border rounded-lg p-4 hover:bg-muted/50 transition-colors"
                >
                  <div className="flex items-center justify-between">
                    <div className="space-y-2">
                      <div className="flex items-center gap-3">
                        <h3 className="font-semibold text-lg">{schedule.flowName}</h3>
                        <ScheduleStatusBadge status={schedule.status} />
                      </div>
                      
                      <div className="flex items-center gap-6 text-sm text-muted-foreground">
                        <div className="flex items-center gap-2">
                          <Clock className="h-4 w-4" />
                          <code className="bg-muted px-2 py-1 rounded">
                            {schedule.cronExpression}
                          </code>
                        </div>
                        
                        {schedule.nextRun && schedule.status === 'active' && (
                          <NextRunDisplay timestamp={schedule.nextRun} />
                        )}
                        
                        <div className="flex items-center gap-2">
                          <span className="text-xs">Parent ID:</span>
                          <code className="text-xs">{schedule.parentRunId.slice(0, 8)}...</code>
                        </div>
                      </div>
                    </div>
                    
                    <div className="flex items-center gap-2">
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => {
                          setSelectedSchedule(schedule)
                          setShowHistoryDialog(true)
                        }}
                        title="View history"
                      >
                        <History className="h-4 w-4" />
                      </Button>
                      
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => handlePauseResume(schedule)}
                        title={schedule.status === 'active' ? 'Pause' : 'Resume'}
                      >
                        {schedule.status === 'active' ? (
                          <PauseCircle className="h-4 w-4" />
                        ) : (
                          <PlayCircle className="h-4 w-4" />
                        )}
                      </Button>
                      
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => handleDelete(schedule)}
                        title="Delete schedule"
                      >
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {showCreateDialog && (
        <CreateScheduleDialog
          isOpen={showCreateDialog}
          onClose={() => setShowCreateDialog(false)}
          onSuccess={() => {
            setShowCreateDialog(false)
            loadScheduledWorkflows()
            refreshAll()
          }}
        />
      )}

      {showHistoryDialog && selectedSchedule && (
        <ScheduleHistoryDialog
          isOpen={showHistoryDialog}
          onClose={() => {
            setShowHistoryDialog(false)
            setSelectedSchedule(null)
          }}
          schedule={selectedSchedule}
        />
      )}
    </div>
  )
}