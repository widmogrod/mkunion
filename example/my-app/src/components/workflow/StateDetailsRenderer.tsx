import React, { useState, useEffect } from 'react'
import { Badge } from '../ui/badge'
import { Button } from '../ui/button'
import { Code, FileText, Play, Pause, RotateCcw, Send } from 'lucide-react'
import { useWorkflowApi } from '../../hooks/use-workflow-api'
import { StatusBadge } from '../tables/StatusBadge'
import { ResultPreview } from './ResultPreview'
import * as workflow from '../../workflow/github_com_widmogrod_mkunion_x_workflow'
import * as schemaless from '../../workflow/github_com_widmogrod_mkunion_x_storage_schemaless'

interface StateDetailsRendererProps {
  data: schemaless.Record<workflow.State>
}

export function StateDetailsRenderer({ data }: StateDetailsRendererProps) {
  const [showCode, setShowCode] = useState(true) // Default to showing code
  const [workflowCode, setWorkflowCode] = useState<string>('')
  const [loading, setLoading] = useState(false)
  const [showCallbackForm, setShowCallbackForm] = useState(false)
  const [callbackResult, setCallbackResult] = useState('')
  const [callbackSubmitting, setCallbackSubmitting] = useState(false)
  const { workflowToStr, submitCallback, stopSchedule, resumeSchedule, tryRecover } = useWorkflowApi()

  const loadWorkflowCode = async () => {
    if (!data.ID || workflowCode) return // Already loaded or no ID
    
    setLoading(true)
    try {
      // Use the state ID with the /workflow-to-str-from-run/:id endpoint
      const code = await workflowToStr(data.ID)
      setWorkflowCode(code)
    } catch (error) {
      console.error('Failed to load workflow code for state:', error)
      setWorkflowCode('// Error loading workflow code from state run')
    } finally {
      setLoading(false)
    }
  }

  // Load workflow code automatically when component mounts (since we default to showing code)
  useEffect(() => {
    if (showCode && !workflowCode && !loading) {
      loadWorkflowCode()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []) // Only run on mount

  const toggleDisplay = () => {
    if (!showCode && !workflowCode) {
      loadWorkflowCode()
    }
    setShowCode(!showCode)
  }

  const handleCallbackSubmit = async (callbackID: string) => {
    if (!callbackResult.trim()) {
      alert('Please enter a callback result')
      return
    }

    setCallbackSubmitting(true)
    try {
      // Parse the result as JSON if possible, otherwise treat as string
      let parsedResult
      try {
        parsedResult = JSON.parse(callbackResult)
      } catch {
        // If JSON parsing fails, treat as string
        parsedResult = callbackResult
      }

      await submitCallback(callbackID, parsedResult)
      alert('Callback submitted successfully!')
      setShowCallbackForm(false)
      setCallbackResult('')
      // Optionally refresh the parent component or notify success
    } catch (error) {
      console.error('Failed to submit callback:', error)
      alert(`Failed to submit callback: ${error instanceof Error ? error.message : 'Unknown error'}`)
    } finally {
      setCallbackSubmitting(false)
    }
  }

  const handleStopSchedule = async (parentRunID: string) => {
    try {
      await stopSchedule(parentRunID)
      alert('Schedule paused successfully!')
      // Optionally refresh the parent component or notify success
    } catch (error) {
      console.error('Failed to stop schedule:', error)
      alert(`Failed to pause schedule: ${error instanceof Error ? error.message : 'Unknown error'}`)
    }
  }

  const handleResumeSchedule = async (parentRunID: string) => {
    try {
      await resumeSchedule(parentRunID)
      alert('Schedule resumed successfully!')
      // Optionally refresh the parent component or notify success
    } catch (error) {
      console.error('Failed to resume schedule:', error)
      alert(`Failed to resume schedule: ${error instanceof Error ? error.message : 'Unknown error'}`)
    }
  }

  const handleRetry = async (runID: string) => {
    try {
      await tryRecover(runID)
      alert('Recovery attempt initiated successfully!')
      // Optionally refresh the parent component or notify success
    } catch (error) {
      console.error('Failed to retry/recover state:', error)
      alert(`Failed to retry: ${error instanceof Error ? error.message : 'Unknown error'}`)
    }
  }

  if (!data.Data) {
    return <div>No state data available</div>
  }

  const state = data.Data

  // Helper function to render schema values in a readable format
  const renderSchemaValue = (value: any, key?: string): React.ReactNode => {
    if (!value) return <span className="text-muted-foreground">null</span>
    
    if (typeof value === 'string') {
      return <span className="text-green-600">"{value}"</span>
    }
    if (typeof value === 'number') {
      return <span className="text-blue-600">{value}</span>
    }
    if (typeof value === 'boolean') {
      return <span className="text-purple-600">{value.toString()}</span>
    }
    
    // Check if it's a schema type object
    if (value && typeof value === 'object' && value.$type) {
      // Handle Binary data as image
      if (value.$type === 'schema.Binary' && value['schema.Binary']) {
        // Try to detect image type from base64 data
        const base64Data = value['schema.Binary']
        let mimeType = 'image/jpeg' // default
        
        // Check for common image format signatures in base64
        if (base64Data.startsWith('iVBORw0KGgo')) {
          mimeType = 'image/png'
        } else if (base64Data.startsWith('R0lGOD')) {
          mimeType = 'image/gif'
        } else if (base64Data.startsWith('Qk')) {
          mimeType = 'image/bmp'
        } else if (base64Data.startsWith('UklGR')) {
          mimeType = 'image/webp'
        }
        
        return (
          <div className="space-y-1">
            <span className="text-xs text-muted-foreground">schema.Binary:</span>
            <div className="mt-2">
              <img 
                src={`data:${mimeType};base64,${base64Data}`}
                alt="Binary result"
                className="max-w-full h-auto rounded border border-border shadow-sm"
                style={{ maxHeight: '400px' }}
                onError={(e) => {
                  // Fallback to JPEG if image fails to load
                  const img = e.target as HTMLImageElement
                  if (img.src !== `data:image/jpeg;base64,${base64Data}`) {
                    img.src = `data:image/jpeg;base64,${base64Data}`
                  }
                }}
              />
            </div>
          </div>
        )
      }
      
      // Handle schema.Map type
      if (value.$type === 'schema.Map' && value['schema.Map']) {
        const mapValue = value['schema.Map']
        return (
          <div className="space-y-1">
            <span className="text-xs text-muted-foreground">{value.$type}:</span>
            <div className="ml-2 space-y-1">
              {Object.entries(mapValue).map(([mapKey, mapVal]) => (
                <div key={mapKey} className="flex items-start gap-2">
                  <span className="text-orange-600 font-mono text-xs">{mapKey}:</span>
                  <div className="flex-1">{renderSchemaValue(mapVal)}</div>
                </div>
              ))}
            </div>
          </div>
        )
      }
      
      // Handle other schema types
      const schemaType = value.$type
      const schemaValue = value[schemaType]
      
      if (schemaValue !== undefined) {
        // For primitive schema types, just show the value
        if (typeof schemaValue !== 'object') {
          return (
            <div className="inline-flex items-center gap-1">
              <span className="text-xs text-muted-foreground">{schemaType}:</span>
              {renderSchemaValue(schemaValue)}
            </div>
          )
        }
        
        // For complex schema types, show them in a nested structure
        return (
          <div className="space-y-1">
            <span className="text-xs text-muted-foreground">{schemaType}:</span>
            <div className="ml-2">{renderSchemaValue(schemaValue)}</div>
          </div>
        )
      }
    }
    
    // For plain objects without $type, check if all values are schema objects
    if (value && typeof value === 'object' && !Array.isArray(value)) {
      const entries = Object.entries(value)
      
      // If it looks like a map of schema values, render it as such
      if (entries.length > 0 && entries.every(([_, v]) => {
        return v && typeof v === 'object' && v !== null && '$type' in (v as Record<string, any>)
      })) {
        return (
          <div className="space-y-1">
            {entries.map(([objKey, objVal]) => (
              <div key={objKey} className="flex items-start gap-2">
                <span className="text-orange-600 font-mono text-xs">{objKey}:</span>
                <div className="flex-1">{renderSchemaValue(objVal)}</div>
              </div>
            ))}
          </div>
        )
      }
    }
    
    // For arrays and other objects, show a compact representation
    return (
      <details className="text-xs">
        <summary className="cursor-pointer text-muted-foreground hover:text-foreground">
          {key || 'object'} {'{...}'}
        </summary>
        <pre className="mt-1 ml-4 text-xs bg-muted/20 p-2 rounded overflow-x-auto">
          {JSON.stringify(value, null, 2)}
        </pre>
      </details>
    )
  }

  // Render base state information (common to all state types)
  const renderBaseState = (baseState: any) => {
    if (!baseState) return null
    
    return (
      <div className="space-y-2 border-l-2 border-muted pl-3">
        <div className="text-xs font-medium text-muted-foreground">Execution Context:</div>
        
        {baseState.RunID && (
          <div className="text-xs">
            <span className="text-muted-foreground">Run ID:</span> 
            <span className="font-mono ml-1">{baseState.RunID}</span>
          </div>
        )}
        
        {baseState.StepID && (
          <div className="text-xs">
            <span className="text-muted-foreground">Step ID:</span> 
            <span className="font-mono ml-1">{baseState.StepID}</span>
          </div>
        )}
        
        {baseState.Variables && Object.keys(baseState.Variables).length > 0 && (
          <div className="text-xs">
            <div className="text-muted-foreground mb-1">Variables:</div>
            <div className="space-y-1 ml-2">
              {Object.entries(baseState.Variables).map(([key, value]) => (
                <div key={key} className="flex items-start gap-2">
                  <span className="text-orange-600 font-mono">{key}:</span>
                  {renderSchemaValue(value, key)}
                </div>
              ))}
            </div>
          </div>
        )}
        
        {baseState.ExprResult && Object.keys(baseState.ExprResult).length > 0 && (
          <div className="text-xs">
            <div className="text-muted-foreground mb-1">Expression Results:</div>
            <div className="space-y-1 ml-2">
              {Object.entries(baseState.ExprResult).map(([key, value]) => (
                <div key={key} className="flex items-start gap-2">
                  <span className="text-cyan-600 font-mono">{key}:</span>
                  {renderSchemaValue(value, key)}
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    )
  }

  // Get quick actions for the header
  const getQuickActions = () => {
    if (!state.$type) return null

    const stateData = state[state.$type as keyof typeof state] as any
    const actions = []

    // Callback submission for Await states
    if (state.$type === 'workflow.Await' && stateData?.CallbackID) {
      actions.push(
        <Button
          key="submit-callback"
          variant="ghost"
          size="sm"
          onClick={() => {
            setCallbackResult(JSON.stringify({
              "$type": "schema.String",
              "schema.String": "Amigo"
            }, null, 2))
            setShowCallbackForm(true)
          }}
          className="h-6 w-6 p-0"
          title="Submit Callback"
        >
          <Send className="h-3 w-3" />
        </Button>
      )
    }

    // Retry for Error states
    if (state.$type === 'workflow.Error' && stateData?.BaseState?.RunID) {
      actions.push(
        <Button
          key="retry"
          variant="ghost"
          size="sm"
          onClick={() => handleRetry(stateData.BaseState.RunID)}
          className="h-6 w-6 p-0"
          title="Retry"
        >
          <RotateCcw className="h-3 w-3" />
        </Button>
      )
    }

    // Scheduling controls for Scheduled states
    if (state.$type === 'workflow.Scheduled' && stateData?.BaseState?.RunOption?.['workflow.ScheduleRun']?.ParentRunID) {
      const parentRunID = stateData.BaseState.RunOption['workflow.ScheduleRun'].ParentRunID
      actions.push(
        <Button
          key="pause-schedule"
          variant="ghost"
          size="sm"
          onClick={() => handleStopSchedule(parentRunID)}
          className="h-6 w-6 p-0"
          title="Pause Schedule"
        >
          <Pause className="h-3 w-3" />
        </Button>
      )
    }

    // Resume controls for ScheduleStopped states
    if (state.$type === 'workflow.ScheduleStopped' && stateData?.BaseState?.RunOption?.['workflow.ScheduleRun']?.ParentRunID) {
      const parentRunID = stateData.BaseState.RunOption['workflow.ScheduleRun'].ParentRunID
      actions.push(
        <Button
          key="resume-schedule"
          variant="ghost"
          size="sm"
          onClick={() => handleResumeSchedule(parentRunID)}
          className="h-6 w-6 p-0"
          title="Resume Schedule"
        >
          <Play className="h-3 w-3" />
        </Button>
      )
    }

    return actions
  }

  // Render detailed state information based on state type
  const renderStateDetails = () => {
    if (!state.$type) {
      return <div className="text-muted-foreground">Unknown state type</div>
    }

    const stateData = state[state.$type as keyof typeof state] as any

    return (
      <div className="space-y-3">
        {/* State-specific information */}
        <div className="space-y-2">
          {state.$type === 'workflow.Done' && (
            <div className="space-y-2">
              <div className="text-sm font-medium text-green-600">✓ Workflow Completed</div>
              {stateData?.Result && (
                <div>
                  <div className="text-xs text-muted-foreground mb-1">Result:</div>
                  <div className="ml-2 p-2 bg-muted/20 rounded">
                    {renderSchemaValue(stateData.Result, 'result')}
                  </div>
                </div>
              )}
            </div>
          )}
          
          {state.$type === 'workflow.Error' && (
            <div className="text-xs space-y-1">
              {stateData?.Code && (
                <div>
                  <span className="text-muted-foreground">Error Code:</span>
                  <span className="text-red-600 font-mono ml-1">{stateData.Code}</span>
                </div>
              )}
              {stateData?.Reason && (
                <div>
                  <span className="text-muted-foreground">Reason:</span>
                  <span className="text-red-600 ml-1">{stateData.Reason}</span>
                </div>
              )}
              {stateData?.Retried !== undefined && (
                <div>
                  <span className="text-muted-foreground">Retries:</span>
                  <span className="text-yellow-600 ml-1">{stateData.Retried}</span>
                </div>
              )}
            </div>
          )}
          
          {state.$type === 'workflow.Await' && (
            <div className="text-xs space-y-1">
              {stateData?.CallbackID && (
                <div>
                  <span className="text-muted-foreground">Callback ID:</span>
                  <span className="text-blue-600 font-mono ml-1">{stateData.CallbackID}</span>
                </div>
              )}
              {stateData?.ExpectedTimeoutTimestamp && (
                <div>
                  <span className="text-muted-foreground">Timeout:</span>
                  <span className="text-yellow-600 ml-1">
                    {new Date(stateData.ExpectedTimeoutTimestamp * 1000).toLocaleString()}
                  </span>
                </div>
              )}
            </div>
          )}
          
          {(state.$type === 'workflow.Scheduled' || state.$type === 'workflow.ScheduleStopped') && (
            <div className="text-xs space-y-1">
              {stateData?.ParentRunID && (
                <div>
                  <span className="text-muted-foreground">Parent Run:</span>
                  <span className="text-purple-600 font-mono ml-1">{stateData.ParentRunID}</span>
                </div>
              )}
              {stateData?.ScheduleID && (
                <div>
                  <span className="text-muted-foreground">Schedule ID:</span>
                  <span className="text-blue-600 font-mono ml-1">{stateData.ScheduleID}</span>
                </div>
              )}
            </div>
          )}
        </div>
        
        {/* Base state information */}
        {stateData?.BaseState && renderBaseState(stateData.BaseState)}
      </div>
    )
  }

  // Get inline summary for quick view
  const getInlineSummary = () => {
    if (!state.$type) return null
    
    const stateData = state[state.$type as keyof typeof state] as any
    
    switch (state.$type) {
      case 'workflow.Done':
        if (stateData?.Result) {
          return (
            <div className="flex items-center gap-1">
              <span className="text-xs text-green-600">✓ Result:</span>
              <ResultPreview result={stateData.Result} />
            </div>
          )
        }
        return <span className="text-xs text-green-600">✓ Completed</span>
        
      case 'workflow.Await':
        return stateData?.CallbackID ? (
          <span className="text-xs text-muted-foreground">
            Callback: <span className="font-mono">{stateData.CallbackID}</span>
          </span>
        ) : null
        
      case 'workflow.Error':
        return stateData?.Reason ? (
          <span className="text-xs text-red-600">
            {stateData.Reason}
          </span>
        ) : null
        
      case 'workflow.Scheduled':
      case 'workflow.ScheduleStopped':
        return stateData?.ParentRunID ? (
          <span className="text-xs text-muted-foreground">
            Parent: <span className="font-mono">{stateData.ParentRunID.slice(0, 8)}...</span>
          </span>
        ) : null
        
      default:
        return null
    }
  }

  const quickActions = getQuickActions()
  const inlineSummary = getInlineSummary()

  return (
    <div className="space-y-2">
      {/* Header with badges, actions, and toggle */}
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2 flex-1 min-w-0">
          <StatusBadge state={state} />
          <Badge variant="secondary" className="text-xs">
            {data.Type}
          </Badge>
          {/* Inline summary for quick info */}
          {inlineSummary && (
            <div className="truncate flex-1">
              {inlineSummary}
            </div>
          )}
        </div>
        
        <div className="flex items-center gap-1">
          {/* Quick action buttons */}
          {quickActions && quickActions.map(action => action)}
          
          {/* Toggle button */}
          <Button
            variant="ghost"
            size="sm"
            onClick={toggleDisplay}
            className="h-6 px-2 text-xs"
            disabled={loading}
          >
            {showCode ? (
              <>
                <FileText className="h-3 w-3 mr-1" />
                Details
              </>
            ) : (
              <>
                <Code className="h-3 w-3 mr-1" />
                Code
              </>
            )}
          </Button>
        </div>
      </div>

      {showCode ? (
        <div className="relative">
          <pre className="text-xs bg-muted/50 p-3 rounded border overflow-x-auto">
            <code className="language-flow text-foreground">
              {loading ? 'Loading workflow code...' : workflowCode || 'No code available'}
            </code>
          </pre>
        </div>
      ) : (
        renderStateDetails()
      )}

      {/* Enhanced Callback Submission Form */}
      {showCallbackForm && state.$type === 'workflow.Await' && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-background border rounded-lg p-4 max-w-md w-full mx-4 space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-medium">Submit Callback</h3>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  setShowCallbackForm(false)
                  setCallbackResult('')
                }}
                className="h-6 w-6 p-0"
              >
                ×
              </Button>
            </div>
            
            <div className="space-y-3">
              <div className="text-xs text-muted-foreground">
                Callback ID: <span className="font-mono">{(state as any)['workflow.Await']?.CallbackID}</span>
              </div>
              
              {/* Quick templates */}
              <div className="space-y-1">
                <label className="text-xs font-medium">Quick Templates:</label>
                <div className="flex gap-1 flex-wrap">
                  <Button
                    variant="outline"
                    size="sm"
                    className="h-6 px-2 text-xs"
                    onClick={() => setCallbackResult(JSON.stringify({
                      "$type": "schema.String",
                      "schema.String": "Amigo"
                    }, null, 2))}
                  >
                    String
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    className="h-6 px-2 text-xs"
                    onClick={() => setCallbackResult(JSON.stringify({
                      "$type": "schema.Number",
                      "schema.Number": 42
                    }, null, 2))}
                  >
                    Number
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    className="h-6 px-2 text-xs"
                    onClick={() => setCallbackResult(JSON.stringify({
                      "$type": "schema.Bool",
                      "schema.Bool": true
                    }, null, 2))}
                  >
                    Boolean
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    className="h-6 px-2 text-xs"
                    onClick={() => setCallbackResult(JSON.stringify({
                      "$type": "schema.Map",
                      "schema.Map": {
                        "key": { "schema.String": "value" }
                      }
                    }, null, 2))}
                  >
                    Map
                  </Button>
                </div>
              </div>
              
              <div>
                <label className="text-xs font-medium">Result (JSON or plain text):</label>
                <textarea
                  value={callbackResult}
                  onChange={(e) => setCallbackResult(e.target.value)}
                  placeholder={`Example:
{
  "$type": "schema.String",
  "schema.String": "Your value here"
}

Or just plain text: "Hello World"`}
                  className="w-full mt-1 p-2 text-xs font-mono border border-input rounded bg-background text-foreground resize-none"
                  rows={8}
                />
              </div>
              
              {/* JSON validation indicator */}
              {callbackResult && (
                <div className="text-xs">
                  {(() => {
                    try {
                      JSON.parse(callbackResult)
                      return <span className="text-green-600">✓ Valid JSON</span>
                    } catch {
                      return <span className="text-muted-foreground">Plain text (will be sent as-is)</span>
                    }
                  })()}
                </div>
              )}
            </div>
            
            <div className="flex gap-2 justify-end">
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  setShowCallbackForm(false)
                  setCallbackResult('')
                }}
                disabled={callbackSubmitting}
                className="h-7 text-xs"
              >
                Cancel
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => handleCallbackSubmit((state as any)['workflow.Await']?.CallbackID)}
                disabled={callbackSubmitting || !callbackResult.trim()}
                className="h-7 text-xs"
              >
                {callbackSubmitting ? 'Submitting...' : 'Submit'}
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}