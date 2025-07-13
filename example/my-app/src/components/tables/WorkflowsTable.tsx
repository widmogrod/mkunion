import React from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../ui/card'
import { Button } from '../ui/button'
import { AppleCheckbox } from '../ui/AppleCheckbox'
import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight } from 'lucide-react'
import { useTableData } from './PaginatedTable/hooks/useTableData'
import { usePagination } from './PaginatedTable/hooks/usePagination'
import { TableContent } from './PaginatedTable/components/TableContent'
import { WorkflowDisplay } from '../workflow/WorkflowDisplay'
import { useWorkflowApi } from '../../hooks/use-workflow-api'
import { useToast } from '../../contexts/ToastContext'
import { TableLoadState } from './TablesSection'
import { TableControls } from './PaginatedTable/components/TableControls'
import { StatusIndicator } from '../ui/StatusIndicator'
import * as workflow from '../../workflow/github_com_widmogrod_mkunion_x_workflow'
import * as schemaless from '../../workflow/github_com_widmogrod_mkunion_x_storage_schemaless'

interface WorkflowsTableProps {
  refreshTrigger: number
  loadFlows: (state: TableLoadState) => Promise<any>
}

export function WorkflowsTable({ refreshTrigger, loadFlows }: WorkflowsTableProps) {
  const pagination = usePagination({ initialPageSize: 10 })
  const { deleteFlows } = useWorkflowApi()
  const toast = useToast()
  const [selected, setSelected] = React.useState<{ [key: string]: boolean }>({})
  const [isDeleting, setIsDeleting] = React.useState(false)
  const [deleteStatus, setDeleteStatus] = React.useState<'idle' | 'success' | 'error'>('idle')
  
  // Adapt load function to work with the new hooks
  const adaptedLoad = React.useCallback(async (state: any) => {
    const tableState: TableLoadState = {
      limit: state.limit,
      offset: state.offset,
      sort: { ID: true },
      where: state.where
    }

    const result = await loadFlows(tableState)
    
    return {
      items: result.Items || [],
      next: result.Next ? 'has-next' : undefined,
      total: undefined
    }
  }, [loadFlows])

  const { data, loading, error, refresh } = useTableData(adaptedLoad, pagination.state)

  const handleDeleteFlows = async () => {
    const selectedIDs = Object.keys(selected).filter(k => selected[k])
    
    if (selectedIDs.length === 0) {
      toast.warning('No Selection', 'Please select workflows to delete')
      return
    }

    // Get the full record objects for selected flows
    const flowsToDelete = data.items.filter((item: schemaless.Record<workflow.Flow>) => 
      item.ID && selectedIDs.includes(item.ID)
    )

    // Use toast for confirmation instead of browser alert
    toast.warning(
      'Confirm Deletion',
      `Are you sure you want to delete ${flowsToDelete.length} workflow(s)? This action cannot be undone.`,
      {
        persistent: true,
        action: {
          label: 'Delete',
          onClick: async () => {
            setIsDeleting(true)
            setDeleteStatus('idle')
            try {
              await deleteFlows(flowsToDelete)
              setSelected({}) // Clear selection
              refresh() // Refresh the table data
              toast.success('Deletion Complete', `Successfully deleted ${flowsToDelete.length} workflow(s)`)
              setDeleteStatus('success')
              // Clear status after 2 seconds
              setTimeout(() => setDeleteStatus('idle'), 2000)
            } catch (error) {
              console.error('Failed to delete workflows:', error)
              toast.error('Deletion Failed', `Failed to delete workflows: ${error instanceof Error ? error.message : 'Unknown error'}`)
              setDeleteStatus('error')
              // Clear status after 3 seconds for errors
              setTimeout(() => setDeleteStatus('idle'), 3000)
            } finally {
              setIsDeleting(false)
            }
          }
        }
      }
    )
  }

  // Table columns configuration
  const columns = React.useMemo(() => [
    {
      key: 'selection',
      className: 'w-12 px-3 py-3', // Optimized spacing for checkbox column
      header: (
        <div className="flex items-center justify-center">
          <AppleCheckbox
            checked={Object.keys(selected).length > 0 && Object.values(selected).every(v => v)}
            onChange={(checked) => {
              const newSelected: { [key: string]: boolean } = {}
              if (checked) {
                data.items.forEach((item: any) => {
                  if (item.ID) newSelected[item.ID] = true
                })
              }
              setSelected(newSelected)
            }}
          />
        </div>
      ),
      render: (value: any, item: schemaless.Record<workflow.Flow>) => {
        const id = item.ID || ''
        return (
          <div className="flex items-center justify-center">
            <AppleCheckbox
              checked={selected[id] || false}
              onChange={(checked) => {
                if (id) {
                  setSelected(prev => ({
                    ...prev,
                    [id]: checked
                  }))
                }
              }}
            />
          </div>
        )
      }
    },
    {
      key: 'content',
      header: 'Data',
      render: (value: any, item: schemaless.Record<workflow.Flow>) => (
        <WorkflowDisplay data={item} />
      )
    }
  ], [selected, data.items])

  return (
    <Card className="w-full h-full flex flex-col overflow-hidden">
      <CardHeader className="flex-shrink-0 border-b py-4">
        <div className="flex items-center justify-between gap-4">
          <div className="flex-shrink-0">
            <CardTitle>Workflows</CardTitle>
            <CardDescription>Manage your workflow definitions</CardDescription>
          </div>
          
          {/* Table Controls */}
          <TableControls
            onRefresh={refresh}
            isLoading={loading}
            refreshTitle="Refresh workflows data"
            showSearch={false}
            showFilters={false}
          />
        </div>
      </CardHeader>
      <CardContent className="p-0 flex-1 flex flex-col overflow-hidden">
        {/* Scrollable table content */}
        <div className="flex-1 overflow-auto">
          <div className="p-6 pb-2">
            {loading ? (
              <div className="flex items-center justify-center py-8">
                <p className="text-muted-foreground">Loading...</p>
              </div>
            ) : error ? (
              <div className="flex items-center justify-center py-8">
                <p className="text-red-500">Error loading data</p>
              </div>
            ) : data.items.length === 0 ? (
              <div className="flex items-center justify-center py-8">
                <p className="text-muted-foreground">No workflows found</p>
              </div>
            ) : (
              <TableContent
                columns={columns}
                data={data.items}
                renderItem={(item, column) => {
                  if (column.render) {
                    return column.render(item, item)
                  }
                  return <pre className="text-xs">{JSON.stringify(item, null, 2)}</pre>
                }}
              />
            )}
          </div>
        </div>
        
        {/* Fixed action buttons and pagination at bottom */}
        <div className="flex-shrink-0 border-t bg-background px-4 py-3">
          <div className="flex items-center justify-between">
            {/* Left: Action buttons */}
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={Object.keys(selected).filter(k => selected[k]).length === 0 || isDeleting}
                onClick={handleDeleteFlows}
                className="flex items-center"
              >
                {isDeleting ? 'Deleting...' : 'Delete'}
                <StatusIndicator status={deleteStatus} />
              </Button>
            </div>
            
            {/* Right: Pagination */}
            {(data.items.length > 0 || pagination.state.offset > 0) && (
              <div className="flex items-center space-x-4">
                <span className="text-xs text-muted-foreground">
                  Page {pagination.currentPage}
                </span>
                
                <div className="flex items-center space-x-1">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={pagination.actions.goToFirstPage}
                    disabled={pagination.state.offset === 0}
                    className="h-7 w-7 p-0"
                  >
                    <ChevronsLeft className="h-3 w-3" />
                  </Button>
                  
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={pagination.actions.goToPreviousPage}
                    disabled={pagination.state.offset === 0}
                    className="h-7 w-7 p-0"
                  >
                    <ChevronLeft className="h-3 w-3" />
                  </Button>
                  
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={pagination.actions.goToNextPage}
                    disabled={!data.next}
                    className="h-7 w-7 p-0"
                  >
                    <ChevronRight className="h-3 w-3" />
                  </Button>
                  
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => pagination.actions.goToLastPage(data.total || 100)}
                    disabled={!data.next}
                    className="h-7 w-7 p-0"
                  >
                    <ChevronsRight className="h-3 w-3" />
                  </Button>
                </div>
              </div>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}