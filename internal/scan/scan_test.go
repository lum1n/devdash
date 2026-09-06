package scan

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRootFindsImmediateGitRepos(t *testing.T) {
	dir := t.TempDir()
	alpha := filepath.Join(dir, "alpha")
	skip := filepath.Join(dir, "node_modules")
	if err := os.MkdirAll(alpha, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(skip, 0o755); err != nil {
		t.Fatal(err)
	}
	initRepo(t, alpha)
	if err := os.WriteFile(filepath.Join(alpha, "go.mod"), []byte("module alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	repos, err := Root(context.Background(), dir, Options{Ignore: []string{"node_modules"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 {
		t.Fatalf("got %d repos, want 1", len(repos))
	}
	if repos[0].Name != "alpha" {
		t.Fatalf("name=%s", repos[0].Name)
	}
	if repos[0].Stack != "go" {
		t.Fatalf("stack=%s", repos[0].Stack)
	}
	if repos[0].Branch == "" {
		t.Fatal("expected branch")
	}
}

func TestRootPrefersChildReposOverWrapperGit(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	child := filepath.Join(dir, "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	initRepo(t, child)

	repos, err := Root(context.Background(), dir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 || repos[0].Name != "child" {
		t.Fatalf("got %#v", repos)
	}
}

func initRepo(t *testing.T, dir string) {
	t.Helper()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=devdash", "GIT_AUTHOR_EMAIL=devdash@example.com",
			"GIT_COMMITTER_NAME=devdash", "GIT_COMMITTER_EMAIL=devdash@example.com")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "devdash@example.com")
	run("config", "user.name", "devdash")
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "README")
	run("commit", "-m", "init")
}
