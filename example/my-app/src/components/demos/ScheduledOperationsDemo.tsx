import React from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../ui/card'
import { Button } from '../ui/button'
import { CalendarIcon } from 'lucide-react'
import { useRefreshStore } from '../../stores/refresh-store'
import { useWorkflowApi } from '../../hooks/use-workflow-api'
import { useToast } from '../../contexts/ToastContext'
import * as workflow from '../../workflow/github_com_widmogrod_mkunion_x_workflow'
import * as schema from '../../workflow/github_com_widmogrod_mkunion_x_schema'
import * as builders from '../../workflows/builders'

interface ScheduledOperationsDemoProps {
  input: string
}

export function ScheduledOperationsDemo({ input }: ScheduledOperationsDemoProps) {
  const { refreshAll } = useRefreshStore()
  const { runCommand, callFunction, flowCreate } = useWorkflowApi()
  const toast = useToast()

  const handleConcatAwait = async () => {
    try {
      // First create and register the workflow - with await functionality restored
      const flow = builders.createFlow('concat_await', 'input', [
        builders.assign(
          'assign1',
          'result',
          {
            $type: 'workflow.Apply',
            'workflow.Apply': {
              ID: 'apply1',
              Name: 'concat',
              Args: [
                builders.getValue('input'),
                builders.setValue(builders.stringValue(' - awaited result'))
              ],
              Await: {
                TimeoutSeconds: 100
              }
            }
          }
        ),
        builders.end('end1', builders.getValue('result'))
      ])

      // Register the workflow first
      await flowCreate(flow)

      // Then run it
      const cmd = builders.createRunCommand(
        flow,
        builders.stringValue(input || 'concat await demo')
      )

      await runCommand(cmd)
      refreshAll()
    } catch (error) {
      console.error('Failed to run concat await:', error)
      toast.error('Concat Await Failed', `Failed to run concat await: ${error instanceof Error ? error.message : 'Unknown error'}`)
    }
  }

  const handleScheduledRun = async () => {
    try {
      // First create and register the workflow
      const flow = builders.createFlow('scheduled_demo', 'input', [
        builders.assign(
          'assign1',
          'result',
          builders.apply(
            'apply1',
            'concat',
            [
              builders.getValue('input'),
              builders.setValue(builders.stringValue(' - scheduled execution'))
            ]
          )
        ),
        builders.end('end1', builders.getValue('result'))
      ])

      // Register the workflow first
      await flowCreate(flow)

      // Use the new builder function for scheduled runs
      const cmd = builders.createScheduledRunCommand(
        flow,
        builders.stringValue(input || 'scheduled run demo'),
        '*/10 * * * * *', // Every 10 seconds
        `parent_${Date.now()}`
      )

      await runCommand(cmd)
      refreshAll()
      toast.success('Scheduled Workflow Created', 'Workflow scheduled successfully! It will run every 10 seconds. View and manage it in the Schedules tab.')
    } catch (error) {
      console.error('Failed to create scheduled run:', error)
      toast.error('Scheduling Failed', `Failed to create scheduled run: ${error instanceof Error ? error.message : 'Unknown error'}`)
    }
  }

  const handleCallFunc = async () => {
    try {
      // Call the concat function directly via /func endpoint
      const args: schema.Schema[] = [
        { 'schema.String': input || 'default input' },
        { 'schema.String': ' - called via /func endpoint' }
      ]

      const result = await callFunction('concat', args)
      
      // Show the result to the user
      if (result.Result) {
        toast.success('Function Result', JSON.stringify(result.Result))
      }
      
      refreshAll()
    } catch (error) {
      console.error('Failed to call func:', error)
      toast.error('Function Call Failed', `Failed to call func: ${error instanceof Error ? error.message : 'Unknown error'}`)
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <CalendarIcon className="h-5 w-5" />
          Scheduled & Async Operations
        </CardTitle>
        <CardDescription>Run workflows with delays and callbacks</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex flex-col gap-2">
          <Button 
            variant="outline" 
            onClick={handleConcatAwait}
            className="w-full"
            size="sm"
          >
            Run concat await
          </Button>
          <Button 
            variant="outline"
            onClick={handleScheduledRun}
            className="w-full"
            size="sm"
          >
            Scheduled Run
          </Button>
          <Button 
            variant="outline"
            onClick={handleCallFunc}
            className="w-full"
            size="sm"
          >
            Call func - Concat with {input}
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}