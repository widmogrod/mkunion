import { useState, useCallback, useMemo } from 'react'
import * as predicate from '../../../../workflow/github_com_widmogrod_mkunion_x_storage_predicate'
import * as schemaless from '../../../../workflow/github_com_widmogrod_mkunion_x_storage_schemaless'
import { PaginatedTableState } from '../types'

interface UsePaginationOptions {
  initialPageSize?: number
  initialOffset?: number
  initialWhere?: predicate.Predicate
  initialSort?: schemaless.SortField[]
}

export function usePagination<T = any>({
  initialPageSize = 10,
  initialOffset = 0,
  initialWhere,
  initialSort
}: UsePaginationOptions = {}) {
  const [state, setState] = useState<PaginatedTableState<T>>({
    limit: initialPageSize,
    offset: initialOffset,
    where: initialWhere,
    sort: initialSort
  })

  const goToNextPage = useCallback(() => {
    setState(prev => ({
      ...prev,
      offset: prev.offset + prev.limit
    }))
  }, [])

  const goToPreviousPage = useCallback(() => {
    setState(prev => ({
      ...prev,
      offset: Math.max(0, prev.offset - prev.limit)
    }))
  }, [])

  const goToFirstPage = useCallback(() => {
    setState(prev => ({
      ...prev,
      offset: 0
    }))
  }, [])

  const goToLastPage = useCallback((totalItems: number) => {
    setState(prev => ({
      ...prev,
      offset: Math.max(0, Math.floor(totalItems / prev.limit) * prev.limit)
    }))
  }, [])

  const setPageSize = useCallback((size: number) => {
    setState(prev => ({
      ...prev,
      limit: size,
      offset: 0 // Reset to first page when changing page size
    }))
  }, [])

  const setWhere = useCallback((where?: predicate.Predicate) => {
    setState(prev => ({
      ...prev,
      where,
      offset: 0 // Reset to first page when filtering
    }))
  }, [])

  const setSort = useCallback((sort?: schemaless.SortField[]) => {
    setState(prev => ({
      ...prev,
      sort,
      offset: 0 // Reset to first page when sorting
    }))
  }, [])

  const updateState = useCallback((updates: Partial<PaginatedTableState<T>>) => {
    setState(prev => ({
      ...prev,
      ...updates
    }))
  }, [])

  const actions = useMemo(() => ({
    goToNextPage,
    goToPreviousPage,
    goToFirstPage,
    goToLastPage,
    setPageSize,
    setWhere,
    setSort,
    updateState
  }), [
    goToNextPage,
    goToPreviousPage,
    goToFirstPage,
    goToLastPage,
    setPageSize,
    setWhere,
    setSort,
    updateState
  ])

  return {
    state,
    actions,
    // Computed values
    currentPage: Math.floor(state.offset / state.limit) + 1,
    hasFilter: !!state.where,
    hasSort: !!state.sort && state.sort.length > 0
  }
}