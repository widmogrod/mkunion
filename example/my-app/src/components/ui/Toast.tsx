import React from 'react'
import { CheckCircle, XCircle, AlertTriangle, Info, X } from 'lucide-react'
import { cn } from '../../utils/cn'

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
    className: 'border-green-200 bg-green-50 text-green-900',
    iconClassName: 'text-green-600'
  },
  error: {
    icon: XCircle,
    className: 'border-red-200 bg-red-50 text-red-900',
    iconClassName: 'text-red-600'
  },
  warning: {
    icon: AlertTriangle,
    className: 'border-yellow-200 bg-yellow-50 text-yellow-900',
    iconClassName: 'text-yellow-600'
  },
  info: {
    icon: Info,
    className: 'border-blue-200 bg-blue-50 text-blue-900',
    iconClassName: 'text-blue-600'
  }
}

export function ToastComponent({ toast, onDismiss }: ToastProps) {
  const config = toastConfig[toast.type]
  const Icon = config.icon

  React.useEffect(() => {
    if (!toast.persistent && toast.duration !== 0) {
      const timer = setTimeout(() => {
        onDismiss(toast.id)
      }, toast.duration || 5000)

      return () => clearTimeout(timer)
    }
  }, [toast.id, toast.duration, toast.persistent, onDismiss])

  return (
    <div
      className={cn(
        "relative flex w-full max-w-sm items-start gap-3 rounded-lg border p-4 shadow-lg transition-all duration-300 ease-out",
        "backdrop-blur-sm",
        config.className
      )}
      role="alert"
      aria-live="polite"
    >
      {/* Icon */}
      <Icon className={cn("h-5 w-5 mt-0.5 flex-shrink-0", config.iconClassName)} />
      
      {/* Content */}
      <div className="flex-1 space-y-1">
        <h4 className="text-sm font-medium leading-none">{toast.title}</h4>
        {toast.description && (
          <p className="text-sm opacity-90">{toast.description}</p>
        )}
        {toast.action && (
          <button
            onClick={toast.action.onClick}
            className="text-sm font-medium underline underline-offset-4 hover:no-underline focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 rounded"
          >
            {toast.action.label}
          </button>
        )}
      </div>

      {/* Close button */}
      <button
        onClick={() => onDismiss(toast.id)}
        className="flex-shrink-0 rounded-md p-1 hover:bg-black/10 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 transition-colors"
        aria-label="Dismiss notification"
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  )
}