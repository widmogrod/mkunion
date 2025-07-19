import { renderHook, act } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { useUrlFilters } from '../hooks/useUrlFilters'

// Wrapper component for React Router
const createWrapper = (initialEntries: string[] = ['/']) => {
  return ({ children }: any) => (
    <MemoryRouter initialEntries={initialEntries}>
      {children}
    </MemoryRouter>
  )
}

describe('useUrlFilters', () => {
  test('should correctly parse excluded status filters from URL', () => {
    const { result } = renderHook(() => useUrlFilters(), {
      wrapper: createWrapper(['/executions?status=!done,!error'])
    })
    
    expect(result.current.filters.status).toEqual(['!done', '!error'])
  })
  
  test('should remove excluded status filter correctly', () => {
    const { result } = renderHook(() => useUrlFilters(), {
      wrapper: createWrapper(['/executions?status=!done,!error'])
    })
    
    // Remove !done filter
    act(() => {
      result.current.removeStatusFilter('!done')
    })
    
    expect(result.current.filters.status).toEqual(['!error'])
  })
  
  test('should toggle filter from exclude to include', () => {
    const { result } = renderHook(() => useUrlFilters(), {
      wrapper: createWrapper(['/executions?status=!done'])
    })
    
    // Toggle from exclude to include
    act(() => {
      result.current.toggleStatusFilterMode('done')
    })
    
    expect(result.current.filters.status).toEqual(['done'])
  })
  
  test('should toggle filter from include to exclude', () => {
    const { result } = renderHook(() => useUrlFilters(), {
      wrapper: createWrapper(['/executions?status=done'])
    })
    
    // Toggle from include to exclude
    act(() => {
      result.current.toggleStatusFilterMode('done')
    })
    
    expect(result.current.filters.status).toEqual(['!done'])
  })
  
  test('should handle mixed include and exclude filters', () => {
    const { result } = renderHook(() => useUrlFilters(), {
      wrapper: createWrapper(['/executions?status=done,!error,await'])
    })
    
    expect(result.current.filters.status).toEqual(['done', '!error', 'await'])
    
    // Remove the excluded filter
    act(() => {
      result.current.removeStatusFilter('!error')
    })
    
    expect(result.current.filters.status).toEqual(['done', 'await'])
  })
  
  test('should clear all filters', () => {
    const { result } = renderHook(() => useUrlFilters(), {
      wrapper: createWrapper(['/executions?workflow=test&status=!done,!error'])
    })
    
    expect(result.current.filters.workflow).toBe('test')
    expect(result.current.filters.status).toEqual(['!done', '!error'])
    
    // Clear all filters
    act(() => {
      result.current.clearAllFilters()
    })
    
    expect(result.current.filters.workflow).toBeNull()
    expect(result.current.filters.status).toEqual([])
  })
})