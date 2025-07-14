import React from 'react'
import { Alert, AlertDescription, AlertTitle } from '../ui/alert'
import { AlertCircle, Database } from 'lucide-react'
import { useWorkflowApi } from '../../hooks/use-workflow-api'
import { useRefreshStore } from '../../stores/refresh-store'
import { WorkflowsTable } from './WorkflowsTable'
import { StatesTable } from './StatesTable'
import { PageHeader } from '../layout/PageHeader'
import * as predicate from '../../workflow/github_com_widmogrod_mkunion_x_storage_predicate'

// Shared type for table load state that matches API expectations
export interface TableLoadState {
  limit: number
  offset: number
  where?: predicate.Predicate
  sort?: { [key: string]: boolean }
}

export function TablesSection() {
  const { listFlows, listStates, error } = useWorkflowApi()
  const { workflowsRefreshTrigger, statesRefreshTrigger } = useRefreshStore()
  
  // Memoize the load functions to prevent infinite re-renders
  const loadFlows = React.useCallback(
    (state: TableLoadState) => {
      return listFlows({
        limit: state.limit,
        where: state.where ? { Predicate: state.where } : undefined,
        sort: state.sort
      })
    },
    [listFlows]
  )
  
  const loadStates = React.useCallback(
    (state: TableLoadState) => {
      return listStates({
        limit: state.limit,
        where: state.where ? { Predicate: state.where } : undefined,
        sort: state.sort
      })
    },
    [listStates]
  )

  return (
    <div className="h-full flex flex-col">
      <PageHeader
        icon={Database}
        title="Workflows & States"
        description="Manage workflow definitions and monitor their execution states"
      />
      
      {error && (
        <Alert variant="destructive" className="mx-6 mt-6">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>Backend Connection Error</AlertTitle>
          <AlertDescription>
            Unable to connect to the workflow engine backend at localhost:8080. 
            Please ensure the server is running with: <code className="font-mono">go run *.go</code>
          </AlertDescription>
        </Alert>
      )}
      
      <div className="flex flex-col gap-6 flex-1 min-h-0 p-6">
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