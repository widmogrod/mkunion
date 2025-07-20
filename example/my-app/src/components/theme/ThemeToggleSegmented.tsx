import React from 'react'
import { Sun, Moon, Monitor } from 'lucide-react'
import { useTheme } from './ThemeContext'
import { cn } from '../../utils/cn'

type Theme = 'light' | 'dark' | 'system'

interface ThemeOption {
  value: Theme
  icon: React.ReactNode
  label: string
}

export function ThemeToggleSegmented() {
  const { theme, setTheme } = useTheme()

  const options: ThemeOption[] = [
    {
      value: 'light',
      icon: <Sun className="h-4 w-4" />,
      label: 'Light'
    },
    {
      value: 'dark',
      icon: <Moon className="h-4 w-4" />,
      label: 'Dark'
    },
    {
      value: 'system',
      icon: <Monitor className="h-4 w-4" />,
      label: 'System'
    }
  ]

  const activeIndex = options.findIndex(opt => opt.value === theme)

  return (
    <div className="w-full">
      <div className="relative bg-muted/30 rounded-lg p-1">
        {/* Sliding indicator */}
        <div
          className={cn(
            "absolute top-1 h-[calc(100%-8px)] bg-background rounded-md shadow-sm",
            "transition-all duration-300 ease-out",
            "border border-border/50"
          )}
          style={{
            width: `${100 / options.length}%`,
            left: `${(activeIndex * 100) / options.length}%`
          }}
        />
        
        {/* Options */}
        <div className="relative flex items-center gap-1">
          {options.map((option) => (
            <button
              key={option.value}
              onClick={() => setTheme(option.value)}
              className={cn(
                "flex-1 flex items-center justify-center gap-2 px-3 py-1.5 rounded-md",
                "text-sm font-medium transition-all duration-200",
                "hover:text-foreground active:scale-95",
                theme === option.value
                  ? "text-foreground"
                  : "text-muted-foreground"
              )}
            >
              <span className={cn(
                "transition-transform duration-200",
                theme === option.value && "scale-110"
              )}>
                {option.icon}
              </span>
              <span className="hidden sm:inline">{option.label}</span>
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}