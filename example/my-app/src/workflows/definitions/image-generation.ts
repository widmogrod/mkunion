import * as workflow from '../../workflow/github_com_widmogrod_mkunion_x_workflow'
import { WORKFLOW_NAMES, FUNCTION_NAMES } from '../../constants/workflow'
import * as builders from '../builders'

export function createImageGenerationFlow(): workflow.Flow {
  return builders.createFlow(WORKFLOW_NAMES.GENERATE_IMAGE, 'input', [
    // Generate image
    builders.assign(
      'assign1',
      'res',
      builders.apply(
        'apply1',
        FUNCTION_NAMES.GEN_IMAGE_B64,
        [builders.getValue('input.prompt')]
      )
    ),
    // Resize image
    builders.assign(
      'assign2',
      'res_small',
      builders.apply(
        'apply2',
        FUNCTION_NAMES.RESIZE_IMG_B64,
        [
          builders.getValue('res'),
          builders.getValue('input.width'),
          builders.getValue('input.height')
        ]
      )
    ),
    // Return resized image
    builders.end('end1', builders.getValue('res_small'))
  ])
}