package scan

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestInspectDetailSignalsAndActivity(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	d, err := InspectDetail(context.Background(), dir, dir)
	if err != nil {
		t.Fatal(err)
	}
	if d.Repo.Dirty != true || len(d.Files) == 0 {
		t.Fatalf("dirty files=%v repo=%+v", d.Files, d.Repo)
	}
	if len(d.Commits) == 0 || d.Commits[0].Subject != "init" {
		t.Fatalf("commits=%v", d.Commits)
	}
	if d.CommitN16w < 1 {
		t.Fatalf("activity empty: %+v", d.Activity)
	}
	if d.Summary != "hi" {
		t.Fatalf("summary=%q", d.Summary)
	}
	foundDirty := false
	for _, s := range d.Signals {
		if s.ID == "dirty" {
			foundDirty = true
		}
	}
	if !foundDirty {
		t.Fatalf("signals=%v", d.Signals)
	}
}

func TestRemoteWebURL(t *testing.T) {
	got := RemoteWebURL("git@github.com:lum1n/sessh.git")
	if got != "https://github.com/lum1n/sessh" {
		t.Fatalf("got %s", got)
	}
	got = RemoteWebURL("https://github.com/lum1n/prui.git")
	if got != "https://github.com/lum1n/prui" {
		t.Fatalf("got %s", got)
	}
}
