import { describe, it, expect } from '@jest/globals'

// Real API data structure for testing
const realApiStates = [
  {
    "Data": {
      "$type": "workflow.Done",
      "workflow.Done": {
        "BaseState": {
          "DefaultMaxRetries": 3,
          "ExprResult": {},
          "Flow": {
            "$type": "workflow.Flow",
            "workflow.Flow": {
              "Name": "hello_world"
            }
          },
          "RunID": "run_id:9181077119013508119",
          "RunOption": {
            "$type": "workflow.ScheduleRun",
            "workflow.ScheduleRun": {
              "Interval": "0 */5 * * * *",
              "ParentRunID": "schedule_hello_world_1752497245218"
            }
          },
          "StepID": "end1",
          "Variables": {
            "input": { "$type": "schema.String", "schema.String": "" },
            "res": { "$type": "schema.String", "schema.String": "hello " }
          }
        },
        "Result": { "$type": "schema.String", "schema.String": "hello " }
      }
    },
    "ID": "run_id:9181077119013508119",
    "Type": "process",
    "Version": 2
  },
  {
    "Data": {
      "$type": "workflow.Scheduled",
      "workflow.Scheduled": {
        "BaseState": {
          "DefaultMaxRetries": 3,
          "ExprResult": {},
          "Flow": {
            "$type": "workflow.Flow",
            "workflow.Flow": {
              "Name": "hello_world"
            }
          },
          "RunID": "run_id:7784080354291031927",
          "RunOption": {
            "$type": "workflow.ScheduleRun",
            "workflow.ScheduleRun": {
              "Interval": "0 */5 * * * *",
              "ParentRunID": "schedule_hello_world_1752497245218"
            }
          },
          "StepID": "",
          "Variables": {
            "input": { "$type": "schema.String", "schema.String": "" }
          }
        },
        "ExpectedRunTimestamp": 1752500100
      }
    },
    "ID": "run_id:7784080354291031927",
    "Type": "process",
    "Version": 1
  },
  {
    "Data": {
      "$type": "workflow.Done",
      "workflow.Done": {
        "BaseState": {
          "DefaultMaxRetries": 3,
          "ExprResult": {},
          "Flow": {
            "$type": "workflow.Flow",
            "workflow.Flow": {
              "Name": "hello_world"
            }
          },
          "RunID": "run_id:6955759398335118867",
          "RunOption": {
            "$type": "workflow.ScheduleRun",
            "workflow.ScheduleRun": {
              "Interval": "0 */5 * * * *",
              "ParentRunID": "schedule_hello_world_1752497245218"
            }
          },
          "StepID": "end1",
          "Variables": {
            "input": { "$type": "schema.String", "schema.String": "" },
            "res": { "$type": "schema.String", "schema.String": "hello " }
          }
        },
        "Result": { "$type": "schema.String", "schema.String": "hello " }
      }
    },
    "ID": "run_id:6955759398335118867",
    "Type": "process",
    "Version": 2
  }
]

// Function to extract ParentRunID from state - same logic as in component
function extractParentRunID(state: any): string | null {
  if (!state.Data) return null
  
  const stateData = state.Data
  const stateType = stateData.$type
  
  // Extract BaseState from different state types
  let baseState: any = null
  if (stateType && stateData[stateType as keyof typeof stateData]) {
    baseState = (stateData[stateType as keyof typeof stateData] as any)?.BaseState
  }
  
  return baseState?.RunOption?.['workflow.ScheduleRun']?.ParentRunID || null
}

describe('ScheduleHistoryDialog Data Processing', () => {
  it('should extract ParentRunID from real API data correctly', () => {
    const expectedParentRunID = "schedule_hello_world_1752497245218"
    
    realApiStates.forEach((state, index) => {
      const extractedParentRunID = extractParentRunID(state)
      expect(extractedParentRunID).toBe(expectedParentRunID)
      console.log(`State ${index}: ${state.Data.$type} -> ParentRunID: ${extractedParentRunID}`)
    })
  })

  it('should group states by RunID correctly', () => {
    const targetParentRunID = "schedule_hello_world_1752497245218"
    
    // Filter states for this ParentRunID
    const relatedStates = realApiStates.filter(state => {
      const parentRunID = extractParentRunID(state)
      return parentRunID === targetParentRunID
    })
    
    expect(relatedStates.length).toBe(3) // Should find all 3 states
    
    // Group by RunID
    const runGroups: { [runId: string]: any[] } = {}
    relatedStates.forEach(state => {
      const stateData = state.Data
      const stateType = stateData.$type
      let baseState: any = null
      if (stateType && stateData[stateType as keyof typeof stateData]) {
        baseState = (stateData[stateType as keyof typeof stateData] as any)?.BaseState
      }
      
      const runId = baseState?.RunID
      if (runId) {
        if (!runGroups[runId]) {
          runGroups[runId] = []
        }
        runGroups[runId].push(state)
      }
    })
    
    // Should have 3 different runs (each state has different RunID)
    expect(Object.keys(runGroups).length).toBe(3)
    
    Object.entries(runGroups).forEach(([runId, states]) => {
      console.log(`Run ${runId}: ${states.length} state(s)`)
      states.forEach(state => console.log(`  - ${state.Data.$type}`))
    })
  })
})