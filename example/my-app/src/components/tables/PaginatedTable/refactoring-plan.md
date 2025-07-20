# PaginatedTable Refactoring Plan

## Overview
The PaginatedTable component has grown too large and complex. This refactoring plan aims to improve maintainability, testability, and performance.

## Phase 1: Extract Types and Interfaces

### Create types.ts
```typescript
export interface PaginatedTableState {
  limit: number
  offset: number
  where?: predicate.Predicate
  sort?: schemaless.Sort[]
}

export interface PaginatedData<T> {
  items: T[]
  next?: Cursor
}

export interface TableColumn<T> {
  key: keyof T
  header: string
  render?: (value: any, item: T) => React.ReactNode
}

export interface PaginatedTableProps<T> {
  columns: TableColumn<T>[]
  loadData: (state: PaginatedTableState) => Promise<PaginatedData<T>>
  className?: string
  emptyMessage?: string
  pageSize?: number
}
```

## Phase 2: Extract Custom Hooks

### usePagination.ts
```typescript
export function usePagination(initialPageSize = 10) {
  const [state, setState] = useState<PaginatedTableState>({
    limit: initialPageSize,
    offset: 0,
  })

  const goToNextPage = useCallback(() => {
    setState(prev => ({ ...prev, offset: prev.offset + prev.limit }))
  }, [])

  const goToPreviousPage = useCallback(() => {
    setState(prev => ({ ...prev, offset: Math.max(0, prev.offset - prev.limit) }))
  }, [])

  const setPageSize = useCallback((size: number) => {
    setState(prev => ({ ...prev, limit: size, offset: 0 }))
  }, [])

  const setFilter = useCallback((where?: predicate.Predicate) => {
    setState(prev => ({ ...prev, where, offset: 0 }))
  }, [])

  return { state, goToNextPage, goToPreviousPage, setPageSize, setFilter }
}
```

### useTableData.ts
```typescript
export function useTableData<T>(
  loadData: (state: PaginatedTableState) => Promise<PaginatedData<T>>,
  state: PaginatedTableState
) {
  const [data, setData] = useState<PaginatedData<T>>({ items: [] })
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<Error | null>(null)

  useEffect(() => {
    let cancelled = false
    
    const fetchData = async () => {
      setLoading(true)
      setError(null)
      
      try {
        const result = await loadData(state)
        if (!cancelled) {
          setData(result)
        }
      } catch (err) {
        if (!cancelled) {
          setError(err as Error)
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    fetchData()
    
    return () => { cancelled = true }
  }, [state, loadData])

  return { data, loading, error }
}
```

## Phase 3: Component Decomposition

### Main PaginatedTable Component
```typescript
export function PaginatedTable<T>({ 
  columns, 
  loadData, 
  className,
  emptyMessage = "No data available",
  pageSize = 10 
}: PaginatedTableProps<T>) {
  const pagination = usePagination(pageSize)
  const { data, loading, error } = useTableData(loadData, pagination.state)

  if (error) {
    return <ErrorState error={error} />
  }

  return (
    <div className={cn("paginated-table", className)}>
      <TableControls 
        onFilterChange={pagination.setFilter}
        currentFilter={pagination.state.where}
      />
      
      {loading ? (
        <LoadingState />
      ) : data.items.length === 0 ? (
        <EmptyState message={emptyMessage} />
      ) : (
        <TableContent columns={columns} data={data.items} />
      )}
      
      <PaginationBar
        hasNext={!!data.next}
        hasPrevious={pagination.state.offset > 0}
        onNext={pagination.goToNextPage}
        onPrevious={pagination.goToPreviousPage}
        pageSize={pagination.state.limit}
        onPageSizeChange={pagination.setPageSize}
        currentPage={Math.floor(pagination.state.offset / pagination.state.limit) + 1}
      />
    </div>
  )
}
```

## Phase 4: Filter Components Refactoring

### PredicateFilter using Component Map Pattern
```typescript
const PREDICATE_COMPONENTS: Record<string, React.FC<PredicateComponentProps>> = {
  'predicate.And': AndFilter,
  'predicate.Or': OrFilter,
  'predicate.Not': NotFilter,
  'predicate.Compare': CompareFilter,
}

export function PredicateFilter({ predicate, onChange }: PredicateFilterProps) {
  if (!predicate?.$type) {
    return null
  }

  const Component = PREDICATE_COMPONENTS[predicate.$type]
  if (!Component) {
    console.error(`Unknown predicate type: ${predicate.$type}`)
    return null
  }

  return <Component predicate={predicate} onChange={onChange} />
}
```

## Phase 5: Performance Optimizations

1. **Memoize expensive computations**
   ```typescript
   const memoizedColumns = useMemo(() => columns, [columns])
   const memoizedLoadData = useCallback(loadData, [])
   ```

2. **Virtual scrolling for large datasets**
   - Use react-window or react-virtualized for tables with many rows

3. **Debounce filter changes**
   ```typescript
   const debouncedSetFilter = useMemo(
     () => debounce(pagination.setFilter, 300),
     [pagination.setFilter]
   )
   ```

## Phase 6: Testing Strategy

1. **Unit tests for each hook**
   - Test pagination state changes
   - Test data fetching with various states
   - Test error handling

2. **Component tests**
   - Test each filter component in isolation
   - Test table rendering with different data shapes
   - Test user interactions

3. **Integration tests**
   - Test the complete table with mock data
   - Test filter application and pagination together

## Benefits of This Refactoring

1. **Improved Maintainability**
   - Smaller, focused files
   - Clear separation of concerns
   - Easier to locate and fix issues

2. **Better Testability**
   - Isolated components are easier to test
   - Hooks can be tested independently
   - Mocking is simplified

3. **Enhanced Reusability**
   - Hooks can be used in other components
   - Filter components can be used standalone
   - Table can handle any data type with proper typing

4. **Performance Improvements**
   - Reduced re-renders through memoization
   - Smaller bundle sizes with code splitting
   - Option for virtualization with large datasets

5. **Better Developer Experience**
   - Clear file structure
   - Type safety throughout
   - Easier onboarding for new developers

## Implementation Steps

1. Create the new folder structure
2. Extract types and interfaces
3. Create and test custom hooks
4. Split components one by one
5. Update imports in consuming components
6. Add comprehensive tests
7. Remove the old monolithic file

## Estimated Timeline

- Phase 1-2: 2 hours (types and hooks)
- Phase 3-4: 4 hours (component splitting)
- Phase 5: 2 hours (optimizations)
- Phase 6: 3 hours (testing)
- Total: ~11 hours of focused work