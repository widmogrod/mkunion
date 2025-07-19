import { useCallback } from 'react'
import { useNavigate, useLocation, useSearchParams } from 'react-router-dom'

/**
 * Enhanced navigation hook that preserves existing URL parameters
 * 
 * This solves the issue where navigation was clearing filters by:
 * 1. Merging new parameters with existing ones
 * 2. Only removing parameters explicitly set to null
 * 3. Preserving unrelated parameters during navigation
 */
export function useNavigationPreserveFilters() {
  const navigate = useNavigate()
  const location = useLocation()
  const [currentSearchParams] = useSearchParams()
  
  // Helper to merge new params with existing ones
  const mergeParams = useCallback((newParams: Record<string, string | null | undefined>) => {
    const merged = new URLSearchParams(currentSearchParams)
    
    Object.entries(newParams).forEach(([key, value]) => {
      if (value === null) {
        merged.delete(key)
      } else if (value !== undefined) {
        merged.set(key, value)
      }
      // If value is undefined, keep existing param unchanged
    })
    
    return merged
  }, [currentSearchParams])
  
  // Navigate to workflows with filter preservation
  const navigateToWorkflows = useCallback((workflow?: string, executionId?: string | null) => {
    const params = mergeParams({
      filter: workflow || null,
      execution: executionId || null
    })
    navigate(`/workflows?${params.toString()}`)
  }, [navigate, mergeParams])
  
  // Navigate to executions with filter preservation
  const navigateToExecutions = useCallback((options?: {
    workflow?: string | null
    runId?: string | null
    status?: string[] | null
    schedule?: string | null
  }) => {
    const params = mergeParams({
      workflow: options?.workflow !== undefined ? options.workflow : undefined,
      runId: options?.runId !== undefined ? options.runId : undefined,
      status: options?.status !== undefined ? (options.status ? options.status.join(',') : null) : undefined,
      schedule: options?.schedule !== undefined ? options.schedule : undefined
    })
    
    // Remove undefined entries (preserve existing)
    const filtered: Record<string, string | null> = {}
    Object.entries({
      workflow: options?.workflow !== undefined ? options.workflow : undefined,
      runId: options?.runId !== undefined ? options.runId : undefined,
      status: options?.status !== undefined ? (options.status ? options.status.join(',') : null) : undefined,
      schedule: options?.schedule !== undefined ? options.schedule : undefined
    }).forEach(([key, value]) => {
      if (value !== undefined) {
        filtered[key] = value
      }
    })
    
    const mergedParams = mergeParams(filtered)
    navigate(`/executions?${mergedParams.toString()}`)
  }, [navigate, mergeParams])
  
  // Navigate to schedules with filter preservation
  const navigateToSchedules = useCallback((options?: {
    parentRunId?: string
    focus?: string
    workflow?: string
  }) => {
    const params = mergeParams({
      parentRunId: options?.parentRunId || null,
      focus: options?.focus || null,
      workflow: options?.workflow || null
    })
    navigate(`/schedules?${params.toString()}`)
  }, [navigate, mergeParams])
  
  // Navigate to calendar with filter preservation
  const navigateToCalendar = useCallback((options?: {
    workflow?: string
    schedule?: string
    status?: string[]
    date?: string
    view?: 'month' | 'week' | 'day'
  }) => {
    const params = mergeParams({
      workflow: options?.workflow || null,
      schedule: options?.schedule || null,
      status: options?.status?.join(',') || null,
      date: options?.date || null,
      view: options?.view || null
    })
    navigate(`/calendar?${params.toString()}`)
  }, [navigate, mergeParams])
  
  return {
    navigateToWorkflows,
    navigateToExecutions,
    navigateToSchedules,
    navigateToCalendar,
    goBack: () => navigate(-1),
    currentPath: location.pathname,
    searchParams: currentSearchParams
  }
}