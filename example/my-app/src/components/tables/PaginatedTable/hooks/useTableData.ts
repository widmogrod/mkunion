import { useState, useEffect, useCallback, useRef } from 'react'
import { PaginatedTableState, PaginatedData } from '../types'

interface UseTableDataOptions {
  debounceMs?: number
  retryOnError?: boolean
  maxRetries?: number
}

export function useTableData<T>(
  loadData: (state: PaginatedTableState<T>) => Promise<PaginatedData<T>>,
  state: PaginatedTableState<T>,
  options: UseTableDataOptions = {}
) {
  const { debounceMs = 0, retryOnError = false, maxRetries = 3 } = options
  
  const [data, setData] = useState<PaginatedData<T>>({ items: [] })
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<Error | null>(null)
  const [retryCount, setRetryCount] = useState(0)
  const retryCountRef = useRef(0)
  
  // Track if component is actually mounted (StrictMode-compatible)
  const mountedRef = useRef(true)
  
  // Track request sequence to ignore stale requests
  const requestSequenceRef = useRef(0)
  
  const abortControllerRef = useRef<AbortController | null>(null)
  const timeoutRef = useRef<NodeJS.Timeout | null>(null)

  // Store loadData in a ref to avoid re-creating fetchData
  const loadDataRef = useRef(loadData)
  loadDataRef.current = loadData
  
  // Store state in a ref to avoid dependencies
  const stateRef = useRef(state)
  stateRef.current = state
  
  // Store options in refs to avoid dependencies
  const optionsRef = useRef({ debounceMs, retryOnError, maxRetries })
  optionsRef.current = { debounceMs, retryOnError, maxRetries }

  const fetchDataFn = useCallback(async () => {
    // Increment request sequence - this request ID
    const currentRequestId = ++requestSequenceRef.current
    
    console.log('useTableData: Starting fetch with request ID:', currentRequestId)
    
    // Only abort previous request if there was one
    if (abortControllerRef.current) {
      console.log('useTableData: Aborting previous request')
      abortControllerRef.current.abort()
    }
    
    // Clear any pending timeouts
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current)
    }

    // Create new abort controller for this request
    const currentController = new AbortController()
    abortControllerRef.current = currentController
    
    const doFetch = async () => {
      try {
        setLoading(true)
        setError(null)
        
        const result = await loadDataRef.current(stateRef.current)
        
        console.log('useTableData: Got result from loadData', {
          requestId: currentRequestId,
          resultItems: result.items.length,
          hasNext: !!result.next
        })
        
        // StrictMode-compatible checks:
        // 1. Is component still mounted?
        // 2. Is this still the latest request?
        if (!mountedRef.current) {
          console.log('useTableData: Component unmounted, ignoring result')
          return
        }
        
        if (currentRequestId !== requestSequenceRef.current) {
          console.log('useTableData: Stale request, ignoring result. Current:', currentRequestId, 'Latest:', requestSequenceRef.current)
          return
        }
        
        if (currentController.signal.aborted) {
          console.log('useTableData: Request was aborted, ignoring result')
          return
        }
        
        setData(result)
        console.log('useTableData: Data set successfully, items count:', result.items.length)
        
        setRetryCount(0) // Reset retry count on success
        retryCountRef.current = 0
      } catch (err) {
        // Ignore aborted requests
        if (err instanceof Error && err.name === 'AbortError') {
          return
        }
        
        const error = err as Error
        setError(error)
        
        // Retry logic
        if (optionsRef.current.retryOnError && retryCountRef.current < optionsRef.current.maxRetries) {
          const newRetryCount = retryCountRef.current + 1
          setRetryCount(newRetryCount)
          retryCountRef.current = newRetryCount
          // Exponential backoff
          const retryDelay = Math.min(1000 * Math.pow(2, retryCountRef.current), 10000)
          timeoutRef.current = setTimeout(doFetch, retryDelay)
        }
      } finally {
        setLoading(false)
      }
    }

    // Apply debounce if specified
    if (optionsRef.current.debounceMs > 0) {
      timeoutRef.current = setTimeout(doFetch, optionsRef.current.debounceMs)
    } else {
      await doFetch()
    }
  }, [])
  
  // Store fetchData in a ref to use in useEffect without dependency
  const fetchDataRef = useRef(fetchDataFn)
  fetchDataRef.current = fetchDataFn

  // Use a ref to track the previous state for deep comparison
  const prevStateRef = useRef<string>('')
  const currentStateStr = JSON.stringify(state)

  useEffect(() => {
    // Only fetch if the state actually changed
    if (prevStateRef.current !== currentStateStr) {
      console.log('TableData: State changed, fetching data', { 
        prev: prevStateRef.current, 
        current: currentStateStr 
      })
      prevStateRef.current = currentStateStr
      fetchDataRef.current().catch(console.error)
    } else {
      console.log('TableData: State unchanged, skipping fetch')
    }
  }, [currentStateStr])

  // Cleanup effect for component lifecycle
  useEffect(() => {
    // Set mounted to true on mount (handles StrictMode remounting)
    mountedRef.current = true
    
    return () => {
      // Mark as unmounted
      mountedRef.current = false
      
      console.log('useTableData: Component cleanup')
      
      // Clean up timeouts (these should always be cleaned up)
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current)
      }
      
      // Don't abort ongoing requests on cleanup - let them complete
      // Only abort if they're truly no longer needed (handled by new requests)
    }
  }, [])

  const refresh = useCallback(() => {
    setRetryCount(0)
    retryCountRef.current = 0
    fetchDataRef.current().catch(console.error)
  }, [])

  console.log('useTableData: Return state', {
    dataItems: data.items.length,
    loading,
    error: !!error,
    currentData: data
  })

  return {
    data,
    loading,
    error,
    refresh,
    hasData: data.items.length > 0,
    isEmpty: !loading && data.items.length === 0
  }
}