package entity

import (
	"errors"
	"time"

	"flow/domain/executor"
)

// ExecutorResource 是一个可复用的节点执行资源（在哪里执行）。
// 全局共享，节点通过 ResourceID 引用。
type ExecutorResource struct {
	ID        int64
	Name      string
	// Type 为运行时类型：local / ssh / k8s。
	Type      string
	// Config 为该类型的连接配置（JSON，结构由 executor 包定义）。
	Config    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Validate 校验资源类型与配置。
func (r *ExecutorResource) Validate() error {
	if r.Name == "" {
		return errors.New("resource name must not be empty")
	}
	if !executor.ValidRuntimeKind(r.Type) {
		return errors.New("invalid resource type")
	}
	if r.Type == executor.RuntimeKindSSH {
		cfg, err := executor.ParseSSHConfig(r.Config)
		if err != nil {
			return err
		}
		if cfg.Host == "" || cfg.Username == "" {
			return errors.New("ssh resource requires host and username")
		}
	}
	if r.Type == executor.RuntimeKindK8s {
		cfg, err := executor.ParseK8sConfig(r.Config)
		if err != nil {
			return err
		}
		if cfg.Kubeconfig == "" {
			return errors.New("k8s resource requires kubeconfig")
		}
	}
	return nil
}

// NewExecutorResource 创建执行资源。
func NewExecutorResource(name, typ, config string) (*ExecutorResource, error) {
	r := &ExecutorResource{Name: name, Type: typ, Config: config, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return r, nil
}
