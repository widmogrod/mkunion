import React from 'react'
import { Sun, Moon, Monitor } from 'lucide-react'
import { Button } from '../ui/button'
import { useTheme } from './ThemeContext'
import { cn } from '../../utils/cn'

export function ThemeToggleCompact() {
  const { theme, setTheme, actualTheme } = useTheme()

  const cycleTheme = () => {
    switch (theme) {
      case 'light':
        setTheme('dark')
        break
      case 'dark':
        setTheme('system')
        break
      case 'system':
        setTheme('light')
        break
    }
  }

  const getIcon = () => {
    // Show actual theme icon when in system mode
    const displayTheme = theme === 'system' ? actualTheme : theme
    
    switch (displayTheme) {
      case 'light':
        return <Sun className="h-4 w-4" />
      case 'dark':
        return <Moon className="h-4 w-4" />
      default:
        return <Monitor className="h-4 w-4" />
    }
  }

  const getTooltip = () => {
    switch (theme) {
      case 'light':
        return 'Light mode'
      case 'dark':
        return 'Dark mode'
      case 'system':
        return `System theme (${actualTheme})`
    }
  }

  return (
    <Button
      variant="ghost"
      size="sm"
      onClick={cycleTheme}
      className={cn(
        "w-full h-10 p-0 flex items-center justify-center",
        "transition-all duration-200 group",
        "hover:bg-muted/50 active:scale-95"
      )}
      title={getTooltip()}
    >
      <div className="relative">
        {getIcon()}
        {theme === 'system' && (
          <div className="absolute -bottom-0.5 -right-0.5 w-1.5 h-1.5 bg-primary rounded-full" />
        )}
      </div>
    </Button>
  )
}