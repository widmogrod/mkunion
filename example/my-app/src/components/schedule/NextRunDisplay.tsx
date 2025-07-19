import React from 'react'
import { Clock } from 'lucide-react'
import { cn } from '../../lib/utils'

interface NextRunDisplayProps {
  timestamp?: number
  className?: string
}

export function NextRunDisplay({ timestamp, className }: NextRunDisplayProps) {
  if (!timestamp) {
    return (
      <div className={cn('flex items-center gap-1 text-xs text-muted-foreground', className)}>
        <Clock className="h-3 w-3" />
        <span>No next run</span>
      </div>
    )
  }
  
  const nextRun = new Date(timestamp * 1000) // Convert from Unix timestamp
  const now = new Date()
  const diff = nextRun.getTime() - now.getTime()
  
  let relativeTime = ''
  let urgency = 'default'
  
  if (diff < 0) {
    relativeTime = 'Overdue'
    urgency = 'error'
  } else if (diff < 60000) {
    relativeTime = 'Less than 1 minute'
    urgency = 'warning'
  } else if (diff < 3600000) {
    const minutes = Math.floor(diff / 60000)
    relativeTime = `${minutes} minute${minutes > 1 ? 's' : ''}`
    urgency = minutes < 5 ? 'warning' : 'default'
  } else if (diff < 86400000) {
    const hours = Math.floor(diff / 3600000)
    relativeTime = `${hours} hour${hours > 1 ? 's' : ''}`
  } else {
    const days = Math.floor(diff / 86400000)
    relativeTime = `${days} day${days > 1 ? 's' : ''}`
  }
  
  const formattedTime = nextRun.toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
    hour12: true
  })
  
  return (
    <div 
      className={cn(
        'flex items-center gap-1 text-xs',
        urgency === 'error' && 'text-destructive',
        urgency === 'warning' && 'text-orange-600 dark:text-orange-400',
        urgency === 'default' && 'text-muted-foreground',
        className
      )}
      title={formattedTime}
    >
      <Clock className="h-3 w-3" />
      <span>{relativeTime}</span>
    </div>
  )
}