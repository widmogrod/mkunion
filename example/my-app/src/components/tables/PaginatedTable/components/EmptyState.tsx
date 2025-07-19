import React from 'react'
import { FileX } from 'lucide-react'

interface EmptyStateProps {
  message?: string
  icon?: React.ReactNode
  className?: string
}

export function EmptyState({ 
  message = "No data available", 
  icon,
  className = ""
}: EmptyStateProps) {
  return (
    <div className={`flex flex-col items-center justify-center py-12 text-muted-foreground ${className}`}>
      {icon || <FileX className="h-12 w-12 mb-4" />}
      <p className="text-sm">{message}</p>
    </div>
  )
}