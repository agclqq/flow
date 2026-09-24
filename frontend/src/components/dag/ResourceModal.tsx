import { Form, Input, InputNumber, Modal, Select, message } from 'antd'
import { useState } from 'react'
import { createExecutorResource } from '../../api/executor-resources'
import type { ExecutorResource, RuntimeKind } from '../../api/types'

type ResourceModalProps = {
  open: boolean
  type: RuntimeKind | string
  onClose: () => void
  onCreated: (resource: ExecutorResource) => void
}

type ResourceFormValues = {
  name: string
  host?: string
  port?: number
  username?: string
  auth_type?: 'password' | 'key'
  password?: string
  private_key?: string
  work_dir?: string
  kubeconfig?: string
  namespace?: string
  image?: string
}

/** 执行资源新建弹窗：按资源类型展示对应连接配置。 */
export default function ResourceModal({ open, type, onClose, onCreated }: ResourceModalProps) {
  const [form] = Form.useForm<ResourceFormValues>()
  const [submitting, setSubmitting] = useState(false)
  const authType = Form.useWatch('auth_type', form)

  const handleOk = async () => {
    try {
      const values = await form.validateFields()
      setSubmitting(true)
      const config =
        type === 'ssh'
          ? {
              host: values.host ?? '',
              port: values.port ?? 22,
              username: values.username ?? '',
              auth_type: values.auth_type ?? 'password',
              password: values.password ?? '',
              private_key: values.private_key ?? '',
              work_dir: values.work_dir ?? '',
            }
          : {
              kubeconfig: values.kubeconfig ?? '',
              namespace: values.namespace ?? 'default',
              image: values.image ?? '',
            }
      const resource = await createExecutorResource({
        name: values.name,
        type,
        config: JSON.stringify(config),
      })
      message.success('资源已创建')
      form.resetFields()
      onCreated(resource)
    } catch (error) {
      if (error instanceof Error && error.message) {
        message.error(error.message)
      }
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Modal
      title="新建执行资源"
      open={open}
      onCancel={onClose}
      onOk={() => void handleOk()}
      confirmLoading={submitting}
      okText="创建"
      cancelText="取消"
      destroyOnHidden
      width={560}
    >
      <Form form={form} layout="vertical" initialValues={{ port: 22, auth_type: 'password' }}>
        <Form.Item label="资源名称" name="name" rules={[{ required: true, message: '请输入资源名称' }]}>
          <Input placeholder="例如：测试服务器 / 集群 A" />
        </Form.Item>
        {type === 'ssh' ? <SshFields authType={authType} /> : <K8sFields />}
      </Form>
    </Modal>
  )
}

function SshFields({ authType }: { authType?: 'password' | 'key' }) {
  return (
    <>
      <Form.Item label="主机" name="host" rules={[{ required: true, message: '请输入主机地址' }]}>
        <Input placeholder="例如：10.0.0.8" />
      </Form.Item>
      <Form.Item label="端口" name="port">
        <InputNumber min={1} max={65535} style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item label="用户名" name="username" rules={[{ required: true, message: '请输入用户名' }]}>
        <Input />
      </Form.Item>
      <Form.Item label="认证方式" name="auth_type">
        <Select
          options={[
            { value: 'password', label: '密码' },
            { value: 'key', label: '私钥' },
          ]}
        />
      </Form.Item>
      {authType === 'key' ? (
        <Form.Item label="私钥 (PEM)" name="private_key">
          <Input.TextArea rows={4} placeholder="-----BEGIN OPENSSH PRIVATE KEY-----" />
        </Form.Item>
      ) : (
        <Form.Item label="密码" name="password">
          <Input.Password />
        </Form.Item>
      )}
      <Form.Item label="工作目录（可选）" name="work_dir">
        <Input placeholder="脚本在远端的工作目录" />
      </Form.Item>
    </>
  )
}

function K8sFields() {
  return (
    <>
      <Form.Item
        label="kubeconfig"
        name="kubeconfig"
        rules={[{ required: true, message: '请粘贴 kubeconfig 内容' }]}
      >
        <Input.TextArea rows={8} placeholder="apiVersion: v1&#10;clusters: ..." />
      </Form.Item>
      <Form.Item label="命名空间" name="namespace" initialValue="default">
        <Input />
      </Form.Item>
      <Form.Item label="默认镜像（可选）" name="image">
        <Input placeholder="默认 busybox:1.36，脚本在其中运行" />
      </Form.Item>
    </>
  )
}
