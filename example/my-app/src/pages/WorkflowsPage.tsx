import React from 'react'
import { Database } from 'lucide-react'
import { Alert, AlertDescription, AlertTitle } from '../components/ui/alert'
import { AlertCircle } from 'lucide-react'
import { useWorkflowApi } from '../hooks/use-workflow-api'
import { useRefreshStore } from '../stores/refresh-store'
import { WorkflowsTable } from '../components/tables/WorkflowsTable'
import { PageHeader } from '../components/layout/PageHeader'
import { TableLoadState } from '../components/tables/TablesSection'
import { ShareLinkButton } from '../components/navigation/ShareLinkButton'

export function WorkflowsPage() {
  const { listFlows, error } = useWorkflowApi()
  const { workflowsRefreshTrigger } = useRefreshStore()
  
  // Memoize the load function to prevent infinite re-renders
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

  return (
    <div className="h-full flex flex-col">
      <PageHeader
        icon={Database}
        title="Workflows"
        description="Manage and monitor your workflow definitions"
        actions={<ShareLinkButton />}
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
        <WorkflowsTable 
          refreshTrigger={workflowsRefreshTrigger}
          loadFlows={loadFlows}
        />
      </div>
    </div>
  )
}