import { apiRequest } from './client'
import type { ExecutorResource, UpsertExecutorResourceRequest } from './types'

export async function listExecutorResources(): Promise<ExecutorResource[]> {
  const res = await apiRequest<{ items: ExecutorResource[] }>('/executor-resources')
  return res.items ?? []
}

export async function createExecutorResource(
  payload: UpsertExecutorResourceRequest,
): Promise<ExecutorResource> {
  return apiRequest<ExecutorResource>('/executor-resources', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export async function updateExecutorResource(
  id: number,
  payload: UpsertExecutorResourceRequest,
): Promise<ExecutorResource> {
  return apiRequest<ExecutorResource>(`/executor-resources/${id}`, {
    method: 'PUT',
    body: JSON.stringify(payload),
  })
}

export async function deleteExecutorResource(id: number): Promise<void> {
  await apiRequest<void>(`/executor-resources/${id}`, { method: 'DELETE' })
}
