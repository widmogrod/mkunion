import React from 'react'
import { AlertTriangle, ArrowRight, Clock } from 'lucide-react'
import { Button } from '../ui/button'
import './GlobalCalendar.css'

interface IncidentBannerProps {
  failingSince: Date
  failureCount: number
  onShowFailures: () => void
}

export function IncidentBanner({ failingSince, failureCount, onShowFailures }: IncidentBannerProps) {
  // Calculate time since first failure
  const timeSinceFailure = () => {
    const now = new Date()
    const diff = now.getTime() - failingSince.getTime()
    const minutes = Math.floor(diff / 60000)
    const hours = Math.floor(minutes / 60)
    const days = Math.floor(hours / 24)
    
    if (days > 0) {
      return `${days}d ${hours % 24}h`
    } else if (hours > 0) {
      return `${hours}h ${minutes % 60}m`
    } else {
      return `${minutes}m`
    }
  }
  
  const bannerClass = failureCount > 5 
    ? 'bg-red-600' // Critical - many failures
    : failureCount > 2 
    ? 'bg-red-500' // Warning - multiple failures
    : 'bg-orange-500' // Alert - few failures
  
  return (
    <div className={`${bannerClass} text-white px-6 py-3 flex items-center justify-between animate-pulse-subtle`}>
      <div className="flex items-center gap-4">
        <div className="flex items-center gap-2">
          <AlertTriangle className="h-5 w-5" />
          <span className="font-semibold text-lg">
            {failureCount} {failureCount === 1 ? 'Schedule' : 'Schedules'} Failing
          </span>
        </div>
        <div className="flex items-center gap-2 text-white/90">
          <Clock className="h-4 w-4" />
          <span>Since {failingSince.toLocaleTimeString()} ({timeSinceFailure()} ago)</span>
        </div>
      </div>
      
      <Button
        variant="secondary"
        size="sm"
        onClick={onShowFailures}
        className="bg-white/20 hover:bg-white/30 text-white border-white/20"
      >
        Show Failures
        <ArrowRight className="h-4 w-4 ml-1" />
      </Button>
    </div>
  )
}