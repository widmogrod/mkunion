export const WORKFLOW_NAMES = {
  HELLO_WORLD: 'hello_world',
  HELLO_WORLD_ERROR: 'do_error',
  GENERATE_IMAGE: 'generateandresizeimage'
} as const

export const FUNCTION_NAMES = {
  CONCAT: 'concat',
  CONCAT_ERROR: 'concat_error',
  GEN_IMAGE_B64: 'genimageb64',
  RESIZE_IMG_B64: 'resizeimgb64'
} as const

export const DEFAULT_IMAGE_DIMENSIONS = {
  WIDTH: 100,
  HEIGHT: 100
}

export const MAGIC_NUMBERS = {
  EVIL_NUMBER: '666',
  EVIL_MESSAGE: 'Do no evil'
} as const