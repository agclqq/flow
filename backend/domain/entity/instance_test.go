package entity

import (
	"testing"

	"flow/domain/valueobject"
)

func TestNewFlowInstance(t *testing.T) {
	fi, err := NewFlowInstance(1, 1, 1, valueobject.TriggerTypeManual)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fi.Status != valueobject.FlowInstanceStatusPending {
		t.Errorf("expected pending status, got %v", fi.Status)
	}
	if fi.TriggerType != valueobject.TriggerTypeManual {
		t.Errorf("expected manual trigger, got %v", fi.TriggerType)
	}

	if _, err := NewFlowInstance(1, 1, 1, valueobject.TriggerType("bad")); err == nil {
		t.Error("expected error for invalid trigger type")
	}
}

func TestFlowInstanceTransition(t *testing.T) {
	fi, _ := NewFlowInstance(1, 1, 1, valueobject.TriggerTypeManual)

	if err := fi.Transition(valueobject.FlowInstanceStatusRunning); err != nil {
		t.Fatalf("unexpected transition error: %v", err)
	}
	if fi.Status != valueobject.FlowInstanceStatusRunning {
		t.Errorf("expected running, got %v", fi.Status)
	}

	// 非法流转：running -> pending 应报错
	if err := fi.Transition(valueobject.FlowInstanceStatusPending); err == nil {
		t.Error("expected error for invalid transition")
	}
}

func TestFlowInstanceEnv(t *testing.T) {
	fi, _ := NewFlowInstance(1, 1, 1, valueobject.TriggerTypeManual)
	fi.SetParams(map[string]string{"k": "v"})
	fi.UpdateEnv(map[string]string{"a": "1", "b": "2"})
	fi.UpdateEnv(map[string]string{"b": "3"})

	if fi.Params["k"] != "v" {
		t.Errorf("expected param k=v, got %v", fi.Params["k"])
	}
	if fi.Env["a"] != "1" || fi.Env["b"] != "3" {
		t.Errorf("expected env a=1 b=3, got %v", fi.Env)
	}
}

func TestNodeInstanceRetry(t *testing.T) {
	ni, _ := NewNodeInstance(1, 1, valueobject.NodeTypeTask)

	ni.Transition(valueobject.NodeInstanceStatusRunning)
	ni.Transition(valueobject.NodeInstanceStatusFailed)

	if err := ni.MarkRetrying(); err != nil {
		t.Fatalf("unexpected retry error: %v", err)
	}
	if ni.Status != valueobject.NodeInstanceStatusRetrying {
		t.Errorf("expected retrying, got %v", ni.Status)
	}
	if ni.RetryCount != 1 {
		t.Errorf("expected retry count 1, got %d", ni.RetryCount)
	}
}

func TestNodeInstanceOutputEnv(t *testing.T) {
	ni, _ := NewNodeInstance(1, 1, valueobject.NodeTypeTask)
	ni.SetOutputEnv(map[string]string{"out": "val"})
	if ni.OutputEnv["out"] != "val" {
		t.Errorf("expected out=val, got %v", ni.OutputEnv)
	}
}
