import { useState, useEffect } from 'react'
import { useWorkflowApi } from './use-workflow-api'
import * as workflow from '../workflow/github_com_widmogrod_mkunion_x_workflow'

export interface Schedule {
  id: string
  parentRunId: string
  flowName: string
  cronExpression: string
  status: 'active' | 'paused'
  nextRun?: Date
  color?: string // For calendar display
}

// Color palette for schedules
const SCHEDULE_COLORS = [
  '#3b82f6', // blue
  '#10b981', // emerald
  '#f59e0b', // amber
  '#8b5cf6', // violet
  '#ec4899', // pink
  '#14b8a6', // teal
  '#f97316', // orange
  '#6366f1', // indigo
  '#84cc16', // lime
  '#06b6d4', // cyan
]

export function useAllSchedules() {
  const [schedules, setSchedules] = useState<Schedule[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [, setFlowNameCache] = useState<Map<string, string>>(new Map())
  
  const { listStates, listFlows } = useWorkflowApi()
  
  useEffect(() => {
    loadAllSchedules()
  }, [])
  
  const loadAllSchedules = async () => {
    try {
      setLoading(true)
      setError(null)
      
      // First, load all flows to build a name cache
      const flowsResponse = await listFlows({ limit: 1000 })
      const flowMap = new Map<string, string>()
      
      if (flowsResponse.Items) {
        flowsResponse.Items.forEach(flowRecord => {
          if (flowRecord.ID && flowRecord.Data?.Name) {
            flowMap.set(flowRecord.ID, flowRecord.Data.Name)
          }
        })
      }
      
      setFlowNameCache(flowMap)
      
      // Query for both active (Scheduled) and paused (ScheduleStopped) schedules
      const response = await listStates({
        where: {
          Predicate: {
            "$type": "predicate.Or",
            "predicate.Or": {
              L: [
                {
                  "$type": "predicate.Compare",
                  "predicate.Compare": {
                    Location: 'Data["$type"]',
                    Operation: "==",
                    BindValue: {
                      "$type": "predicate.Literal",
                      "predicate.Literal": {
                        Value: {
                          "$type": "schema.String",
                          "schema.String": "workflow.Scheduled"
                        }
                      }
                    }
                  }
                },
                {
                  "$type": "predicate.Compare",
                  "predicate.Compare": {
                    Location: 'Data["$type"]',
                    Operation: "==",
                    BindValue: {
                      "$type": "predicate.Literal",
                      "predicate.Literal": {
                        Value: {
                          "$type": "schema.String",
                          "schema.String": "workflow.ScheduleStopped"
                        }
                      }
                    }
                  }
                }
              ]
            }
          }
        },
        limit: 1000 // Should be enough for most use cases
      })
      
      if (response.Items) {
        const processedSchedules = response.Items
          .map((item, index) => {
            const state = item.Data
            if (!state) return null // Skip if no data
            
            const stateType = state.$type
            
            // Extract schedule data based on state type
            let baseState: any = null
            let nextRun: Date | undefined
            
            if (stateType === 'workflow.Scheduled') {
              const scheduledState = state['workflow.Scheduled'] as workflow.Scheduled
              baseState = scheduledState?.BaseState
              if (scheduledState?.ExpectedRunTimestamp) {
                nextRun = new Date(scheduledState.ExpectedRunTimestamp * 1000)
              }
            } else if (stateType === 'workflow.ScheduleStopped') {
              const stoppedState = state['workflow.ScheduleStopped'] as workflow.ScheduleStopped
              baseState = stoppedState?.BaseState
            }
            
            const scheduleRun = baseState?.RunOption?.['workflow.ScheduleRun']
            
            // Extract flow name - handle both Flow and FlowRef types
            let flowName = 'Unknown'
            if (baseState?.Flow) {
              if (baseState.Flow.$type === 'workflow.Flow' && baseState.Flow['workflow.Flow']) {
                flowName = baseState.Flow['workflow.Flow'].Name || 'Unknown'
              } else if (baseState.Flow.$type === 'workflow.FlowRef' && baseState.Flow['workflow.FlowRef']) {
                const flowID = baseState.Flow['workflow.FlowRef'].FlowID
                if (flowID) {
                  flowName = flowMap.get(flowID) || `FlowRef:${flowID}`
                }
              }
            }
            
            const parentRunId = scheduleRun?.ParentRunID || ''
            return {
              id: parentRunId || item.ID, // Use parentRunId as the primary ID
              parentRunId,
              flowName,
              cronExpression: scheduleRun?.Interval || '',
              status: stateType === 'workflow.Scheduled' ? 'active' : 'paused',
              nextRun,
              color: SCHEDULE_COLORS[index % SCHEDULE_COLORS.length]
            } as Schedule
          })
          .filter((schedule): schedule is Schedule => schedule !== null)
        
        // Remove duplicates by parentRunId (keep the first occurrence)
        const uniqueSchedules = processedSchedules.filter((schedule, index, array) => 
          array.findIndex(s => s.parentRunId === schedule.parentRunId) === index
        )
        
        // Sort by flow name for consistent ordering
        uniqueSchedules.sort((a, b) => a.flowName.localeCompare(b.flowName))
        
        setSchedules(uniqueSchedules)
      }
    } catch (err) {
      console.error('Failed to load schedules:', err)
      setError('Failed to load schedules')
    } finally {
      setLoading(false)
    }
  }
  
  return {
    schedules,
    loading,
    error,
    refetch: loadAllSchedules
  }
}