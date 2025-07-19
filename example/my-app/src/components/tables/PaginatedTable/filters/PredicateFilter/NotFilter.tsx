import React from 'react'
import * as predicate from '../../../../../workflow/github_com_widmogrod_mkunion_x_storage_predicate'
import { Button } from '../../../../ui/button'
import { X } from 'lucide-react'
import { PredicateFilter } from './index'

interface NotFilterProps {
  predicate: predicate.Predicate
  onChange?: (predicate?: predicate.Predicate) => void
}

export function NotFilter({ predicate: pred, onChange }: NotFilterProps) {
  if (pred.$type !== 'predicate.Not') {
    return null
  }
  
  const notPredicate = pred['predicate.Not']
  
  if (!notPredicate?.P) {
    return null
  }

  const handleRemove = () => {
    onChange?.(undefined)
  }

  const handleChildChange = (newPredicate?: predicate.Predicate) => {
    if (!newPredicate) {
      onChange?.(undefined)
    } else {
      onChange?.({
        $type: 'predicate.Not',
        'predicate.Not': { P: newPredicate }
      })
    }
  }

  return (
    <div className="border rounded-md p-2 space-y-2 bg-red-50">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium text-red-700">NOT</span>
        <Button
          variant="ghost"
          size="icon"
          className="h-6 w-6"
          onClick={handleRemove}
        >
          <X className="h-4 w-4" />
        </Button>
      </div>
      <div className="pl-4">
        <PredicateFilter
          predicate={notPredicate.P}
          onChange={handleChildChange}
        />
      </div>
    </div>
  )
}