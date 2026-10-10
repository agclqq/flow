import { CopyOutlined, MinusCircleOutlined, PlusOutlined } from '@ant-design/icons'
import { Button, Card, Form, Input, message, Select, Space, Switch, Typography } from 'antd'
import { useEffect, useState } from 'react'
import type { ConfigFormValues, FlowParam, NotifyTemplate } from '../../api/types'

type ConfigTabProps = {
  form: ReturnType<typeof Form.useForm<ConfigFormValues>>[0]
  notifyTemplates: NotifyTemplate[]
  loading?: boolean
  isNew?: boolean
  templateId?: number | null
  saving?: boolean
  onSave?: () => void | Promise<void>
}

const paramTypeOptions = [
  { value: 'string', label: '字符串' },
  { value: 'number', label: '数字' },
  { value: 'boolean', label: '布尔' },
]

/** 生成随机 webhook 密钥（6 位字母数字）。 */
export function randomWebhookSecret(): string {
  const charset = 'abcdefghjkmnpqrstuvwxyz23456789'
  const bytes = new Uint8Array(6)
  crypto.getRandomValues(bytes)
  return Array.from(bytes, (b) => charset[b % charset.length]).join('')
}

const PARAM_GRID = '1.2fr 0.8fr 1fr 72px 1.4fr 32px'

export default function ConfigTab({
  form,
  notifyTemplates,
  loading,
  isNew,
  templateId,
  saving,
  onSave,
}: ConfigTabProps) {
  const webhookEnabled = Form.useWatch('webhook', form)
  const cronEnabled = Form.useWatch('cron', form)
  const cronExpr = Form.useWatch('cron_expr', form)
  const webhookSecret = Form.useWatch('webhook_secret', form)

  useEffect(() => {
    if (isNew && !loading && !form.getFieldValue('params')) {
      form.setFieldValue('params', [])
    }
  }, [form, isNew, loading])

  // 打开 webhook 且密钥为空时，自动预生成一个默认密钥。
  useEffect(() => {
    if (webhookEnabled && !form.getFieldValue('webhook_secret')) {
      form.setFieldValue('webhook_secret', randomWebhookSecret())
    }
  }, [webhookEnabled, form])

  const webhookUrl = templateId
    ? `${window.location.origin}/api/v1/flow-templates/${templateId}/webhook`
    : ''

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', minHeight: 0 }}>
      <div style={{ flex: 1, overflowY: 'auto', minHeight: 0 }}>
        <Form
          form={form}
          layout="vertical"
          disabled={loading}
          initialValues={{
            manual: true,
            webhook: false,
            cron: false,
            cron_expr: '',
            require_params: false,
            params: [],
            owners: [],
            notify_timings: [],
          }}
        >
      <Card title="基本信息" size="small">
        <Form.Item label="流水线名称" name="name" rules={[{ required: true, message: '请输入名称' }]}>
          <Input placeholder="请输入流水线名称" />
        </Form.Item>
      </Card>

      <Card title="参数列表" size="small" style={{ marginTop: 16 }}>
        <Form.List name="params">
          {(fields, { add, remove }) => (
            <>
              {fields.length > 0 && (
                <div
                  style={{
                    display: 'grid',
                    gridTemplateColumns: PARAM_GRID,
                    gap: 8,
                    marginBottom: 4,
                    color: '#8c8c8c',
                    fontSize: 12,
                  }}
                >
                  <span>参数名</span>
                  <span>类型</span>
                  <span>默认值</span>
                  <span>必填</span>
                  <span>说明</span>
                  <span />
                </div>
              )}
              {fields.map((field) => (
                <div
                  key={field.key}
                  style={{
                    display: 'grid',
                    gridTemplateColumns: PARAM_GRID,
                    gap: 8,
                    alignItems: 'center',
                    marginBottom: 8,
                  }}
                >
                  <Form.Item
                    name={[field.name, 'name']}
                    rules={[{ required: true, message: '请输入参数名' }]}
                    style={{ marginBottom: 0 }}
                  >
                    <Input placeholder="例如 a" />
                  </Form.Item>
                  <Form.Item
                    name={[field.name, 'type']}
                    initialValue="string"
                    rules={[{ required: true, message: '请选择类型' }]}
                    style={{ marginBottom: 0 }}
                  >
                    <Select options={paramTypeOptions} />
                  </Form.Item>
                  <Form.Item name={[field.name, 'default_value']} style={{ marginBottom: 0 }}>
                    <Input placeholder="类型默认值" />
                  </Form.Item>
                  <Form.Item
                    name={[field.name, 'required']}
                    valuePropName="checked"
                    initialValue={false}
                    style={{ marginBottom: 0 }}
                  >
                    <Switch size="small" />
                  </Form.Item>
                  <Form.Item name={[field.name, 'description']} style={{ marginBottom: 0 }}>
                    <Input placeholder="参数说明" />
                  </Form.Item>
                  <Button type="text" danger icon={<MinusCircleOutlined />} onClick={() => remove(field.name)} />
                  <Form.Item name={[field.name, 'id']} hidden>
                    <Input />
                  </Form.Item>
                </div>
              ))}
              <Button type="dashed" block icon={<PlusOutlined />} onClick={() => add({ type: 'string' } as FlowParam)}>
                添加参数
              </Button>
              <Typography.Paragraph type="secondary" style={{ marginTop: 8, marginBottom: 0 }}>
                触发时未提供的参数使用默认值；未设置默认值时使用类型默认值（字符串空、数字 0、布尔 false）。
              </Typography.Paragraph>
            </>
          )}
        </Form.List>
      </Card>

      <TriggerSection
        form={form}
        webhookEnabled={webhookEnabled}
        cronEnabled={cronEnabled}
        cronExpr={cronExpr}
        webhookSecret={webhookSecret}
        webhookUrl={webhookUrl}
        templateId={templateId}
      />

      <Card title="负责人与通知" size="small" style={{ marginTop: 16 }}>
        <Form.Item label="模板负责人" name="owners">
          <Select mode="tags" placeholder="选择或输入负责人" />
        </Form.Item>
        <Form.Item label="通知时机" name="notify_timings">
          <Select
            mode="multiple"
            options={[
              { value: 'success', label: '流水线成功' },
              { value: 'failure', label: '流水线失败' },
            ]}
          />
        </Form.Item>
        <Form.Item label="通知模板" name="notify_template_id">
          <Select
            placeholder="选择通知模板"
            allowClear
            options={notifyTemplates.map((item) => ({ value: item.id, label: item.name }))}
          />
        </Form.Item>
      </Card>
        </Form>
      </div>
      <div
        style={{
          flexShrink: 0,
          display: 'flex',
          justifyContent: 'center',
          padding: '12px 16px',
          borderTop: '1px solid #f0f0f0',
          background: '#fff',
        }}
      >
        <Button type="primary" loading={saving} disabled={loading} onClick={() => void onSave?.()}>
          保存
        </Button>
      </div>
    </div>
  )
}

function TriggerSection({
  form,
  webhookEnabled,
  cronEnabled,
  cronExpr,
  webhookSecret,
  webhookUrl,
  templateId,
}: {
  form: ConfigTabProps['form']
  webhookEnabled?: boolean
  cronEnabled?: boolean
  cronExpr?: string
  webhookSecret?: string
  webhookUrl: string
  templateId?: number | null
}) {
  // cron 表达式防抖预览：解析并计算下次执行时间。
  const [cronNext, setCronNext] = useState<{ text: string; ok: boolean } | null>(null)
  // webhook curl 请求示例展开状态。
  const [showCurl, setShowCurl] = useState(false)
  const declaredParams = Form.useWatch('params', form) as FlowParam[] | undefined

  // 依据当前参数定义生成 curl 请求示例，body 取各参数默认值（无默认值用 <参数名> 占位）。
  const curlExample = (() => {
    const body = Object.fromEntries(
      (declaredParams ?? []).map((p) => [p.name, p.default_value || `<${p.name}>`]),
    )
    const secret = webhookSecret || '<secret>'
    const url = templateId ? `${webhookUrl}?secret=${secret}` : '<webhook_url>?secret=<secret>'
    return [
      `curl -X POST '${url}' \\`,
      `  -H 'Content-Type: application/json' \\`,
      `  -d '${JSON.stringify(body)}'`,
    ].join('\n')
  })()

  const copyCurl = async () => {
    if (!templateId || !webhookSecret) {
      message.warning('请先保存模板并设置密钥')
      return
    }
    await navigator.clipboard.writeText(curlExample)
    message.success('已复制 curl 示例')
  }
  useEffect(() => {
    if (!cronEnabled) {
      setCronNext(null)
      return
    }
    const expr = (cronExpr ?? '').trim()
    if (!expr) {
      setCronNext(null)
      return
    }
    const timer = setTimeout(async () => {
      try {
        const res = await fetch(`/api/v1/cron/next?expr=${encodeURIComponent(expr)}`)
        const body = (await res.json()) as {
          code: number
          data?: { valid: boolean; next?: string; error?: string }
        }
        const data = body.data
        if (data?.valid && data.next) {
          setCronNext({ text: `下次: ${data.next}`, ok: true })
        } else {
          setCronNext({ text: `表达式无效：${data?.error ?? ''}`, ok: false })
        }
      } catch {
        setCronNext(null)
      }
    }, 400)
    return () => clearTimeout(timer)
  }, [cronEnabled, cronExpr])

  return (
    <Card title="启动方式" size="small" style={{ marginTop: 16 }}>
      <Form.Item label="手动启动" name="manual" valuePropName="checked">
        <Switch />
      </Form.Item>

      <Form.Item label="Webhook" style={{ marginBottom: webhookEnabled ? 0 : undefined }}>
        <Space wrap size={8}>
          <Form.Item
            name="webhook"
            valuePropName="checked"
            noStyle
            onChange={() => {
              if (!form.getFieldValue('webhook_secret')) {
                form.setFieldValue('webhook_secret', randomWebhookSecret())
              }
            }}
          >
            <Switch />
          </Form.Item>
          {webhookEnabled && (
            <>
              {templateId ? (
                <span style={{ fontFamily: 'monospace', fontSize: 13 }}>{webhookUrl}?secret=</span>
              ) : (
                <Typography.Text type="secondary">保存模板后生成地址，密钥：</Typography.Text>
              )}
              <Form.Item noStyle name="webhook_secret">
                <Input style={{ width: 84, fontFamily: 'monospace', textAlign: 'center' }} />
              </Form.Item>
              <Button
                icon={<CopyOutlined />}
                onClick={async () => {
                  const secret = form.getFieldValue('webhook_secret')
                  if (!webhookUrl || !secret) {
                    message.warning('请先保存模板并设置密钥')
                    return
                  }
                  await navigator.clipboard.writeText(`${webhookUrl}?secret=${secret}`)
                  message.success('已复制完整 Webhook 地址')
                }}
              />
            </>
          )}
        </Space>
      </Form.Item>
      {webhookEnabled && (
        <>
          <Typography.Paragraph type="secondary" style={{ marginTop: 0 }}>
            POST 该地址即触发流水线，body JSON 对象作为入参。
            <Typography.Link onClick={() => setShowCurl((v) => !v)} style={{ marginLeft: 4 }}>
              查看示例
            </Typography.Link>
            <CopyOutlined
              title="复制 curl 示例"
              style={{ marginLeft: 8, cursor: 'pointer' }}
              onClick={() => void copyCurl()}
            />
          </Typography.Paragraph>
          {showCurl && (
            <pre
              style={{
                background: '#f6f6f6',
                padding: 12,
                borderRadius: 6,
                fontSize: 12,
                margin: '0 0 12px',
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-all',
              }}
            >
              {curlExample}
            </pre>
          )}
        </>
      )}

      <Form.Item label="定时 Cron" name="cron" valuePropName="checked" style={{ marginTop: 16 }}>
        <Switch />
      </Form.Item>
      {cronEnabled && (
        <>
          <Form.Item style={{ marginBottom: 4 }}>
            <div style={{ fontFamily: 'monospace', fontSize: 12, color: '#8c8c8c' }}>
              minute&nbsp;&nbsp;hour&nbsp;&nbsp;day&nbsp;&nbsp;month&nbsp;&nbsp;weekday
            </div>
          </Form.Item>
          <Form.Item
            name="cron_expr"
            rules={[{ required: true, message: '请输入 Cron 表达式' }]}
            style={{ marginBottom: cronNext ? 4 : undefined }}
          >
            <Input placeholder="0 0 * * *" style={{ fontFamily: 'monospace' }} />
          </Form.Item>
          {cronNext && (
            <div style={{ marginTop: 0, marginBottom: 8, fontSize: 13, color: cronNext.ok ? '#52c41a' : '#ff4d4f' }}>
              {cronNext.text}
            </div>
          )}
        </>
      )}
    </Card>
  )
}
