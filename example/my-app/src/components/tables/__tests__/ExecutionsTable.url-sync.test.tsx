import React from 'react'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'
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

// Simple mock load function that returns immediately
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
        }
      ],
      Next: null
    })
  )
}

// Component to display and track URL changes
const URLTracker: React.FC<{ onUrlChange: (params: URLSearchParams) => void }> = ({ onUrlChange }) => {
  const [searchParams] = useSearchParams()
  
  React.useEffect(() => {
    onUrlChange(searchParams)
  }, [searchParams, onUrlChange])
  
  return (
    <div data-testid="url-display">
      {Array.from(searchParams.entries()).map(([key, value]) => (
        <div key={key} data-testid={`param-${key}`}>{key}={value}</div>
      ))}
    </div>
  )
}

describe('ExecutionsTable URL Synchronization Bug', () => {
  it('should remove workflow filter from URL when clicking remove button', async () => {
    const mockLoadStates = createMockLoadStates()
    const urlChanges: URLSearchParams[] = []
    
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
                <URLTracker onUrlChange={(params) => urlChanges.push(params)} />
              </>
            } />
          </Routes>
        </ToastProvider>
      </MemoryRouter>
    )

    // Wait for the filter to be displayed
    await waitFor(() => {
      expect(screen.getByText('test-workflow')).toBeInTheDocument()
    })

    // Find and click the remove button on the workflow filter
    const filterPill = screen.getByText('test-workflow').closest('[data-state-type="workflow"]')
    const removeButton = filterPill?.querySelector('button[aria-label*="Remove"]')
    
    expect(removeButton).toBeInTheDocument()
    
    // Click remove button
    await act(async () => {
      fireEvent.click(removeButton!)
    })

    // Wait and check URL changes
    await waitFor(() => {
      // The last URL change should have no workflow parameter
      const lastUrlChange = urlChanges[urlChanges.length - 1]
      expect(lastUrlChange.has('workflow')).toBe(false)
    })

    // Verify filter is no longer displayed
    expect(screen.queryByText('test-workflow')).not.toBeInTheDocument()
  })

  it('should toggle filter mode without losing URL sync', async () => {
    const mockLoadStates = createMockLoadStates()
    const urlChanges: URLSearchParams[] = []
    
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
                <URLTracker onUrlChange={(params) => urlChanges.push(params)} />
              </>
            } />
          </Routes>
        </ToastProvider>
      </MemoryRouter>
    )

    // Wait for the filter to be displayed
    await waitFor(() => {
      expect(screen.getByText('Done')).toBeInTheDocument()
    })

    // Click on the filter pill to toggle exclude mode
    const filterPill = screen.getByText('Done').closest('[data-state-type]')
    
    await act(async () => {
      fireEvent.click(filterPill!)
    })

    // URL should still have the status parameter
    await waitFor(() => {
      const lastUrlChange = urlChanges[urlChanges.length - 1]
      expect(lastUrlChange.get('status')).toBe('done')
    })
  })

  it('should clear all filters from URL', async () => {
    const mockLoadStates = createMockLoadStates()
    const urlChanges: URLSearchParams[] = []
    
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
                <URLTracker onUrlChange={(params) => urlChanges.push(params)} />
              </>
            } />
          </Routes>
        </ToastProvider>
      </MemoryRouter>
    )

    // Wait for clear all button
    await waitFor(() => {
      expect(screen.getByText('Clear all')).toBeInTheDocument()
    })

    // Click Clear all
    await act(async () => {
      fireEvent.click(screen.getByText('Clear all'))
    })

    // URL should be completely clean
    await waitFor(() => {
      const lastUrlChange = urlChanges[urlChanges.length - 1]
      expect(lastUrlChange.has('workflow')).toBe(false)
      expect(lastUrlChange.has('status')).toBe(false)
      expect(Array.from(lastUrlChange.entries()).length).toBe(0)
    })
  })
})

// Comparison test with WorkflowsTable to verify it works correctly
describe('WorkflowsTable URL Sync (for comparison)', () => {
  // Mock for workflows table
  jest.mock('../../../hooks/use-workflow-api', () => ({
    useWorkflowApi: () => ({
      deleteFlows: jest.fn(),
      listFlows: jest.fn(),
    }),
  }))

  const createMockLoadFlows = () => {
    return jest.fn().mockImplementation(() => 
      Promise.resolve({
        Items: [
          {
            ID: 'flow-1',
            Data: { Name: 'test-workflow' }
          }
        ],
        Next: null
      })
    )
  }

  it('WorkflowsTable should correctly sync filters to URL', async () => {
    const { WorkflowsTable } = require('../WorkflowsTable')
    const mockLoadFlows = createMockLoadFlows()
    const urlChanges: URLSearchParams[] = []
    
    render(
      <MemoryRouter initialEntries={['/workflows?filter=test-workflow']}>
        <ToastProvider>
          <Routes>
            <Route path="/workflows" element={
              <>
                <WorkflowsTable
                  refreshTrigger={0}
                  loadFlows={mockLoadFlows}
                  workflowFilter="test-workflow"
                />
                <URLTracker onUrlChange={(params) => urlChanges.push(params)} />
              </>
            } />
          </Routes>
        </ToastProvider>
      </MemoryRouter>
    )

    // Wait for the filter to be displayed
    await waitFor(() => {
      expect(screen.getByText('test-workflow')).toBeInTheDocument()
    })

    // Find and click the remove button
    const filterPill = screen.getByText('test-workflow').closest('[data-state-type="workflow"]')
    const removeButton = filterPill?.querySelector('button[aria-label*="Remove"]')
    
    await act(async () => {
      fireEvent.click(removeButton!)
    })

    // WorkflowsTable should properly clear the filter parameter
    await waitFor(() => {
      const lastUrlChange = urlChanges[urlChanges.length - 1]
      expect(lastUrlChange.has('filter')).toBe(false)
    })
  })
})