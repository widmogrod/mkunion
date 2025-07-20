import React from 'react'
import { CheckCircle, XCircle, AlertTriangle, Info, X } from 'lucide-react'
import { cn } from '../../lib/utils'

export type ToastType = 'success' | 'error' | 'warning' | 'info'

export interface Toast {
  id: string
  type: ToastType
  title: string
  description?: string
  duration?: number
  persistent?: boolean
  action?: {
    label: string
    onClick: () => void
  }
}

interface ToastProps {
  toast: Toast
  onDismiss: (id: string) => void
}

const toastConfig = {
  success: {
    icon: CheckCircle,
    className: 'border-green-200 bg-green-50 text-green-900 shadow-green-100',
    iconClassName: 'text-green-600',
    pulseColor: 'animate-pulse'
  },
  error: {
    icon: XCircle,
    className: 'border-red-200 bg-red-50 text-red-900 shadow-red-100',
    iconClassName: 'text-red-600',
    pulseColor: 'animate-pulse'
  },
  warning: {
    icon: AlertTriangle,
    className: 'border-yellow-200 bg-yellow-50 text-yellow-900 shadow-yellow-100',
    iconClassName: 'text-yellow-600',
    pulseColor: 'animate-pulse'
  },
  info: {
    icon: Info,
    className: 'border-blue-200 bg-blue-50 text-blue-900 shadow-blue-100',
    iconClassName: 'text-blue-600',
    pulseColor: 'animate-pulse'
  }
}

export function ToastComponent({ toast, onDismiss }: ToastProps) {
  const config = toastConfig[toast.type]
  const Icon = config.icon
  const [isVisible, setIsVisible] = React.useState(false)

  React.useEffect(() => {
    // Trigger enter animation
    const timer = setTimeout(() => setIsVisible(true), 50)
    return () => clearTimeout(timer)
  }, [])

  React.useEffect(() => {
    if (!toast.persistent && toast.duration !== 0) {
      const timer = setTimeout(() => {
        setIsVisible(false)
        // Give time for exit animation before dismissing
        setTimeout(() => onDismiss(toast.id), 300)
      }, toast.duration || 5000)

      return () => clearTimeout(timer)
    }
  }, [toast.id, toast.duration, toast.persistent, onDismiss])

  const handleDismiss = () => {
    setIsVisible(false)
    setTimeout(() => onDismiss(toast.id), 300)
  }

  return (
    <div
      className={cn(
        "relative flex w-full max-w-sm items-start gap-3 rounded-lg border p-4 shadow-lg transition-all duration-300 ease-out transform",
        "backdrop-blur-sm",
        isVisible 
          ? "translate-x-0 opacity-100 scale-100" 
          : "translate-x-full opacity-0 scale-95",
        config.className
      )}
      role="alert"
      aria-live="polite"
    >
      {/* Icon with subtle animation */}
      <Icon className={cn(
        "h-5 w-5 mt-0.5 flex-shrink-0 transition-all duration-500",
        config.iconClassName,
        toast.type === 'success' && "animate-bounce",
        toast.type === 'error' && "animate-pulse"
      )} />
      
      {/* Content */}
      <div className="flex-1 space-y-1">
        <h4 className="text-sm font-medium leading-none">{toast.title}</h4>
        {toast.description && (
          <p className="text-sm opacity-90">{toast.description}</p>
        )}
        {toast.action && (
          <button
            onClick={toast.action.onClick}
            className="text-sm font-medium underline underline-offset-4 hover:no-underline focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 rounded transition-all duration-150 hover:scale-105"
          >
            {toast.action.label}
          </button>
        )}
      </div>

      {/* Close button */}
      <button
        onClick={handleDismiss}
        className="flex-shrink-0 rounded-md p-1 hover:bg-black/10 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 transition-all duration-150 hover:scale-110"
        aria-label="Dismiss notification"
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  )
}