import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { ExecutionsTable } from '../ExecutionsTable'
import { MemoryRouter, Routes, Route, useSearchParams } from 'react-router-dom'
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

// Wrapper component that simulates ExecutionsPage behavior
const ExecutionsPageWrapper: React.FC<{ 
  loadStates: any,
  onUrlChange?: (params: URLSearchParams) => void 
}> = ({ loadStates, onUrlChange }) => {
  const [searchParams] = useSearchParams()
  
  // Extract URL parameters like ExecutionsPage does
  const workflowFilter = searchParams.get('workflow')
  const statusFilter = searchParams.get('status')?.split(',').filter(Boolean) || []
  const runIdFilter = searchParams.get('runId')
  const scheduleFilter = searchParams.get('schedule')
  
  React.useEffect(() => {
    onUrlChange?.(searchParams)
  }, [searchParams, onUrlChange])
  
  return (
    <ExecutionsTable
      refreshTrigger={0}
      loadStates={loadStates}
      workflowFilter={workflowFilter}
      runIdFilter={runIdFilter}
      statusFilter={statusFilter}
      scheduleFilter={scheduleFilter}
    />
  )
}

describe('ExecutionsTable Integration - Real-world URL Sync', () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  it('should update URL and props correctly when removing a workflow filter', async () => {
    const mockLoadStates = createMockLoadStates()
    const urlChanges: string[] = []
    
    render(
      <MemoryRouter initialEntries={['/executions?workflow=test-workflow']}>
        <ToastProvider>
          <Routes>
            <Route path="/executions" element={
              <ExecutionsPageWrapper 
                loadStates={mockLoadStates}
                onUrlChange={(params) => urlChanges.push(params.toString())}
              />
            } />
          </Routes>
        </ToastProvider>
      </MemoryRouter>
    )

    // Wait for initial render with filter
    await waitFor(() => {
      expect(screen.getByText('Filtering:')).toBeInTheDocument()
      expect(screen.getByText('test-workflow')).toBeInTheDocument()
    })

    // Verify initial URL state
    expect(urlChanges[0]).toBe('workflow=test-workflow')

    // Find and click the remove button
    const filterPill = screen.getByText('test-workflow').closest('div')
    const removeButton = filterPill?.querySelector('button[aria-label*="Remove"]')
    
    expect(removeButton).toBeInTheDocument()
    fireEvent.click(removeButton!)

    // Wait for filter to be removed and URL to update
    await waitFor(() => {
      expect(screen.queryByText('test-workflow')).not.toBeInTheDocument()
      expect(screen.queryByText('Filtering:')).not.toBeInTheDocument()
    })

    // Verify URL was cleared
    const finalUrlState = urlChanges[urlChanges.length - 1]
    expect(finalUrlState).toBe('')
  })

  it('should handle multiple filter operations correctly', async () => {
    const mockLoadStates = createMockLoadStates()
    const urlChanges: string[] = []
    
    render(
      <MemoryRouter initialEntries={['/executions']}>
        <ToastProvider>
          <Routes>
            <Route path="/executions" element={
              <ExecutionsPageWrapper 
                loadStates={mockLoadStates}
                onUrlChange={(params) => urlChanges.push(params.toString())}
              />
            } />
          </Routes>
        </ToastProvider>
      </MemoryRouter>
    )

    // Wait for table to load
    await waitFor(() => {
      expect(mockLoadStates).toHaveBeenCalled()
    })

    // Add Done filter by clicking on the status pill
    const donePill = screen.getAllByText('Done')[0].closest('span[data-state-type]')
    fireEvent.click(donePill!)

    // Wait for filter to appear
    await waitFor(() => {
      expect(screen.getByText('Filtering:')).toBeInTheDocument()
      const filterPills = screen.getAllByText('Done').filter(el => 
        el.closest('div[title*="Click to"]')
      )
      expect(filterPills).toHaveLength(1)
    })

    // Verify URL updated
    expect(urlChanges[urlChanges.length - 1]).toContain('status=done')

    // Toggle filter mode
    const filterPill = screen.getAllByText('Done').find(el => 
      el.closest('div[title*="Click to"]')
    )?.closest('div')
    fireEvent.click(filterPill!)

    // Verify filter is in exclude mode but URL still has the parameter
    await waitFor(() => {
      const excludeText = filterPill?.querySelector('.line-through')
      expect(excludeText).toBeInTheDocument()
    })
    expect(urlChanges[urlChanges.length - 1]).toContain('status=done')

    // Remove the filter
    const removeButton = filterPill?.querySelector('button[aria-label*="Remove"]')
    fireEvent.click(removeButton!)

    // Verify filter removed and URL cleared
    await waitFor(() => {
      expect(screen.queryByText('Filtering:')).not.toBeInTheDocument()
    })
    expect(urlChanges[urlChanges.length - 1]).toBe('')
  })

  it('should clear all filters and update URL', async () => {
    const mockLoadStates = createMockLoadStates()
    const urlChanges: string[] = []
    
    render(
      <MemoryRouter initialEntries={['/executions?workflow=test-workflow&status=done,error']}>
        <ToastProvider>
          <Routes>
            <Route path="/executions" element={
              <ExecutionsPageWrapper 
                loadStates={mockLoadStates}
                onUrlChange={(params) => urlChanges.push(params.toString())}
              />
            } />
          </Routes>
        </ToastProvider>
      </MemoryRouter>
    )

    // Wait for all filters to be displayed
    await waitFor(() => {
      expect(screen.getByText('test-workflow')).toBeInTheDocument()
      expect(screen.getByText('Done')).toBeInTheDocument()
      expect(screen.getByText('Error')).toBeInTheDocument()
      expect(screen.getByText('Clear all')).toBeInTheDocument()
    })

    // Click Clear all
    fireEvent.click(screen.getByText('Clear all'))

    // Wait for all filters to be removed
    await waitFor(() => {
      expect(screen.queryByText('test-workflow')).not.toBeInTheDocument()
      expect(screen.queryByText('Done')).not.toBeInTheDocument()
      expect(screen.queryByText('Error')).not.toBeInTheDocument()
      expect(screen.queryByText('Filtering:')).not.toBeInTheDocument()
      expect(screen.queryByText('Clear all')).not.toBeInTheDocument()
    })

    // Verify URL is completely clean
    const finalUrlState = urlChanges[urlChanges.length - 1]
    expect(finalUrlState).toBe('')
  })

  it('should maintain URL sync after rapid filter changes', async () => {
    const mockLoadStates = createMockLoadStates()
    const urlChanges: string[] = []
    
    render(
      <MemoryRouter initialEntries={['/executions']}>
        <ToastProvider>
          <Routes>
            <Route path="/executions" element={
              <ExecutionsPageWrapper 
                loadStates={mockLoadStates}
                onUrlChange={(params) => {
                  const paramStr = params.toString()
                  urlChanges.push(paramStr)
                  console.log('URL changed to:', paramStr)
                }}
              />
            } />
          </Routes>
        </ToastProvider>
      </MemoryRouter>
    )

    // Wait for initial load
    await waitFor(() => {
      expect(mockLoadStates).toHaveBeenCalled()
    })

    // Rapidly add and remove filters
    const donePill = screen.getAllByText('Done')[0].closest('span[data-state-type]')
    const errorPill = screen.getAllByText('Error')[0].closest('span[data-state-type]')
    
    // Add Done
    fireEvent.click(donePill!)
    
    // Add Error
    fireEvent.click(errorPill!)
    
    // Wait for both filters
    await waitFor(() => {
      const filterPills = screen.getAllByText(/Done|Error/).filter(el => 
        el.closest('div[title*="Click to"]')
      )
      expect(filterPills).toHaveLength(2)
    })

    // Verify URL has both filters
    const currentUrl = urlChanges[urlChanges.length - 1]
    expect(currentUrl).toMatch(/status=(done,error|error,done)/)

    // Remove Done filter
    const doneFilterPill = screen.getAllByText('Done').find(el => 
      el.closest('div[title*="Click to"]')
    )?.closest('div')
    const removeButton = doneFilterPill?.querySelector('button[aria-label*="Remove"]')
    fireEvent.click(removeButton!)

    // Wait and verify only Error remains
    await waitFor(() => {
      const filterPills = screen.getAllByText(/Done|Error/).filter(el => 
        el.closest('div[title*="Click to"]')
      )
      expect(filterPills).toHaveLength(1)
      expect(filterPills[0]).toHaveTextContent('Error')
    })

    // Verify URL only has error
    const finalUrl = urlChanges[urlChanges.length - 1]
    expect(finalUrl).toBe('status=error')
  })
})