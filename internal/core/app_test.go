package core

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/lum1n/devdash/internal/plugin"
)

func TestAddRootPersistsAndScans(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("listen: 127.0.0.1:8789\nroots: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(dir, "repos")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	app, err := New(cfgPath, &plugin.Registry{})
	if err != nil {
		t.Fatal(err)
	}
	if err := app.AddRoot(root); err != nil {
		t.Fatal(err)
	}
	if err := app.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	ov, err := app.Overview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(ov.Roots) != 1 {
		t.Fatalf("roots=%v", ov.Roots)
	}
}
