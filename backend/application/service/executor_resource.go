package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"flow/application/dto"
	"flow/domain/entity"
	"flow/domain/executor"
	"flow/domain/repository"
)

const secretMask = "******"

// 各运行时类型的敏感字段（列表/查询时掩码，更新时空串或掩码保持原值）。
var secretFieldsByType = map[string][]string{
	executor.RuntimeKindSSH: {"password", "private_key"},
	executor.RuntimeKindK8s: {"kubeconfig"},
}

// ExecutorResourceService 管理节点执行资源（在哪里执行）。
type ExecutorResourceService struct {
	repo    repository.ExecutorResourceRepository
	k8sRepo repository.K8sConfigRepository
}

// NewExecutorResourceService 构造执行资源服务。
func NewExecutorResourceService(repo repository.ExecutorResourceRepository, k8sRepo repository.K8sConfigRepository) *ExecutorResourceService {
	return &ExecutorResourceService{repo: repo, k8sRepo: k8sRepo}
}

// List 列出全部执行资源；敏感字段以掩码返回。
func (s *ExecutorResourceService) List(ctx context.Context) ([]dto.ExecutorResourceResponse, error) {
	items, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]dto.ExecutorResourceResponse, 0, len(items))
	for _, item := range items {
		out = append(out, toResourceResponse(item, true))
	}
	return out, nil
}

// Get 查询单个资源（含掩码）。
func (s *ExecutorResourceService) Get(ctx context.Context, id int64) (*dto.ExecutorResourceResponse, error) {
	res, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	out := toResourceResponse(res, true)
	return &out, nil
}

// Create 创建执行资源。
func (s *ExecutorResourceService) Create(ctx context.Context, req dto.UpsertExecutorResourceRequest) (*dto.ExecutorResourceResponse, error) {
	res, err := entity.NewExecutorResource(req.Name, req.Type, req.Config)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, res); err != nil {
		return nil, err
	}
	out := toResourceResponse(res, false)
	return &out, nil
}

// Update 更新执行资源；掩码/空字符串的敏感字段保持原值。
func (s *ExecutorResourceService) Update(ctx context.Context, id int64, req dto.UpsertExecutorResourceRequest) (*dto.ExecutorResourceResponse, error) {
	res, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	res.Name = req.Name
	res.Type = req.Type
	res.Config = mergeResourceConfig(res.Type, res.Config, req.Config)
	if err := res.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, res); err != nil {
		return nil, err
	}
	out := toResourceResponse(res, false)
	return &out, nil
}

// Delete 删除执行资源。
func (s *ExecutorResourceService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// GetEntity 供调度器读取原始资源（含完整配置）。
func (s *ExecutorResourceService) GetEntity(ctx context.Context, id int64) (*entity.ExecutorResource, error) {
	return s.repo.GetByID(ctx, id)
}

func toResourceResponse(res *entity.ExecutorResource, maskSecrets bool) dto.ExecutorResourceResponse {
	resp := dto.ExecutorResourceResponse{
		ID:        res.ID,
		Name:      res.Name,
		Type:      res.Type,
		Config:    res.Config,
		CreatedAt: res.CreatedAt,
		UpdatedAt: res.UpdatedAt,
	}
	if maskSecrets {
		resp.Config = maskResourceConfig(res.Type, res.Config)
	}
	return resp
}

func secretFields(typ string) []string {
	return secretFieldsByType[typ]
}

// maskResourceConfig 将敏感字段替换为掩码。
func maskResourceConfig(typ, config string) string {
	fields := secretFields(typ)
	if len(fields) == 0 || strings.TrimSpace(config) == "" {
		return config
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(config), &m); err != nil {
		return config
	}
	for _, f := range fields {
		if v, ok := m[f].(string); ok && v != "" {
			m[f] = secretMask
		}
	}
	data, err := json.Marshal(m)
	if err != nil {
		return config
	}
	return string(data)
}

// mergeResourceConfig 更新时对掩码或空白的敏感字段回填旧值。
func mergeResourceConfig(typ, oldConfig, newConfig string) string {
	fields := secretFields(typ)
	if len(fields) == 0 || strings.TrimSpace(newConfig) == "" {
		if strings.TrimSpace(newConfig) == "" {
			return oldConfig
		}
		return newConfig
	}
	var oldM, newM map[string]any
	if json.Unmarshal([]byte(oldConfig), &oldM) != nil || json.Unmarshal([]byte(newConfig), &newM) != nil {
		return newConfig
	}
	for _, f := range fields {
		v, ok := newM[f].(string)
		if !ok {
			continue
		}
		if v == "" || v == secretMask {
			if oldV, ok := oldM[f].(string); ok {
				newM[f] = oldV
			}
		}
	}
	data, err := json.Marshal(newM)
	if err != nil {
		return newConfig
	}
	return string(data)
}

// ResolveNodeExecutor 根据节点运行时配置构造执行器。
func (s *ExecutorResourceService) ResolveNodeExecutor(ctx context.Context, n *entity.Node) (executor.Executor, error) {
	switch n.RuntimeKind {
	case "", executor.RuntimeKindLocal:
		return executor.NewRuntime(executor.RuntimeKindLocal, nil)
	case executor.RuntimeKindSSH:
		return s.resolveRuntime(ctx, n.RuntimeKind, n.ResourceID)
	case executor.RuntimeKindK8s:
		if n.K8sAuthID > 0 {
			if s.k8sRepo == nil {
				return nil, fmt.Errorf("k8s config repository is not configured")
			}
			auth, err := s.k8sRepo.GetByID(ctx, n.K8sAuthID)
			if err != nil {
				return nil, fmt.Errorf("load k8s auth %d: %w", n.K8sAuthID, err)
			}
			ns := n.K8sNamespace
			if ns == "" {
				ns = auth.Namespace
			}
			cfg := executor.K8sConfig{
				Kubeconfig: auth.Kubeconfig,
				Server:     auth.Server,
				Token:      auth.Token,
				Namespace:  ns,
			}
			return executor.NewK8sExecutor(cfg)
		}
		return s.resolveRuntime(ctx, executor.RuntimeKindK8s, n.ResourceID)
	default:
		return nil, errors.New("unknown runtime kind: " + n.RuntimeKind)
	}
}

// resolveRuntime 根据节点的运行时类型与资源 ID 构造执行器。
func (s *ExecutorResourceService) resolveRuntime(ctx context.Context, kind string, resourceID int64) (executor.Executor, error) {
	switch kind {
	case "", executor.RuntimeKindLocal:
		return executor.NewRuntime(executor.RuntimeKindLocal, nil)
	case executor.RuntimeKindSSH, executor.RuntimeKindK8s:
		if resourceID == 0 {
			return nil, fmt.Errorf("runtime %s requires a resource", kind)
		}
		res, err := s.repo.GetByID(ctx, resourceID)
		if err != nil {
			return nil, fmt.Errorf("load executor resource %d: %w", resourceID, err)
		}
		if res.Type != kind {
			return nil, fmt.Errorf("resource %d type %s does not match node runtime %s", resourceID, res.Type, kind)
		}
		return executor.NewRuntime(kind, []byte(res.Config))
	default:
		return nil, errors.New("unknown runtime kind: " + kind)
	}
}
