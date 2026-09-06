package scan

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectDiffWorktreeOneFile(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("hi\nchanged\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("fresh\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	readme, err := InspectDiff(context.Background(), dir, "worktree", "README")
	if err != nil {
		t.Fatal(err)
	}
	if readme.Path != "README" || readme.Empty || !strings.Contains(readme.Patch, "changed") {
		t.Fatalf("readme=%+v", readme)
	}
	if strings.Contains(readme.Patch, "fresh") || contains(readme.Files, "new.txt") {
		t.Fatalf("readme leaked other file: %+v", readme)
	}

	fresh, err := InspectDiff(context.Background(), dir, "worktree", "new.txt")
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Empty || !strings.Contains(fresh.Patch, "fresh") || strings.Contains(fresh.Patch, "changed") {
		t.Fatalf("new=%+v", fresh)
	}
}

func TestInspectDiffWorktreeNeedsFile(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	if _, err := InspectDiff(context.Background(), dir, "worktree", ""); err == nil {
		t.Fatal("expected path required")
	}
	if _, err := InspectDiff(context.Background(), dir, "worktree", "../oops"); err == nil {
		t.Fatal("expected invalid path")
	}
}

func TestInspectDiffCommitShowsInit(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	hash := gitHash(t, dir)

	d, err := InspectDiff(context.Background(), dir, hash, "")
	if err != nil {
		t.Fatal(err)
	}
	if d.Subject != "init" || d.Empty {
		t.Fatalf("diff=%+v", d)
	}
	if !strings.Contains(d.Patch, "README") {
		t.Fatalf("patch=%s", d.Patch)
	}
}

func TestInspectDiffRejectsBadRef(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	if _, err := InspectDiff(context.Background(), dir, "../oops", ""); err == nil {
		t.Fatal("expected invalid ref")
	}
	if _, err := InspectDiff(context.Background(), dir, "deadbeef", ""); err == nil {
		t.Fatal("expected missing commit")
	}
}

func gitHash(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--short", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
