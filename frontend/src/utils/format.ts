import type { FlowInstanceStatus, NodeInstanceStatus, TemplateStatus } from '../api/types'

export function formatDateTime(value?: string): string {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN')
}

export function formatDuration(start?: string, end?: string): string {
  if (!start || !end) return '-'
  const startMs = new Date(start).getTime()
  const endMs = new Date(end).getTime()
  if (Number.isNaN(startMs) || Number.isNaN(endMs) || endMs < startMs) return '-'
  const seconds = Math.floor((endMs - startMs) / 1000)
  if (seconds < 60) return `${seconds} 秒`
  const minutes = Math.floor(seconds / 60)
  const remain = seconds % 60
  if (minutes < 60) return `${minutes} 分 ${remain} 秒`
  const hours = Math.floor(minutes / 60)
  const remainMin = minutes % 60
  return `${hours} 时 ${remainMin} 分`
}

export function templateStatusLabel(status: TemplateStatus): string {
  return status === 'published' ? '已发布' : '草稿'
}

export function instanceStatusLabel(status: FlowInstanceStatus): string {
  const map: Record<FlowInstanceStatus, string> = {
    pending: '未开始',
    running: '执行中',
    succeeded: '成功',
    failed: '失败',
    retry: '重试中',
  }
  return map[status] ?? status
}

export function nodeStatusLabel(status: NodeInstanceStatus): string {
  const map: Record<NodeInstanceStatus, string> = {
    pending: '未运行',
    running: '运行中',
    succeeded: '成功',
    failed: '失败',
    failed_skipped: '失败跳过',
    retrying: '重试中',
    waiting_approval: '待审批',
    skipped: '跳过',
  }
  return map[status] ?? status
}
