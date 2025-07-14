import React from 'react'
import { Input } from '../ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../ui/select'
import { Info, Clock } from 'lucide-react'
import { Button } from '../ui/button'

interface CronExpressionBuilderProps {
  value: string
  onChange: (value: string) => void
}

const PRESET_SCHEDULES = [
  { label: 'Every minute', value: '* * * * * *' },
  { label: 'Every 5 minutes', value: '0 */5 * * * *' },
  { label: 'Every 10 minutes', value: '0 */10 * * * *' },
  { label: 'Every 30 minutes', value: '0 */30 * * * *' },
  { label: 'Every hour', value: '0 0 * * * *' },
  { label: 'Every day at midnight', value: '0 0 0 * * *' },
  { label: 'Every day at noon', value: '0 0 12 * * *' },
  { label: 'Every Monday at 9 AM', value: '0 0 9 * * 1' },
  { label: 'First day of month', value: '0 0 0 1 * *' },
]

export function CronExpressionBuilder({ value, onChange }: CronExpressionBuilderProps) {
  const [mode, setMode] = React.useState<'preset' | 'custom'>('preset')
  const [customExpression, setCustomExpression] = React.useState(value)
  const [nextRuns, setNextRuns] = React.useState<Date[]>([])
  const [parseError, setParseError] = React.useState<string | null>(null)

  // Calculate next run times based on cron expression
  React.useEffect(() => {
    try {
      const runs = calculateNextRuns(value, 5)
      setNextRuns(runs)
      setParseError(null)
    } catch (error) {
      setNextRuns([])
      setParseError('Invalid cron expression')
    }
  }, [value])

  const calculateNextRuns = (cronExpression: string, count: number): Date[] => {
    // This is a simplified implementation
    // In a real app, you'd use a proper cron parser library
    const runs: Date[] = []
    const now = new Date()
    
    // Simple parsing for demo purposes
    const parts = cronExpression.split(' ')
    if (parts.length !== 6) {
      throw new Error('Invalid cron expression format')
    }

    // For demonstration, we'll show approximate times
    if (cronExpression === '* * * * * *') {
      // Every minute
      for (let i = 1; i <= count; i++) {
        const next = new Date(now)
        next.setMinutes(now.getMinutes() + i)
        next.setSeconds(0)
        runs.push(next)
      }
    } else if (cronExpression.includes('*/5 * * * *')) {
      // Every 5 minutes
      for (let i = 1; i <= count; i++) {
        const next = new Date(now)
        next.setMinutes(now.getMinutes() + (i * 5))
        next.setSeconds(0)
        runs.push(next)
      }
    } else if (cronExpression.includes('*/10 * * * *')) {
      // Every 10 minutes
      for (let i = 1; i <= count; i++) {
        const next = new Date(now)
        next.setMinutes(now.getMinutes() + (i * 10))
        next.setSeconds(0)
        runs.push(next)
      }
    } else {
      // Default: show hourly for other patterns
      for (let i = 1; i <= count; i++) {
        const next = new Date(now)
        next.setHours(now.getHours() + i)
        next.setMinutes(0)
        next.setSeconds(0)
        runs.push(next)
      }
    }

    return runs
  }

  const handlePresetChange = (preset: string) => {
    onChange(preset)
    setCustomExpression(preset)
  }

  const handleCustomChange = (expression: string) => {
    setCustomExpression(expression)
    if (expression.trim()) {
      onChange(expression)
    }
  }

  const formatDateTime = (date: Date): string => {
    return date.toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: 'numeric',
      minute: '2-digit',
      hour12: true
    })
  }

  return (
    <div className="space-y-4">
      <div className="flex gap-2">
        <Button
          type="button"
          variant={mode === 'preset' ? 'default' : 'outline'}
          size="sm"
          onClick={() => setMode('preset')}
        >
          Presets
        </Button>
        <Button
          type="button"
          variant={mode === 'custom' ? 'default' : 'outline'}
          size="sm"
          onClick={() => setMode('custom')}
        >
          Custom
        </Button>
      </div>

      {mode === 'preset' ? (
        <Select value={value} onValueChange={handlePresetChange}>
          <SelectTrigger>
            <SelectValue placeholder="Select a schedule preset" />
          </SelectTrigger>
          <SelectContent>
            {PRESET_SCHEDULES.map((preset) => (
              <SelectItem key={preset.value} value={preset.value}>
                <div className="flex items-center justify-between w-full">
                  <span>{preset.label}</span>
                  <code className="text-xs text-muted-foreground ml-4">
                    {preset.value}
                  </code>
                </div>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      ) : (
        <div className="space-y-2">
          <Input
            value={customExpression}
            onChange={(e) => handleCustomChange(e.target.value)}
            placeholder="* * * * * * (seconds minutes hours day month weekday)"
            className={parseError ? 'border-destructive' : ''}
          />
          {parseError && (
            <p className="text-sm text-destructive">{parseError}</p>
          )}
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Info className="h-3 w-3" />
            <span>Format: seconds minutes hours day month weekday</span>
          </div>
        </div>
      )}

      <div className="bg-muted/50 rounded-lg p-4 space-y-2">
        <div className="flex items-center gap-2 text-sm font-medium">
          <Clock className="h-4 w-4" />
          <span>Next {nextRuns.length} runs:</span>
        </div>
        {nextRuns.length > 0 ? (
          <div className="space-y-1">
            {nextRuns.map((run, index) => (
              <div key={index} className="text-sm text-muted-foreground">
                {formatDateTime(run)}
              </div>
            ))}
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">
            {parseError || 'Unable to calculate next runs'}
          </p>
        )}
      </div>
    </div>
  )
}