import React from 'react'
import { Search } from 'lucide-react'
import { Input } from '../../../ui/input'
import { RefreshButton } from '../../../ui/RefreshButton'
import { FilterPill } from '../../FilterPill'
import { Button } from '../../../ui/button'

export interface FilterItem {
  stateType: string
  label: string
  color: string
  isExclude: boolean
}

interface TableControlsProps {
  // Search functionality
  searchText?: string
  onSearchChange?: (value: string) => void
  searchPlaceholder?: string
  
  // Refresh functionality
  onRefresh?: () => void
  isLoading?: boolean
  refreshTitle?: string
  
  // Filter functionality
  activeFilters?: FilterItem[]
  onRemoveFilter?: (index: number) => void
  onToggleFilterMode?: (index: number) => void
  onClearAllFilters?: () => void
  
  // Layout options
  showSearch?: boolean
  showRefresh?: boolean
  showFilters?: boolean
  className?: string
  
  // Additional content
  children?: React.ReactNode
}

/**
 * Standardized table controls component with Apple-inspired design
 * Provides search, refresh, and filter functionality for tables
 */
export function TableControls({
  searchText = '',
  onSearchChange,
  searchPlaceholder = 'Search...',
  onRefresh,
  isLoading = false,
  refreshTitle = 'Refresh data',
  activeFilters = [],
  onRemoveFilter,
  onToggleFilterMode,
  onClearAllFilters,
  showSearch = true,
  showRefresh = true,
  showFilters = true,
  className = '',
  children
}: TableControlsProps) {
  const hasActiveFilters = activeFilters.length > 0

  return (
    <div className={`flex items-center gap-3 flex-1 justify-end ${className}`}>
      {/* Active Filters */}
      {showFilters && hasActiveFilters && (
        <>
          <span className="text-xs text-muted-foreground flex-shrink-0">Filtering:</span>
          <div className="flex items-center gap-2 flex-wrap">
            {activeFilters.map((filter, index) => (
              <FilterPill
                key={`${filter.stateType}-${index}`}
                label={filter.label}
                color={filter.color}
                isExclude={filter.isExclude}
                onRemove={() => onRemoveFilter?.(index)}
                onClick={() => onToggleFilterMode?.(index)}
                stateType={filter.stateType}
              />
            ))}
            {activeFilters.length > 0 && onClearAllFilters && (
              <Button
                variant="ghost"
                size="sm"
                onClick={onClearAllFilters}
                className="h-6 text-xs px-2"
              >
                Clear all
              </Button>
            )}
          </div>
        </>
      )}
      
      {/* Additional content slot */}
      {children}
      
      {/* Search and Refresh */}
      <div className="flex items-center gap-2 flex-shrink-0">
        {/* Search */}
        {showSearch && onSearchChange && (
          <div className="relative">
            <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-3 w-3 text-muted-foreground" />
            <Input
              placeholder={searchPlaceholder}
              value={searchText}
              onChange={(e) => onSearchChange(e.target.value)}
              className="h-7 w-48 pl-7 text-xs"
            />
          </div>
        )}
        
        {/* Refresh */}
        {showRefresh && onRefresh && (
          <RefreshButton
            onRefresh={onRefresh}
            isLoading={isLoading}
            title={refreshTitle}
          />
        )}
      </div>
    </div>
  )
}