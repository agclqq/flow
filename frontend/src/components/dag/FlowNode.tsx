import { CloseOutlined, CodeOutlined, PlusOutlined, UserOutlined } from '@ant-design/icons'
import { Button } from 'antd'
import { Handle, Position, type NodeProps } from '@xyflow/react'
import { useState } from 'react'
import { useDagNodeContext } from './dagNodeContext'
import { nodeBorderColor } from './dagUtils'
import { formatDateTime } from '../../utils/format'

type FlowNodeData = {
  backendId?: number
  label?: string
  nodeType?: string
  status?: string
  hasOutgoing?: boolean
  owners?: string[]
  approvers?: string[]
  last_start_at?: string
  last_end_at?: string
}

const HIGHLIGHT_COLOR = '#1677ff'

const addButtonStyle: React.CSSProperties = {
  width: 22,
  height: 22,
  padding: 0,
  borderRadius: '50%',
  border: '1px solid #d9d9d9',
  background: '#fff',
  color: '#595959',
  fontSize: 12,
  lineHeight: 1,
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  cursor: 'pointer',
  pointerEvents: 'all',
}

/** 无出边节点右侧的延伸线 + 添加按钮（n8n 风格）。 */
function NodeExtendAdd({ onClick }: { onClick: (event: React.MouseEvent) => void }) {
  // 按住 + 时模拟从 source Handle 按下，交给 React Flow 走原生连线流程；
  // 松开未连接到目标节点时连线自动取消，延伸线随之恢复。
  const handlePress = (event: React.MouseEvent<HTMLButtonElement>) => {
    const nodeEl = event.currentTarget.closest('.react-flow__node')
    const handle = nodeEl?.querySelector<HTMLElement>('.react-flow__handle.source')
    if (!handle) return
    handle.dispatchEvent(
      new MouseEvent('mousedown', {
        bubbles: true,
        cancelable: true,
        view: window,
        clientX: event.clientX,
        clientY: event.clientY,
        button: 0,
      }),
    )
  }

  return (
    <div
      style={{
        position: 'absolute',
        left: '100%',
        top: '50%',
        transform: 'translateY(-50%)',
        width: 56,
        display: 'flex',
        alignItems: 'center',
        pointerEvents: 'none',
      }}
    >
      <div style={{ flex: 1, height: 2, background: '#bfbfbf' }} />
      <button
        className="nodrag nopan"
        aria-label="添加后续节点"
        title="点击选择节点类型添加；按住拖动可直接连线"
        onMouseDown={(event) => {
          event.stopPropagation()
          event.preventDefault()
          handlePress(event)
        }}
        onClick={(event) => {
          event.stopPropagation()
          onClick(event)
        }}
        style={addButtonStyle}
      >
        <PlusOutlined style={{ fontSize: 10 }} />
      </button>
    </div>
  )
}

/** 节点类型图标：普通任务 / 人工审批，用颜色 + 图标区分类型。 */
const TYPE_META: Record<string, { icon: typeof CodeOutlined; color: string }> = {
  task: { icon: CodeOutlined, color: '#1677ff' },
  manual: { icon: UserOutlined, color: '#fa8c16' },
  input: { icon: CodeOutlined, color: '#1677ff' },
}

/** 实例视图下节点上的扩展信息：负责人/审批人、审批按钮、最近一次运行时间。 */
function NodeInstanceInfo({ nodeData }: { nodeData: FlowNodeData }) {
  const { onApproveNode, onRejectNode } = useDagNodeContext()
  const isManual = nodeData.nodeType === 'manual'
  const people = isManual
    ? nodeData.approvers?.filter(Boolean).join('、')
    : nodeData.owners?.filter(Boolean).join('、')
  // 人工节点：审批人始终展示；普通节点仅有负责人时展示。
  const showPeople = isManual || !!people
  const waitingApproval = isManual && nodeData.status === 'waiting_approval'
  const hasRun = !!nodeData.last_start_at

  return (
    <>
      {showPeople && (
        <div style={{ fontSize: 11, color: '#8c8c8c', display: 'flex', alignItems: 'center', gap: 4 }}>
          <span>{isManual ? '审批人' : '负责人'}</span>
          <span style={{ color: '#595959' }}>{people || '-'}</span>
        </div>
      )}
      {waitingApproval && onApproveNode && onRejectNode && (
        <div style={{ display: 'flex', gap: 4, marginTop: 2 }}>
          <Button
            size="small"
            danger
            className="nodrag nopan"
            style={{ height: 20, padding: '0 8px', fontSize: 12 }}
            onClick={(event) => {
              event.stopPropagation()
              onRejectNode(Number(nodeData.backendId))
            }}
          >
            拒绝
          </Button>
          <Button
            size="small"
            type="primary"
            className="nodrag nopan"
            style={{ height: 20, padding: '0 8px', fontSize: 12 }}
            onClick={(event) => {
              event.stopPropagation()
              onApproveNode(Number(nodeData.backendId))
            }}
          >
            通过
          </Button>
        </div>
      )}
      {hasRun && (
        <div style={{ fontSize: 11, color: '#8c8c8c' }}>
          {formatDateTime(nodeData.last_start_at)} ~{' '}
          {nodeData.last_end_at ? formatDateTime(nodeData.last_end_at) : '进行中'}
        </div>
      )}
    </>
  )
}

function TaskNode({ id, data, selected }: NodeProps) {
  const nodeData = data as FlowNodeData
  const { editable, onDeleteNode, onAddFromNode, connectingNodeId } = useDagNodeContext()
  const [hovered, setHovered] = useState(false)
  const baseBorderColor = nodeBorderColor(nodeData.status)
  // 只读实例视图：hover 不改变边框颜色（高亮仅用于可编辑画布的交互提示）。
  const borderColor = hovered && editable ? HIGHLIGHT_COLOR : baseBorderColor
  const isSkipped = nodeData.status === 'failed_skipped'
  const typeMeta = TYPE_META[nodeData.nodeType ?? 'task'] ?? TYPE_META.task
  const TypeIcon = typeMeta.icon
  const canDelete = editable && !!onDeleteNode && hovered
  const canExtend =
    editable && !!onAddFromNode && !nodeData.hasOutgoing && connectingNodeId !== id
  const nodeName = nodeData.label || id
  const isTaskNode = nodeData.nodeType === 'task'

  const handleDelete = (event: React.MouseEvent) => {
    event.stopPropagation()
    onDeleteNode?.(id, nodeName)
  }

  return (
    <div
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        position: 'relative',
        minWidth: 120,
        padding: '8px 12px',
        borderRadius: 8,
        border: isSkipped && !hovered ? `2px dashed ${baseBorderColor}` : `2px solid ${borderColor}`,
        background: selected ? '#f0f5ff' : isSkipped ? '#fafafa' : '#fff',
        boxShadow: hovered || selected ? '0 0 0 2px rgba(22,119,255,0.2)' : undefined,
        opacity: isSkipped && !hovered ? 0.85 : 1,
        cursor: isTaskNode ? 'pointer' : undefined,
      }}
    >
      {canDelete && (
        <Button
          type="text"
          size="small"
          aria-label={`删除${nodeName}`}
          icon={<CloseOutlined style={{ fontSize: 10 }} />}
          onClick={handleDelete}
          onMouseDown={(event) => event.stopPropagation()}
          style={{
            position: 'absolute',
            top: -10,
            right: -10,
            width: 20,
            height: 20,
            minWidth: 20,
            padding: 0,
            borderRadius: '50%',
            border: '1px solid #ffccc7',
            background: '#fff2f0',
            color: '#ff4d4f',
            lineHeight: 1,
            zIndex: 1,
          }}
        />
      )}
      <Handle type="target" position={Position.Left} style={{ background: borderColor }} />
      <div style={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <TypeIcon style={{ color: typeMeta.color, fontSize: 15 }} aria-label={nodeData.nodeType === 'manual' ? '人工节点' : '任务节点'} />
          <span style={{ fontWeight: 600 }}>{nodeData.label}</span>
          {isSkipped ? (
            <span
              style={{
                fontSize: 11,
                lineHeight: '16px',
                padding: '0 5px',
                borderRadius: 4,
                color: '#d46b08',
                background: '#fff2e8',
                border: '1px solid #ffbb96',
              }}
            >
              已跳过
            </span>
          ) : null}
        </div>
        {/* 实例视图扩展信息：负责人/审批人、审批操作、最近一次运行时间 */}
        {!editable && <NodeInstanceInfo nodeData={nodeData} />}
      </div>
      <Handle type="source" position={Position.Right} style={{ background: borderColor }} />
      {canExtend && <NodeExtendAdd onClick={(event) => onAddFromNode?.(id, { x: event.clientX, y: event.clientY })} />}
    </div>
  )
}

function TerminalNode({ id, data, selected }: NodeProps) {
  const nodeData = data as FlowNodeData
  const { editable, onAddFromNode, connectingNodeId } = useDagNodeContext()
  const [hovered, setHovered] = useState(false)
  const isStart = nodeData.nodeType === 'start'
  // 只读实例视图：hover 不改变边框颜色。
  const borderColor = hovered && editable ? HIGHLIGHT_COLOR : nodeBorderColor(nodeData.status)
  const label = isStart ? '开始' : '结束'
  const size = 56
  const canExtend =
    editable && !!onAddFromNode && isStart && !nodeData.hasOutgoing && connectingNodeId !== id

  return (
    <div
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        width: size,
        height: size,
        borderRadius: '50%',
        border: `2px solid ${borderColor}`,
        background: selected ? '#f0f5ff' : '#fff',
        boxShadow: hovered || selected ? '0 0 0 2px rgba(22,119,255,0.2)' : undefined,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        fontSize: 12,
        fontWeight: 600,
        color: '#262626',
        cursor: 'grab',
        userSelect: 'none',
      }}
    >
      {!isStart && (
        <Handle
          type="target"
          position={Position.Left}
          style={{ background: borderColor, left: -1 }}
        />
      )}
      <span>{label}</span>
      {isStart && (
        <Handle
          type="source"
          position={Position.Right}
          style={{ background: borderColor, right: -1 }}
        />
      )}
      {canExtend && <NodeExtendAdd onClick={(event) => onAddFromNode?.(id, { x: event.clientX, y: event.clientY })} />}
    </div>
  )
}

export const flowNodeTypes = {
  task: TaskNode,
  manual: TaskNode,
  start: TerminalNode,
  end: TerminalNode,
  input: TaskNode,
}
