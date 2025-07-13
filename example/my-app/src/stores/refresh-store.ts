import { create } from 'zustand'

/**
 * Refresh Store - Manages table refresh triggers
 * 
 * This store provides a simple, clean way to communicate between
 * different parts of the app when data needs to be refreshed.
 * 
 * How it works:
 * 1. Actions in the sidebar (DemosSection) call refreshAll() after completing
 * 2. Tables in the main content (TablesSection) use the timestamps as React keys
 * 3. When the key changes, React remounts the component, triggering a fresh data fetch
 * 
 * This approach:
 * - Follows React best practices (using keys for remounting)
 * - Avoids prop drilling
 * - Is easy to understand and extend
 * - Keeps components loosely coupled
 */
interface RefreshStore {
  // Timestamps for last refresh requests
  workflowsRefreshTrigger: number
  statesRefreshTrigger: number
  
  // Actions to trigger refreshes
  refreshWorkflows: () => void
  refreshStates: () => void
  refreshAll: () => void
}

export const useRefreshStore = create<RefreshStore>((set) => ({
  workflowsRefreshTrigger: 0,
  statesRefreshTrigger: 0,
  
  refreshWorkflows: () => set({ workflowsRefreshTrigger: Date.now() }),
  refreshStates: () => set({ statesRefreshTrigger: Date.now() }),
  refreshAll: () => set({ 
    workflowsRefreshTrigger: Date.now(),
    statesRefreshTrigger: Date.now() 
  }),
}))