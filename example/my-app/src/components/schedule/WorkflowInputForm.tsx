import React from 'react'
import { Label } from '../ui/label'
import { Input } from '../ui/input'
import { Textarea } from '../ui/textarea'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../ui/select'
import { Checkbox } from '../ui/checkbox'
import { StatusIcon } from '../ui/icons'
import * as schema from '../../workflow/github_com_widmogrod_mkunion_x_schema'
import * as builders from '../../workflows/builders'

interface WorkflowInputFormProps {
  workflow: { Name?: string; Arg?: string }
  value: schema.Schema | undefined
  onChange: (value: schema.Schema) => void
  error?: string
}

export function WorkflowInputForm({ workflow, value, onChange, error }: WorkflowInputFormProps) {
  // For now, we'll support common input types
  // In the future, this could be enhanced to parse workflow.Arg for more complex schemas
  
  const [inputType, setInputType] = React.useState<'string' | 'number' | 'boolean' | 'json'>('string')
  const [stringValue, setStringValue] = React.useState('')
  const [numberValue, setNumberValue] = React.useState('0')
  const [booleanValue, setBooleanValue] = React.useState(false)
  const [jsonValue, setJsonValue] = React.useState('{}')
  
  React.useEffect(() => {
    // Initialize from existing value
    if (value) {
      const type = value.$type
      if (type === 'schema.String') {
        setInputType('string')
        setStringValue((value as any)['schema.String'] || '')
      } else if (type === 'schema.Number') {
        setInputType('number')
        setNumberValue(String((value as any)['schema.Number'] || 0))
      } else if (type === 'schema.Bool') {
        setInputType('boolean')
        setBooleanValue((value as any)['schema.Bool'] || false)
      } else {
        setInputType('json')
        setJsonValue(JSON.stringify(value, null, 2))
      }
    }
  }, [value])
  
  const handleTypeChange = (newType: typeof inputType) => {
    setInputType(newType)
    
    // Update the value based on the new type
    switch (newType) {
      case 'string':
        onChange(builders.stringValue(stringValue))
        break
      case 'number':
        onChange(builders.numberValue(parseFloat(numberValue) || 0))
        break
      case 'boolean':
        onChange(builders.booleanValue(booleanValue))
        break
      case 'json':
        try {
          const parsed = JSON.parse(jsonValue)
          onChange(parsed)
        } catch {
          // Keep current value if JSON is invalid
        }
        break
    }
  }
  
  
  return (
    <div className="space-y-4">
      <div>
        <Label htmlFor="workflow-name">Workflow</Label>
        <div className="flex items-center gap-2 mt-1">
          <StatusIcon status="info" size="sm" />
          <span className="font-medium">{workflow.Name || 'Unknown'}</span>
          {workflow.Arg && (
            <span className="text-sm text-muted-foreground">
              (expects: {workflow.Arg})
            </span>
          )}
        </div>
      </div>
      
      <div className="space-y-2">
        <Label htmlFor="input-type">Input Type</Label>
        <Select value={inputType} onValueChange={(v: any) => handleTypeChange(v)}>
          <SelectTrigger id="input-type">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="string">String</SelectItem>
            <SelectItem value="number">Number</SelectItem>
            <SelectItem value="boolean">Boolean</SelectItem>
            <SelectItem value="json">JSON Object</SelectItem>
          </SelectContent>
        </Select>
      </div>
      
      <div className="space-y-2">
        <Label htmlFor="input-value">
          Input Value
          {inputType === 'json' && (
            <span className="text-xs text-muted-foreground ml-2">
              (must be valid JSON)
            </span>
          )}
        </Label>
        
        {inputType === 'string' && (
          <Input
            id="input-value"
            value={stringValue}
            onChange={(e) => {
              setStringValue(e.target.value)
              onChange(builders.stringValue(e.target.value))
            }}
            placeholder="Enter string value..."
          />
        )}
        
        {inputType === 'number' && (
          <Input
            id="input-value"
            type="number"
            value={numberValue}
            onChange={(e) => {
              setNumberValue(e.target.value)
              onChange(builders.numberValue(parseFloat(e.target.value) || 0))
            }}
            placeholder="Enter numeric value..."
          />
        )}
        
        {inputType === 'boolean' && (
          <div className="flex items-center space-x-2">
            <Checkbox
              id="input-value"
              checked={booleanValue}
              onCheckedChange={(checked) => {
                const value = checked === true
                setBooleanValue(value)
                onChange(builders.booleanValue(value))
              }}
            />
            <label
              htmlFor="input-value"
              className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
            >
              {booleanValue ? 'True' : 'False'}
            </label>
          </div>
        )}
        
        {inputType === 'json' && (
          <Textarea
            id="input-value"
            value={jsonValue}
            onChange={(e) => {
              setJsonValue(e.target.value)
              try {
                const parsed = JSON.parse(e.target.value)
                onChange(parsed)
              } catch {
                // Don't update if JSON is invalid
              }
            }}
            placeholder='{"key": "value"}'
            rows={6}
            className="font-mono text-sm"
          />
        )}
      </div>
      
      {error && (
        <div className="flex items-center gap-2 text-sm text-red-600">
          <StatusIcon status="error" size="xs" />
          {error}
        </div>
      )}
      
      <div className="text-sm text-muted-foreground">
        <p>This input will be passed to the workflow on each scheduled execution.</p>
        {workflow.Arg && (
          <p className="mt-1">
            The workflow expects a parameter named <code className="bg-muted px-1 py-0.5 rounded">{workflow.Arg}</code>
          </p>
        )}
      </div>
    </div>
  )
}