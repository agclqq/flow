import { CloseOutlined, PlusOutlined } from '@ant-design/icons'
import { useEffect, useRef, useState } from 'react'
import {
  BaseEdge,
  EdgeLabelRenderer,
  getBezierPath,
  type EdgeProps,
} from '@xyflow/react'
import { useDagNodeContext } from './dagNodeContext'

const HIGHLIGHT_COLOR = '#1677ff'

const actionButtonStyle: React.CSSProperties = {
  width: 20,
  height: 20,
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

/** 可交互连线：hover/选中时高亮，中点显示添加节点与删除按钮。 */
export function DeletableFlowEdge({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  selected,
  style,
}: EdgeProps) {
  const { editable, onAddFromEdge, onDeleteEdge } = useDagNodeContext()
  const [hovered, setHovered] = useState(false)
  const hideTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const active = hovered || selected
  const [path, labelX, labelY] = getBezierPath({
    sourceX,
    sourceY,
    targetX,
    targetY,
    sourcePosition,
    targetPosition,
  })

  // 边线与中点按钮分属 SVG 与 HTML 覆盖层，直接切换 hover 会让按钮在指针
  // 到达前消失；离开后延迟隐藏，进入按钮容器时取消隐藏。
  const clearHideTimer = () => {
    if (hideTimerRef.current) {
      clearTimeout(hideTimerRef.current)
      hideTimerRef.current = null
    }
  }
  const handleEnter = () => {
    clearHideTimer()
    setHovered(true)
  }
  const handleLeave = () => {
    clearHideTimer()
    hideTimerRef.current = setTimeout(() => setHovered(false), 200)
  }
  useEffect(() => clearHideTimer, [])

  return (
    <>
      <g onMouseEnter={handleEnter} onMouseLeave={handleLeave}>
        <BaseEdge
          id={id}
          path={path}
          style={{
            ...style,
            stroke: active ? HIGHLIGHT_COLOR : style?.stroke,
            strokeWidth: selected ? 2 : style?.strokeWidth,
          }}
        />
      </g>
      {editable && active && (
        <EdgeLabelRenderer>
          <div
            onMouseEnter={handleEnter}
            onMouseLeave={handleLeave}
            style={{
              position: 'absolute',
              transform: `translate(-50%, -50%) translate(${labelX}px, ${labelY}px)`,
              display: 'flex',
              gap: 4,
              pointerEvents: 'none',
            }}
          >
            {onAddFromEdge && (
              <button
                aria-label="在此处添加节点"
                title="在此处添加节点"
                onMouseDown={(event) => event.stopPropagation()}
                onClick={(event) => {
                  event.stopPropagation()
                  onAddFromEdge(id, { x: event.clientX, y: event.clientY })
                }}
                style={actionButtonStyle}
              >
                <PlusOutlined style={{ fontSize: 10 }} />
              </button>
            )}
            <button
              aria-label="删除连线"
              title="删除连线"
              onMouseDown={(event) => event.stopPropagation()}
              onClick={(event) => {
                event.stopPropagation()
                onDeleteEdge?.(id)
              }}
              style={{
                ...actionButtonStyle,
                borderColor: '#ffccc7',
                color: '#ff4d4f',
              }}
            >
              <CloseOutlined style={{ fontSize: 10 }} />
            </button>
          </div>
        </EdgeLabelRenderer>
      )}
    </>
  )
}

export const flowEdgeTypes = {
  flow: DeletableFlowEdge,
}
