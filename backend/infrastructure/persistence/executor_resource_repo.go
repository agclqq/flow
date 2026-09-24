package persistence

import (
	"context"
	"errors"

	"flow/domain/entity"
	"flow/domain/repository"

	"gorm.io/gorm"
)

// ExecutorResourceRepository 是执行资源的持久化实现。
type ExecutorResourceRepository struct {
	db *gorm.DB
}

// NewExecutorResourceRepository 构造执行资源仓储。
func NewExecutorResourceRepository(db *gorm.DB) *ExecutorResourceRepository {
	return &ExecutorResourceRepository{db: db}
}

func toResourceEntity(m *ExecutorResourceModel) *entity.ExecutorResource {
	return &entity.ExecutorResource{
		ID:        m.ID,
		Name:      m.Name,
		Type:      m.Type,
		Config:    m.Config,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func toResourceModel(r *entity.ExecutorResource) *ExecutorResourceModel {
	return &ExecutorResourceModel{
		ID:        r.ID,
		Name:      r.Name,
		Type:      r.Type,
		Config:    r.Config,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func (r *ExecutorResourceRepository) Create(ctx context.Context, res *entity.ExecutorResource) error {
	m := toResourceModel(res)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	res.ID, res.CreatedAt, res.UpdatedAt = m.ID, m.CreatedAt, m.UpdatedAt
	return nil
}

func (r *ExecutorResourceRepository) Update(ctx context.Context, res *entity.ExecutorResource) error {
	return r.db.WithContext(ctx).Save(toResourceModel(res)).Error
}

func (r *ExecutorResourceRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&ExecutorResourceModel{}, id).Error
}

func (r *ExecutorResourceRepository) GetByID(ctx context.Context, id int64) (*entity.ExecutorResource, error) {
	var m ExecutorResourceModel
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return toResourceEntity(&m), nil
}

func (r *ExecutorResourceRepository) List(ctx context.Context) ([]*entity.ExecutorResource, error) {
	var models []ExecutorResourceModel
	if err := r.db.WithContext(ctx).Order("id DESC").Find(&models).Error; err != nil {
		return nil, err
	}
	items := make([]*entity.ExecutorResource, 0, len(models))
	for i := range models {
		items = append(items, toResourceEntity(&models[i]))
	}
	return items, nil
}
