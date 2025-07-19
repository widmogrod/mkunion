import React from 'react'
import { ArrowRight } from 'lucide-react'
import { useNavigationWithContext } from '../../hooks/useNavigation'
import { cn } from '../../lib/utils'

interface ClickableWorkflowNameProps {
  workflowName: string
  workflowId?: string
  className?: string
  showArrow?: boolean
}

export function ClickableWorkflowName({ 
  workflowName, 
  workflowId,
  className,
  showArrow = false
}: ClickableWorkflowNameProps) {
  const { navigateToExecutions } = useNavigationWithContext()

  const handleClick = (e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    navigateToExecutions({ workflow: workflowName })
  }

  return (
    <button
      onClick={handleClick}
      className={cn(
        "group inline-flex items-center gap-1 text-left",
        "text-primary hover:text-primary/80",
        "underline decoration-dotted underline-offset-4 decoration-muted-foreground/40",
        "hover:decoration-primary/60 transition-all duration-200",
        "focus:outline-none focus:ring-2 focus:ring-primary/20 focus:ring-offset-1 rounded-sm",
        className
      )}
      title={`View executions for workflow: ${workflowName}`}
    >
      <span className="font-medium">{workflowName}</span>
      {showArrow && (
        <ArrowRight className="h-3 w-3 opacity-0 group-hover:opacity-100 transition-opacity" />
      )}
    </button>
  )
}