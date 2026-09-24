package valueobject

import "testing"

func TestFlowInstanceStateTransitions(t *testing.T) {
	tests := []struct {
		from   FlowInstanceStatus
		to     FlowInstanceStatus
		expect bool
	}{
		{FlowInstanceStatusPending, FlowInstanceStatusRunning, true},
		{FlowInstanceStatusRunning, FlowInstanceStatusSucceeded, true},
		{FlowInstanceStatusRunning, FlowInstanceStatusFailed, true},
		{FlowInstanceStatusFailed, FlowInstanceStatusRetry, true},
		{FlowInstanceStatusRetry, FlowInstanceStatusRunning, true},
		// 非法流转
		{FlowInstanceStatusPending, FlowInstanceStatusSucceeded, false},
		{FlowInstanceStatusRunning, FlowInstanceStatusPending, false},
		{FlowInstanceStatusSucceeded, FlowInstanceStatusRunning, false},
	}
	for _, tt := range tests {
		got := tt.from.CanTransitionTo(tt.to)
		if got != tt.expect {
			t.Errorf("%s -> %s: expected %v, got %v", tt.from, tt.to, tt.expect, got)
		}
	}
}

func TestNodeInstanceStateTransitions(t *testing.T) {
	tests := []struct {
		from   NodeInstanceStatus
		to     NodeInstanceStatus
		expect bool
	}{
		{NodeInstanceStatusPending, NodeInstanceStatusRunning, true},
		{NodeInstanceStatusRunning, NodeInstanceStatusSucceeded, true},
		{NodeInstanceStatusRunning, NodeInstanceStatusFailedSkipped, true},
		{NodeInstanceStatusRunning, NodeInstanceStatusFailed, true},
		{NodeInstanceStatusRunning, NodeInstanceStatusWaitingApproval, true},
		{NodeInstanceStatusWaitingApproval, NodeInstanceStatusSucceeded, true},
		{NodeInstanceStatusWaitingApproval, NodeInstanceStatusFailed, true},
		{NodeInstanceStatusFailed, NodeInstanceStatusRetrying, true},
		{NodeInstanceStatusRetrying, NodeInstanceStatusRunning, true},
		// 非法流转
		{NodeInstanceStatusPending, NodeInstanceStatusSucceeded, false},
		{NodeInstanceStatusFailedSkipped, NodeInstanceStatusRunning, false},
		{NodeInstanceStatusSucceeded, NodeInstanceStatusRetrying, false},
	}
	for _, tt := range tests {
		got := tt.from.CanTransitionTo(tt.to)
		if got != tt.expect {
			t.Errorf("%s -> %s: expected %v, got %v", tt.from, tt.to, tt.expect, got)
		}
	}
}

func TestFlowInstanceTerminal(t *testing.T) {
	if !FlowInstanceStatusSucceeded.IsTerminal() {
		t.Error("succeeded should be terminal")
	}
	if FlowInstanceStatusFailed.IsTerminal() {
		t.Error("failed should not be terminal (can retry)")
	}
}

func TestNodeTypeValid(t *testing.T) {
	for _, typ := range []NodeType{NodeTypeStart, NodeTypeEnd, NodeTypeTask, NodeTypeEdge, NodeTypeManual} {
		if !typ.Valid() {
			t.Errorf("%s should be valid", typ)
		}
	}
	if NodeTypeJoin.Valid() {
		t.Error("deprecated join should not be valid for new nodes")
	}
	if !NodeTypeJoin.ValidStored() {
		t.Error("join should still be valid in stored data")
	}
	if NodeType("invalid").Valid() {
		t.Error("invalid type should not be valid")
	}
}
