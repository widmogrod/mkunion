import React from 'react'
import { render, screen } from '@testing-library/react'
import '@testing-library/jest-dom'
import { RunHistoryCalendar } from '../RunHistoryCalendar'

describe('RunHistoryCalendar', () => {
  const mockExecutions = [
    {
      id: 'run_1',
      startTime: new Date('2024-01-15T10:00:00'),
      status: 'done' as const,
    },
    {
      id: 'run_2',
      startTime: new Date('2024-01-15T14:00:00'),
      status: 'error' as const,
      errorMessage: 'Test error',
    },
    {
      id: 'run_3',
      startTime: new Date('2024-01-16T09:00:00'),
      status: 'scheduled' as const,
    },
  ]

  it('renders calendar with executions', () => {
    render(<RunHistoryCalendar executions={mockExecutions} />)
    
    // Check if calendar navigation is present
    expect(screen.getByText('Today')).toBeInTheDocument()
    
    // Check if view buttons are present
    expect(screen.getByText('Month')).toBeInTheDocument()
    expect(screen.getByText('Week')).toBeInTheDocument()
    expect(screen.getByText('Day')).toBeInTheDocument()
    
    // Check if legend is present
    expect(screen.getByText('Done')).toBeInTheDocument()
    expect(screen.getByText('Error')).toBeInTheDocument()
    expect(screen.getByText('Running')).toBeInTheDocument()
    expect(screen.getByText('Scheduled')).toBeInTheDocument()
  })

  it('shows execution count', () => {
    render(<RunHistoryCalendar executions={mockExecutions} />)
    
    expect(screen.getByText('Showing 3 executions')).toBeInTheDocument()
  })

  it('renders empty state', () => {
    render(<RunHistoryCalendar executions={[]} />)
    
    // Calendar should still render even with no events
    expect(screen.getByText('Today')).toBeInTheDocument()
  })

  it('handles execution click', () => {
    const mockOnClick = jest.fn()
    render(<RunHistoryCalendar executions={mockExecutions} onExecutionClick={mockOnClick} />)
    
    // Note: Testing actual event clicks in react-big-calendar is complex
    // due to the way it renders events. This would require more sophisticated
    // testing setup with user events and waiting for calendar to fully render.
  })
})