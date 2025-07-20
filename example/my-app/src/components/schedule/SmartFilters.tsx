import React from 'react'
import { Filter, Clock, CheckCircle, XCircle, Loader2 } from 'lucide-react'
import { Button } from '../ui/button'
import { Badge } from '../ui/badge'
import { Schedule } from '../../hooks/useAllSchedules'

interface SmartFiltersProps {
  schedules: Schedule[]
  selectedSchedules: string[]
  onScheduleChange: (schedules: string[]) => void
  statusFilter: string[]
  onStatusChange: (statuses: string[]) => void
  onQuickFilter: (preset: string) => void
  viewMode: 'normal' | 'incident'
}

export function SmartFilters({
  schedules,
  selectedSchedules,
  onScheduleChange,
  statusFilter,
  onStatusChange,
  onQuickFilter,
  viewMode
}: SmartFiltersProps) {
  const toggleSchedule = (parentRunId: string) => {
    if (selectedSchedules.includes(parentRunId)) {
      onScheduleChange(selectedSchedules.filter(id => id !== parentRunId))
    } else {
      onScheduleChange([...selectedSchedules, parentRunId])
    }
  }
  
  const toggleStatus = (status: string) => {
    if (status === 'all') {
      onStatusChange(['all'])
    } else {
      // Remove 'all' if selecting specific status
      const filtered = statusFilter.filter(s => s !== 'all')
      
      if (statusFilter.includes(status)) {
        const newFilter = filtered.filter(s => s !== status)
        onStatusChange(newFilter.length === 0 ? ['all'] : newFilter)
      } else {
        onStatusChange([...filtered, status])
      }
    }
  }
  
  const selectAllSchedules = () => {
    onScheduleChange(schedules.map(s => s.parentRunId))
  }
  
  const clearAllSchedules = () => {
    onScheduleChange([])
  }
  
  return (
    <div className="border-b bg-muted/10">
      <div className="p-4 space-y-4">
        {/* Quick Filters - Prominent in incident mode */}
        {viewMode === 'incident' && (
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-muted-foreground">Quick Filters:</span>
            <Button
              variant="outline"
              size="sm"
              onClick={() => onQuickFilter('failures-only')}
              className="border-red-200 text-red-600 hover:bg-red-50 dark:border-red-800 dark:hover:bg-red-950"
            >
              <XCircle className="h-3 w-3 mr-1" />
              Failures Only
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => onQuickFilter('last-6h')}
            >
              <Clock className="h-3 w-3 mr-1" />
              Last 6 Hours
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => onQuickFilter('last-24h')}
            >
              <Clock className="h-3 w-3 mr-1" />
              Last 24 Hours
            </Button>
          </div>
        )}
        
        {/* Status Filters */}
        <div className="flex items-center gap-2">
          <Filter className="h-4 w-4 text-muted-foreground" />
          <span className="text-sm font-medium text-muted-foreground">Status:</span>
          <div className="flex items-center gap-1">
            <Badge
              variant={statusFilter.includes('all') ? 'default' : 'outline'}
              className="cursor-pointer"
              onClick={() => toggleStatus('all')}
            >
              All
            </Badge>
            <Badge
              variant={statusFilter.includes('done') ? 'default' : 'outline'}
              className="cursor-pointer bg-green-500/10 text-green-700 dark:text-green-400"
              onClick={() => toggleStatus('done')}
            >
              <CheckCircle className="h-3 w-3 mr-1" />
              Done
            </Badge>
            <Badge
              variant={statusFilter.includes('error') ? 'default' : 'outline'}
              className="cursor-pointer bg-red-500/10 text-red-700 dark:text-red-400"
              onClick={() => toggleStatus('error')}
            >
              <XCircle className="h-3 w-3 mr-1" />
              Error
            </Badge>
            <Badge
              variant={statusFilter.includes('running') ? 'default' : 'outline'}
              className="cursor-pointer bg-blue-500/10 text-blue-700 dark:text-blue-400"
              onClick={() => toggleStatus('running')}
            >
              <Loader2 className="h-3 w-3 mr-1 animate-spin" />
              Running
            </Badge>
            <Badge
              variant={statusFilter.includes('scheduled') ? 'default' : 'outline'}
              className="cursor-pointer bg-yellow-500/10 text-yellow-700 dark:text-yellow-400"
              onClick={() => toggleStatus('scheduled')}
            >
              <Clock className="h-3 w-3 mr-1" />
              Scheduled
            </Badge>
          </div>
        </div>
        
        {/* Schedule Filters */}
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Filter className="h-4 w-4 text-muted-foreground" />
              <span className="text-sm font-medium text-muted-foreground">
                Schedules ({selectedSchedules.length}/{schedules.length}):
              </span>
            </div>
            <div className="flex items-center gap-1">
              <Button
                variant="ghost"
                size="sm"
                onClick={selectAllSchedules}
                className="h-7 text-xs"
              >
                Select All
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={clearAllSchedules}
                className="h-7 text-xs"
              >
                Clear All
              </Button>
            </div>
          </div>
          
          <div className="flex flex-wrap gap-2">
            {schedules.map(schedule => (
              <Badge
                key={schedule.parentRunId}
                variant={selectedSchedules.includes(schedule.parentRunId) ? 'default' : 'outline'}
                className="cursor-pointer"
                onClick={() => toggleSchedule(schedule.parentRunId)}
                style={{
                  '--schedule-color': schedule.color,
                  backgroundColor: selectedSchedules.includes(schedule.parentRunId) 
                    ? schedule.color 
                    : undefined,
                  borderColor: schedule.color,
                  color: selectedSchedules.includes(schedule.parentRunId) 
                    ? 'white' 
                    : schedule.color
                } as React.CSSProperties & { '--schedule-color': string }}
              >
                <div
                  className="w-2 h-2 rounded-full mr-1"
                  style={{ backgroundColor: schedule.color }}
                />
                {schedule.flowName}
                {schedule.status === 'paused' && (
                  <span className="ml-1 opacity-60">(paused)</span>
                )}
              </Badge>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}