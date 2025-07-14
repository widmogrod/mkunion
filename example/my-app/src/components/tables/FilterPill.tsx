import React from 'react'
import { X } from 'lucide-react'
import { cn } from '../../lib/utils'
import { StatusIcon, StatusType } from '../ui/icons'

interface FilterPillProps {
  label: string
  color: string
  isExclude?: boolean
  onRemove: () => void
  onClick: () => void
  stateType?: string
}

export function FilterPill({ label, color, isExclude = false, onRemove, onClick, stateType }: FilterPillProps) {
  // Map state types to status icon types
  const getStatusFromStateType = (type?: string): StatusType | null => {
    if (!type) return null
    
    const mapping: Record<string, StatusType> = {
      'workflow.Done': 'done',
      'workflow.Error': 'error',
      'workflow.Await': 'info',
      'workflow.Scheduled': 'scheduled',
      'workflow.ScheduleStopped': 'paused',
      'workflow.NextOperation': 'info'
    }
    
    return mapping[type] as StatusType || null
  }
  
  const status = getStatusFromStateType(stateType)
  return (
    <div
      className={cn(
        "inline-flex items-center gap-1 px-3 py-1 rounded-full text-xs font-medium transition-all duration-200 cursor-pointer select-none animate-in fade-in zoom-in-95",
        "hover:scale-105 active:scale-95",
        isExclude ? [
          "bg-background border-2",
          "hover:bg-muted"
        ] : [
          "text-white",
          "hover:opacity-90"
        ]
      )}
      style={{
        backgroundColor: isExclude ? undefined : color,
        borderColor: isExclude ? color : undefined,
        color: isExclude ? color : undefined
      }}
      onClick={onClick}
      title={isExclude ? `Excluding ${label} - Click to include` : `Including ${label} - Click to exclude`}
    >
      {status && (
        <StatusIcon 
          status={status} 
          size="xs" 
          colored={!isExclude}
        />
      )}
      {isExclude && <span className="line-through">{label}</span>}
      {!isExclude && <span>{label}</span>}
      <button
        className={cn(
          "ml-1 rounded-full p-0.5 transition-all duration-150 -mr-0.5 hover:scale-110",
          isExclude ? "hover:bg-muted-foreground/20" : "hover:bg-white/20"
        )}
        onClick={(e) => {
          e.stopPropagation()
          onRemove()
        }}
        aria-label={`Remove ${label} filter`}
      >
        <X className="h-3 w-3" />
      </button>
    </div>
  )
}