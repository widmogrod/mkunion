import React, { createContext, useContext, useEffect, useState } from 'react'

type Theme = 'light' | 'dark' | 'system'

interface ThemeContextType {
  theme: Theme
  actualTheme: 'light' | 'dark' // The actual theme being applied
  setTheme: (theme: Theme) => void
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined)

export function useTheme() {
  const context = useContext(ThemeContext)
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider')
  }
  return context
}

interface ThemeProviderProps {
  children: React.ReactNode
}

export function ThemeProvider({ children }: ThemeProviderProps) {
  const [theme, setTheme] = useState<Theme>('system')
  const [actualTheme, setActualTheme] = useState<'light' | 'dark'>('light')

  // Get system theme preference
  const getSystemTheme = (): 'light' | 'dark' => {
    if (typeof window !== 'undefined') {
      return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
    }
    return 'light'
  }

  // Update actual theme based on current theme setting
  const updateActualTheme = (currentTheme: Theme) => {
    if (currentTheme === 'system') {
      setActualTheme(getSystemTheme())
    } else {
      setActualTheme(currentTheme)
    }
  }

  // Load theme from localStorage on mount with proper validation
  useEffect(() => {
    try {
      const savedTheme = localStorage.getItem('theme')
      // Strict validation to ensure only allowed values
      const validThemes: Theme[] = ['light', 'dark', 'system']
      if (savedTheme && validThemes.includes(savedTheme as Theme)) {
        const validatedTheme = savedTheme as Theme
        setTheme(validatedTheme)
        updateActualTheme(validatedTheme)
      } else {
        // Default to system if invalid or missing
        setTheme('system')
        updateActualTheme('system')
      }
    } catch (error) {
      // If localStorage access fails, fall back to system theme
      setTheme('system')
      updateActualTheme('system')
    }
  }, [])

  // Listen for system theme changes
  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
    
    const handleChange = () => {
      if (theme === 'system') {
        setActualTheme(getSystemTheme())
      }
    }

    mediaQuery.addEventListener('change', handleChange)
    return () => mediaQuery.removeEventListener('change', handleChange)
  }, [theme])

  // Apply theme to document
  useEffect(() => {
    const root = document.documentElement
    
    // Remove previous theme classes
    root.classList.remove('light', 'dark')
    
    // Add current theme class
    root.classList.add(actualTheme)
    
    // Update data attribute for CSS
    root.setAttribute('data-theme', actualTheme)
  }, [actualTheme])

  const handleSetTheme = (newTheme: Theme) => {
    // Validate theme before setting
    const validThemes: Theme[] = ['light', 'dark', 'system']
    if (!validThemes.includes(newTheme)) {
      console.error(`Invalid theme: ${newTheme}`)
      return
    }
    
    setTheme(newTheme)
    try {
      localStorage.setItem('theme', newTheme)
    } catch (error) {
      // Handle localStorage errors gracefully
      console.error('Failed to save theme preference:', error)
    }
    updateActualTheme(newTheme)
  }

  return (
    <ThemeContext.Provider 
      value={{ 
        theme, 
        actualTheme, 
        setTheme: handleSetTheme 
      }}
    >
      {children}
    </ThemeContext.Provider>
  )
}