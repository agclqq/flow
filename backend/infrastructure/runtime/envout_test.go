package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveEnvOutInstanceDirFallback(t *testing.T) {
	dir := ResolveEnvOutInstanceDir("", 12, 34)
	want := filepath.Join("/tmp", "flow", "env", "12", "34")
	if dir != want {
		t.Fatalf("got %q want %q", dir, want)
	}
}

func TestResolveEnvOutInstanceDirConfigured(t *testing.T) {
	root := t.TempDir()
	dir := ResolveEnvOutInstanceDir(root, 1, 2)
	want := filepath.Join(root, "1", "2")
	if dir != want {
		t.Fatalf("got %q want %q", dir, want)
	}
}

func TestEnsureEnvOutFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "9.env.out")
	if err := EnsureEnvOutFile(path); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Size() != 0 {
		t.Fatalf("expected empty file, size %d", st.Size())
	}
}
