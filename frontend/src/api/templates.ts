import { ApiError, apiRequest, buildQuery } from './client'
import type {
  ConfigFormValues,
  FlowNode,
  FlowParam,
  FlowTemplate,
  GraphPayload,
  Paginated,
  TriggerConfig,
  VersionDetail,
  VersionListItem,
} from './types'

export async function listTemplates(offset = 0, limit = 20): Promise<Paginated<FlowTemplate>> {
  return apiRequest<Paginated<FlowTemplate>>(`/flow-templates${buildQuery({ offset, limit })}`)
}

export async function getTemplate(id: number): Promise<FlowTemplate> {
  return apiRequest<FlowTemplate>(`/flow-templates/${id}`)
}

export async function createTemplate(name: string): Promise<FlowTemplate> {
  return apiRequest<FlowTemplate>('/flow-templates', {
    method: 'POST',
    body: JSON.stringify({ name }),
  })
}

export async function updateTemplate(
  id: number,
  payload: { name: string; owners?: string[]; description?: string },
): Promise<FlowTemplate> {
  return apiRequest<FlowTemplate>(`/flow-templates/${id}`, {
    method: 'PUT',
    body: JSON.stringify(payload),
  })
}

export async function cloneTemplate(id: number): Promise<FlowTemplate> {
  return apiRequest<FlowTemplate>(`/flow-templates/${id}/clone`, {
    method: 'POST',
    body: JSON.stringify({}),
  })
}

export async function deleteTemplate(id: number): Promise<void> {
  await apiRequest<void>(`/flow-templates/${id}`, { method: 'DELETE' })
}

export async function publishTemplate(id: number): Promise<FlowTemplate> {
  return apiRequest<FlowTemplate>(`/flow-templates/${id}/publish`, {
    method: 'POST',
    body: JSON.stringify({}),
  })
}

export async function listVersions(templateId: number): Promise<VersionListItem[]> {
  const res = await apiRequest<{ items: VersionListItem[] }>(`/flow-templates/${templateId}/versions`)
  return res.items ?? []
}

export async function getVersion(templateId: number, versionId: number): Promise<VersionDetail> {
  return apiRequest<VersionDetail>(`/flow-templates/${templateId}/versions/${versionId}`)
}

function pickContentVersionFromList(versions: VersionListItem[]): VersionListItem | null {
  const draft = versions.find((item) => item.status === 'draft')
  if (draft) return draft
  const published = versions.find((item) => item.status === 'published')
  return published ?? null
}

export async function getEditingVersion(templateId: number): Promise<{
  version_id: number
  version: number
  is_draft: boolean
  status: string
}> {
  try {
    return await apiRequest(`/flow-templates/${templateId}/editing-version`)
  } catch (err) {
    // 兼容未重启的后端（尚无 editing-version 路由时回退到 versions 列表）。
    // 1004 = 统一响应格式的资源不存在；404 兼容旧版后端。
    if (err instanceof ApiError && (err.status === 1004 || err.status === 404)) {
      const versions = await listVersions(templateId)
      const pick = pickContentVersionFromList(versions)
      if (!pick) throw err
      return {
        version_id: pick.id,
        version: pick.version,
        is_draft: pick.status === 'draft',
        status: pick.status,
      }
    }
    throw err
  }
}

export async function ensureDraftVersion(templateId: number): Promise<{
  version_id: number
  version: number
  is_draft: boolean
  status: string
}> {
  return apiRequest(`/flow-templates/${templateId}/editing-version/ensure-draft`, {
    method: 'POST',
    body: JSON.stringify({}),
  })
}

/** @deprecated 使用 getEditingVersion */
export async function getDraftVersionId(templateId: number): Promise<number> {
  const editing = await getEditingVersion(templateId)
  return editing.version_id
}

export async function listParams(templateId: number, versionId: number): Promise<FlowParam[]> {
  const res = await apiRequest<{ items: FlowParam[] }>(
    `/flow-templates/${templateId}/versions/${versionId}/params`,
  )
  return res.items ?? []
}

export async function listNodes(templateId: number, versionId: number): Promise<FlowNode[]> {
  const res = await apiRequest<{ items: FlowNode[] }>(
    `/flow-templates/${templateId}/versions/${versionId}/nodes`,
  )
  return res.items ?? []
}

export async function getTrigger(templateId: number): Promise<TriggerConfig | null> {
  try {
    return await apiRequest<TriggerConfig>(`/flow-templates/${templateId}/trigger`)
  } catch {
    return null
  }
}

export async function upsertTrigger(templateId: number, config: Omit<TriggerConfig, 'id' | 'template_id'>) {
  return apiRequest<TriggerConfig>(`/flow-templates/${templateId}/trigger`, {
    method: 'PUT',
    body: JSON.stringify(config),
  })
}

async function syncParams(
  templateId: number,
  versionId: number,
  existing: FlowParam[],
  next: FlowParam[],
): Promise<void> {
  const existingById = new Map(existing.filter((item) => item.id).map((item) => [item.id!, item]))
  const nextIds = new Set(next.filter((item) => item.id).map((item) => item.id!))

  for (const item of existing) {
    if (item.id && !nextIds.has(item.id)) {
      await apiRequest<void>(
        `/flow-templates/${templateId}/versions/${versionId}/params/${item.id}`,
        { method: 'DELETE' },
      )
    }
  }

  for (const item of next) {
    const payload = {
      name: item.name,
      type: item.type,
      default_value: item.default_value ?? '',
      required: item.required ?? false,
      description: item.description ?? '',
    }

    if (item.id && existingById.has(item.id)) {
      await apiRequest<FlowParam>(
        `/flow-templates/${templateId}/versions/${versionId}/params/${item.id}`,
        { method: 'PUT', body: JSON.stringify(payload) },
      )
    } else if (item.name) {
      await apiRequest<FlowParam>(`/flow-templates/${templateId}/versions/${versionId}/params`, {
        method: 'POST',
        body: JSON.stringify(payload),
      })
    }
  }
}

export async function saveDraftConfig(
  templateId: number,
  versionId: number,
  values: ConfigFormValues,
  existingParams: FlowParam[],
): Promise<void> {
  await updateTemplate(templateId, {
    name: values.name,
    owners: values.owners ?? [],
  })
  await syncParams(templateId, versionId, existingParams, values.params ?? [])
  await upsertTrigger(templateId, {
    manual: values.manual ?? false,
    webhook: values.webhook ?? false,
    cron: values.cron ?? false,
    cron_expr: values.cron_expr ?? '',
    require_params: values.require_params ?? false,
    webhook_secret: values.webhook_secret ?? '',
  })
}

export async function putGraph(
  templateId: number,
  versionId: number,
  graph: GraphPayload,
): Promise<FlowNode[]> {
  const refByClient = new Map<string, number>()
  let tempRef = -1
  const nextRef = (clientId: string, backendId?: number) => {
    if (backendId) {
      refByClient.set(clientId, backendId)
      return backendId
    }
    const ref = tempRef--
    refByClient.set(clientId, ref)
    return ref
  }

  const nodePayloads = graph.nodes.map((node) => ({
    id: nextRef(node.clientId, node.backendId),
    type: node.type,
    name: node.name,
    pre_script: node.pre_script,
    script: node.script,
    post_script: node.post_script,
    owners: node.owners ?? [],
    approvers: node.approvers ?? [],
    description: node.description ?? '',
    start_mode: node.start_mode || 'any',
    start_script: node.start_script ?? '',
    required_predecessor_ids: node.required_predecessor_ids ?? [],
    position_x: node.position.x,
    position_y: node.position.y,
  }))

  const refByBackend = new Map<number, number>()
  for (const node of graph.nodes) {
    const ref = refByClient.get(node.clientId)
    if (ref != null && node.backendId) {
      refByBackend.set(node.backendId, ref)
    }
  }

  for (const payload of nodePayloads) {
    if (!payload.required_predecessor_ids?.length) continue
    payload.required_predecessor_ids = payload.required_predecessor_ids.map(
      (predId) => refByBackend.get(predId) ?? predId,
    )
  }

  const edgePayloads = graph.edges.map((edge) => {
    const edgeRef = edge.backendId ?? tempRef--
    return {
      id: edgeRef,
      type: 'edge' as const,
      name: `edge_${edge.source}_${edge.target}`,
      from_node_id: refByClient.get(edge.source) ?? 0,
      to_node_id: refByClient.get(edge.target) ?? 0,
    }
  })

  const res = await apiRequest<{ items: FlowNode[] }>(
    `/flow-templates/${templateId}/versions/${versionId}/graph`,
    { method: 'PUT', body: JSON.stringify({ nodes: [...nodePayloads, ...edgePayloads] }) },
  )
  return res.items ?? []
}

export async function copyTemplate(sourceId: number, _sourceName?: string): Promise<FlowTemplate> {
  return cloneTemplate(sourceId)
}
