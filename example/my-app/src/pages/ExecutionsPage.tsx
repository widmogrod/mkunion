import React from 'react'
import { Activity } from 'lucide-react'
import { Alert, AlertDescription, AlertTitle } from '../components/ui/alert'
import { AlertCircle } from 'lucide-react'
import { useWorkflowApi } from '../hooks/use-workflow-api'
import { useRefreshStore } from '../stores/refresh-store'
import { ExecutionsTable } from '../components/tables/ExecutionsTable'
import { PageHeader } from '../components/layout/PageHeader'
import { TableLoadState } from '../components/tables/TablesSection'
import { useUrlParams } from '../hooks/useNavigation'

export function ExecutionsPage() {
  const { listStates, error } = useWorkflowApi()
  const { executionsRefreshTrigger } = useRefreshStore()
  const { getParam, getArrayParam } = useUrlParams()
  
  // Get URL parameters
  const workflowFilter = getParam('workflow')
  const runIdFilter = getParam('runId')
  const statusFilter = getArrayParam('status')
  const scheduleFilter = getParam('schedule')
  
  // Memoize the load function to prevent infinite re-renders
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
        icon={Activity}
        title="Executions"
        description="Monitor workflow executions and their states"
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
      
      <div className="flex-1 p-6 min-h-0">
        <ExecutionsTable 
          refreshTrigger={executionsRefreshTrigger}
          loadStates={loadStates}
          workflowFilter={workflowFilter}
          runIdFilter={runIdFilter}
          statusFilter={statusFilter}
          scheduleFilter={scheduleFilter}
        />
      </div>
    </div>
  )
}