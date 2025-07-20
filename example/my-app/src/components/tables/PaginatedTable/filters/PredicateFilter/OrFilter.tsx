import React from 'react'
import * as predicate from '../../../../../workflow/github_com_widmogrod_mkunion_x_storage_predicate'
import { Button } from '../../../../ui/button'
import { X } from 'lucide-react'
import { PredicateFilter } from './index'

interface OrFilterProps {
  predicate: predicate.Predicate
  onChange?: (predicate?: predicate.Predicate) => void
}

export function OrFilter({ predicate: pred, onChange }: OrFilterProps) {
  if (pred.$type !== 'predicate.Or') {
    return null
  }
  
  const orPredicate = pred['predicate.Or']
  
  if (!orPredicate?.L || orPredicate.L.length === 0) {
    return null
  }

  const handleRemove = () => {
    onChange?.(undefined)
  }

  const handleChildChange = (index: number) => (newPredicate?: predicate.Predicate) => {
    if (!orPredicate.L) return
    
    const newPredicates = orPredicate.L
      .map((p, i) => i === index ? newPredicate : p)
      .filter((p): p is predicate.Predicate => p !== undefined)
    
    if (newPredicates.length === 0) {
      onChange?.(undefined)
    } else if (newPredicates.length === 1) {
      onChange?.(newPredicates[0])
    } else {
      onChange?.({
        $type: 'predicate.Or',
        'predicate.Or': { L: newPredicates }
      })
    }
  }

  return (
    <div className="border rounded-md p-2 space-y-2 bg-blue-50">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium text-blue-700">OR</span>
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
        {orPredicate.L.map((childPredicate, index) => (
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