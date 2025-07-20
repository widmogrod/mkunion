import { useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'

/**
 * Simple hook for managing filters directly in URL
 * 
 * This eliminates the need for complex state sync by:
 * 1. Using URL as the single source of truth
 * 2. Providing simple methods to update URL parameters
 * 3. No store, no sync, no timing issues
 */
export function useUrlFilters() {
  const [searchParams, setSearchParams] = useSearchParams()
  
  // Parse filters from URL
  const filters = {
    workflow: searchParams.get('workflow'),
    status: searchParams.get('status')?.split(',').filter(Boolean) || [],
    runId: searchParams.get('runId'),
    schedule: searchParams.get('schedule')
  }
  
  // Add or update a filter
  const setFilter = useCallback((key: string, value: string | string[] | null) => {
    setSearchParams(prev => {
      const newParams = new URLSearchParams(prev)
      
      if (value === null || (Array.isArray(value) && value.length === 0)) {
        newParams.delete(key)
      } else if (Array.isArray(value)) {
        newParams.set(key, value.join(','))
      } else {
        newParams.set(key, value)
      }
      
      return newParams
    })
  }, [setSearchParams])
  
  // Add a status filter
  const addStatusFilter = useCallback((status: string) => {
    const currentStatuses = filters.status
    if (!currentStatuses.includes(status)) {
      setFilter('status', [...currentStatuses, status])
    }
  }, [filters.status, setFilter])
  
  // Remove a status filter
  const removeStatusFilter = useCallback((status: string) => {
    const currentStatuses = filters.status
    setFilter('status', currentStatuses.filter(s => s !== status))
  }, [filters.status, setFilter])
  
  // Toggle status filter exclude mode
  const toggleStatusFilterMode = useCallback((status: string) => {
    const currentStatuses = filters.status
    const normalStatus = status.replace(/^!/, '')
    
    // Check if we have this status in any form
    const hasExcluded = currentStatuses.includes(`!${normalStatus}`)
    const hasIncluded = currentStatuses.includes(normalStatus)
    
    if (hasExcluded) {
      // Change from exclude to include
      setFilter('status', currentStatuses.map(s => s === `!${normalStatus}` ? normalStatus : s))
    } else if (hasIncluded) {
      // Change from include to exclude
      setFilter('status', currentStatuses.map(s => s === normalStatus ? `!${normalStatus}` : s))
    }
  }, [filters.status, setFilter])
  
  // Toggle workflow filter exclude mode
  const toggleWorkflowFilterMode = useCallback(() => {
    const currentWorkflow = filters.workflow
    if (!currentWorkflow) return
    
    const isExcluded = currentWorkflow.startsWith('!')
    const normalWorkflow = currentWorkflow.replace(/^!/, '')
    
    if (isExcluded) {
      // Change from exclude to include
      setFilter('workflow', normalWorkflow)
    } else {
      // Change from include to exclude  
      setFilter('workflow', `!${normalWorkflow}`)
    }
  }, [filters.workflow, setFilter])
  
  // Clear all filters
  const clearAllFilters = useCallback(() => {
    setSearchParams(prev => {
      const newParams = new URLSearchParams(prev)
      newParams.delete('workflow')
      newParams.delete('status')
      // Preserve runId and schedule as they might be navigation params
      return newParams
    })
  }, [setSearchParams])
  
  // Clear specific filter type
  const clearFilter = useCallback((key: string) => {
    setFilter(key, null)
  }, [setFilter])
  
  return {
    // Current filter values
    filters,
    
    // Filter management
    setFilter,
    addStatusFilter,
    removeStatusFilter,
    toggleStatusFilterMode,
    toggleWorkflowFilterMode,
    clearAllFilters,
    clearFilter,
    
    // Computed values
    hasFilters: !!(filters.workflow || filters.status.length > 0),
    activeFilterCount: (filters.workflow ? 1 : 0) + filters.status.length
  }
}