import { Button, Space, Table, Tag, Typography, message } from 'antd'
import { useCallback, useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { listInstances } from '../api/instances'
import { getTemplate } from '../api/templates'
import type { FlowInstance } from '../api/types'
import { formatDateTime, formatDuration, instanceStatusLabel } from '../utils/format'

export default function InstanceListPage() {
  const { id } = useParams()
  const templateId = Number(id)
  const navigate = useNavigate()
  const [templateName, setTemplateName] = useState('')
  const [data, setData] = useState<FlowInstance[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const pageSize = 20

  const loadData = useCallback(async (opts?: { silent?: boolean }) => {
    if (!templateId) return
    // 静默刷新（定时轮询）不显示加载态，避免表格闪屏。
    if (!opts?.silent) setLoading(true)
    try {
      const [template, instances] = await Promise.all([
        getTemplate(templateId),
        listInstances(templateId, (page - 1) * pageSize, pageSize),
      ])
      setTemplateName(template.name)
      setData(instances.items)
      setTotal(instances.total)
    } catch (error) {
      if (!opts?.silent) {
        message.error(error instanceof Error ? error.message : '加载实例列表失败')
      }
    } finally {
      if (!opts?.silent) setLoading(false)
    }
  }, [page, templateId])

  useEffect(() => {
    void loadData()
  }, [loadData])

  // 每分钟静默刷新一次，保持列表内容最新且不闪屏。
  useEffect(() => {
    const timer = setInterval(() => {
      void loadData({ silent: true })
    }, 60_000)
    return () => clearInterval(timer)
  }, [loadData])

  const finishedAt = (record: FlowInstance) =>
    record.status === 'succeeded' || record.status === 'failed' ? record.updated_at : undefined

  return (
    <Space orientation="vertical" size={16} style={{ width: '100%' }}>
      <Space style={{ width: '100%', justifyContent: 'space-between' }}>
        <Typography.Title level={3} style={{ margin: 0 }}>
          {templateName ? `${templateName} - 实例列表` : '流水线实例'}
        </Typography.Title>
        <Button onClick={() => navigate(`/templates/${templateId}/edit`)}>返回模板</Button>
      </Space>
      <Table
        rowKey="id"
        loading={loading}
        columns={[
          { title: '实例 ID', dataIndex: 'id', width: 100 },
          {
            title: '触发方式',
            dataIndex: 'trigger_type',
            width: 100,
            render: (type: FlowInstance['trigger_type']) => {
              const map: Record<string, { label: string; color: string }> = {
                manual: { label: '手动', color: 'blue' },
                webhook: { label: 'Webhook', color: 'orange' },
                cron: { label: '定时', color: 'purple' },
              }
              const item = map[type] ?? { label: type, color: 'default' }
              return <Tag color={item.color}>{item.label}</Tag>
            },
          },
          {
            title: '启动时间',
            dataIndex: 'created_at',
            render: (value: string) => formatDateTime(value),
          },
          {
            title: '结束时间',
            key: 'finished_at',
            render: (_, record) => formatDateTime(finishedAt(record)),
          },
          {
            title: '耗时',
            key: 'duration',
            render: (_, record) => formatDuration(record.created_at, finishedAt(record)),
          },
          {
            title: '状态',
            dataIndex: 'status',
            render: (status: FlowInstance['status']) => (
              <Tag
                color={
                  status === 'succeeded'
                    ? 'green'
                    : status === 'failed'
                      ? 'red'
                      : status === 'running'
                        ? 'blue'
                        : 'default'
                }
              >
                {instanceStatusLabel(status)}
              </Tag>
            ),
          },
          {
            title: '操作',
            key: 'actions',
            render: (_, record) => (
              <Button type="link" onClick={() => navigate(`/templates/${templateId}/instances/${record.id}`)}>
                查看详情
              </Button>
            ),
          },
        ]}
        dataSource={data}
        pagination={{
          current: page,
          pageSize,
          total,
          onChange: setPage,
          showTotal: (count) => `共 ${count} 条`,
        }}
      />
    </Space>
  )
}
