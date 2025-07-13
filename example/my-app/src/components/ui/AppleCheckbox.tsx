import React from 'react'
import { Check } from 'lucide-react'
import { cn } from '../../utils/cn'

interface AppleCheckboxProps {
  checked: boolean
  onChange: (checked: boolean) => void
  disabled?: boolean
  className?: string
}

export function AppleCheckbox({ checked, onChange, disabled = false, className }: AppleCheckboxProps) {
  return (
    <button
      type="button"
      role="checkbox"
      aria-checked={checked}
      disabled={disabled}
      onClick={() => !disabled && onChange(!checked)}
      className={cn(
        // Base styling with Apple-inspired dimensions
        "relative inline-flex items-center justify-center w-3.5 h-3.5 rounded-sm",
        "border transition-all duration-150 ease-out",
        "focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:ring-offset-1",
        
        // Unchecked state
        !checked && "bg-white border-gray-300 hover:border-gray-400 hover:bg-gray-50",
        
        // Checked state with Apple blue
        checked && "bg-blue-500 border-blue-500 hover:bg-blue-600 hover:border-blue-600",
        
        // Disabled state
        disabled && "opacity-50 cursor-not-allowed",
        !disabled && "cursor-pointer",
        
        className
      )}
    >
      {/* Checkmark with smooth scaling animation */}
      <Check 
        className={cn(
          "w-2.5 h-2.5 text-white transition-transform duration-150 ease-out",
          checked ? "scale-100 opacity-100" : "scale-0 opacity-0"
        )}
        strokeWidth={2.5}
      />
    </button>
  )
}