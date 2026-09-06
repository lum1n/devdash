package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAppliesListenAndRootsEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("listen: 127.0.0.1:1\nroots:\n  - /old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEVDASH_LISTEN", "0.0.0.0:8789")
	t.Setenv("DEVDASH_ROOTS", "/repos"+string(os.PathListSeparator)+"/work")

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != "0.0.0.0:8789" {
		t.Fatalf("listen=%s", cfg.Listen)
	}
	if len(cfg.Roots) != 2 || cfg.Roots[0] != "/repos" || cfg.Roots[1] != "/work" {
		t.Fatalf("roots=%v", cfg.Roots)
	}
}
