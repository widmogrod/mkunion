import React from 'react'
import { TableColumn } from '../types'

interface TableContentProps<T> {
  columns: TableColumn<T>[]
  data: T[]
  renderItem: (item: T, column: TableColumn<T>) => React.ReactNode
  className?: string
}

export function TableContent<T>({ 
  columns, 
  data, 
  renderItem,
  className = ""
}: TableContentProps<T>) {
  if (data.length === 0) {
    return (
      <div className="flex items-center justify-center py-8 text-muted-foreground">
        <p>No data to display</p>
      </div>
    )
  }

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