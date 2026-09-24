package entity

import (
	"testing"

	"flow/domain/valueobject"
)

func TestNewNode(t *testing.T) {
	n, err := NewNode(1, 1, valueobject.NodeTypeTask, "build")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !n.IsTask() {
		t.Error("expected task node")
	}
	if n.IsEdge() {
		t.Error("task node should not be edge")
	}
	if n.TemplateID != 1 {
		t.Errorf("expected template id 1, got %d", n.TemplateID)
	}
	if n.StartMode != valueobject.StartModeAny {
		t.Errorf("expected default start mode any, got %v", n.StartMode)
	}

	if _, err := NewNode(1, 1, valueobject.NodeType("bad"), "x"); err == nil {
		t.Error("expected error for invalid type")
	}
	if _, err := NewNode(1, 1, valueobject.NodeTypeJoin, "x"); err == nil {
		t.Error("expected error for deprecated join type")
	}
}

func TestNodeSetEdge(t *testing.T) {
	edge, _ := NewNode(1, 1, valueobject.NodeTypeEdge, "edge")
	if err := edge.SetEdge(1, 2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if edge.FromNodeID != 1 || edge.ToNodeID != 2 {
		t.Errorf("expected from 1 to 2, got %d -> %d", edge.FromNodeID, edge.ToNodeID)
	}

	task, _ := NewNode(1, 1, valueobject.NodeTypeTask, "task")
	if err := task.SetEdge(1, 2); err == nil {
		t.Error("expected error when setting edge on task node")
	}
}

func TestNodeAddOwner(t *testing.T) {
	task, _ := NewNode(1, 1, valueobject.NodeTypeTask, "build")
	if err := task.AddOwner("alice"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := task.AddOwner("bob"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(task.Owners) != 2 {
		t.Errorf("expected 2 owners, got %d", len(task.Owners))
	}

	edge, _ := NewNode(1, 1, valueobject.NodeTypeEdge, "edge")
	if err := edge.AddOwner("alice"); err == nil {
		t.Error("expected error when adding owner to edge node")
	}

	manual, _ := NewNode(1, 1, valueobject.NodeTypeManual, "approve")
	if err := manual.AddOwner("alice"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
