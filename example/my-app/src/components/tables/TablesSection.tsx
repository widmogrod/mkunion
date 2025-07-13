import React from 'react'
import { Alert, AlertDescription, AlertTitle } from '../ui/alert'
import { AlertCircle } from 'lucide-react'
import { useWorkflowApi } from '../../hooks/use-workflow-api'
import { useRefreshStore } from '../../stores/refresh-store'
import { PaginatedTableState } from './PaginatedTable/LegacyAdapter'
import { WorkflowsTable } from './WorkflowsTable'
import { StatesTable } from './StatesTable'
import * as workflow from '../../workflow/github_com_widmogrod_mkunion_x_workflow'

export function TablesSection() {
  const { listFlows, listStates, error } = useWorkflowApi()
  const { workflowsRefreshTrigger, statesRefreshTrigger } = useRefreshStore()
  
  // Memoize the load functions to prevent infinite re-renders
  const loadFlows = React.useCallback(
    (state: PaginatedTableState<workflow.Flow>) => {
      return listFlows({
        limit: state.limit,
        where: state.where ? { Predicate: state.where } : undefined,
        nextPage: state.nextPage,
        prevPage: state.prevPage,
        sort: state.sort
      })
    },
    [listFlows]
  )
  
  const loadStates = React.useCallback(
    (state: PaginatedTableState<workflow.State>) => {
      return listStates({
        limit: state.limit,
        where: state.where ? { Predicate: state.where } : undefined,
        nextPage: state.nextPage,
        prevPage: state.prevPage,
        sort: state.sort
      })
    },
    [listStates]
  )

  return (
    <div className="h-full flex flex-col">
      {error && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>Backend Connection Error</AlertTitle>
          <AlertDescription>
            Unable to connect to the workflow engine backend at localhost:8080. 
            Please ensure the server is running with: <code className="font-mono">go run *.go</code>
          </AlertDescription>
        </Alert>
      )}
      
      <div className="flex flex-col gap-6 flex-1 min-h-0">
        <div className="flex-1 min-h-0 isolate">
          <WorkflowsTable 
            refreshTrigger={workflowsRefreshTrigger}
            loadFlows={loadFlows}
          />
        </div>
        <div className="flex-1 min-h-0 isolate">
          <StatesTable 
            refreshTrigger={statesRefreshTrigger}
            loadStates={loadStates}
          />
        </div>
      </div>
    </div>
  )
}