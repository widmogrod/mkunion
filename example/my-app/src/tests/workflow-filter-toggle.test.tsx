import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { ExecutionsTableSimple } from '../components/tables/ExecutionsTableSimple'

// Mock the API and other dependencies
jest.mock('../hooks/use-workflow-api', () => ({
  useWorkflowApi: () => ({
    deleteStates: jest.fn(),
    tryRecover: jest.fn()
  })
}))

jest.mock('../contexts/ToastContext', () => ({
  useToast: () => ({
    success: jest.fn(),
    error: jest.fn(),
    warning: jest.fn()
  })
}))

const mockLoadStates = jest.fn().mockResolvedValue({
  Items: [],
  Next: null
})

describe('Workflow Filter Toggle', () => {
  test('should toggle workflow filter between include and exclude modes', async () => {
    const TestWrapper = () => {
      return (
        <MemoryRouter initialEntries={['/executions?workflow=scheduled_demo']}>
          <Routes>
            <Route path="/executions" element={
              <ExecutionsTableSimple 
                refreshTrigger={0}
                loadStates={mockLoadStates}
              />
            } />
          </Routes>
        </MemoryRouter>
      )
    }
    
    render(<TestWrapper />)
    
    // Wait for the filter to appear
    await waitFor(() => {
      expect(screen.getByText('scheduled_demo')).toBeInTheDocument()
    })
    
    // Find the filter pill by looking for the one with scheduled_demo
    const filterPill = screen.getByText('scheduled_demo').closest('div')
    expect(filterPill).toBeInTheDocument()
    
    // Click on the filter to toggle it
    if (filterPill) {
      fireEvent.click(filterPill)
    }
    
    // Check that the URL has been updated to exclude mode
    await waitFor(() => {
      const currentUrl = window.location.href
      expect(currentUrl).toContain('workflow=!scheduled_demo')
    })
  })
  
  test('should toggle workflow filter from exclude back to include mode', async () => {
    const TestWrapper = () => {
      return (
        <MemoryRouter initialEntries={['/executions?workflow=!scheduled_demo']}>
          <Routes>
            <Route path="/executions" element={
              <ExecutionsTableSimple 
                refreshTrigger={0}
                loadStates={mockLoadStates}
              />
            } />
          </Routes>
        </MemoryRouter>
      )
    }
    
    render(<TestWrapper />)
    
    // Wait for the filter to appear
    await waitFor(() => {
      expect(screen.getByText('scheduled_demo')).toBeInTheDocument()
    })
    
    // Find the filter pill
    const filterPill = screen.getByText('scheduled_demo').closest('div')
    expect(filterPill).toBeInTheDocument()
    
    // The filter should show as excluded (check for visual indicator if any)
    // Click on the filter to toggle it back to include
    if (filterPill) {
      fireEvent.click(filterPill)
    }
    
    // Check that the URL has been updated to include mode
    await waitFor(() => {
      const currentUrl = window.location.href
      expect(currentUrl).toContain('workflow=scheduled_demo')
      expect(currentUrl).not.toContain('workflow=!')
    })
  })
})