import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { ExecutionsTable } from '../ExecutionsTable'
import { BrowserRouter, useSearchParams } from 'react-router-dom'
import { ToastProvider } from '../../../contexts/ToastContext'
import '@testing-library/jest-dom'

// Mock the API hooks
jest.mock('../../../hooks/use-workflow-api', () => ({
  useWorkflowApi: () => ({
    deleteStates: jest.fn(),
    tryRecover: jest.fn(),
  }),
}))

// Mock data for testing
const mockLoadStates = jest.fn().mockResolvedValue({
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
              'workflow.Flow': {
                Name: 'test-workflow',
              },
            },
          },
        },
      },
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
              'workflow.FlowRef': {
                FlowID: 'another-workflow',
              },
            },
          },
        },
      },
    },
  ],
  Next: null,
})

// Test wrapper component to capture URL changes
const TestWrapper: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  return (
    <BrowserRouter>
      <ToastProvider>
        {children}
      </ToastProvider>
    </BrowserRouter>
  )
}

// Helper component to display current URL params
const URLDisplay: React.FC = () => {
  const [searchParams] = useSearchParams()
  return (
    <div data-testid="url-params">
      {Array.from(searchParams.entries()).map(([key, value]) => (
        <span key={key} data-testid={`url-param-${key}`}>
          {key}={value}
        </span>
      ))}
    </div>
  )
}

describe('ExecutionsTable Filter Operations', () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  describe('URL Synchronization', () => {
    it('should update URL when adding a status filter', async () => {
      const { container } = render(
        <TestWrapper>
          <ExecutionsTable
            refreshTrigger={0}
            loadStates={mockLoadStates}
          />
          <URLDisplay />
        </TestWrapper>
      )

      // Wait for table to load
      await waitFor(() => {
        expect(screen.getByText(/test-workflow/)).toBeInTheDocument()
      })

      // Click on the Done status pill to add it as a filter
      const donePill = container.querySelector('[data-state-type="workflow.Done"]')
      expect(donePill).toBeInTheDocument()
      fireEvent.click(donePill!)

      // Check that URL was updated with status filter
      await waitFor(() => {
        const statusParam = screen.getByTestId('url-param-status')
        expect(statusParam).toHaveTextContent('status=done')
      })
    })

    it('should clear specific URL param when removing a filter', async () => {
      // Start with filters in URL
      window.history.pushState({}, '', '?workflow=test-workflow&status=done')

      const { container } = render(
        <TestWrapper>
          <ExecutionsTable
            refreshTrigger={0}
            loadStates={mockLoadStates}
            workflowFilter="test-workflow"
            statusFilter={['done']}
          />
          <URLDisplay />
        </TestWrapper>
      )

      // Wait for filters to be displayed
      await waitFor(() => {
        expect(screen.getByText('Filtering:')).toBeInTheDocument()
      })

      // Find and click remove button on workflow filter
      const workflowFilterPill = container.querySelector('[data-state-type="workflow"]')
      const removeButton = workflowFilterPill?.querySelector('button')
      expect(removeButton).toBeInTheDocument()
      fireEvent.click(removeButton!)

      // Check that only workflow param was removed, status remains
      await waitFor(() => {
        const urlParams = screen.getByTestId('url-params')
        expect(urlParams).not.toHaveTextContent('workflow=')
        expect(urlParams).toHaveTextContent('status=done')
      })
    })

    it('should toggle filter mode and update URL', async () => {
      window.history.pushState({}, '', '?status=done')

      const { container } = render(
        <TestWrapper>
          <ExecutionsTable
            refreshTrigger={0}
            loadStates={mockLoadStates}
            statusFilter={['done']}
          />
          <URLDisplay />
        </TestWrapper>
      )

      // Wait for filter to be displayed
      await waitFor(() => {
        expect(screen.getByText('Filtering:')).toBeInTheDocument()
      })

      // Click on the filter pill to toggle exclude mode
      const filterPill = container.querySelector('[data-state-type="workflow.Done"]')
      expect(filterPill).toBeInTheDocument()
      fireEvent.click(filterPill!)

      // Verify the filter is still in URL (toggling mode shouldn't remove it)
      await waitFor(() => {
        const statusParam = screen.getByTestId('url-param-status')
        expect(statusParam).toHaveTextContent('status=done')
      })

      // Verify the visual indicator changed to exclude mode
      const excludeIndicator = filterPill?.querySelector('.text-destructive')
      expect(excludeIndicator).toBeInTheDocument()
    })

    it('should clear all URL params when clicking Clear all', async () => {
      window.history.pushState({}, '', '?workflow=test-workflow&status=done,error')

      render(
        <TestWrapper>
          <ExecutionsTable
            refreshTrigger={0}
            loadStates={mockLoadStates}
            workflowFilter="test-workflow"
            statusFilter={['done', 'error']}
          />
          <URLDisplay />
        </TestWrapper>
      )

      // Wait for filters and clear button to be displayed
      await waitFor(() => {
        expect(screen.getByText('Clear all')).toBeInTheDocument()
      })

      // Click Clear all button
      fireEvent.click(screen.getByText('Clear all'))

      // Check that all filter-related URL params were removed
      await waitFor(() => {
        const urlParams = screen.getByTestId('url-params')
        expect(urlParams).not.toHaveTextContent('workflow=')
        expect(urlParams).not.toHaveTextContent('status=')
        expect(urlParams.textContent).toBe('')
      })
    })

    it('should handle multiple status filters correctly', async () => {
      const { container } = render(
        <TestWrapper>
          <ExecutionsTable
            refreshTrigger={0}
            loadStates={mockLoadStates}
          />
          <URLDisplay />
        </TestWrapper>
      )

      // Wait for table to load
      await waitFor(() => {
        expect(screen.getByText(/test-workflow/)).toBeInTheDocument()
      })

      // Add Done filter
      const donePill = container.querySelector('[data-state-type="workflow.Done"]')
      fireEvent.click(donePill!)

      // Add Error filter
      const errorPill = container.querySelector('[data-state-type="workflow.Error"]')
      fireEvent.click(errorPill!)

      // Check that URL has both status filters
      await waitFor(() => {
        const statusParam = screen.getByTestId('url-param-status')
        expect(statusParam.textContent).toMatch(/status=done,error|status=error,done/)
      })
    })
  })

  describe('Filter Persistence', () => {
    it('should restore filters from URL on mount', async () => {
      window.history.pushState({}, '', '?workflow=test-workflow&status=done')

      render(
        <TestWrapper>
          <ExecutionsTable
            refreshTrigger={0}
            loadStates={mockLoadStates}
            workflowFilter="test-workflow"
            statusFilter={['done']}
          />
        </TestWrapper>
      )

      // Verify filters are displayed
      await waitFor(() => {
        expect(screen.getByText('Filtering:')).toBeInTheDocument()
        expect(screen.getByText('test-workflow')).toBeInTheDocument()
        expect(screen.getByText('Done')).toBeInTheDocument()
      })
    })

    it('should maintain filters after page refresh', async () => {
      // Set initial URL state
      window.history.pushState({}, '', '?workflow=my-workflow&status=error')

      // Render component
      const { unmount } = render(
        <TestWrapper>
          <ExecutionsTable
            refreshTrigger={0}
            loadStates={mockLoadStates}
            workflowFilter="my-workflow"
            statusFilter={['error']}
          />
        </TestWrapper>
      )

      // Verify filters are displayed
      await waitFor(() => {
        expect(screen.getByText('my-workflow')).toBeInTheDocument()
        expect(screen.getByText('Error')).toBeInTheDocument()
      })

      // Simulate page refresh by unmounting and remounting
      unmount()

      render(
        <TestWrapper>
          <ExecutionsTable
            refreshTrigger={0}
            loadStates={mockLoadStates}
            workflowFilter="my-workflow"
            statusFilter={['error']}
          />
        </TestWrapper>
      )

      // Verify filters are still displayed after "refresh"
      await waitFor(() => {
        expect(screen.getByText('my-workflow')).toBeInTheDocument()
        expect(screen.getByText('Error')).toBeInTheDocument()
      })
    })
  })

  describe('Edge Cases', () => {
    it('should handle removing last filter correctly', async () => {
      window.history.pushState({}, '', '?status=done')

      const { container } = render(
        <TestWrapper>
          <ExecutionsTable
            refreshTrigger={0}
            loadStates={mockLoadStates}
            statusFilter={['done']}
          />
          <URLDisplay />
        </TestWrapper>
      )

      // Wait for filter to be displayed
      await waitFor(() => {
        expect(screen.getByText('Done')).toBeInTheDocument()
      })

      // Remove the only filter
      const removeButton = container.querySelector('[data-state-type="workflow.Done"] button')
      fireEvent.click(removeButton!)

      // Verify URL is completely clean
      await waitFor(() => {
        const urlParams = screen.getByTestId('url-params')
        expect(urlParams.textContent).toBe('')
      })

      // Verify "Filtering:" label is not shown
      expect(screen.queryByText('Filtering:')).not.toBeInTheDocument()
    })

    it('should handle rapid filter changes without race conditions', async () => {
      const { container } = render(
        <TestWrapper>
          <ExecutionsTable
            refreshTrigger={0}
            loadStates={mockLoadStates}
          />
          <URLDisplay />
        </TestWrapper>
      )

      // Rapidly add multiple filters
      const donePill = container.querySelector('[data-state-type="workflow.Done"]')
      const errorPill = container.querySelector('[data-state-type="workflow.Error"]')
      
      fireEvent.click(donePill!)
      fireEvent.click(errorPill!)
      fireEvent.click(donePill!) // Click again to remove
      
      // Final state should only have error filter
      await waitFor(() => {
        const statusParam = screen.getByTestId('url-param-status')
        expect(statusParam).toHaveTextContent('status=error')
      })
    })
  })
})