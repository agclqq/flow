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
	Kubeconfig string `json:"kubeconfig"`
	Server     string `json:"server"`
	Token      string `json:"token"`
	Namespace  string `json:"namespace"`
	Image      string `json:"image"`
}

// PodVolume 与 PodVolumeMount 用于构造 Pod spec。
type PodVolume struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Source string `json:"source,omitempty"`
}

type PodVolumeMount struct {
	Name      string `json:"name"`
	MountPath string `json:"mount_path"`
	SubPath   string `json:"sub_path,omitempty"`
	ReadOnly  bool   `json:"read_only,omitempty"`
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
