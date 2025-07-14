// Design System Constants
// This file ensures consistency across all UI components

// Animation durations
export const TRANSITIONS = {
  fast: 'transition-all duration-150',
  normal: 'transition-all duration-200',
  slow: 'transition-all duration-300'
} as const

// Spacing scale
export const SPACING = {
  xs: 'gap-1',
  sm: 'gap-2',
  md: 'gap-3',
  lg: 'gap-4',
  xl: 'gap-6'
} as const

// Border radius
export const RADIUS = {
  sm: 'rounded-sm',
  md: 'rounded-md',
  lg: 'rounded-lg',
  xl: 'rounded-xl',
  full: 'rounded-full'
} as const

// Shadow effects
export const SHADOWS = {
  sm: 'shadow-sm',
  md: 'shadow-md',
  lg: 'shadow-lg',
  xl: 'shadow-xl'
} as const

// Status colors (matching Tailwind classes)
export const STATUS_COLORS = {
  success: {
    bg: 'bg-green-50 dark:bg-green-900/20',
    border: 'border-green-200 dark:border-green-800',
    text: 'text-green-700 dark:text-green-300',
    icon: 'text-green-600 dark:text-green-400'
  },
  error: {
    bg: 'bg-red-50 dark:bg-red-900/20',
    border: 'border-red-200 dark:border-red-800',
    text: 'text-red-700 dark:text-red-300',
    icon: 'text-red-600 dark:text-red-400'
  },
  warning: {
    bg: 'bg-yellow-50 dark:bg-yellow-900/20',
    border: 'border-yellow-200 dark:border-yellow-800',
    text: 'text-yellow-700 dark:text-yellow-300',
    icon: 'text-yellow-600 dark:text-yellow-400'
  },
  info: {
    bg: 'bg-blue-50 dark:bg-blue-900/20',
    border: 'border-blue-200 dark:border-blue-800',
    text: 'text-blue-700 dark:text-blue-300',
    icon: 'text-blue-600 dark:text-blue-400'
  },
  neutral: {
    bg: 'bg-gray-50 dark:bg-gray-900/20',
    border: 'border-gray-200 dark:border-gray-800',
    text: 'text-gray-700 dark:text-gray-300',
    icon: 'text-gray-600 dark:text-gray-400'
  }
} as const

// Text sizes
export const TEXT_SIZES = {
  xs: 'text-xs',
  sm: 'text-sm',
  base: 'text-base',
  lg: 'text-lg',
  xl: 'text-xl',
  '2xl': 'text-2xl'
} as const

// Component size presets
export const COMPONENT_SIZES = {
  xs: {
    padding: 'px-2 py-1',
    text: TEXT_SIZES.xs,
    height: 'h-6'
  },
  sm: {
    padding: 'px-3 py-1.5',
    text: TEXT_SIZES.sm,
    height: 'h-8'
  },
  md: {
    padding: 'px-4 py-2',
    text: TEXT_SIZES.base,
    height: 'h-10'
  },
  lg: {
    padding: 'px-6 py-3',
    text: TEXT_SIZES.lg,
    height: 'h-12'
  }
} as const

// Hover states
export const HOVER_STATES = {
  subtle: 'hover:bg-muted/50',
  normal: 'hover:bg-muted',
  strong: 'hover:bg-muted hover:shadow-sm'
} as const

// Focus states
export const FOCUS_STATES = {
  ring: 'focus:outline-none focus:ring-2 focus:ring-primary/20',
  outline: 'focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary'
} as const