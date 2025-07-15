import React, { useEffect } from 'react'
import { ChevronLeft, ChevronRight, MessageSquare, Image, Clock, Zap, Layers, Menu } from 'lucide-react'
import { Button } from '../ui/button'
import { useSidebar } from './SidebarContext'
import { ThemeToggle } from '../theme/ThemeToggle'
import { NavigationSection } from '../navigation/NavigationSection'
import { CollapsibleSection } from './CollapsibleSection'
import { cn } from '../../utils/cn'

interface MainLayoutProps {
  children: React.ReactNode
  sidebar?: React.ReactNode
  activeTab?: 'workflows' | 'executions' | 'schedules' | 'calendar'
  onTabChange?: (tab: 'workflows' | 'executions' | 'schedules' | 'calendar') => void
}

export function MainLayout({ children, sidebar, activeTab, onTabChange }: MainLayoutProps) {
  const { isCollapsed, toggleSidebar, selectedDemo, setSelectedDemo } = useSidebar()
  const [isMobile, setIsMobile] = React.useState(false)
  const [isSidebarOpen, setIsSidebarOpen] = React.useState(false)

  // Check if mobile on mount and window resize
  useEffect(() => {
    const checkMobile = () => {
      setIsMobile(window.innerWidth < 768)
    }
    checkMobile()
    window.addEventListener('resize', checkMobile)
    return () => window.removeEventListener('resize', checkMobile)
  }, [])

  // Close sidebar on mobile when route changes
  useEffect(() => {
    if (isMobile) {
      setIsSidebarOpen(false)
    }
  }, [activeTab, isMobile])

  // Demo icons for collapsed state
  const demoIcons = [
    { icon: Zap, label: 'Hello World', id: 'hello' as const },
    { icon: Image, label: 'Image Generation', id: 'image' as const },
    { icon: Clock, label: 'Scheduled Async', id: 'schedule' as const },
    { icon: MessageSquare, label: 'Chat Interface', id: 'chat' as const },
  ]

  const handleIconClick = (demoId: typeof demoIcons[0]['id']) => {
    setSelectedDemo(demoId)
    if (isCollapsed && !isMobile) {
      toggleSidebar() // Expand sidebar when icon is clicked (desktop only)
    }
  }

  const handleNavigationClick = (tab: string) => {
    onTabChange?.(tab as 'workflows' | 'executions' | 'schedules' | 'calendar')
    if (isMobile) {
      setIsSidebarOpen(false)
    }
  }

  return (
    <div className="min-h-screen bg-background">
      <div className="flex h-screen overflow-hidden">
        {/* Mobile backdrop */}
        {isMobile && isSidebarOpen && (
          <div 
            className="fixed inset-0 bg-black/50 z-40 md:hidden animate-in fade-in duration-200"
            onClick={() => setIsSidebarOpen(false)}
          />
        )}

        {/* Sidebar */}
        {sidebar && (
          <aside className={`
            ${isMobile 
              ? `fixed left-0 top-0 z-50 h-full w-80 ${isSidebarOpen ? 'translate-x-0' : '-translate-x-full'} shadow-2xl`
              : `flex-shrink-0 relative ${isCollapsed ? 'w-16' : 'w-96 max-w-96'}`
            }
            border-r bg-card transition-all duration-300 ease-in-out overflow-hidden
          `}>
            <div className="flex h-full flex-col">
              {/* Sidebar Header with Toggle */}
              <div className="flex items-center justify-between px-4 py-3 border-b">
                {(!isCollapsed || isMobile) && (
                  <h2 className="text-sm font-medium text-muted-foreground">Menu</h2>
                )}
                {!isMobile && (
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
                )}
              </div>

              {/* Sidebar Content */}
              <div className="flex-1 overflow-y-auto">
                {isCollapsed && !isMobile ? (
                  /* Collapsed State - Icon Menu (Desktop only) */
                  <div className="p-2 space-y-6">
                    {/* Navigation Section */}
                    <div className="space-y-1">
                      <NavigationSection
                        activeTab={activeTab}
                        onTabChange={handleNavigationClick}
                        isCollapsed={true}
                      />
                    </div>
                    
                    {/* Divider */}
                    <div className="h-px bg-border" />
                    
                    {/* Demo Icons */}
                    <div className="space-y-1">
                      {demoIcons.map(({ icon: Icon, label, id }) => (
                        <Button
                          key={id}
                          variant={selectedDemo === id ? "secondary" : "ghost"}
                          size="sm"
                          className={cn(
                            "w-full h-10 p-0 flex items-center justify-center group transition-all duration-200",
                            selectedDemo === id && "bg-secondary/80 hover:bg-secondary/90"
                          )}
                          title={label}
                          onClick={() => handleIconClick(id)}
                        >
                          <Icon className={cn(
                            "h-4 w-4 transition-colors duration-200",
                            selectedDemo === id ? "text-primary" : "text-muted-foreground group-hover:text-foreground"
                          )} />
                        </Button>
                      ))}
                    </div>
                  </div>
                ) : (
                  /* Expanded State - Full Content */
                  <div className="flex flex-col h-full">
                    {/* Navigation Section */}
                    <div className="p-4 pb-0">
                      <NavigationSection
                        activeTab={activeTab}
                        onTabChange={handleNavigationClick}
                        isCollapsed={false}
                      />
                    </div>
                    
                    {/* Divider */}
                    <div className="mx-4 my-4 h-px bg-border" />
                    
                    {/* Demos Section */}
                    <div className="flex-1 px-4 pb-4 overflow-y-auto">
                      <CollapsibleSection
                        title="Demos"
                        icon={Layers}
                        defaultOpen={true}
                      >
                        <div className="max-w-full">
                          {sidebar}
                        </div>
                      </CollapsibleSection>
                    </div>
                  </div>
                )}
              </div>
            </div>
          </aside>
        )}
        
        {/* Main Content */}
        <main className="flex-1 overflow-y-auto">
          {/* Top bar with theme toggle */}
          <div className="flex items-center justify-between p-4 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
            {/* Hamburger menu for mobile */}
            {isMobile && sidebar && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setIsSidebarOpen(!isSidebarOpen)}
                className="h-9 w-9 p-0 md:hidden"
              >
                <Menu className="h-5 w-5" />
              </Button>
            )}
            
            {/* Spacer for desktop */}
            {!isMobile && <div />}
            
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