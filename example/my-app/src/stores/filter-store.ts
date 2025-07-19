import { create } from 'zustand'

/**
 * Filter Store - Manages filter state for tables
 * 
 * This store provides a single source of truth for filter state,
 * decoupling it from URL parameters and component props.
 * 
 * Benefits:
 * - Eliminates circular dependencies between URL and component state
 * - Provides consistent filter management across the app
 * - Makes filter operations predictable and testable
 * - Supports both workflow and status filters
 */

export interface FilterItem {
  stateType: string
  label: string
  color: string
  isExclude: boolean
}

interface FilterStore {
  // State
  executionFilters: FilterItem[]
  workflowFilters: FilterItem[]
  
  // Actions for execution filters
  setExecutionFilters: (filters: FilterItem[]) => void
  addExecutionFilter: (filter: FilterItem) => void
  removeExecutionFilter: (index: number) => void
  toggleExecutionFilterMode: (index: number) => void
  clearExecutionFilters: () => void
  
  // Actions for workflow filters (if needed later)
  setWorkflowFilters: (filters: FilterItem[]) => void
  
  // Utility functions
  getExecutionWorkflowFilters: () => FilterItem[]
  getExecutionStatusFilters: () => FilterItem[]
}

export const useFilterStore = create<FilterStore>((set, get) => ({
  // Initial state
  executionFilters: [],
  workflowFilters: [],
  
  // Actions for execution filters
  setExecutionFilters: (filters) => set({ executionFilters: filters }),
  
  addExecutionFilter: (filter) => set((state) => {
    // Check if filter already exists
    const exists = state.executionFilters.some(
      f => f.stateType === filter.stateType && 
          f.label === filter.label && 
          f.isExclude === filter.isExclude
    )
    if (exists) return state
    
    return { executionFilters: [...state.executionFilters, filter] }
  }),
  
  removeExecutionFilter: (index) => set((state) => ({
    executionFilters: state.executionFilters.filter((_, i) => i !== index)
  })),
  
  toggleExecutionFilterMode: (index) => set((state) => ({
    executionFilters: state.executionFilters.map((filter, i) => 
      i === index ? { ...filter, isExclude: !filter.isExclude } : filter
    )
  })),
  
  clearExecutionFilters: () => set({ executionFilters: [] }),
  
  // Actions for workflow filters
  setWorkflowFilters: (filters) => set({ workflowFilters: filters }),
  
  // Utility functions
  getExecutionWorkflowFilters: () => {
    const state = get()
    return state.executionFilters.filter(f => f.stateType === 'workflow')
  },
  
  getExecutionStatusFilters: () => {
    const state = get()
    return state.executionFilters.filter(f => f.stateType !== 'workflow')
  }
}))

// Helper functions for URL synchronization
export const filtersToUrlParams = (filters: FilterItem[]): Record<string, string | null> => {
  const params: Record<string, string | null> = {}
  
  // Extract workflow filters
  const workflowFilters = filters.filter(f => f.stateType === 'workflow')
  if (workflowFilters.length > 0) {
    params.workflow = workflowFilters[0].label
  } else {
    params.workflow = null
  }
  
  // Extract status filters
  const statusFilters = filters.filter(f => f.stateType !== 'workflow')
  if (statusFilters.length > 0) {
    // Map state types back to URL-friendly status names
    const statusToUrlMap: Record<string, string> = {
      'workflow.Done': 'done',
      'workflow.Error': 'error', 
      'workflow.Await': 'await',
      'workflow.Scheduled': 'scheduled',
      'workflow.ScheduleStopped': 'paused',
      'workflow.NextOperation': 'next'
    }
    
    const statusParams = statusFilters.map(f => statusToUrlMap[f.stateType] || f.stateType)
    params.status = statusParams.join(',')
  } else {
    params.status = null
  }
  
  return params
}

export const urlParamsToFilters = (
  workflowParam: string | null,
  statusParam: string[]
): FilterItem[] => {
  const filters: FilterItem[] = []
  
  // Handle workflow filter
  if (workflowParam) {
    filters.push({
      stateType: 'workflow',
      label: workflowParam,
      color: '#3b82f6', // Blue color for workflow filters
      isExclude: false
    })
  }
  
  // Handle status filters
  if (statusParam && statusParam.length > 0) {
    const statusToType: Record<string, string> = {
      'done': 'workflow.Done',
      'error': 'workflow.Error',
      'await': 'workflow.Await',
      'scheduled': 'workflow.Scheduled',
      'paused': 'workflow.ScheduleStopped',
      'next': 'workflow.NextOperation'
    }
    
    const STATE_TYPE_CONFIG: Record<string, { label: string; color: string }> = {
      'workflow.Done': { label: 'Done', color: '#10b981' },
      'workflow.Error': { label: 'Error', color: '#ef4444' },
      'workflow.Await': { label: 'Await', color: '#3b82f6' },
      'workflow.Scheduled': { label: 'Scheduled', color: '#eab308' },
      'workflow.ScheduleStopped': { label: 'Paused', color: '#6b7280' },
      'workflow.NextOperation': { label: 'Next', color: '#a855f7' },
    }
    
    statusParam.forEach(status => {
      const stateType = statusToType[status.toLowerCase()] || status
      const config = STATE_TYPE_CONFIG[stateType]
      
      if (config) {
        filters.push({
          stateType,
          label: config.label,
          color: config.color,
          isExclude: false
        })
      }
    })
  }
  
  return filters
}