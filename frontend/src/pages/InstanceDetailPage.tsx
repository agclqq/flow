import { Button, Card, Collapse, Descriptions, Drawer, Form, Input, Modal, Space, Tag, Typography, message } from 'antd'
import { lazy, Suspense, useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { approveNodeInstance, getInstance, listNodeOperationLogs, rejectNodeInstance, retryInstance, skipNodeInstance } from '../api/instances'
import { getTemplate, getVersion } from '../api/templates'
import type { FlowInstance, FlowNode, FlowTemplate, NodeInstance, NodeOperationLog } from '../api/types'
import { INSTANCE_SYSTEM_PARAM_KEYS } from '../constants/systemParams'

const ScriptEditor = lazy(() => import('../components/dag/ScriptEditor'))
import DagEditor, { DagStatusLegend } from '../components/dag/DagEditor'
import { formatDateTime, formatDuration, instanceStatusLabel, nodeStatusLabel } from '../utils/format'

function renderKeyValues(data: Record<string, string> = {}) {
  const entries = Object.entries(data)
  if (entries.length === 0) return '-'
  return entries.map(([key, value]) => `${key}=${value}`).join(', ')
}

/** 环境变量按每个一行展示：key= 灰色前缀，value 可换行。 */
function renderKeyValuesPerLine(data: Record<string, string> = {}) {
  const entries = Object.entries(data ?? {})
  if (entries.length === 0) return '-'
  return (
    <div style={{ display: 'grid', gap: 2 }}>
      {entries.map(([key, value]) => (
        <div key={key} style={{ fontFamily: 'monospace', fontSize: 12 }}>
          <span style={{ color: '#8c8c8c' }}>{key}=</span>
          <span style={{ wordBreak: 'break-all' }}>{value}</span>
        </div>
      ))}
    </div>
  )
}

/** 只读脚本展示：带语法高亮的代码编辑器；空内容显示占位符。 */
function Pre({ value, height = 100 }: { value?: string; height?: number }) {
  if (!value?.trim()) {
    return <span style={{ color: '#bfbfbf' }}>-</span>
  }
  return (
    <div style={{ width: '100%', minWidth: 320 }}>
      <Suspense fallback={<span>加载中...</span>}>
        <ScriptEditor value={value} readOnly height={height} />
      </Suspense>
    </div>
  )
}

export default function InstanceDetailPage() {
  const { templateId, instanceId } = useParams()
  const tplId = Number(templateId)
  const instId = Number(instanceId)
  const navigate = useNavigate()
  const [instance, setInstance] = useState<FlowInstance | null>(null)
  const [template, setTemplate] = useState<FlowTemplate | null>(null)
  const [loading, setLoading] = useState(true)
  const [selectedNode, setSelectedNode] = useState<NodeInstance | null>(null)
  const [acting, setActing] = useState(false)

  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const [tpl, inst] = await Promise.all([getTemplate(tplId), getInstance(tplId, instId)])
      setTemplate(tpl)
      setInstance(inst)
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载实例详情失败')
    } finally {
      setLoading(false)
    }
  }, [instId, tplId])

  useEffect(() => {
    void loadData()
  }, [loadData])

  // 静默刷新实例：只更新数据，不触发 loading，避免界面闪烁。
  const refreshInstance = useCallback(async () => {
    try {
      const inst = await getInstance(tplId, instId)
      setInstance(inst)
    } catch {
      // 轮询失败静默处理，等待下一次轮询重试。
    }
  }, [instId, tplId])

  const isRunning =
    instance?.status === 'pending' || instance?.status === 'running' || instance?.status === 'retry'

  // 加载实例对应版本快照的节点定义，用于展示节点配置信息。
  const versionId = instance?.template_version_id
  const [versionNodes, setVersionNodes] = useState<FlowNode[]>([])
  useEffect(() => {
    if (!versionId) return
    let cancelled = false
    getVersion(tplId, versionId)
      .then((detail) => {
        if (!cancelled) setVersionNodes(detail.nodes)
      })
      .catch(() => setVersionNodes([]))
    return () => {
      cancelled = true
    }
  }, [tplId, versionId])

  useEffect(() => {
    if (!isRunning) return
    const timer = window.setInterval(() => {
      void refreshInstance()
    }, 1000)
    return () => window.clearInterval(timer)
  }, [isRunning, refreshInstance])

  const paramNames = useMemo(
    () => Object.keys(instance?.params ?? {}),
    [instance?.params],
  )

  // 系统参数取值：优先实例 env（后端创建实例时注入）；旧实例或节点级参数回退展示。
  const systemParamValue = (key: string): string => {
    const legacyKey = key.toLowerCase()
    const fromEnv = instance?.env?.[key] ?? instance?.env?.[legacyKey]
    if (fromEnv !== undefined && fromEnv !== '') return fromEnv
    switch (key) {
      case 'TEMPLATE_ID':
        return String(tplId)
      case 'TEMPLATE_NAME':
        return template?.name ?? '-'
      case 'TEMPLATE_OWNER':
        return template?.owners?.join('、') || '-'
      case 'INSTANCE_ID':
        return instance?.id != null ? String(instance.id) : '-'
      default:
        return '-'
    }
  }

  // 当前选中节点对应的模板节点配置（来自版本快照）。
  const selectedNodeConfig = useMemo(
    () =>
      selectedNode ? versionNodes.find((item) => item.id === selectedNode.node_id) : undefined,
    [selectedNode, versionNodes],
  )

  // 人工节点操作日志：抽屉打开/动作完成后刷新。
  const [operationLogs, setOperationLogs] = useState<NodeOperationLog[]>([])
  const [operator, setOperator] = useState(() => localStorage.getItem('flow_operator') ?? '')

  const refreshOperationLogs = useCallback(async () => {
    if (!selectedNode) return
    try {
      setOperationLogs(await listNodeOperationLogs(tplId, instId, selectedNode.node_id))
    } catch {
      setOperationLogs([])
    }
  }, [instId, selectedNode, tplId])

  useEffect(() => {
    if (selectedNode) {
      void refreshOperationLogs()
    } else {
      setOperationLogs([])
    }
  }, [selectedNode, refreshOperationLogs])

  const handleOperatorChange = (value: string) => {
    setOperator(value)
    localStorage.setItem('flow_operator', value)
  }

  // 重试失败实例：状态回到运行中，失败的节点重新执行，页面恢复秒级轮询。
  const handleRetry = async () => {
    setActing(true)
    try {
      await retryInstance(tplId, instId)
      message.success('实例已重新开始运行')
      await loadData()
    } catch (error) {
      message.error(error instanceof Error ? error.message : '重试失败')
    } finally {
      setActing(false)
    }
  }

  // 画布节点上的审批操作（实例视图）。
  const handleCanvasApprove = (nodeId: number) => {
    setActing(true)
    void approveNodeInstance(tplId, instId, nodeId)
      .then(() => {
        message.success('已通过审批')
        return loadData()
      })
      .catch((error) => message.error(error instanceof Error ? error.message : '审批失败'))
      .finally(() => setActing(false))
  }

  const handleCanvasReject = (nodeId: number) => {
    setActing(true)
    void rejectNodeInstance(tplId, instId, nodeId)
      .then(() => {
        message.success('已拒绝')
        return loadData()
      })
      .catch((error) => message.error(error instanceof Error ? error.message : '拒绝失败'))
      .finally(() => setActing(false))
  }

  // 跳过运行中的节点：终止其脚本执行，节点标记为跳过，调度继续后续节点。
  const handleSkip = () => {
    if (!selectedNode) return
    Modal.confirm({
      title: `确认跳过节点「${selectedNodeConfig?.name ?? selectedNode.node_id}」？`,
      content: '将终止该节点正在执行的内容，节点标记为跳过，后续节点继续调度。',
      okText: '跳过',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          await skipNodeInstance(tplId, instId, selectedNode.node_id)
          message.success('节点已跳过')
          await loadData()
          await refreshOperationLogs()
        } catch (error) {
          message.error(error instanceof Error ? error.message : '跳过失败')
        }
      },
    })
  }

  if (loading && !instance) {
    return <Typography.Paragraph>加载中...</Typography.Paragraph>
  }

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        width: '100%',
        height: 'calc(100vh - 64px - 48px)',
        minHeight: 420,
      }}
    >
      <Space style={{ width: '100%', justifyContent: 'space-between', marginBottom: 16, flexShrink: 0 }}>
        <Typography.Title level={3} style={{ margin: 0 }}>
          {template ? `${template.name} - 实例 #${instId}` : `实例 #${instId}`}
        </Typography.Title>
        <Space>
          <Button onClick={() => navigate(`/templates/${tplId}/instances`)}>返回列表</Button>
          {instance?.status === 'failed' && (
            <Button type="primary" danger loading={acting} onClick={() => void handleRetry()}>
              重试
            </Button>
          )}
        </Space>
      </Space>

      <div
        style={{
          flex: 1,
          minHeight: 0,
          display: 'grid',
          gridTemplateColumns: '300px 1fr',
          gap: 16,
        }}
      >
        <Collapse
          style={{ overflow: 'auto', alignSelf: 'stretch' }}
          defaultActiveKey={['info']}
          items={[
            {
              key: 'info',
              label: '实例信息',
              children: (
                <Descriptions column={1} size="small">
                  <Descriptions.Item label="状态">
                    <Tag>{instance ? instanceStatusLabel(instance.status) : '-'}</Tag>
                  </Descriptions.Item>
                  <Descriptions.Item label="启动时间">
                    {formatDateTime(instance?.created_at)}
                  </Descriptions.Item>
                  <Descriptions.Item label="更新时间">
                    {formatDateTime(instance?.updated_at)}
                  </Descriptions.Item>
                </Descriptions>
              ),
            },
            {
              key: 'params',
              label: '入参',
              children: <Typography.Paragraph>{renderKeyValues(instance?.params)}</Typography.Paragraph>,
            },
            {
              key: 'env',
              label: '系统参数',
              children: (
                <Descriptions column={1} size="small">
                  {INSTANCE_SYSTEM_PARAM_KEYS.map((key) => (
                    <Descriptions.Item key={key} label={key}>
                      {systemParamValue(key)}
                    </Descriptions.Item>
                  ))}
                </Descriptions>
              ),
            },
          ]}
        />

        <Card
          title="执行视图"
          size="small"
          extra={<DagStatusLegend />}
          style={{ height: '100%', minHeight: 0, display: 'flex', flexDirection: 'column' }}
          styles={{ body: { flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column', padding: 12 } }}
        >
          <div style={{ flex: 1, minHeight: 0 }}>
            <DagEditor
              readOnly
              templateId={tplId}
              versionId={instance?.template_version_id}
              paramNames={paramNames}
              nodeInstances={instance?.node_instances}
              onNodeInstanceClick={setSelectedNode}
              onApproveNode={handleCanvasApprove}
              onRejectNode={handleCanvasReject}
            />
          </div>
        </Card>
      </div>

      <Drawer
        title={`节点运行详情${selectedNodeConfig?.name ? ` - ${selectedNodeConfig.name}` : ''}`}
        open={!!selectedNode}
        onClose={() => setSelectedNode(null)}
        size={520}
        extra={
          selectedNode?.status === 'running' ? (
            <Button type="primary" danger onClick={handleSkip}>
              跳过
            </Button>
          ) : null
        }
      >
        {selectedNode && (
          <>
            <Typography.Title level={5} style={{ marginTop: 0 }}>
              运行详情
            </Typography.Title>
            <Descriptions column={1} size="small">
              <Descriptions.Item label="节点 ID">{selectedNode.node_id}</Descriptions.Item>
              <Descriptions.Item label="类型">{selectedNode.type}</Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag>{nodeStatusLabel(selectedNode.status)}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="运行时间">
                {selectedNode.run_times?.length ? (
                  <div style={{ display: 'grid', gap: 2 }}>
                    {selectedNode.run_times.map((span, index) => {
                      const label =
                        selectedNode.run_times!.length > 1 ? `第 ${index + 1} 次：` : ''
                      return (
                        <span key={index}>
                          {label}
                          {formatDateTime(span.start_at)} ~{' '}
                          {span.end_at ? (
                            <>
                              {formatDateTime(span.end_at)}
                              <Typography.Text type="secondary" style={{ marginLeft: 6 }}>
                                （{formatDuration(span.start_at, span.end_at)}）
                              </Typography.Text>
                            </>
                          ) : (
                            <Tag color="processing" style={{ marginLeft: 4 }}>
                              进行中
                            </Tag>
                          )}
                        </span>
                      )
                    })}
                  </div>
                ) : (
                  '-'
                )}
              </Descriptions.Item>
              <Descriptions.Item label="重试次数">{selectedNode.retry_count}</Descriptions.Item>
              <Descriptions.Item label="环境变量">{renderKeyValuesPerLine(selectedNode.env)}</Descriptions.Item>
              <Descriptions.Item label="输出变量">{renderKeyValuesPerLine(selectedNode.output_env)}</Descriptions.Item>
            </Descriptions>

            {selectedNode.status === 'waiting_approval' && (
              <Form.Item label="操作人" style={{ marginTop: 12, marginBottom: 12 }}>
                <Input
                  value={operator}
                  onChange={(event) => handleOperatorChange(event.target.value)}
                  placeholder="操作人姓名（将记入操作日志）"
                  allowClear
                />
              </Form.Item>
            )}

            <Typography.Title level={5} style={{ margin: '8px 0 8px' }}>
              操作日志
            </Typography.Title>
            {operationLogs.length === 0 ? (
              <Typography.Text type="secondary">暂无操作记录</Typography.Text>
            ) : (
              <div style={{ display: 'grid', gap: 4 }}>
                {operationLogs.map((log) => (
                  <div key={log.id} style={{ display: 'flex', gap: 8, fontSize: 13 }}>
                    <span style={{ color: '#8c8c8c', minWidth: 150 }}>
                      {formatDateTime(log.created_at)}
                    </span>
                    <span style={{ minWidth: 80 }}>
                      {log.action === 'approve' ? (
                        <Tag color="success">通过</Tag>
                      ) : (
                        <Tag color="error">拒绝</Tag>
                      )}
                    </span>
                    <span>{log.operator}</span>
                  </div>
                ))}
              </div>
            )}

            <Collapse
              style={{ marginTop: 16 }}
              defaultActiveKey={[]}
              items={[
                {
                  key: 'node-info',
                  label: '节点信息',
                  children: (
                    <Descriptions column={1} size="small">
                      <Descriptions.Item label="名称">{selectedNodeConfig?.name ?? '-'}</Descriptions.Item>
                      <Descriptions.Item label="说明">
                        {selectedNodeConfig?.description || '-'}
                      </Descriptions.Item>
                      <Descriptions.Item label="负责人">
                        {selectedNodeConfig?.owners?.join('、') || '-'}
                      </Descriptions.Item>
                      <Descriptions.Item label="启动脚本">
                        <Pre value={selectedNodeConfig?.start_script} height={80} />
                      </Descriptions.Item>
                      <Descriptions.Item label="pre_script">
                        <Pre value={selectedNodeConfig?.pre_script} height={100} />
                      </Descriptions.Item>
                      <Descriptions.Item label="script">
                        <Pre value={selectedNodeConfig?.script} height={160} />
                      </Descriptions.Item>
                      <Descriptions.Item label="post_script">
                        <Pre value={selectedNodeConfig?.post_script} height={100} />
                      </Descriptions.Item>
                    </Descriptions>
                  ),
                },
              ]}
            />
          </>
        )}
      </Drawer>
    </div>
  )
}
