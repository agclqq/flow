package executor

import (
	"encoding/json"
	"fmt"
	"strings"
)

// 运行时类型常量，与节点上的 runtime_kind 对应。
const (
	RuntimeKindLocal = "local"
	RuntimeKindSSH   = "ssh"
	RuntimeKindK8s   = "k8s"
)

// SSHConfig 描述一个远程 SSH 执行资源。
type SSHConfig struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	AuthType   string `json:"auth_type"` // password | key
	Password   string `json:"password"`
	PrivateKey string `json:"private_key"`
	// WorkDir 为远端工作目录，脚本在其中执行。
	WorkDir string `json:"work_dir"`
}

// K8sConfig 描述一个 k8s 集群执行资源。
type K8sConfig struct {
	// Kubeconfig 为 kubeconfig 文件内容（YAML），从中解析 server / 凭证 / CA。
	Kubeconfig string `json:"kubeconfig"`
	// Namespace 为 Pod 运行命名空间。
	Namespace string `json:"namespace"`
	// Image 为执行镜像，脚本在其中运行；为空时使用默认镜像。
	Image string `json:"image"`
}

// ValidRuntimeKind 判断运行时类型是否合法。
func ValidRuntimeKind(kind string) bool {
	switch kind {
	case RuntimeKindLocal, RuntimeKindSSH, RuntimeKindK8s:
		return true
	default:
		return false
	}
}

// ParseSSHConfig 从 JSON 解析 SSH 资源配置。
func ParseSSHConfig(raw string) (SSHConfig, error) {
	var cfg SSHConfig
	if strings.TrimSpace(raw) == "" {
		return cfg, nil
	}
	err := json.Unmarshal([]byte(raw), &cfg)
	return cfg, err
}

// ParseK8sConfig 从 JSON 解析 k8s 资源配置。
func ParseK8sConfig(raw string) (K8sConfig, error) {
	var cfg K8sConfig
	if strings.TrimSpace(raw) == "" {
		return cfg, nil
	}
	err := json.Unmarshal([]byte(raw), &cfg)
	return cfg, err
}

// NewRuntime 根据类型与资源配置 JSON 构造执行器。
// kind 为空或 local 时返回本地执行器（仅调试用途）。
func NewRuntime(kind string, rawConfig []byte) (Executor, error) {
	switch kind {
	case "", RuntimeKindLocal:
		return NewLocalExecutor(), nil
	case RuntimeKindSSH:
		var cfg SSHConfig
		if err := json.Unmarshal(rawConfig, &cfg); err != nil {
			return nil, fmt.Errorf("parse ssh resource config: %w", err)
		}
		return NewSSHExecutor(cfg)
	case RuntimeKindK8s:
		var cfg K8sConfig
		if err := json.Unmarshal(rawConfig, &cfg); err != nil {
			return nil, fmt.Errorf("parse k8s resource config: %w", err)
		}
		return NewK8sExecutor(cfg)
	default:
		return nil, fmt.Errorf("unknown runtime kind: %s", kind)
	}
}
