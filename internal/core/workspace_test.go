package core

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lum1n/devdash/internal/config"
	"github.com/lum1n/devdash/internal/plugin"
)

func TestSelectWorkspaceScansItsRoots(t *testing.T) {
	rootA := t.TempDir()
	rootB := t.TempDir()
	initGit(t, filepath.Join(rootA, "alpha"))
	initGit(t, filepath.Join(rootB, "beta"))
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("listen: 127.0.0.1:8789\nroots:\n  - "+rootA+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	app, err := New(cfgPath, &plugin.Registry{})
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	ov, err := app.Overview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(ov.Repos) != 1 || ov.Repos[0].ID != "alpha" {
		t.Fatalf("first scan=%v", ov.Repos)
	}
	if len(ov.Workspaces) != 1 || ov.Workspace.ID != "local" {
		t.Fatalf("ws=%v active=%v", ov.Workspaces, ov.Workspace)
	}

	if err := app.AddWorkspace(context.Background(), config.Workspace{
		ID: "private", Name: "private", Roots: []string{rootB},
	}); err != nil {
		t.Fatal(err)
	}
	ov, err = app.Overview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if ov.Workspace.ID != "private" || len(ov.Repos) != 1 || ov.Repos[0].ID != "beta" {
		t.Fatalf("after switch workspace=%v repos=%v", ov.Workspace, ov.Repos)
	}
	if err := app.SelectWorkspace(context.Background(), "local"); err != nil {
		t.Fatal(err)
	}
	ov, err = app.Overview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if ov.Workspace.ID != "local" || ov.Repos[0].ID != "alpha" {
		t.Fatalf("back=%v repos=%v", ov.Workspace, ov.Repos)
	}
}

func TestOverviewEmptySlicesAreJSONArrays(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	empty := filepath.Join(dir, "empty")
	if err := os.MkdirAll(empty, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, []byte("listen: 127.0.0.1:8789\nroots:\n  - "+empty+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	app, err := New(cfgPath, &plugin.Registry{})
	if err != nil {
		t.Fatal(err)
	}
	ov, err := app.Overview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if ov.Repos == nil || ov.Today == nil || ov.Palette == nil || ov.Roots == nil {
		t.Fatalf("nil slices repos=%v today=%v palette=%v roots=%v", ov.Repos, ov.Today, ov.Palette, ov.Roots)
	}
	raw, err := json.Marshal(ov)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	for _, key := range []string{"repos", "today", "palette", "roots", "workspaces"} {
		if strings.Contains(s, `"`+key+`":null`) {
			t.Fatalf("%s encoded as null: %s", key, s)
		}
	}
}

func initGit(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init", "-b", "main")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init %s: %v\n%s", dir, err, out)
	}
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"config", "user.email", "devdash@test"},
		{"config", "user.name", "devdash"},
		{"add", "."},
		{"commit", "-m", "init"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}
