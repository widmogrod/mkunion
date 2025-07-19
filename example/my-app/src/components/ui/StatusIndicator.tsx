import React from 'react'
import { CheckCircle, XCircle, AlertTriangle } from 'lucide-react'
import { cn } from '../../lib/utils'

type StatusType = 'success' | 'error' | 'warning' | 'idle'

interface StatusIndicatorProps {
  status: StatusType
  className?: string
}

const statusConfig = {
  success: {
    icon: CheckCircle,
    className: 'text-green-600',
    animation: 'animate-bounce'
  },
  error: {
    icon: XCircle,
    className: 'text-red-600', 
    animation: 'animate-pulse'
  },
  warning: {
    icon: AlertTriangle,
    className: 'text-yellow-600',
    animation: 'animate-pulse'
  },
  idle: {
    icon: null,
    className: '',
    animation: ''
  }
}

export function StatusIndicator({ status, className }: StatusIndicatorProps) {
  if (status === 'idle') return null
  
  const config = statusConfig[status]
  const Icon = config.icon!
  
  return (
    <Icon 
      className={cn(
        "h-4 w-4 ml-2 transition-all duration-300",
        config.className,
        config.animation,
        className
      )}
    />
  )
}