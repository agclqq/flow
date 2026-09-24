package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"flow/application/dto"
	"flow/domain/entity"
	"flow/domain/repository"
	"flow/domain/valueobject"
)

// mockFlowTemplateRepo 是仓储接口的 mock 实现。
type mockFlowTemplateRepo struct {
	mock.Mock
}

func (m *mockFlowTemplateRepo) Create(ctx context.Context, tpl *entity.FlowTemplate) error {
	args := m.Called(ctx, tpl)
	return args.Error(0)
}

func (m *mockFlowTemplateRepo) GetByID(ctx context.Context, id int64) (*entity.FlowTemplate, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.FlowTemplate), args.Error(1)
}

func (m *mockFlowTemplateRepo) Update(ctx context.Context, tpl *entity.FlowTemplate) error {
	args := m.Called(ctx, tpl)
	return args.Error(0)
}

func (m *mockFlowTemplateRepo) Delete(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockFlowTemplateRepo) List(ctx context.Context, offset, limit int) ([]*entity.FlowTemplate, int64, error) {
	args := m.Called(ctx, offset, limit)
	return args.Get(0).([]*entity.FlowTemplate), args.Get(1).(int64), args.Error(2)
}

// mockVersionRepo 是版本仓储接口的 mock 实现。
type mockVersionRepo struct {
	mock.Mock
}

func (m *mockVersionRepo) Create(ctx context.Context, ver *entity.FlowTemplateVersion) error {
	args := m.Called(ctx, ver)
	return args.Error(0)
}

func (m *mockVersionRepo) GetByID(ctx context.Context, id int64) (*entity.FlowTemplateVersion, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.FlowTemplateVersion), args.Error(1)
}

func (m *mockVersionRepo) GetByTemplateAndVersion(ctx context.Context, templateID int64, version int) (*entity.FlowTemplateVersion, error) {
	args := m.Called(ctx, templateID, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.FlowTemplateVersion), args.Error(1)
}

func (m *mockVersionRepo) GetLatestPublishedByTemplate(ctx context.Context, templateID int64) (*entity.FlowTemplateVersion, error) {
	args := m.Called(ctx, templateID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.FlowTemplateVersion), args.Error(1)
}

func (m *mockVersionRepo) ListByTemplate(ctx context.Context, templateID int64) ([]*entity.FlowTemplateVersion, error) {
	args := m.Called(ctx, templateID)
	return args.Get(0).([]*entity.FlowTemplateVersion), args.Error(1)
}

func (m *mockVersionRepo) Update(ctx context.Context, ver *entity.FlowTemplateVersion) error {
	args := m.Called(ctx, ver)
	return args.Error(0)
}

type stubNodeRepo struct{}

func (stubNodeRepo) Create(context.Context, *entity.Node) error                         { return nil }
func (stubNodeRepo) GetByID(context.Context, int64) (*entity.Node, error)             { return nil, repository.ErrNotFound }
func (stubNodeRepo) Update(context.Context, *entity.Node) error                         { return nil }
func (stubNodeRepo) Delete(context.Context, int64) error                                { return nil }
func (stubNodeRepo) ListByVersion(context.Context, int64, int64) ([]*entity.Node, error) { return nil, nil }
func (stubNodeRepo) DeleteByVersion(context.Context, int64, int64) error                { return nil }

type stubParamRepo struct{}

func (stubParamRepo) Create(context.Context, *entity.FlowParam) error { return nil }
func (stubParamRepo) GetByID(context.Context, int64) (*entity.FlowParam, error) {
	return nil, repository.ErrNotFound
}
func (stubParamRepo) Update(context.Context, *entity.FlowParam) error { return nil }
func (stubParamRepo) Delete(context.Context, int64) error             { return nil }
func (stubParamRepo) ListByVersion(context.Context, int64, int64) ([]*entity.FlowParam, error) {
	return nil, nil
}

type stubTriggerRepo struct{}

func (stubTriggerRepo) Create(context.Context, *entity.TriggerConfig) error { return nil }
func (stubTriggerRepo) GetByTemplateID(context.Context, int64) (*entity.TriggerConfig, error) {
	return nil, repository.ErrNotFound
}
func (stubTriggerRepo) Update(context.Context, *entity.TriggerConfig) error { return nil }
func (stubTriggerRepo) Delete(context.Context, int64) error                   { return nil }

type stubBindingRepo struct{}

func (stubBindingRepo) Create(context.Context, *entity.FlowNotifyBinding) error { return nil }
func (stubBindingRepo) GetByID(context.Context, int64) (*entity.FlowNotifyBinding, error) {
	return nil, repository.ErrNotFound
}
func (stubBindingRepo) Update(context.Context, *entity.FlowNotifyBinding) error { return nil }
func (stubBindingRepo) Delete(context.Context, int64) error                      { return nil }
func (stubBindingRepo) ListByTemplate(context.Context, int64) ([]*entity.FlowNotifyBinding, error) {
	return nil, nil
}
func (stubBindingRepo) ReplaceByTemplate(context.Context, int64, []*entity.FlowNotifyBinding) error {
	return nil
}

func newTestFlowTemplateService(repo repository.FlowTemplateRepository, vRepo repository.FlowTemplateVersionRepository) *FlowTemplateService {
	return NewFlowTemplateService(repo, vRepo, stubNodeRepo{}, stubParamRepo{}, stubTriggerRepo{}, stubBindingRepo{})
}

func TestFlowTemplateServiceCreate(t *testing.T) {
	repo := new(mockFlowTemplateRepo)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*entity.FlowTemplate")).Return(nil).Run(func(args mock.Arguments) {
		tpl := args.Get(1).(*entity.FlowTemplate)
		tpl.ID = 1
	})

	vRepo := new(mockVersionRepo)
	vRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.FlowTemplateVersion")).Return(nil)

	svc := newTestFlowTemplateService(repo, vRepo)
	resp, err := svc.Create(context.Background(), dto.CreateFlowTemplateRequest{Name: "build"})

	assert.NoError(t, err)
	assert.Equal(t, "build", resp.Name)
	assert.Equal(t, "draft", resp.Status)
	repo.AssertExpectations(t)
	vRepo.AssertExpectations(t)
}

func TestFlowTemplateServiceGet(t *testing.T) {
	tpl, _ := entity.NewFlowTemplate("build")
	tpl.ID = 1

	repo := new(mockFlowTemplateRepo)
	repo.On("GetByID", mock.Anything, int64(1)).Return(tpl, nil)

	vRepo := new(mockVersionRepo)
	// 未发布过：查不到发布快照，manual_trigger 为 false。
	vRepo.On("GetLatestPublishedByTemplate", mock.Anything, int64(1)).Return((*entity.FlowTemplateVersion)(nil), repository.ErrNotFound)

	svc := newTestFlowTemplateService(repo, vRepo)
	resp, err := svc.Get(context.Background(), 1)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), resp.ID)
	assert.False(t, resp.ManualTrigger)
	repo.AssertExpectations(t)
}

func TestFlowTemplateServiceGetNotFound(t *testing.T) {
	repo := new(mockFlowTemplateRepo)
	repo.On("GetByID", mock.Anything, int64(999)).Return(nil, repository.ErrNotFound)

	vRepo := new(mockVersionRepo)

	svc := newTestFlowTemplateService(repo, vRepo)
	_, err := svc.Get(context.Background(), 999)

	assert.ErrorIs(t, err, repository.ErrNotFound)
	repo.AssertExpectations(t)
}

func TestFlowTemplateServicePublish(t *testing.T) {
	tpl, _ := entity.NewFlowTemplate("build")
	tpl.ID = 1
	draftVer, _ := entity.NewFlowTemplateVersion(1, 1)

	repo := new(mockFlowTemplateRepo)
	repo.On("GetByID", mock.Anything, int64(1)).Return(tpl, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*entity.FlowTemplate")).Return(nil)

	vRepo := new(mockVersionRepo)
	vRepo.On("ListByTemplate", mock.Anything, int64(1)).Return([]*entity.FlowTemplateVersion{draftVer}, nil)
	vRepo.On("GetByID", mock.Anything, draftVer.ID).Return(draftVer, nil)
	vRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.FlowTemplateVersion")).Return(nil)
	vRepo.On("Update", mock.Anything, mock.AnythingOfType("*entity.FlowTemplateVersion")).Return(nil)

	svc := newTestFlowTemplateService(repo, vRepo)
	resp, err := svc.Publish(context.Background(), 1)

	assert.NoError(t, err)
	assert.Equal(t, "published", resp.Status)
	// 首个快照版本号为 1，current_version 对齐快照号。
	assert.Equal(t, 1, resp.Version)
	// git 模型：发布打 tag，草稿保留继续编辑且版本号顺延。
	assert.Equal(t, valueobject.TemplateStatusDraft, draftVer.Status)
	assert.Equal(t, 2, draftVer.Version)
	repo.AssertExpectations(t)
	vRepo.AssertExpectations(t)
}

func TestFlowTemplateServicePublishWithoutDraft(t *testing.T) {
	tpl, _ := entity.NewFlowTemplate("build")
	tpl.ID = 1

	repo := new(mockFlowTemplateRepo)
	repo.On("GetByID", mock.Anything, int64(1)).Return(tpl, nil)

	vRepo := new(mockVersionRepo)
	vRepo.On("ListByTemplate", mock.Anything, int64(1)).Return([]*entity.FlowTemplateVersion{}, nil)

	svc := newTestFlowTemplateService(repo, vRepo)
	_, err := svc.Publish(context.Background(), 1)

	assert.Error(t, err)
	repo.AssertExpectations(t)
	vRepo.AssertExpectations(t)
}

func TestFlowTemplateServiceDelete(t *testing.T) {
	tpl, _ := entity.NewFlowTemplate("build")
	tpl.ID = 1

	repo := new(mockFlowTemplateRepo)
	repo.On("GetByID", mock.Anything, int64(1)).Return(tpl, nil)
	repo.On("Delete", mock.Anything, int64(1)).Return(nil)

	vRepo := new(mockVersionRepo)

	svc := newTestFlowTemplateService(repo, vRepo)
	err := svc.Delete(context.Background(), 1)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

// 保证 status 常量导出可用。
var _ = valueobject.TemplateStatusDraft
