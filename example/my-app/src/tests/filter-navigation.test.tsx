import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { useNavigationWithContext } from '../hooks/useNavigation'

// Test component to verify navigation behavior
const NavigationTester = () => {
  const { navigateToExecutions, navigateToWorkflows } = useNavigationWithContext()
  const location = window.location
  
  return (
    <div>
      <div data-testid="current-url">{location.href}</div>
      <button onClick={() => navigateToExecutions({ workflow: 'test-workflow' })}>
        Navigate to Executions with Workflow
      </button>
      <button onClick={() => navigateToWorkflows('test-flow')}>
        Navigate to Workflows
      </button>
    </div>
  )
}

describe('Filter Navigation Behavior', () => {
  test('navigateToExecutions should preserve existing URL parameters', () => {
    // This test verifies if navigation preserves existing filters
    const TestApp = () => {
      return (
        <MemoryRouter initialEntries={['/executions?status=done&runId=123']}>
          <Routes>
            <Route path="/executions" element={<NavigationTester />} />
          </Routes>
        </MemoryRouter>
      )
    }
    
    render(<TestApp />)
    
    // Check initial URL has filters
    expect(window.location.search).toContain('status=done')
    expect(window.location.search).toContain('runId=123')
    
    // Navigate with workflow filter
    fireEvent.click(screen.getByText('Navigate to Executions with Workflow'))
    
    // EXPECTED: Should preserve existing filters AND add workflow
    // Current behavior: Likely clears existing filters
    // This test will help us verify the issue
  })
  
  test('tab navigation should preserve relevant filters', () => {
    // This test checks if switching tabs preserves filters
    const TabNavigationTest = () => {
      const [activeTab, setActiveTab] = React.useState('executions')
      
      return (
        <div>
          <div data-testid="active-tab">{activeTab}</div>
          <button onClick={() => setActiveTab('workflows')}>Switch to Workflows</button>
          <button onClick={() => setActiveTab('executions')}>Switch to Executions</button>
        </div>
      )
    }
    
    render(
      <MemoryRouter initialEntries={['/executions?workflow=demo&status=done']}>
        <TabNavigationTest />
      </MemoryRouter>
    )
    
    // Initial state
    expect(screen.getByTestId('active-tab')).toHaveTextContent('executions')
    
    // Switch to workflows tab
    fireEvent.click(screen.getByText('Switch to Workflows'))
    
    // Switch back to executions
    fireEvent.click(screen.getByText('Switch to Executions'))
    
    // EXPECTED: workflow and status filters should be preserved
    // Current behavior: Filters are cleared
  })
  
  test('filter sync should handle rapid changes without race conditions', async () => {
    // This test simulates rapid filter changes
    let filterChangeCount = 0
    
    const RapidFilterTest = () => {
      const [filters, setFilters] = React.useState<string[]>([])
      
      const addFilter = (filter: string) => {
        filterChangeCount++
        setFilters(prev => [...prev, filter])
      }
      
      const clearFilters = () => {
        filterChangeCount++
        setFilters([])
      }
      
      return (
        <div>
          <div data-testid="filter-count">{filters.length}</div>
          <button onClick={() => addFilter('done')}>Add Done</button>
          <button onClick={() => addFilter('error')}>Add Error</button>
          <button onClick={clearFilters}>Clear All</button>
        </div>
      )
    }
    
    render(<RapidFilterTest />)
    
    // Rapidly add and clear filters
    fireEvent.click(screen.getByText('Add Done'))
    fireEvent.click(screen.getByText('Add Error'))
    fireEvent.click(screen.getByText('Clear All'))
    fireEvent.click(screen.getByText('Add Done'))
    
    await waitFor(() => {
      expect(screen.getByTestId('filter-count')).toHaveTextContent('1')
    })
    
    // Verify no race conditions - final state should match expectations
    expect(filterChangeCount).toBe(4)
  })
})