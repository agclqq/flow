package entity

import (
	"testing"

	"flow/domain/valueobject"
)

func TestNewFlowTemplateVersion(t *testing.T) {
	v, err := NewFlowTemplateVersion(1, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Status != valueobject.TemplateStatusDraft {
		t.Errorf("expected draft status, got %v", v.Status)
	}
	if len(v.Nodes) != 0 {
		t.Errorf("expected empty nodes, got %d", len(v.Nodes))
	}

	if _, err := NewFlowTemplateVersion(1, 0); err == nil {
		t.Error("expected error for non-positive version")
	}
}

func TestFlowTemplateVersionPublish(t *testing.T) {
	v, _ := NewFlowTemplateVersion(1, 1)
	if err := v.Publish(); err != nil {
		t.Fatalf("unexpected publish error: %v", err)
	}
	if v.Status != valueobject.TemplateStatusPublished {
		t.Errorf("expected published, got %v", v.Status)
	}
	if err := v.Publish(); err == nil {
		t.Error("expected error when publishing twice")
	}
}

func TestFlowTemplateVersionStartEndNode(t *testing.T) {
	v, _ := NewFlowTemplateVersion(1, 1)

	start, _ := NewNode(1, 1, valueobject.NodeTypeStart, "start")
	end, _ := NewNode(1, 1, valueobject.NodeTypeEnd, "end")
	task, _ := NewNode(1, 1, valueobject.NodeTypeTask, "task")

	v.AddNode(start)
	v.AddNode(end)
	v.AddNode(task)

	s, err := v.StartNode()
	if err != nil || s == nil {
		t.Errorf("expected start node, got %v err %v", s, err)
	}
	e, err := v.EndNode()
	if err != nil || e == nil {
		t.Errorf("expected end node, got %v err %v", e, err)
	}
}

func TestFlowTemplateVersionMultipleStart(t *testing.T) {
	v, _ := NewFlowTemplateVersion(1, 1)
	s1, _ := NewNode(1, 1, valueobject.NodeTypeStart, "start1")
	s2, _ := NewNode(1, 1, valueobject.NodeTypeStart, "start2")
	v.AddNode(s1)
	v.AddNode(s2)

	if _, err := v.StartNode(); err == nil {
		t.Error("expected error for multiple start nodes")
	}
}
