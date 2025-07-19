import React, { useState, useEffect, useRef } from 'react'
import { Button, ButtonProps } from './button'
import { cn } from '../../lib/utils'

interface ConfirmButtonProps extends Omit<ButtonProps, 'onClick'> {
  onConfirm: () => void | Promise<void>
  confirmText?: string
  confirmDuration?: number // milliseconds
  children: React.ReactNode
}

export function ConfirmButton({
  onConfirm,
  confirmText = 'Click to confirm',
  confirmDuration = 3000,
  children,
  className,
  variant = 'outline',
  disabled,
  ...props
}: ConfirmButtonProps) {
  const [isConfirming, setIsConfirming] = useState(false)
  const [countdown, setCountdown] = useState(0)
  const [isExecuting, setIsExecuting] = useState(false)
  const timeoutRef = useRef<NodeJS.Timeout | null>(null)
  const intervalRef = useRef<NodeJS.Timeout | null>(null)
  const buttonRef = useRef<HTMLButtonElement>(null)

  // Clean up timers on unmount
  useEffect(() => {
    return () => {
      if (timeoutRef.current) clearTimeout(timeoutRef.current)
      if (intervalRef.current) clearInterval(intervalRef.current)
    }
  }, [])

  // Handle click outside to cancel
  useEffect(() => {
    if (!isConfirming) return

    const handleClickOutside = (event: MouseEvent) => {
      if (buttonRef.current && !buttonRef.current.contains(event.target as Node)) {
        handleCancel()
      }
    }

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        handleCancel()
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    document.addEventListener('keydown', handleEscape)

    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
      document.removeEventListener('keydown', handleEscape)
    }
  }, [isConfirming])

  const handleCancel = () => {
    setIsConfirming(false)
    setCountdown(0)
    if (timeoutRef.current) clearTimeout(timeoutRef.current)
    if (intervalRef.current) clearInterval(intervalRef.current)
  }

  const handleConfirm = async () => {
    if (isExecuting || disabled) return

    if (!isConfirming) {
      // First click - enter confirmation mode
      setIsConfirming(true)
      setCountdown(Math.ceil(confirmDuration / 1000))

      // Start countdown
      intervalRef.current = setInterval(() => {
        setCountdown(prev => {
          if (prev <= 1) {
            if (intervalRef.current) clearInterval(intervalRef.current)
            return 0
          }
          return prev - 1
        })
      }, 1000)

      // Auto-confirm after duration
      timeoutRef.current = setTimeout(() => {
        executeAction()
      }, confirmDuration)
    } else {
      // Second click - immediate confirmation
      if (timeoutRef.current) clearTimeout(timeoutRef.current)
      if (intervalRef.current) clearInterval(intervalRef.current)
      executeAction()
    }
  }

  const executeAction = async () => {
    setIsExecuting(true)
    try {
      await onConfirm()
    } finally {
      setIsExecuting(false)
      setIsConfirming(false)
      setCountdown(0)
    }
  }

  const displayText = isConfirming 
    ? countdown > 0 
      ? `${confirmText} (${countdown}s)`
      : confirmText
    : children

  return (
    <Button
      ref={buttonRef}
      className={cn(
        'transition-all duration-200',
        isConfirming && 'min-w-[140px]',
        className
      )}
      variant={isConfirming ? 'destructive' : variant}
      onClick={handleConfirm}
      disabled={disabled || isExecuting}
      {...props}
    >
      <span className={cn(
        'inline-flex items-center gap-2',
        isConfirming && 'animate-pulse'
      )}>
        {displayText}
      </span>
    </Button>
  )
}