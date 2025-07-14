import React from 'react'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { Label } from '../ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../ui/select'
import { Textarea } from '../ui/textarea'
import { X, Calendar, Info } from 'lucide-react'
import { useWorkflowApi } from '../../hooks/use-workflow-api'
import { useToast } from '../../contexts/ToastContext'
import { CronExpressionBuilder } from './CronExpressionBuilder'
import * as workflow from '../../workflow/github_com_widmogrod_mkunion_x_workflow'
import * as schema from '../../workflow/github_com_widmogrod_mkunion_x_schema'
import * as builders from '../../workflows/builders'

interface CreateScheduleDialogProps {
  isOpen: boolean
  onClose: () => void
  onSuccess: () => void
}

export function CreateScheduleDialog({ isOpen, onClose, onSuccess }: CreateScheduleDialogProps) {
  const { listFlows, runCommand, flowCreate } = useWorkflowApi()
  const toast = useToast()
  
  const [flows, setFlows] = React.useState<workflow.Flow[]>([])
  const [selectedFlow, setSelectedFlow] = React.useState<string>('')
  const [cronExpression, setCronExpression] = React.useState('0 */5 * * * *') // Every 5 minutes
  const [inputValue, setInputValue] = React.useState('')
  const [parentRunId, setParentRunId] = React.useState(`schedule_${Date.now()}`)
  const [loading, setLoading] = React.useState(false)

  React.useEffect(() => {
    if (isOpen) {
      loadFlows()
    }
  }, [isOpen])

  const loadFlows = async () => {
    try {
      const response = await listFlows({ limit: 100 })
      if (response.Items) {
        const flowData = response.Items
          .map(item => item.Data)
          .filter((data): data is workflow.Flow => data !== undefined)
        setFlows(flowData)
        if (flowData.length > 0 && !selectedFlow && flowData[0].Name) {
          setSelectedFlow(flowData[0].Name)
        }
      }
    } catch (error) {
      console.error('Failed to load flows:', error)
      toast.error('Failed to load workflows', 'Unable to retrieve available workflows')
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!selectedFlow || !cronExpression) {
      toast.error('Missing required fields', 'Please select a workflow and set a schedule')
      return
    }

    try {
      setLoading(true)
      
      // Find the selected flow
      const flow = flows.find(f => f.Name === selectedFlow)
      if (!flow || !flow.Name) {
        throw new Error('Selected workflow not found')
      }

      // Create the scheduled run command
      const cmd: workflow.Command = {
        $type: 'workflow.Run',
        'workflow.Run': {
          Flow: {
            $type: 'workflow.Flow',
            'workflow.Flow': flow
          },
          Input: inputValue ? builders.stringValue(inputValue) : builders.stringValue(''),
          RunOption: {
            $type: 'workflow.ScheduleRun',
            'workflow.ScheduleRun': {
              Interval: cronExpression,
              ParentRunID: parentRunId
            }
          }
        }
      }

      await runCommand(cmd)
      toast.success('Schedule created', `${selectedFlow} has been scheduled successfully`)
      onSuccess()
    } catch (error) {
      console.error('Failed to create schedule:', error)
      toast.error('Failed to create schedule', error instanceof Error ? error.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-background rounded-lg shadow-lg w-full max-w-2xl max-h-[90vh] overflow-hidden">
        <div className="flex items-center justify-between p-6 border-b">
          <h2 className="text-xl font-semibold">Create Scheduled Workflow</h2>
          <Button
            variant="ghost"
            size="icon"
            onClick={onClose}
            className="rounded-full"
          >
            <X className="h-4 w-4" />
          </Button>
        </div>

        <form onSubmit={handleSubmit} className="p-6 space-y-6 overflow-y-auto max-h-[calc(90vh-200px)]">
          <div className="space-y-2">
            <Label htmlFor="workflow">Workflow</Label>
            <Select value={selectedFlow} onValueChange={setSelectedFlow}>
              <SelectTrigger id="workflow">
                <SelectValue placeholder="Select a workflow to schedule" />
              </SelectTrigger>
              <SelectContent>
                {flows.filter(flow => flow.Name).map((flow) => (
                  <SelectItem key={flow.Name} value={flow.Name!}>
                    {flow.Name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label>Schedule (Cron Expression)</Label>
            <CronExpressionBuilder
              value={cronExpression}
              onChange={setCronExpression}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="input">Workflow Input (Optional)</Label>
            <Textarea
              id="input"
              placeholder="Enter input data for the workflow..."
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              rows={3}
            />
            <p className="text-sm text-muted-foreground">
              This input will be passed to the workflow on each scheduled execution
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="parentRunId">Parent Run ID</Label>
            <Input
              id="parentRunId"
              value={parentRunId}
              onChange={(e) => setParentRunId(e.target.value)}
              placeholder="Unique identifier for this schedule"
            />
            <p className="text-sm text-muted-foreground flex items-center gap-2">
              <Info className="h-3 w-3" />
              This ID is used to manage (pause/resume) the schedule
            </p>
          </div>
        </form>

        <div className="flex justify-end gap-3 p-6 border-t">
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={loading}>
            {loading ? 'Creating...' : 'Create Schedule'}
          </Button>
        </div>
      </div>
    </div>
  )
}