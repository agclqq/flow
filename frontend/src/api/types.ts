export type TemplateStatus = 'draft' | 'published'

export type FlowTemplate = {
  id: number
  name: string
  version: number
  status: TemplateStatus
  owners?: string[]
  description?: string
  // 最新发布快照是否开启手动启动（未发布过为 false）。
  manual_trigger?: boolean
  // 最新发布快照的版本 ID（未发布过为空）。
  published_version_id?: number
  // 最新发布快照是否开启定时触发及其表达式。
  cron_enabled?: boolean
  cron_expr?: string
  created_at: string
  updated_at: string
}

export type FlowParam = {
  id?: number
  template_id?: number
  version_id?: number
  name: string
  type: 'string' | 'number' | 'boolean'
  default_value: string
  required: boolean
  description: string
}

export type TriggerConfig = {
  id?: number
  template_id?: number
  manual: boolean
  webhook: boolean
  cron: boolean
  cron_expr: string
  require_params: boolean
  webhook_secret?: string
}

export type NodeType = 'start' | 'end' | 'task' | 'edge' | 'manual'

export type StartMode = 'any' | 'all' | 'custom'

export type FlowNode = {
  id?: number
  template_id?: number
  version_id?: number
  type: NodeType | 'join'
  name: string
  pre_script: string
  script: string
  post_script: string
  owners: string[]
  approvers?: string[]
  description?: string
  from_node_id?: number
  to_node_id?: number
  start_mode?: StartMode | string
  start_script?: string
  required_predecessor_ids?: number[]
  position_x?: number | null
  position_y?: number | null
  // 执行资源：runtime_kind = local / ssh / k8s。
  runtime_kind?: string
  resource_id?: number
  k8s_auth_id?: number
  k8s_namespace?: string
  k8s_image?: string
  k8s_volumes?: K8sVolume[]
  k8s_volume_mounts?: K8sVolumeMount[]
  k8s_resources?: K8sResources
}

export type K8sResources = {
  requests?: Record<string, string>
  limits?: Record<string, string>
}

export type K8sVolume = {
  name: string
  type: 'emptyDir' | 'hostPath' | 'configMap' | 'secret' | 'pvc' | string
  source?: string
}

export type K8sVolumeMount = {
  name: string
  mount_path: string
  sub_path?: string
  read_only?: boolean
}

export type K8sConfig = {
  id: number
  name: string
  server: string
  token: string
  kubeconfig: string
  namespace: string
  created_at: string
  updated_at: string
}

export type UpsertK8sConfigRequest = {
  name: string
  server?: string
  token?: string
  kubeconfig?: string
  namespace?: string
}

export type VersionListItem = {
  id: number
  version: number
  status: TemplateStatus
  created_at: string
}

export type VersionDetail = VersionListItem & {
  template_id: number
  nodes: FlowNode[]
  updated_at: string
}

export type NotifyTemplate = {
  id: number
  name: string
  content: string
}

export type NotifyBinding = {
  id?: number
  template_id?: number
  node_id?: number
  notify_template_id: number
  timings: ('success' | 'failure')[]
}

export type FlowInstanceStatus = 'pending' | 'running' | 'succeeded' | 'failed' | 'retry'

export type NodeInstanceStatus =
  | 'pending'
  | 'running'
  | 'succeeded'
  | 'failed'
  | 'failed_skipped'
  | 'retrying'
  | 'waiting_approval'
  | 'skipped'

export type NodeRunTimeSpan = {
  start_at: string
  end_at?: string
}

export type NodeInstance = {
  id: number
  flow_instance_id: number
  node_id: number
  type: NodeType
  status: NodeInstanceStatus
  env: Record<string, string>
  output_env: Record<string, string>
  run_times?: NodeRunTimeSpan[]
  retry_count: number
}

export type FlowInstance = {
  id: number
  template_id: number
  template_version_id: number
  template_version: number
  status: FlowInstanceStatus
  trigger_type: string
  params: Record<string, string>
  env: Record<string, string>
  node_instances?: NodeInstance[]
  created_at: string
  updated_at: string
  finished_at?: string | null
  duration_seconds?: number | null
}

export type Paginated<T> = {
  items: T[]
  total: number
}

export type ConfigFormValues = {
  name: string
  owners: string[]
  params: FlowParam[]
  manual: boolean
  webhook: boolean
  cron: boolean
  cron_expr: string
  require_params: boolean
  webhook_secret?: string
  notify_timings: ('success' | 'failure')[]
  notify_template_id?: number
}

export type GraphNodePayload = {
  clientId: string
  backendId?: number
  type: NodeType
  name: string
  pre_script: string
  script: string
  post_script: string
  owners: string[]
  approvers?: string[]
  description?: string
  position: { x: number; y: number }
  start_mode?: StartMode | string
  start_script?: string
  required_predecessor_ids?: number[]
  runtime_kind?: string
  resource_id?: number
  k8s_auth_id?: number
  k8s_namespace?: string
  k8s_image?: string
  k8s_volumes?: K8sVolume[]
  k8s_volume_mounts?: K8sVolumeMount[]
  k8s_resources?: K8sResources
}

export type GraphEdgePayload = {
  clientId: string
  backendId?: number
  source: string
  target: string
}

export type GraphPayload = {
  nodes: GraphNodePayload[]
  edges: GraphEdgePayload[]
}

// --- 执行资源（节点在哪里执行）---

export type RuntimeKind = 'local' | 'ssh' | 'k8s'

export type ExecutorResource = {
  id: number
  name: string
  type: RuntimeKind | string
  // JSON 字符串：ssh 为 {host,port,username,auth_type,password,private_key,work_dir}；
  // k8s 为 {kubeconfig,namespace,image}。列表返回时敏感字段为掩码。
  config: string
  created_at: string
  updated_at: string
}

export type UpsertExecutorResourceRequest = {
  name: string
  type: string
  config: string
}
