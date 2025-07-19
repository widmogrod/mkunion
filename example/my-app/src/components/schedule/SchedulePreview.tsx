import React, { useEffect, useState } from 'react'
import { Clock, Calendar, ChevronRight } from 'lucide-react'
import { cn } from '../../lib/utils'

interface SchedulePreviewProps {
  cronExpression: string
  className?: string
}

interface NextRun {
  date: Date
  relativeTime: string
  dayOfWeek: string
  time: string
}

// Simple cron parser for common patterns
function calculateNextRuns(cronExpression: string, count: number = 5): NextRun[] {
  const runs: NextRun[] = []
  const now = new Date()
  
  // Parse cron parts (seconds minutes hours day month weekday)
  const parts = cronExpression.split(' ')
  if (parts.length !== 6) return []
  
  const [seconds, minutes, hours, , , weekday] = parts
  
  // Helper to add time to date
  const addTime = (date: Date, unit: 'seconds' | 'minutes' | 'hours', amount: number): Date => {
    const newDate = new Date(date)
    switch (unit) {
      case 'seconds':
        newDate.setSeconds(newDate.getSeconds() + amount)
        break
      case 'minutes':
        newDate.setMinutes(newDate.getMinutes() + amount)
        break
      case 'hours':
        newDate.setHours(newDate.getHours() + amount)
        break
    }
    return newDate
  }
  
  // Simple pattern matching for common cases
  let interval = 0
  let unit: 'seconds' | 'minutes' | 'hours' = 'minutes'
  
  if (seconds === '*' && minutes === '*' && hours === '*') {
    // Every second
    interval = 1
    unit = 'seconds'
  } else if (seconds.startsWith('*/')) {
    // Every N seconds
    interval = parseInt(seconds.slice(2))
    unit = 'seconds'
  } else if (minutes.startsWith('*/')) {
    // Every N minutes
    interval = parseInt(minutes.slice(2))
    unit = 'minutes'
  } else if (hours.startsWith('*/')) {
    // Every N hours
    interval = parseInt(hours.slice(2))
    unit = 'hours'
  } else if (minutes === '0' && hours === '0' && seconds === '0') {
    // Daily at midnight
    interval = 24
    unit = 'hours'
  } else if (seconds === '0' && minutes === '0' && hours !== '*') {
    // Daily at specific hour
    const targetHour = parseInt(hours)
    const nextRun = new Date(now)
    nextRun.setHours(targetHour, 0, 0, 0)
    if (nextRun <= now) {
      nextRun.setDate(nextRun.getDate() + 1)
    }
    
    for (let i = 0; i < count; i++) {
      const runDate = new Date(nextRun)
      runDate.setDate(runDate.getDate() + i)
      runs.push(formatNextRun(runDate))
    }
    return runs
  } else if (weekday !== '*' && seconds === '0') {
    // Weekly on specific day
    const targetDay = parseInt(weekday)
    const targetHour = parseInt(hours) || 0
    const targetMinute = parseInt(minutes) || 0
    
    const nextRun = new Date(now)
    nextRun.setHours(targetHour, targetMinute, 0, 0)
    
    // Find next occurrence of target day
    const daysUntilTarget = (targetDay - nextRun.getDay() + 7) % 7
    if (daysUntilTarget === 0 && nextRun <= now) {
      nextRun.setDate(nextRun.getDate() + 7)
    } else if (daysUntilTarget > 0) {
      nextRun.setDate(nextRun.getDate() + daysUntilTarget)
    }
    
    for (let i = 0; i < count; i++) {
      const runDate = new Date(nextRun)
      runDate.setDate(runDate.getDate() + (i * 7))
      runs.push(formatNextRun(runDate))
    }
    return runs
  }
  
  // Generate runs based on interval
  if (interval > 0) {
    let nextRun = new Date(now)
    for (let i = 0; i < count; i++) {
      nextRun = addTime(nextRun, unit, interval)
      runs.push(formatNextRun(nextRun))
    }
  }
  
  return runs
}

function formatNextRun(date: Date): NextRun {
  const now = new Date()
  const diff = date.getTime() - now.getTime()
  
  let relativeTime = ''
  if (diff < 60000) {
    relativeTime = 'in less than a minute'
  } else if (diff < 3600000) {
    const minutes = Math.floor(diff / 60000)
    relativeTime = `in ${minutes} minute${minutes > 1 ? 's' : ''}`
  } else if (diff < 86400000) {
    const hours = Math.floor(diff / 3600000)
    relativeTime = `in ${hours} hour${hours > 1 ? 's' : ''}`
  } else {
    const days = Math.floor(diff / 86400000)
    relativeTime = `in ${days} day${days > 1 ? 's' : ''}`
  }
  
  const dayOfWeek = date.toLocaleDateString('en-US', { weekday: 'short' })
  const time = date.toLocaleTimeString('en-US', { 
    hour: 'numeric', 
    minute: '2-digit',
    hour12: true 
  })
  
  return { date, relativeTime, dayOfWeek, time }
}

export function SchedulePreview({ cronExpression, className }: SchedulePreviewProps) {
  const [nextRuns, setNextRuns] = useState<NextRun[]>([])
  const [expanded, setExpanded] = useState(false)
  
  useEffect(() => {
    const runs = calculateNextRuns(cronExpression, 10)
    setNextRuns(runs)
    
    // Update every minute to keep relative times fresh
    const interval = setInterval(() => {
      const runs = calculateNextRuns(cronExpression, 10)
      setNextRuns(runs)
    }, 60000)
    
    return () => clearInterval(interval)
  }, [cronExpression])
  
  const displayedRuns = expanded ? nextRuns : nextRuns.slice(0, 3)
  
  if (nextRuns.length === 0) {
    return (
      <div className={cn('space-y-2 p-3 bg-muted/50 rounded-lg', className)}>
        <div className='flex items-center gap-2 text-sm text-muted-foreground'>
          <Clock className='h-4 w-4' />
          <span>Unable to calculate schedule</span>
        </div>
      </div>
    )
  }
  
  return (
    <div className={cn('space-y-2 p-3 bg-muted/50 rounded-lg', className)}>
      <div className='flex items-center justify-between'>
        <div className='flex items-center gap-2 text-sm font-medium'>
          <Calendar className='h-4 w-4' />
          <span>Upcoming runs</span>
        </div>
        {nextRuns.length > 3 && (
          <button
            type='button'
            onClick={() => setExpanded(!expanded)}
            className='text-xs text-muted-foreground hover:text-foreground transition-colors'
          >
            {expanded ? 'Show less' : `Show all (${nextRuns.length})`}
          </button>
        )}
      </div>
      
      <div className='space-y-2'>
        {displayedRuns.map((run, index) => (
          <div 
            key={index} 
            className={cn(
              'flex items-center justify-between text-sm p-2 rounded',
              index === 0 && 'bg-primary/10 border border-primary/20'
            )}
          >
            <div className='flex items-center gap-3'>
              <div className='flex items-center gap-1'>
                <Clock className='h-3 w-3 text-muted-foreground' />
                <span className={cn(
                  'font-medium',
                  index === 0 && 'text-primary'
                )}>
                  {run.relativeTime}
                </span>
              </div>
              {index === 0 && (
                <ChevronRight className='h-3 w-3 text-muted-foreground' />
              )}
            </div>
            <div className='text-muted-foreground text-xs'>
              {run.dayOfWeek}, {run.time}
            </div>
          </div>
        ))}
      </div>
      
      {!expanded && nextRuns.length > 3 && (
        <div className='text-center text-xs text-muted-foreground pt-1'>
          and {nextRuns.length - 3} more...
        </div>
      )}
    </div>
  )
}