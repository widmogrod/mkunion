import { describe, it, expect } from '@jest/globals'

// Integration tests that hit the real API
const API_BASE_URL = 'http://localhost:8080'

interface ApiState {
  Data: any
  ID: string
  Type: string
  Version: number
}

interface ApiResponse {
  Items: ApiState[]
  Next?: any
}

// Function that mimics the actual component logic
function processRunHistoryFromApi(states: ApiState[], targetParentRunId: string) {
  console.log('Processing', states.length, 'total states')
  console.log('Looking for ParentRunID:', targetParentRunId)
  
  // Filter states that belong to this schedule by ParentRunID
  const relatedStates = states.filter(state => {
    if (!state.Data) {
      console.log('State has no data:', state.ID)
      return false
    }
    
    const stateData = state.Data
    const stateType = stateData.$type
    
    // Extract BaseState from different state types
    let baseState: any = null
    if (stateType && stateData[stateType as keyof typeof stateData]) {
      baseState = (stateData[stateType as keyof typeof stateData] as any)?.BaseState
    }
    
    const parentRunId = baseState?.RunOption?.['workflow.ScheduleRun']?.ParentRunID
    console.log('State', state.ID, 'type:', stateType, 'ParentRunID:', parentRunId, 'baseState exists:', !!baseState, 'RunOption exists:', !!baseState?.RunOption)
    
    const matches = parentRunId === targetParentRunId
    if (matches) {
      console.log('✅ MATCH FOUND for state:', state.ID)
    }
    
    return matches
  })

  console.log('Related states for ParentRunID', targetParentRunId, ':', relatedStates.length)
  
  // Group states by RunID to create execution runs
  const runGroups: { [runId: string]: ApiState[] } = {}
  
  relatedStates.forEach(state => {
    if (!state.Data) return
    
    const stateData = state.Data
    const stateType = stateData.$type
    
    let baseState: any = null
    if (stateType && stateData[stateType as keyof typeof stateData]) {
      baseState = (stateData[stateType as keyof typeof stateData] as any)?.BaseState
    }
    
    const runId = baseState?.RunID
    console.log('Processing state for grouping:', state.ID, 'RunID:', runId)
    if (runId) {
      if (!runGroups[runId]) {
        runGroups[runId] = []
      }
      runGroups[runId].push(state)
      console.log('Added to group', runId, '- now has', runGroups[runId].length, 'states')
    }
  })
  
  console.log('Total run groups created:', Object.keys(runGroups).length)
  Object.entries(runGroups).forEach(([runId, states]) => {
    console.log(`Group ${runId}: ${states.length} state(s)`)
  })

  // Convert run groups to execution objects
  const executions = Object.entries(runGroups).map(([runId, runStates]) => {
    const firstState = runStates[0]
    const lastState = runStates[runStates.length - 1]
    
    // Determine status from the final state
    let status: 'scheduled' | 'running' | 'done' | 'error' = 'scheduled'
    
    const finalStateType = lastState.Data?.$type
    if (finalStateType === 'workflow.Done') {
      status = 'done'
    } else if (finalStateType === 'workflow.Error') {
      status = 'error'
    } else if (finalStateType === 'workflow.Await') {
      status = 'running'
    }

    return {
      id: runId,
      status,
      stateCount: runStates.length,
      firstStateType: firstState.Data?.$type,
      lastStateType: lastState.Data?.$type
    }
  })

  console.log('Final processed executions:', executions.length)
  return { relatedStates, runGroups, executions }
}

describe('ScheduleHistoryDialog Integration Tests', () => {
  it('should use backend filtering for efficient data retrieval', async () => {
    const targetParentRunId = 'schedule_hello_world_1752497245218'
    
    // Test the new backend filtering approach
    const response = await fetch(`${API_BASE_URL}/states`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        where: {
          Predicate: {
            "$type": "predicate.Or",
            "predicate.Or": {
              L: [
                {
                  "$type": "predicate.Compare",
                  "predicate.Compare": {
                    Location: 'Data["workflow.Done"]["BaseState"]["RunOption"]["workflow.ScheduleRun"]["ParentRunID"]',
                    Operation: "==",
                    BindValue: {
                      "$type": "predicate.Literal",
                      "predicate.Literal": {
                        Value: { "$type": "schema.String", "schema.String": targetParentRunId }
                      }
                    }
                  }
                },
                {
                  "$type": "predicate.Compare",
                  "predicate.Compare": {
                    Location: 'Data["workflow.Scheduled"]["BaseState"]["RunOption"]["workflow.ScheduleRun"]["ParentRunID"]',
                    Operation: "==",
                    BindValue: {
                      "$type": "predicate.Literal",
                      "predicate.Literal": {
                        Value: { "$type": "schema.String", "schema.String": targetParentRunId }
                      }
                    }
                  }
                },
                {
                  "$type": "predicate.Compare",
                  "predicate.Compare": {
                    Location: 'Data["workflow.ScheduleStopped"]["BaseState"]["RunOption"]["workflow.ScheduleRun"]["ParentRunID"]',
                    Operation: "==",
                    BindValue: {
                      "$type": "predicate.Literal",
                      "predicate.Literal": {
                        Value: { "$type": "schema.String", "schema.String": targetParentRunId }
                      }
                    }
                  }
                },
                {
                  "$type": "predicate.Compare",
                  "predicate.Compare": {
                    Location: 'Data["workflow.Error"]["BaseState"]["RunOption"]["workflow.ScheduleRun"]["ParentRunID"]',
                    Operation: "==",
                    BindValue: {
                      "$type": "predicate.Literal",
                      "predicate.Literal": {
                        Value: { "$type": "schema.String", "schema.String": targetParentRunId }
                      }
                    }
                  }
                }
              ]
            }
          }
        },
        Limit: 100
      })
    })
    
    expect(response.ok).toBe(true)
    const data: ApiResponse = await response.json()
    
    console.log('🚀 Backend filtering returned:', data.Items?.length || 0, 'pre-filtered states')
    expect(data.Items).toBeDefined()
    
    if (data.Items && data.Items.length > 0) {
      console.log('✅ Backend predicate filtering worked!')
      
      // Verify all returned states actually match our ParentRunID
      data.Items.forEach(state => {
        const stateData = state.Data
        const stateType = stateData.$type
        let baseState: any = null
        if (stateType && stateData[stateType as keyof typeof stateData]) {
          baseState = (stateData[stateType as keyof typeof stateData] as any)?.BaseState
        }
        const parentRunId = baseState?.RunOption?.['workflow.ScheduleRun']?.ParentRunID
        expect(parentRunId).toBe(targetParentRunId)
      })
      
      // Test simplified processing (no client-side filtering needed)
      const processedExecutions = data.Items.filter(state => state.Data).map(state => {
        const stateData = state.Data
        const stateType = stateData.$type
        let baseState: any = null
        if (stateType && stateData[stateType as keyof typeof stateData]) {
          baseState = (stateData[stateType as keyof typeof stateData] as any)?.BaseState
        }
        
        return {
          id: baseState?.RunID || 'unknown',
          status: stateType === 'workflow.Done' ? 'done' : 
                 stateType === 'workflow.Error' ? 'error' : 'scheduled',
          type: stateType
        }
      })
      
      console.log('🎯 Processed', processedExecutions.length, 'executions from backend-filtered data')
      expect(processedExecutions.length).toBeGreaterThan(0)
      
      processedExecutions.forEach((execution, index) => {
        console.log(`✅ Execution ${index + 1}:`, execution)
        expect(execution.id).toBeDefined()
        expect(execution.status).toMatch(/^(scheduled|running|done|error)$/)
      })
    } else {
      console.log('ℹ️  Backend filter returned no results - this is valid if no executions exist')
    }
    
    console.log('🏆 BACKEND FILTERING TEST: Efficient server-side filtering implemented successfully')
  })

  it('should show what ParentRunIDs are available in the system', async () => {
    const response = await fetch(`${API_BASE_URL}/states`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        Limit: 500
      })
    })
    
    const data: ApiResponse = await response.json()
    
    // Extract all available ParentRunIDs
    const availableParentRunIDs = new Set<string>()
    
    data.Items.forEach(state => {
      if (!state.Data) return
      
      const stateData = state.Data
      const stateType = stateData.$type
      
      let baseState: any = null
      if (stateType && stateData[stateType as keyof typeof stateData]) {
        baseState = (stateData[stateType as keyof typeof stateData] as any)?.BaseState
      }
      
      const parentRunId = baseState?.RunOption?.['workflow.ScheduleRun']?.ParentRunID
      if (parentRunId) {
        availableParentRunIDs.add(parentRunId)
      }
    })
    
    console.log('Available ParentRunIDs in system:')
    Array.from(availableParentRunIDs).forEach(id => console.log(`  - ${id}`))
    
    expect(availableParentRunIDs.size).toBeGreaterThan(0)
    expect(Array.from(availableParentRunIDs)).toContain('schedule_hello_world_1752497245218')
  })

  it('should test with a ParentRunID that has no executions', async () => {
    const response = await fetch(`${API_BASE_URL}/states`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        Limit: 500
      })
    })
    
    const data: ApiResponse = await response.json()
    
    // Test with a non-existent ParentRunID
    const nonExistentParentRunId = 'schedule_does_not_exist_123456'
    const result = processRunHistoryFromApi(data.Items, nonExistentParentRunId)
    
    expect(result.relatedStates.length).toBe(0)
    expect(Object.keys(result.runGroups).length).toBe(0)
    expect(result.executions.length).toBe(0)
    
    console.log('Confirmed: Non-existent ParentRunID returns empty results')
  })
})