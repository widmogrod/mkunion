import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter, Routes, Route, useLocation } from 'react-router-dom'
import { ExecutionsPageRefactored } from '../pages/ExecutionsPageRefactored'
import { useFilterStore } from '../stores/filter-store'

// Mock the API hooks
jest.mock('../hooks/use-workflow-api', () => ({
  useWorkflowApi: () => ({
    listStates: jest.fn().mockResolvedValue({ Items: [] }),
    error: null
  })
}))

// Mock the refresh store
jest.mock('../stores/refresh-store', () => ({
  useRefreshStore: () => ({
    executionsRefreshTrigger: 0
  })
}))

describe('Filter URL Synchronization', () => {
  beforeEach(() => {
    // Reset the filter store before each test
    useFilterStore.setState({ executionFilters: [] })
  })

  test('URL parameters are correctly synchronized to filter store', async () => {
    const TestWrapper = () => {
      const { executionFilters } = useFilterStore()
      
      return (
        <div>
          <ExecutionsPageRefactored />
          <div data-testid="filter-count">{executionFilters.length}</div>
          <div data-testid="filter-details">
            {executionFilters.map((f, i) => (
              <div key={i}>
                {f.stateType}: {f.label}
              </div>
            ))}
          </div>
        </div>
      )
    }

    render(
      <MemoryRouter initialEntries={['/executions?workflow=scheduled_demo&status=done,error']}>
        <Routes>
          <Route path="/executions" element={<TestWrapper />} />
        </Routes>
      </MemoryRouter>
    )

    // Wait for the filter sync to happen
    await waitFor(() => {
      expect(screen.getByTestId('filter-count')).toHaveTextContent('3')
    })

    const filterDetails = screen.getByTestId('filter-details')
    expect(filterDetails).toHaveTextContent('workflow: scheduled_demo')
    expect(filterDetails).toHaveTextContent('workflow.Done: Done')
    expect(filterDetails).toHaveTextContent('workflow.Error: Error')
  })

  test('Clearing filters updates the URL', async () => {
    let currentLocation: string = ''
    
    // Component to track location changes
    const LocationTracker = () => {
      const location = useLocation()
      React.useEffect(() => {
        currentLocation = location.pathname + location.search
      }, [location])
      return null
    }
    
    const TestWrapper = () => {
      const { executionFilters, clearExecutionFilters } = useFilterStore()
      
      return (
        <MemoryRouter initialEntries={['/executions?workflow=test_workflow']}>
          <Routes>
            <Route 
              path="/executions" 
              element={
                <>
                  <ExecutionsPageRefactored />
                  <button onClick={clearExecutionFilters}>Clear All Filters</button>
                  <div data-testid="filter-count">{executionFilters.length}</div>
                  <LocationTracker />
                </>
              } 
            />
          </Routes>
        </MemoryRouter>
      )
    }

    const { rerender } = render(<TestWrapper />)

    // Wait for initial filter to be set
    await waitFor(() => {
      expect(screen.getByTestId('filter-count')).toHaveTextContent('1')
    })

    // Clear all filters
    fireEvent.click(screen.getByText('Clear All Filters'))

    // Wait for filters to be cleared
    await waitFor(() => {
      expect(screen.getByTestId('filter-count')).toHaveTextContent('0')
    })

    // Force a re-render to check URL
    rerender(<TestWrapper />)

    // Check that URL parameters are cleared
    await waitFor(() => {
      expect(currentLocation).toBe('/executions')
    })
  })

  test('Adding a filter updates the URL', async () => {
    const TestWrapper = () => {
      const { executionFilters, addExecutionFilter } = useFilterStore()
      const [urlParams, setUrlParams] = React.useState('')
      
      React.useEffect(() => {
        const params = new URLSearchParams(window.location.search)
        setUrlParams(params.toString())
      }, [executionFilters])
      
      const handleAddFilter = () => {
        addExecutionFilter({
          stateType: 'workflow.Done',
          label: 'Done',
          color: '#10b981',
          isExclude: false
        })
      }
      
      return (
        <div>
          <ExecutionsPageRefactored />
          <button onClick={handleAddFilter}>Add Done Filter</button>
          <div data-testid="filter-count">{executionFilters.length}</div>
          <div data-testid="url-params">{urlParams}</div>
        </div>
      )
    }

    render(
      <MemoryRouter initialEntries={['/executions']}>
        <Routes>
          <Route path="/executions" element={<TestWrapper />} />
        </Routes>
      </MemoryRouter>
    )

    // Initially no filters
    expect(screen.getByTestId('filter-count')).toHaveTextContent('0')

    // Add a filter
    fireEvent.click(screen.getByText('Add Done Filter'))

    // Wait for filter to be added and URL to update
    await waitFor(() => {
      expect(screen.getByTestId('filter-count')).toHaveTextContent('1')
      expect(screen.getByTestId('url-params')).toContain('status=done')
    })
  })

  test('Removing a single filter updates the URL correctly', async () => {
    const TestWrapper = () => {
      const { executionFilters, removeExecutionFilter } = useFilterStore()
      const [urlParams, setUrlParams] = React.useState('')
      
      React.useEffect(() => {
        const params = new URLSearchParams(window.location.search)
        setUrlParams(params.toString())
      }, [executionFilters])
      
      const handleRemoveFirst = () => {
        if (executionFilters.length > 0) {
          removeExecutionFilter(0)
        }
      }
      
      return (
        <div>
          <ExecutionsPageRefactored />
          <button onClick={handleRemoveFirst}>Remove First Filter</button>
          <div data-testid="filter-count">{executionFilters.length}</div>
          <div data-testid="url-params">{urlParams}</div>
        </div>
      )
    }

    render(
      <MemoryRouter initialEntries={['/executions?workflow=test_workflow&status=done']}>
        <Routes>
          <Route path="/executions" element={<TestWrapper />} />
        </Routes>
      </MemoryRouter>
    )

    // Wait for initial filters to be set
    await waitFor(() => {
      expect(screen.getByTestId('filter-count')).toHaveTextContent('2')
    })

    // Remove the first filter (workflow filter)
    fireEvent.click(screen.getByText('Remove First Filter'))

    // Wait for filter to be removed and URL to update
    await waitFor(() => {
      expect(screen.getByTestId('filter-count')).toHaveTextContent('1')
      expect(screen.getByTestId('url-params')).not.toContain('workflow=')
      expect(screen.getByTestId('url-params')).toContain('status=done')
    })
  })
})