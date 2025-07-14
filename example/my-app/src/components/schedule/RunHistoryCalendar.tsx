import React from 'react'
import { Calendar, momentLocalizer, View } from 'react-big-calendar'
import moment from 'moment'
import 'react-big-calendar/lib/css/react-big-calendar.css'
import './RunHistoryCalendar.css'
import { Badge } from '../ui/badge'

const localizer = momentLocalizer(moment)

interface RunExecution {
  id: string
  startTime: Date
  endTime?: Date
  status: 'scheduled' | 'running' | 'done' | 'error'
  duration?: number
  errorMessage?: string
  inputData?: any
  outputData?: any
}

interface RunHistoryCalendarProps {
  executions: RunExecution[]
  onExecutionClick?: (execution: RunExecution) => void
}

interface CalendarEvent {
  id: string
  title: string
  start: Date
  end: Date
  resource: RunExecution
}

export function RunHistoryCalendar({ executions, onExecutionClick }: RunHistoryCalendarProps) {
  const [view, setView] = React.useState<View>('month')
  const [date, setDate] = React.useState(new Date())

  // Transform executions to calendar events
  const events: CalendarEvent[] = executions.map(execution => {
    // Since we don't have end times, create a 30-minute event for visibility
    const endTime = execution.endTime || new Date(execution.startTime.getTime() + 30 * 60 * 1000)
    
    const statusIcon = {
      done: '✓',
      error: '✗',
      running: '⟳',
      scheduled: '◷'
    }[execution.status]

    return {
      id: execution.id,
      title: `${statusIcon} ${execution.status}`,
      start: execution.startTime,
      end: endTime,
      resource: execution
    }
  })

  // Custom event style getter
  const eventStyleGetter = (event: CalendarEvent) => {
    const colors = {
      done: { backgroundColor: '#10b981', borderColor: '#059669' },
      error: { backgroundColor: '#ef4444', borderColor: '#dc2626' },
      running: { backgroundColor: '#3b82f6', borderColor: '#2563eb' },
      scheduled: { backgroundColor: '#f59e0b', borderColor: '#d97706' }
    }

    const color = colors[event.resource.status]
    
    return {
      style: {
        ...color,
        color: 'white',
        border: 'none',
        borderRadius: '4px',
        fontSize: '12px',
        padding: '2px 4px'
      }
    }
  }

  // Custom components for better styling
  const components = {
    event: ({ event }: { event: CalendarEvent }) => (
      <div className="flex items-center gap-1 px-1 h-full">
        <span className="text-xs truncate">{event.title}</span>
      </div>
    ),
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
        </div>
      </div>
    )
  }

  const handleSelectEvent = (event: CalendarEvent) => {
    if (onExecutionClick) {
      onExecutionClick(event.resource)
    }
  }

  // Custom day prop getter for highlighting days with executions
  const dayPropGetter = (date: Date) => {
    const hasExecution = events.some(event => 
      moment(event.start).isSame(date, 'day')
    )
    
    if (hasExecution) {
      return {
        className: 'has-execution',
        style: {
          backgroundColor: 'var(--muted)'
        }
      }
    }
    return {}
  }

  return (
    <div className="h-[600px] relative">
      {/* Legend */}
      <div className="absolute top-0 right-0 z-10 bg-background/95 backdrop-blur-sm p-3 rounded-lg shadow-sm">
        <div className="flex items-center gap-4 text-xs">
          <div className="flex items-center gap-1">
            <div className="w-3 h-3 bg-green-500 rounded" />
            <span>Done</span>
          </div>
          <div className="flex items-center gap-1">
            <div className="w-3 h-3 bg-red-500 rounded" />
            <span>Error</span>
          </div>
          <div className="flex items-center gap-1">
            <div className="w-3 h-3 bg-blue-500 rounded" />
            <span>Running</span>
          </div>
          <div className="flex items-center gap-1">
            <div className="w-3 h-3 bg-yellow-500 rounded" />
            <span>Scheduled</span>
          </div>
        </div>
      </div>

      {/* Summary Stats */}
      {events.length > 0 && (
        <div className="absolute bottom-0 left-0 z-10 bg-background/95 backdrop-blur-sm p-3 rounded-lg shadow-sm">
          <div className="text-xs text-muted-foreground">
            Showing {events.length} executions
          </div>
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
        onNavigate={setDate}
        onSelectEvent={handleSelectEvent}
        eventPropGetter={eventStyleGetter}
        dayPropGetter={dayPropGetter}
        components={components}
        popup
        showMultiDayTimes
      />
    </div>
  )
}