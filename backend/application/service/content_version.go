package service

import (
	"errors"

	"flow/domain/entity"
	"flow/domain/valueobject"
)

// pickContentVersion 返回应作为内容来源的版本：优先最新草稿，否则最新已发布版本。
// vers 需按 version DESC 排序（与 ListByTemplate 一致）。
func pickContentVersion(vers []*entity.FlowTemplateVersion) (*entity.FlowTemplateVersion, error) {
	for _, v := range vers {
		if v.Status == valueobject.TemplateStatusDraft {
			return v, nil
		}
	}
	for _, v := range vers {
		if v.Status == valueobject.TemplateStatusPublished {
			return v, nil
		}
	}
	return nil, errors.New("no content version found")
}

func cloneFloat64Ptr(v *float64) *float64 {
	if v == nil {
		return nil
	}
	copied := *v
	return &copied
}
