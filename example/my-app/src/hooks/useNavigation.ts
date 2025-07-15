import { useNavigate, useSearchParams, useLocation } from 'react-router-dom'
import { useCallback } from 'react'

// Custom hook for navigation with context preservation
export function useNavigationWithContext() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const location = useLocation()

  // Navigate to workflows page with optional filter
  const navigateToWorkflows = useCallback((workflowName?: string, workflowId?: string) => {
    const params = new URLSearchParams()
    if (workflowName) params.set('filter', workflowName)
    if (workflowId) params.set('id', workflowId)
    navigate(`/workflows?${params.toString()}`)
  }, [navigate])

  // Navigate to executions page with optional filters
  const navigateToExecutions = useCallback((options?: {
    workflow?: string
    runId?: string
    status?: string[]
    schedule?: string
  }) => {
    const params = new URLSearchParams()
    if (options?.workflow) params.set('workflow', options.workflow)
    if (options?.runId) params.set('runId', options.runId)
    if (options?.status && options.status.length > 0) {
      params.set('status', options.status.join(','))
    }
    if (options?.schedule) params.set('schedule', options.schedule)
    navigate(`/executions?${params.toString()}`)
  }, [navigate])

  // Navigate to schedules page with optional filters
  const navigateToSchedules = useCallback((options?: {
    parentRunId?: string
    focus?: string
    workflow?: string
  }) => {
    const params = new URLSearchParams()
    if (options?.parentRunId) params.set('parentRunId', options.parentRunId)
    if (options?.focus) params.set('focus', options.focus)
    if (options?.workflow) params.set('workflow', options.workflow)
    navigate(`/schedules?${params.toString()}`)
  }, [navigate])

  // Navigate to calendar with optional filters
  const navigateToCalendar = useCallback((options?: {
    workflow?: string
    schedule?: string
    status?: string[]
    date?: string
    view?: 'month' | 'week' | 'day'
  }) => {
    const params = new URLSearchParams()
    if (options?.workflow) params.set('workflow', options.workflow)
    if (options?.schedule) params.set('schedule', options.schedule)
    if (options?.status && options.status.length > 0) {
      params.set('status', options.status.join(','))
    }
    if (options?.date) params.set('date', options.date)
    if (options?.view) params.set('view', options.view)
    navigate(`/calendar?${params.toString()}`)
  }, [navigate])

  // Go back with context preservation
  const goBack = useCallback(() => {
    navigate(-1)
  }, [navigate])

  return {
    navigateToWorkflows,
    navigateToExecutions,
    navigateToSchedules,
    navigateToCalendar,
    goBack,
    currentPath: location.pathname,
    searchParams
  }
}

// Hook to parse and use URL search params
export function useUrlParams() {
  const [searchParams, setSearchParams] = useSearchParams()

  const getParam = useCallback((key: string): string | null => {
    return searchParams.get(key)
  }, [searchParams])

  const getArrayParam = useCallback((key: string): string[] => {
    const value = searchParams.get(key)
    return value ? value.split(',').filter(Boolean) : []
  }, [searchParams])

  const setParam = useCallback((key: string, value: string | null) => {
    setSearchParams(prev => {
      const newParams = new URLSearchParams(prev)
      if (value === null || value === '') {
        newParams.delete(key)
      } else {
        newParams.set(key, value)
      }
      return newParams
    })
  }, [setSearchParams])

  const setArrayParam = useCallback((key: string, values: string[]) => {
    setSearchParams(prev => {
      const newParams = new URLSearchParams(prev)
      if (values.length === 0) {
        newParams.delete(key)
      } else {
        newParams.set(key, values.join(','))
      }
      return newParams
    })
  }, [setSearchParams])

  const clearParams = useCallback(() => {
    setSearchParams(new URLSearchParams())
  }, [setSearchParams])

  return {
    getParam,
    getArrayParam,
    setParam,
    setArrayParam,
    clearParams,
    searchParams
  }
}

// Hook to generate shareable links
export function useShareableLink() {
  const location = useLocation()

  const getShareableLink = useCallback((): string => {
    return `${window.location.origin}${location.pathname}${location.search}`
  }, [location])

  const copyToClipboard = useCallback(async (): Promise<boolean> => {
    try {
      await navigator.clipboard.writeText(getShareableLink())
      return true
    } catch (error) {
      console.error('Failed to copy link:', error)
      return false
    }
  }, [getShareableLink])

  return {
    shareableLink: getShareableLink(),
    copyToClipboard
  }
}