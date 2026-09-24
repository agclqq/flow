import { createContext, useContext } from 'react'

export type DagNodeContextValue = {
  editable: boolean
  onDeleteNode?: (nodeId: string, nodeName: string) => void
  /** 删除边（带二次确认，由编辑器实现） */
  onDeleteEdge?: (edgeId: string) => void
  /** 从节点延伸的 + 触发：进入“选择节点类型”模式，pos 为鼠标位置（就近弹出菜单） */
  onAddFromNode?: (nodeId: string, pos?: { x: number; y: number }) => void
  /** 从边中点的 + 触发：进入“选择节点类型”模式（节点将插入该边中间），pos 为鼠标位置 */
  onAddFromEdge?: (edgeId: string, pos?: { x: number; y: number }) => void
  /** 正在从该节点拖出连线（隐藏其延伸线，拖动结束未连接则恢复） */
  connectingNodeId?: string | null
  /** 实例视图：审批通过 / 拒绝（仅待审批的人工节点展示按钮） */
  onApproveNode?: (nodeId: number) => void
  onRejectNode?: (nodeId: number) => void
}

export const DagNodeContext = createContext<DagNodeContextValue>({ editable: false })

export function useDagNodeContext() {
  return useContext(DagNodeContext)
}
