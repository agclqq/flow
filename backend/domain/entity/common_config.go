package entity

import (
	"errors"
	"time"

	"flow/domain/valueobject"
)

// K8sConfig 是 k8s 集群及认证相关配置。
type K8sConfig struct {
	ID     int64
	Name   string
	Server string // k8s API Server 地址
	// Token 认证令牌。
	Token string
	// Kubeconfig 内容（可选，与 token 二选一）。
	Kubeconfig string
	Namespace  string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// NewK8sConfig 创建 k8s 配置。
// ID 由数据库自增生成。
func NewK8sConfig(name, server string) (*K8sConfig, error) {
	if name == "" {
		return nil, errors.New("k8s config name must not be empty")
	}
	if server == "" {
		return nil, errors.New("k8s server must not be empty")
	}
	now := time.Now()
	return &K8sConfig{
		Name:      name,
		Server:    server,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// BareMetalConfig 是裸机执行环境配置（URL/ip 及认证）。
type BareMetalConfig struct {
	ID   int64
	Name string
	Host string // URL 或 IP
	Port int
	// AuthType 认证方式：password（用户+密码）或 sshkey（SSH 私钥），二者互斥使用。
	AuthType  valueobject.AuthType
	User      string
	Password  string // AuthType=password 时使用
	SSHKey    string // AuthType=sshkey 时使用
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewBareMetalConfig 创建裸机配置，默认认证方式为 password。
// ID 由数据库自增生成。
func NewBareMetalConfig(name, host string, port int) (*BareMetalConfig, error) {
	if name == "" {
		return nil, errors.New("bare metal config name must not be empty")
	}
	if host == "" {
		return nil, errors.New("bare metal host must not be empty")
	}
	if port <= 0 {
		return nil, errors.New("bare metal port must be positive")
	}
	now := time.Now()
	return &BareMetalConfig{
		Name:      name,
		Host:      host,
		Port:      port,
		AuthType:  valueobject.AuthTypePassword,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// SetAuthType 设置认证方式并校验合法性。
func (c *BareMetalConfig) SetAuthType(authType valueobject.AuthType) error {
	if !authType.Valid() {
		return errors.New("invalid auth type")
	}
	c.AuthType = authType
	return nil
}

// RepoConfig 是仓库配置（远程地址及认证）。
type RepoConfig struct {
	ID        int64
	Name      string
	URL       string // 远程仓库地址
	User      string
	Token     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewRepoConfig 创建仓库配置。
// ID 由数据库自增生成。
func NewRepoConfig(name, url string) (*RepoConfig, error) {
	if name == "" {
		return nil, errors.New("repo config name must not be empty")
	}
	if url == "" {
		return nil, errors.New("repo url must not be empty")
	}
	now := time.Now()
	return &RepoConfig{
		Name:      name,
		URL:       url,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}
