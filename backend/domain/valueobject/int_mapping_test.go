package valueobject

import "testing"

func TestTemplateStatusIntMapping(t *testing.T) {
	if TemplateStatusDraft.Int() != TemplateStatusDraftInt {
		t.Error("draft int mismatch")
	}
	if TemplateStatusPublished.Int() != TemplateStatusPublishedInt {
		t.Error("published int mismatch")
	}
	if TemplateStatusFromInt(TemplateStatusDraftInt) != TemplateStatusDraft {
		t.Error("draft from int mismatch")
	}
	if TemplateStatusFromInt(TemplateStatusPublishedInt) != TemplateStatusPublished {
		t.Error("published from int mismatch")
	}
}

func TestFlowInstanceStatusIntMapping(t *testing.T) {
	cases := []FlowInstanceStatus{
		FlowInstanceStatusPending,
		FlowInstanceStatusRunning,
		FlowInstanceStatusSucceeded,
		FlowInstanceStatusFailed,
		FlowInstanceStatusRetry,
	}
	for _, s := range cases {
		if FlowInstanceStatusFromInt(s.Int()) != s {
			t.Errorf("roundtrip mismatch for %s", s)
		}
	}
}

func TestNodeInstanceStatusIntMapping(t *testing.T) {
	cases := []NodeInstanceStatus{
		NodeInstanceStatusPending,
		NodeInstanceStatusRunning,
		NodeInstanceStatusSucceeded,
		NodeInstanceStatusFailedSkipped,
		NodeInstanceStatusFailed,
		NodeInstanceStatusRetrying,
		NodeInstanceStatusWaitingApproval,
	}
	for _, s := range cases {
		if NodeInstanceStatusFromInt(s.Int()) != s {
			t.Errorf("roundtrip mismatch for %s", s)
		}
	}
}

func TestNodeTypeIntMapping(t *testing.T) {
	cases := []NodeType{NodeTypeStart, NodeTypeEnd, NodeTypeTask, NodeTypeEdge, NodeTypeJoin, NodeTypeManual}
	for _, s := range cases {
		if NodeTypeFromInt(s.Int()) != s {
			t.Errorf("roundtrip mismatch for %s", s)
		}
	}
}

func TestStartModeIntMapping(t *testing.T) {
	if StartModeAll.Int() != StartModeAllInt {
		t.Error("all int mismatch")
	}
	if StartModeAny.Int() != StartModeAnyInt {
		t.Error("any int mismatch")
	}
	if StartModeCustom.Int() != StartModeCustomInt {
		t.Error("custom int mismatch")
	}
	if StartModeFromInt(StartModeAnyInt) != StartModeAny {
		t.Error("any from int mismatch")
	}
	if StartModeFromInt(StartModeAllInt) != StartModeAll {
		t.Error("all from int mismatch")
	}
	if StartModeFromInt(StartModeCustomInt) != StartModeCustom {
		t.Error("custom from int mismatch")
	}
}

func TestParamTypeIntMapping(t *testing.T) {
	cases := []ParamType{ParamTypeString, ParamTypeNumber, ParamTypeBoolean}
	for _, s := range cases {
		if ParamTypeFromInt(s.Int()) != s {
			t.Errorf("roundtrip mismatch for %s", s)
		}
	}
}

func TestAuthTypeIntMapping(t *testing.T) {
	if AuthTypePassword.Int() != AuthTypePasswordInt {
		t.Error("password int mismatch")
	}
	if AuthTypeSSHKey.Int() != AuthTypeSSHKeyInt {
		t.Error("sshkey int mismatch")
	}
	if AuthTypeFromInt(AuthTypeSSHKeyInt) != AuthTypeSSHKey {
		t.Error("sshkey from int mismatch")
	}
}

func TestTriggerTypeIntMapping(t *testing.T) {
	cases := []TriggerType{TriggerTypeManual, TriggerTypeWebhook, TriggerTypeCron}
	for _, s := range cases {
		if TriggerTypeFromInt(s.Int()) != s {
			t.Errorf("roundtrip mismatch for %s", s)
		}
	}
}
