import { CopyOutlined, DeleteOutlined, EditOutlined, PlayCircleOutlined } from '@ant-design/icons'
import { Button, Form, Input, Modal, Space, Table, Tag, Typography, message } from 'antd'
import { useCallback, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { startInstance } from '../api/instances'
import { copyTemplate, deleteTemplate, getTemplate, listParams, listTemplates } from '../api/templates'
import type { FlowParam, FlowTemplate } from '../api/types'
import { formatDateTime, templateStatusLabel } from '../utils/format'

export default function TemplateListPage() {
  const navigate = useNavigate()
  const [data, setData] = useState<FlowTemplate[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const pageSize = 20

  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const offset = (page - 1) * pageSize
      const res = await listTemplates(offset, pageSize)
      setData(res.items)
      setTotal(res.total)
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载模板列表失败')
    } finally {
      setLoading(false)
    }
  }, [page])

  useEffect(() => {
    void loadData()
  }, [loadData])

  // 手动启动参数弹窗：开启手动启动且参数列表不为空时弹出。
  const [startTarget, setStartTarget] = useState<FlowTemplate | null>(null)
  const [startParams, setStartParams] = useState<FlowParam[]>([])
  const [starting, setStarting] = useState(false)
  const [startForm] = Form.useForm()

  const doStart = async (record: FlowTemplate, params: Record<string, string>) => {
    const instance = await startInstance(record.id, params)
    message.success('实例已启动')
    navigate(`/templates/${record.id}/instances/${instance.id}`)
  }

  const handleStart = async (record: FlowTemplate) => {
    try {
      // 开启手动启动且最新发布快照定义了参数：弹窗收集，必填项校验。
      if (record.manual_trigger && record.published_version_id) {
        const params = await listParams(record.id, record.published_version_id)
        if (params.length > 0) {
          startForm.setFieldsValue(
            Object.fromEntries(params.map((item) => [item.name, item.default_value ?? ''])),
          )
          setStartParams(params)
          setStartTarget(record)
          return
        }
      }
      await doStart(record, {})
    } catch (error) {
      message.error(error instanceof Error ? error.message : '启动失败')
    }
  }

  const handleStartOk = async () => {
    if (!startTarget) return
    let values: Record<string, string>
    try {
      values = await startForm.validateFields()
    } catch {
      return // 必填项未填写：表单内提示，不关闭弹窗
    }
    setStarting(true)
    try {
      const params: Record<string, string> = {}
      for (const [key, value] of Object.entries(values)) {
        if (value !== '' && value != null) params[key] = String(value)
      }
      await doStart(startTarget, params)
      setStartTarget(null)
    } catch (error) {
      message.error(error instanceof Error ? error.message : '启动失败')
    } finally {
      setStarting(false)
    }
  }

  const handleDelete = (record: FlowTemplate) => {
    Modal.confirm({
      title: '确认删除',
      content: `确定删除模板「${record.name}」吗？`,
      okType: 'danger',
      onOk: async () => {
        await deleteTemplate(record.id)
        message.success('已删除')
        await loadData()
      },
    })
  }

  const handleCopy = async (record: FlowTemplate) => {
    try {
      const copied = await copyTemplate(record.id, record.name)
      message.success('复制成功')
      navigate(`/templates/${copied.id}/edit`)
    } catch (error) {
      message.error(error instanceof Error ? error.message : '复制失败')
    }
  }

  return (
    <Space orientation="vertical" size={16} style={{ width: '100%' }}>
      <Space style={{ width: '100%', justifyContent: 'space-between' }}>
        <Typography.Title level={3} style={{ margin: 0 }}>
          流水线模板
        </Typography.Title>
        <Button type="primary" onClick={() => navigate('/templates/new')}>
          新建模板
        </Button>
      </Space>
      <Table
        rowKey="id"
        loading={loading}
        columns={[
          { title: '名称', dataIndex: 'name' },
          { title: '版本', dataIndex: 'version' },
          {
            title: '状态',
            dataIndex: 'status',
            render: (status: FlowTemplate['status']) => (
              <Tag color={status === 'published' ? 'green' : 'default'}>{templateStatusLabel(status)}</Tag>
            ),
          },
          {
            title: '创建时间',
            dataIndex: 'created_at',
            render: (value: string) => formatDateTime(value),
          },
          {
            title: '操作',
            key: 'actions',
            render: (_, record) => (
              <Space wrap>
                {/* 仅已发布且开启手动触发的模板可启动，实例始终以最新发布快照运行 */}
                {record.status === 'published' && record.manual_trigger && (
                  <Button type="link" icon={<PlayCircleOutlined />} onClick={() => void handleStart(record)}>
                    启动
                  </Button>
                )}
                <Button type="link" icon={<EditOutlined />} onClick={() => navigate(`/templates/${record.id}/edit`)}>
                  编辑
                </Button>
                <Button type="link" icon={<CopyOutlined />} onClick={() => void handleCopy(record)}>
                  复制
                </Button>
                <Button type="link" onClick={() => navigate(`/templates/${record.id}/instances`)}>
                  查看实例
                </Button>
                <Button type="link" danger icon={<DeleteOutlined />} onClick={() => handleDelete(record)}>
                  删除
                </Button>
              </Space>
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

      <Modal
        title="启动参数"
        open={!!startTarget}
        onOk={() => void handleStartOk()}
        onCancel={() => setStartTarget(null)}
        confirmLoading={starting}
        okText="启动"
        cancelText="取消"
      >
        <Form form={startForm} labelCol={{ span: 6 }} wrapperCol={{ span: 18 }}>
          {startParams.map((item) => (
            <Form.Item
              key={item.name}
              name={item.name}
              label={item.name}
              required={item.required}
              rules={item.required ? [{ required: true, message: `请输入 ${item.name}` }] : undefined}
              extra={item.description || undefined}
            >
              <Input
                placeholder={
                  item.type === 'number'
                    ? '请输入数字'
                    : item.type === 'boolean'
                      ? 'true / false'
                      : '请输入字符串'
                }
              />
            </Form.Item>
          ))}
        </Form>
      </Modal>
    </Space>
  )
}
