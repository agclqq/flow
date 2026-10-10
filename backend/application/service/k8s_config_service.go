package service

import (
	"context"
	"encoding/json"

	"flow/application/dto"
	"flow/domain/entity"
	"flow/domain/repository"
)

const k8sSecretMask = "******"

var k8sSecretFields = []string{"kubeconfig", "token"}

// K8sConfigService 管理 K8s 集群认证配置。
type K8sConfigService struct {
	repo repository.K8sConfigRepository
}

func NewK8sConfigService(repo repository.K8sConfigRepository) *K8sConfigService {
	return &K8sConfigService{repo: repo}
}

func (s *K8sConfigService) List(ctx context.Context) ([]dto.K8sConfigResponse, error) {
	items, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]dto.K8sConfigResponse, 0, len(items))
	for _, item := range items {
		out = append(out, toK8sConfigResponse(item, true))
	}
	return out, nil
}

func (s *K8sConfigService) Get(ctx context.Context, id int64) (*dto.K8sConfigResponse, error) {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := toK8sConfigResponse(item, true)
	return &resp, nil
}

func (s *K8sConfigService) GetEntity(ctx context.Context, id int64) (*entity.K8sConfig, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *K8sConfigService) Create(ctx context.Context, req dto.UpsertK8sConfigRequest) (*dto.K8sConfigResponse, error) {
	cfg, err := entity.NewK8sConfig(req.Name, req.Server, req.Kubeconfig, req.Token, req.Namespace)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, cfg); err != nil {
		return nil, err
	}
	resp := toK8sConfigResponse(cfg, false)
	return &resp, nil
}

func (s *K8sConfigService) Update(ctx context.Context, id int64, req dto.UpsertK8sConfigRequest) (*dto.K8sConfigResponse, error) {
	cfg, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	cfg.Name = req.Name
	cfg.Server = req.Server
	cfg.Namespace = req.Namespace
	cfg.Kubeconfig = mergeK8sSecret(cfg.Kubeconfig, req.Kubeconfig)
	cfg.Token = mergeK8sSecret(cfg.Token, req.Token)
	if err := s.repo.Update(ctx, cfg); err != nil {
		return nil, err
	}
	resp := toK8sConfigResponse(cfg, false)
	return &resp, nil
}

func (s *K8sConfigService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func toK8sConfigResponse(cfg *entity.K8sConfig, mask bool) dto.K8sConfigResponse {
	resp := dto.K8sConfigResponse{
		ID:        cfg.ID,
		Name:      cfg.Name,
		Server:    cfg.Server,
		Namespace: cfg.Namespace,
		Kubeconfig: cfg.Kubeconfig,
		Token:     cfg.Token,
		CreatedAt: cfg.CreatedAt,
		UpdatedAt: cfg.UpdatedAt,
	}
	if mask {
		if resp.Kubeconfig != "" {
			resp.Kubeconfig = k8sSecretMask
		}
		if resp.Token != "" {
			resp.Token = k8sSecretMask
		}
	}
	return resp
}

func mergeK8sSecret(oldVal, newVal string) string {
	if newVal == "" || newVal == k8sSecretMask {
		return oldVal
	}
	return newVal
}

func K8sAuthToExecutorConfig(auth *entity.K8sConfig, namespaceOverride string) (map[string]string, error) {
	ns := namespaceOverride
	if ns == "" {
		ns = auth.Namespace
	}
	if ns == "" {
		ns = "default"
	}
	data := map[string]string{
		"kubeconfig": auth.Kubeconfig,
		"server":     auth.Server,
		"token":      auth.Token,
		"namespace":  ns,
	}
	_, err := json.Marshal(data)
	return data, err
}
