import React, { useMemo, useRef, useState } from 'react'
import { GlobalExecution } from '../../hooks/useGlobalExecutions'

interface TimelineScrubberProps {
  dateRange: { start: Date; end: Date }
  executions: GlobalExecution[]
  onDateRangeChange: (range: { start: Date; end: Date }) => void
}

export function TimelineScrubber({ dateRange, executions, onDateRangeChange }: TimelineScrubberProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const [isDragging, setIsDragging] = useState(false)
  
  // Group executions by hour for the timeline
  const hourlyData = useMemo(() => {
    const hours: Map<number, { successful: number; failed: number; total: number }> = new Map()
    
    // Initialize hours in range
    const startHour = new Date(dateRange.start)
    startHour.setMinutes(0, 0, 0)
    const endHour = new Date(dateRange.end)
    endHour.setMinutes(59, 59, 999)
    
    const currentHour = new Date(startHour)
    while (currentHour <= endHour) {
      hours.set(currentHour.getTime(), { successful: 0, failed: 0, total: 0 })
      currentHour.setHours(currentHour.getHours() + 1)
    }
    
    // Count executions per hour
    executions.forEach(execution => {
      const hour = new Date(execution.startTime)
      hour.setMinutes(0, 0, 0)
      const hourKey = hour.getTime()
      
      const data = hours.get(hourKey)
      if (data) {
        data.total++
        if (execution.status === 'done') {
          data.successful++
        } else if (execution.status === 'error') {
          data.failed++
        }
      }
    })
    
    return Array.from(hours.entries())
      .sort(([a], [b]) => a - b)
      .map(([timestamp, data]) => ({
        timestamp: new Date(timestamp),
        ...data
      }))
  }, [dateRange, executions])
  
  const maxCount = Math.max(...hourlyData.map(h => h.total), 1)
  
  const handleTimelineClick = (e: React.MouseEvent) => {
    if (!containerRef.current) return
    
    const rect = containerRef.current.getBoundingClientRect()
    const x = e.clientX - rect.left
    const percentage = x / rect.width
    
    // Calculate new center time based on click position
    const totalMs = dateRange.end.getTime() - dateRange.start.getTime()
    const clickMs = dateRange.start.getTime() + (totalMs * percentage)
    const clickTime = new Date(clickMs)
    
    // Create new range centered on click (maintaining current range size)
    const rangeSize = totalMs
    const newStart = new Date(clickTime.getTime() - rangeSize / 2)
    const newEnd = new Date(clickTime.getTime() + rangeSize / 2)
    
    onDateRangeChange({ start: newStart, end: newEnd })
  }
  
  return (
    <div className="bg-muted/30 border-b px-6 py-4">
      <div className="space-y-2">
        <div className="flex items-center justify-between text-sm">
          <span className="text-muted-foreground">Timeline</span>
          <span className="text-muted-foreground">
            {dateRange.start.toLocaleString()} - {dateRange.end.toLocaleString()}
          </span>
        </div>
        
        <div 
          ref={containerRef}
          className="relative h-16 bg-background rounded-lg border cursor-pointer overflow-hidden"
          onClick={handleTimelineClick}
        >
          {/* Hour bars */}
          <div className="absolute inset-0 flex">
            {hourlyData.map((hour, index) => {
              const heightPercentage = (hour.total / maxCount) * 100
              const failurePercentage = hour.total > 0 ? (hour.failed / hour.total) * 100 : 0
              
              return (
                <div
                  key={hour.timestamp.getTime()}
                  className="flex-1 relative group"
                  style={{ minWidth: '20px' }}
                >
                  {/* Background bar */}
                  <div
                    className="absolute bottom-0 left-0 right-0 bg-muted transition-all"
                    style={{ height: `${heightPercentage}%` }}
                  />
                  
                  {/* Success portion */}
                  {hour.successful > 0 && (
                    <div
                      className="absolute bottom-0 left-0 right-0 bg-green-500 transition-all"
                      style={{ 
                        height: `${(hour.successful / maxCount) * 100}%` 
                      }}
                    />
                  )}
                  
                  {/* Failure portion */}
                  {hour.failed > 0 && (
                    <div
                      className="absolute left-0 right-0 bg-red-500 transition-all"
                      style={{ 
                        bottom: `${(hour.successful / maxCount) * 100}%`,
                        height: `${(hour.failed / maxCount) * 100}%` 
                      }}
                    />
                  )}
                  
                  {/* Hover tooltip */}
                  <div className="opacity-0 group-hover:opacity-100 absolute bottom-full left-1/2 transform -translate-x-1/2 mb-2 pointer-events-none transition-opacity">
                    <div className="bg-popover text-popover-foreground text-xs rounded px-2 py-1 shadow-lg whitespace-nowrap">
                      <div className="font-medium">{hour.timestamp.toLocaleTimeString()}</div>
                      <div className="text-green-600">✓ {hour.successful}</div>
                      <div className="text-red-600">✗ {hour.failed}</div>
                    </div>
                  </div>
                  
                  {/* Hour marker for every 6th hour */}
                  {index % 6 === 0 && (
                    <div className="absolute bottom-0 left-0 text-xs text-muted-foreground transform -translate-y-5">
                      {hour.timestamp.getHours()}:00
                    </div>
                  )}
                </div>
              )
            })}
          </div>
          
          {/* Failure indicators */}
          {hourlyData.some(h => h.failed > 0) && (
            <div className="absolute top-1 right-2 flex items-center gap-2 text-xs">
              <div className="flex items-center gap-1">
                <div className="w-2 h-2 bg-green-500 rounded-sm" />
                <span className="text-muted-foreground">Success</span>
              </div>
              <div className="flex items-center gap-1">
                <div className="w-2 h-2 bg-red-500 rounded-sm" />
                <span className="text-muted-foreground">Failed</span>
              </div>
            </div>
          )}
        </div>
        
        <div className="text-xs text-muted-foreground text-center">
          Click on timeline to navigate • Bars show execution volume and status
        </div>
      </div>
    </div>
  )
}