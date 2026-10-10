package entity

// K8sVolume 描述 Pod 卷定义。
type K8sVolume struct {
	Name   string `json:"name"`
	Type   string `json:"type"` // emptyDir | hostPath | configMap | secret | pvc
	Source string `json:"source,omitempty"`
}

// K8sVolumeMount 描述容器挂载点。
type K8sVolumeMount struct {
	Name      string `json:"name"`
	MountPath string `json:"mount_path"`
	SubPath   string `json:"sub_path,omitempty"`
	ReadOnly  bool   `json:"read_only,omitempty"`
}

// K8sResourceRequirements 容器 resources.requests / limits（键为 K8s 资源名，如 cpu、memory、nvidia.com/gpu）。
type K8sResourceRequirements struct {
	Requests map[string]string `json:"requests,omitempty"`
	Limits   map[string]string `json:"limits,omitempty"`
}

// IsEmpty 是否未配置任何 request/limit。
func (r K8sResourceRequirements) IsEmpty() bool {
	return len(r.Requests) == 0 && len(r.Limits) == 0
}
