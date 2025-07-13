/**
 * Type helper utilities for exhaustive pattern matching and type safety
 */

/**
 * Helper function for exhaustiveness checking in switch statements.
 * This will cause a TypeScript compile error if not all cases of a union type are handled.
 * 
 * @example
 * ```typescript
 * type Status = 'active' | 'inactive' | 'pending'
 * 
 * function getStatusColor(status: Status) {
 *   switch (status) {
 *     case 'active': return 'green'
 *     case 'inactive': return 'red'
 *     // Compile error: Argument of type '"pending"' is not assignable to parameter of type 'never'
 *     default: return assertNever(status)
 *   }
 * }
 * ```
 */
export function assertNever(value: never): never {
  throw new Error(`Unhandled discriminated union member: ${JSON.stringify(value)}`)
}

/**
 * Type guard to check if a value is not null or undefined
 */
export function isDefined<T>(value: T | null | undefined): value is T {
  return value !== null && value !== undefined
}

/**
 * Simple pattern matching utility for discriminated unions
 * 
 * @example
 * ```typescript
 * const result = match(state, {
 *   'workflow.Done': () => 'Complete',
 *   'workflow.Error': () => 'Failed',
 *   'workflow.Await': () => 'Waiting',
 * })
 * ```
 */
export function match<T extends { $type?: string }, R>(
  value: T,
  cases: { [K in NonNullable<T['$type']>]: (value: T) => R }
): R {
  const type = value.$type
  if (!type || !(type in cases)) {
    throw new Error(`No case found for type: ${type}`)
  }
  return (cases as any)[type](value)
}

/**
 * Utility type to extract the discriminated union member by its type
 * 
 * @example
 * ```typescript
 * type DoneState = ExtractByType<workflow.State, 'workflow.Done'>
 * ```
 */
export type ExtractByType<T extends { $type?: string }, Type extends string> = 
  T extends { $type?: Type } ? T : never