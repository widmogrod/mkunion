import React from 'react'
import * as predicate from '../../../../../workflow/github_com_widmogrod_mkunion_x_storage_predicate'
import { assertNever } from '../../../../../utils/type-helpers'
import { PredicateFilterProps } from '../../types'
import { AndFilter } from './AndFilter'
import { OrFilter } from './OrFilter'
import { NotFilter } from './NotFilter'
import { CompareFilter } from './CompareFilter'

export function PredicateFilter({ predicate: pred, onChange }: PredicateFilterProps) {
  if (!pred) {
    return null
  }

  const predicateType = pred.$type
  if (!predicateType) {
    console.error('PredicateFilter: predicate.$type is undefined', pred)
    return <div className="text-red-500">Unknown predicate type</div>
  }

  switch (predicateType) {
    case 'predicate.And':
      return <AndFilter predicate={pred} onChange={onChange} />
    
    case 'predicate.Or':
      return <OrFilter predicate={pred} onChange={onChange} />
    
    case 'predicate.Not':
      return <NotFilter predicate={pred} onChange={onChange} />
    
    case 'predicate.Compare':
      return <CompareFilter predicate={pred} onChange={onChange} />
    
    default:
      return assertNever(predicateType)
  }
}