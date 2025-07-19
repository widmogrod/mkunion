import { analyzeWorkflowParams, buildParamTree } from '../workflow-analyzer'
import * as workflow from '../../workflow/github_com_widmogrod_mkunion_x_workflow'

describe('workflow-analyzer', () => {
  it('should extract simple parameter paths', () => {
    const flow: workflow.Flow = {
      Name: 'test-flow',
      Arg: 'input',
      Body: [
        {
          $type: 'workflow.Apply',
          'workflow.Apply': {
            ID: 'apply1',
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

    const params = analyzeWorkflowParams(flow)
    
    expect(params).toHaveLength(2)
    expect(params.map(p => p.path)).toContain('input.first')
    expect(params.map(p => p.path)).toContain('input.second')
  })

  it('should build parameter tree from flat paths', () => {
    const params = [
      {
        path: 'input.name',
        required: true,
        usageContext: [{ type: 'function' as const, details: 'concat' }]
      },
      {
        path: 'input.config.timeout',
        required: true,
        usageContext: [{ type: 'comparison' as const, details: '>' }]
      }
    ]

    const tree = buildParamTree(params)
    
    expect(tree.input).toBeDefined()
    expect(tree.input.name).toEqual({
      type: undefined,
      required: true,
      usageContext: [{ type: 'function', details: 'concat' }]
    })
    expect(tree.input.config).toBeDefined()
    expect(tree.input.config.timeout).toBeDefined()
  })

  it('should handle workflows with no parameters', () => {
    const flow: workflow.Flow = {
      Name: 'no-params',
      Arg: 'input',
      Body: [
        {
          $type: 'workflow.End',
          'workflow.End': {
            ID: 'end1',
            Result: {
              $type: 'workflow.SetValue',
              'workflow.SetValue': {
                Value: {
                  $type: 'schema.String',
                  'schema.String': 'fixed result'
                }
              }
            }
          }
        }
      ]
    }

    const params = analyzeWorkflowParams(flow)
    expect(params).toHaveLength(0)
  })

  it('should extract parameters from conditionals', () => {
    const flow: workflow.Flow = {
      Name: 'conditional-flow',
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
                    Path: 'input.enabled'
                  }
                },
                Operation: '==',
                Right: {
                  $type: 'workflow.SetValue',
                  'workflow.SetValue': {
                    Value: {
                      $type: 'schema.Bool',
                      'schema.Bool': true
                    }
                  }
                }
              }
            },
            Then: [
              {
                $type: 'workflow.End',
                'workflow.End': {
                  ID: 'end1',
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
        }
      ]
    }

    const params = analyzeWorkflowParams(flow)
    
    expect(params).toHaveLength(2)
    expect(params.map(p => p.path)).toContain('input.enabled')
    expect(params.map(p => p.path)).toContain('input.result')
    
    // Check usage context
    const enabledParam = params.find(p => p.path === 'input.enabled')
    expect(enabledParam?.usageContext).toContainEqual({
      type: 'comparison',
      details: 'Comparison: =='
    })
  })
})