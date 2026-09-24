package entity

import (
	"testing"

	"flow/domain/valueobject"
)

func TestNewFlowParam(t *testing.T) {
	p, err := NewFlowParam(1, 1, "branch", valueobject.ParamTypeString)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != "branch" || p.Type != valueobject.ParamTypeString {
		t.Errorf("unexpected param: %+v", p)
	}
	if p.Required {
		t.Error("expected not required by default")
	}

	// 缺 name 应报错
	if _, err := NewFlowParam(1, 1, "", valueobject.ParamTypeString); err == nil {
		t.Error("expected error for empty name")
	}
	// 非法类型应报错
	if _, err := NewFlowParam(1, 1, "branch", valueobject.ParamType("bad")); err == nil {
		t.Error("expected error for invalid type")
	}
}
