import * as workflow from '../workflow/github_com_widmogrod_mkunion_x_workflow'
import * as schema from '../workflow/github_com_widmogrod_mkunion_x_schema'

// Helper functions to create workflow expressions with less boilerplate

export function getValue(path: string): workflow.Reshaper {
  return {
    $type: 'workflow.GetValue',
    'workflow.GetValue': { Path: path }
  }
}

export function setValue(value: schema.Schema): workflow.Reshaper {
  return {
    $type: 'workflow.SetValue',
    'workflow.SetValue': { Value: value }
  }
}

export function stringValue(str: string): schema.Schema {
  return { 
    '$type': 'schema.String',
    'schema.String': str 
  }
}

export function numberValue(num: number): schema.Schema {
  return { 
    '$type': 'schema.Number',
    'schema.Number': num 
  }
}

export function booleanValue(bool: boolean): schema.Schema {
  return { 
    '$type': 'schema.Bool',
    'schema.Bool': bool 
  }
}

export function mapValue(obj: { [key: string]: schema.Schema }): schema.Schema {
  return { 
    '$type': 'schema.Map',
    'schema.Map': obj 
  }
}

export function compare(
  left: workflow.Reshaper,
  operation: string,
  right: workflow.Reshaper
): workflow.Predicate {
  return {
    $type: 'workflow.Compare',
    'workflow.Compare': {
      Operation: operation,
      Left: left,
      Right: right
    }
  }
}

export function choose(
  id: string,
  condition: workflow.Predicate,
  then: workflow.Expr[]
): workflow.Expr {
  return {
    $type: 'workflow.Choose',
    'workflow.Choose': {
      ID: id,
      If: condition,
      Then: then
    }
  }
}

export function assign(
  id: string,
  varName: string,
  val: workflow.Expr
): workflow.Expr {
  return {
    $type: 'workflow.Assign',
    'workflow.Assign': {
      ID: id,
      VarOk: varName,
      VarErr: '',
      Val: val
    }
  }
}

export function apply(
  id: string,
  functionName: string,
  args: workflow.Reshaper[]
): workflow.Expr {
  return {
    $type: 'workflow.Apply',
    'workflow.Apply': {
      ID: id,
      Name: functionName,
      Args: args
    }
  }
}

export function end(id: string, result: workflow.Reshaper): workflow.Expr {
  return {
    $type: 'workflow.End',
    'workflow.End': {
      ID: id,
      Result: result
    }
  }
}

export function createFlow(name: string, arg: string, body: workflow.Expr[]): workflow.Flow {
  return {
    Name: name,
    Arg: arg,
    Body: body
  }
}

export function createRunCommand(flow: workflow.Flow, input: schema.Schema): workflow.Command {
  return {
    $type: 'workflow.Run',
    'workflow.Run': {
      Flow: {
        $type: 'workflow.Flow',
        'workflow.Flow': flow
      },
      Input: input
    }
  }
}

export function createScheduledRunCommand(
  flow: workflow.Flow, 
  input: schema.Schema,
  cronExpression: string,
  parentRunId?: string
): workflow.Command {
  return {
    $type: 'workflow.Run',
    'workflow.Run': {
      Flow: {
        $type: 'workflow.Flow',
        'workflow.Flow': flow
      },
      Input: input,
      RunOption: {
        $type: 'workflow.ScheduleRun',
        'workflow.ScheduleRun': {
          Interval: cronExpression,
          ParentRunID: parentRunId || `schedule_${Date.now()}`
        }
      }
    }
  }
}

export function createDelayedRunCommand(
  flow: workflow.Flow,
  input: schema.Schema,
  delaySeconds: number
): workflow.Command {
  return {
    $type: 'workflow.Run',
    'workflow.Run': {
      Flow: {
        $type: 'workflow.Flow',
        'workflow.Flow': flow
      },
      Input: input,
      RunOption: {
        $type: 'workflow.DelayRun',
        'workflow.DelayRun': {
          DelayBySeconds: delaySeconds
        }
      }
    }
  }
}