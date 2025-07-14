interface RunExecution {
  id: string
  startTime: Date
  endTime?: Date
  status: 'scheduled' | 'running' | 'done' | 'error'
  duration?: number
  errorMessage?: string
  inputData?: any
  outputData?: any
}

export function exportToCSV(executions: RunExecution[], scheduleName: string): void {
  // Prepare CSV headers
  const headers = ['Run ID', 'Status', 'Start Time', 'Duration', 'Error Message']
  
  // Convert executions to CSV rows
  const rows = executions.map(execution => [
    execution.id,
    execution.status,
    execution.startTime.toISOString(),
    execution.duration ? `${Math.round(execution.duration / 1000)}s` : 'N/A',
    execution.errorMessage || ''
  ])
  
  // Combine headers and rows
  const csvContent = [
    headers.join(','),
    ...rows.map(row => row.map(cell => `"${cell}"`).join(','))
  ].join('\n')
  
  // Create and download the file
  const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
  const link = document.createElement('a')
  const filename = `${scheduleName.replace(/\s+/g, '_')}_history_${new Date().toISOString().split('T')[0]}.csv`
  
  link.href = URL.createObjectURL(blob)
  link.download = filename
  link.click()
  
  // Clean up
  URL.revokeObjectURL(link.href)
}

export function exportToJSON(executions: RunExecution[], scheduleName: string): void {
  // Prepare JSON data with metadata
  const exportData = {
    schedule: scheduleName,
    exportDate: new Date().toISOString(),
    totalExecutions: executions.length,
    summary: {
      successful: executions.filter(e => e.status === 'done').length,
      failed: executions.filter(e => e.status === 'error').length,
      running: executions.filter(e => e.status === 'running').length,
      scheduled: executions.filter(e => e.status === 'scheduled').length
    },
    executions: executions.map(execution => ({
      id: execution.id,
      status: execution.status,
      startTime: execution.startTime.toISOString(),
      endTime: execution.endTime?.toISOString() || null,
      duration: execution.duration || null,
      errorMessage: execution.errorMessage || null,
      // Optionally include input/output data (can be large)
      inputData: execution.inputData || null,
      outputData: execution.outputData || null
    }))
  }
  
  // Create and download the file
  const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: 'application/json' })
  const link = document.createElement('a')
  const filename = `${scheduleName.replace(/\s+/g, '_')}_history_${new Date().toISOString().split('T')[0]}.json`
  
  link.href = URL.createObjectURL(blob)
  link.download = filename
  link.click()
  
  // Clean up
  URL.revokeObjectURL(link.href)
}