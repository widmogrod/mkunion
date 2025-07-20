import { useState, useCallback } from 'react'
import * as workflow from '../workflow/github_com_widmogrod_mkunion_x_workflow'
import * as schemaless from '../workflow/github_com_widmogrod_mkunion_x_storage_schemaless'
import * as predicate from '../workflow/github_com_widmogrod_mkunion_x_storage_predicate'
import * as schema from '../workflow/github_com_widmogrod_mkunion_x_schema'
import * as openai from '../workflow/github_com_sashabaranov_go-openai'
import * as app from '../workflow/github_com_widmogrod_mkunion_exammple_my-app'

// Use environment variable with fallback to localhost for development
const API_BASE_URL = process.env.REACT_APP_API_BASE_URL || 'http://localhost:8080'

export interface ListProps<T> {
  baseURL?: string
  path?: string
  sort?: {
    [key: string]: boolean
  }
  limit?: number
  where?: predicate.WherePredicates
  prevPage?: string
  nextPage?: string
}

export function useWorkflowApi() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<Error | null>(null)

  const flowCreate = useCallback(async (flow: workflow.Flow) => {
    setLoading(true)
    setError(null)
    try {
      // Wrap the Flow in a Workflow union type
      const workflowData: workflow.Workflow = {
        $type: 'workflow.Flow',
        'workflow.Flow': flow
      }
      
      const response = await fetch(`${API_BASE_URL}/flow`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(workflowData),
      })
      const data = await response.text()
      return data
    } catch (err) {
      setError(err as Error)
      throw err
    } finally {
      setLoading(false)
    }
  }, [])

  const storageList = useCallback(async <T,>(input: ListProps<T>): Promise<schemaless.PageResult<schemaless.Record<T>>> => {
    // Don't set loading here - let individual components manage their own loading state
    setError(null)
    try {
      let url = input.baseURL || API_BASE_URL + '/'
      url = url + input.path

      const response = await fetch(url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          Limit: input.limit || 30,
          Sort: input.sort && Object.keys(input.sort).map((key) => ({
            Field: key,
            Descending: input.sort?.[key],
          })),
          Where: input.where,
          After: input.nextPage,
          Before: input.prevPage,
        } as schemaless.FindingRecords<schemaless.Record<T>>),
      })
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }
      
      return await response.json()
    } catch (err) {
      setError(err as Error)
      // Return empty result instead of throwing to prevent app crash
      return {
        Items: [],
        Next: undefined,
        Prev: undefined
      }
    } finally {
      // Don't change loading state here - let individual components manage their own loading
    }
  }, [])

  const listStates = useCallback((input?: ListProps<workflow.State>) => {
    return storageList<workflow.State>({
      ...input,
      path: 'states',
    })
  }, [storageList])

  const listFlows = useCallback((input?: ListProps<workflow.Flow>) => {
    return storageList<workflow.Flow>({
      ...input,
      path: 'flows',
    })
  }, [storageList])

  const runCommand = useCallback(async (cmd: workflow.Command): Promise<workflow.State> => {
    setLoading(true)
    setError(null)
    try {
      const response = await fetch(`${API_BASE_URL}/`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(cmd),
      })
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }
      
      return await response.json() as workflow.State
    } catch (err) {
      setError(err as Error)
      throw err
    } finally {
      setLoading(false)
    }
  }, [])

  const workflowToStr = useCallback(async (runId: string): Promise<string> => {
    setError(null)
    try {
      const response = await fetch(`${API_BASE_URL}/workflow-to-str-from-run/${runId}`, {
        method: 'GET',
        headers: {
          'Accept': 'text/plain',
        },
      })
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }
      
      return await response.text()
    } catch (err) {
      setError(err as Error)
      return '' // Return empty string on error
    }
  }, [])

  const workflowAstToStr = useCallback(async (workflowAst: workflow.Workflow): Promise<string> => {
    setError(null)
    try {
      const response = await fetch(`${API_BASE_URL}/workflow-to-str`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Accept': 'text/plain',
        },
        body: JSON.stringify(workflowAst),
      })
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }
      
      return await response.text()
    } catch (err) {
      setError(err as Error)
      return '' // Return empty string on error
    }
  }, [])

  const submitCallback = useCallback(async (callbackID: string, result: schema.Schema): Promise<workflow.State> => {
    setLoading(true)
    setError(null)
    try {
      // Create callback command structure
      const callbackCommand: workflow.Command = {
        $type: 'workflow.Callback',
        'workflow.Callback': {
          CallbackID: callbackID,
          Result: result
        }
      }

      const response = await fetch(`${API_BASE_URL}/callback`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(callbackCommand),
      })
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }
      
      return await response.json() as workflow.State
    } catch (err) {
      setError(err as Error)
      throw err
    } finally {
      setLoading(false)
    }
  }, [])

  const stopSchedule = useCallback(async (parentRunID: string): Promise<workflow.State> => {
    const stopCommand: workflow.Command = {
      $type: 'workflow.StopSchedule',
      'workflow.StopSchedule': {
        ParentRunID: parentRunID
      }
    }
    return runCommand(stopCommand)
  }, [runCommand])

  const resumeSchedule = useCallback(async (parentRunID: string): Promise<workflow.State> => {
    const resumeCommand: workflow.Command = {
      $type: 'workflow.ResumeSchedule',
      'workflow.ResumeSchedule': {
        ParentRunID: parentRunID
      }
    }
    return runCommand(resumeCommand)
  }, [runCommand])

  const tryRecover = useCallback(async (runID: string): Promise<workflow.State> => {
    const recoverCommand: workflow.Command = {
      $type: 'workflow.TryRecover',
      'workflow.TryRecover': {
        RunID: runID
      }
    }
    return runCommand(recoverCommand)
  }, [runCommand])

  const deleteStates = useCallback(async (states: schemaless.Record<workflow.State>[]): Promise<void> => {
    setLoading(true)
    setError(null)
    try {
      // Build the deleting map from full record objects
      const deleting: { [key: string]: schemaless.Record<workflow.State> } = {}
      states.forEach(state => {
        if (!state.ID) {
          return
        }
        deleting[state.ID] = state
      })

      const updateRequest: schemaless.UpdateRecords<schemaless.Record<workflow.State>> = {
        UpdatingPolicy: 1, // PolicyOverwriteServerChanges
        Saving: {},
        Deleting: deleting
      }

      const response = await fetch(`${API_BASE_URL}/state-updating`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(updateRequest),
      })
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }
    } catch (err) {
      setError(err as Error)
      throw err
    } finally {
      setLoading(false)
    }
  }, [])

  const deleteFlows = useCallback(async (flows: schemaless.Record<workflow.Flow>[]): Promise<void> => {
    setLoading(true)
    setError(null)
    try {
      // Build the deleting map - need to handle the type mismatch
      const deleting: { [key: string]: schemaless.Record<workflow.Flow> } = {}
      flows.forEach(flow => {
        if (!flow.ID) {
          return
        }
        
        // The server stores flows directly, not wrapped in union types
        // But the CDC process expects everything to be States, which is a server bug
        deleting[flow.ID] = flow
      })

      const updateRequest = {
        UpdatingPolicy: 1, // PolicyOverwriteServerChanges
        Saving: {},
        Deleting: deleting
      }

      // WARNING: Flow deletion will cause server panic due to CDC process bug.
      // The CDC is typed for States only but processes all store changes.

      const response = await fetch(`${API_BASE_URL}/flows-updating`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(updateRequest),
      })
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }
    } catch (err) {
      setError(err as Error)
      throw err
    } finally {
      setLoading(false)
    }
  }, [])

  const callFunction = useCallback(async (name: string, args: schema.Schema[]): Promise<workflow.FunctionOutput> => {
    setLoading(true)
    setError(null)
    try {
      const functionInput: workflow.FunctionInput = {
        Name: name,
        Args: args
      }

      const response = await fetch(`${API_BASE_URL}/func`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(functionInput),
      })
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }
      
      return await response.json() as workflow.FunctionOutput
    } catch (err) {
      setError(err as Error)
      throw err
    } finally {
      setLoading(false)
    }
  }, [])

  const workflowRun = useCallback(async (flow: workflow.Flow, input: schema.Schema): Promise<workflow.State> => {
    // Wrap the Flow in a Workflow union type
    const workflowData: workflow.Workflow = {
      $type: 'workflow.Flow',
      'workflow.Flow': flow
    }
    
    const cmd: workflow.Command = {
      $type: 'workflow.Run',
      'workflow.Run': {
        Flow: workflowData,
        Input: input
      }
    }
    return runCommand(cmd)
  }, [runCommand])

  return {
    loading,
    error,
    flowCreate,
    storageList,
    listStates,
    listFlows,
    runCommand,
    workflowToStr,
    workflowAstToStr,
    submitCallback,
    stopSchedule,
    resumeSchedule,
    tryRecover,
    deleteStates,
    deleteFlows,
    callFunction,
    workflowRun,
  }
}

// Re-export types for convenience and type safety
export type {
  workflow,
  schemaless,
  predicate,
  schema,
  openai,
  app,
}