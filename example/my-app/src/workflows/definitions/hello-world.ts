import * as workflow from '../../workflow/github_com_widmogrod_mkunion_x_workflow'
import { WORKFLOW_NAMES, FUNCTION_NAMES, MAGIC_NUMBERS } from '../../constants/workflow'
import * as builders from '../builders'

export function createHelloWorldFlow(withError: boolean = false): workflow.Flow {
  const workflowName = withError ? WORKFLOW_NAMES.HELLO_WORLD_ERROR : WORKFLOW_NAMES.HELLO_WORLD
  const functionName = withError ? FUNCTION_NAMES.CONCAT_ERROR : FUNCTION_NAMES.CONCAT

  return builders.createFlow(workflowName, 'input', [
    // Check if input equals "666"
    builders.choose(
      'choose1',
      builders.compare(
        builders.getValue('input'),
        '=',
        builders.setValue(builders.stringValue(MAGIC_NUMBERS.EVIL_NUMBER))
      ),
      [
        builders.end('end2', builders.setValue(builders.stringValue(MAGIC_NUMBERS.EVIL_MESSAGE)))
      ]
    ),
    // Assign concatenated result to 'res'
    builders.assign(
      'assign1',
      'res',
      builders.apply(
        'apply1',
        functionName,
        [
          builders.setValue(builders.stringValue('hello ')),
          builders.getValue('input')
        ]
      )
    ),
    // Return the result
    builders.end('end1', builders.getValue('res'))
  ])
}