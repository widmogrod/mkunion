import React from 'react'
import * as predicate from '../../../../../workflow/github_com_widmogrod_mkunion_x_storage_predicate'
import { Button } from '../../../../ui/button'
import { X } from 'lucide-react'
import { PredicateFilter } from './index'

interface AndFilterProps {
  predicate: predicate.Predicate
  onChange?: (predicate?: predicate.Predicate) => void
}

export function AndFilter({ predicate: pred, onChange }: AndFilterProps) {
  if (pred.$type !== 'predicate.And') {
    return null
  }
  
  const andPredicate = pred['predicate.And']
  
  if (!andPredicate?.L || andPredicate.L.length === 0) {
    return null
  }

  const handleRemove = () => {
    onChange?.(undefined)
  }

  const handleChildChange = (index: number) => (newPredicate?: predicate.Predicate) => {
    if (!andPredicate.L) return
    
    const newPredicates = andPredicate.L
      .map((p, i) => i === index ? newPredicate : p)
      .filter((p): p is predicate.Predicate => p !== undefined)
    
    if (newPredicates.length === 0) {
      onChange?.(undefined)
    } else if (newPredicates.length === 1) {
      onChange?.(newPredicates[0])
    } else {
      onChange?.({
        $type: 'predicate.And',
        'predicate.And': { L: newPredicates }
      })
    }
  }

  return (
    <div className="border rounded-md p-2 space-y-2 bg-gray-50">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium text-gray-700">AND</span>
        <Button
          variant="ghost"
          size="icon"
          className="h-6 w-6"
          onClick={handleRemove}
        >
          <X className="h-4 w-4" />
        </Button>
      </div>
      <div className="space-y-2 pl-4">
        {andPredicate.L.map((childPredicate, index) => (
          <PredicateFilter
            key={index}
            predicate={childPredicate}
            onChange={handleChildChange(index)}
          />
        ))}
      </div>
    </div>
  )
}