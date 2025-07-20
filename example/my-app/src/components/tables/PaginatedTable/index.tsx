import React from 'react'
import { cn } from '../../../lib/utils'
import { PaginatedTableProps } from './types'
import { usePagination } from './hooks/usePagination'
import { useTableData } from './hooks/useTableData'
import { PaginationBar } from './components/PaginationBar'
import { TableContent } from './components/TableContent'
import { VirtualizedTableContent } from './components/VirtualizedTableContent'
import { LoadingState } from './components/LoadingState'
import { ErrorState } from './components/ErrorState'
import { EmptyState } from './components/EmptyState'

export function PaginatedTable<T>({
  columns,
  load,
  renderItem,
  className,
  emptyMessage = "No data available",
  pageSize = 10,
  enableFilters = false,
  enableSearch = false,
  enableVirtualization = true
}: PaginatedTableProps<T>) {
  const pagination = usePagination({ initialPageSize: pageSize })
  const { data, loading, error, refresh } = useTableData(load, pagination.state)

  // Debug logging removed for production

  if (error) {
    return <ErrorState error={error} onRetry={refresh} />
  }

  return (
    <div className={cn("h-full flex flex-col", className)}>
      {/* TODO: Add TableControls for filters and search */}
      
      {/* Main content area with proper scrolling */}
      <div className="flex-1 flex flex-col min-h-0">
        {loading ? (
          <LoadingState />
        ) : data.items.length === 0 ? (
          <EmptyState message={emptyMessage} />
        ) : enableVirtualization ? (
          <VirtualizedTableContent
            columns={columns}
            data={data.items}
            renderItem={renderItem}
          />
        ) : (
          <TableContent
            columns={columns}
            data={data.items}
            renderItem={renderItem}
          />
        )}
        
        {/* Pagination at bottom - always visible */}
        {(data.items.length > 0 || pagination.state.offset > 0) && (
          <PaginationBar
            hasNext={!!data.next}
            hasPrevious={pagination.state.offset > 0}
            onNext={pagination.actions.goToNextPage}
            onPrevious={pagination.actions.goToPreviousPage}
            onFirst={pagination.actions.goToFirstPage}
            onLast={pagination.actions.goToLastPage}
            pageSize={pagination.state.limit}
            onPageSizeChange={pagination.actions.setPageSize}
            currentPage={pagination.currentPage}
            totalItems={data.total}
          />
        )}
      </div>
    </div>
  )
 }

// Re-export types for convenience
export * from './types'