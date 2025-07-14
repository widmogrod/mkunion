import React from 'react'
import { MainLayout } from './components/layout/main-layout'
import { DemosSection } from './components/demos/DemosSection'
import { TablesSection } from './components/tables/TablesSection'
import { SchedulesPage } from './pages/SchedulesPage'
import { SidebarProvider } from './components/layout/SidebarContext'
import { ThemeProvider } from './components/theme/ThemeContext'
import { ToastProvider } from './contexts/ToastContext'

export default function App() {
  const [activeTab, setActiveTab] = React.useState<'tables' | 'schedules'>('tables')

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
              {activeTab === 'tables' ? <TablesSection /> : <SchedulesPage />}
            </div>
          </MainLayout>
        </SidebarProvider>
      </ToastProvider>
    </ThemeProvider>
  )
}