import { apiRequest } from './client'
import type { K8sConfig, UpsertK8sConfigRequest } from './types'

export async function listK8sConfigs(): Promise<K8sConfig[]> {
  const res = await apiRequest<{ items: K8sConfig[] }>('/k8s-configs')
  return res.items ?? []
}

export async function createK8sConfig(payload: UpsertK8sConfigRequest): Promise<K8sConfig> {
  return apiRequest<K8sConfig>('/k8s-configs', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}
