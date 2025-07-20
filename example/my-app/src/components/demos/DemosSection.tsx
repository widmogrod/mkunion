import React from 'react'
import { HelloWorldDemo } from './HelloWorldDemo'
import { ImageGenerationDemo } from './ImageGenerationDemo'
import { ScheduledOperationsDemo } from './ScheduledOperationsDemo'
import { ChatDemo } from './ChatDemo'
import { useSidebar } from '../layout/SidebarContext'
import { Button } from '../ui/button'

export function DemosSection() {
  // Input state shared with ScheduledOperationsDemo
  const [input] = React.useState('Amigo')
  const { selectedDemo, setSelectedDemo } = useSidebar()

  const demos = [
    { id: 'hello', label: 'Hello World', component: <HelloWorldDemo /> },
    { id: 'image', label: 'Image Generation', component: <ImageGenerationDemo /> },
    { id: 'schedule', label: 'Scheduled Async', component: <ScheduledOperationsDemo input={input} /> },
    { id: 'chat', label: 'Chat Interface', component: <ChatDemo /> },
  ]

  // If no demo is selected, show all demos
  if (!selectedDemo) {
    return (
      <div className="space-y-4 max-w-full overflow-hidden">
        <div className="flex flex-wrap gap-2 mb-4">
          {demos.map(({ id, label }) => (
            <Button
              key={id}
              variant="outline"
              size="sm"
              onClick={() => setSelectedDemo(id as any)}
              className="text-xs"
            >
              {label}
            </Button>
          ))}
        </div>
        <div className="space-y-4 max-w-full overflow-hidden">
          <HelloWorldDemo />
          <ImageGenerationDemo />
          <ScheduledOperationsDemo input={input} />
          <ChatDemo />
        </div>
      </div>
    )
  }

  // Show selected demo
  const selectedDemoData = demos.find(demo => demo.id === selectedDemo)
  
  return (
    <div className="space-y-4 max-w-full overflow-hidden">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold truncate">{selectedDemoData?.label}</h3>
        <Button
          variant="outline"
          size="sm"
          onClick={() => setSelectedDemo(null)}
          className="flex-shrink-0"
        >
          Show All
        </Button>
      </div>
      <div className="max-w-full overflow-hidden">
        {selectedDemoData?.component}
      </div>
    </div>
  )
}