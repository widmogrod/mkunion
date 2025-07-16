import React from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '../ui/card'
import { Button } from '../ui/button'
import { ConfirmButton } from '../ui/ConfirmButton'
import { AppleCheckbox } from '../ui/AppleCheckbox'
import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight } from 'lucide-react'
import { useTableData } from './PaginatedTable/hooks/useTableData'
import { usePagination } from './PaginatedTable/hooks/usePagination'
import { TableContent } from './PaginatedTable/components/TableContent'
import { StateDetailsRenderer } from '../workflow/StateDetailsRenderer'
import { useWorkflowApi } from '../../hooks/use-workflow-api'
import { useToast } from '../../contexts/ToastContext'
import { TableLoadState } from './TablesSection'
import { TableControls } from './PaginatedTable/components/TableControls'
import { StatusIndicator } from '../ui/StatusIndicator'
import { ClickableStateRow } from '../navigation/ClickableStateRow'
import { useFilterStore, FilterItem } from '../../stores/filter-store'
import * as workflow from '../../workflow/github_com_widmogrod_mkunion_x_workflow'
import * as schemaless from '../../workflow/github_com_widmogrod_mkunion_x_storage_schemaless'
import * as predicate from '../../workflow/github_com_widmogrod_mkunion_x_storage_predicate'
import * as schema from '../../workflow/github_com_widmogrod_mkunion_x_schema'

interface ExecutionsTableProps {
  refreshTrigger: number
  loadStates: (state: TableLoadState) => Promise<any>
  runIdFilter?: string | null
  scheduleFilter?: string | null
}

// Helper functions for creating predicates
const createCompare = (location: string, operation: string, value: schema.Schema): predicate.Predicate => ({
  "$type": "predicate.Compare",
  "predicate.Compare": {
    Location: location,
    Operation: operation,
    BindValue: {
      "$type": "predicate.Literal",
      "predicate.Literal": {
        Value: value
      }
    }
  }
})

const createOr = (predicates: predicate.Predicate[]): predicate.Predicate => ({
  "$type": "predicate.Or",
  "predicate.Or": {
    L: predicates
  }
})

const createAnd = (predicates: predicate.Predicate[]): predicate.Predicate => ({
  "$type": "predicate.And",
  "predicate.And": {
    L: predicates
  }
})

const createNot = (p: predicate.Predicate): predicate.Predicate => ({
  "$type": "predicate.Not",
  "predicate.Not": {
    P: p
  }
})

// Map state types to their display properties
const STATE_TYPE_CONFIG: Record<string, { label: string; color: string }> = {
  'workflow.Done': { label: 'Done', color: '#10b981' },
  'workflow.Error': { label: 'Error', color: '#ef4444' },
  'workflow.Await': { label: 'Await', color: '#3b82f6' },
  'workflow.Scheduled': { label: 'Scheduled', color: '#eab308' },
  'workflow.ScheduleStopped': { label: 'Paused', color: '#6b7280' },
  'workflow.NextOperation': { label: 'Next', color: '#a855f7' },
}

// Helper to build initial where clause
function buildInitialWhereClause(
  executionFilters: FilterItem[],
  searchText: string,
  runIdFilter?: string | null,
  scheduleFilter?: string | null
): predicate.Predicate | undefined {
  const predicates: predicate.Predicate[] = []
  
  // Add workflow filters
  const workflowFilters = executionFilters.filter(f => f.stateType === 'workflow')
  if (workflowFilters.length > 0) {
    const stateTypes = ['workflow.Done', 'workflow.Error', 'workflow.Await', 'workflow.Scheduled', 'workflow.ScheduleStopped', 'workflow.NextOperation']
    
    workflowFilters.forEach(filter => {
      const stateTypePredicates = stateTypes.map(stateType => 
        createOr([
          createCompare(
            `Data["${stateType}"]["BaseState"]["Flow"]["workflow.Flow"]["Name"]`,
            '==',
            { "$type": "schema.String", "schema.String": filter.label }
          ),
          createCompare(
            `Data["${stateType}"]["BaseState"]["Flow"]["workflow.FlowRef"]["FlowID"]`,
            '==',
            { "$type": "schema.String", "schema.String": filter.label }
          )
        ])
      )
      
      predicates.push(createOr(stateTypePredicates))
    })
  }
  
  if (predicates.length === 0) return undefined
  if (predicates.length === 1) return predicates[0]
  return createAnd(predicates)
}

export function ExecutionsTableRefactored({ 
  refreshTrigger, 
  loadStates,
  runIdFilter,
  scheduleFilter
}: ExecutionsTableProps) {
  const { deleteStates, tryRecover } = useWorkflowApi()
  const toast = useToast()
  const [selected, setSelected] = React.useState<{ [key: string]: boolean }>({})
  const [searchText, setSearchText] = React.useState('')
  const [isDeleting, setIsDeleting] = React.useState(false)
  const [isRecovering, setIsRecovering] = React.useState(false)
  const [deleteStatus, setDeleteStatus] = React.useState<'idle' | 'success' | 'error'>('idle')
  const [recoverStatus, setRecoverStatus] = React.useState<'idle' | 'success' | 'error'>('idle')
  const [flowNameCache, setFlowNameCache] = React.useState<Map<string, string>>(new Map())
  
  // Use filter store instead of local state
  const { 
    executionFilters,
    addExecutionFilter,
    removeExecutionFilter,
    toggleExecutionFilterMode,
    clearExecutionFilters 
  } = useFilterStore()
  
  // Initialize pagination with initial where clause
  const initialWhere = React.useMemo(
    () => buildInitialWhereClause(executionFilters, searchText, runIdFilter, scheduleFilter),
    [] // Only compute on mount
  )
  
  const pagination = usePagination({ 
    initialPageSize: 10,
    initialWhere: initialWhere 
  })
  
  // Extract setWhere to avoid dependency issues
  const { setWhere } = pagination.actions
  
  // Build filter conditions based on current filters
  const buildWhereClause = React.useCallback((): predicate.Predicate | undefined => {
    console.log('buildWhereClause called with filters:', {
      executionFilters,
      searchText,
      runIdFilter,
      scheduleFilter
    })
    const predicates: predicate.Predicate[] = []
    
    // Add runId filter if present
    if (runIdFilter) {
      predicates.push(
        createCompare(
          'ID',
          '==',
          { "$type": "schema.String", "schema.String": runIdFilter }
        )
      )
    }
    
    // Add schedule filter if present
    if (scheduleFilter) {
      predicates.push(
        createOr([
          createCompare(
            'Data["workflow.Scheduled"]["BaseState"]["RunOption"]["workflow.ScheduleRun"]["ParentRunID"]',
            '==',
            { "$type": "schema.String", "schema.String": scheduleFilter }
          ),
          createCompare(
            'Data["workflow.ScheduleStopped"]["BaseState"]["RunOption"]["workflow.ScheduleRun"]["ParentRunID"]',
            '==',
            { "$type": "schema.String", "schema.String": scheduleFilter }
          )
        ])
      )
    }
    
    // Group filters by include/exclude and filter type
    const includeFilters = executionFilters.filter(f => !f.isExclude)
    const excludeFilters = executionFilters.filter(f => f.isExclude)
    
    // Separate workflow filters from state type filters
    const includeStateFilters = includeFilters.filter(f => f.stateType !== 'workflow')
    const excludeStateFilters = excludeFilters.filter(f => f.stateType !== 'workflow')
    
    // Handle include state filters (OR logic)
    if (includeStateFilters.length > 0) {
      const includePredicates = includeStateFilters.map(filter => 
        createCompare(
          'Data["$type"]',
          '==',
          { "$type": "schema.String", "schema.String": filter.stateType }
        )
      )
      
      predicates.push(
        includePredicates.length === 1 
          ? includePredicates[0] 
          : createOr(includePredicates)
      )
    }
    
    // Handle exclude state filters (NOT logic for each)
    excludeStateFilters.forEach(filter => {
      predicates.push(
        createNot(
          createCompare(
            'Data["$type"]',
            '==',
            { "$type": "schema.String", "schema.String": filter.stateType }
          )
        )
      )
    })
    
    // Handle workflow filters
    const workflowFilters = executionFilters.filter(f => f.stateType === 'workflow')
    workflowFilters.forEach(filter => {
      const stateTypes = ['workflow.Done', 'workflow.Error', 'workflow.Await', 'workflow.Scheduled', 'workflow.ScheduleStopped', 'workflow.NextOperation']
      
      const stateTypePredicates = stateTypes.map(stateType => 
        createOr([
          createCompare(
            `Data["${stateType}"]["BaseState"]["Flow"]["workflow.Flow"]["Name"]`,
            '==',
            { "$type": "schema.String", "schema.String": filter.label }
          ),
          createCompare(
            `Data["${stateType}"]["BaseState"]["Flow"]["workflow.FlowRef"]["FlowID"]`,
            '==',
            { "$type": "schema.String", "schema.String": filter.label }
          )
        ])
      )
      
      const workflowPredicate = createOr(stateTypePredicates)
      
      if (!filter.isExclude) {
        predicates.push(workflowPredicate)
      } else {
        predicates.push(createNot(workflowPredicate))
      }
    })
    
    // Text search - exact ID match
    if (searchText.trim()) {
      predicates.push(
        createCompare(
          'ID',
          '==',
          { "$type": "schema.String", "schema.String": searchText.trim() }
        )
      )
    }
    
    // Combine all predicates with AND
    if (predicates.length === 0) {
      console.log('buildWhereClause: No predicates, returning undefined')
      return undefined
    }
    if (predicates.length === 1) {
      console.log('buildWhereClause: Single predicate:', predicates[0])
      return predicates[0]
    }
    const andPredicate = createAnd(predicates)
    console.log('buildWhereClause: AND predicate:', andPredicate)
    return andPredicate
  }, [executionFilters, searchText, runIdFilter, scheduleFilter])

  // Update pagination where clause when filters change  
  // We need to be careful here to avoid infinite loops
  const whereClause = React.useMemo(() => buildWhereClause(), [buildWhereClause])
  
  React.useEffect(() => {
    console.log('Setting where clause:', whereClause)
    console.log('Current pagination.state.where BEFORE setWhere:', pagination.state.where)
    setWhere(whereClause)
    // Check immediately after
    setTimeout(() => {
      console.log('Current pagination.state.where AFTER setWhere (should be updated):', pagination.state.where)
    }, 100)
  }, [whereClause, setWhere]) // Remove pagination.state.where from deps to avoid loops
  
  // Adapt load function to work with the new hooks
  const adaptedLoad = React.useCallback(async (state: any) => {
    console.log('adaptedLoad called with state:', state)
    const tableState: TableLoadState = {
      limit: state.limit,
      offset: state.offset,
      sort: { ID: true },
      where: state.where
    }
    console.log('Calling loadStates with tableState:', tableState)

    const result = await loadStates(tableState)
    
    return {
      items: result.Items || [],
      next: result.Next ? 'has-next' : undefined,
      total: undefined
    }
  }, [loadStates])

  const { data, loading, error, refresh } = useTableData(adaptedLoad, pagination.state)
  
  // Debug logging
  React.useEffect(() => {
    console.log('ExecutionsTableRefactored render:', {
      executionFilters,
      paginationState: pagination.state,
      paginationWhereClause: pagination.state.where,
      whereClause,
      dataLength: data.items.length,
      whereClauseStringified: JSON.stringify(whereClause),
      paginationWhereStringified: JSON.stringify(pagination.state.where)
    })
  }, [executionFilters, pagination.state, whereClause, data.items.length])

  // Refresh the table when refreshTrigger changes
  React.useEffect(() => {
    if (refreshTrigger > 0) {
      refresh()
    }
  }, [refreshTrigger, refresh])

  // Filter management functions
  const addFilter = React.useCallback((stateType: string, label?: string) => {
    if (stateType === 'workflow') {
      if (!label) return
      
      addExecutionFilter({
        stateType: 'workflow',
        label: label,
        color: '#3b82f6',
        isExclude: false
      })
      return
    }
    
    const config = STATE_TYPE_CONFIG[stateType]
    if (!config) return
    
    addExecutionFilter({
      stateType,
      label: config.label,
      color: config.color,
      isExclude: false
    })
  }, [addExecutionFilter])
  
  const clearAllFilters = React.useCallback(() => {
    clearExecutionFilters()
    setSearchText('')
  }, [clearExecutionFilters])

  const handleDeleteStates = async () => {
    const selectedIDs = Object.keys(selected).filter(k => selected[k])
    
    if (selectedIDs.length === 0) {
      toast.warning('No Selection', 'Please select states to delete')
      return
    }

    const statesToDelete = data.items.filter((item: schemaless.Record<workflow.State>) => 
      item.ID && selectedIDs.includes(item.ID)
    )

    setIsDeleting(true)
    setDeleteStatus('idle')
    try {
      await deleteStates(statesToDelete)
      setSelected({})
      refresh()
      toast.success('Deletion Complete', `Successfully deleted ${statesToDelete.length} state(s)`)
      setDeleteStatus('success')
      setTimeout(() => setDeleteStatus('idle'), 2000)
    } catch (error) {
      console.error('Failed to delete states:', error)
      toast.error('Deletion Failed', `Failed to delete states: ${error instanceof Error ? error.message : 'Unknown error'}`)
      setDeleteStatus('error')
      setTimeout(() => setDeleteStatus('idle'), 3000)
    } finally {
      setIsDeleting(false)
    }
  }

  const handleRecoverStates = async () => {
    const selectedIDs = Object.keys(selected).filter(k => selected[k])
    
    if (selectedIDs.length === 0) {
      toast.warning('No Selection', 'Please select states to recover')
      return
    }

    const statesToRecover: string[] = []
    selectedIDs.forEach(recordID => {
      const stateRecord = data.items.find((item: schemaless.Record<workflow.State>) => item.ID === recordID)
      if (stateRecord?.Data) {
        const state = stateRecord.Data
        const runID = (state[state.$type as keyof typeof state] as any)?.BaseState?.RunID
        if (runID) {
          statesToRecover.push(runID)
        }
      }
    })

    if (statesToRecover.length === 0) {
      toast.warning('No Recoverable States', 'States must have a RunID to be recovered.')
      return
    }

    setIsRecovering(true)
    setRecoverStatus('idle')
    try {
      const recoveryPromises = statesToRecover.map(runID => tryRecover(runID))
      await Promise.all(recoveryPromises)
      
      setSelected({})
      refresh()
      toast.success('Recovery Initiated', `Successfully initiated recovery for ${statesToRecover.length} state(s)`)
      setRecoverStatus('success')
      setTimeout(() => setRecoverStatus('idle'), 2000)
    } catch (error) {
      console.error('Failed to recover states:', error)
      toast.error('Recovery Failed', `Failed to recover states: ${error instanceof Error ? error.message : 'Unknown error'}`)
      setRecoverStatus('error')
      setTimeout(() => setRecoverStatus('idle'), 3000)
    } finally {
      setIsRecovering(false)
    }
  }

  // Check if a state type is actively filtered
  const isStateTypeFiltered = React.useCallback((stateType: string) => {
    return executionFilters.some(f => f.stateType === stateType)
  }, [executionFilters])

  // Utility function to extract flow name from state data
  const extractFlowNameFromState = React.useCallback((item: schemaless.Record<workflow.State>): string | undefined => {
    if (!item.Data || !item.Data.$type) return undefined
    
    const stateData = item.Data[item.Data.$type as keyof typeof item.Data] as any
    const baseState = stateData?.BaseState
    
    if (baseState?.Flow) {
      if (baseState.Flow.$type === 'workflow.Flow' && baseState.Flow['workflow.Flow']) {
        return baseState.Flow['workflow.Flow'].Name || undefined
      } else if (baseState.Flow.$type === 'workflow.FlowRef' && baseState.Flow['workflow.FlowRef']) {
        return baseState.Flow['workflow.FlowRef'].FlowID || undefined
      }
    }
    
    return undefined
  }, [])

  // Table columns configuration
  const columns = React.useMemo(() => [
    {
      key: 'selection',
      className: 'w-12 px-3 py-3',
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
      render: (value: any, item: schemaless.Record<workflow.State>) => {
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
      render: (value: any, item: schemaless.Record<workflow.State>) => {
        const flowName = extractFlowNameFromState(item) || 
                          flowNameCache.get(item.ID || '') || 
                          undefined
        
        return (
          <div className="space-y-2">
            <StateDetailsRenderer 
              data={item} 
              onAddFilter={addFilter}
              isFilterActive={item.Data ? isStateTypeFiltered(item.Data.$type || '') : false}
              flowName={flowName}
            />
            <ClickableStateRow 
              state={item}
              flowName={flowName}
              className="mt-2"
              showWorkflowIcon={false}
            />
          </div>
        )
      }
    }
  ], [selected, data.items, addFilter, isStateTypeFiltered, flowNameCache, extractFlowNameFromState])

  return (
    <Card className="w-full h-full flex flex-col overflow-hidden">
      <CardHeader className="flex-shrink-0 border-b py-3">
        <div className="flex items-center justify-between gap-4">
          <CardTitle className="text-base">Workflow Executions</CardTitle>
          
          <TableControls
            searchText={searchText}
            onSearchChange={setSearchText}
            searchPlaceholder="Search by exact ID"
            onRefresh={refresh}
            isLoading={loading}
            refreshTitle="Refresh states data"
            activeFilters={executionFilters}
            onRemoveFilter={removeExecutionFilter}
            onToggleFilterMode={toggleExecutionFilterMode}
            onClearAllFilters={clearAllFilters}
          />
        </div>
      </CardHeader>
      <CardContent className="p-0 flex-1 flex flex-col overflow-hidden">
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
        
        <div className="flex-shrink-0 border-t bg-background px-4 py-3">
          <div className="flex items-center justify-between">
            <div className="flex gap-2">
              <ConfirmButton
                variant="outline"
                size="sm"
                disabled={Object.keys(selected).filter(k => selected[k]).length === 0 || isDeleting || isRecovering}
                onConfirm={handleDeleteStates}
                confirmText={`Delete ${Object.keys(selected).filter(k => selected[k]).length} state(s)`}
                className="flex items-center"
              >
                {isDeleting ? 'Deleting...' : `Delete${Object.keys(selected).filter(k => selected[k]).length > 0 ? ` (${Object.keys(selected).filter(k => selected[k]).length})` : ''}`}
                <StatusIndicator status={deleteStatus} />
              </ConfirmButton>
              <ConfirmButton
                variant="outline"
                size="sm"
                disabled={Object.keys(selected).filter(k => selected[k]).length === 0 || isDeleting || isRecovering}
                onConfirm={handleRecoverStates}
                confirmText={`Recover ${Object.keys(selected).filter(k => selected[k]).length} state(s)`}
                className="flex items-center"
              >
                {isRecovering ? 'Recovering...' : `Try recover${Object.keys(selected).filter(k => selected[k]).length > 0 ? ` (${Object.keys(selected).filter(k => selected[k]).length})` : ''}`}
                <StatusIndicator status={recoverStatus} />
              </ConfirmButton>
            </div>
            
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