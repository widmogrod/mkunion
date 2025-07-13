import React, { createContext, useContext, useState } from 'react'

type DemoType = 'hello' | 'image' | 'schedule' | 'chat' | null

interface SidebarContextType {
  isCollapsed: boolean
  setIsCollapsed: (collapsed: boolean) => void
  selectedDemo: DemoType
  setSelectedDemo: (demo: DemoType) => void
  toggleSidebar: () => void
}

const SidebarContext = createContext<SidebarContextType | undefined>(undefined)

export function SidebarProvider({ children }: { children: React.ReactNode }) {
  const [isCollapsed, setIsCollapsed] = useState(false)
  const [selectedDemo, setSelectedDemo] = useState<DemoType>(null)

  const toggleSidebar = () => {
    setIsCollapsed(!isCollapsed)
  }

  return (
    <SidebarContext.Provider
      value={{
        isCollapsed,
        setIsCollapsed,
        selectedDemo,
        setSelectedDemo,
        toggleSidebar,
      }}
    >
      {children}
    </SidebarContext.Provider>
  )
}

export function useSidebar() {
  const context = useContext(SidebarContext)
  if (context === undefined) {
    throw new Error('useSidebar must be used within a SidebarProvider')
  }
  return context
}