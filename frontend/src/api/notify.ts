import { apiRequest } from './client'
import type { NotifyBinding, NotifyTemplate, Paginated } from './types'

let notifyTemplatesCache: NotifyTemplate[] | null = null
let notifyTemplatesInflight: Promise<NotifyTemplate[]> | null = null

export async function listNotifyTemplates(options?: { refresh?: boolean }): Promise<NotifyTemplate[]> {
  if (!options?.refresh && notifyTemplatesCache) {
    return notifyTemplatesCache
  }
  if (!options?.refresh && notifyTemplatesInflight) {
    return notifyTemplatesInflight
  }
  notifyTemplatesInflight = (async () => {
    try {
      const res = await apiRequest<Paginated<NotifyTemplate>>('/notify-templates')
      notifyTemplatesCache = res.items ?? []
      return notifyTemplatesCache
    } catch {
      return notifyTemplatesCache ?? []
    } finally {
      notifyTemplatesInflight = null
    }
  })()
  return notifyTemplatesInflight
}

export async function getNotifyBindings(templateId: number): Promise<NotifyBinding[]> {
  try {
    const res = await apiRequest<{ items: NotifyBinding[] }>(`/flow-templates/${templateId}/notify-bindings`)
    return res.items ?? []
  } catch {
    return []
  }
}

export async function upsertNotifyBindings(
  templateId: number,
  binding: Omit<NotifyBinding, 'id' | 'template_id'>,
): Promise<NotifyBinding[]> {
  const res = await apiRequest<{ items: NotifyBinding[] }>(`/flow-templates/${templateId}/notify-bindings`, {
    method: 'PUT',
    body: JSON.stringify({
      bindings: [{ ...binding, node_id: binding.node_id ?? 0 }],
    }),
  })
  return res.items ?? []
}
