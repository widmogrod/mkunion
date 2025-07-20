import React from 'react'
import { renderHook, act } from '@testing-library/react'
import { BrowserRouter, useSearchParams } from 'react-router-dom'

// Test to understand the exact behavior of setSearchParams
describe('Debug URL Sync Behavior', () => {
  it('shows how setSearchParams works with callbacks', () => {
    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <BrowserRouter>{children}</BrowserRouter>
    )

    const { result } = renderHook(() => {
      const [searchParams, setSearchParams] = useSearchParams()
      
      const setParam = React.useCallback((key: string, value: string | null) => {
        console.log(`setParam called: ${key} = ${value}`)
        setSearchParams(prev => {
          console.log('Previous params:', prev.toString())
          const newParams = new URLSearchParams(prev)
          if (value === null || value === '') {
            newParams.delete(key)
          } else {
            newParams.set(key, value)
          }
          console.log('New params:', newParams.toString())
          return newParams
        })
      }, [setSearchParams])

      return { searchParams, setParam, setSearchParams }
    }, { wrapper })

    // Set initial state
    act(() => {
      result.current.setSearchParams({ workflow: 'test-workflow', status: 'done' })
    })

    console.log('Initial URL params:', result.current.searchParams.toString())

    // Remove workflow param
    act(() => {
      result.current.setParam('workflow', null)
    })

    console.log('After removing workflow:', result.current.searchParams.toString())

    // Check final state
    expect(result.current.searchParams.has('workflow')).toBe(false)
    expect(result.current.searchParams.get('status')).toBe('done')
  })

  it('tests ExecutionsTable filter initialization pattern', () => {
    // Simulate the ExecutionsTable pattern
    const TestComponent: React.FC<{
      workflowFilter?: string | null
      statusFilter?: string[]
    }> = ({ workflowFilter, statusFilter }) => {
      const [searchParams, setSearchParams] = useSearchParams()
      const [activeFilters, setActiveFilters] = React.useState<any[]>([])

      // This simulates the ExecutionsTable useEffect pattern
      React.useEffect(() => {
        console.log('useEffect triggered with:', { workflowFilter, statusFilter })
        const initialFilters: any[] = []
        
        if (workflowFilter) {
          initialFilters.push({
            stateType: 'workflow',
            label: workflowFilter,
            color: '#3b82f6',
            isExclude: false
          })
        }
        
        if (statusFilter && statusFilter.length > 0) {
          statusFilter.forEach(status => {
            initialFilters.push({
              stateType: `workflow.${status}`,
              label: status,
              color: '#ef4444',
              isExclude: false
            })
          })
        }
        
        console.log('Setting activeFilters to:', initialFilters)
        setActiveFilters(initialFilters)
      }, [statusFilter, workflowFilter])

      const syncFiltersToUrl = React.useCallback((filters: any[]) => {
        console.log('syncFiltersToUrl called with:', filters)
        
        const workflowFilters = filters.filter(f => f.stateType === 'workflow')
        const statusFilters = filters.filter(f => f.stateType !== 'workflow')
        
        setSearchParams(prev => {
          const newParams = new URLSearchParams(prev)
          
          // Handle workflow
          if (workflowFilters.length > 0) {
            newParams.set('workflow', workflowFilters[0].label)
          } else {
            newParams.delete('workflow')
          }
          
          // Handle status
          if (statusFilters.length > 0) {
            newParams.set('status', statusFilters.map(f => f.label).join(','))
          } else {
            newParams.delete('status')
          }
          
          console.log('URL params changing from', prev.toString(), 'to', newParams.toString())
          return newParams
        })
      }, [setSearchParams])

      const removeFilter = React.useCallback((index: number) => {
        console.log('removeFilter called for index:', index)
        const newFilters = activeFilters.filter((_, i) => i !== index)
        console.log('New filters after removal:', newFilters)
        setActiveFilters(newFilters)
        syncFiltersToUrl(newFilters)
      }, [activeFilters, syncFiltersToUrl])

      return (
        <div>
          <div data-testid="url">{searchParams.toString()}</div>
          <div data-testid="filters">{JSON.stringify(activeFilters)}</div>
          <button onClick={() => removeFilter(0)}>Remove First Filter</button>
        </div>
      )
    }

    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <BrowserRouter>{children}</BrowserRouter>
    )

    const { result, rerender } = renderHook(
      (props) => {
        const [searchParams] = useSearchParams()
        return { searchParams, Component: <TestComponent {...props} /> }
      },
      { 
        wrapper,
        initialProps: { workflowFilter: 'test-workflow', statusFilter: ['done'] }
      }
    )

    console.log('=== Test complete ===')
  })
})