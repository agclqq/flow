package service

import (
	"context"
	"fmt"

	"flow/application/dto"
	"flow/domain/entity"
	"flow/domain/repository"
)

// NotifyService 管理通知内容模板。
type NotifyService struct {
	repo repository.NotifyTemplateRepository
}

// NewNotifyService 构造通知服务。
func NewNotifyService(repo repository.NotifyTemplateRepository) *NotifyService {
	return &NotifyService{repo: repo}
}

// Create 创建通知模板。
func (s *NotifyService) Create(ctx context.Context, req dto.CreateNotifyTemplateRequest) (*dto.NotifyTemplateResponse, error) {
	tpl, err := entity.NewNotifyTemplate(req.Name, req.Content)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, tpl); err != nil {
		return nil, err
	}
	resp := dto.ToNotifyTemplateResponse(tpl)
	return &resp, nil
}

// Get 获取通知模板。
func (s *NotifyService) Get(ctx context.Context, id int64) (*dto.NotifyTemplateResponse, error) {
	tpl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := dto.ToNotifyTemplateResponse(tpl)
	return &resp, nil
}

// List 分页列出通知模板。
func (s *NotifyService) List(ctx context.Context, offset, limit int) (*dto.ListNotifyTemplateResponse, error) {
	items, total, err := s.repo.List(ctx, offset, limit)
	if err != nil {
		return nil, err
	}
	respItems := make([]dto.NotifyTemplateResponse, 0, len(items))
	for _, item := range items {
		respItems = append(respItems, dto.ToNotifyTemplateResponse(item))
	}
	return &dto.ListNotifyTemplateResponse{Items: respItems, Total: total}, nil
}

// Update 更新通知模板。
func (s *NotifyService) Update(ctx context.Context, id int64, req dto.UpdateNotifyTemplateRequest) (*dto.NotifyTemplateResponse, error) {
	tpl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != "" {
		tpl.Name = req.Name
	}
	if req.Content != "" {
		tpl.Content = req.Content
	}
	if err := s.repo.Update(ctx, tpl); err != nil {
		return nil, err
	}
	resp := dto.ToNotifyTemplateResponse(tpl)
	return &resp, nil
}

// Delete 删除通知模板。
func (s *NotifyService) Delete(ctx context.Context, id int64) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

// ValidateExists 校验通知模板存在（供绑定使用）。
func (s *NotifyService) ValidateExists(ctx context.Context, id int64) error {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("notify template not found: %w", err)
	}
	return nil
}
