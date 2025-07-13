import React from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../ui/card'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight, Search } from 'lucide-react'
import { useTableData } from './PaginatedTable/hooks/useTableData'
import { usePagination } from './PaginatedTable/hooks/usePagination'
import { TableContent } from './PaginatedTable/components/TableContent'
import { StateDetailsRenderer } from '../workflow/StateDetailsRenderer'
import { useWorkflowApi } from '../../hooks/use-workflow-api'
import { TableLoadState } from './TablesSection'
import { FilterPill } from './FilterPill'
import * as workflow from '../../workflow/github_com_widmogrod_mkunion_x_workflow'
import * as schemaless from '../../workflow/github_com_widmogrod_mkunion_x_storage_schemaless'
import * as predicate from '../../workflow/github_com_widmogrod_mkunion_x_storage_predicate'
import * as schema from '../../workflow/github_com_widmogrod_mkunion_x_schema'

interface StatesTableProps {
  refreshTrigger: number
  loadStates: (state: TableLoadState) => Promise<any>
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

// Type for filter items
interface FilterItem {
  stateType: string
  label: string
  color: string
  isExclude: boolean
}

// Map state types to their display properties
const STATE_TYPE_CONFIG: Record<string, { label: string; color: string }> = {
  'workflow.Done': { label: 'Done', color: '#10b981' },
  'workflow.Error': { label: 'Error', color: '#ef4444' },
  'workflow.Await': { label: 'Await', color: '#3b82f6' },
  'workflow.Scheduled': { label: 'Scheduled', color: '#eab308' },
  'workflow.ScheduleStopped': { label: 'Stopped', color: '#6b7280' },
  'workflow.NextOperation': { label: 'Next', color: '#a855f7' },
}

export function StatesTable({ refreshTrigger, loadStates }: StatesTableProps) {
  const pagination = usePagination({ initialPageSize: 10 })
  const { deleteStates, tryRecover } = useWorkflowApi()
  const [selected, setSelected] = React.useState<{ [key: string]: boolean }>({})
  const [activeFilters, setActiveFilters] = React.useState<FilterItem[]>([])
  const [searchText, setSearchText] = React.useState('')
  
  // Build filter conditions based on current filters
  const buildWhereClause = React.useCallback((): predicate.Predicate | undefined => {
    const predicates: predicate.Predicate[] = []
    
    // Group filters by include/exclude
    const includeFilters = activeFilters.filter(f => !f.isExclude)
    const excludeFilters = activeFilters.filter(f => f.isExclude)
    
    // Handle include filters (OR logic)
    if (includeFilters.length > 0) {
      const includePredicates = includeFilters.map(filter => 
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
    
    // Handle exclude filters (NOT logic for each)
    excludeFilters.forEach(filter => {
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
    
    // Text search
    if (searchText.trim()) {
      predicates.push(
        createCompare(
          'ID',
          'LIKE',
          { "$type": "schema.String", "schema.String": `%${searchText.trim()}%` }
        )
      )
    }
    
    // Combine all predicates with AND
    if (predicates.length === 0) return undefined
    if (predicates.length === 1) return predicates[0]
    return createAnd(predicates)
  }, [activeFilters, searchText])

  // Update pagination where clause when filters change
  React.useEffect(() => {
    const whereClause = buildWhereClause()
    pagination.actions.setWhere(whereClause)
  }, [activeFilters, searchText, buildWhereClause, pagination.actions])
  
  // Adapt load function to work with the new hooks
  const adaptedLoad = React.useCallback(async (state: any) => {
    const tableState: TableLoadState = {
      limit: state.limit,
      offset: state.offset,
      sort: { ID: true },
      where: state.where
    }

    console.log('Sending tableState to API:', JSON.stringify(tableState, null, 2))
    
    const result = await loadStates(tableState)
    
    console.log('API returned items:', result.Items?.length, 'items')
    
    return {
      items: result.Items || [],
      next: result.Next ? 'has-next' : undefined,
      total: undefined
    }
  }, [loadStates])

  const { data, loading, error, refresh } = useTableData(adaptedLoad, pagination.state)

  // Filter management functions
  const addFilter = React.useCallback((stateType: string) => {
    // Check if filter already exists
    const exists = activeFilters.some(f => f.stateType === stateType && !f.isExclude)
    if (exists) return
    
    const config = STATE_TYPE_CONFIG[stateType]
    if (!config) return
    
    setActiveFilters(prev => [...prev, {
      stateType,
      label: config.label,
      color: config.color,
      isExclude: false
    }])
  }, [activeFilters])
  
  const removeFilter = React.useCallback((index: number) => {
    setActiveFilters(prev => prev.filter((_, i) => i !== index))
  }, [])
  
  const toggleFilterMode = React.useCallback((index: number) => {
    setActiveFilters(prev => prev.map((filter, i) => 
      i === index ? { ...filter, isExclude: !filter.isExclude } : filter
    ))
  }, [])
  
  const clearAllFilters = React.useCallback(() => {
    setActiveFilters([])
    setSearchText('')
  }, [])

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

  // Check if a state type is actively filtered
  const isStateTypeFiltered = React.useCallback((stateType: string) => {
    return activeFilters.some(f => f.stateType === stateType)
  }, [activeFilters])

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
        <StateDetailsRenderer 
          data={item} 
          onAddFilter={addFilter}
          isFilterActive={item.Data ? isStateTypeFiltered(item.Data.$type || '') : false}
        />
      )
    }
  ], [selected, data.items, addFilter, isStateTypeFiltered])

  return (
    <Card className="w-full h-full flex flex-col overflow-hidden">
      <CardHeader className="flex-shrink-0 border-b">
        <div className="flex items-center justify-between mb-2">
          <div>
            <CardTitle>States</CardTitle>
            <CardDescription>View workflow execution states</CardDescription>
          </div>
        </div>
        
        {/* Filter Bar */}
        <div className="flex items-center gap-3 min-h-[32px]">
          {activeFilters.length > 0 && (
            <>
              <span className="text-xs text-muted-foreground">Filtering:</span>
              <div className="flex items-center gap-2 flex-wrap flex-1">
                {activeFilters.map((filter, index) => (
                  <FilterPill
                    key={`${filter.stateType}-${index}`}
                    label={filter.label}
                    color={filter.color}
                    isExclude={filter.isExclude}
                    onRemove={() => removeFilter(index)}
                    onClick={() => toggleFilterMode(index)}
                  />
                ))}
                {activeFilters.length > 1 && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={clearAllFilters}
                    className="h-6 text-xs px-2"
                  >
                    Clear all
                  </Button>
                )}
              </div>
            </>
          )}
          
          {/* Search */}
          <div className="ml-auto flex items-center gap-2">
            <div className="relative">
              <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-3 w-3 text-muted-foreground" />
              <Input
                placeholder="Search by ID..."
                value={searchText}
                onChange={(e) => setSearchText(e.target.value)}
                className="h-7 w-48 pl-7 text-xs"
              />
            </div>
          </div>
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