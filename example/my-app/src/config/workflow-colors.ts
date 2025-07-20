// Workflow State Color Configuration
// These colors are used consistently throughout the application for workflow states

export const WORKFLOW_STATE_COLORS = {
  // Success states - Green
  'workflow.Done': '#10b981', // emerald-500
  
  // Error states - Red
  'workflow.Error': '#ef4444', // red-500
  
  // Running/Active states - Blue
  'workflow.Await': '#3b82f6', // blue-500
  'workflow': '#3b82f6', // blue-500 (for workflow filters)
  
  // Scheduled states - Yellow
  'workflow.Scheduled': '#eab308', // yellow-600
  
  // Paused/Stopped states - Gray
  'workflow.ScheduleStopped': '#6b7280', // gray-500
  
  // Operation states - Purple
  'workflow.NextOperation': '#a855f7', // purple-500
} as const

// Helper function to get color with fallback
export function getWorkflowStateColor(stateType: string): string {
  return WORKFLOW_STATE_COLORS[stateType as keyof typeof WORKFLOW_STATE_COLORS] || '#6b7280'
}