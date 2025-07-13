import React, { useState, useEffect } from 'react'
import { Badge } from '../ui/badge'
import { Button } from '../ui/button'
import { Code, FileText } from 'lucide-react'
import { useWorkflowApi } from '../../hooks/use-workflow-api'
import * as workflow from '../../workflow/github_com_widmogrod_mkunion_x_workflow'
import * as schemaless from '../../workflow/github_com_widmogrod_mkunion_x_storage_schemaless'

interface WorkflowDisplayProps {
  data: schemaless.Record<workflow.Flow>
}

export function WorkflowDisplay({ data }: WorkflowDisplayProps) {
  const [showCode, setShowCode] = useState(true) // Default to showing code
  const [workflowCode, setWorkflowCode] = useState<string>('')
  const [loading, setLoading] = useState(false)
  const { workflowAstToStr } = useWorkflowApi()

  const loadWorkflowCode = async () => {
    if (!data.Data || workflowCode) return // Already loaded or no data
    
    setLoading(true)
    try {
      // Wrap the Flow in a Workflow union type, same as flowCreate does
      const workflowData = {
        $type: 'workflow.Flow',
        'workflow.Flow': data.Data
      }
      
      console.log('WorkflowDisplay: Sending workflow data to API:', workflowData)
      
      // Send the properly formatted workflow AST data to the API
      const code = await workflowAstToStr(workflowData)
      setWorkflowCode(code)
    } catch (error) {
      console.error('Failed to load workflow code:', error)
      setWorkflowCode('// Error loading workflow code from API')
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

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <Badge variant="secondary" className="text-xs">{data.Type}</Badge>
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