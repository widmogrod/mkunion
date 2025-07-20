import { useEffect, useRef } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useFilterStore, filtersToUrlParams, urlParamsToFilters } from '../stores/filter-store'

/**
 * Hook to synchronize filter store with URL parameters
 * 
 * This hook ensures bidirectional sync between the filter store and URL:
 * - When URL changes (e.g., user navigates), update the store
 * - When store changes (e.g., user interacts with filters), update the URL
 * 
 * The hook prevents circular updates by tracking the source of changes
 */
export function useExecutionFilterSync() {
  const [searchParams, setSearchParams] = useSearchParams()
  const { executionFilters, setExecutionFilters } = useFilterStore()
  const isUpdatingFromStore = useRef(false)
  const isUpdatingFromUrl = useRef(false)
  
  // Read URL parameters
  const workflowParam = searchParams.get('workflow')
  const statusParam = searchParams.get('status')?.split(',').filter(Boolean) || []
  const runIdParam = searchParams.get('runId')
  const scheduleParam = searchParams.get('schedule')
  
  // Sync from URL to store
  useEffect(() => {
    // Skip if we're updating from store (to prevent circular updates)
    if (isUpdatingFromStore.current) return
    
    const filters = urlParamsToFilters(workflowParam, statusParam)
    
    // Only update if filters have actually changed (deep comparison)
    const currentFilters = useFilterStore.getState().executionFilters
    const filtersChanged = JSON.stringify(filters) !== JSON.stringify(currentFilters)
    
    console.log('useFilterSync: URL to Store sync', {
      workflowParam,
      statusParam,
      parsedFilters: filters,
      currentFilters,
      filtersChanged
    })
    
    if (filtersChanged) {
      isUpdatingFromUrl.current = true
      console.log('useFilterSync: Setting filters in store:', filters)
      setExecutionFilters(filters)
      
      // Reset flag after a microtask to ensure state updates have propagated
      setTimeout(() => {
        isUpdatingFromUrl.current = false
        console.log('useFilterSync: Reset isUpdatingFromUrl flag')
      }, 0)
    }
  }, [workflowParam, statusParam, setExecutionFilters])
  
  // Sync from store to URL
  useEffect(() => {
    console.log('useFilterSync: Store to URL sync triggered', {
      executionFilters,
      isUpdatingFromUrl: isUpdatingFromUrl.current,
      isUpdatingFromStore: isUpdatingFromStore.current
    })
    
    // Skip if we're updating from URL (to prevent circular updates)
    if (isUpdatingFromUrl.current) {
      console.log('useFilterSync: Skipping URL update - currently updating from URL')
      return
    }
    
    // Skip if we're already updating to prevent loops
    if (isUpdatingFromStore.current) {
      console.log('useFilterSync: Skipping URL update - already updating')
      return
    }
    
    const params = filtersToUrlParams(executionFilters)
    console.log('useFilterSync: Computed URL params:', params)
    
    setSearchParams(prev => {
      const newParams = new URLSearchParams(prev)
      
      // Check if URL actually needs updating
      const currentWorkflow = newParams.get('workflow')
      const currentStatus = newParams.get('status')
      
      let needsUpdate = false
      
      // Update workflow parameter
      if (params.workflow === null && currentWorkflow !== null) {
        newParams.delete('workflow')
        needsUpdate = true
        console.log('useFilterSync: Removing workflow param')
      } else if (params.workflow !== null && params.workflow !== currentWorkflow) {
        newParams.set('workflow', params.workflow)
        needsUpdate = true
        console.log('useFilterSync: Setting workflow param:', params.workflow)
      }
      
      // Update status parameter
      if (params.status === null && currentStatus !== null) {
        newParams.delete('status')
        needsUpdate = true
        console.log('useFilterSync: Removing status param')
      } else if (params.status !== null && params.status !== currentStatus) {
        newParams.set('status', params.status)
        needsUpdate = true
        console.log('useFilterSync: Setting status param:', params.status)
      }
      
      if (needsUpdate) {
        isUpdatingFromStore.current = true
        // Reset flag after navigation completes
        setTimeout(() => {
          isUpdatingFromStore.current = false
          console.log('useFilterSync: Reset isUpdatingFromStore flag')
        }, 0)
      }
      
      // Only return new params if something actually changed
      return needsUpdate ? newParams : prev
    })
  }, [executionFilters, setSearchParams])
  
  return {
    // Other URL parameters that aren't managed by the filter store
    runIdFilter: runIdParam,
    scheduleFilter: scheduleParam
  }
}