import React from 'react'
import { Routes, Route, Navigate, useLocation, useNavigate } from 'react-router-dom'
import { MainLayout } from './components/layout/main-layout'
import { DemosSection } from './components/demos/DemosSection'
import { WorkflowsPage } from './pages/WorkflowsPage'
import { ExecutionsPageSimple } from './pages/ExecutionsPageSimple'
import { SchedulesPage } from './pages/SchedulesPage'
import { GlobalScheduleCalendar } from './pages/GlobalScheduleCalendar'
import { SidebarProvider } from './components/layout/SidebarContext'
import { ThemeProvider } from './components/theme/ThemeContext'
import { ToastProvider } from './contexts/ToastContext'
import { ErrorBoundary } from './components/ErrorBoundary'
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
    // When switching tabs, preserve filters that make sense across views
    const searchParams = new URLSearchParams(location.search)
    
    // Define which params to keep for each tab
    const paramsToKeep: Record<string, string[]> = {
      workflows: ['workflow'], // Keep workflow filter when going to workflows
      executions: ['workflow', 'status', 'runId', 'schedule'], // Keep all filters
      schedules: ['workflow', 'parentRunId', 'focus'], // Keep relevant filters
      calendar: ['workflow', 'schedule', 'status', 'date', 'view'] // Keep calendar filters
    }
    
    // Get params to keep for the target tab
    const keepParams = paramsToKeep[tab] || []
    
    // Create new params with only the relevant ones
    const newParams = new URLSearchParams()
    keepParams.forEach(param => {
      const value = searchParams.get(param)
      if (value) {
        newParams.set(param, value)
      }
    })
    
    navigate(`/${tab}${newParams.toString() ? `?${newParams.toString()}` : ''}`)
  }

  return (
    <ErrorBoundary>
      <ThemeProvider>
        <ToastProvider>
          <SidebarProvider>
            <MainLayout 
              sidebar={<DemosSection />}
              activeTab={activeTab}
              onTabChange={handleTabChange}
            >
              <div className="h-full">
                <ErrorBoundary>
                  <Routes>
                    <Route path="/" element={<Navigate to="/workflows" replace />} />
                    <Route path="/workflows" element={<WorkflowsPage />} />
                    <Route path="/states" element={<Navigate to="/executions" replace />} />
                    <Route path="/executions" element={<ExecutionsPageSimple />} />
                    <Route path="/schedules" element={<SchedulesPage />} />
                    <Route path="/calendar" element={<GlobalScheduleCalendar />} />
                    <Route path="*" element={<Navigate to="/workflows" replace />} />
                  </Routes>
                </ErrorBoundary>
              </div>
            </MainLayout>
          </SidebarProvider>
        </ToastProvider>
      </ThemeProvider>
    </ErrorBoundary>
  )
}