// Design System Constants
// This file contains design tokens and constants that should be used throughout the application
// instead of hardcoded values

// Color System
export const colors = {
  // Semantic colors for different states
  success: 'text-green-600 dark:text-green-400',
  error: 'text-red-600 dark:text-red-400',
  warning: 'text-yellow-600 dark:text-yellow-400',
  info: 'text-blue-600 dark:text-blue-400',
  
  // Background colors
  successBg: 'bg-green-100 dark:bg-green-900/20',
  errorBg: 'bg-red-100 dark:bg-red-900/20',
  warningBg: 'bg-yellow-100 dark:bg-yellow-900/20',
  infoBg: 'bg-blue-100 dark:bg-blue-900/20',
  
  // Text colors for different content types
  primary: 'text-foreground',
  secondary: 'text-muted-foreground',
  disabled: 'text-muted-foreground/50',
  
  // Interactive elements
  link: 'text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300',
  linkUnderline: 'underline underline-offset-2',
  
  // Specific use cases
  codeString: 'text-green-600 dark:text-green-400',
  codeNumber: 'text-blue-600 dark:text-blue-400',
  codeBoolean: 'text-purple-600 dark:text-purple-400',
} as const

// Icon sizes
export const iconSizes = {
  xs: 'h-3 w-3',
  sm: 'h-4 w-4',
  md: 'h-5 w-5',
  lg: 'h-6 w-6',
  xl: 'h-8 w-8',
} as const

// Spacing
export const spacing = {
  xs: '0.5rem',
  sm: '1rem',
  md: '1.5rem',
  lg: '2rem',
  xl: '3rem',
} as const

// Border radius
export const borderRadius = {
  sm: 'rounded-sm',
  md: 'rounded-md',
  lg: 'rounded-lg',
  full: 'rounded-full',
} as const

// Transitions
export const transitions = {
  fast: 'transition-all duration-150 ease-out',
  normal: 'transition-all duration-200 ease-out',
  slow: 'transition-all duration-300 ease-out',
} as const

// Shadows
export const shadows = {
  sm: 'shadow-sm',
  md: 'shadow-md',
  lg: 'shadow-lg',
  xl: 'shadow-xl',
} as const