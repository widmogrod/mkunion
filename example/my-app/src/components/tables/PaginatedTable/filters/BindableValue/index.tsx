import React from 'react'
import * as predicate from '../../../../../workflow/github_com_widmogrod_mkunion_x_storage_predicate'
import * as schema from '../../../../../workflow/github_com_widmogrod_mkunion_x_schema'
import { BindableValueProps } from '../../types'
import { SchemaValue } from '../SchemaValue'

export function BindableValue({ bindable, onChange, disabled = false }: BindableValueProps) {
  if (!bindable) {
    return <div className="text-sm text-gray-500">No value</div>
  }

  const bindableType = bindable.$type
  if (!bindableType) {
    console.error('BindableValue: bindable.$type is undefined', bindable)
    return <div className="text-sm text-red-500">Unknown bindable type</div>
  }

  switch (bindableType) {
    case 'predicate.BindValue':
      const bindValue = bindable['predicate.BindValue']
      return (
        <input
          type="text"
          value={bindValue?.BindName || ''}
          disabled={disabled}
          className="px-2 py-1 text-sm border rounded"
          placeholder="Bind name"
        />
      )

    case 'predicate.Literal':
      const literal = bindable['predicate.Literal']
      return <SchemaValue data={literal?.Value} />

    case 'predicate.Locatable':
      const locatable = bindable['predicate.Locatable']
      return (
        <div className="text-sm text-gray-600">
          {locatable?.Location || 'Unknown location'}
        </div>
      )

    default:
      // This should never happen due to exhaustiveness checking
      const _exhaustive: never = bindableType
      return <div className="text-sm text-red-500">Unknown type: {_exhaustive}</div>
  }
}