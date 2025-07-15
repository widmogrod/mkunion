import { useEffect } from 'react'
import { useNavigationWithContext } from './useNavigation'

export function useKeyboardShortcuts() {
  const { 
    navigateToWorkflows, 
    navigateToExecutions, 
    navigateToSchedules, 
    navigateToCalendar,
    goBack 
  } = useNavigationWithContext()

  useEffect(() => {
    const handleKeyPress = (event: KeyboardEvent) => {
      // Check if user is typing in an input field
      const activeElement = document.activeElement
      const isTyping = activeElement?.tagName === 'INPUT' || 
                      activeElement?.tagName === 'TEXTAREA' ||
                      (activeElement as HTMLElement)?.contentEditable === 'true'
      
      if (isTyping) return

      // Command/Ctrl key combinations
      if (event.metaKey || event.ctrlKey) {
        switch (event.key) {
          case '1':
            event.preventDefault()
            navigateToWorkflows()
            break
          case '2':
            event.preventDefault()
            navigateToExecutions()
            break
          case '3':
            event.preventDefault()
            navigateToSchedules()
            break
          case '4':
            event.preventDefault()
            navigateToCalendar()
            break
          case 'b':
          case 'B':
            event.preventDefault()
            goBack()
            break
        }
      }
      
      // Single key shortcuts (when not typing)
      if (!event.metaKey && !event.ctrlKey && !event.altKey) {
        switch (event.key) {
          case 'w':
          case 'W':
            navigateToWorkflows()
            break
          case 'e':
          case 'E':
            navigateToExecutions()
            break
          case 'h':
          case 'H':
            navigateToSchedules()
            break
          case 'c':
          case 'C':
            navigateToCalendar()
            break
          case '?':
            // Show keyboard shortcuts help
            showKeyboardShortcutsHelp()
            break
        }
      }
    }

    window.addEventListener('keydown', handleKeyPress)
    return () => window.removeEventListener('keydown', handleKeyPress)
  }, [navigateToWorkflows, navigateToExecutions, navigateToSchedules, navigateToCalendar, goBack])
}

function showKeyboardShortcutsHelp() {
  // Create a simple modal to show keyboard shortcuts
  const modal = document.createElement('div')
  modal.className = 'fixed inset-0 bg-black/50 flex items-center justify-center z-50'
  modal.innerHTML = `
    <div class="bg-background border rounded-lg p-6 max-w-md mx-4">
      <h2 class="text-lg font-bold mb-4">Keyboard Shortcuts</h2>
      <div class="space-y-2 text-sm">
        <div class="flex justify-between">
          <span class="text-muted-foreground">Go to Workflows</span>
          <kbd class="px-2 py-1 bg-muted rounded text-xs">W</kbd>
        </div>
        <div class="flex justify-between">
          <span class="text-muted-foreground">Go to Executions</span>
          <kbd class="px-2 py-1 bg-muted rounded text-xs">E</kbd>
        </div>
        <div class="flex justify-between">
          <span class="text-muted-foreground">Go to Schedules</span>
          <kbd class="px-2 py-1 bg-muted rounded text-xs">H</kbd>
        </div>
        <div class="flex justify-between">
          <span class="text-muted-foreground">Go to Calendar</span>
          <kbd class="px-2 py-1 bg-muted rounded text-xs">C</kbd>
        </div>
        <div class="flex justify-between">
          <span class="text-muted-foreground">Go Back</span>
          <kbd class="px-2 py-1 bg-muted rounded text-xs">⌘B</kbd>
        </div>
      </div>
      <button class="mt-4 px-4 py-2 bg-primary text-primary-foreground rounded text-sm w-full">
        Close (ESC)
      </button>
    </div>
  `
  
  const closeModal = () => modal.remove()
  
  modal.addEventListener('click', (e) => {
    if (e.target === modal) closeModal()
  })
  
  modal.querySelector('button')?.addEventListener('click', closeModal)
  
  const handleEscape = (e: KeyboardEvent) => {
    if (e.key === 'Escape') {
      closeModal()
      window.removeEventListener('keydown', handleEscape)
    }
  }
  
  window.addEventListener('keydown', handleEscape)
  document.body.appendChild(modal)
}