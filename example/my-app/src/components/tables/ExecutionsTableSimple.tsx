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
import { useUrlFilters } from '../../hooks/useUrlFilters'
import * as workflow from '../../workflow/github_com_widmogrod_mkunion_x_workflow'
import * as schemaless from '../../workflow/github_com_widmogrod_mkunion_x_storage_schemaless'
import * as predicate from '../../workflow/github_com_widmogrod_mkunion_x_storage_predicate'
import * as schema from '../../workflow/github_com_widmogrod_mkunion_x_schema'

interface ExecutionsTableProps {
  refreshTrigger: number
  loadStates: (state: TableLoadState) => Promise<any>
}

// Map state types to their display properties
const STATE_TYPE_CONFIG: Record<string, { label: string; color: string; urlParam: string }> = {
  'workflow.Done': { label: 'Done', color: '#10b981', urlParam: 'done' },
  'workflow.Error': { label: 'Error', color: '#ef4444', urlParam: 'error' },
  'workflow.Await': { label: 'Await', color: '#3b82f6', urlParam: 'await' },
  'workflow.Scheduled': { label: 'Scheduled', color: '#eab308', urlParam: 'scheduled' },
  'workflow.ScheduleStopped': { label: 'Paused', color: '#6b7280', urlParam: 'paused' },
  'workflow.NextOperation': { label: 'Next', color: '#a855f7', urlParam: 'next' },
}

// Reverse mapping for URL params to state types
const URL_TO_STATE_TYPE: Record<string, string> = Object.entries(STATE_TYPE_CONFIG).reduce(
  (acc, [stateType, config]) => ({ ...acc, [config.urlParam]: stateType }),
  {}
)

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

export function ExecutionsTableSimple({ refreshTrigger, loadStates }: ExecutionsTableProps) {
  const { deleteStates, tryRecover } = useWorkflowApi()
  const toast = useToast()
  const [selected, setSelected] = React.useState<{ [key: string]: boolean }>({})
  const [searchText, setSearchText] = React.useState('')
  const [isDeleting, setIsDeleting] = React.useState(false)
  const [isRecovering, setIsRecovering] = React.useState(false)
  const [deleteStatus, setDeleteStatus] = React.useState<'idle' | 'success' | 'error'>('idle')
  const [recoverStatus, setRecoverStatus] = React.useState<'idle' | 'success' | 'error'>('idle')
  
  // Use URL filters directly - no store, no sync needed!
  const {
    filters,
    setFilter,
    addStatusFilter,
    removeStatusFilter,
    toggleStatusFilterMode,
    toggleWorkflowFilterMode,
    clearAllFilters,
    hasFilters
  } = useUrlFilters()
  
  // Build filter conditions based on URL filters
  const buildWhereClause = React.useCallback((): predicate.Predicate | undefined => {
    const predicates: predicate.Predicate[] = []
    
    // Add workflow filter
    if (filters.workflow) {
      const isExcluded = filters.workflow.startsWith('!')
      const workflowName = filters.workflow.replace(/^!/, '')
      
      const stateTypes = Object.keys(STATE_TYPE_CONFIG)
      const stateTypePredicates = stateTypes.map(stateType => 
        createOr([
          createCompare(
            `Data["${stateType}"]["BaseState"]["Flow"]["workflow.Flow"]["Name"]`,
            '==',
            { "$type": "schema.String", "schema.String": workflowName }
          ),
          createCompare(
            `Data["${stateType}"]["BaseState"]["Flow"]["workflow.FlowRef"]["FlowID"]`,
            '==',
            { "$type": "schema.String", "schema.String": workflowName }
          )
        ])
      )
      
      const workflowPredicate = createOr(stateTypePredicates)
      predicates.push(isExcluded ? createNot(workflowPredicate) : workflowPredicate)
    }
    
    // Add status filters
    const includeStatuses = filters.status.filter(s => !s.startsWith('!'))
    const excludeStatuses = filters.status.filter(s => s.startsWith('!')).map(s => s.substring(1))
    
    if (includeStatuses.length > 0) {
      const includePredicates = includeStatuses.map(status => {
        const stateType = URL_TO_STATE_TYPE[status]
        if (!stateType) return null
        return createCompare(
          'Data["$type"]',
          '==',
          { "$type": "schema.String", "schema.String": stateType }
        )
      }).filter(Boolean) as predicate.Predicate[]
      
      if (includePredicates.length > 0) {
        predicates.push(includePredicates.length === 1 ? includePredicates[0] : createOr(includePredicates))
      }
    }
    
    // Add exclude filters
    excludeStatuses.forEach(status => {
      const stateType = URL_TO_STATE_TYPE[status]
      if (stateType) {
        predicates.push(createNot(
          createCompare(
            'Data["$type"]',
            '==',
            { "$type": "schema.String", "schema.String": stateType }
          )
        ))
      }
    })
    
    // Add search filter
    if (searchText.trim()) {
      predicates.push(
        createCompare(
          'ID',
          '==',
          { "$type": "schema.String", "schema.String": searchText.trim() }
        )
      )
    }
    
    // Add runId filter
    if (filters.runId) {
      predicates.push(
        createCompare(
          'ID',
          '==',
          { "$type": "schema.String", "schema.String": filters.runId }
        )
      )
    }
    
    // Add schedule filter
    if (filters.schedule) {
      predicates.push(
        createOr([
          createCompare(
            'Data["workflow.Scheduled"]["BaseState"]["RunOption"]["workflow.ScheduleRun"]["ParentRunID"]',
            '==',
            { "$type": "schema.String", "schema.String": filters.schedule }
          ),
          createCompare(
            'Data["workflow.ScheduleStopped"]["BaseState"]["RunOption"]["workflow.ScheduleRun"]["ParentRunID"]',
            '==',
            { "$type": "schema.String", "schema.String": filters.schedule }
          )
        ])
      )
    }
    
    if (predicates.length === 0) return undefined
    if (predicates.length === 1) return predicates[0]
    return createAnd(predicates)
  }, [filters, searchText])
  
  // Initialize pagination with where clause
  const whereClause = buildWhereClause()
  const pagination = usePagination({ 
    initialPageSize: 10,
    initialWhere: whereClause
  })
  
  // Update pagination when where clause changes
  React.useEffect(() => {
    pagination.actions.setWhere(whereClause)
  }, [whereClause]) // eslint-disable-line react-hooks/exhaustive-deps
  
  // Adapt load function
  const adaptedLoad = React.useCallback(async (state: any) => {
    const tableState: TableLoadState = {
      limit: state.limit,
      offset: state.offset,
      sort: { ID: true },
      where: state.where
    }
    
    const result = await loadStates(tableState)
    return {
      items: result.Items || [],
      next: result.Next ? 'has-next' : undefined,
      total: undefined
    }
  }, [loadStates])
  
  const { data, loading, error, refresh } = useTableData(adaptedLoad, pagination.state)
  
  // Refresh when trigger changes
  React.useEffect(() => {
    if (refreshTrigger > 0) {
      refresh()
    }
  }, [refreshTrigger, refresh])
  
  // Filter management
  const handleAddFilter = React.useCallback((stateType: string, label?: string) => {
    if (stateType === 'workflow' && label) {
      // If we already have this workflow filter in exclude mode, don't override it
      if (filters.workflow === `!${label}`) return
      setFilter('workflow', label)
    } else {
      const config = STATE_TYPE_CONFIG[stateType]
      if (config) {
        addStatusFilter(config.urlParam)
      }
    }
  }, [filters.workflow, setFilter, addStatusFilter])
  
  // Convert URL filters to display format for TableControls
  const displayFilters = React.useMemo(() => {
    const result: Array<{
      stateType: string
      label: string
      color: string
      isExclude: boolean
    }> = []
    
    // Add workflow filter
    if (filters.workflow) {
      const isExcluded = filters.workflow.startsWith('!')
      const workflowName = filters.workflow.replace(/^!/, '')
      
      result.push({
        stateType: 'workflow',
        label: workflowName,
        color: '#3b82f6',
        isExclude: isExcluded
      })
    }
    
    // Add status filters
    filters.status.forEach(status => {
      const isExclude = status.startsWith('!')
      const cleanStatus = status.replace(/^!/, '')
      const stateType = URL_TO_STATE_TYPE[cleanStatus]
      const config = stateType ? STATE_TYPE_CONFIG[stateType] : null
      
      if (config) {
        result.push({
          stateType,
          label: config.label,
          color: config.color,
          isExclude
        })
      }
    })
    
    return result
  }, [filters])
  
  const handleRemoveFilter = React.useCallback((index: number) => {
    const filter = displayFilters[index]
    if (!filter) return
    
    if (filter.stateType === 'workflow') {
      setFilter('workflow', null)
    } else {
      const config = STATE_TYPE_CONFIG[filter.stateType]
      if (config) {
        // Need to remove the actual value from URL, including the ! prefix if excluded
        const urlParam = filter.isExclude ? `!${config.urlParam}` : config.urlParam
        removeStatusFilter(urlParam)
      }
    }
  }, [displayFilters, setFilter, removeStatusFilter])
  
  const handleToggleFilterMode = React.useCallback((index: number) => {
    const filter = displayFilters[index]
    if (!filter) return
    
    if (filter.stateType === 'workflow') {
      toggleWorkflowFilterMode()
    } else {
      const config = STATE_TYPE_CONFIG[filter.stateType]
      if (config) {
        toggleStatusFilterMode(config.urlParam)
      }
    }
  }, [displayFilters, toggleStatusFilterMode, toggleWorkflowFilterMode])
  
  const handleClearAllFilters = React.useCallback(() => {
    clearAllFilters()
    setSearchText('')
  }, [clearAllFilters])
  
  // Delete and recover handlers (same as before)
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
  
  // Check if a state type is filtered
  const isStateTypeFiltered = React.useCallback((stateType: string) => {
    const config = STATE_TYPE_CONFIG[stateType]
    if (!config) return false
    return filters.status.includes(config.urlParam) || filters.status.includes(`!${config.urlParam}`)
  }, [filters.status])
  
  // Extract flow name from state
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
  
  // Table columns
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
        const flowName = extractFlowNameFromState(item)
        
        return (
          <div className="space-y-2">
            <StateDetailsRenderer 
              data={item} 
              onAddFilter={handleAddFilter}
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
  ], [selected, data.items, handleAddFilter, isStateTypeFiltered, extractFlowNameFromState])
  
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
            activeFilters={displayFilters}
            onRemoveFilter={handleRemoveFilter}
            onToggleFilterMode={handleToggleFilterMode}
            onClearAllFilters={handleClearAllFilters}
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