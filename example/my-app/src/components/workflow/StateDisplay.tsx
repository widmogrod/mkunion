import React, { useState, useEffect } from 'react'
import { Badge } from '../ui/badge'
import { Button } from '../ui/button'
import { Code, FileText } from 'lucide-react'
import { useWorkflowApi } from '../../hooks/use-workflow-api'
import { StatusBadge } from '../tables/StatusBadge'
import * as workflow from '../../workflow/github_com_widmogrod_mkunion_x_workflow'
import * as schemaless from '../../workflow/github_com_widmogrod_mkunion_x_storage_schemaless'

interface StateDisplayProps {
  data: schemaless.Record<workflow.State>
}

export function StateDisplay({ data }: StateDisplayProps) {
  const [showCode, setShowCode] = useState(true) // Default to showing code
  const [workflowCode, setWorkflowCode] = useState<string>('')
  const [loading, setLoading] = useState(false)
  const { workflowToStr } = useWorkflowApi()

  const loadWorkflowCode = async () => {
    if (!data.ID || workflowCode) return // Already loaded or no ID
    
    setLoading(true)
    try {
      console.log('StateDisplay: Loading workflow code for state ID:', data.ID)
      
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
  }, []) // Only run on mount

  const toggleDisplay = () => {
    if (!showCode && !workflowCode) {
      loadWorkflowCode()
    }
    setShowCode(!showCode)
  }

  if (!data.Data) {
    return <div>No state data available</div>
  }

  const state = data.Data

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <StatusBadge state={state} />
          <Badge variant="secondary" className="text-xs">
            {data.Type}
          </Badge>
        </div>
        <Button
          variant="ghost"
          size="sm"
          onClick={toggleDisplay}
          className="h-6 text-xs"
          disabled={loading}
        >
          {showCode ? (
            <>
              <FileText className="h-3 w-3 mr-1" />
              Show JSON
            </>
          ) : (
            <>
              <Code className="h-3 w-3 mr-1" />
              Show Code
            </>
          )}
        </Button>
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
        <pre className="text-xs overflow-x-auto">
          {JSON.stringify(data.Data, null, 2)}
        </pre>
      )}
    </div>
  )
}