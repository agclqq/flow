import {
  BranchesOutlined,
  CodeOutlined,
  FlagOutlined,
  PlayCircleOutlined,
  UserOutlined,
} from '@ant-design/icons'
import { Alert, Button, Card, Drawer, Form, Input, Modal, Select, Space, Spin, Typography, message } from 'antd'
import {
  Background,
  Controls,
  MiniMap,
  ReactFlow,
  ReactFlowProvider,
  addEdge,
  useEdgesState,
  useNodesState,
  useReactFlow,
  type Connection,
  type Edge,
  type Node,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import './dag.css'
import {
  forwardRef,
  lazy,
  Suspense,
  useCallback,
  useEffect,
  useImperativeHandle,
  useMemo,
  useRef,
  useState,
} from 'react'
import type { ExecutorResource, FlowNode as BackendFlowNode, NodeInstance, RuntimeKind, StartMode } from '../../api/types'
import { createExecutorResource, listExecutorResources } from '../../api/executor-resources'
import { listNodes, putGraph } from '../../api/templates'
import { DagNodeContext } from './dagNodeContext'
import { flowEdgeTypes } from './FlowEdge'
import { flowNodeTypes } from './FlowNode'
import ResourceModal from './ResourceModal'
import {
  buildNodeStatusMap,
  deriveEdgeStyle,
  findConnectivityIssues,
  flowNodesToGraph,
  graphStateToPayload,
  graphStateToTopologyPayload,
  mergeBackendIdsAfterTopologySave,
  nextTempId,
  nodeBorderColor,
} from './dagUtils'

const ScriptEditor = lazy(() => import('./ScriptEditor'))

function LazyScriptEditor(props: {
  value?: string
  onChange?: (value: string) => void
  readOnly?: boolean
  height?: number
  paramNames?: string[]
}) {
  return (
    <Suspense fallback={<Spin size="small" />}>
      <ScriptEditor {...props} />
    </Suspense>
  )
}

const EMPTY_NODE_INSTANCES: NodeInstance[] = []

const palette = [
  { type: 'task', label: '任务节点', icon: <CodeOutlined style={{ color: '#1677ff' }} /> },
  { type: 'manual', label: '人工节点', icon: <UserOutlined style={{ color: '#fa8c16' }} /> },
  { type: 'start', label: '开始节点', icon: <PlayCircleOutlined style={{ color: '#52c41a' }} /> },
  { type: 'end', label: '结束节点', icon: <FlagOutlined style={{ color: '#ff4d4f' }} /> },
]

/** 节点初始数据：开始/结束节点只有类型与名称，其余为任务默认值。 */
const buildNodeData = (nodeType: string): Record<string, unknown> => {
  if (nodeType === 'start' || nodeType === 'end') {
    return { label: nodeType === 'start' ? '开始' : '结束', nodeType }
  }
  return {
    label: nodeType === 'manual' ? '人工节点' : '任务节点',
    nodeType,
    pre_script: '',
    script: '',
    post_script: '',
    owners: [],
    start_mode: 'any',
    start_script: '',
    required_predecessor_ids: [],
    runtime_kind: 'local',
    resource_id: 0,
  }
}

const START_MODE_OPTIONS = [
  { value: 'any', label: '任一前驱完成 (any)' },
  { value: 'all', label: '所有前驱完成 (all)' },
  { value: 'custom', label: '自定义 (custom)' },
]

type DagEditorProps = {
  templateId?: number
  versionId?: number
  editable?: boolean
  readOnly?: boolean
  /** 当前版本是否可写（已发布版本为 false：跳过自动保存，改动随「保存」写入新草稿） */
  writable?: boolean
  paramNames?: string[]
  nodeInstances?: NodeInstance[]
  prefetchedNodes?: BackendFlowNode[] | null
  prefetchedKey?: string
  onNodeInstanceClick?: (nodeInstance: NodeInstance) => void
  /** 实例视图：在节点上直接审批通过 / 拒绝 */
  onApproveNode?: (nodeId: number) => void
  onRejectNode?: (nodeId: number) => void
}

export type DagEditorHandle = {
  saveNodeConfigs: (targetVersionId: number) => Promise<void>
  /** 校验画布连通性，返回问题描述列表（空数组表示通过） */
  validateConnectivity: () => string[]
}

const DagEditorInner = forwardRef<DagEditorHandle, DagEditorProps>(function DagEditorInner(
  {
    templateId,
    versionId,
    editable = false,
    readOnly = false,
    writable = true,
    paramNames = [],
    nodeInstances: nodeInstancesProp,
    prefetchedNodes,
    prefetchedKey,
    onNodeInstanceClick,
    onApproveNode,
    onRejectNode,
  },
  ref,
) {
  const nodeInstances = nodeInstancesProp ?? EMPTY_NODE_INSTANCES
  const { screenToFlowPosition, fitView } = useReactFlow()
  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([])
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([])
  const [selectedNode, setSelectedNode] = useState<Node | null>(null)
  const [selectedNodeInstance, setSelectedNodeInstance] = useState<NodeInstance | null>(null)
  const [executorResources, setExecutorResources] = useState<ExecutorResource[]>([])
  const [resourceModalOpen, setResourceModalOpen] = useState(false)
  const [, setBackendNodes] = useState<BackendFlowNode[]>([])
  const [loading, setLoading] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)
  const [pendingAdd, setPendingAdd] = useState<{
    kind: 'node' | 'edge'
    id: string
    /** 触发「+」时的鼠标位置，用于就近弹出节点类型菜单 */
    pos?: { x: number; y: number }
  } | null>(null)
  const [connectingNodeId, setConnectingNodeId] = useState<string | null>(null)
  const nodesRef = useRef(nodes)
  const edgesRef = useRef(edges)
  const flowWrapperRef = useRef<HTMLDivElement | null>(null)
  const topologySaveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const topologySavingRef = useRef(false)
  nodesRef.current = nodes
  edgesRef.current = edges
  // 画布高度填满当前窗口剩余空间（仅编辑模式自适应，只读画布保持固定高度）。
  const [flowHeight, setFlowHeight] = useState(520)
  useEffect(() => {
    if (!editable) return
    const measure = () => {
      const el = flowWrapperRef.current
      if (!el) return
      const top = el.getBoundingClientRect().top
      setFlowHeight(Math.max(360, window.innerHeight - top - 24))
    }
    measure()
    window.addEventListener('resize', measure)
    const observer = new ResizeObserver(measure)
    observer.observe(document.body)
    return () => {
      window.removeEventListener('resize', measure)
      observer.disconnect()
    }
  }, [editable, saveError])
  const [nodeForm] = Form.useForm()
  const startMode = Form.useWatch('start_mode', nodeForm) as StartMode | undefined
  const runtimeKind = (Form.useWatch('runtime_kind', nodeForm) ?? 'local') as RuntimeKind | string

  const statusMap = useMemo(() => buildNodeStatusMap(nodeInstances), [nodeInstances])

  const predecessorOptions = useMemo(() => {
    if (!selectedNode) return []
    return nodes
      .filter(
        (node) =>
          node.id !== selectedNode.id &&
          node.data.nodeType !== 'start' &&
          node.data.nodeType !== 'end' &&
          edges.some((edge) => edge.target === selectedNode.id && edge.source === node.id),
      )
      .map((node) => ({
        value: node.data.backendId as number,
        label: String(node.data.label ?? node.id),
      }))
      .filter((item) => typeof item.value === 'number')
  }, [edges, nodes, selectedNode])

  const applyBackendNodes = useCallback(
    (items: BackendFlowNode[]) => {
      setBackendNodes(items)
      // 不在此处注入 statusMap：节点/边状态由下方合并 effect 就地更新，
      // 避免 statusMap 变化导致整图重建与节点定义重新拉取（轮询时闪烁）。
      const graph = flowNodesToGraph(items, { readOnly })
      setNodes(graph.nodes)
      setEdges(graph.edges)
      // 已有图加载完成后视角适配一次（不超过 100%，避免小图被放大）；
      // 空画布不触发，放入第一个节点时也就不会再自动缩放。
      if (graph.nodes.length > 0) {
        // 双 rAF 确保节点尺寸测量完成后再适配；minZoom 放宽到 0.05，
        // 节点相距很远时也能缩到足够小看到全部（maxZoom 封顶 100% 避免小图被放大）。
        requestAnimationFrame(() =>
          requestAnimationFrame(() => fitView({ padding: 0.3, minZoom: 0.05, maxZoom: 1 })),
        )
      }
    },
    [fitView, readOnly, setEdges, setNodes],
  )

  useEffect(() => {
    let cancelled = false
    void (async () => {
      if (!templateId || !versionId) {
        setLoading(false)
        return
      }
      const cacheKey = `${templateId}:${versionId}`
      if (prefetchedNodes && prefetchedKey === cacheKey) {
        if (!cancelled) {
          applyBackendNodes(prefetchedNodes)
          setLoading(false)
        }
        return
      }
      setLoading(true)
      try {
        const items = await listNodes(templateId, versionId)
        if (cancelled) return
        applyBackendNodes(items)
      } catch (error) {
        if (!cancelled) {
          message.error(error instanceof Error ? error.message : '加载 DAG 失败')
        }
      } finally {
        setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [applyBackendNodes, prefetchedKey, prefetchedNodes, templateId, versionId])

  useEffect(() => {
    return () => {
      if (topologySaveTimerRef.current) {
        clearTimeout(topologySaveTimerRef.current)
      }
    }
  }, [])

  // 同步节点的 hasOutgoing 标记：无出边的普通节点/开始节点显示延伸线 + 添加按钮。
  useEffect(() => {
    setNodes((current) => {
      let changed = false
      const next = current.map((node) => {
        const hasOutgoing = edges.some((edge) => edge.source === node.id)
        if (node.data.hasOutgoing === hasOutgoing) return node
        changed = true
        return { ...node, data: { ...node.data, hasOutgoing } }
      })
      return changed ? next : current
    })
  }, [edges, setNodes])

  // 只读实例视图：实例状态变化时就地合并节点/边样式，避免整图重建闪烁。
  useEffect(() => {
    if (!readOnly) return
    const infoByNodeId = new Map<string, NodeInstance | undefined>()
    for (const node of nodesRef.current) {
      const backendId = node.data.backendId as number | undefined
      infoByNodeId.set(node.id, backendId ? statusMap.get(backendId) : undefined)
    }
    setNodes((current) => {
      let changed = false
      const next = current.map((node) => {
        const info = infoByNodeId.get(node.id)
        const status = info?.status
        // 最近一次运行的起止时间（重试会有多段，取最后一段）。
        const spans = info?.run_times ?? []
        const last = spans.length > 0 ? spans[spans.length - 1] : undefined
        const lastStart = last?.start_at
        const lastEnd = last?.end_at
        if (
          node.data.status === status &&
          node.data.last_start_at === lastStart &&
          node.data.last_end_at === lastEnd
        ) {
          return node
        }
        changed = true
        return {
          ...node,
          data: { ...node.data, status, last_start_at: lastStart, last_end_at: lastEnd },
        }
      })
      return changed ? next : current
    })
    setEdges((current) => {
      let changed = false
      const next = current.map((edge) => {
        const style = deriveEdgeStyle(
          infoByNodeId.get(edge.source)?.status,
          infoByNodeId.get(edge.target)?.status,
        )
        if (
          edge.style?.stroke === style.stroke &&
          edge.style?.strokeWidth === style.strokeWidth &&
          edge.style?.strokeDasharray === style.strokeDasharray
        ) {
          return edge
        }
        changed = true
        return { ...edge, style }
      })
      return changed ? next : current
    })
  }, [nodes, readOnly, statusMap, setEdges, setNodes])

  // Esc 取消“待添加”模式。
  useEffect(() => {
    if (!pendingAdd) return
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setPendingAdd(null)
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [pendingAdd])

  // 就近菜单打开时，点击菜单外部仅关闭菜单（保留待添加模式，可继续从左侧面板选择）。
  useEffect(() => {
    if (!pendingAdd?.pos) return
    const onMouseDown = (event: MouseEvent) => {
      const target = event.target as HTMLElement | null
      if (target?.closest('[data-add-node-menu]')) return
      setPendingAdd((current) => (current ? { kind: current.kind, id: current.id } : null))
    }
    window.addEventListener('mousedown', onMouseDown, true)
    return () => window.removeEventListener('mousedown', onMouseDown, true)
  }, [pendingAdd?.pos])

  const persistTopology = useCallback(
    async (targetVersionId: number) => {
      if (!templateId || readOnly || !editable || !writable) return
      if (topologySavingRef.current) return
      topologySavingRef.current = true
      try {
        const payload = graphStateToTopologyPayload(nodesRef.current, edgesRef.current)
        const saved = await putGraph(templateId, targetVersionId, payload)
        const merged = mergeBackendIdsAfterTopologySave(nodesRef.current, edgesRef.current, saved)
        setNodes(merged.nodes)
        setEdges(merged.edges)
        setBackendNodes(saved)
        setSaveError(null)
      } catch (error) {
        setSaveError(error instanceof Error ? error.message : '保存编排失败')
      } finally {
        topologySavingRef.current = false
      }
    },
    [editable, readOnly, setEdges, setNodes, templateId, writable],
  )

  const scheduleTopologySave = useCallback(() => {
    if (!versionId || readOnly || !editable || !writable) return
    if (topologySaveTimerRef.current) {
      clearTimeout(topologySaveTimerRef.current)
    }
    topologySaveTimerRef.current = setTimeout(() => {
      void persistTopology(versionId)
    }, 800)
  }, [editable, persistTopology, readOnly, versionId, writable])

  const onConnect = useCallback(
    (connection: Connection) => {
      if (readOnly) return
      setEdges((current) =>
        addEdge(
          {
            ...connection,
            id: nextTempId('edge'),
            type: 'flow',
            style: { stroke: '#d9d9d9', strokeWidth: 1 },
          },
          current,
        ),
      )
      scheduleTopologySave()
    },
    [readOnly, scheduleTopologySave, setEdges],
  )

  const onNodesChangeWrapped = useCallback(
    (changes: Parameters<typeof onNodesChange>[0]) => {
      onNodesChange(changes)
      if (!editable || readOnly || !versionId) return
      const topologyChanged = changes.some(
        (change) => change.type === 'position' || change.type === 'remove' || change.type === 'add',
      )
      if (topologyChanged) {
        scheduleTopologySave()
      }
    },
    [editable, onNodesChange, readOnly, scheduleTopologySave, versionId],
  )

  const onEdgesChangeWrapped = useCallback(
    (changes: Parameters<typeof onEdgesChange>[0]) => {
      onEdgesChange(changes)
      if (!editable || readOnly || !versionId) return
      const topologyChanged = changes.some((change) => change.type === 'remove' || change.type === 'add')
      if (topologyChanged) {
        scheduleTopologySave()
      }
    },
    [editable, onEdgesChange, readOnly, scheduleTopologySave, versionId],
  )

  const onDragStart = (event: React.DragEvent, nodeType: string) => {
    event.dataTransfer.setData('application/reactflow', nodeType)
    event.dataTransfer.effectAllowed = 'move'
  }

  /** 开始/结束节点画布内唯一，重复添加给出提示。 */
  const ensureTerminalAvailable = useCallback(
    (nodeType: string) => {
      if (nodeType !== 'start' && nodeType !== 'end') return true
      const exists = nodesRef.current.some(
        (node) => (node.data as { nodeType?: string }).nodeType === nodeType,
      )
      if (exists) {
        message.warning(`画布上已存在「${nodeType === 'start' ? '开始' : '结束'}」节点`)
      }
      return !exists
    },
    [],
  )

  const onDrop = useCallback(
    (event: React.DragEvent) => {
      event.preventDefault()
      if (readOnly || !editable) return
      const nodeType = event.dataTransfer.getData('application/reactflow')
      if (!nodeType) return
      if (!ensureTerminalAvailable(nodeType)) return
      const position = screenToFlowPosition({ x: event.clientX, y: event.clientY })
      setNodes((current) => [
        ...current,
        {
          id: nextTempId('node'),
          type: nodeType,
          position,
          data: buildNodeData(nodeType),
        },
      ])
      scheduleTopologySave()
    },
    [editable, ensureTerminalAvailable, readOnly, scheduleTopologySave, screenToFlowPosition, setNodes],
  )

  // 点击 palette 的开始/结束节点：直接添加到画布中心。
  const addTerminalFromPalette = useCallback(
    (nodeType: 'start' | 'end') => {
      if (readOnly || !editable) return
      if (!ensureTerminalAvailable(nodeType)) return
      const rect = flowWrapperRef.current?.getBoundingClientRect()
      const position = rect
        ? screenToFlowPosition({ x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 })
        : { x: 0, y: 0 }
      setNodes((current) => [
        ...current,
        { id: nextTempId('node'), type: nodeType, position, data: buildNodeData(nodeType) },
      ])
      scheduleTopologySave()
    },
    [editable, ensureTerminalAvailable, readOnly, scheduleTopologySave, screenToFlowPosition, setNodes],
  )

  const onDragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault()
    event.dataTransfer.dropEffect = 'move'
  }, [])

  const handleDeleteNode = useCallback(
    (nodeId: string, nodeName: string) => {
      if (readOnly || !editable) return
      Modal.confirm({
        title: `确认是否要删除${nodeName}节点？`,
        okText: '删除',
        okType: 'danger',
        cancelText: '取消',
        onOk: () => {
          setNodes((current) => current.filter((node) => node.id !== nodeId))
          setEdges((current) =>
            current.filter((edge) => edge.source !== nodeId && edge.target !== nodeId),
          )
          setSelectedNode((current) => (current?.id === nodeId ? null : current))
          scheduleTopologySave()
        },
      })
    },
    [editable, readOnly, scheduleTopologySave, setEdges, setNodes],
  )

  const handleAddFromNode = useCallback((nodeId: string, pos?: { x: number; y: number }) => {
    setPendingAdd((current) =>
      current?.kind === 'node' && current.id === nodeId ? null : { kind: 'node', id: nodeId, pos },
    )
  }, [])

  const handleAddFromEdge = useCallback((edgeId: string, pos?: { x: number; y: number }) => {
    setPendingAdd((current) =>
      current?.kind === 'edge' && current.id === edgeId ? null : { kind: 'edge', id: edgeId, pos },
    )
  }, [])

  const handleConnectStart = useCallback(
    (_: unknown, params: { nodeId: string | null; handleType: string }) => {
      if (params.handleType === 'source' && params.nodeId) {
        setConnectingNodeId(params.nodeId)
      }
    },
    [],
  )

  const handleConnectEnd = useCallback(() => {
    setConnectingNodeId(null)
  }, [])

  // 悬停时把边临时提到最上层（高于选中节点的 1000），避免被节点遮挡；离开后恢复。
  const handleEdgeMouseEnter = useCallback(
    (_: React.MouseEvent, edge: Edge) => {
      setEdges((current) =>
        current.map((item) => (item.id === edge.id && !item.zIndex ? { ...item, zIndex: 1005 } : item)),
      )
    },
    [setEdges],
  )

  const handleEdgeMouseLeave = useCallback(
    (_: React.MouseEvent, edge: Edge) => {
      setEdges((current) =>
        current.map((item) => (item.id === edge.id && item.zIndex ? { ...item, zIndex: undefined } : item)),
      )
    },
    [setEdges],
  )

  const handleDeleteEdge = useCallback(
    (edgeId: string) => {
      if (readOnly || !editable) return
      const edge = edgesRef.current.find((item) => item.id === edgeId)
      if (!edge) return
      const sourceName = nodesRef.current.find((node) => node.id === edge.source)?.data.label
      const targetName = nodesRef.current.find((node) => node.id === edge.target)?.data.label
      Modal.confirm({
        title: `确认删除「${sourceName ?? edge.source} → ${targetName ?? edge.target}」的连线？`,
        okText: '删除',
        okType: 'danger',
        cancelText: '取消',
        onOk: () => {
          setEdges((current) => current.filter((item) => item.id !== edgeId))
          scheduleTopologySave()
        },
      })
    },
    [editable, readOnly, scheduleTopologySave, setEdges],
  )

  const dagNodeContextValue = useMemo(
    () => ({
      editable: editable && !readOnly,
      onDeleteNode: handleDeleteNode,
      onDeleteEdge: editable && !readOnly ? handleDeleteEdge : undefined,
      onAddFromNode: editable && !readOnly ? handleAddFromNode : undefined,
      onAddFromEdge: editable && !readOnly ? handleAddFromEdge : undefined,
      connectingNodeId,
      onApproveNode,
      onRejectNode,
    }),
    [
      connectingNodeId,
      editable,
      handleAddFromEdge,
      handleAddFromNode,
      handleDeleteEdge,
      handleDeleteNode,
      onApproveNode,
      onRejectNode,
      readOnly,
    ],
  )

  const openNodeDrawer = useCallback(
    (node: Node) => {
      const nodeType = node.data.nodeType as string | undefined
      const backendId = node.data.backendId as number | undefined
      // 只读实例视图：不打开配置抽屉，把运行实例交给外层（运行详情）展示。
      if (readOnly) {
        if (backendId && statusMap.has(backendId)) {
          onNodeInstanceClick?.(statusMap.get(backendId)!)
        }
        return
      }
      if (nodeType === 'start' || nodeType === 'end') {
        if (backendId && statusMap.has(backendId)) {
          onNodeInstanceClick?.(statusMap.get(backendId)!)
        }
        return
      }
      setSelectedNode(node)
      void listExecutorResources()
        .then(setExecutorResources)
        .catch(() => setExecutorResources([]))
      nodeForm.setFieldsValue({
        name: node.data.label,
        description: node.data.description ?? '',
        owners: node.data.owners ?? [],
        pre_script: node.data.pre_script ?? '',
        script: node.data.script ?? '',
        post_script: node.data.post_script ?? '',
        start_mode: node.data.start_mode ?? 'any',
        start_script: node.data.start_script ?? '',
        required_predecessor_ids: node.data.required_predecessor_ids ?? [],
        runtime_kind: node.data.runtime_kind ?? 'local',
        resource_id: node.data.resource_id || undefined,
      })
      if (backendId && statusMap.has(backendId)) {
        const instance = statusMap.get(backendId)!
        setSelectedNodeInstance(instance)
        onNodeInstanceClick?.(instance)
      } else {
        setSelectedNodeInstance(null)
      }
    },
    [nodeForm, onNodeInstanceClick, readOnly, statusMap],
  )

  // 待添加模式下从左侧 palette 选择节点类型：创建节点并连线，随后弹出编辑抽屉。
  // 结束节点也允许通过「+」连接在后继位置（开始节点不行，需从画布直接添加）。
  const addNodeFromPalette = useCallback(
    (nodeType: string) => {
      if (!pendingAdd || readOnly || !editable) return
      if (nodeType !== 'task' && nodeType !== 'manual' && nodeType !== 'end') return
      if (!ensureTerminalAvailable(nodeType)) return
      const nodes = nodesRef.current
      const edges = edgesRef.current
      const newEdgeStyle = { stroke: '#d9d9d9', strokeWidth: 1 }
      const buildNode = (position: { x: number; y: number }): Node => ({
        id: nextTempId('node'),
        type: nodeType,
        position,
        data: buildNodeData(nodeType),
      })

      let newNode: Node
      if (pendingAdd.kind === 'node') {
        const source = nodes.find((node) => node.id === pendingAdd.id)
        if (!source) return
        newNode = buildNode({ x: source.position.x + 220, y: source.position.y })
        setNodes((current) => [...current, newNode])
        setEdges((current) => [
          ...current,
          {
            id: nextTempId('edge'),
            source: source.id,
            target: newNode.id,
            type: 'flow',
            style: newEdgeStyle,
          },
        ])
      } else {
        const edge = edges.find((item) => item.id === pendingAdd.id)
        const sourceNode = nodes.find((node) => node.id === edge?.source)
        const targetNode = nodes.find((node) => node.id === edge?.target)
        if (!edge || !sourceNode || !targetNode) return
        newNode = buildNode({
          x: (sourceNode.position.x + targetNode.position.x) / 2 - 60,
          y: (sourceNode.position.y + targetNode.position.y) / 2 - 20,
        })
        setNodes((current) => [...current, newNode])
        setEdges((current) =>
          current.flatMap((item) =>
            item.id === edge.id
              ? [
                  {
                    id: nextTempId('edge'),
                    source: edge.source,
                    target: newNode.id,
                    type: 'flow',
                    style: newEdgeStyle,
                  },
                  {
                    id: nextTempId('edge'),
                    source: newNode.id,
                    target: edge.target,
                    type: 'flow',
                    style: newEdgeStyle,
                  },
                ]
              : [item],
          ),
        )
      }

      setPendingAdd(null)
      // 结束节点无编辑内容，不弹抽屉。
      if (nodeType !== 'end') {
        openNodeDrawer(newNode)
      }
      scheduleTopologySave()
    },
    [editable, ensureTerminalAvailable, openNodeDrawer, pendingAdd, readOnly, scheduleTopologySave, setEdges, setNodes],
  )

  const saveNodeForm = async () => {
    if (!selectedNode) return
    try {
      const values = await nodeForm.validateFields()
      setNodes((current) =>
        current.map((node) =>
          node.id === selectedNode.id
            ? {
                ...node,
                data: {
                  ...node.data,
                  label: values.name,
                  description: values.description,
                  owners: values.owners,
                  pre_script: values.pre_script,
                  script: values.script,
                  post_script: values.post_script,
                  start_mode: values.start_mode,
                  start_script: values.start_script,
                  required_predecessor_ids: values.required_predecessor_ids ?? [],
                  runtime_kind: values.runtime_kind ?? 'local',
                  resource_id: values.resource_id ?? 0,
                },
              }
            : node,
        ),
      )
      setSelectedNode(null)
      // 节点配置随拓扑自动保存持久化，刷新页面不丢失。
      scheduleTopologySave()
    } catch {
      // 表单校验失败时 antd 会抛出错误对象，无需额外提示。
    }
  }

  const saveNodeConfigs = async (targetVersionId: number) => {
    if (!templateId) {
      throw new Error('请先保存模板基本信息')
    }
    if (topologySaveTimerRef.current) {
      clearTimeout(topologySaveTimerRef.current)
      topologySaveTimerRef.current = null
    }
    while (topologySavingRef.current) {
      await new Promise((resolve) => setTimeout(resolve, 50))
    }
    try {
      const payload = graphStateToPayload(nodesRef.current, edgesRef.current)
      const saved = await putGraph(templateId, targetVersionId, payload)
      setBackendNodes(saved)
      const graph = flowNodesToGraph(saved)
      setNodes(graph.nodes)
      setEdges(graph.edges)
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存节点配置失败')
      throw error
    }
  }

  const validateConnectivity = useCallback(
    (): string[] =>
      findConnectivityIssues(
        nodesRef.current.map((node) => ({
          id: node.id,
          type: String(node.data.nodeType ?? ''),
          name: String(node.data.label ?? node.id),
        })),
        edgesRef.current.map((edge) => ({ source: edge.source, target: edge.target })),
      ),
    [],
  )

  useImperativeHandle(ref, () => ({
    saveNodeConfigs,
    validateConnectivity,
  }))

  const isExecutableNode =
    selectedNode?.data.nodeType === 'task' || selectedNode?.data.nodeType === 'manual'

  return (
    <div style={{ display: 'grid', gridTemplateColumns: editable ? '220px 1fr' : '1fr', gap: 16 }}>
      {editable && (
        <Card title="节点类型" size="small">
          <Space orientation="vertical" size={8} style={{ width: '100%' }}>
            {palette.map((item) => {
              // 开始/结束节点画布内唯一：已存在则禁用该选项。
              const disabled =
                (item.type === 'start' || item.type === 'end') &&
                nodes.some((node) => (node.data as { nodeType?: string }).nodeType === item.type)
              return (
                <div
                  key={item.type}
                  draggable={!disabled}
                  onDragStart={(event) => {
                    if (disabled) {
                      event.preventDefault()
                      return
                    }
                    onDragStart(event, item.type)
                  }}
                  onClick={() => {
                    if (disabled) return
                    if (item.type === 'start') {
                      addTerminalFromPalette('start')
                    } else if (item.type === 'end') {
                      // 「+」待添加模式下选择结束节点：连接到目标节点/边；否则直接放到画布中心。
                      if (pendingAdd) {
                        addNodeFromPalette('end')
                      } else {
                        addTerminalFromPalette('end')
                      }
                    } else {
                      addNodeFromPalette(item.type)
                    }
                  }}
                  title={disabled ? `画布中已存在「${item.label.replace('节点', '')}」节点` : undefined}
                  style={{
                    cursor: disabled ? 'not-allowed' : 'grab',
                    padding: '4px 8px',
                    borderRadius: 6,
                    border: pendingAdd ? '1px dashed #1677ff' : '1px dashed transparent',
                    background: pendingAdd ? '#f0f5ff' : undefined,
                    opacity: disabled ? 0.45 : 1,
                  }}
                >
                  <Space>
                    {item.icon}
                    <span>{item.label}</span>
                    {disabled && (
                      <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                        已存在
                      </Typography.Text>
                    )}
                  </Space>
                </div>
              )
            })}
          </Space>
          {pendingAdd && (
            <Typography.Paragraph type="warning" style={{ marginTop: 12, marginBottom: 0 }}>
              请选择要添加的节点类型（Esc 取消）
            </Typography.Paragraph>
          )}
          <Typography.Paragraph type="secondary" style={{ marginTop: 12, marginBottom: 0 }}>
            拖拽节点到画布；连线表示拓扑，启动条件在节点上配置。
          </Typography.Paragraph>
        </Card>
      )}

      <div>
        {editable && saveError && (
          <Alert
            type="error"
            showIcon
            closable
            message="编排自动保存失败"
            description={saveError}
            onClose={() => setSaveError(null)}
            style={{ marginBottom: 8 }}
          />
        )}
        <div
          ref={flowWrapperRef}
          style={{ height: flowHeight, border: '1px solid #f0f0f0', borderRadius: 8, position: 'relative' }}
          onDrop={onDrop}
          onDragOver={onDragOver}
        >
          <DagNodeContext.Provider value={dagNodeContextValue}>
            <ReactFlow
              nodes={nodes}
              edges={edges}
              nodeTypes={flowNodeTypes}
              edgeTypes={flowEdgeTypes}
              deleteKeyCode={['Backspace', 'Delete']}
              onNodesChange={readOnly ? undefined : onNodesChangeWrapped}
              onEdgesChange={readOnly ? undefined : onEdgesChangeWrapped}
              onConnect={readOnly ? undefined : onConnect}
              onConnectStart={readOnly ? undefined : handleConnectStart}
              onConnectEnd={readOnly ? undefined : handleConnectEnd}
              onEdgeMouseEnter={handleEdgeMouseEnter}
              onEdgeMouseLeave={handleEdgeMouseLeave}
              onNodeClick={(_, node) => openNodeDrawer(node)}
              panActivationKeyCode={null}
              disableKeyboardA11y={!!selectedNode}
              nodesFocusable={!selectedNode}
              nodesDraggable={!readOnly}
              nodesConnectable={!readOnly}
              minZoom={0.05}
              elementsSelectable
              edgesFocusable={false}
            >
              <Background />
              <MiniMap />
              <Controls />
            </ReactFlow>
          </DagNodeContext.Provider>
          {loading && (
            <div style={{ position: 'absolute', inset: 0, display: 'grid', placeItems: 'center' }}>
              加载中...
            </div>
          )}
        </div>
      </div>

      {/* 「+」触发的就近节点类型菜单：与左侧面板选择效果一致 */}
      {pendingAdd?.pos && (
        <div
          data-add-node-menu
          className="nodrag nopan"
          style={{
            position: 'fixed',
            left: Math.min(pendingAdd.pos.x + 8, window.innerWidth - 190),
            top: Math.min(pendingAdd.pos.y + 8, window.innerHeight - 160),
            zIndex: 1010,
            background: '#fff',
            border: '1px solid #f0f0f0',
            borderRadius: 8,
            boxShadow: '0 6px 16px rgba(0,0,0,0.12)',
            padding: 4,
            minWidth: 140,
          }}
          onMouseDown={(event) => event.stopPropagation()}
        >
          {palette
            .filter((item) => item.type !== 'start')
            .map((item) => (
              <div
                key={item.type}
                onClick={() => addNodeFromPalette(item.type)}
                style={{
                  cursor: 'pointer',
                  padding: '6px 10px',
                  borderRadius: 6,
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                }}
                onMouseEnter={(event) => {
                  event.currentTarget.style.background = '#f5f5f5'
                }}
                onMouseLeave={(event) => {
                  event.currentTarget.style.background = 'transparent'
                }}
              >
                {item.icon}
                <span>{item.label}</span>
              </div>
            ))}
        </div>
      )}

      <Drawer
        title={selectedNode ? `节点：${selectedNode.data.label ?? selectedNode.id}` : '节点详情'}
        open={!!selectedNode}
        onClose={() => setSelectedNode(null)}
        size={560}
        extra={
          !readOnly && (
            <Button type="primary" onClick={() => void saveNodeForm()}>
              确定
            </Button>
          )
        }
      >
        {selectedNode && (
          <>
            {selectedNodeInstance && (
              <Typography.Paragraph type="secondary">
                运行状态：{selectedNodeInstance.status}
                {selectedNodeInstance.status === 'failed_skipped' ? '（已跳过）' : ''}
              </Typography.Paragraph>
            )}
            <Form form={nodeForm} layout="vertical" disabled={readOnly}>
              <Form.Item label="名称" name="name" rules={[{ required: true, message: '请输入名称' }]}>
                <Input />
              </Form.Item>
              <Form.Item label="说明" name="description">
                <Input.TextArea rows={2} />
              </Form.Item>
              <Form.Item label="负责人" name="owners" rules={[{ required: !readOnly, message: '请设置负责人' }]}>
                <Select mode="tags" placeholder="选择或输入负责人" />
              </Form.Item>
              {isExecutableNode && !readOnly && (
                <>
                  <Form.Item label="资源类型" name="runtime_kind" initialValue="local">
                    <Select
                      options={[
                        { value: 'local', label: '本地（仅调试）' },
                        { value: 'ssh', label: '远程 SSH' },
                        { value: 'k8s', label: 'Kubernetes' },
                      ]}
                      onChange={() => nodeForm.setFieldValue('resource_id', undefined)}
                    />
                  </Form.Item>
                  {runtimeKind !== 'local' && (
                    <>
                      <Form.Item
                        label="执行资源"
                        name="resource_id"
                        rules={[{ required: true, message: '请选择执行资源' }]}
                      >
                        <Select
                          placeholder="选择执行资源"
                          allowClear
                          options={executorResources
                            .filter((item) => item.type === runtimeKind)
                            .map((item) => ({ value: item.id, label: item.name }))}
                        />
                      </Form.Item>
                      <Form.Item>
                        <Button size="small" onClick={() => setResourceModalOpen(true)}>
                          新建{runtimeKind === 'ssh' ? ' SSH' : ' K8s'}资源
                        </Button>
                      </Form.Item>
                    </>
                  )}
                </>
              )}
              {isExecutableNode && (
                <>
                  <Form.Item label="启动策略" name="start_mode">
                    <Select options={START_MODE_OPTIONS} />
                  </Form.Item>
                  {startMode === 'custom' && (
                    <>
                      <Form.Item
                        label="启动脚本 (start_script)"
                        name="start_script"
                        extra="exit 0 才启动；非 0 则跳过该节点"
                      >
                        <LazyScriptEditor readOnly={readOnly} paramNames={paramNames} height={160} />
                      </Form.Item>
                      <Form.Item label="必须完成的前驱节点" name="required_predecessor_ids">
                        <Select
                          mode="multiple"
                          options={predecessorOptions}
                          placeholder="选择前驱节点"
                          allowClear
                        />
                      </Form.Item>
                    </>
                  )}
                </>
              )}
              <Form.Item label="pre_script" name="pre_script">
                <LazyScriptEditor readOnly={readOnly} paramNames={paramNames} />
              </Form.Item>
              <Form.Item label="script" name="script">
                <LazyScriptEditor readOnly={readOnly} paramNames={paramNames} height={200} />
              </Form.Item>
              <Form.Item label="post_script" name="post_script">
                <LazyScriptEditor readOnly={readOnly} paramNames={paramNames} />
              </Form.Item>
            </Form>
          </>
        )}
      </Drawer>

      <ResourceModal
        open={resourceModalOpen}
        type={runtimeKind}
        onClose={() => setResourceModalOpen(false)}
        onCreated={(resource) => {
          setExecutorResources((current) => [resource, ...current])
          nodeForm.setFieldValue('resource_id', resource.id)
          setResourceModalOpen(false)
        }}
      />
    </div>
  )
})

const DagEditor = forwardRef<DagEditorHandle, DagEditorProps>(function DagEditor(props, ref) {
  return (
    <ReactFlowProvider>
      <DagEditorInner {...props} ref={ref} />
    </ReactFlowProvider>
  )
})

export default DagEditor

export function DagStatusLegend() {
  return (
    <Space wrap>
      <Space>
        <FlagOutlined /> 未运行：灰色
      </Space>
      <Space>
        <PlayCircleOutlined style={{ color: nodeBorderColor('running') }} /> 运行中：蓝框
      </Space>
      <Space>
        <BranchesOutlined style={{ color: nodeBorderColor('succeeded') }} /> 成功：绿框
      </Space>
      <Space>
        <span style={{ color: nodeBorderColor('failed_skipped') }}>●</span> 已跳过：灰虚线框
      </Space>
      <Space>
        <span style={{ color: nodeBorderColor('failed') }}>●</span> 失败：红框
      </Space>
      <Space>
        <span style={{ color: nodeBorderColor('waiting_approval') }}>●</span> 待审批：橙框
      </Space>
    </Space>
  )
}
