import React from 'react'
import { MainLayout } from './components/layout/main-layout'
import { DemosSection } from './components/demos/DemosSection'
import { WorkflowsPage } from './pages/WorkflowsPage'
import { StatesPage } from './pages/StatesPage'
import { SchedulesPage } from './pages/SchedulesPage'
import { GlobalScheduleCalendar } from './pages/GlobalScheduleCalendar'
import { SidebarProvider } from './components/layout/SidebarContext'
import { ThemeProvider } from './components/theme/ThemeContext'
import { ToastProvider } from './contexts/ToastContext'

export default function App() {
  const [activeTab, setActiveTab] = React.useState<'workflows' | 'states' | 'schedules' | 'calendar'>('workflows')

  return (
    <ThemeProvider>
      <ToastProvider>
        <SidebarProvider>
          <MainLayout 
            sidebar={<DemosSection />}
            activeTab={activeTab}
            onTabChange={setActiveTab}
          >
            <div className="h-full">
              {activeTab === 'workflows' ? (
                <WorkflowsPage />
              ) : activeTab === 'states' ? (
                <StatesPage />
              ) : activeTab === 'schedules' ? (
                <SchedulesPage />
              ) : (
                <GlobalScheduleCalendar />
              )}
            </div>
          </MainLayout>
        </SidebarProvider>
      </ToastProvider>
    </ThemeProvider>
  )
}