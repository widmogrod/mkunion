import React, { useRef, useEffect } from 'react'
import { TableColumn } from '../types'

interface VirtualizedTableContentProps<T> {
  columns: TableColumn<T>[]
  data: T[]
  renderItem: (item: T, column: TableColumn<T>) => React.ReactNode
  className?: string
  rowHeight?: number
  overscan?: number
}

// Constants for virtualization
const DEFAULT_ROW_HEIGHT = 53 // Default row height in pixels
const DEFAULT_OVERSCAN = 5 // Number of rows to render outside visible area
const HEADER_HEIGHT = 40 // Height of the table header

export function VirtualizedTableContent<T>({ 
  columns, 
  data, 
  renderItem,
  className = "",
  rowHeight = DEFAULT_ROW_HEIGHT,
  overscan = DEFAULT_OVERSCAN
}: VirtualizedTableContentProps<T>) {
  const scrollContainerRef = useRef<HTMLDivElement>(null)
  const [scrollTop, setScrollTop] = React.useState(0)
  const [containerHeight, setContainerHeight] = React.useState(600)

  // Calculate visible range
  const startIndex = Math.max(0, Math.floor(scrollTop / rowHeight) - overscan)
  const endIndex = Math.min(
    data.length - 1, 
    Math.ceil((scrollTop + containerHeight) / rowHeight) + overscan
  )

  // Calculate total height for scrollbar
  const totalHeight = data.length * rowHeight

  // Handle scroll events
  const handleScroll = (e: React.UIEvent<HTMLDivElement>) => {
    const target = e.target as HTMLDivElement
    setScrollTop(target.scrollTop)
  }

  // Update container height on resize
  useEffect(() => {
    const updateHeight = () => {
      if (scrollContainerRef.current) {
        setContainerHeight(scrollContainerRef.current.clientHeight)
      }
    }

    updateHeight()
    window.addEventListener('resize', updateHeight)
    return () => window.removeEventListener('resize', updateHeight)
  }, [])

  if (data.length === 0) {
    return (
      <div className="flex items-center justify-center py-8 text-muted-foreground">
        <p>No data to display</p>
      </div>
    )
  }

  // Only use virtualization for large datasets
  const useVirtualization = data.length > 100

  if (!useVirtualization) {
    // For small datasets, render normally without virtualization
    return (
      <div className={`flex-1 overflow-auto ${className}`}>
        <table className="w-full text-sm text-left">
          <thead className="sticky top-0 text-xs uppercase bg-muted/95 backdrop-blur-sm z-5">
            <tr>
              {columns.map((column, index) => (
                <th 
                  key={column.key as string || index} 
                  scope="col" 
                  className={`px-6 py-3 ${column.className || ''}`}
                >
                  {column.header}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {data.map((item, rowIndex) => (
              <tr key={rowIndex} className="bg-background border-b hover:bg-muted/50">
                {columns.map((column, colIndex) => (
                  <td 
                    key={`${rowIndex}-${column.key as string || colIndex}`} 
                    className={`px-6 py-4 ${column.className || ''}`}
                  >
                    {column.render 
                      ? column.render(item[column.key as keyof T], item)
                      : renderItem(item, column)
                    }
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    )
  }

  // Virtualized rendering for large datasets
  const visibleRows = data.slice(startIndex, endIndex + 1)

  return (
    <div 
      ref={scrollContainerRef}
      className={`flex-1 overflow-auto ${className}`}
      onScroll={handleScroll}
    >
      <div className="relative" style={{ height: totalHeight + HEADER_HEIGHT }}>
        {/* Fixed header */}
        <div className="sticky top-0 z-10">
          <table className="w-full text-sm text-left">
            <thead className="text-xs uppercase bg-muted/95 backdrop-blur-sm">
              <tr>
                {columns.map((column, index) => (
                  <th 
                    key={column.key as string || index} 
                    scope="col" 
                    className={`px-6 py-3 ${column.className || ''}`}
                  >
                    {column.header}
                  </th>
                ))}
              </tr>
            </thead>
          </table>
        </div>

        {/* Virtualized rows */}
        <table className="w-full text-sm text-left absolute top-0" style={{ paddingTop: HEADER_HEIGHT }}>
          <tbody>
            {/* Spacer for rows above the visible area */}
            {startIndex > 0 && (
              <tr style={{ height: startIndex * rowHeight }} aria-hidden="true">
                <td colSpan={columns.length} />
              </tr>
            )}
            
            {/* Visible rows */}
            {visibleRows.map((item, index) => {
              const actualIndex = startIndex + index
              return (
                <tr 
                  key={actualIndex} 
                  className="bg-background border-b hover:bg-muted/50"
                  style={{ height: rowHeight }}
                >
                  {columns.map((column, colIndex) => (
                    <td 
                      key={`${actualIndex}-${column.key as string || colIndex}`} 
                      className={`px-6 py-4 ${column.className || ''}`}
                    >
                      {column.render 
                        ? column.render(item[column.key as keyof T], item)
                        : renderItem(item, column)
                      }
                    </td>
                  ))}
                </tr>
              )
            })}
            
            {/* Spacer for rows below the visible area */}
            {endIndex < data.length - 1 && (
              <tr 
                style={{ height: (data.length - endIndex - 1) * rowHeight }} 
                aria-hidden="true"
              >
                <td colSpan={columns.length} />
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}