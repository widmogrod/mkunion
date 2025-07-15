import React from 'react'
import { Routes, Route, Navigate, useLocation, useNavigate } from 'react-router-dom'
import { MainLayout } from './components/layout/main-layout'
import { DemosSection } from './components/demos/DemosSection'
import { WorkflowsPage } from './pages/WorkflowsPage'
import { ExecutionsPage } from './pages/ExecutionsPage'
import { SchedulesPage } from './pages/SchedulesPage'
import { GlobalScheduleCalendar } from './pages/GlobalScheduleCalendar'
import { SidebarProvider } from './components/layout/SidebarContext'
import { ThemeProvider } from './components/theme/ThemeContext'
import { ToastProvider } from './contexts/ToastContext'
import { useKeyboardShortcuts } from './hooks/useKeyboardShortcuts'

export default function App() {
  const location = useLocation()
  const navigate = useNavigate()
  
  // Enable keyboard shortcuts
  useKeyboardShortcuts()
  
  // Derive active tab from current route
  const getActiveTab = () => {
    const path = location.pathname
    if (path.startsWith('/workflows')) return 'workflows'
    if (path.startsWith('/states') || path.startsWith('/executions')) return 'executions'
    if (path.startsWith('/schedules')) return 'schedules'
    if (path.startsWith('/calendar')) return 'calendar'
    return 'workflows'
  }
  
  const activeTab = getActiveTab() as 'workflows' | 'executions' | 'schedules' | 'calendar'
  
  const handleTabChange = (tab: 'workflows' | 'executions' | 'schedules' | 'calendar') => {
    // Preserve query params when switching tabs, but clear view-specific filter parameters
    const searchParams = new URLSearchParams(location.search)
    
    // Clear filter parameters that are specific to different views
    // These parameters don't make sense when switching between different view types
    searchParams.delete('filter')      // workflows page filter
    searchParams.delete('id')          // workflows page specific item
    searchParams.delete('workflow')    // executions page workflow filter
    searchParams.delete('runId')       // executions page run filter
    searchParams.delete('status')      // executions page status filter
    searchParams.delete('schedule')    // executions page schedule filter
    searchParams.delete('parentRunId') // schedules page filter
    searchParams.delete('focus')       // schedules page focus
    
    navigate(`/${tab}?${searchParams.toString()}`)
  }

  return (
    <ThemeProvider>
      <ToastProvider>
        <SidebarProvider>
          <MainLayout 
            sidebar={<DemosSection />}
            activeTab={activeTab}
            onTabChange={handleTabChange}
          >
            <div className="h-full">
              <Routes>
                <Route path="/" element={<Navigate to="/workflows" replace />} />
                <Route path="/workflows" element={<WorkflowsPage />} />
                <Route path="/states" element={<Navigate to="/executions" replace />} />
                <Route path="/executions" element={<ExecutionsPage />} />
                <Route path="/schedules" element={<SchedulesPage />} />
                <Route path="/calendar" element={<GlobalScheduleCalendar />} />
                <Route path="*" element={<Navigate to="/workflows" replace />} />
              </Routes>
            </div>
          </MainLayout>
        </SidebarProvider>
      </ToastProvider>
    </ThemeProvider>
  )
}