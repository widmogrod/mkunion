import React from 'react'
import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight } from 'lucide-react'
import { Button } from '../../../ui/button'
import { PaginationBarProps } from '../types'

const PAGE_SIZE_OPTIONS = [10, 25, 50, 100]

export function PaginationBar({
  hasNext,
  hasPrevious,
  onNext,
  onPrevious,
  onFirst,
  onLast,
  pageSize,
  onPageSizeChange,
  currentPage,
  totalItems
}: PaginationBarProps) {
  const totalPages = totalItems ? Math.ceil(totalItems / pageSize) : undefined

  return (
    <div className="flex-shrink-0 bg-background border-t border-border px-3 py-2">
      <div className="flex items-center justify-between">
        {/* Left: Page info - more compact */}
        <div className="flex items-center space-x-3">
          <span className="text-xs text-muted-foreground font-medium">
            Page {currentPage}{totalPages && ` of ${totalPages}`}
          </span>
          
          {/* Page size selector - more compact */}
          <div className="flex items-center space-x-1">
            <span className="text-xs text-muted-foreground">Rows:</span>
            <select
              value={pageSize}
              onChange={(e) => onPageSizeChange(Number(e.target.value))}
              className="h-6 text-xs rounded border border-input bg-background px-2 py-0"
            >
              {PAGE_SIZE_OPTIONS.map((size) => (
                <option key={size} value={size}>
                  {size}
                </option>
              ))}
            </select>
          </div>
        </div>
        
        {/* Right: Navigation buttons - compact */}
        <div className="flex items-center space-x-1">
          <Button
            variant="ghost"
            size="sm"
            onClick={onFirst}
            disabled={!hasPrevious}
            aria-label="First page"
            className="h-7 w-7 p-0"
          >
            <ChevronsLeft className="h-3 w-3" />
          </Button>
          
          <Button
            variant="ghost"
            size="sm"
            onClick={onPrevious}
            disabled={!hasPrevious}
            aria-label="Previous page"
            className="h-7 w-7 p-0"
          >
            <ChevronLeft className="h-3 w-3" />
          </Button>
          
          <Button
            variant="ghost"
            size="sm"
            onClick={onNext}
            disabled={!hasNext}
            aria-label="Next page"
            className="h-7 w-7 p-0"
          >
            <ChevronRight className="h-3 w-3" />
          </Button>
          
          <Button
            variant="ghost"
            size="sm"
            onClick={() => totalItems && onLast(totalItems)}
            disabled={!hasNext || !totalItems}
            aria-label="Last page"
            className="h-7 w-7 p-0"
          >
            <ChevronsRight className="h-3 w-3" />
          </Button>
        </div>
      </div>
    </div>
  )
}