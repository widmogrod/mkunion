import React from 'react'
import { MainLayout } from './components/layout/main-layout'
import { DemosSection } from './components/demos/DemosSection'
import { TablesSection } from './components/tables/TablesSection'
import { SidebarProvider } from './components/layout/SidebarContext'
import { ThemeProvider } from './components/theme/ThemeContext'
import { ToastProvider } from './contexts/ToastContext'

export default function App() {
  return (
    <ThemeProvider>
      <ToastProvider>
        <SidebarProvider>
          <MainLayout sidebar={<DemosSection />}>
            <TablesSection />
          </MainLayout>
        </SidebarProvider>
      </ToastProvider>
    </ThemeProvider>
  )
}