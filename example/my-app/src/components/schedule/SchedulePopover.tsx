import React, { useState } from 'react'
import { Popover, PopoverContent, PopoverTrigger } from '../ui/popover'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { Label } from '../ui/label'
import { Calendar, Sparkles, ChevronRight } from 'lucide-react'
import { useWorkflowApi } from '../../hooks/use-workflow-api'
import { useToast } from '../../contexts/ToastContext'
import { useRefreshStore } from '../../stores/refresh-store'
import { SchedulePreview } from './SchedulePreview'
import { WorkflowInputForm } from './WorkflowInputForm'
import * as workflow from '../../workflow/github_com_widmogrod_mkunion_x_workflow'
import * as schemaless from '../../workflow/github_com_widmogrod_mkunion_x_storage_schemaless'
import * as schema from '../../workflow/github_com_widmogrod_mkunion_x_schema'
import * as builders from '../../workflows/builders'

interface SchedulePopoverProps {
  workflow: schemaless.Record<workflow.Flow>
  children: React.ReactNode
}

// Natural language patterns to cron expression mapping
const NATURAL_LANGUAGE_PATTERNS: { [key: string]: { cron: string; description: string } } = {
  'every minute': { cron: '* * * * * *', description: 'Runs every minute' },
  'every 5 minutes': { cron: '0 */5 * * * *', description: 'Runs every 5 minutes' },
  'every 10 minutes': { cron: '0 */10 * * * *', description: 'Runs every 10 minutes' },
  'every 30 minutes': { cron: '0 */30 * * * *', description: 'Runs every 30 minutes' },
  'every hour': { cron: '0 0 * * * *', description: 'Runs at the start of every hour' },
  'hourly': { cron: '0 0 * * * *', description: 'Runs at the start of every hour' },
  'daily': { cron: '0 0 0 * * *', description: 'Runs daily at midnight' },
  'every day': { cron: '0 0 0 * * *', description: 'Runs daily at midnight' },
  'weekly': { cron: '0 0 0 * * 0', description: 'Runs weekly on Sunday at midnight' },
  'every week': { cron: '0 0 0 * * 0', description: 'Runs weekly on Sunday at midnight' },
  'monthly': { cron: '0 0 0 1 * *', description: 'Runs monthly on the 1st at midnight' },
  'every month': { cron: '0 0 0 1 * *', description: 'Runs monthly on the 1st at midnight' },
}

// Parse natural language with time specifications
export function parseNaturalLanguage(input: string): { cron: string; description: string } | null {
  const normalized = input.toLowerCase().trim()
  
  // Check direct patterns first
  if (NATURAL_LANGUAGE_PATTERNS[normalized]) {
    return NATURAL_LANGUAGE_PATTERNS[normalized]
  }
  
  // Parse "every X minutes/hours"
  const everyMatch = normalized.match(/every\s+(\d+)\s+(minutes?|hours?|seconds?)/)
  if (everyMatch) {
    const [, amount, unit] = everyMatch
    const num = parseInt(amount)
    
    if (unit.startsWith('second')) {
      return { cron: `*/${num} * * * * *`, description: `Runs every ${num} seconds` }
    } else if (unit.startsWith('minute')) {
      return { cron: `0 */${num} * * * *`, description: `Runs every ${num} minutes` }
    } else if (unit.startsWith('hour')) {
      return { cron: `0 0 */${num} * * *`, description: `Runs every ${num} hours` }
    }
  }
  
  // Parse "daily at X"
  const dailyAtMatch = normalized.match(/daily at (\d{1,2}):?(\d{2})?\s*(am|pm)?/)
  if (dailyAtMatch) {
    const [, hourStr, minuteStr = '00', ampm] = dailyAtMatch
    let hour = parseInt(hourStr)
    const minute = parseInt(minuteStr)
    
    if (ampm === 'pm' && hour !== 12) hour += 12
    if (ampm === 'am' && hour === 12) hour = 0
    
    return { 
      cron: `0 ${minute} ${hour} * * *`, 
      description: `Runs daily at ${hour.toString().padStart(2, '0')}:${minute.toString().padStart(2, '0')}` 
    }
  }
  
  // Parse "every Monday/Tuesday/etc at X"
  const weekdayMatch = normalized.match(/(every )?(monday|tuesday|wednesday|thursday|friday|saturday|sunday)s? at (\d{1,2}):?(\d{2})?\s*(am|pm)?/)
  if (weekdayMatch) {
    const [, , day, hourStr, minuteStr = '00', ampm] = weekdayMatch
    let hour = parseInt(hourStr)
    const minute = parseInt(minuteStr)
    
    if (ampm === 'pm' && hour !== 12) hour += 12
    if (ampm === 'am' && hour === 12) hour = 0
    
    const dayMap: { [key: string]: number } = {
      'sunday': 0, 'monday': 1, 'tuesday': 2, 'wednesday': 3,
      'thursday': 4, 'friday': 5, 'saturday': 6
    }
    
    return {
      cron: `0 ${minute} ${hour} * * ${dayMap[day]}`,
      description: `Runs every ${day.charAt(0).toUpperCase() + day.slice(1)} at ${hour.toString().padStart(2, '0')}:${minute.toString().padStart(2, '0')}`
    }
  }
  
  return null
}

export function SchedulePopover({ workflow, children }: SchedulePopoverProps) {
  const [open, setOpen] = useState(false)
  const [naturalLanguageInput, setNaturalLanguageInput] = useState('')
  const [cronExpression, setCronExpression] = useState('')
  const [parsedDescription, setParsedDescription] = useState('')
  const [mode, setMode] = useState<'natural' | 'advanced'>('natural')
  const [workflowInput, setWorkflowInput] = useState<schema.Schema | undefined>(undefined)
  const [inputError, setInputError] = useState<string | undefined>(undefined)
  const [isCreating, setIsCreating] = useState(false)
  
  const { runCommand } = useWorkflowApi()
  const toast = useToast()
  const { refreshAll } = useRefreshStore()
  
  const handleNaturalLanguageChange = (value: string) => {
    setNaturalLanguageInput(value)
    
    const parsed = parseNaturalLanguage(value)
    if (parsed) {
      setCronExpression(parsed.cron)
      setParsedDescription(parsed.description)
    } else {
      setParsedDescription('')
    }
  }
  
  const handleSchedule = async () => {
    if (!cronExpression) {
      toast.error('Invalid Schedule', 'Please enter a valid schedule expression')
      return
    }
    
    if (!workflow.Data) {
      toast.error('Invalid Workflow', 'Workflow data is missing')
      return
    }
    
    if (inputError) {
      toast.error('Invalid Input', inputError)
      return
    }
    
    try {
      setIsCreating(true)
      
      // Use workflowInput or default to empty string
      const inputSchema = workflowInput || builders.stringValue('')
      
      const cmd = builders.createScheduledRunCommand(
        workflow.Data,
        inputSchema,
        cronExpression,
        `schedule_${workflow.Data.Name}_${Date.now()}`
      )
      
      await runCommand(cmd)
      toast.success('Schedule Created', `${workflow.Data.Name} scheduled successfully!`)
      refreshAll() // Refresh all tables including schedules
      setOpen(false)
      
      // Reset form
      setNaturalLanguageInput('')
      setCronExpression('')
      setParsedDescription('')
      setWorkflowInput(undefined)
      setInputError(undefined)
    } catch (error) {
      console.error('Failed to create schedule:', error)
      toast.error('Scheduling Failed', error instanceof Error ? error.message : 'Unknown error')
    } finally {
      setIsCreating(false)
    }
  }
  
  const commonPresets = [
    { label: 'Every 5 minutes', value: 'every 5 minutes' },
    { label: 'Hourly', value: 'hourly' },
    { label: 'Daily at 9am', value: 'daily at 9am' },
    { label: 'Every Monday at 8am', value: 'every monday at 8am' },
  ]
  
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        {children}
      </PopoverTrigger>
      <PopoverContent className="w-96 p-0" align="start">
        <div className="p-4 border-b">
          <h3 className="font-semibold flex items-center gap-2">
            <Calendar className="h-4 w-4" />
            Schedule {workflow.Data?.Name || 'Workflow'}
          </h3>
        </div>
        
        <div className="p-4 space-y-4">
          {mode === 'natural' ? (
            <>
              <div className="space-y-2">
                <Label htmlFor="natural-input" className="flex items-center gap-2">
                  <Sparkles className="h-3 w-3" />
                  When should this run?
                </Label>
                <Input
                  id="natural-input"
                  placeholder="e.g., every 5 minutes, daily at 9am..."
                  value={naturalLanguageInput}
                  onChange={(e) => handleNaturalLanguageChange(e.target.value)}
                  className="font-mono text-sm"
                />
                {parsedDescription && (
                  <p className="text-sm text-green-600 dark:text-green-400">
                    ✓ {parsedDescription}
                  </p>
                )}
              </div>
              
              <div className="space-y-2">
                <p className="text-xs text-muted-foreground">Quick presets:</p>
                <div className="flex flex-wrap gap-2">
                  {commonPresets.map((preset) => (
                    <Button
                      key={preset.value}
                      variant="outline"
                      size="sm"
                      className="text-xs"
                      onClick={() => handleNaturalLanguageChange(preset.value)}
                    >
                      {preset.label}
                    </Button>
                  ))}
                </div>
              </div>
            </>
          ) : (
            <div className="space-y-2">
              <Label htmlFor="cron-input">Cron Expression</Label>
              <Input
                id="cron-input"
                placeholder="* * * * * * (seconds minutes hours...)"
                value={cronExpression}
                onChange={(e) => setCronExpression(e.target.value)}
                className="font-mono text-sm"
              />
              <p className="text-xs text-muted-foreground">
                Format: seconds minutes hours day month weekday
              </p>
            </div>
          )}
          
          {workflow.Data && (
            <div className="space-y-2">
              <Label>Workflow Input</Label>
              <WorkflowInputForm
                workflow={workflow.Data}
                value={workflowInput}
                onChange={(value) => {
                  setWorkflowInput(value)
                  setInputError(undefined)
                }}
                error={inputError}
              />
            </div>
          )}
          
          {cronExpression && (
            <SchedulePreview cronExpression={cronExpression} />
          )}
          
          <div className="flex items-center justify-between pt-2">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setMode(mode === 'natural' ? 'advanced' : 'natural')}
              className="text-xs"
            >
              {mode === 'natural' ? 'Advanced mode' : 'Simple mode'}
              <ChevronRight className="h-3 w-3 ml-1" />
            </Button>
            
            <div className="flex gap-2">
              <Button variant="outline" size="sm" onClick={() => setOpen(false)}>
                Cancel
              </Button>
              <Button
                size="sm"
                onClick={handleSchedule}
                disabled={!cronExpression || isCreating}
              >
                {isCreating ? 'Creating...' : 'Schedule'}
              </Button>
            </div>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  )
}