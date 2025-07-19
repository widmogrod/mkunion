import React from 'react'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'
import { ExecutionsTable } from '../ExecutionsTable'
import { MemoryRouter, Routes, Route, useSearchParams, useLocation } from 'react-router-dom'
import { ToastProvider } from '../../../contexts/ToastContext'
import '@testing-library/jest-dom'

// Mock the API hooks
jest.mock('../../../hooks/use-workflow-api', () => ({
  useWorkflowApi: () => ({
    deleteStates: jest.fn(),
    tryRecover: jest.fn(),
  }),
}))

// Create a mock load function
const createMockLoadStates = () => {
  return jest.fn().mockImplementation(() => 
    Promise.resolve({
      Items: [
        {
          ID: 'test-1',
          Data: {
            $type: 'workflow.Done',
            'workflow.Done': {
              BaseState: {
                RunID: 'run-1',
                Flow: {
                  $type: 'workflow.Flow',
                  'workflow.Flow': { Name: 'test-workflow' }
                }
              }
            }
          }
        },
        {
          ID: 'test-2',
          Data: {
            $type: 'workflow.Error',
            'workflow.Error': {
              BaseState: {
                RunID: 'run-2',
                Flow: {
                  $type: 'workflow.FlowRef',
                  'workflow.FlowRef': { FlowID: 'another-workflow' }
                }
              }
            }
          }
        }
      ],
      Next: null
    })
  )
}

// Component to track URL changes
const URLMonitor: React.FC<{ onUrlChange: (url: string) => void }> = ({ onUrlChange }) => {
  const location = useLocation()
  const [searchParams] = useSearchParams()
  
  React.useEffect(() => {
    onUrlChange(location.pathname + location.search)
  }, [location, onUrlChange])
  
  return null
}

describe('ExecutionsTable Fixed - URL Synchronization', () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  it('should properly update URL when removing a workflow filter', async () => {
    const mockLoadStates = createMockLoadStates()
    const urlHistory: string[] = []
    
    render(
      <MemoryRouter initialEntries={['/executions?workflow=test-workflow']}>
        <ToastProvider>
          <Routes>
            <Route path="/executions" element={
              <>
                <ExecutionsTable
                  refreshTrigger={0}
                  loadStates={mockLoadStates}
                  workflowFilter="test-workflow"
                />
                <URLMonitor onUrlChange={(url) => urlHistory.push(url)} />
              </>
            } />
          </Routes>
        </ToastProvider>
      </MemoryRouter>
    )

    // Wait for initial render and filter to be displayed
    await waitFor(() => {
      expect(screen.getByText('Filtering:')).toBeInTheDocument()
      expect(screen.getByText('test-workflow')).toBeInTheDocument()
    })

    // Find and click the remove button on the workflow filter
    const filterPill = screen.getByText('test-workflow').closest('div')
    const removeButton = filterPill?.querySelector('button[aria-label*="Remove"]')
    
    expect(removeButton).toBeInTheDocument()
    
    // Click remove button
    fireEvent.click(removeButton!)

    // Wait for the filter to be removed from UI
    await waitFor(() => {
      expect(screen.queryByText('test-workflow')).not.toBeInTheDocument()
      expect(screen.queryByText('Filtering:')).not.toBeInTheDocument()
    })

    // Verify URL was updated to remove the workflow parameter
    const finalUrl = urlHistory[urlHistory.length - 1]
    expect(finalUrl).toBe('/executions')
    expect(finalUrl).not.toContain('workflow=')
  })

  it('should properly update URL when toggling filter mode', async () => {
    const mockLoadStates = createMockLoadStates()
    const urlHistory: string[] = []
    
    render(
      <MemoryRouter initialEntries={['/executions?status=done']}>
        <ToastProvider>
          <Routes>
            <Route path="/executions" element={
              <>
                <ExecutionsTable
                  refreshTrigger={0}
                  loadStates={mockLoadStates}
                  statusFilter={['done']}
                />
                <URLMonitor onUrlChange={(url) => urlHistory.push(url)} />
              </>
            } />
          </Routes>
        </ToastProvider>
      </MemoryRouter>
    )

    // Wait for filter to be displayed
    await waitFor(() => {
      expect(screen.getByText('Done')).toBeInTheDocument()
    })

    // Click on the filter pill to toggle exclude mode
    const filterPill = screen.getByText('Done').closest('div[title*="Click to"]')
    fireEvent.click(filterPill!)

    // Wait for visual change (line-through text indicates exclude mode)
    await waitFor(() => {
      const excludeText = screen.getByText('Done').closest('div')?.querySelector('.line-through')
      expect(excludeText).toBeInTheDocument()
    })

    // URL should still contain the status parameter
    const currentUrl = urlHistory[urlHistory.length - 1]
    expect(currentUrl).toContain('status=done')
  })

  it('should clear all URL parameters when clicking Clear all', async () => {
    const mockLoadStates = createMockLoadStates()
    const urlHistory: string[] = []
    
    render(
      <MemoryRouter initialEntries={['/executions?workflow=test-workflow&status=done,error']}>
        <ToastProvider>
          <Routes>
            <Route path="/executions" element={
              <>
                <ExecutionsTable
                  refreshTrigger={0}
                  loadStates={mockLoadStates}
                  workflowFilter="test-workflow"
                  statusFilter={['done', 'error']}
                />
                <URLMonitor onUrlChange={(url) => urlHistory.push(url)} />
              </>
            } />
          </Routes>
        </ToastProvider>
      </MemoryRouter>
    )

    // Wait for filters and clear button to be displayed
    await waitFor(() => {
      expect(screen.getByText('test-workflow')).toBeInTheDocument()
      expect(screen.getByText('Done')).toBeInTheDocument()
      expect(screen.getByText('Error')).toBeInTheDocument()
      expect(screen.getByText('Clear all')).toBeInTheDocument()
    })

    // Click Clear all button
    fireEvent.click(screen.getByText('Clear all'))

    // Wait for all filters to be removed
    await waitFor(() => {
      expect(screen.queryByText('test-workflow')).not.toBeInTheDocument()
      expect(screen.queryByText('Done')).not.toBeInTheDocument()
      expect(screen.queryByText('Error')).not.toBeInTheDocument()
      expect(screen.queryByText('Filtering:')).not.toBeInTheDocument()
    })

    // Verify URL is completely clean
    const finalUrl = urlHistory[urlHistory.length - 1]
    expect(finalUrl).toBe('/executions')
    expect(finalUrl).not.toContain('workflow=')
    expect(finalUrl).not.toContain('status=')
  })

  it('should handle rapid filter operations without losing URL sync', async () => {
    const mockLoadStates = createMockLoadStates()
    const urlHistory: string[] = []
    
    render(
      <MemoryRouter initialEntries={['/executions']}>
        <ToastProvider>
          <Routes>
            <Route path="/executions" element={
              <>
                <ExecutionsTable
                  refreshTrigger={0}
                  loadStates={mockLoadStates}
                />
                <URLMonitor onUrlChange={(url) => urlHistory.push(url)} />
              </>
            } />
          </Routes>
        </ToastProvider>
      </MemoryRouter>
    )

    // Wait for table to load
    await waitFor(() => {
      expect(mockLoadStates).toHaveBeenCalled()
    })

    // Rapidly click on status pills to add filters
    const donePill = screen.getAllByText('Done')[0].closest('span[data-state-type]')
    const errorPill = screen.getAllByText('Error')[0].closest('span[data-state-type]')
    
    // Add Done filter
    fireEvent.click(donePill!)
    
    // Add Error filter
    fireEvent.click(errorPill!)
    
    // Wait for both filters to appear
    await waitFor(() => {
      expect(screen.getByText('Filtering:')).toBeInTheDocument()
      const filterPills = screen.getAllByText(/Done|Error/).filter(el => 
        el.closest('div[title*="Click to"]')
      )
      expect(filterPills).toHaveLength(2)
    })

    // Remove Done filter
    const doneFilterPill = screen.getAllByText('Done').find(el => 
      el.closest('div[title*="Click to"]')
    )?.closest('div')
    const removeButton = doneFilterPill?.querySelector('button[aria-label*="Remove"]')
    fireEvent.click(removeButton!)

    // Final URL should only have error filter
    await waitFor(() => {
      const finalUrl = urlHistory[urlHistory.length - 1]
      expect(finalUrl).toContain('status=error')
      expect(finalUrl).not.toContain('done')
    })
  })

  it('should maintain URL sync after component re-renders', async () => {
    const mockLoadStates = createMockLoadStates()
    const urlHistory: string[] = []
    
    const { rerender } = render(
      <MemoryRouter initialEntries={['/executions?workflow=test-workflow']}>
        <ToastProvider>
          <Routes>
            <Route path="/executions" element={
              <>
                <ExecutionsTable
                  refreshTrigger={0}
                  loadStates={mockLoadStates}
                  workflowFilter="test-workflow"
                />
                <URLMonitor onUrlChange={(url) => urlHistory.push(url)} />
              </>
            } />
          </Routes>
        </ToastProvider>
      </MemoryRouter>
    )

    // Wait for initial render
    await waitFor(() => {
      expect(screen.getByText('test-workflow')).toBeInTheDocument()
    })

    // Force a re-render with a new refresh trigger
    rerender(
      <MemoryRouter initialEntries={['/executions?workflow=test-workflow']}>
        <ToastProvider>
          <Routes>
            <Route path="/executions" element={
              <>
                <ExecutionsTable
                  refreshTrigger={1}
                  loadStates={mockLoadStates}
                  workflowFilter="test-workflow"
                />
                <URLMonitor onUrlChange={(url) => urlHistory.push(url)} />
              </>
            } />
          </Routes>
        </ToastProvider>
      </MemoryRouter>
    )

    // Wait for re-render to complete
    await waitFor(() => {
      expect(mockLoadStates).toHaveBeenCalledTimes(2)
    })

    // Remove filter after re-render
    const filterPill = screen.getByText('test-workflow').closest('div')
    const removeButton = filterPill?.querySelector('button[aria-label*="Remove"]')
    fireEvent.click(removeButton!)

    // Verify URL sync still works after re-render
    await waitFor(() => {
      const finalUrl = urlHistory[urlHistory.length - 1]
      expect(finalUrl).toBe('/executions')
      expect(finalUrl).not.toContain('workflow=')
    })
  })
})