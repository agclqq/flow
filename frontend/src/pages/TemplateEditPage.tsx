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

function parseEditTab(tab: string | null, defaultTab: EditTabKey): EditTabKey {
  if (tab && EDIT_TAB_KEYS.includes(tab as EditTabKey)) {
    return tab as EditTabKey
  }
  return defaultTab
}

export default function TemplateEditPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const isNew = !id
  const defaultEditTab: EditTabKey = isNew ? 'config' : 'dag'
  const activeTab = parseEditTab(searchParams.get('tab'), defaultEditTab)
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
  const [configSaving, setConfigSaving] = useState(false)
  const [publishing, setPublishing] = useState(false)
  const [notifyTemplates, setNotifyTemplates] = useState<{ id: number; name: string; content: string }[]>([])
  const [existingParams, setExistingParams] = useState<FlowParam[]>([])
  const [dagNeedsName, setDagNeedsName] = useState(false)

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
    const nodes = await listNodes(created.id, editing.version_id)
    setCreatedTemplateId(created.id)
    setVersionId(editing.version_id)
    setIsDraftVersion(true)
    syncPrefetchedDag(created.id, editing.version_id, nodes)
    setDagNeedsName(false)
    const tabQuery = activeTab !== defaultEditTab ? `?tab=${activeTab}` : ''
    navigate(`/templates/${created.id}/edit${tabQuery}`, { replace: true })
    return { templateId: created.id, versionId: editing.version_id }
  }

  // 新建页进入 DAG：与编辑页一致，有名称则自动创建模板并加载草稿（含默认开始节点）。
  useEffect(() => {
    if (activeTab !== 'dag' || templateId) {
      setDagNeedsName(false)
      return
    }
    const name = String(form.getFieldValue('name') ?? '').trim()
    if (!name) {
      setDagNeedsName(true)
      return
    }
    setDagNeedsName(false)
    void (async () => {
      try {
        await ensureTemplate()
      } catch {
        setDagNeedsName(true)
      }
    })()
    // eslint-disable-next-line react-hooks/exhaustive-deps -- 仅在进入 DAG 且尚未有模板时引导创建
  }, [activeTab, templateId])

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

  const handleSaveConfig = async () => {
    setConfigSaving(true)
    try {
      await form.validateFields()
      const values = form.getFieldsValue(true) as ConfigFormValues
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
      if (writableVersionId !== ids.versionId) {
        setVersionId(writableVersionId)
        setIsDraftVersion(true)
      }
      const params = await listParams(ids.templateId, writableVersionId)
      setExistingParams(params)
      form.setFieldValue('params', params)
      message.success('配置已保存')
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存配置失败')
    } finally {
      setConfigSaving(false)
    }
  }

  const handlePublish = async () => {
    setPublishing(true)
    try {
      await form.validateFields()
      // 未打开「配置信息」Tab 时 Switch 未挂载，需从 Form 全量 store 读取（preserve 字段）。
      const values = form.getFieldsValue(true) as ConfigFormValues
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
            <Typography.Text type="secondary">（基于已发布版本编辑，发布前将写入新草稿）</Typography.Text>
          )}
        </Space>
        <Button type="primary" loading={publishing} disabled={isNew && !templateId} onClick={() => void handlePublish()}>
          发布
        </Button>
      </Space>
      <Tabs
        activeKey={activeTab}
        onChange={(key) => {
          setSearchParams(
            (prev) => {
              const next = new URLSearchParams(prev)
              if (key === defaultEditTab) {
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
            forceRender: true,
            children: (
              <div style={{ height: 'calc(100vh - 200px)', minHeight: 360 }}>
                <ConfigTab
                  form={form}
                  notifyTemplates={notifyTemplates}
                  loading={loading}
                  isNew={isNew}
                  templateId={templateId}
                  saving={configSaving}
                  onSave={handleSaveConfig}
                />
              </div>
            ),
          },
          {
            key: 'dag',
            label: 'DAG 编排',
            children: (
              <>
                {dagNeedsName && (
                  <Typography.Paragraph type="warning" style={{ marginBottom: 8 }}>
                    请先在「配置信息」中填写模板名称；填写后回到本页或切换 Tab 即可开始编排（与编辑已有模板相同）。
                  </Typography.Paragraph>
                )}
                <Suspense fallback={<Spin description="加载 DAG 编辑器..." />}>
                  <DagEditor
                    ref={dagRef}
                    editable
                    writable={isDraftVersion && !!templateId && !!versionId}
                    templateId={templateId ?? undefined}
                    versionId={versionId ?? undefined}
                    paramNames={paramNames}
                    prefetchedNodes={prefetchedDagNodes}
                    prefetchedKey={prefetchedDagKey}
                  />
                </Suspense>
              </>
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
