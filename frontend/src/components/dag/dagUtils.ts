import type { Edge, Node } from '@xyflow/react'
import type { FlowNode, GraphEdgePayload, GraphNodePayload, K8sResources, NodeInstance } from '../../api/types'

/** K8s 扩展资源名：NVIDIA GPU（requests/limits 键）。 */
export const K8S_GPU_RESOURCE = 'nvidia.com/gpu'

const K8S_BUILTIN_RESOURCES = new Set(['cpu', 'memory', K8S_GPU_RESOURCE])

function trimResourceEntry(key: string, value?: string) {
  const k = key.trim()
  const v = (value ?? '').trim()
  if (!k || !v) return null
  return { k, v }
}

/** 将节点 k8s_resources 展开为表单字段。 */
export function k8sResourcesToFormFields(r?: K8sResources) {
  const req = r?.requests ?? {}
  const lim = r?.limits ?? {}
  const names = new Set([...Object.keys(req), ...Object.keys(lim)])
  const extra: { name: string; request?: string; limit?: string }[] = []
  for (const name of names) {
    if (K8S_BUILTIN_RESOURCES.has(name)) continue
    extra.push({ name, request: req[name], limit: lim[name] })
  }
  return {
    k8s_cpu_request: req.cpu ?? '',
    k8s_cpu_limit: lim.cpu ?? '',
    k8s_mem_request: req.memory ?? '',
    k8s_mem_limit: lim.memory ?? '',
    k8s_gpu_request: req[K8S_GPU_RESOURCE] ?? '',
    k8s_gpu_limit: lim[K8S_GPU_RESOURCE] ?? '',
    k8s_extra_resources: extra,
  }
}

/** 从表单字段组装 k8s_resources（空则返回 undefined 便于 JSON 精简）。 */
export function formFieldsToK8sResources(values: {
  k8s_cpu_request?: string
  k8s_cpu_limit?: string
  k8s_mem_request?: string
  k8s_mem_limit?: string
  k8s_gpu_request?: string
  k8s_gpu_limit?: string
  k8s_extra_resources?: { name?: string; request?: string; limit?: string }[]
}): K8sResources | undefined {
  const requests: Record<string, string> = {}
  const limits: Record<string, string> = {}
  const put = (target: Record<string, string>, key: string, val?: string) => {
    const e = trimResourceEntry(key, val)
    if (e) target[e.k] = e.v
  }
  put(requests, 'cpu', values.k8s_cpu_request)
  put(limits, 'cpu', values.k8s_cpu_limit)
  put(requests, 'memory', values.k8s_mem_request)
  put(limits, 'memory', values.k8s_mem_limit)
  put(requests, K8S_GPU_RESOURCE, values.k8s_gpu_request)
  put(limits, K8S_GPU_RESOURCE, values.k8s_gpu_limit)
  for (const row of values.k8s_extra_resources ?? []) {
    if (!row?.name?.trim()) continue
    put(requests, row.name, row.request)
    put(limits, row.name, row.limit)
  }
  if (Object.keys(requests).length === 0 && Object.keys(limits).length === 0) {
    return undefined
  }
  return { requests, limits }
}

const VISUAL_TYPES = new Set(['task', 'join', 'manual', 'start', 'end'])

function defaultNodePosition(index: number) {
  return { x: 120 + (index % 4) * 180, y: 80 + Math.floor(index / 4) * 120 }
}

function resolveNodePosition(node: FlowNode, index: number) {
  if (node.position_x != null && node.position_y != null) {
    return { x: node.position_x, y: node.position_y }
  }
  return defaultNodePosition(index)
}

function normalizeVisualType(type: FlowNode['type']): string {
  return type === 'join' ? 'task' : type
}

export function flowNodesToGraph(
  nodes: FlowNode[],
  options?: {
    readOnly?: boolean
    statusMap?: Map<number, NodeInstance>
  },
): { nodes: Node[]; edges: Edge[] } {
  const visualNodes = nodes.filter((node) => VISUAL_TYPES.has(node.type))
  const edgeNodes = nodes.filter((node) => node.type === 'edge')

  const rfNodes: Node[] = visualNodes.map((node, index) => ({
    id: String(node.id),
    type: normalizeVisualType(node.type),
    position: resolveNodePosition(node, index),
    data: {
      label:
        node.type === 'start' ? '开始' : node.type === 'end' ? '结束' : node.name,
      backendId: node.id,
      nodeType: normalizeVisualType(node.type),
      pre_script: node.pre_script,
      script: node.script,
      post_script: node.post_script,
      owners: node.owners ?? [],
      approvers: node.approvers ?? [],
      start_mode: node.start_mode || 'any',
      start_script: node.start_script ?? '',
      required_predecessor_ids: node.required_predecessor_ids ?? [],
      runtime_kind: node.runtime_kind || 'local',
      resource_id: node.resource_id ?? 0,
      k8s_auth_id: node.k8s_auth_id ?? 0,
      k8s_namespace: node.k8s_namespace ?? '',
      k8s_image: node.k8s_image ?? '',
      k8s_volumes: node.k8s_volumes ?? [],
      k8s_volume_mounts: node.k8s_volume_mounts ?? [],
      k8s_resources: node.k8s_resources,
    },
    deletable: node.type !== 'start' && node.type !== 'end',
  }))

  if (options?.readOnly && options.statusMap && options.statusMap.size > 0) {
    for (const node of rfNodes) {
      const backendId = node.data.backendId as number | undefined
      const status = backendId ? options.statusMap.get(backendId)?.status : undefined
      if (status) {
        node.data = { ...node.data, status }
      }
    }
  }

  const statusByNodeId = new Map<string, string>()
  if (options?.readOnly && options.statusMap) {
    for (const node of rfNodes) {
      const backendId = node.data.backendId as number | undefined
      const status = backendId ? options.statusMap.get(backendId)?.status : undefined
      if (status) {
        statusByNodeId.set(node.id, status)
      }
    }
  }

  const rfEdges: Edge[] = edgeNodes
    .filter((edge) => edge.from_node_id && edge.to_node_id)
    .map((edge) => {
      const source = String(edge.from_node_id)
      const target = String(edge.to_node_id)
      const edgeStyle =
        options?.readOnly && options.statusMap
          ? deriveEdgeStyle(statusByNodeId.get(source), statusByNodeId.get(target))
          : { stroke: '#d9d9d9', strokeWidth: 1 }
      return {
        id: String(edge.id),
        source,
        target,
        type: 'flow',
        style: edgeStyle,
        data: {
          backendId: edge.id,
        },
      }
    })

  // 只读实例视图：节点因附加信息变大，做一次去重叠布局（保持相对位置，向右排开）。
  const laidOutNodes = options?.readOnly ? relieveNodeOverlap(rfNodes) : rfNodes

  return { nodes: laidOutNodes, edges: rfEdges }
}

export function graphStateToPayload(nodes: Node[], edges: Edge[]): {
  nodes: GraphNodePayload[]
  edges: GraphEdgePayload[]
} {
  const payloadNodes: GraphNodePayload[] = nodes.map((node) => nodeConfigPayloadFromNode(node))

  const payloadEdges: GraphEdgePayload[] = edges.map((edge) => ({
    clientId: edge.id,
    backendId: edge.data?.backendId as number | undefined,
    source: edge.source,
    target: edge.target,
  }))

  return { nodes: payloadNodes, edges: payloadEdges }
}

function nodeConfigPayloadFromNode(node: Node): GraphNodePayload {
  const nodeType = (node.data.nodeType as GraphNodePayload['type']) ?? 'task'
  if (nodeType === 'start' || nodeType === 'end') {
    return {
      clientId: node.id,
      backendId: node.data.backendId as number | undefined,
      type: nodeType,
      name: nodeType === 'start' ? '开始' : '结束',
      pre_script: '',
      script: '',
      post_script: '',
      owners: [],
      description: '',
      position: node.position,
      start_mode: 'any',
      start_script: '',
      required_predecessor_ids: [],
      runtime_kind: 'local',
      resource_id: 0,
    }
  }
  return {
    clientId: node.id,
    backendId: node.data.backendId as number | undefined,
    type: (node.data.nodeType as GraphNodePayload['type']) ?? 'task',
    name: String(node.data.label ?? node.id),
    pre_script: String(node.data.pre_script ?? ''),
    script: String(node.data.script ?? ''),
    post_script: String(node.data.post_script ?? ''),
    owners: (node.data.owners as string[]) ?? [],
    description: String(node.data.description ?? ''),
    position: node.position,
    start_mode: (node.data.start_mode as string) || 'any',
    start_script: String(node.data.start_script ?? ''),
    required_predecessor_ids: (node.data.required_predecessor_ids as number[]) ?? [],
    runtime_kind: String(node.data.runtime_kind ?? 'local'),
    resource_id: Number(node.data.resource_id ?? 0),
    k8s_auth_id: Number(node.data.k8s_auth_id ?? 0),
    k8s_namespace: String(node.data.k8s_namespace ?? ''),
    k8s_image: String(node.data.k8s_image ?? ''),
    k8s_volumes: (node.data.k8s_volumes as GraphNodePayload['k8s_volumes']) ?? [],
    k8s_volume_mounts: (node.data.k8s_volume_mounts as GraphNodePayload['k8s_volume_mounts']) ?? [],
    k8s_resources: node.data.k8s_resources as GraphNodePayload['k8s_resources'],
  }
}

/** 拓扑自动保存：节点配置随画布当前内容一并持久化。 */
export function graphStateToTopologyPayload(
  nodes: Node[],
  edges: Edge[],
): { nodes: GraphNodePayload[]; edges: GraphEdgePayload[] } {
  const payloadNodes: GraphNodePayload[] = nodes.map((node) => nodeConfigPayloadFromNode(node))

  const payloadEdges: GraphEdgePayload[] = edges.map((edge) => ({
    clientId: edge.id,
    backendId: edge.data?.backendId as number | undefined,
    source: edge.source,
    target: edge.target,
  }))

  return { nodes: payloadNodes, edges: payloadEdges }
}

/**
 * 连通性校验：除开始/结束节点外，每个普通节点都必须有上游和下游连接。
 * 返回问题描述列表，为空表示通过。
 */
export function findConnectivityIssues(
  nodes: Array<{ id: string | number; type: string; name: string }>,
  edges: Array<{ source: string | number; target: string | number }>,
): string[] {
  const issues: string[] = []
  for (const node of nodes) {
    if (node.type === 'start' || node.type === 'end') continue
    const hasIncoming = edges.some((edge) => edge.target === node.id)
    const hasOutgoing = edges.some((edge) => edge.source === node.id)
    if (!hasIncoming || !hasOutgoing) {
      const missing = [
        ...(!hasIncoming ? ['上游'] : []),
        ...(!hasOutgoing ? ['下游'] : []),
      ]
      issues.push(`节点「${node.name}」缺少${missing.join('和')}连接`)
    }
  }
  return issues
}

/**
 * 拓扑自动保存后原地合并后端 ID。
 * 后端 SaveGraph 是删除重建，返回顺序与请求顺序一致（先非边节点、后边节点），
 * 因此按索引把新 ID 写回现有对象即可；保持对象身份不变，避免画布全量重渲染闪烁。
 */
export function mergeBackendIdsAfterTopologySave(
  currentNodes: Node[],
  currentEdges: Edge[],
  saved: FlowNode[],
): { nodes: Node[]; edges: Edge[] } {
  const savedNodes = saved.filter((item) => item.type !== 'edge')
  const savedEdges = saved.filter((item) => item.type === 'edge')

  if (savedNodes.length !== currentNodes.length || savedEdges.length !== currentEdges.length) {
    return { nodes: currentNodes, edges: currentEdges }
  }

  const nodes = currentNodes.map((node, index) => {
    const savedNode = savedNodes[index]
    const backendId = savedNode?.id
    if (!backendId) {
      return node
    }
    const savedPreds = savedNode.required_predecessor_ids ?? []
    const currentPreds = (node.data.required_predecessor_ids as number[]) ?? []
    const predsEqual =
      savedPreds.length === currentPreds.length && savedPreds.every((id, i) => id === currentPreds[i])
    if ((node.data.backendId as number | undefined) === backendId && predsEqual) {
      return node
    }
    return {
      ...node,
      data: {
        ...node.data,
        backendId,
        required_predecessor_ids: savedPreds,
      },
    }
  })

  const edges = currentEdges.map((edge, index) => {
    const backendId = savedEdges[index]?.id
    if (!backendId || (edge.data?.backendId as number | undefined) === backendId) {
      return edge
    }
    return { ...edge, data: { ...edge.data, backendId } }
  })

  return { nodes, edges }
}

export function buildNodeStatusMap(nodeInstances: NodeInstance[] = []): Map<number, NodeInstance> {
  return new Map(nodeInstances.map((item) => [item.node_id, item]))
}

// 只读实例视图的节点估算尺寸：节点上额外展示负责人/审批人与运行时间，
// 宽度以时间行（两个日期时间）为最大项，高度按名称+人员+时间三行估算。
const INSTANCE_NODE_WIDTH = 270
const INSTANCE_NODE_HEIGHT = 100

/**
 * 只读实例视图的去重叠布局：节点信息加高加宽后，按原保存位置可能相互覆盖。
 * 保持 y（基本相对位置）不变，将发生覆盖的节点沿 x 轴向右排开，
 * 多轮迭代直到稳定，保证结果两两不重叠。
 */
export function relieveNodeOverlap(nodes: Node[], gap = 24): Node[] {
  const placed = nodes
    .map((node) => ({ node, x: node.position.x, y: node.position.y }))
    .sort((a, b) => a.x - b.x || a.y - b.y)

  const overlaps = (a: { x: number; y: number }, b: { x: number; y: number }) =>
    a.x < b.x + INSTANCE_NODE_WIDTH + gap &&
    b.x < a.x + INSTANCE_NODE_WIDTH + gap &&
    a.y < b.y + INSTANCE_NODE_HEIGHT + 12 &&
    b.y < a.y + INSTANCE_NODE_HEIGHT + 12

  for (let pass = 0; pass < placed.length; pass += 1) {
    let moved = false
    for (let i = 1; i < placed.length; i += 1) {
      for (let j = 0; j < i; j += 1) {
        if (overlaps(placed[j], placed[i])) {
          // 向右排到前一个节点右侧
          placed[i].x = placed[j].x + INSTANCE_NODE_WIDTH + gap
          moved = true
        }
      }
    }
    if (!moved) break
  }

  return placed.map(({ node, x, y }) => ({ ...node, position: { x, y } }))
}

export function nodeBorderColor(status?: string): string {
  switch (status) {
    case 'running':
      return '#1677ff'
    case 'succeeded':
      return '#52c41a'
    case 'failed':
      return '#ff4d4f'
    case 'waiting_approval':
      return '#fa8c16'
    case 'failed_skipped':
      return '#bfbfbf'
    case 'skipped':
      return '#8c8c8c'
    default:
      return '#d9d9d9'
  }
}

export function deriveEdgeStyle(
  sourceStatus?: string,
  targetStatus?: string,
): { stroke: string; strokeWidth: number; strokeDasharray?: string } {
  if (!sourceStatus || !targetStatus) {
    return { stroke: '#d9d9d9', strokeWidth: 1 }
  }

  if (targetStatus === 'failed_skipped') {
    return { stroke: '#d9d9d9', strokeWidth: 1, strokeDasharray: '6 4' }
  }

  if (targetStatus === 'failed') {
    return { stroke: '#ff4d4f', strokeWidth: 1.5, strokeDasharray: '6 4' }
  }

  if (sourceStatus === 'succeeded' && (targetStatus === 'running' || targetStatus === 'waiting_approval')) {
    return { stroke: '#1677ff', strokeWidth: 1.5 }
  }

  if (sourceStatus === 'succeeded' && targetStatus === 'succeeded') {
    return { stroke: '#52c41a', strokeWidth: 1.5 }
  }

  if (sourceStatus === 'succeeded' && targetStatus === 'pending') {
    return { stroke: '#d9d9d9', strokeWidth: 1 }
  }

  return { stroke: '#d9d9d9', strokeWidth: 1 }
}

let tempId = 0
export function nextTempId(prefix: string): string {
  tempId += 1
  return `${prefix}-${Date.now()}-${tempId}`
}
