import React, { useMemo } from 'react'
import { Calendar, momentLocalizer, View } from 'react-big-calendar'
import moment from 'moment'
import 'react-big-calendar/lib/css/react-big-calendar.css'
import './RunHistoryCalendar.css'
import { Schedule } from '../../hooks/useAllSchedules'
import { GlobalExecution } from '../../hooks/useGlobalExecutions'
import { ExecutionDetailsModal } from './ExecutionDetailsModal'

const localizer = momentLocalizer(moment)

interface GlobalCalendarViewProps {
  schedules: Schedule[]
  executions: GlobalExecution[]
  viewMode: 'normal' | 'incident'
  dateRange: { start: Date; end: Date }
  onDateRangeChange: (range: { start: Date; end: Date }) => void
}

interface CalendarEvent {
  id: string
  title: string
  start: Date
  end: Date
  resource: {
    execution: GlobalExecution
    schedule?: Schedule
  }
}

export function GlobalCalendarView({
  schedules,
  executions,
  viewMode,
  dateRange,
  onDateRangeChange
}: GlobalCalendarViewProps) {
  const [view, setView] = React.useState<View>(viewMode === 'incident' ? 'week' : 'month')
  const [date, setDate] = React.useState(dateRange.start)
  const [selectedExecution, setSelectedExecution] = React.useState<any>(null)
  const [showDetailsModal, setShowDetailsModal] = React.useState(false)
  
  // Create a map of parentRunId to schedule for quick lookup
  const scheduleMap = useMemo(() => {
    const map = new Map<string, Schedule>()
    schedules.forEach(schedule => {
      map.set(schedule.parentRunId, schedule)
    })
    return map
  }, [schedules])
  
  // Transform executions to calendar events
  const events: CalendarEvent[] = useMemo(() => {
    return executions.map(execution => {
      const schedule = scheduleMap.get(execution.parentRunId)
      const endTime = execution.endTime || new Date(execution.startTime.getTime() + 30 * 60 * 1000)
      
      const statusIcon = {
        done: '✓',
        error: '✗',
        running: '⟳',
        scheduled: '◷'
      }[execution.status]
      
      return {
        id: execution.id,
        title: `${statusIcon} ${execution.scheduleName || 'Unknown'}`,
        start: execution.startTime,
        end: endTime,
        resource: {
          execution,
          schedule
        }
      }
    })
  }, [executions, scheduleMap])
  
  // Custom event style getter
  const eventStyleGetter = (event: CalendarEvent) => {
    const { execution, schedule } = event.resource
    
    // Use schedule color as base, override with status color for failures
    let backgroundColor = schedule?.color || '#6b7280'
    let borderColor = backgroundColor
    
    if (execution.status === 'error') {
      backgroundColor = '#ef4444'
      borderColor = '#dc2626'
    } else if (execution.status === 'running') {
      backgroundColor = '#3b82f6'
      borderColor = '#2563eb'
    } else if (execution.status === 'done') {
      // Darken the schedule color slightly for success
      backgroundColor = schedule?.color || '#10b981'
    }
    
    // In incident mode, dim non-error events
    const opacity = viewMode === 'incident' && execution.status !== 'error' ? 0.5 : 1
    
    return {
      style: {
        backgroundColor,
        borderColor,
        color: 'white',
        border: 'none',
        borderRadius: '4px',
        fontSize: '11px',
        padding: '2px 4px',
        opacity
      }
    }
  }
  
  // Custom components for better styling
  const components = {
    event: ({ event }: { event: CalendarEvent }) => {
      const { execution } = event.resource
      const isError = execution.status === 'error'
      
      return (
        <div className={`flex items-center gap-1 px-1 h-full ${isError ? 'font-semibold' : ''}`}>
          <span className="text-xs truncate">{event.title}</span>
          {isError && execution.errorMessage && (
            <span className="text-xs opacity-75">!</span>
          )}
        </div>
      )
    },
    toolbar: (props: any) => (
      <div className="flex items-center justify-between mb-4 p-4 bg-muted/30 rounded-lg">
        <div className="flex items-center gap-2">
          <button
            onClick={() => props.onNavigate('PREV')}
            className="p-2 hover:bg-muted rounded transition-colors"
          >
            ←
          </button>
          <button
            onClick={() => props.onNavigate('TODAY')}
            className="px-3 py-1 text-sm hover:bg-muted rounded transition-colors"
          >
            Today
          </button>
          <button
            onClick={() => props.onNavigate('NEXT')}
            className="p-2 hover:bg-muted rounded transition-colors"
          >
            →
          </button>
        </div>
        
        <h2 className="text-lg font-semibold">
          {props.label}
        </h2>
        
        <div className="flex items-center gap-1">
          <button
            onClick={() => props.onView('month')}
            className={`px-3 py-1 text-sm rounded transition-colors ${
              props.view === 'month' ? 'bg-primary text-primary-foreground' : 'hover:bg-muted'
            }`}
          >
            Month
          </button>
          <button
            onClick={() => props.onView('week')}
            className={`px-3 py-1 text-sm rounded transition-colors ${
              props.view === 'week' ? 'bg-primary text-primary-foreground' : 'hover:bg-muted'
            }`}
          >
            Week
          </button>
          <button
            onClick={() => props.onView('day')}
            className={`px-3 py-1 text-sm rounded transition-colors ${
              props.view === 'day' ? 'bg-primary text-primary-foreground' : 'hover:bg-muted'
            }`}
          >
            Day
          </button>
          <button
            onClick={() => props.onView('agenda')}
            className={`px-3 py-1 text-sm rounded transition-colors ${
              props.view === 'agenda' ? 'bg-primary text-primary-foreground' : 'hover:bg-muted'
            }`}
          >
            Agenda
          </button>
        </div>
      </div>
    )
  }
  
  const handleSelectEvent = (event: CalendarEvent) => {
    const { execution } = event.resource
    // Transform to match the existing ExecutionDetailsModal interface
    setSelectedExecution({
      id: execution.id,
      startTime: execution.startTime,
      endTime: execution.endTime,
      status: execution.status,
      errorMessage: execution.errorMessage,
      duration: undefined, // Not available
      inputData: undefined, // Would need to fetch full state data
      outputData: undefined // Would need to fetch full state data
    })
    setShowDetailsModal(true)
  }
  
  const handleNavigate = (newDate: Date) => {
    setDate(newDate)
    // Update date range based on view
    let start: Date
    let end: Date
    
    switch (view) {
      case 'month':
        start = moment(newDate).startOf('month').toDate()
        end = moment(newDate).endOf('month').toDate()
        break
      case 'week':
        start = moment(newDate).startOf('week').toDate()
        end = moment(newDate).endOf('week').toDate()
        break
      case 'day':
        start = moment(newDate).startOf('day').toDate()
        end = moment(newDate).endOf('day').toDate()
        break
      case 'agenda':
        // For agenda view, show next 7 days
        start = moment(newDate).startOf('day').toDate()
        end = moment(newDate).add(7, 'days').endOf('day').toDate()
        break
      default:
        start = moment(newDate).startOf('month').toDate()
        end = moment(newDate).endOf('month').toDate()
    }
    
    onDateRangeChange({ start, end })
  }
  
  // Highlight days with failures in incident mode
  const dayPropGetter = (date: Date) => {
    if (viewMode === 'incident') {
      const hasFailure = events.some(event => 
        moment(event.start).isSame(date, 'day') && 
        event.resource.execution.status === 'error'
      )
      
      if (hasFailure) {
        return {
          className: 'has-failure',
          style: {
            backgroundColor: 'rgba(239, 68, 68, 0.1)'
          }
        }
      }
    }
    
    const hasExecution = events.some(event => 
      moment(event.start).isSame(date, 'day')
    )
    
    if (hasExecution) {
      return {
        className: 'has-execution'
      }
    }
    
    return {}
  }
  
  return (
    <>
      <div className="h-full relative">
        {/* Legend - positioned below the toolbar */}
        <div className="absolute top-16 right-4 z-10 bg-background/95 backdrop-blur-sm p-3 rounded-lg shadow-sm max-h-48 overflow-y-auto border">
          <h4 className="text-xs font-medium mb-2">Active Schedules</h4>
          <div className="space-y-1">
            {schedules.map(schedule => (
              <div key={schedule.parentRunId} className="flex items-center gap-2 text-xs">
                <div 
                  className="w-3 h-3 rounded" 
                  style={{ backgroundColor: schedule.color }}
                />
                <span className="truncate max-w-[150px]">{schedule.flowName}</span>
                {schedule.status === 'paused' && (
                  <span className="text-muted-foreground">(paused)</span>
                )}
              </div>
            ))}
          </div>
        </div>
        
        {/* Incident mode indicator */}
        {viewMode === 'incident' && (
          <div className="absolute top-0 left-0 z-10 bg-red-500 text-white px-3 py-1 rounded-lg text-sm font-medium">
            Incident Mode Active
          </div>
        )}
        
        <Calendar
          localizer={localizer}
          events={events}
          startAccessor="start"
          endAccessor="end"
          style={{ height: '100%' }}
          view={view}
          onView={setView}
          date={date}
          onNavigate={handleNavigate}
          onSelectEvent={handleSelectEvent}
          eventPropGetter={eventStyleGetter}
          dayPropGetter={dayPropGetter}
          components={components}
          popup
          showMultiDayTimes
        />
      </div>
      
      <ExecutionDetailsModal
        isOpen={showDetailsModal}
        onClose={() => setShowDetailsModal(false)}
        execution={selectedExecution}
      />
    </>
  )
}