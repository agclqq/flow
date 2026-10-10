package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"gorm.io/gorm"

	"flow/domain/entity"
	"flow/domain/repository"
	"flow/domain/valueobject"
)

// FlowTemplateVersionRepository 是 FlowTemplateVersionRepository 接口的 gorm 实现。
type FlowTemplateVersionRepository struct {
	db *gorm.DB
}

// NewFlowTemplateVersionRepository 构造仓储实现。
func NewFlowTemplateVersionRepository(db *gorm.DB) repository.FlowTemplateVersionRepository {
	return &FlowTemplateVersionRepository{db: db}
}

// Create 持久化版本快照。
func (r *FlowTemplateVersionRepository) Create(ctx context.Context, ver *entity.FlowTemplateVersion) error {
	model := toVersionModel(ver)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return err
	}
	ver.ID = model.ID
	return nil
}

// GetByID 按主键查询版本，同时加载其节点列表。
func (r *FlowTemplateVersionRepository) GetByID(ctx context.Context, id int64) (*entity.FlowTemplateVersion, error) {
	var model FlowTemplateVersionModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	ver := toVersionEntity(&model)
	// 加载节点。
	nodes, err := r.loadNodes(ctx, ver.TemplateID, ver.ID)
	if err != nil {
		return nil, err
	}
	ver.Nodes = nodes
	return ver, nil
}

// GetByTemplateAndVersion 按模板 ID 和版本号查询。
func (r *FlowTemplateVersionRepository) GetByTemplateAndVersion(ctx context.Context, templateID int64, version int) (*entity.FlowTemplateVersion, error) {
	var model FlowTemplateVersionModel
	if err := r.db.WithContext(ctx).
		Where("template_id = ? AND version = ?", templateID, version).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	ver := toVersionEntity(&model)
	nodes, err := r.loadNodes(ctx, ver.TemplateID, ver.ID)
	if err != nil {
		return nil, err
	}
	ver.Nodes = nodes
	return ver, nil
}

// GetLatestPublishedByTemplate 返回模板最新的已发布版本（含节点）。
func (r *FlowTemplateVersionRepository) GetLatestPublishedByTemplate(ctx context.Context, templateID int64) (*entity.FlowTemplateVersion, error) {
	var m FlowTemplateVersionModel
	if err := r.db.WithContext(ctx).
		Where("template_id = ? AND status = ?", templateID, int8(1)).
		Order("version DESC").
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	ver := toVersionEntity(&m)
	nodes, err := r.loadNodes(ctx, ver.TemplateID, ver.ID)
	if err != nil {
		return nil, err
	}
	ver.Nodes = nodes
	return ver, nil
}

// ListPublishedCronTriggers 返回各模板最新已发布版本中开启 cron 的规则。
func (r *FlowTemplateVersionRepository) ListPublishedCronTriggers(ctx context.Context) ([]entity.PublishedCronRule, error) {
	var models []FlowTemplateVersionModel
	published := valueobject.TemplateStatusPublishedInt
	err := r.db.WithContext(ctx).Raw(`
		SELECT v.*
		FROM flow_template_versions v
		INNER JOIN (
			SELECT template_id, MAX(version) AS max_version
			FROM flow_template_versions
			WHERE status = ?
			GROUP BY template_id
		) latest ON v.template_id = latest.template_id AND v.version = latest.max_version
		WHERE v.status = ?
	`, published, published).Scan(&models).Error
	if err != nil {
		return nil, err
	}
	out := make([]entity.PublishedCronRule, 0)
	for i := range models {
		trigger := unmarshalTriggerConfig(models[i].TriggerConfig)
		if !trigger.Cron || trigger.CronExpr == "" {
			continue
		}
		out = append(out, entity.PublishedCronRule{
			TemplateID: models[i].TemplateID,
			VersionID:  models[i].ID,
			CronExpr:   trigger.CronExpr,
		})
	}
	return out, nil
}

// ListByTemplate 查询某个模板下的所有版本（不加载节点）。
func (r *FlowTemplateVersionRepository) ListByTemplate(ctx context.Context, templateID int64) ([]*entity.FlowTemplateVersion, error) {
	var models []FlowTemplateVersionModel
	if err := r.db.WithContext(ctx).
		Where("template_id = ?", templateID).
		Order("version DESC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	vers := make([]*entity.FlowTemplateVersion, 0, len(models))
	for i := range models {
		vers = append(vers, toVersionEntity(&models[i]))
	}
	return vers, nil
}

// Update 更新版本快照（不含节点，节点由 NodeRepository 管理）。
func (r *FlowTemplateVersionRepository) Update(ctx context.Context, ver *entity.FlowTemplateVersion) error {
	return r.db.WithContext(ctx).Save(toVersionModel(ver)).Error
}

// loadNodes 加载版本下的所有节点。
func (r *FlowTemplateVersionRepository) loadNodes(ctx context.Context, templateID, versionID int64) ([]*entity.Node, error) {
	var models []NodeModel
	if err := r.db.WithContext(ctx).
		Where("template_id = ? AND version_id = ?", templateID, versionID).
		Find(&models).Error; err != nil {
		return nil, err
	}
	nodes := make([]*entity.Node, 0, len(models))
	for i := range models {
		nodes = append(nodes, toNodeEntity(&models[i]))
	}
	return nodes, nil
}

// --- 转换函数 ---

// triggerConfigJSON 是触发配置快照的存储结构。
type triggerConfigJSON struct {
	Manual        bool   `json:"manual"`
	Webhook       bool   `json:"webhook"`
	Cron          bool   `json:"cron"`
	CronExpr      string `json:"cron_expr"`
	RequireParams bool   `json:"require_params"`
	WebhookSecret string `json:"webhook_secret"`
}

func marshalTriggerConfig(t entity.TriggerConfig) string {
	data, err := json.Marshal(triggerConfigJSON{
		Manual:        t.Manual,
		Webhook:       t.Webhook,
		Cron:          t.Cron,
		CronExpr:      t.CronExpr,
		RequireParams: t.RequireParams,
		WebhookSecret: t.WebhookSecret,
	})
	if err != nil {
		return ""
	}
	return string(data)
}

// unmarshalTriggerConfig 解析触发配置快照；历史数据没有快照时默认开启手动触发。
func unmarshalTriggerConfig(raw string) entity.TriggerConfig {
	cfg := entity.TriggerConfig{Manual: true}
	if strings.TrimSpace(raw) == "" {
		return cfg
	}
	var parsed triggerConfigJSON
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return cfg
	}
	cfg.Manual = parsed.Manual
	cfg.Webhook = parsed.Webhook
	cfg.Cron = parsed.Cron
	cfg.CronExpr = parsed.CronExpr
	cfg.RequireParams = parsed.RequireParams
	cfg.WebhookSecret = parsed.WebhookSecret
	return cfg
}

func toVersionModel(ver *entity.FlowTemplateVersion) *FlowTemplateVersionModel {
	return &FlowTemplateVersionModel{
		ID:            ver.ID,
		TemplateID:    ver.TemplateID,
		Version:       ver.Version,
		Status:        ver.Status.Int(),
		TriggerConfig: marshalTriggerConfig(ver.Trigger),
		CreatedAt:     ver.CreatedAt,
		UpdatedAt:     ver.UpdatedAt,
	}
}

func toVersionEntity(m *FlowTemplateVersionModel) *entity.FlowTemplateVersion {
	return &entity.FlowTemplateVersion{
		ID:         m.ID,
		TemplateID: m.TemplateID,
		Version:    m.Version,
		Status:     valueobject.TemplateStatusFromInt(m.Status),
		Nodes:      []*entity.Node{},
		Trigger:    unmarshalTriggerConfig(m.TriggerConfig),
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}
