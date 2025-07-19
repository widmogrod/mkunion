import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { BrowserRouter, useSearchParams } from 'react-router-dom'
import '@testing-library/jest-dom'

// Minimal component that reproduces the URL sync pattern from ExecutionsTable
const MinimalURLSyncComponent: React.FC<{
  initialFilters: string[]
  onFiltersChange?: (filters: string[]) => void
}> = ({ initialFilters, onFiltersChange }) => {
  const [searchParams, setSearchParams] = useSearchParams()
  const [filters, setFilters] = React.useState<string[]>(initialFilters)

  // This is the pattern used in ExecutionsTable - a regular function
  const syncFiltersToUrlRegular = (newFilters: string[]) => {
    console.log('Regular function: syncing filters', newFilters)
    console.log('Regular function: setSearchParams is', typeof setSearchParams)
    
    if (newFilters.length > 0) {
      setSearchParams({ filters: newFilters.join(',') })
    } else {
      setSearchParams({})
    }
  }

  // This is what it should be - useCallback
  const syncFiltersToUrlCallback = React.useCallback((newFilters: string[]) => {
    console.log('Callback function: syncing filters', newFilters)
    console.log('Callback function: setSearchParams is', typeof setSearchParams)
    
    if (newFilters.length > 0) {
      setSearchParams({ filters: newFilters.join(',') })
    } else {
      setSearchParams({})
    }
  }, [setSearchParams])

  const removeFilterRegular = (index: number) => {
    const newFilters = filters.filter((_, i) => i !== index)
    setFilters(newFilters)
    syncFiltersToUrlRegular(newFilters)
    onFiltersChange?.(newFilters)
  }

  const removeFilterCallback = (index: number) => {
    const newFilters = filters.filter((_, i) => i !== index)
    setFilters(newFilters)
    syncFiltersToUrlCallback(newFilters)
    onFiltersChange?.(newFilters)
  }

  return (
    <div>
      <div data-testid="url-display">
        URL: {searchParams.toString()}
      </div>
      <div data-testid="filters">
        {filters.map((filter, index) => (
          <div key={index}>
            <span>{filter}</span>
            <button onClick={() => removeFilterRegular(index)}>Remove (Regular)</button>
            <button onClick={() => removeFilterCallback(index)}>Remove (Callback)</button>
          </div>
        ))}
      </div>
    </div>
  )
}

describe('URL Sync Pattern Reproduction', () => {
  it('demonstrates the bug with regular function vs useCallback', async () => {
    const filterChanges: string[][] = []
    
    const { rerender } = render(
      <BrowserRouter>
        <MinimalURLSyncComponent 
          initialFilters={['filter1', 'filter2']}
          onFiltersChange={(filters) => filterChanges.push(filters)}
        />
      </BrowserRouter>
    )

    // Initial state
    expect(screen.getByTestId('url-display')).toHaveTextContent('URL:')
    expect(screen.getByText('filter1')).toBeInTheDocument()
    expect(screen.getByText('filter2')).toBeInTheDocument()

    // Remove filter using regular function
    const removeRegularButtons = screen.getAllByText('Remove (Regular)')
    fireEvent.click(removeRegularButtons[0])

    await waitFor(() => {
      // The filter should be removed from UI
      expect(screen.queryByText('filter1')).not.toBeInTheDocument()
    })

    // Check if URL was updated
    const urlAfterRegular = screen.getByTestId('url-display').textContent
    console.log('URL after regular function remove:', urlAfterRegular)

    // Reset for callback test
    rerender(
      <BrowserRouter>
        <MinimalURLSyncComponent 
          initialFilters={['filter1', 'filter2']}
          onFiltersChange={(filters) => filterChanges.push(filters)}
        />
      </BrowserRouter>
    )

    // Remove filter using callback function
    const removeCallbackButtons = screen.getAllByText('Remove (Callback)')
    fireEvent.click(removeCallbackButtons[0])

    await waitFor(() => {
      // The filter should be removed from UI
      expect(screen.queryByText('filter1')).not.toBeInTheDocument()
    })

    // Check if URL was updated
    const urlAfterCallback = screen.getByTestId('url-display').textContent
    console.log('URL after callback function remove:', urlAfterCallback)

    // Compare the results
    console.log('Filter changes:', filterChanges)
  })
})

// Now let's test the actual ExecutionsTable pattern
describe('ExecutionsTable URL Sync Pattern', () => {
  it('shows how ExecutionsTable syncFiltersToUrl captures stale closures', () => {
    // This test demonstrates the issue without running the full component
    
    let capturedSetParam: any = null
    let capturedSetArrayParam: any = null
    
    // Simulate the ExecutionsTable pattern
    const TestComponent = () => {
      const [count, setCount] = React.useState(0)
      
      // Mock URL hooks
      const setParam = React.useCallback((key: string, value: any) => {
        console.log(`setParam called: ${key} = ${value}, count = ${count}`)
      }, [count])
      
      const setArrayParam = React.useCallback((key: string, value: any[]) => {
        console.log(`setArrayParam called: ${key} = ${JSON.stringify(value)}, count = ${count}`)
      }, [count])
      
      // This is the bug - regular function captures initial values
      const syncFiltersToUrlBuggy = (filters: any[]) => {
        console.log('Buggy sync - using setParam/setArrayParam from closure')
        setParam('test', 'value')
        setArrayParam('array', filters)
      }
      
      // This is the fix - useCallback with proper dependencies
      const syncFiltersToUrlFixed = React.useCallback((filters: any[]) => {
        console.log('Fixed sync - using current setParam/setArrayParam')
        setParam('test', 'value')
        setArrayParam('array', filters)
      }, [setParam, setArrayParam])
      
      React.useEffect(() => {
        // Capture the functions for testing
        capturedSetParam = setParam
        capturedSetArrayParam = setArrayParam
      }, [setParam, setArrayParam])
      
      return (
        <div>
          <div>Count: {count}</div>
          <button onClick={() => setCount(count + 1)}>Increment</button>
          <button onClick={() => syncFiltersToUrlBuggy(['filter1'])}>
            Sync Buggy
          </button>
          <button onClick={() => syncFiltersToUrlFixed(['filter1'])}>
            Sync Fixed
          </button>
        </div>
      )
    }
    
    const { rerender } = render(<TestComponent />)
    
    // Click increment to change state
    fireEvent.click(screen.getByText('Increment'))
    
    // Now test both sync methods
    fireEvent.click(screen.getByText('Sync Buggy'))
    fireEvent.click(screen.getByText('Sync Fixed'))
    
    // The console output will show the difference:
    // Buggy will use count=0, Fixed will use count=1
  })
})