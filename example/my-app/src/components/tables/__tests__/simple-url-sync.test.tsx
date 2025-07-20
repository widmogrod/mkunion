import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { BrowserRouter, useSearchParams } from 'react-router-dom'
import '@testing-library/jest-dom'

// Simplified ExecutionsTable implementation for testing
const SimplifiedExecutionsTable: React.FC<{
  workflowFilter?: string | null
  statusFilter?: string[]
}> = ({ workflowFilter, statusFilter }) => {
  const [searchParams, setSearchParams] = useSearchParams()
  const [activeFilters, setActiveFilters] = React.useState<any[]>([])
  const [isInitialized, setIsInitialized] = React.useState(false)

  // Initialize filters from props only once
  React.useEffect(() => {
    if (isInitialized) {
      console.log('Skipping initialization, already initialized')
      return
    }

    console.log('Initializing filters from props:', { workflowFilter, statusFilter })
    const initialFilters: any[] = []
    
    if (workflowFilter) {
      initialFilters.push({
        type: 'workflow',
        value: workflowFilter
      })
    }
    
    if (statusFilter && statusFilter.length > 0) {
      statusFilter.forEach(status => {
        initialFilters.push({
          type: 'status',
          value: status
        })
      })
    }
    
    setActiveFilters(initialFilters)
    setIsInitialized(true)
  }, [workflowFilter, statusFilter, isInitialized])

  const syncFiltersToUrl = React.useCallback((filters: any[]) => {
    console.log('syncFiltersToUrl called with:', filters)
    
    const workflowFilters = filters.filter(f => f.type === 'workflow')
    const statusFilters = filters.filter(f => f.type === 'status')
    
    setSearchParams(prev => {
      const newParams = new URLSearchParams(prev)
      
      if (workflowFilters.length > 0) {
        newParams.set('workflow', workflowFilters[0].value)
      } else {
        newParams.delete('workflow')
      }
      
      if (statusFilters.length > 0) {
        newParams.set('status', statusFilters.map(f => f.value).join(','))
      } else {
        newParams.delete('status')
      }
      
      console.log('Setting URL params to:', newParams.toString())
      return newParams
    })
  }, [setSearchParams])

  const removeFilter = React.useCallback((index: number) => {
    console.log('removeFilter called for index:', index)
    const newFilters = activeFilters.filter((_, i) => i !== index)
    console.log('New filters:', newFilters)
    setActiveFilters(newFilters)
    syncFiltersToUrl(newFilters)
  }, [activeFilters, syncFiltersToUrl])

  return (
    <div>
      <div data-testid="current-url">{searchParams.toString()}</div>
      <div data-testid="filters">
        {activeFilters.map((filter, index) => (
          <div key={index} data-testid={`filter-${filter.type}-${filter.value}`}>
            {filter.type}: {filter.value}
            <button onClick={() => removeFilter(index)}>Remove</button>
          </div>
        ))}
      </div>
    </div>
  )
}

// Wrapper that simulates parent component reading URL
const ParentWrapper: React.FC = () => {
  const [searchParams] = useSearchParams()
  
  const workflowFilter = searchParams.get('workflow')
  const statusFilter = searchParams.get('status')?.split(',').filter(Boolean) || []
  
  console.log('Parent reading URL:', { workflowFilter, statusFilter })
  
  return (
    <SimplifiedExecutionsTable
      workflowFilter={workflowFilter}
      statusFilter={statusFilter}
    />
  )
}

describe('Simplified URL Sync Test', () => {
  it('demonstrates the circular dependency issue', async () => {
    const { container } = render(
      <BrowserRouter>
        <ParentWrapper />
      </BrowserRouter>
    )

    // Set initial URL
    window.history.pushState({}, '', '?workflow=test-workflow')
    
    // Force re-render to pick up URL change
    fireEvent(window, new Event('popstate'))
    
    // Wait for filter to appear
    await waitFor(() => {
      expect(screen.getByTestId('filter-workflow-test-workflow')).toBeInTheDocument()
    })

    console.log('--- Clicking remove button ---')
    
    // Click remove
    const removeButton = screen.getByTestId('filter-workflow-test-workflow').querySelector('button')
    fireEvent.click(removeButton!)

    // Check what happens
    await waitFor(() => {
      const currentUrl = screen.getByTestId('current-url').textContent
      console.log('Final URL:', currentUrl)
    })
  })

  it('shows the fix with proper state management', async () => {
    // Component that doesn't re-initialize from props after initial load
    const FixedExecutionsTable: React.FC<{
      workflowFilter?: string | null
      statusFilter?: string[]
    }> = ({ workflowFilter: initialWorkflowFilter, statusFilter: initialStatusFilter }) => {
      const [_, setSearchParams] = useSearchParams()
      const [activeFilters, setActiveFilters] = React.useState<any[]>(() => {
        // Initialize state directly from props
        const filters: any[] = []
        if (initialWorkflowFilter) {
          filters.push({ type: 'workflow', value: initialWorkflowFilter })
        }
        if (initialStatusFilter) {
          initialStatusFilter.forEach(status => {
            filters.push({ type: 'status', value: status })
          })
        }
        return filters
      })

      const syncFiltersToUrl = React.useCallback((filters: any[]) => {
        const workflowFilters = filters.filter(f => f.type === 'workflow')
        const statusFilters = filters.filter(f => f.type === 'status')
        
        setSearchParams(prev => {
          const newParams = new URLSearchParams(prev)
          
          if (workflowFilters.length > 0) {
            newParams.set('workflow', workflowFilters[0].value)
          } else {
            newParams.delete('workflow')
          }
          
          if (statusFilters.length > 0) {
            newParams.set('status', statusFilters.map(f => f.value).join(','))
          } else {
            newParams.delete('status')
          }
          
          return newParams
        })
      }, [setSearchParams])

      const removeFilter = React.useCallback((index: number) => {
        const newFilters = activeFilters.filter((_, i) => i !== index)
        setActiveFilters(newFilters)
        syncFiltersToUrl(newFilters)
      }, [activeFilters, syncFiltersToUrl])

      return (
        <div>
          <div data-testid="filters-fixed">
            {activeFilters.map((filter, index) => (
              <div key={index} data-testid={`filter-fixed-${filter.type}-${filter.value}`}>
                {filter.type}: {filter.value}
                <button onClick={() => removeFilter(index)}>Remove</button>
              </div>
            ))}
          </div>
        </div>
      )
    }

    const FixedParentWrapper: React.FC = () => {
      const [searchParams] = useSearchParams()
      
      const workflowFilter = searchParams.get('workflow')
      const statusFilter = searchParams.get('status')?.split(',').filter(Boolean) || []
      
      return (
        <FixedExecutionsTable
          workflowFilter={workflowFilter}
          statusFilter={statusFilter}
        />
      )
    }

    render(
      <BrowserRouter>
        <FixedParentWrapper />
      </BrowserRouter>
    )

    // Set initial URL
    window.history.pushState({}, '', '?workflow=test-workflow')
    fireEvent(window, new Event('popstate'))
    
    // Wait for filter
    await waitFor(() => {
      expect(screen.getByTestId('filter-fixed-workflow-test-workflow')).toBeInTheDocument()
    })

    // Remove filter
    const removeButton = screen.getByTestId('filter-fixed-workflow-test-workflow').querySelector('button')
    fireEvent.click(removeButton!)

    // Verify filter is removed
    await waitFor(() => {
      expect(screen.queryByTestId('filter-fixed-workflow-test-workflow')).not.toBeInTheDocument()
    })
  })
})