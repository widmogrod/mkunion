import React from 'react'
import { ChevronLeft, ChevronRight, MessageSquare, Image, Clock, Zap, CalendarDays, Calendar, Database, Activity } from 'lucide-react'
import { Button } from '../ui/button'
import { useSidebar } from './SidebarContext'
import { ThemeToggle } from '../theme/ThemeToggle'

interface MainLayoutProps {
  children: React.ReactNode
  sidebar?: React.ReactNode
  activeTab?: 'workflows' | 'executions' | 'schedules' | 'calendar'
  onTabChange?: (tab: 'workflows' | 'executions' | 'schedules' | 'calendar') => void
}

export function MainLayout({ children, sidebar, activeTab, onTabChange }: MainLayoutProps) {
  const { isCollapsed, toggleSidebar, selectedDemo, setSelectedDemo } = useSidebar()

  // Demo icons for collapsed state
  const demoIcons = [
    { icon: Zap, label: 'Hello World', id: 'hello' as const },
    { icon: Image, label: 'Image Generation', id: 'image' as const },
    { icon: Clock, label: 'Scheduled Async', id: 'schedule' as const },
    { icon: MessageSquare, label: 'Chat Interface', id: 'chat' as const },
  ]

  const handleIconClick = (demoId: typeof demoIcons[0]['id']) => {
    setSelectedDemo(demoId)
    if (isCollapsed) {
      toggleSidebar() // Expand sidebar when icon is clicked
    }
  }

  return (
    <div className="min-h-screen bg-background">
      <div className="flex h-screen overflow-hidden">
        {/* Sidebar */}
        {sidebar && (
          <aside className={`flex-shrink-0 border-r bg-card transition-all duration-300 ease-in-out overflow-hidden relative ${
            isCollapsed ? 'w-16' : 'w-96 max-w-96'
          }`}>
            <div className="flex h-full flex-col">
              {/* Sidebar Header with Toggle */}
              <div className="flex items-center justify-between p-4 border-b">
                {!isCollapsed && (
                  <h2 className="text-lg font-semibold">Demos</h2>
                )}
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={toggleSidebar}
                  className="h-8 w-8 p-0"
                >
                  {isCollapsed ? (
                    <ChevronRight className="h-4 w-4" />
                  ) : (
                    <ChevronLeft className="h-4 w-4" />
                  )}
                </Button>
              </div>

              {/* Sidebar Content */}
              <div className="flex-1 overflow-y-auto">
                {isCollapsed ? (
                  /* Collapsed State - Icon Menu */
                  <div className="p-2 space-y-2">
                    {demoIcons.map(({ icon: Icon, label, id }) => (
                      <Button
                        key={id}
                        variant={selectedDemo === id ? "default" : "ghost"}
                        size="sm"
                        className="w-full h-10 p-0 flex items-center justify-center"
                        title={label}
                        onClick={() => handleIconClick(id)}
                      >
                        <Icon className="h-5 w-5" />
                      </Button>
                    ))}
                  </div>
                ) : (
                  /* Expanded State - Full Content */
                  <div className="p-4 overflow-y-auto">
                    <div className="max-w-full">
                      {sidebar}
                    </div>
                  </div>
                )}
              </div>
            </div>
          </aside>
        )}
        
        {/* Main Content */}
        <main className="flex-1 overflow-y-auto">
          {/* Top bar with navigation tabs and theme toggle */}
          <div className="flex items-center justify-between p-4 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
            {/* Navigation Tabs - Apple-inspired design */}
            {activeTab && onTabChange && (
              <div className="flex items-center gap-1 bg-muted/50 rounded-lg p-1">
                <Button
                  variant={activeTab === 'workflows' ? 'default' : 'ghost'}
                  size="sm"
                  onClick={() => onTabChange('workflows')}
                  className="flex items-center gap-2 h-8 px-3 rounded-md transition-all duration-200"
                >
                  <Database className="h-4 w-4" />
                  <span className="hidden sm:inline">Workflows</span>
                  <span className="sm:hidden">Workflows</span>
                </Button>
                <Button
                  variant={activeTab === 'executions' ? 'default' : 'ghost'}
                  size="sm"
                  onClick={() => onTabChange('executions')}
                  className="flex items-center gap-2 h-8 px-3 rounded-md transition-all duration-200"
                >
                  <Activity className="h-4 w-4" />
                  <span className="hidden sm:inline">Executions</span>
                  <span className="sm:hidden">Executions</span>
                </Button>
                <Button
                  variant={activeTab === 'schedules' ? 'default' : 'ghost'}
                  size="sm"
                  onClick={() => onTabChange('schedules')}
                  className="flex items-center gap-2 h-8 px-3 rounded-md transition-all duration-200"
                >
                  <CalendarDays className="h-4 w-4" />
                  <span className="hidden sm:inline">Schedules</span>
                  <span className="sm:hidden">Schedules</span>
                </Button>
                <Button
                  variant={activeTab === 'calendar' ? 'default' : 'ghost'}
                  size="sm"
                  onClick={() => onTabChange('calendar')}
                  className="flex items-center gap-2 h-8 px-3 rounded-md transition-all duration-200"
                >
                  <Calendar className="h-4 w-4" />
                  <span className="hidden sm:inline">Operations Calendar</span>
                  <span className="sm:hidden">Calendar</span>
                </Button>
              </div>
            )}
            
            {/* Theme Toggle */}
            <ThemeToggle />
          </div>
          
          <div className="p-6 w-full h-full">
            {children}
          </div>
        </main>
      </div>
    </div>
  )
}