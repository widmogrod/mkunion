import React from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../ui/card'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { PlayIcon } from 'lucide-react'
import { useWorkflowApi } from '../../hooks/use-workflow-api'
import { useRefreshStore } from '../../stores/refresh-store'
import { createHelloWorldFlow } from '../../workflows/definitions/hello-world'
import * as builders from '../../workflows/builders'

export function HelloWorldDemo() {
  const [input, setInput] = React.useState('Amigo')
  const [loading, setLoading] = React.useState(false)
  const { flowCreate, runCommand } = useWorkflowApi()
  const { refreshAll } = useRefreshStore()

  const runHelloWorldWorkflow = async (withError = false) => {
    setLoading(true)
    try {
      const flow = createHelloWorldFlow(withError)
      await flowCreate(flow)
      
      const cmd = builders.createRunCommand(
        flow,
        builders.stringValue(input)
      )
      
      const result = await runCommand(cmd)
      console.log('Workflow result:', result)
      
      // Refresh tables to show new workflow state
      refreshAll()
    } catch (error) {
      console.error('Error running workflow:', error)
    } finally {
      setLoading(false)
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <PlayIcon className="h-5 w-5" />
          Hello World Demo
        </CardTitle>
        <CardDescription>Run a simple workflow with or without errors</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <Input
          type="text"
          placeholder="Enter your name"
          value={input}
          onChange={(e) => setInput(e.target.value)}
        />
        <div className="flex flex-col gap-2">
          <Button 
            onClick={() => runHelloWorldWorkflow(false)}
            disabled={loading}
            variant="secondary"
            className="w-full"
            size="sm"
          >
            Run hello world workflow
          </Button>
          <Button 
            onClick={() => runHelloWorldWorkflow(true)}
            disabled={loading}
            variant="outline"
            className="w-full"
            size="sm"
          >
            Run hello world workflow with error
          </Button>
        </div>
        {loading && <p className="text-sm text-muted-foreground">Loading...</p>}
      </CardContent>
    </Card>
  )
}