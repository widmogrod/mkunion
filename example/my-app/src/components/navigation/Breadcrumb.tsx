import React from 'react'
import { ChevronRight, Home } from 'lucide-react'
import { useLocation } from 'react-router-dom'
import { useNavigationWithContext, useUrlParams } from '../../hooks/useNavigation'
import { cn } from '../../lib/utils'

interface BreadcrumbItem {
  label: string
  href?: string
  onClick?: () => void
  icon?: React.ReactNode
}

export function Breadcrumb() {
  const location = useLocation()
  const { navigateToWorkflows, navigateToExecutions, navigateToSchedules, navigateToCalendar } = useNavigationWithContext()
  const { getParam, getArrayParam } = useUrlParams()

  // Build breadcrumb items based on current route and params
  const getBreadcrumbItems = (): BreadcrumbItem[] => {
    const items: BreadcrumbItem[] = []
    
    // Always start with home
    items.push({
      label: 'Home',
      icon: <Home className="h-3 w-3" />,
      onClick: () => navigateToWorkflows()
    })

    // Add current page
    const path = location.pathname.replace('/', '')
    
    switch (path) {
      case 'states': // Handle backward compatibility
      case 'executions': {
        items.push({
          label: 'Executions',
          onClick: () => navigateToExecutions()
        })
        
        const workflowFilter = getParam('workflow')
        const runIdFilter = getParam('runId')
        const statusFilter = getArrayParam('status')
        
        if (workflowFilter) {
          items.push({
            label: `Workflow: ${workflowFilter}`
          })
        } else if (runIdFilter) {
          items.push({
            label: `Run: ${runIdFilter.substring(0, 8)}...`
          })
        } else if (statusFilter.length > 0) {
          items.push({
            label: `Status: ${statusFilter.join(', ')}`
          })
        }
        break
      }
      case 'workflows': {
        items.push({
          label: 'Workflows',
          onClick: () => navigateToWorkflows()
        })
        
        const workflowId = getParam('id')
        if (workflowId) {
          items.push({
            label: `Workflow: ${workflowId.substring(0, 8)}...`
          })
        }
        break
      }
      
      case 'schedules': {
        items.push({
          label: 'Schedules',
          onClick: () => navigateToSchedules()
        })
        
        const parentRunId = getParam('parentRunId')
        const focus = getParam('focus')
        
        if (parentRunId) {
          items.push({
            label: `Schedule: ${parentRunId.substring(0, 8)}...`
          })
          
          if (focus) {
            items.push({
              label: `Focus: ${focus.substring(0, 8)}...`
            })
          }
        }
        break
      }
      
      case 'calendar': {
        items.push({
          label: 'Operations Calendar',
          onClick: () => navigateToCalendar()
        })
        
        const workflow = getParam('workflow')
        const schedule = getParam('schedule')
        const view = getParam('view')
        
        if (workflow) {
          items.push({
            label: `Workflow: ${workflow}`
          })
        } else if (schedule) {
          items.push({
            label: `Schedule: ${schedule.substring(0, 8)}...`
          })
        }
        
        if (view) {
          items.push({
            label: view.charAt(0).toUpperCase() + view.slice(1) + ' View'
          })
        }
        break
      }
    }
    
    return items
  }

  const items = getBreadcrumbItems()
  
  if (items.length <= 1) return null // Don't show breadcrumb for just home

  return (
    <nav className="flex items-center gap-1 text-xs text-muted-foreground mb-4">
      {items.map((item, index) => (
        <React.Fragment key={index}>
          {index > 0 && <ChevronRight className="h-3 w-3 mx-1" />}
          {item.onClick ? (
            <button
              onClick={item.onClick}
              className={cn(
                "inline-flex items-center gap-1",
                "hover:text-foreground transition-colors",
                "focus:outline-none focus:ring-2 focus:ring-primary/20 focus:ring-offset-1 rounded-sm px-1"
              )}
            >
              {item.icon}
              {item.label}
            </button>
          ) : (
            <span className="inline-flex items-center gap-1 px-1 text-foreground font-medium">
              {item.icon}
              {item.label}
            </span>
          )}
        </React.Fragment>
      ))}
    </nav>
  )
}