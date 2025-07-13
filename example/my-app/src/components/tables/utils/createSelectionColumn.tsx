import React from 'react'
import { AppleCheckbox } from '../../ui/AppleCheckbox'
import * as schemaless from '../../../workflow/github_com_widmogrod_mkunion_x_storage_schemaless'

interface SelectionColumnConfig<T> {
  selected: { [key: string]: boolean }
  setSelected: React.Dispatch<React.SetStateAction<{ [key: string]: boolean }>>
  data: schemaless.Record<T>[]
}

/**
 * Creates a standardized selection column for tables using schemaless.Record data
 * Applies Apple-inspired checkbox styling and optimized spacing
 */
export function createSelectionColumn<T>({ selected, setSelected, data }: SelectionColumnConfig<T>) {
  return {
    key: 'selection' as const,
    className: 'w-12 px-3 py-3', // Optimized spacing for checkbox column
    header: (
      <div className="flex items-center justify-center">
        <AppleCheckbox
          checked={Object.keys(selected).length > 0 && Object.values(selected).every(v => v)}
          onChange={(checked) => {
            const newSelected: { [key: string]: boolean } = {}
            if (checked) {
              data.forEach((item: schemaless.Record<T>) => {
                if (item.ID) newSelected[item.ID] = true
              })
            }
            setSelected(newSelected)
          }}
        />
      </div>
    ),
    render: (value: any, item: schemaless.Record<T>) => {
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
  }
}