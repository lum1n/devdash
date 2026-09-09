package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeMigratesLegacyRoots(t *testing.T) {
	cfg := Config{Roots: []string{"/home/u/repos"}, Focus: Focus{Pinned: []string{"sessh"}}}
	cfg.NormalizeWorkspaces()
	if len(cfg.Workspaces) != 1 || cfg.Workspaces[0].ID != "local" {
		t.Fatalf("ws=%v", cfg.Workspaces)
	}
	if cfg.Active != "local" || cfg.Workspaces[0].Focus.IsPinned("sessh") == false {
		t.Fatalf("active=%s focus=%v", cfg.Active, cfg.Workspaces[0].Focus)
	}
	if got := cfg.ScanRoots(); len(got) != 1 || got[0] != "/home/u/repos" {
		t.Fatalf("roots=%v", got)
	}
}

func TestLoadWorkspacesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	raw := []byte(`listen: 127.0.0.1:8789
active: work
workspaces:
  - id: work
    name: work
    roots: [/work]
  - id: private
    name: private
    roots: [/home/u/repos]
  - id: private-remote
    name: private remote
    kind: ssh
    host: vegard@box
    url: 127.0.0.1:8790
`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Active != "work" || cfg.ActiveWorkspace().ID != "work" {
		t.Fatalf("active=%+v", cfg.ActiveWorkspace())
	}
	if cfg.ScanRoots()[0] != "/work" {
		t.Fatalf("roots=%v", cfg.ScanRoots())
	}
	remote := cfg.Workspaces[2]
	if !remote.IsSSH() || remote.Host != "vegard@box" {
		t.Fatalf("remote=%+v", remote)
	}
}

func TestScanRootsEnvDoesNotRewriteWorkspace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	raw := []byte(`listen: 127.0.0.1:8789
active: work
workspaces:
  - id: work
    name: work
    roots: [/Users/you/work]
`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEVDASH_ROOTS", "/repos")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Workspaces[0].Roots[0] != "/Users/you/work" {
		t.Fatalf("workspace roots=%v", cfg.Workspaces[0].Roots)
	}
	if got := cfg.ScanRoots(); len(got) != 1 || got[0] != "/repos" {
		t.Fatalf("scan=%v", got)
	}
}

func TestSlugID(t *testing.T) {
	if slugID("Private Remote") != "private-remote" {
		t.Fatal(slugID("Private Remote"))
	}
}
