import { WorkflowParam } from './workflow-analyzer'
import * as workflow from '../workflow/github_com_widmogrod_mkunion_x_workflow'
import * as schema from '../workflow/github_com_widmogrod_mkunion_x_schema'

/**
 * Type inference rules and constraints
 */
export type InferredType = 'string' | 'number' | 'boolean' | 'array' | 'object' | 'any'

interface TypeConstraint {
  possibleTypes: Set<InferredType>
  evidence: string[]
}

/**
 * Infer types for workflow parameters based on their usage
 */
export function inferParamTypes(
  params: WorkflowParam[],
  flow: workflow.Flow
): Map<string, InferredType> {
  const typeMap = new Map<string, InferredType>()
  const constraints = new Map<string, TypeConstraint>()
  
  // Initialize constraints for each parameter
  params.forEach(param => {
    constraints.set(param.path, {
      possibleTypes: new Set<InferredType>(['string', 'number', 'boolean', 'array', 'object', 'any']),
      evidence: []
    })
  })
  
  // Analyze workflow to collect type constraints
  if (flow.Body) {
    flow.Body.forEach(expr => {
      collectTypeConstraints(expr, constraints)
    })
  }
  
  // Resolve constraints to determine types
  params.forEach(param => {
    const constraint = constraints.get(param.path)
    if (constraint && constraint.possibleTypes.size > 0) {
      // Priority order for type selection
      const typePriority: InferredType[] = ['string', 'number', 'boolean', 'array', 'object', 'any']
      const resolvedType = typePriority.find(t => constraint.possibleTypes.has(t)) || 'any'
      typeMap.set(param.path, resolvedType)
    } else {
      typeMap.set(param.path, 'any')
    }
  })
  
  return typeMap
}

/**
 * Collect type constraints from expressions
 */
function collectTypeConstraints(
  expr: workflow.Expr,
  constraints: Map<string, TypeConstraint>
): void {
  if (!expr.$type) return
  
  switch (expr.$type) {
    case 'workflow.Apply':
      const apply = expr['workflow.Apply']
      if (apply?.Name && apply.Args) {
        // Infer types based on function expectations
        const functionTypes = getFunctionArgTypes(apply.Name)
        apply.Args.forEach((arg, index) => {
          if (functionTypes[index]) {
            constrainReshaper(arg, functionTypes[index], constraints, `Function ${apply.Name} arg ${index}`)
          }
        })
      }
      break
      
    case 'workflow.Choose':
      const choose = expr['workflow.Choose']
      if (choose?.If) {
        // Conditions often involve booleans
        constrainPredicate(choose.If, constraints)
      }
      if (choose?.Then) {
        choose.Then.forEach(e => collectTypeConstraints(e, constraints))
      }
      if (choose?.Else) {
        choose.Else.forEach(e => collectTypeConstraints(e, constraints))
      }
      break
      
    case 'workflow.Assign':
      const assign = expr['workflow.Assign']
      if (assign?.Val) {
        // Recursively analyze the assigned expression
        collectTypeConstraints(assign.Val, constraints)
      }
      break
      
    case 'workflow.End':
      const end = expr['workflow.End']
      if (end?.Result) {
        // End result is a Reshaper
        inferReshaperType(end.Result, constraints)
      }
      break
  }
}

/**
 * Constrain a reshaper to a specific type
 */
function constrainReshaper(
  reshaper: workflow.Reshaper,
  expectedType: InferredType,
  constraints: Map<string, TypeConstraint>,
  evidence: string
): void {
  if (reshaper.$type === 'workflow.GetValue') {
    const getValue = reshaper['workflow.GetValue']
    if (getValue?.Path) {
      const constraint = constraints.get(getValue.Path)
      if (constraint) {
        // Keep only the expected type
        constraint.possibleTypes = new Set([expectedType])
        constraint.evidence.push(evidence)
      }
    }
  }
}

/**
 * Infer type from reshaper usage
 */
function inferReshaperType(
  reshaper: workflow.Reshaper,
  constraints: Map<string, TypeConstraint>
): void {
  if (!reshaper.$type) return
  
  switch (reshaper.$type) {
    case 'workflow.SetValue':
      const setValue = reshaper['workflow.SetValue']
      if (setValue?.Value?.$type === 'schema.String') {
        // String literal indicates string type
        if (setValue.Value['schema.String'] !== undefined) {
          // This is a string value
        }
      } else if (setValue?.Value?.$type === 'schema.Number') {
        // Number literal
      } else if (setValue?.Value?.$type === 'schema.Bool') {
        // Boolean literal
      }
      break
  }
}

/**
 * Constrain predicate parameters
 */
function constrainPredicate(
  predicate: workflow.Predicate,
  constraints: Map<string, TypeConstraint>
): void {
  if (!predicate.$type) return
  
  switch (predicate.$type) {
    case 'workflow.Compare':
      const compare = predicate['workflow.Compare']
      if (compare?.Operation && compare.Left && compare.Right) {
        // Infer types based on comparison operation
        switch (compare.Operation) {
          case '==':
          case '!=':
            // Could be any type, check the other operand
            inferComparisonTypes(compare.Left, compare.Right, constraints)
            break
          case '>':
          case '<':
          case '>=':
          case '<=':
            // Numeric comparisons
            constrainReshaper(compare.Left, 'number', constraints, `Numeric comparison ${compare.Operation}`)
            constrainReshaper(compare.Right, 'number', constraints, `Numeric comparison ${compare.Operation}`)
            break
        }
      }
      break
      
    case 'workflow.And':
      const and = predicate['workflow.And']
      if (and?.L) {
        and.L.forEach(p => constrainPredicate(p, constraints))
      }
      break
      
    case 'workflow.Or':
      const or = predicate['workflow.Or']
      if (or?.L) {
        or.L.forEach(p => constrainPredicate(p, constraints))
      }
      break
  }
}

/**
 * Infer types from comparison operands
 */
function inferComparisonTypes(
  left: workflow.Reshaper,
  right: workflow.Reshaper,
  constraints: Map<string, TypeConstraint>
): void {
  // If one side is a literal, constrain the other side to match
  if (right.$type === 'workflow.SetValue') {
    const setValue = right['workflow.SetValue']
    if (setValue?.Value?.$type) {
      switch (setValue.Value.$type) {
        case 'schema.String':
          constrainReshaper(left, 'string', constraints, 'Compared with string literal')
          break
        case 'schema.Number':
          constrainReshaper(left, 'number', constraints, 'Compared with number literal')
          break
        case 'schema.Bool':
          constrainReshaper(left, 'boolean', constraints, 'Compared with boolean literal')
          break
      }
    }
  }
}

/**
 * Constrain a reshaper as an object type
 */
function constrainAsObject(
  reshaper: workflow.Reshaper,
  constraints: Map<string, TypeConstraint>
): void {
  if (reshaper.$type === 'workflow.GetValue') {
    const getValue = reshaper['workflow.GetValue']
    if (getValue?.Path) {
      const constraint = constraints.get(getValue.Path)
      if (constraint) {
        // Remove non-object types
        constraint.possibleTypes.delete('string')
        constraint.possibleTypes.delete('number')
        constraint.possibleTypes.delete('boolean')
        constraint.evidence.push('Used as object (field access)')
      }
    }
  }
}

/**
 * Get expected argument types for known functions
 */
function getFunctionArgTypes(functionName: string): InferredType[] {
  // This would ideally come from a function registry
  // For now, we'll hardcode some common functions
  const knownFunctions: Record<string, InferredType[]> = {
    'concat': ['string', 'string'],
    'add': ['number', 'number'],
    'multiply': ['number', 'number'],
    'and': ['boolean', 'boolean'],
    'or': ['boolean', 'boolean'],
    'len': ['string'],
    'count': ['array'],
    'get': ['object', 'string'],
  }
  
  return knownFunctions[functionName] || []
}

/**
 * Generate a JSON schema from inferred types
 */
export function generateJsonSchema(
  paramTree: any,
  typeMap: Map<string, InferredType>
): any {
  const schema: any = {
    type: 'object',
    properties: {},
    required: []
  }
  
  function processNode(node: any, path: string, target: any): void {
    Object.keys(node).forEach(key => {
      const fullPath = path ? `${path}.${key}` : key
      const value = node[key]
      
      if (value && typeof value === 'object' && !value.type) {
        // Nested object
        target[key] = {
          type: 'object',
          properties: {}
        }
        processNode(value, fullPath, target[key].properties)
      } else if (value && value.type) {
        // Leaf node with type info
        const inferredType = typeMap.get(fullPath) || 'any'
        target[key] = mapToJsonSchemaType(inferredType)
        if (value.required) {
          schema.required.push(key)
        }
      }
    })
  }
  
  processNode(paramTree, '', schema.properties)
  
  return schema
}

/**
 * Map inferred types to JSON schema types
 */
function mapToJsonSchemaType(type: InferredType): any {
  switch (type) {
    case 'string':
      return { type: 'string' }
    case 'number':
      return { type: 'number' }
    case 'boolean':
      return { type: 'boolean' }
    case 'array':
      return { type: 'array', items: { type: 'string' } }
    case 'object':
      return { type: 'object' }
    default:
      return { type: 'string' } // Default to string for 'any'
  }
}