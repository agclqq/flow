package persistence

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"flow/domain/entity"
	"flow/domain/repository"
)

// K8sConfigRepository 是 K8sConfigRepository 的 gorm 实现。
type K8sConfigRepository struct {
	db *gorm.DB
}

func NewK8sConfigRepository(db *gorm.DB) repository.K8sConfigRepository {
	return &K8sConfigRepository{db: db}
}

func (r *K8sConfigRepository) Create(ctx context.Context, cfg *entity.K8sConfig) error {
	m := toK8sConfigModel(cfg)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	cfg.ID = m.ID
	return nil
}

func (r *K8sConfigRepository) GetByID(ctx context.Context, id int64) (*entity.K8sConfig, error) {
	var m K8sConfigModel
	if err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return toK8sConfigEntity(&m), nil
}

func (r *K8sConfigRepository) List(ctx context.Context) ([]*entity.K8sConfig, error) {
	var models []K8sConfigModel
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]*entity.K8sConfig, 0, len(models))
	for i := range models {
		out = append(out, toK8sConfigEntity(&models[i]))
	}
	return out, nil
}

func (r *K8sConfigRepository) Update(ctx context.Context, cfg *entity.K8sConfig) error {
	return r.db.WithContext(ctx).Save(toK8sConfigModel(cfg)).Error
}

func (r *K8sConfigRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&K8sConfigModel{}, "id = ?", id).Error
}

func toK8sConfigModel(c *entity.K8sConfig) *K8sConfigModel {
	return &K8sConfigModel{
		ID:         c.ID,
		Name:       c.Name,
		Server:     c.Server,
		Token:      c.Token,
		Kubeconfig: c.Kubeconfig,
		Namespace:  c.Namespace,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}
}

func toK8sConfigEntity(m *K8sConfigModel) *entity.K8sConfig {
	return &entity.K8sConfig{
		ID:         m.ID,
		Name:       m.Name,
		Server:     m.Server,
		Token:      m.Token,
		Kubeconfig: m.Kubeconfig,
		Namespace:  m.Namespace,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}
