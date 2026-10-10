export const SYSTEM_PARAM_KEYS = [
  'TEMPLATE_ID',
  'TEMPLATE_NAME',
  'TEMPLATE_OWNER',
  'INSTANCE_ID',
  'NODE_ID',
  'NODE_NAME',
  'NODE_OWNER',
  'FLOW_ENV_OUT',
] as const

/** 实例详情页展示的系统参数（不含节点级与 FLOW_ENV_OUT）。 */
export const INSTANCE_SYSTEM_PARAM_KEYS = [
  'TEMPLATE_ID',
  'TEMPLATE_NAME',
  'TEMPLATE_OWNER',
  'INSTANCE_ID',
] as const

export type SystemParamRow = {
  key: string
  name: string
  /** {{KEY}} 占位符写法（执行前替换） */
  example: string
  /** $KEY shell 环境变量写法（运行时展开） */
  shellExample: string
  description: string
}

export const SYSTEM_PARAMS: SystemParamRow[] = [
  {
    key: 'TEMPLATE_ID',
    name: 'TEMPLATE_ID',
    example: '{{TEMPLATE_ID}}',
    shellExample: '$TEMPLATE_ID',
    description: '流水线模板 ID（实例级）',
  },
  {
    key: 'TEMPLATE_NAME',
    name: 'TEMPLATE_NAME',
    example: '{{TEMPLATE_NAME}}',
    shellExample: '$TEMPLATE_NAME',
    description: '流水线模板名称（实例级）',
  },
  {
    key: 'TEMPLATE_OWNER',
    name: 'TEMPLATE_OWNER',
    example: '{{TEMPLATE_OWNER}}',
    shellExample: '$TEMPLATE_OWNER',
    description: '模板负责人，逗号分隔（实例级）',
  },
  {
    key: 'INSTANCE_ID',
    name: 'INSTANCE_ID',
    example: '{{INSTANCE_ID}}',
    shellExample: '$INSTANCE_ID',
    description: '当前流水线实例 ID（实例级）',
  },
  {
    key: 'NODE_ID',
    name: 'NODE_ID',
    example: '{{NODE_ID}}',
    shellExample: '$NODE_ID',
    description: '当前运行节点的 ID（节点级）',
  },
  {
    key: 'NODE_NAME',
    name: 'NODE_NAME',
    example: '{{NODE_NAME}}',
    shellExample: '$NODE_NAME',
    description: '当前运行节点的名称（节点级）',
  },
  {
    key: 'NODE_OWNER',
    name: 'NODE_OWNER',
    example: '{{NODE_OWNER}}',
    shellExample: '$NODE_OWNER',
    description: '当前节点负责人，逗号分隔（节点级）',
  },
  {
    key: 'FLOW_ENV_OUT',
    name: 'FLOW_ENV_OUT',
    example: '{{FLOW_ENV_OUT}}',
    shellExample: '"$FLOW_ENV_OUT"',
    description:
      '本节点导出文件路径。写入请用 echo "k=v" >> "$FLOW_ENV_OUT" 或 >> {{FLOW_ENV_OUT}}；导出会合并进实例 env，下游节点可直接用 $k。',
  },
]
