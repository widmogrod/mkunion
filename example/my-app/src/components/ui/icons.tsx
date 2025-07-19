import React from 'react'
import { 
  Calendar, 
  CheckCircle, 
  XCircle, 
  Clock, 
  Loader2, 
  PauseCircle,
  PlayCircle,
  AlertCircle,
  Info,
  ChevronRight,
  History,
  Filter,
  Download,
  RefreshCw,
  Settings,
  Search,
  Plus,
  Trash2,
  Edit,
  Eye,
  EyeOff,
  Moon,
  Sun,
  Activity,
  type LucideIcon
} from 'lucide-react'

// Workflow Status Icons
export const StatusIcons = {
  done: CheckCircle,
  error: XCircle,
  running: Loader2,
  scheduled: Clock,
  paused: PauseCircle,
  active: PlayCircle,
  warning: AlertCircle,
  info: Info
} as const

// Status Icon Colors (for consistency with badges)
export const StatusColors = {
  done: 'text-green-600 dark:text-green-400',
  error: 'text-red-600 dark:text-red-400',
  running: 'text-blue-600 dark:text-blue-400',
  scheduled: 'text-yellow-600 dark:text-yellow-400',
  paused: 'text-gray-600 dark:text-gray-400',
  active: 'text-green-600 dark:text-green-400',
  warning: 'text-orange-600 dark:text-orange-400',
  info: 'text-blue-600 dark:text-blue-400'
} as const

// Action Icons
export const ActionIcons = {
  view: Eye,
  hide: EyeOff,
  edit: Edit,
  delete: Trash2,
  add: Plus,
  refresh: RefreshCw,
  download: Download,
  filter: Filter,
  search: Search,
  settings: Settings,
  history: History,
  next: ChevronRight
} as const

// Theme Icons
export const ThemeIcons = {
  light: Sun,
  dark: Moon
} as const

// Feature Icons
export const FeatureIcons = {
  calendar: Calendar,
  activity: Activity
} as const

// Icon Component with consistent sizing and styling
interface IconProps {
  icon: LucideIcon
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl'
  className?: string
  spin?: boolean
}

const sizeMap = {
  xs: 'h-3 w-3',
  sm: 'h-4 w-4',
  md: 'h-5 w-5',
  lg: 'h-6 w-6',
  xl: 'h-8 w-8'
} as const

export function Icon({ icon: IconComponent, size = 'sm', className = '', spin = false }: IconProps) {
  const sizeClass = sizeMap[size]
  const spinClass = spin ? 'animate-spin' : ''
  
  return <IconComponent className={`${sizeClass} ${spinClass} ${className}`} />
}

// Status Icon Component with automatic color
interface StatusIconProps extends Omit<IconProps, 'icon'> {
  status: keyof typeof StatusIcons
  colored?: boolean
}

export function StatusIcon({ status, colored = true, className = '', ...props }: StatusIconProps) {
  const IconComponent = StatusIcons[status]
  const colorClass = colored ? StatusColors[status] : ''
  
  return <Icon icon={IconComponent} className={`${colorClass} ${className}`} {...props} />
}

// Helper to get status icon props for consistent usage
export function getStatusIconProps(status: string): { icon: LucideIcon; color: string } {
  const normalizedStatus = status.toLowerCase() as keyof typeof StatusIcons
  
  return {
    icon: StatusIcons[normalizedStatus] || StatusIcons.info,
    color: StatusColors[normalizedStatus] || StatusColors.info
  }
}

// Export all icon types for use in other components
export type StatusType = keyof typeof StatusIcons
export type ActionType = keyof typeof ActionIcons
export type ThemeType = keyof typeof ThemeIcons
export type FeatureType = keyof typeof FeatureIcons