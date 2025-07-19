import React from 'react'
import { Database, Activity, CalendarDays, Calendar, LucideIcon } from 'lucide-react'
import { Button } from '../ui/button'
import { cn } from '../../utils/cn'

interface NavigationItem {
  id: string
  label: string
  icon: LucideIcon
  shortLabel?: string
}

interface NavigationSectionProps {
  activeTab?: string
  onTabChange?: (tab: string) => void
  isCollapsed?: boolean
}

const navigationItems: NavigationItem[] = [
  {
    id: 'workflows',
    label: 'Workflows',
    icon: Database,
  },
  {
    id: 'executions', 
    label: 'Executions',
    icon: Activity,
  },
  {
    id: 'schedules',
    label: 'Schedules',
    icon: CalendarDays,
  },
  {
    id: 'calendar',
    label: 'Operations Calendar',
    icon: Calendar,
    shortLabel: 'Calendar',
  },
]

export function NavigationSection({ activeTab, onTabChange, isCollapsed = false }: NavigationSectionProps) {
  return (
    <div className="space-y-1">
      {navigationItems.map((item) => {
        const Icon = item.icon
        const isActive = activeTab === item.id
        
        return (
          <Button
            key={item.id}
            variant={isActive ? 'secondary' : 'ghost'}
            size="sm"
            onClick={() => onTabChange?.(item.id)}
            className={cn(
              'w-full justify-start gap-3 transition-all duration-200 group relative',
              isCollapsed ? 'px-0 justify-center' : 'px-3',
              isActive && 'bg-secondary/80 hover:bg-secondary/90',
              !isActive && 'hover:bg-muted/50'
            )}
            title={isCollapsed ? item.label : undefined}
          >
            {/* Active indicator */}
            {isActive && (
              <div className="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-6 bg-primary rounded-r-full transition-all duration-300" />
            )}
            
            <Icon className={cn(
              'h-4 w-4 flex-shrink-0 transition-colors duration-200',
              isActive ? 'text-primary' : 'text-muted-foreground group-hover:text-foreground'
            )} />
            {!isCollapsed && (
              <span className={cn(
                'truncate text-sm font-medium transition-colors duration-200',
                isActive ? 'text-foreground' : 'text-muted-foreground group-hover:text-foreground'
              )}>
                {item.shortLabel || item.label}
              </span>
            )}
          </Button>
        )
      })}
    </div>
  )
}