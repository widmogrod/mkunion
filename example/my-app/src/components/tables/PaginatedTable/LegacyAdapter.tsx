import React, { useCallback, useMemo, useRef } from 'react'
import * as schemaless from '../../../workflow/github_com_widmogrod_mkunion_x_storage_schemaless'
import * as schema from '../../../workflow/github_com_widmogrod_mkunion_x_schema'
import * as predicate from '../../../workflow/github_com_widmogrod_mkunion_x_storage_predicate'
import { PaginatedTable } from './index'
import { PaginatedTableState as NewTableState } from './types'
import { Button } from '../../ui/button'

// Legacy types (from old PaginatedTable)
export type PaginatedTableSort = { [key: string]: boolean }

export type PaginatedTableState<T> = {
  limit: number
  offset: number
  selected: { [key: string]: boolean }
  sort?: PaginatedTableSort
  where?: predicate.Predicate
  nextPage?: string
  prevPage?: string
}

export type PaginatedTableAction<T> = {
  name: string
  action: (state: PaginatedTableState<T>, ctx: PaginatedTableContext<T>) => Promise<void>
}

export type PaginatedTableContext<T> = {
  refresh: () => void
  clearSelection: () => void
  filter: (x: predicate.WherePredicates) => void
}

export type LegacyPaginatedTableProps<T> = {
  limit?: number
  sort?: PaginatedTableSort
  load: (input: PaginatedTableState<T>) => Promise<schemaless.PageResult<schemaless.Record<T>>>
  mapData?: (data: schemaless.Record<T>, ctx: PaginatedTableContext<T>) => JSX.Element
  actions?: PaginatedTableAction<T>[]
}

// Adapter component that wraps the new PaginatedTable
export function LegacyPaginatedTable<T>(props: LegacyPaginatedTableProps<T>) {
  const [selected, setSelected] = React.useState<{ [key: string]: boolean }>({})
  const [refreshKey, setRefreshKey] = React.useState(0)

  // Convert legacy sort to new sort format
  const initialSort = useMemo(() => {
    if (!props.sort) return undefined
    return Object.entries(props.sort).map(([field, asc]) => ({
      Field: field,
      Asc: asc
    }))
  }, [props.sort])

  // Store props.load in a ref to avoid recreating adaptedLoad
  const propsLoadRef = useRef(props.load)
  propsLoadRef.current = props.load

  // Adapter for the load function
  const adaptedLoad = useCallback(async (state: NewTableState<T>) => {
    const legacyState: PaginatedTableState<T> = {
      limit: state.limit,
      offset: state.offset,
      selected,
      sort: props.sort,
      where: state.where,
      // The old system doesn't use offset, it uses nextPage/prevPage cursors
      nextPage: undefined,
      prevPage: undefined
    }

    const result = await propsLoadRef.current(legacyState)
    
    console.log('LegacyAdapter: Load result', { 
      resultItems: result.Items?.length || 0,
      hasNext: !!result.Next,
      sampleItem: result.Items?.[0],
      sampleType: result.Items?.[0]?.Type // This will help us identify if it's flows or states
    })
    
    return {
      items: result.Items || [],
      next: result.Next ? 'has-next' : undefined, // Convert to simple cursor
      total: undefined // Legacy API doesn't provide total
    }
  }, [selected, props.sort])

  // Context for legacy components
  const ctx: PaginatedTableContext<T> = useMemo(() => ({
    refresh: () => setRefreshKey(prev => prev + 1),
    clearSelection: () => setSelected({}),
    filter: (x: predicate.WherePredicates) => {
      console.warn('Legacy filter method called - not implemented in adapter')
    }
  }), [])

  // Render function for table cells
  const renderItem = useCallback((item: schemaless.Record<T>) => {
    if (!props.mapData) {
      return <pre className="text-xs">{JSON.stringify(item.Data, null, 2)}</pre>
    }
    return props.mapData(item, ctx)
  }, [props, ctx])

  // Columns configuration
  const columns = useMemo(() => [
    {
      key: 'selection',
      header: (
        <input
          type="checkbox"
          checked={Object.keys(selected).length > 0 && Object.values(selected).every(v => v)}
          onChange={(e) => {
            const newSelected: { [key: string]: boolean } = {}
            // This is a simplified version - in real implementation you'd need access to all items
            setSelected(e.target.checked ? newSelected : {})
          }}
        />
      ),
      render: (value: any, item: schemaless.Record<T>) => {
        const id = item.ID || ''
        return (
          <input
            type="checkbox"
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
      render: (value: any, item: schemaless.Record<T>) => renderItem(item)
    }
  ], [selected, renderItem])

  return (
    <div className="space-y-4">
      <PaginatedTable<schemaless.Record<T>>
        key={refreshKey}
        columns={columns}
        load={adaptedLoad}
        renderItem={(item, column) => {
          if (column.render) {
            return column.render(item, item)
          }
          return renderItem(item)
        }}
        pageSize={props.limit}
      />
      
      {props.actions && props.actions.length > 0 && (
        <div className="flex gap-2">
          {props.actions.map((action, index) => (
            <Button
              key={index}
              variant="outline"
              size="sm"
              disabled={Object.keys(selected).filter(k => selected[k]).length === 0}
              onClick={() => {
                const legacyState: PaginatedTableState<T> = {
                  limit: props.limit || 3,
                  offset: 0,
                  selected,
                  sort: props.sort
                }
                action.action(legacyState, ctx)
              }}
            >
              {action.name}
            </Button>
          ))}
        </div>
      )}
    </div>
  )
}