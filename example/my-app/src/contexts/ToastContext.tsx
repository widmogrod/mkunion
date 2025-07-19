import React, { createContext, useContext, useState, useCallback } from 'react'
import { Toast, ToastComponent } from '../components/ui/Toast'

interface ToastContextType {
  toasts: Toast[]
  addToast: (toast: Omit<Toast, 'id'>) => string
  removeToast: (id: string) => void
  clearAll: () => void
  // Convenience methods
  success: (title: string, description?: string, options?: Partial<Toast>) => string
  error: (title: string, description?: string, options?: Partial<Toast>) => string
  warning: (title: string, description?: string, options?: Partial<Toast>) => string
  info: (title: string, description?: string, options?: Partial<Toast>) => string
}

const ToastContext = createContext<ToastContextType | undefined>(undefined)

export function useToast() {
  const context = useContext(ToastContext)
  if (!context) {
    throw new Error('useToast must be used within a ToastProvider')
  }
  return context
}

interface ToastProviderProps {
  children: React.ReactNode
  maxToasts?: number
}

export function ToastProvider({ children, maxToasts = 5 }: ToastProviderProps) {
  const [toasts, setToasts] = useState<Toast[]>([])

  const generateId = useCallback(() => {
    return `toast-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`
  }, [])

  const addToast = useCallback((toastData: Omit<Toast, 'id'>) => {
    const id = generateId()
    const newToast: Toast = {
      id,
      duration: 5000,
      persistent: false,
      ...toastData
    }

    setToasts(prev => {
      // Remove oldest toasts if we exceed maxToasts
      const updatedToasts = [...prev, newToast]
      if (updatedToasts.length > maxToasts) {
        return updatedToasts.slice(-maxToasts)
      }
      return updatedToasts
    })

    return id
  }, [generateId, maxToasts])

  const removeToast = useCallback((id: string) => {
    setToasts(prev => prev.filter(toast => toast.id !== id))
  }, [])

  const clearAll = useCallback(() => {
    setToasts([])
  }, [])

  // Convenience methods
  const success = useCallback((title: string, description?: string, options?: Partial<Toast>) => {
    return addToast({
      type: 'success',
      title,
      description,
      duration: 4000,
      ...options
    })
  }, [addToast])

  const error = useCallback((title: string, description?: string, options?: Partial<Toast>) => {
    return addToast({
      type: 'error',
      title,
      description,
      persistent: true, // Errors should be persistent by default
      ...options
    })
  }, [addToast])

  const warning = useCallback((title: string, description?: string, options?: Partial<Toast>) => {
    return addToast({
      type: 'warning',
      title,
      description,
      duration: 6000,
      ...options
    })
  }, [addToast])

  const info = useCallback((title: string, description?: string, options?: Partial<Toast>) => {
    return addToast({
      type: 'info',
      title,
      description,
      duration: 5000,
      ...options
    })
  }, [addToast])

  const value: ToastContextType = {
    toasts,
    addToast,
    removeToast,
    clearAll,
    success,
    error,
    warning,
    info
  }

  return (
    <ToastContext.Provider value={value}>
      {children}
      {/* Toast Container */}
      <div
        className="fixed top-4 right-4 z-50 flex flex-col gap-2 pointer-events-none"
        aria-live="polite"
        aria-label="Notifications"
      >
        {toasts.map(toast => (
          <div key={toast.id} className="pointer-events-auto">
            <ToastComponent
              toast={toast}
              onDismiss={removeToast}
            />
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  )
}