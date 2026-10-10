package entity

import (
	"testing"

	"flow/domain/valueobject"
)

func TestNewK8sConfig(t *testing.T) {
	c, err := NewK8sConfig("prod", "https://k8s.example.com", "", "token", "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Server != "https://k8s.example.com" {
		t.Errorf("expected server, got %v", c.Server)
	}
	if _, err := NewK8sConfig("prod", "", "", "", ""); err == nil {
		t.Error("expected error for empty server")
	}
}

func TestNewBareMetalConfig(t *testing.T) {
	c, err := NewBareMetalConfig("node1", "10.0.0.1", 22)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Host != "10.0.0.1" || c.Port != 22 {
		t.Errorf("unexpected config: %+v", c)
	}
	if c.AuthType != valueobject.AuthTypePassword {
		t.Errorf("expected default auth type password, got %v", c.AuthType)
	}
	if _, err := NewBareMetalConfig("node1", "10.0.0.1", 0); err == nil {
		t.Error("expected error for non-positive port")
	}
}

func TestBareMetalConfigAuthType(t *testing.T) {
	c, _ := NewBareMetalConfig("node1", "10.0.0.1", 22)
	if err := c.SetAuthType(valueobject.AuthTypeSSHKey); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.AuthType != valueobject.AuthTypeSSHKey {
		t.Errorf("expected sshkey auth type, got %v", c.AuthType)
	}
	if err := c.SetAuthType(valueobject.AuthType("bad")); err == nil {
		t.Error("expected error for invalid auth type")
	}
}

func TestNewRepoConfig(t *testing.T) {
	c, err := NewRepoConfig("repo", "git@example.com:team/repo.git")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.URL != "git@example.com:team/repo.git" {
		t.Errorf("expected url, got %v", c.URL)
	}
	if _, err := NewRepoConfig("repo", ""); err == nil {
		t.Error("expected error for empty url")
	}
}

func TestTriggerConfig(t *testing.T) {
	c, err := NewTriggerConfig(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c.Manual = true
	c.Cron = true
	c.CronExpr = "0 * * * * *"

	if !c.HasTrigger(valueobject.TriggerTypeManual) {
		t.Error("expected manual trigger enabled")
	}
	if !c.HasTrigger(valueobject.TriggerTypeCron) {
		t.Error("expected cron trigger enabled")
	}
	if c.HasTrigger(valueobject.TriggerTypeWebhook) {
		t.Error("expected webhook disabled")
	}
}

func TestNotifyTemplate(t *testing.T) {
	nt, err := NewNotifyTemplate("build-notify", "构建完成 {{flow_name}}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nt.Content != "构建完成 {{flow_name}}" {
		t.Errorf("expected content, got %v", nt.Content)
	}
	if _, err := NewNotifyTemplate("name", ""); err == nil {
		t.Error("expected error for empty content")
	}
}

func TestFlowNotifyBinding(t *testing.T) {
	b, err := NewFlowNotifyBinding(1, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := b.AddTiming(valueobject.NotifyTimingSuccess); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := b.AddTiming(valueobject.NotifyTimingFailure); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(b.Timings) != 2 {
		t.Errorf("expected 2 timings, got %d", len(b.Timings))
	}
	if b.NotifyTemplateID != 1 {
		t.Errorf("expected notify template id 1, got %d", b.NotifyTemplateID)
	}
	if err := b.AddTiming(valueobject.NotifyTiming("bad")); err == nil {
		t.Error("expected error for invalid timing")
	}
}
