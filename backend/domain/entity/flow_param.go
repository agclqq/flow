package entity

import (
	"errors"
	"time"

	"flow/domain/valueobject"
)

// FlowParam 是流水线的参数定义，声明流水线接受哪些触发入参。
// 参数定义属于版本快照内容，随 FlowTemplateVersion 一起版本化。
type FlowParam struct {
	ID         int64
	TemplateID int64
	VersionID  int64
	Name       string
	Type       valueobject.ParamType
	// DefaultValue 参数默认值（可选）。
	DefaultValue string
	// Required 是否必填。
	Required bool
	// Description 参数说明（可选）。
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewFlowParam 创建参数定义，校验必填项与类型合法性。
// ID 由数据库自增生成。
func NewFlowParam(templateID, versionID int64, name string, paramType valueobject.ParamType) (*FlowParam, error) {
	if templateID == 0 || versionID == 0 {
		return nil, errors.New("template id and version id must not be zero")
	}
	if name == "" {
		return nil, errors.New("param name must not be empty")
	}
	if !paramType.Valid() {
		return nil, errors.New("invalid param type")
	}
	now := time.Now()
	return &FlowParam{
		TemplateID: templateID,
		VersionID:  versionID,
		Name:       name,
		Type:       paramType,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}
