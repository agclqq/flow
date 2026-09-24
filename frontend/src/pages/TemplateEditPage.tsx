import { ArrowLeftOutlined } from '@ant-design/icons'
import { Button, Form, Modal, Space, Spin, Tabs, Typography, message } from 'antd'
import { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { getNotifyBindings, listNotifyTemplates, upsertNotifyBindings } from '../api/notify'
import {
  createTemplate,
  ensureDraftVersion,
  getEditingVersion,
  getTemplate,
  getTrigger,
  listNodes,
  listParams,
  publishTemplate,
  saveDraftConfig,
} from '../api/templates'
import type { ConfigFormValues, FlowNode, FlowParam } from '../api/types'
import type { DagEditorHandle } from '../components/dag/DagEditor'
import { findConnectivityIssues } from '../components/dag/dagUtils'
import ConfigTab from '../components/template/ConfigTab'
import SystemParamsTab from '../components/template/SystemParamsTab'

const DagEditor = lazy(() => import('../components/dag/DagEditor'))

const EDIT_TAB_KEYS = ['config', 'dag', 'system-params'] as const
type EditTabKey = (typeof EDIT_TAB_KEYS)[number]

function parseEditTab(tab: string | null): EditTabKey {
  if (tab && EDIT_TAB_KEYS.includes(tab as EditTabKey)) {
    return tab as EditTabKey
  }
  return 'config'
}

export default function TemplateEditPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const activeTab = parseEditTab(searchParams.get('tab'))
  const isNew = !id
  const routeTemplateId = id ? Number(id) : null
  const [form] = Form.useForm<ConfigFormValues>()
  const dagRef = useRef<DagEditorHandle>(null)
  const [createdTemplateId, setCreatedTemplateId] = useState<number | null>(null)
  const templateId = routeTemplateId ?? createdTemplateId
  const [versionId, setVersionId] = useState<number | null>(null)
  const [isDraftVersion, setIsDraftVersion] = useState(true)
  const [prefetchedDagNodes, setPrefetchedDagNodes] = useState<FlowNode[] | null>(null)
  const [prefetchedDagKey, setPrefetchedDagKey] = useState('')
  const [loading, setLoading] = useState(!isNew)
  const [saving, setSaving] = useState(false)
  const [publishing, setPublishing] = useState(false)
  const [notifyTemplates, setNotifyTemplates] = useState<{ id: number; name: string; content: string }[]>([])
  const [existingParams, setExistingParams] = useState<FlowParam[]>([])

  const watchedParams = Form.useWatch('params', form) as FlowParam[] | undefined
  const paramNames = useMemo(
    () => watchedParams?.map((item) => item.name).filter(Boolean) ?? [],
    [watchedParams],
  )

  const syncPrefetchedDag = useCallback((tid: number, vid: number, nodes: FlowNode[]) => {
    setPrefetchedDagNodes(nodes)
    setPrefetchedDagKey(`${tid}:${vid}`)
  }, [])

  const loadTemplate = useCallback(async (tid: number) => {
    setLoading(true)
    setVersionId(null)
    setPrefetchedDagNodes(null)
    setPrefetchedDagKey('')
    try {
      const [tpl, editing] = await Promise.all([getTemplate(tid), getEditingVersion(tid)])
      // 兼容旧数据：仅有已发布版本（无草稿）时自动创建草稿，
      // 保证编辑始终发生在可写草稿上，节点配置可随自动保存持久化。
      const effective = editing.is_draft ? editing : await ensureDraftVersion(tid)
      setVersionId(effective.version_id)
      setIsDraftVersion(effective.is_draft)
      const [params, trigger, bindings, templates, nodes] = await Promise.all([
        listParams(tid, effective.version_id),
        getTrigger(tid),
        getNotifyBindings(tid),
        listNotifyTemplates(),
        listNodes(tid, effective.version_id),
      ])
      setExistingParams(params)
      setNotifyTemplates(templates)
      syncPrefetchedDag(tid, effective.version_id, nodes)
      const binding = bindings.find((item) => item.node_id === 0 || item.node_id == null)
      form.setFieldsValue({
        name: tpl.name,
        owners: tpl.owners ?? [],
        params,
        manual: trigger?.manual ?? true,
        webhook: trigger?.webhook ?? false,
        cron: trigger?.cron ?? false,
        cron_expr: trigger?.cron_expr ?? '',
        require_params: trigger?.require_params ?? false,
        webhook_secret: trigger?.webhook_secret ?? '',
        notify_timings: binding?.timings ?? [],
        notify_template_id: binding?.notify_template_id,
      })
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载模板失败')
    } finally {
      setLoading(false)
    }
  }, [form, syncPrefetchedDag])

  useEffect(() => {
    if (routeTemplateId) {
      setCreatedTemplateId(null)
    }
  }, [routeTemplateId])

  useEffect(() => {
    if (!isNew && templateId) {
      void loadTemplate(templateId)
    } else if (isNew) {
      void listNotifyTemplates().then(setNotifyTemplates)
      form.setFieldsValue({
        manual: true,
        webhook: false,
        cron: false,
        cron_expr: '',
        require_params: false,
        params: [],
        owners: [],
        notify_timings: [],
      })
    }
  }, [form, isNew, loadTemplate, templateId])

  const ensureTemplate = async (): Promise<{ templateId: number; versionId: number }> => {
    const values = await form.validateFields(['name'])
    if (templateId && versionId) {
      return { templateId, versionId }
    }
    const created = await createTemplate(values.name)
    const editing = await getEditingVersion(created.id)
    setCreatedTemplateId(created.id)
    setVersionId(editing.version_id)
    setIsDraftVersion(true)
    navigate(`/templates/${created.id}/edit`, { replace: true })
    return { templateId: created.id, versionId: editing.version_id }
  }

  const resolveWritableVersion = async (
    tid: number,
    vid: number,
    draft: boolean,
  ): Promise<number> => {
    if (draft) {
      return vid
    }
    const next = await ensureDraftVersion(tid)
    return next.version_id
  }

  /** 收集流水线连通性问题：优先用画布实时数据，编辑器未挂载时回退到服务端数据。 */
  const collectConnectivityIssues = useCallback(async (): Promise<string[]> => {
    const editor = dagRef.current
    if (editor) {
      return editor.validateConnectivity()
    }
    if (!templateId || !versionId) {
      return []
    }
    const nodes = await listNodes(templateId, versionId)
    return findConnectivityIssues(
      nodes
        .filter((node) => node.type !== 'edge' && node.id)
        .map((node) => ({ id: node.id!, type: node.type, name: node.name })),
      nodes
        .filter((node) => node.type === 'edge' && node.from_node_id && node.to_node_id)
        .map((node) => ({ source: node.from_node_id!, target: node.to_node_id! })),
    )
  }, [templateId, versionId])

  const handleSave = async () => {
    setSaving(true)
    try {
      const values = await form.validateFields()
      const issues = await collectConnectivityIssues()
      const ids = await ensureTemplate()
      const writableVersionId = await resolveWritableVersion(
        ids.templateId,
        ids.versionId,
        isDraftVersion,
      )
      await saveDraftConfig(ids.templateId, writableVersionId, values, existingParams)
      if (values.notify_template_id && values.notify_timings?.length) {
        await upsertNotifyBindings(ids.templateId, {
          notify_template_id: values.notify_template_id,
          timings: values.notify_timings,
        })
      }
      await dagRef.current?.saveNodeConfigs(writableVersionId)
      if (writableVersionId !== ids.versionId) {
        setVersionId(writableVersionId)
        setIsDraftVersion(true)
      }
      const params = await listParams(ids.templateId, writableVersionId)
      setExistingParams(params)
      form.setFieldValue('params', params)
      message.success('已保存')
      // 连通性不完整只提示，不阻拦保存。
      if (issues.length > 0) {
        message.warning(`流水线尚不完整：${issues.join('；')}`)
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const handlePublish = async () => {
    setPublishing(true)
    try {
      const values = await form.validateFields()
      // 连通性不完整则禁止发布，弹窗二次提醒。
      const issues = await collectConnectivityIssues()
      if (issues.length > 0) {
        Modal.warning({
          title: '流水线尚不完整，无法发布',
          content: (
            <ul style={{ margin: '8px 0 0', paddingLeft: 20 }}>
              {issues.map((issue) => (
                <li key={issue}>{issue}</li>
              ))}
            </ul>
          ),
          okText: '知道了',
        })
        return
      }
      const ids = await ensureTemplate()
      const writableVersionId = await resolveWritableVersion(
        ids.templateId,
        ids.versionId,
        isDraftVersion,
      )
      await saveDraftConfig(ids.templateId, writableVersionId, values, existingParams)
      if (values.notify_template_id && values.notify_timings?.length) {
        await upsertNotifyBindings(ids.templateId, {
          notify_template_id: values.notify_template_id,
          timings: values.notify_timings,
        })
      }
      await dagRef.current?.saveNodeConfigs(writableVersionId)
      await publishTemplate(ids.templateId)
      message.success('发布成功')
      await loadTemplate(ids.templateId)
    } catch (error) {
      message.error(error instanceof Error ? error.message : '发布失败')
    } finally {
      setPublishing(false)
    }
  }

  return (
    <Space orientation="vertical" size={16} style={{ width: '100%' }}>
      <Space style={{ width: '100%', justifyContent: 'space-between' }}>
        <Space>
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/templates')}>
            返回列表
          </Button>
          <Typography.Title level={3} style={{ margin: 0 }}>
            {isNew ? '新建流水线模板' : `编辑流水线模板 #${templateId}`}
          </Typography.Title>
          {!isNew && !isDraftVersion && !loading && (
            <Typography.Text type="secondary">（基于已发布版本编辑，保存时将创建新版本）</Typography.Text>
          )}
        </Space>
        <Space>
          <Button loading={saving} onClick={() => void handleSave()}>
            保存
          </Button>
          <Button type="primary" loading={publishing} disabled={isNew && !templateId} onClick={() => void handlePublish()}>
            发布
          </Button>
        </Space>
      </Space>
      <Tabs
        activeKey={activeTab}
        onChange={(key) => {
          setSearchParams(
            (prev) => {
              const next = new URLSearchParams(prev)
              if (key === 'config') {
                next.delete('tab')
              } else {
                next.set('tab', key)
              }
              return next
            },
            { replace: true },
          )
        }}
        items={[
          {
            key: 'config',
            label: '配置信息',
            children: (
              <ConfigTab
                form={form}
                notifyTemplates={notifyTemplates}
                loading={loading}
                isNew={isNew}
                templateId={templateId}
              />
            ),
          },
          {
            key: 'dag',
            label: 'DAG 编排',
            children: (
              <Suspense fallback={<Spin description="加载 DAG 编辑器..." />}>
                <DagEditor
                  ref={dagRef}
                  editable
                  writable={isDraftVersion}
                  templateId={templateId ?? undefined}
                  versionId={versionId ?? undefined}
                  paramNames={paramNames}
                  prefetchedNodes={prefetchedDagNodes}
                  prefetchedKey={prefetchedDagKey}
                />
              </Suspense>
            ),
          },
          {
            key: 'system-params',
            label: '系统参数',
            children: <SystemParamsTab />,
          },
        ]}
      />
    </Space>
  )
}
