import React from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../ui/card'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { PlayIcon } from 'lucide-react'
import { useWorkflowApi } from '../../hooks/use-workflow-api'
import { useRefreshStore } from '../../stores/refresh-store'
import { useToast } from '../../contexts/ToastContext'
import { createHelloWorldFlow } from '../../workflows/definitions/hello-world'
import * as builders from '../../workflows/builders'
import * as workflow from '../../workflow/github_com_widmogrod_mkunion_x_workflow'
import * as schema from '../../workflow/github_com_widmogrod_mkunion_x_schema'

export function HelloWorldDemo() {
  const [input, setInput] = React.useState('Amigo')
  const [loading, setLoading] = React.useState(false)
  const { flowCreate, runCommand } = useWorkflowApi()
  const { refreshAll } = useRefreshStore()
  const toast = useToast()

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

  const registerParameterizedWorkflow = async () => {
    try {
      // Create a workflow that takes name and age as parameters
      const flow: workflow.Flow = {
        Name: 'greet-user',
        Arg: 'input',
        Body: [
          {
            $type: 'workflow.Assign',
            'workflow.Assign': {
              ID: 'build-greeting',
              VarOk: 'greeting',
              VarErr: '',
              Val: {
                $type: 'workflow.Apply',
                'workflow.Apply': {
                  ID: 'concat-greeting',
                  Name: 'concat',
                  Args: [
                    {
                      $type: 'workflow.SetValue',
                      'workflow.SetValue': {
                        Value: {
                          $type: 'schema.String',
                          'schema.String': 'Hello, '
                        }
                      }
                    },
                    {
                      $type: 'workflow.GetValue',
                      'workflow.GetValue': {
                        Path: 'input.name'
                      }
                    },
                    {
                      $type: 'workflow.SetValue',
                      'workflow.SetValue': {
                        Value: {
                          $type: 'schema.String',
                          'schema.String': '! You are '
                        }
                      }
                    },
                    {
                      $type: 'workflow.GetValue',
                      'workflow.GetValue': {
                        Path: 'input.age'
                      }
                    },
                    {
                      $type: 'workflow.SetValue',
                      'workflow.SetValue': {
                        Value: {
                          $type: 'schema.String',
                          'schema.String': ' years old.'
                        }
                      }
                    }
                  ]
                }
              }
            }
          },
          {
            $type: 'workflow.Choose',
            'workflow.Choose': {
              If: {
                $type: 'workflow.Compare',
                'workflow.Compare': {
                  Left: {
                    $type: 'workflow.GetValue',
                    'workflow.GetValue': {
                      Path: 'input.age'
                    }
                  },
                  Operation: '>',
                  Right: {
                    $type: 'workflow.SetValue',
                    'workflow.SetValue': {
                      Value: {
                        $type: 'schema.Number',
                        'schema.Number': 18
                      }
                    }
                  }
                }
              },
              Then: [
                {
                  $type: 'workflow.End',
                  'workflow.End': {
                    ID: 'adult-greeting',
                    Result: {
                      $type: 'workflow.SetValue',
                      'workflow.SetValue': {
                        Value: {
                          $type: 'schema.String',
                          'schema.String': 'Adult user detected! Welcome!'
                        }
                      }
                    }
                  }
                }
              ],
              Else: [
                {
                  $type: 'workflow.End',
                  'workflow.End': {
                    ID: 'minor-greeting',
                    Result: {
                      $type: 'workflow.GetValue',
                      'workflow.GetValue': {
                        Path: 'greeting'
                      }
                    }
                  }
                }
              ]
            }
          }
        ]
      }
      
      await flowCreate(flow)
      toast.success('Workflow Registered', 'Parameterized workflow "greet-user" has been registered')
      refreshAll()
    } catch (error) {
      console.error('Failed to register workflow:', error)
      toast.error('Registration Failed', `Failed to register workflow: ${error instanceof Error ? error.message : 'Unknown error'}`)
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
          <Button
            onClick={registerParameterizedWorkflow}
            disabled={loading}
            variant="default"
            className="w-full"
            size="sm"
          >
            Register Parameterized Workflow
          </Button>
        </div>
        {loading && <p className="text-sm text-muted-foreground">Loading...</p>}
      </CardContent>
    </Card>
  )
}