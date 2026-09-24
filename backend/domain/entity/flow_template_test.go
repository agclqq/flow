package entity

import (
	"testing"

	"flow/domain/valueobject"
)

func TestNewFlowTemplate(t *testing.T) {
	t.Run("创建成功", func(t *testing.T) {
		tpl, err := NewFlowTemplate("build")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tpl.Name != "build" {
			t.Errorf("expected name 'build', got %q", tpl.Name)
		}
		if tpl.Version != 1 {
			t.Errorf("expected version 1, got %d", tpl.Version)
		}
		if tpl.Status != valueobject.TemplateStatusDraft {
			t.Errorf("expected draft status, got %v", tpl.Status)
		}
	})

	t.Run("名称为空应报错", func(t *testing.T) {
		if _, err := NewFlowTemplate(""); err == nil {
			t.Fatal("expected error for empty name, got nil")
		}
	})

	t.Run("名称超长应报错", func(t *testing.T) {
		longName := make([]byte, maxNameLength+1)
		for i := range longName {
			longName[i] = 'a'
		}
		if _, err := NewFlowTemplate(string(longName)); err == nil {
			t.Fatal("expected error for overlong name, got nil")
		}
	})
}

func TestFlowTemplatePublish(t *testing.T) {
	tpl, _ := NewFlowTemplate("build")

	if err := tpl.PublishSnapshot(2); err != nil {
		t.Fatalf("unexpected publish error: %v", err)
	}
	if tpl.Status != valueobject.TemplateStatusPublished {
		t.Errorf("expected published status, got %v", tpl.Status)
	}
	if tpl.Version != 2 {
		t.Errorf("expected version 2 after publish, got %d", tpl.Version)
	}

	// git 模型：发布可重复（打 tag），current_version 跟随最新快照号。
	if err := tpl.PublishSnapshot(3); err != nil {
		t.Fatalf("unexpected error on repeated publish: %v", err)
	}
	if tpl.Version != 3 {
		t.Errorf("expected version 3 after second publish, got %d", tpl.Version)
	}

	if err := tpl.PublishSnapshot(0); err == nil {
		t.Error("expected error for non-positive snapshot version")
	}
}

func TestFlowTemplateSoftDelete(t *testing.T) {
	tpl, _ := NewFlowTemplate("build")
	if tpl.IsDeleted() {
		t.Fatal("expected not deleted initially")
	}
	tpl.SoftDelete()
	if !tpl.IsDeleted() {
		t.Fatal("expected deleted after SoftDelete")
	}
}
