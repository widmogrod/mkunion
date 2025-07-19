import React, { useState, useEffect } from 'react'
import ReactDOM from 'react-dom'
import { X } from 'lucide-react'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { Label } from '../ui/label'
import { analyzeWorkflowParams, WorkflowParam } from '../../utils/workflow-analyzer'
import { inferParamTypes } from '../../utils/type-inference'
import { useWorkflowApi } from '../../hooks/use-workflow-api'
import { useToast } from '../../contexts/ToastContext'
import { useRefreshStore } from '../../stores/refresh-store'
import * as workflow from '../../workflow/github_com_widmogrod_mkunion_x_workflow'
import * as schemaless from '../../workflow/github_com_widmogrod_mkunion_x_storage_schemaless'
import * as schema from '../../workflow/github_com_widmogrod_mkunion_x_schema'

interface RunWorkflowDialogProps {
  isOpen: boolean
  onClose: () => void
  workflow: schemaless.Record<workflow.Flow> | null
}

export function RunWorkflowDialog({ isOpen, onClose, workflow: workflowRecord }: RunWorkflowDialogProps) {
  const [parameters, setParameters] = useState<WorkflowParam[]>([])
  const [formData, setFormData] = useState<Record<string, any>>({})
  const [isRunning, setIsRunning] = useState(false)
  const { workflowRun } = useWorkflowApi()
  const toast = useToast()
  const { refreshAll } = useRefreshStore()
  
  useEffect(() => {
    if (!workflowRecord?.Data) return
    
    const flow = workflowRecord.Data
    
    // Analyze workflow parameters
    const params = analyzeWorkflowParams(flow)
    setParameters(params)
    
    // Infer types
    const typeMap = inferParamTypes(params, flow)
    
    // Initialize form data with default values
    const initialData: Record<string, any> = {}
    params.forEach(param => {
      const type = typeMap.get(param.path)
      const path = param.path.replace(flow.Arg + '.', '')
      
      // Set default values based on type
      switch (type) {
        case 'number':
          setNestedValue(initialData, path, 0)
          break
        case 'boolean':
          setNestedValue(initialData, path, false)
          break
        case 'array':
          setNestedValue(initialData, path, [])
          break
        case 'object':
          setNestedValue(initialData, path, {})
          break
        default:
          setNestedValue(initialData, path, '')
      }
    })
    
    setFormData(initialData)
  }, [workflowRecord])
  
  const setNestedValue = (obj: any, path: string, value: any) => {
    const parts = path.split('.')
    let current = obj
    
    for (let i = 0; i < parts.length - 1; i++) {
      if (!(parts[i] in current)) {
        current[parts[i]] = {}
      }
      current = current[parts[i]]
    }
    
    current[parts[parts.length - 1]] = value
  }
  
  const getNestedValue = (obj: any, path: string): any => {
    const parts = path.split('.')
    let current = obj
    
    for (const part of parts) {
      if (current && typeof current === 'object' && part in current) {
        current = current[part]
      } else {
        return undefined
      }
    }
    
    return current
  }
  
  const handleInputChange = (path: string, value: any) => {
    const newData = { ...formData }
    setNestedValue(newData, path, value)
    setFormData(newData)
  }
  
  const convertToSchema = (value: any): schema.Schema => {
    if (value === null || value === undefined) {
      return { $type: 'schema.None', 'schema.None': {} }
    }
    
    switch (typeof value) {
      case 'boolean':
        return { $type: 'schema.Bool', 'schema.Bool': value }
      case 'number':
        return { $type: 'schema.Number', 'schema.Number': value }
      case 'string':
        return { $type: 'schema.String', 'schema.String': value }
      case 'object':
        if (Array.isArray(value)) {
          return {
            $type: 'schema.List',
            'schema.List': value.map(convertToSchema)
          }
        } else {
          const schemaMap: Record<string, schema.Schema> = {}
          Object.entries(value).forEach(([k, v]) => {
            schemaMap[k] = convertToSchema(v)
          })
          return {
            $type: 'schema.Map',
            'schema.Map': schemaMap
          }
        }
      default:
        return { $type: 'schema.String', 'schema.String': String(value) }
    }
  }
  
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!workflowRecord?.Data || !workflowRecord.Data.Name) {
      toast.error('Invalid Workflow', 'Workflow name is missing')
      return
    }
    
    setIsRunning(true)
    
    try {
      // Convert form data to schema
      const inputSchema = convertToSchema(formData)
      
      // Run the workflow
      await workflowRun(workflowRecord.Data, inputSchema)
      
      toast.success('Workflow Started', `Successfully started workflow: ${workflowRecord.Data.Name}`)
      refreshAll() // Refresh tables to show new workflow state
      onClose()
    } catch (error) {
      console.error('Failed to run workflow:', error)
      toast.error('Run Failed', `Failed to run workflow: ${error instanceof Error ? error.message : 'Unknown error'}`)
    } finally {
      setIsRunning(false)
    }
  }
  
  const renderFormField = (param: WorkflowParam, typeMap: Map<string, string>) => {
    const flow = workflowRecord?.Data
    if (!flow) return null
    
    const fieldPath = param.path.replace(flow.Arg + '.', '')
    const type = typeMap.get(param.path) || 'string'
    const value = getNestedValue(formData, fieldPath)
    
    return (
      <div key={param.path} className="space-y-2">
        <Label htmlFor={param.path}>{fieldPath}</Label>
        {type === 'boolean' ? (
          <input
            id={param.path}
            type="checkbox"
            checked={value || false}
            onChange={(e) => handleInputChange(fieldPath, e.target.checked)}
            className="h-4 w-4"
          />
        ) : type === 'number' ? (
          <Input
            id={param.path}
            type="number"
            value={value || 0}
            onChange={(e) => handleInputChange(fieldPath, parseFloat(e.target.value) || 0)}
          />
        ) : (
          <Input
            id={param.path}
            type="text"
            value={value || ''}
            onChange={(e) => handleInputChange(fieldPath, e.target.value)}
            placeholder={`Enter ${type} value`}
          />
        )}
        {param.usageContext.length > 0 && (
          <p className="text-xs text-muted-foreground">
            Used in: {param.usageContext.map(c => c.details).join(', ')}
          </p>
        )}
      </div>
    )
  }
  
  if (!isOpen || !workflowRecord) return null
  
  const flow = workflowRecord.Data
  if (!flow) return null
  
  const typeMap = inferParamTypes(parameters, flow)
  
  return ReactDOM.createPortal(
    <div className="fixed inset-0 z-[9999] flex items-center justify-center">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} />
      
      {/* Dialog */}
      <div className="relative bg-background rounded-lg shadow-2xl w-full max-w-md max-h-[80vh] overflow-hidden z-[10000] border">
        <div className="flex items-center justify-between p-4 border-b">
          <h2 className="text-lg font-semibold">Run Workflow: {flow.Name}</h2>
          <button
            onClick={onClose}
            className="rounded-full p-1 hover:bg-muted transition-colors"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
        
        <form onSubmit={handleSubmit} className="p-4 space-y-4 overflow-y-auto max-h-[60vh]">
          {parameters.length === 0 ? (
            <p className="text-muted-foreground">This workflow doesn't require any input parameters.</p>
          ) : (
            <>
              <p className="text-sm text-muted-foreground">
                Enter the required parameters for this workflow:
              </p>
              {parameters.map(param => renderFormField(param, typeMap))}
            </>
          )}
          
          <div className="flex gap-2 pt-4">
            <Button type="submit" disabled={isRunning}>
              {isRunning ? 'Running...' : 'Run Workflow'}
            </Button>
            <Button type="button" variant="outline" onClick={onClose}>
              Cancel
            </Button>
          </div>
        </form>
      </div>
    </div>,
    document.body
  )
}