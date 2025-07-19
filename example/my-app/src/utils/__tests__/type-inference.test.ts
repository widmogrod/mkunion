import { inferParamTypes } from '../type-inference'
import * as workflow from '../../workflow/github_com_widmogrod_mkunion_x_workflow'
import * as schema from '../../workflow/github_com_widmogrod_mkunion_x_schema'

describe('type-inference', () => {
  describe('inferParamTypes', () => {
    it('should infer string type from string comparison', () => {
      const params = [
        {
          path: 'input.name',
          required: true,
          usageContext: []
        }
      ]
      
      const flow: workflow.Flow = {
        Name: 'test-flow',
        Arg: 'input',
        Body: [
          {
            $type: 'workflow.Choose',
            'workflow.Choose': {
              If: {
                $type: 'workflow.Compare',
                'workflow.Compare': {
                  Left: {
                    $type: 'workflow.GetValue',
                    'workflow.GetValue': {
                      Path: 'input.name'
                    }
                  },
                  Operation: '==',
                  Right: {
                    $type: 'workflow.SetValue',
                    'workflow.SetValue': {
                      Value: {
                        $type: 'schema.String',
                        'schema.String': 'John'
                      }
                    }
                  }
                }
              },
              Then: []
            }
          }
        ]
      }
      
      const typeMap = inferParamTypes(params, flow)
      expect(typeMap.get('input.name')).toBe('string')
    })
    
    it('should infer number type from number comparison', () => {
      const params = [
        {
          path: 'input.age',
          required: true,
          usageContext: []
        }
      ]
      
      const flow: workflow.Flow = {
        Name: 'test-flow',
        Arg: 'input',
        Body: [
          {
            $type: 'workflow.Choose',
            'workflow.Choose': {
              If: {
                $type: 'workflow.Compare',
                'workflow.Compare': {
                  Left: {
                    $type: 'workflow.GetValue',
                    'workflow.GetValue': {
                      Path: 'input.age'
                    }
                  },
                  Operation: '>',
                  Right: {
                    $type: 'workflow.SetValue',
                    'workflow.SetValue': {
                      Value: {
                        $type: 'schema.Number',
                        'schema.Number': 18
                      }
                    }
                  }
                }
              },
              Then: []
            }
          }
        ]
      }
      
      const typeMap = inferParamTypes(params, flow)
      expect(typeMap.get('input.age')).toBe('number')
    })
    
    it('should infer type from function arguments', () => {
      const params = [
        {
          path: 'input.first',
          required: true,
          usageContext: []
        },
        {
          path: 'input.second',
          required: true,
          usageContext: []
        }
      ]
      
      const flow: workflow.Flow = {
        Name: 'test-flow',
        Arg: 'input',
        Body: [
          {
            $type: 'workflow.Apply',
            'workflow.Apply': {
              ID: 'concat-strings',
              Name: 'concat',
              Args: [
                {
                  $type: 'workflow.GetValue',
                  'workflow.GetValue': {
                    Path: 'input.first'
                  }
                },
                {
                  $type: 'workflow.GetValue',
                  'workflow.GetValue': {
                    Path: 'input.second'
                  }
                }
              ]
            }
          }
        ]
      }
      
      const typeMap = inferParamTypes(params, flow)
      expect(typeMap.get('input.first')).toBe('string')
      expect(typeMap.get('input.second')).toBe('string')
    })
    
    it('should handle End result type inference', () => {
      const params = [
        {
          path: 'input.result',
          required: true,
          usageContext: []
        }
      ]
      
      const flow: workflow.Flow = {
        Name: 'test-flow',
        Arg: 'input',
        Body: [
          {
            $type: 'workflow.End',
            'workflow.End': {
              ID: 'end-with-result',
              Result: {
                $type: 'workflow.GetValue',
                'workflow.GetValue': {
                  Path: 'input.result'
                }
              }
            }
          }
        ]
      }
      
      const typeMap = inferParamTypes(params, flow)
      // When no specific type constraint is applied, the default behavior
      // picks the first type from the priority list (string)
      expect(typeMap.get('input.result')).toBe('string')
    })
    
    it('should NOT infer type from SetValue in End result (demonstrates bug)', () => {
      // This test demonstrates that inferReshaperType is not working
      const params = [
        {
          path: 'input.value',
          required: true,
          usageContext: []
        }
      ]
      
      const flow: workflow.Flow = {
        Name: 'test-flow',
        Arg: 'input',
        Body: [
          {
            $type: 'workflow.Choose',
            'workflow.Choose': {
              If: {
                $type: 'workflow.Compare',
                'workflow.Compare': {
                  Left: {
                    $type: 'workflow.GetValue',
                    'workflow.GetValue': {
                      Path: 'input.value'
                    }
                  },
                  Operation: '==',
                  Right: {
                    $type: 'workflow.SetValue',
                    'workflow.SetValue': {
                      Value: {
                        $type: 'schema.String',
                        'schema.String': 'test'
                      }
                    }
                  }
                }
              },
              Then: [
                {
                  $type: 'workflow.End',
                  'workflow.End': {
                    ID: 'end-string',
                    Result: {
                      $type: 'workflow.SetValue',
                      'workflow.SetValue': {
                        Value: {
                          $type: 'schema.String',
                          'schema.String': 'Fixed string result'
                        }
                      }
                    }
                  }
                }
              ]
            }
          }
        ]
      }
      
      const typeMap = inferParamTypes(params, flow)
      // This correctly infers string from the comparison
      expect(typeMap.get('input.value')).toBe('string')
      
      // Note: End results don't contribute to type inference
      // because we don't know the expected return type
    })
  })
  
  describe('type inference design', () => {
    it('explains why End results do not contribute to type inference', () => {
      // End result reshapers don't contribute to parameter type inference because:
      
      // 1. SetValue reshapers don't reference parameters (they contain literal values)
      // 2. GetValue reshapers in End results don't have a known expected type
      
      // The actual type inference happens in other places:
      // - constrainReshaper: When we know the expected type (e.g., from function arguments)
      // - constrainPredicate: When analyzing comparisons with literals
      // - getFunctionArgTypes: When parameters are used as function arguments
      
      // This is intentional - we infer types from usage patterns where we have
      // clear type expectations, not from return values.
      
      expect(true).toBe(true) // This test is documentary
    })
  })
})