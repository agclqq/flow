package executor

import "context"

// ExecuteSpec 描述一次执行请求所需的全部信息，
// 组装方式对齐 k8s deployment 的内容（命令、参数、环境变量等）。
type ExecuteSpec struct {
	// InstanceID 为所属流水线实例 ID（用于资源命名隔离等）。
	InstanceID int64
	// Command 为待执行的 shell 脚本内容。
	Command string
	// Args 为执行参数。
	Args []string
	// Env 为注入执行环境的环境变量（key-value）。
	Env map[string]string
	// Image 为执行镜像（k8s 模式下使用）。
	Image string
	// WorkingDir 为工作目录。
	WorkingDir string
	// Volumes / VolumeMounts 为 k8s Pod 卷与挂载（可选）。
	Volumes      []PodVolume
	VolumeMounts []PodVolumeMount
	// Resources 为容器 resources.requests / limits（可选）。
	Resources ContainerResources
}

// ContainerResources 对应 K8s 容器 resourceRequirements。
type ContainerResources struct {
	Requests map[string]string
	Limits   map[string]string
}

// ExecuteResult 描述一次执行的返回结果。
type ExecuteResult struct {
	// ExitCode 为进程退出码，0 表示成功。
	ExitCode int
	// Output 为执行输出（stdout/stderr 合并）。
	Output string
	// ExportedEnv 为脚本写入 FLOW_ENV_OUT 文件后捕获的变量。
	ExportedEnv map[string]string
}

// Executor 是执行器的统一抽象，为后续本地裸机与 k8s 两种执行方式预留扩展点。
// 本阶段仅定义接口占位，不做具体实现。
type Executor interface {
	// Type 返回执行器类型，如 "local" 或 "k8s"。
	Type() string
	// Execute 执行一次任务。
	Execute(ctx context.Context, spec ExecuteSpec) (ExecuteResult, error)
}
