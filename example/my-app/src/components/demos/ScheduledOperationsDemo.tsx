import React from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../ui/card'
import { Button } from '../ui/button'
import { CalendarIcon } from 'lucide-react'
import { useRefreshStore } from '../../stores/refresh-store'
import { useWorkflowApi } from '../../hooks/use-workflow-api'
import * as workflow from '../../workflow/github_com_widmogrod_mkunion_x_workflow'
import * as schema from '../../workflow/github_com_widmogrod_mkunion_x_schema'

interface ScheduledOperationsDemoProps {
  input: string
}

export function ScheduledOperationsDemo({ input }: ScheduledOperationsDemoProps) {
  const { refreshAll } = useRefreshStore()
  const { runCommand, callFunction } = useWorkflowApi()

  const handleConcatAwait = async () => {
    try {
      const cmd: workflow.Command = {
        $type: 'workflow.Run',
        'workflow.Run': {
          Flow: {
            $type: 'workflow.Flow',
            'workflow.Flow': {
              Name: 'concat_await',
              Arg: 'input',
              Body: [
                {
                  $type: 'workflow.Apply',
                  'workflow.Apply': {
                    Name: 'concat',
                    Args: [
                      { $type: 'workflow.GetValue', 'workflow.GetValue': { Path: 'input' } },
                      { $type: 'workflow.SetValue', 'workflow.SetValue': { Value: { 'schema.String': ' - awaited result' } } }
                    ],
                    Await: {
                      TimeoutSeconds: 100
                    }
                  }
                }
              ]
            }
          },
          Input: { 'schema.String': input || 'concat await demo' }
        }
      }

      await runCommand(cmd)
      refreshAll()
    } catch (error) {
      console.error('Failed to run concat await:', error)
      alert(`Failed to run concat await: ${error instanceof Error ? error.message : 'Unknown error'}`)
    }
  }

  const handleScheduledRun = async () => {
    try {
      const cmd: workflow.Command = {
        $type: 'workflow.Run',
        'workflow.Run': {
          Flow: {
            $type: 'workflow.Flow',
            'workflow.Flow': {
              Name: 'scheduled_demo',
              Arg: 'input',
              Body: [
                {
                  $type: 'workflow.Apply',
                  'workflow.Apply': {
                    Name: 'concat',
                    Args: [
                      { $type: 'workflow.GetValue', 'workflow.GetValue': { Path: 'input' } },
                      { $type: 'workflow.SetValue', 'workflow.SetValue': { Value: { 'schema.String': ' - scheduled execution' } } }
                    ]
                  }
                }
              ]
            }
          },
          Input: { 'schema.String': input || 'scheduled run demo' },
          RunOption: {
            $type: 'workflow.ScheduleRun',
            'workflow.ScheduleRun': {
              Interval: '*/10 * * * * *', // Every 10 seconds
              ParentRunID: `parent_${Date.now()}`
            }
          }
        }
      }

      await runCommand(cmd)
      refreshAll()
      alert('Scheduled workflow created! It will run every 10 seconds.')
    } catch (error) {
      console.error('Failed to create scheduled run:', error)
      alert(`Failed to create scheduled run: ${error instanceof Error ? error.message : 'Unknown error'}`)
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
        alert(`Function result: ${JSON.stringify(result.Result)}`)
      }
      
      refreshAll()
    } catch (error) {
      console.error('Failed to call func:', error)
      alert(`Failed to call func: ${error instanceof Error ? error.message : 'Unknown error'}`)
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