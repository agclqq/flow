import { apiRequest, buildQuery } from './client'
import type { FlowInstance, Paginated } from './types'

export async function listInstances(
  templateId: number,
  offset = 0,
  limit = 20,
): Promise<Paginated<FlowInstance>> {
  return apiRequest<Paginated<FlowInstance>>(
    `/flow-templates/${templateId}/instances${buildQuery({ offset, limit })}`,
  )
}

export async function getInstance(templateId: number, instanceId: number): Promise<FlowInstance> {
  return apiRequest<FlowInstance>(`/flow-templates/${templateId}/instances/${instanceId}`)
}

export async function startInstance(
  templateId: number,
  params: Record<string, string> = {},
): Promise<FlowInstance> {
  return apiRequest<FlowInstance>(`/flow-templates/${templateId}/instances`, {
    method: 'POST',
    body: JSON.stringify({ trigger_type: 'manual', params }),
  })
}

export async function stopInstance(templateId: number, instanceId: number): Promise<void> {
  await apiRequest<void>(`/flow-templates/${templateId}/instances/${instanceId}/stop`, {
    method: 'POST',
    body: JSON.stringify({}),
  })
}

export async function retryInstance(templateId: number, instanceId: number): Promise<FlowInstance> {
  return apiRequest<FlowInstance>(`/flow-templates/${templateId}/instances/${instanceId}/retry`, {
    method: 'POST',
    body: JSON.stringify({}),
  })
}

export async function approveNodeInstance(
  templateId: number,
  instanceId: number,
  nodeId: number,
  operator?: string,
): Promise<void> {
  await apiRequest<void>(
    `/flow-templates/${templateId}/instances/${instanceId}/nodes/${nodeId}/approve`,
    { method: 'POST', body: JSON.stringify({ operator: operator ?? '' }) },
  )
}

export async function rejectNodeInstance(
  templateId: number,
  instanceId: number,
  nodeId: number,
  operator?: string,
): Promise<void> {
  await apiRequest<void>(
    `/flow-templates/${templateId}/instances/${instanceId}/nodes/${nodeId}/reject`,
    { method: 'POST', body: JSON.stringify({ operator: operator ?? '' }) },
  )
}

export async function skipNodeInstance(
  templateId: number,
  instanceId: number,
  nodeId: number,
): Promise<FlowInstance> {
  return apiRequest<FlowInstance>(
    `/flow-templates/${templateId}/instances/${instanceId}/nodes/${nodeId}/skip`,
    { method: 'POST', body: JSON.stringify({}) },
  )
}

export type NodeOperationLog = {
  id: number
  operator: string
  action: string
  created_at: string
}

export async function listNodeOperationLogs(
  templateId: number,
  instanceId: number,
  nodeId: number,
): Promise<NodeOperationLog[]> {
  const res = await apiRequest<{ items: NodeOperationLog[] }>(
    `/flow-templates/${templateId}/instances/${instanceId}/nodes/${nodeId}/logs`,
  )
  return res.items ?? []
}
