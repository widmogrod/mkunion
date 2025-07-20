import React from 'react'
import * as predicate from '../../../../../workflow/github_com_widmogrod_mkunion_x_storage_predicate'
import { Button } from '../../../../ui/button'
import { X } from 'lucide-react'
import { BindableValue } from '../BindableValue'

interface CompareFilterProps {
  predicate: predicate.Predicate
  onChange?: (predicate?: predicate.Predicate) => void
}

export function CompareFilter({ predicate: pred, onChange }: CompareFilterProps) {
  if (pred.$type !== 'predicate.Compare') {
    return null
  }
  
  const comparePredicate = pred['predicate.Compare']
  
  if (!comparePredicate) {
    return null
  }

  const handleRemove = () => {
    onChange?.(undefined)
  }

  const handleOperationChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    onChange?.({
      $type: 'predicate.Compare',
      'predicate.Compare': {
        ...comparePredicate,
        Operation: e.target.value
      }
    })
  }

  const handleBindValueChange = (bindValue?: predicate.Bindable) => {
    if (!bindValue) {
      handleRemove()
      return
    }
    
    onChange?.({
      $type: 'predicate.Compare',
      'predicate.Compare': {
        ...comparePredicate,
        BindValue: bindValue
      }
    })
  }

  return (
    <div className="flex items-center gap-2 p-2 border rounded-md bg-white">
      <Button
        variant="ghost"
        size="icon"
        className="h-6 w-6"
        onClick={handleRemove}
      >
        <X className="h-4 w-4" />
      </Button>
      
      <input
        type="text"
        value={comparePredicate.Location || ''}
        disabled
        className="px-2 py-1 text-sm border rounded bg-gray-50"
        placeholder="Location"
      />
      
      <select
        value={comparePredicate.Operation}
        onChange={handleOperationChange}
        className="px-2 py-1 text-sm border rounded"
      >
        <option value="==">==</option>
        <option value="!=">!=</option>
        <option value="<">{"<"}</option>
        <option value="<=">{"<="}</option>
        <option value=">">{">"}</option>
        <option value=">=">{">="}</option>
      </select>
      
      <BindableValue
        bindable={comparePredicate.BindValue}
        onChange={handleBindValueChange}
        disabled={false}
      />
    </div>
  )
}