import * as schemaless from '../../../workflow/github_com_widmogrod_mkunion_x_storage_schemaless'
import * as predicate from '../../../workflow/github_com_widmogrod_mkunion_x_storage_predicate'

export type Cursor = string

export interface PaginatedTableState<T = any> {
  limit: number
  offset: number
  where?: predicate.Predicate
  sort?: schemaless.SortField[]
}

export interface PaginatedData<T> {
  items: T[]
  next?: Cursor
  total?: number
}

export interface TableColumn<T> {
  key: keyof T | string
  header: string | React.ReactNode
  render?: (value: any, item: T) => React.ReactNode
  className?: string
}

export interface PaginatedTableProps<T> {
  columns: TableColumn<T>[]
  load: (state: PaginatedTableState<T>) => Promise<PaginatedData<T>>
  renderItem: (item: T, column: TableColumn<T>) => React.ReactNode
  className?: string
  emptyMessage?: string
  pageSize?: number
  enableFilters?: boolean
  enableSearch?: boolean
}

export interface PaginationBarProps {
  hasNext: boolean
  hasPrevious: boolean
  onNext: () => void
  onPrevious: () => void
  onFirst: () => void
  onLast: (totalItems: number) => void
  pageSize: number
  onPageSizeChange: (size: number) => void
  currentPage: number
  totalItems?: number
}

export interface TableControlsProps {
  onFilterChange?: (filter?: predicate.Predicate) => void
  currentFilter?: predicate.Predicate
  enableSearch?: boolean
  onSearchChange?: (search: string) => void
  searchValue?: string
}

export interface PredicateFilterProps {
  predicate?: predicate.Predicate
  onChange?: (predicate?: predicate.Predicate) => void
}

export interface BindableValueProps {
  bindable?: predicate.Bindable
  onChange?: (bindable?: predicate.Bindable) => void
  disabled?: boolean
}

export interface SchemaValueProps {
  data?: any
  className?: string
}