export const SYSTEM_PARAM_KEYS = [
  'template_id',
  'template_name',
  'template_owner',
  'node_owner',
] as const

export const SYSTEM_PARAMS = SYSTEM_PARAM_KEYS.map((key) => ({
  key,
  name: key,
  example: `{{${key}}}`,
}))
