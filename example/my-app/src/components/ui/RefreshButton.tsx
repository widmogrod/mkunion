import React from 'react'
import { RotateCcw } from 'lucide-react'
import { cn } from '../../utils/cn'

interface RefreshButtonProps {
  onRefresh: () => void
  isLoading?: boolean
  disabled?: boolean
  className?: string
  title?: string
}

export function RefreshButton({ 
  onRefresh, 
  isLoading = false, 
  disabled = false,
  className,
  title = "Refresh data"
}: RefreshButtonProps) {
  const handleClick = () => {
    if (!isLoading && !disabled) {
      onRefresh()
    }
  }

  return (
    <button
      type="button"
      onClick={handleClick}
      disabled={disabled || isLoading}
      title={title}
      aria-label={title}
      className={cn(
        // Base styling with Apple-inspired proportions
        "inline-flex items-center justify-center w-7 h-7 rounded-md",
        "border-0 bg-transparent transition-all duration-150 ease-out",
        "focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:ring-offset-1",
        
        // Normal state - subtle and unobtrusive
        "text-muted-foreground hover:text-foreground",
        "hover:bg-muted/50 active:bg-muted",
        
        // Active state with subtle scale feedback
        "active:scale-95 hover:scale-105 transform",
        
        // Disabled state
        (disabled || isLoading) && "opacity-50 cursor-not-allowed",
        !disabled && !isLoading && "cursor-pointer",
        
        className
      )}
    >
      <RotateCcw 
        className={cn(
          "w-4 h-4 transition-transform duration-150 ease-out",
          // Spinning animation when loading
          isLoading && "animate-spin"
        )}
      />
    </button>
  )
}