import { Form, Input, Modal, Radio, message } from 'antd'
import { useState } from 'react'
import { createK8sConfig } from '../../api/k8s-configs'
import type { K8sConfig } from '../../api/types'

type K8sAuthModalProps = {
  open: boolean
  onClose: () => void
  onCreated: (config: K8sConfig) => void
}

type FormValues = {
  name: string
  auth_mode: 'kubeconfig' | 'token'
  kubeconfig?: string
  server?: string
  token?: string
  namespace?: string
}

export default function K8sAuthModal({ open, onClose, onCreated }: K8sAuthModalProps) {
  const [form] = Form.useForm<FormValues>()
  const [submitting, setSubmitting] = useState(false)
  const authMode = Form.useWatch('auth_mode', form) ?? 'kubeconfig'

  const handleOk = async () => {
    try {
      const values = await form.validateFields()
      setSubmitting(true)
      const created = await createK8sConfig({
        name: values.name,
        namespace: values.namespace ?? 'default',
        kubeconfig: values.auth_mode === 'kubeconfig' ? values.kubeconfig ?? '' : '',
        server: values.auth_mode === 'token' ? values.server ?? '' : '',
        token: values.auth_mode === 'token' ? values.token ?? '' : '',
      })
      message.success('K8s 认证已创建')
      form.resetFields()
      onCreated(created)
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
      title="新建 K8s 认证"
      open={open}
      onCancel={onClose}
      onOk={() => void handleOk()}
      confirmLoading={submitting}
      okText="创建"
      cancelText="取消"
      destroyOnHidden
      width={560}
    >
      <Form form={form} layout="vertical" initialValues={{ auth_mode: 'kubeconfig', namespace: 'default' }}>
        <Form.Item label="名称" name="name" rules={[{ required: true, message: '请输入名称' }]}>
          <Input placeholder="例如：生产集群" />
        </Form.Item>
        <Form.Item label="认证方式" name="auth_mode">
          <Radio.Group>
            <Radio value="kubeconfig">kubeconfig</Radio>
            <Radio value="token">Server + Token</Radio>
          </Radio.Group>
        </Form.Item>
        {authMode === 'kubeconfig' ? (
          <Form.Item
            label="kubeconfig"
            name="kubeconfig"
            rules={[{ required: true, message: '请粘贴 kubeconfig' }]}
          >
            <Input.TextArea rows={8} placeholder="apiVersion: v1&#10;clusters: ..." />
          </Form.Item>
        ) : (
          <>
            <Form.Item label="API Server" name="server" rules={[{ required: true, message: '请输入 API Server' }]}>
              <Input placeholder="https://k8s.example.com:6443" />
            </Form.Item>
            <Form.Item label="Token" name="token" rules={[{ required: true, message: '请输入 Token' }]}>
              <Input.Password />
            </Form.Item>
          </>
        )}
        <Form.Item label="默认命名空间" name="namespace">
          <Input />
        </Form.Item>
      </Form>
    </Modal>
  )
}
