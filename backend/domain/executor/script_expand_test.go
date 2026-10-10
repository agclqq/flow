package executor

import "testing"

func TestExpandScriptPlaceholders(t *testing.T) {
	env := map[string]string{
		"FLOW_ENV_OUT": "/tmp/out.env",
		"NODE_ID":      "42",
	}
	got := ExpandScriptPlaceholders(`echo "x=1" >> {{FLOW_ENV_OUT}} # {{NODE_ID}}`, env)
	want := `echo "x=1" >> /tmp/out.env # 42`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestExpandScriptPlaceholdersMissingKey(t *testing.T) {
	got := ExpandScriptPlaceholders(`{{UNKNOWN}}`, map[string]string{})
	if got != `{{UNKNOWN}}` {
		t.Fatalf("got %q", got)
	}
}
