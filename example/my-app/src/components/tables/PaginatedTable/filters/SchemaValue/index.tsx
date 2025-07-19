import React from 'react'
import * as schema from '../../../../../workflow/github_com_widmogrod_mkunion_x_schema'
import { SchemaValueProps } from '../../types'

export function SchemaValue({ data, className = "" }: SchemaValueProps) {
  if (!data) {
    return null
  }

  const schemaType = data.$type
  if (!schemaType) {
    console.error('SchemaValue: data.$type is undefined', data)
    return <div className="text-sm text-red-500">Unknown schema type</div>
  }

  switch (schemaType) {
    case 'schema.None':
      return <span className={`text-gray-400 ${className}`}>none</span>

    case 'schema.String':
      return (
        <input
          type="text"
          value={data['schema.String'] || ''}
          disabled
          className={`px-2 py-1 text-sm border rounded bg-gray-50 ${className}`}
        />
      )

    case 'schema.Number':
      return <span className={className}>{data['schema.Number']}</span>

    case 'schema.Binary':
      return <span className={`text-gray-600 ${className}`}>binary</span>

    case 'schema.Bool':
      return <span className={className}>{data['schema.Bool'] ? 'true' : 'false'}</span>

    case 'schema.List':
      const listData = data['schema.List']
      if (!listData || listData.length === 0) {
        return <span className={`text-gray-400 ${className}`}>[]</span>
      }
      return (
        <ul className={`text-sm ${className}`}>
          {listData.map((item: schema.Schema, index: number) => (
            <li key={index}>
              <SchemaValue data={item} />
            </li>
          ))}
        </ul>
      )

    case 'schema.Map':
      const mapData = data['schema.Map']
      const keys = Object.keys(mapData || {})
      
      if (keys.length === 0) {
        return <span className={`text-gray-400 ${className}`}>{'{}'}</span>
      }
      
      return (
        <div className={`text-sm ${className}`}>
          {keys.map(key => (
            <div key={key} className="flex gap-2">
              <span className="font-medium">{key}:</span>
              <SchemaValue data={mapData[key]} />
            </div>
          ))}
        </div>
      )

    default:
      console.warn('SchemaValue: Unhandled schema type', schemaType)
      return <span className={`text-red-500 ${className}`}>Unknown: {schemaType}</span>
  }
}