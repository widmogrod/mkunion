import React from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../ui/card'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight, Filter, X } from 'lucide-react'
import { PaginatedTableState } from './PaginatedTable/LegacyAdapter'
import { useTableData } from './PaginatedTable/hooks/useTableData'
import { usePagination } from './PaginatedTable/hooks/usePagination'
import { TableContent } from './PaginatedTable/components/TableContent'
import { StateDetailsRenderer } from '../workflow/StateDetailsRenderer'
import { useWorkflowApi } from '../../hooks/use-workflow-api'
import * as workflow from '../../workflow/github_com_widmogrod_mkunion_x_workflow'
import * as schemaless from '../../workflow/github_com_widmogrod_mkunion_x_storage_schemaless'

interface StatesTableProps {
  refreshTrigger: number
  loadStates: (state: PaginatedTableState<workflow.State>) => Promise<any>
}

export function StatesTable({ refreshTrigger, loadStates }: StatesTableProps) {
  const pagination = usePagination({ initialPageSize: 10 })
  const { deleteStates, tryRecover } = useWorkflowApi()
  const [selected, setSelected] = React.useState<{ [key: string]: boolean }>({})
  const [filters, setFilters] = React.useState({
    stateType: 'all',
    searchText: '',
    showFilters: false
  })
  
  // Build filter conditions based on current filters
  const buildWhereClause = React.useCallback(() => {
    const conditions: any = {}
    
    // Filter by state type
    if (filters.stateType !== 'all') {
      conditions['Data.$type'] = filters.stateType
    }
    
    // Text search across multiple fields
    if (filters.searchText.trim()) {
      const searchTerm = filters.searchText.trim()
      // For now, search in ID field - could be expanded to other fields
      conditions['ID'] = { $regex: searchTerm, $options: 'i' }
    }
    
    return conditions
  }, [filters])

  // Adapt load function to work with the new hooks
  const adaptedLoad = React.useCallback(async (state: any) => {
    const whereClause = buildWhereClause()
    
    const legacyState: PaginatedTableState<workflow.State> = {
      limit: state.limit,
      offset: state.offset,
      selected,
      sort: { ID: true },
      where: Object.keys(whereClause).length > 0 ? whereClause : state.where
    }

    const result = await loadStates(legacyState)
    
    return {
      items: result.Items || [],
      next: result.Next ? 'has-next' : undefined,
      total: undefined
    }
  }, [loadStates, selected, buildWhereClause])

  const { data, loading, error, refresh } = useTableData(adaptedLoad, pagination.state)

  // Filter helper functions
  const clearFilters = () => {
    setFilters({
      stateType: 'all',
      searchText: '',
      showFilters: false
    })
    pagination.actions.goToFirstPage()
  }

  const hasActiveFilters = filters.stateType !== 'all' || filters.searchText.trim()

  const handleFilterChange = (key: string, value: string) => {
    setFilters(prev => ({ ...prev, [key]: value }))
    // Reset to first page when filters change
    pagination.actions.goToFirstPage()
  }

  const handleDeleteStates = async () => {
    const selectedIDs = Object.keys(selected).filter(k => selected[k])
    
    if (selectedIDs.length === 0) {
      alert('No states selected for deletion')
      return
    }

    // Get the full record objects for selected states
    const statesToDelete = data.items.filter((item: schemaless.Record<workflow.State>) => 
      item.ID && selectedIDs.includes(item.ID)
    )

    const confirmMessage = `Are you sure you want to delete ${statesToDelete.length} state(s)? This action cannot be undone.`
    if (!window.confirm(confirmMessage)) {
      return
    }

    try {
      await deleteStates(statesToDelete)
      setSelected({}) // Clear selection
      refresh() // Refresh the table data
      alert(`Successfully deleted ${statesToDelete.length} state(s)`)
    } catch (error) {
      console.error('Failed to delete states:', error)
      alert(`Failed to delete states: ${error instanceof Error ? error.message : 'Unknown error'}`)
    }
  }

  const handleRecoverStates = async () => {
    const selectedIDs = Object.keys(selected).filter(k => selected[k])
    
    if (selectedIDs.length === 0) {
      alert('No states selected for recovery')
      return
    }

    // Find the actual states and their RunIDs
    const statesToRecover: string[] = []
    selectedIDs.forEach(recordID => {
      const stateRecord = data.items.find((item: schemaless.Record<workflow.State>) => item.ID === recordID)
      if (stateRecord?.Data) {
        const state = stateRecord.Data
        // Get RunID from the state's BaseState
        const runID = (state[state.$type as keyof typeof state] as any)?.BaseState?.RunID
        if (runID) {
          statesToRecover.push(runID)
        }
      }
    })

    if (statesToRecover.length === 0) {
      alert('No recoverable states found. States must have a RunID to be recovered.')
      return
    }

    const confirmMessage = `Are you sure you want to attempt recovery for ${statesToRecover.length} state(s)?`
    if (!window.confirm(confirmMessage)) {
      return
    }

    try {
      // Use tryRecover for each state with its actual RunID
      const recoveryPromises = statesToRecover.map(runID => tryRecover(runID))
      await Promise.all(recoveryPromises)
      
      setSelected({}) // Clear selection
      refresh() // Refresh the table data
      alert(`Successfully initiated recovery for ${statesToRecover.length} state(s)`)
    } catch (error) {
      console.error('Failed to recover states:', error)
      alert(`Failed to recover states: ${error instanceof Error ? error.message : 'Unknown error'}`)
    }
  }

  // Table columns configuration
  const columns = React.useMemo(() => [
    {
      key: 'selection',
      header: (
        <input
          type="checkbox"
          className="rounded border border-input"
          checked={Object.keys(selected).length > 0 && Object.values(selected).every(v => v)}
          onChange={(e) => {
            const newSelected: { [key: string]: boolean } = {}
            if (e.target.checked) {
              data.items.forEach((item: any) => {
                if (item.ID) newSelected[item.ID] = true
              })
            }
            setSelected(newSelected)
          }}
        />
      ),
      render: (value: any, item: schemaless.Record<workflow.State>) => {
        const id = item.ID || ''
        return (
          <input
            type="checkbox"
            className="rounded border border-input"
            checked={selected[id] || false}
            onChange={(e) => {
              if (id) {
                setSelected(prev => ({
                  ...prev,
                  [id]: e.target.checked
                }))
              }
            }}
          />
        )
      }
    },
    {
      key: 'content',
      header: 'Data',
      render: (value: any, item: schemaless.Record<workflow.State>) => (
        <StateDetailsRenderer data={item} />
      )
    }
  ], [selected, data.items])

  return (
    <Card className="w-full h-full flex flex-col overflow-hidden">
      <CardHeader className="flex-shrink-0 border-b">
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>States</CardTitle>
            <CardDescription>View workflow execution states</CardDescription>
          </div>
          <div className="flex items-center gap-2">
            {hasActiveFilters && (
              <Button
                variant="ghost"
                size="sm"
                onClick={clearFilters}
                className="h-7 text-xs"
              >
                <X className="h-3 w-3 mr-1" />
                Clear
              </Button>
            )}
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setFilters(prev => ({ ...prev, showFilters: !prev.showFilters }))}
              className="h-7 text-xs"
            >
              <Filter className="h-3 w-3 mr-1" />
              Filter
            </Button>
          </div>
        </div>
        
        {/* Filter Controls */}
        {filters.showFilters && (
          <div className="flex flex-wrap gap-3 mt-3 pt-3 border-t">
            <div className="flex items-center gap-2">
              <label className="text-xs font-medium text-muted-foreground">State Type:</label>
              <select
                value={filters.stateType}
                onChange={(e) => handleFilterChange('stateType', e.target.value)}
                className="h-7 px-2 rounded border border-input bg-background text-xs text-foreground w-40"
              >
                <option value="all">All Types</option>
                <option value="workflow.Done">Done</option>
                <option value="workflow.Error">Error</option>
                <option value="workflow.Await">Await</option>
                <option value="workflow.Scheduled">Scheduled</option>
                <option value="workflow.ScheduleStopped">Schedule Stopped</option>
                <option value="workflow.Apply">Apply</option>
                <option value="workflow.Choose">Choose</option>
                <option value="workflow.Fork">Fork</option>
              </select>
            </div>
            
            <div className="flex items-center gap-2">
              <label className="text-xs font-medium text-muted-foreground">Search:</label>
              <Input
                placeholder="Search by ID..."
                value={filters.searchText}
                onChange={(e) => handleFilterChange('searchText', e.target.value)}
                className="h-7 w-48"
              />
            </div>
          </div>
        )}
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
                <p className="text-muted-foreground">No states found</p>
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
                disabled={Object.keys(selected).filter(k => selected[k]).length === 0}
                onClick={handleDeleteStates}
              >
                Delete
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={Object.keys(selected).filter(k => selected[k]).length === 0}
                onClick={handleRecoverStates}
              >
                Try recover
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