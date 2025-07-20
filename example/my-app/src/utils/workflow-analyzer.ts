import * as workflow from '../workflow/github_com_widmogrod_mkunion_x_workflow'

/**
 * Represents a parameter extracted from a workflow
 */
export interface WorkflowParam {
  path: string          // e.g., "input.name", "input.config.timeout"
  required: boolean     // based on usage in conditions
  inferredType?: string // "string", "number", "boolean", etc.
  usageContext: Array<{
    type: 'function' | 'comparison' | 'condition' | 'assignment'
    details: string
  }>
}

/**
 * Analyzes a workflow to extract input parameters
 */
export function analyzeWorkflowParams(flow: workflow.Flow): WorkflowParam[] {
  const params = new Map<string, WorkflowParam>()
  const inputArg = flow.Arg || 'input'
  
  // Analyze the workflow body
  if (flow.Body) {
    flow.Body.forEach(expr => {
      analyzeExpression(expr, inputArg, params)
    })
  }
  
  return Array.from(params.values())
}

/**
 * Recursively analyze an expression to find parameter usage
 */
function analyzeExpression(
  expr: workflow.Expr,
  inputArg: string,
  params: Map<string, WorkflowParam>
): void {
  if (!expr.$type) return
  
  switch (expr.$type) {
    case 'workflow.Assign':
      const assign = expr['workflow.Assign']
      if (assign?.Val) {
        // Recursively analyze the assigned expression
        analyzeExpression(assign.Val, inputArg, params)
      }
      break
      
    case 'workflow.Apply':
      const apply = expr['workflow.Apply']
      if (apply?.Args) {
        apply.Args.forEach(arg => {
          analyzeReshaper(arg, inputArg, params, {
            type: 'function',
            details: `Function: ${apply.Name}`
          })
        })
      }
      break
      
    case 'workflow.Choose':
      const choose = expr['workflow.Choose']
      if (choose?.If) {
        analyzePredicate(choose.If, inputArg, params)
      }
      if (choose?.Then) {
        choose.Then.forEach(thenExpr => analyzeExpression(thenExpr, inputArg, params))
      }
      if (choose?.Else) {
        choose.Else.forEach(elseExpr => analyzeExpression(elseExpr, inputArg, params))
      }
      break
      
    case 'workflow.End':
      const end = expr['workflow.End']
      if (end?.Result) {
        analyzeReshaper(end.Result, inputArg, params, {
          type: 'assignment',
          details: 'Return value'
        })
      }
      break
  }
}

/**
 * Analyze a reshaper for parameter usage
 */
function analyzeReshaper(
  reshaper: workflow.Reshaper,
  inputArg: string,
  params: Map<string, WorkflowParam>,
  context: { type: 'function' | 'comparison' | 'condition' | 'assignment', details: string }
): void {
  if (!reshaper.$type) return
  
  switch (reshaper.$type) {
    case 'workflow.GetValue':
      const getValue = reshaper['workflow.GetValue']
      if (getValue?.Path && getValue.Path.startsWith(inputArg)) {
        const existing = params.get(getValue.Path) || {
          path: getValue.Path,
          required: true,
          usageContext: []
        }
        existing.usageContext.push(context)
        params.set(getValue.Path, existing)
      }
      break
      
    case 'workflow.SetValue':
      // SetValue contains a literal value, not a parameter reference
      // No parameter analysis needed here
      break
  }
}

/**
 * Analyze predicates for parameter usage
 */
function analyzePredicate(
  predicate: workflow.Predicate,
  inputArg: string,
  params: Map<string, WorkflowParam>
): void {
  if (!predicate.$type) return
  
  switch (predicate.$type) {
    case 'workflow.Compare':
      const compare = predicate['workflow.Compare']
      if (compare?.Left) {
        analyzeReshaper(compare.Left, inputArg, params, {
          type: 'comparison',
          details: `Comparison: ${compare.Operation}`
        })
      }
      if (compare?.Right) {
        analyzeReshaper(compare.Right, inputArg, params, {
          type: 'comparison',
          details: `Comparison: ${compare.Operation}`
        })
      }
      break
      
    case 'workflow.And':
      const and = predicate['workflow.And']
      if (and?.L) {
        and.L.forEach(p => analyzePredicate(p, inputArg, params))
      }
      break
      
    case 'workflow.Or':
      const or = predicate['workflow.Or']
      if (or?.L) {
        or.L.forEach(p => analyzePredicate(p, inputArg, params))
      }
      break
      
    case 'workflow.Not':
      const not = predicate['workflow.Not']
      if (not?.P) {
        analyzePredicate(not.P, inputArg, params)
      }
      break
  }
}

/**
 * Build a tree structure from flat parameter paths
 */
export interface ParamTree {
  [key: string]: ParamTree | { 
    type?: string
    required: boolean
    usageContext: Array<{ type: string, details: string }>
  }
}

export function buildParamTree(params: WorkflowParam[]): ParamTree {
  const tree: ParamTree = {}
  
  params.forEach(param => {
    const parts = param.path.split('.')
    let current: any = tree
    
    for (let i = 0; i < parts.length; i++) {
      const part = parts[i]
      const isLast = i === parts.length - 1
      
      if (isLast) {
        current[part] = {
          type: param.inferredType,
          required: param.required,
          usageContext: param.usageContext
        }
      } else {
        if (!current[part] || typeof current[part] !== 'object' || current[part].type) {
          current[part] = {}
        }
        current = current[part]
      }
    }
  })
  
  return tree
}