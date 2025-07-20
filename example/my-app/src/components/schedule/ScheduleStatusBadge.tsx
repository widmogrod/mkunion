import React from 'react'
import { Badge } from '../ui/badge'
import { cn } from '../../lib/utils'

interface ScheduleStatusBadgeProps {
  status: 'active' | 'paused'
  className?: string
}

export function ScheduleStatusBadge({ status, className }: ScheduleStatusBadgeProps) {
  const config = {
    active: {
      label: 'Active',
      className: 'bg-green-500/10 text-green-700 border-green-500/20',
    },
    paused: {
      label: 'Paused',
      className: 'bg-gray-500/10 text-gray-700 border-gray-500/20',
    }
  }

  const { label, className: statusClassName } = config[status]

  return (
    <Badge 
      variant="outline"
      className={cn(statusClassName, className)}
    >
      {label}
    </Badge>
  )
}