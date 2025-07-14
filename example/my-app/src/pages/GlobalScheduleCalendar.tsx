import React, { useState, useEffect } from 'react'
import { Calendar, AlertCircle } from 'lucide-react'
import { Button } from '../components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/card'
import { useAllSchedules } from '../hooks/useAllSchedules'
import { useGlobalExecutions } from '../hooks/useGlobalExecutions'
import { IncidentBanner } from '../components/schedule/IncidentBanner'
import { GlobalCalendarView } from '../components/schedule/GlobalCalendarView'
import { SmartFilters } from '../components/schedule/SmartFilters'
import { TimelineScrubber } from '../components/schedule/TimelineScrubber'
import { PageHeader } from '../components/layout/PageHeader'

type ViewMode = 'normal' | 'incident'

export function GlobalScheduleCalendar() {
  const [viewMode, setViewMode] = useState<ViewMode>('normal')
  const [dateRange, setDateRange] = useState(() => {
    const today = new Date()
    // Default to current month to show more relevant data
    const start = new Date(today.getFullYear(), today.getMonth(), 1)
    const end = new Date(today.getFullYear(), today.getMonth() + 1, 0)
    return { start, end }
  })
  const [statusFilter, setStatusFilter] = useState<string[]>(['all'])
  const [selectedSchedules, setSelectedSchedules] = useState<string[]>([])
  
  // Fetch all schedules
  const { schedules, loading: schedulesLoading } = useAllSchedules()
  
  // Fetch executions for the date range
  const { executions, failingSince, loading: executionsLoading } = useGlobalExecutions({
    dateRange,
    schedules: selectedSchedules.length > 0 ? selectedSchedules : schedules.map(s => s.parentRunId)
  })
  
  // Auto-detect incident mode
  useEffect(() => {
    const hasRecentFailures = executions.some(e => 
      e.status === 'error' && 
      new Date(e.startTime).getTime() > Date.now() - 6 * 60 * 60 * 1000 // Last 6 hours
    )
    
    if (hasRecentFailures && viewMode === 'normal') {
      setViewMode('incident')
      // Auto-filter to show only failures
      if (statusFilter.includes('all')) {
        setStatusFilter(['error'])
      }
    }
  }, [executions, viewMode, statusFilter])
  
  // Initialize selected schedules
  useEffect(() => {
    if (schedules.length > 0 && selectedSchedules.length === 0) {
      setSelectedSchedules(schedules.map(s => s.parentRunId))
    }
  }, [schedules, selectedSchedules.length])
  
  const loading = schedulesLoading || executionsLoading
  
  // Filter executions based on current filters
  const filteredExecutions = executions.filter(execution => {
    // Status filter
    if (!statusFilter.includes('all') && !statusFilter.includes(execution.status)) {
      return false
    }
    
    // Schedule filter
    if (selectedSchedules.length > 0 && !selectedSchedules.includes(execution.parentRunId)) {
      return false
    }
    
    return true
  })
  
  // Calculate summary statistics
  const stats = {
    total: filteredExecutions.length,
    failing: filteredExecutions.filter(e => e.status === 'error').length,
    successful: filteredExecutions.filter(e => e.status === 'done').length,
    running: filteredExecutions.filter(e => e.status === 'running').length,
    scheduled: filteredExecutions.filter(e => e.status === 'scheduled').length
  }
  
  const handleQuickFilter = (preset: string) => {
    const now = new Date()
    
    switch (preset) {
      case 'failures-only':
        setStatusFilter(['error'])
        break
      case 'last-6h':
        const sixHoursAgo = new Date(now.getTime() - 6 * 60 * 60 * 1000)
        setDateRange({ start: sixHoursAgo, end: now })
        break
      case 'last-24h':
        const dayAgo = new Date(now.getTime() - 24 * 60 * 60 * 1000)
        setDateRange({ start: dayAgo, end: now })
        break
      case 'last-week':
        const weekAgo = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000)
        setDateRange({ start: weekAgo, end: now })
        break
    }
  }

  return (
    <div className="h-full flex flex-col">
      {/* Incident Banner - Shows when failures detected */}
      {failingSince && stats.failing > 0 && (
        <IncidentBanner
          failingSince={failingSince}
          failureCount={stats.failing}
          onShowFailures={() => {
            setViewMode('incident')
            setStatusFilter(['error'])
          }}
        />
      )}
      
      {/* Header */}
      <PageHeader
        icon={Calendar}
        title="Operations Calendar"
        description="Monitor all scheduled workflows in one place"
        actions={
          <div className="flex items-center gap-2">
            <Button
              variant={viewMode === 'normal' ? 'default' : 'outline'}
              size="sm"
              onClick={() => {
                setViewMode('normal')
                // Reset filters to show all when switching to normal mode
                setStatusFilter(['all'])
              }}
            >
              Normal View
            </Button>
            <Button
              variant={viewMode === 'incident' ? 'default' : 'outline'}
              size="sm"
              onClick={() => setViewMode('incident')}
              className={viewMode === 'incident' ? 'bg-red-500 hover:bg-red-600' : ''}
            >
              <AlertCircle className="h-4 w-4 mr-1" />
              Incident Mode
            </Button>
          </div>
        }
      />
      
      {/* Summary Statistics */}
      <div className="grid grid-cols-5 gap-4 p-6 bg-muted/30">
        <Card className="bg-background">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Total Executions
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.total}</div>
          </CardContent>
        </Card>
        
        <Card className="bg-background">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Successful
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-600">{stats.successful}</div>
          </CardContent>
        </Card>
        
        <Card className="bg-background">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Failed
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-red-600">{stats.failing}</div>
          </CardContent>
        </Card>
        
        <Card className="bg-background">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Running
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-blue-600">{stats.running}</div>
          </CardContent>
        </Card>
        
        <Card className="bg-background">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Active Schedules
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{schedules.length}</div>
          </CardContent>
        </Card>
      </div>
      
      {/* Timeline Scrubber */}
      {viewMode === 'incident' && (
        <TimelineScrubber
          dateRange={dateRange}
          executions={filteredExecutions}
          onDateRangeChange={setDateRange}
        />
      )}
      
      {/* Filters */}
      <SmartFilters
        schedules={schedules}
        selectedSchedules={selectedSchedules}
        onScheduleChange={setSelectedSchedules}
        statusFilter={statusFilter}
        onStatusChange={setStatusFilter}
        onQuickFilter={handleQuickFilter}
        viewMode={viewMode}
      />
      
      {/* Main Calendar View */}
      <div className="flex-1 p-6">
        {loading ? (
          <div className="flex items-center justify-center h-full">
            <div className="text-center">
              <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary mx-auto mb-4"></div>
              <p className="text-muted-foreground">Loading schedules and executions...</p>
            </div>
          </div>
        ) : (
          <GlobalCalendarView
            schedules={schedules}
            executions={filteredExecutions}
            viewMode={viewMode}
            dateRange={dateRange}
            onDateRangeChange={setDateRange}
          />
        )}
      </div>
    </div>
  )
}