import React from 'react'
import { Button } from '../ui/button'
import { Table, CalendarDays, Workflow } from 'lucide-react'

interface NavigationTabsProps {
  activeTab: 'tables' | 'schedules'
  onTabChange: (tab: 'tables' | 'schedules') => void
}

export function NavigationTabs({ activeTab, onTabChange }: NavigationTabsProps) {
  return (
    <div className="flex gap-2 p-4 border-b">
      <Button
        variant={activeTab === 'tables' ? 'default' : 'ghost'}
        size="sm"
        onClick={() => onTabChange('tables')}
        className="flex items-center gap-2"
      >
        <Table className="h-4 w-4" />
        Workflows & States
      </Button>
      <Button
        variant={activeTab === 'schedules' ? 'default' : 'ghost'}
        size="sm"
        onClick={() => onTabChange('schedules')}
        className="flex items-center gap-2"
      >
        <CalendarDays className="h-4 w-4" />
        Schedules
      </Button>
    </div>
  )
}