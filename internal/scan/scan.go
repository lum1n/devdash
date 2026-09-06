package scan

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Repo is one discovered git working tree.
type Repo struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Root        string    `json:"root"`
	Branch      string    `json:"branch"`
	Dirty       bool      `json:"dirty"`
	Changed     int       `json:"changed"`
	Stash       int       `json:"stash"`
	Ahead       int       `json:"ahead"`
	Behind      int       `json:"behind"`
	LastCommit  time.Time `json:"last_commit"`
	LastMessage string    `json:"last_message"`
	Remote      string    `json:"remote,omitempty"`
	Stack       string    `json:"stack"`
	Attention   bool      `json:"attention"`
}

// Options control a root walk.
type Options struct {
	Ignore []string
}

// Roots walks each root and inspects immediate child git repos.
func Roots(ctx context.Context, roots []string, opt Options) ([]Repo, error) {
	var out []Repo
	seen := map[string]bool{}
	for _, root := range roots {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		repos, err := Root(ctx, root, opt)
		if err != nil {
			return out, err
		}
		for _, r := range repos {
			if seen[r.Path] {
				continue
			}
			seen[r.Path] = true
			out = append(out, r)
		}
	}
	return out, nil
}

// Root lists git repos that are immediate children of root.
// If root itself is a git repo, it is returned alone.
func Root(ctx context.Context, root string, opt Options) ([]Repo, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if !isDir(root) {
		return nil, os.ErrNotExist
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	ignore := ignoreSet(opt.Ignore)
	var paths []string
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || ignore[e.Name()] {
			continue
		}
		p := filepath.Join(root, e.Name())
		if isGit(p) {
			paths = append(paths, p)
		}
	}
	// A root that is itself a git checkout (and has no child repos) is the project.
	if len(paths) == 0 && isGit(root) {
		r, err := inspect(ctx, root, root)
		if err != nil {
			return nil, err
		}
		return []Repo{r}, nil
	}

	out := make([]Repo, 0, len(paths))
	var mu sync.Mutex
	var wg sync.WaitGroup
	errCh := make(chan error, 1)
	sem := make(chan struct{}, 8)
	for _, p := range paths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				reportErr(errCh, ctx.Err())
				return
			case sem <- struct{}{}:
				defer func() { <-sem }()
			}
			r, err := inspect(ctx, root, p)
			if err != nil {
				reportErr(errCh, err)
				return
			}
			mu.Lock()
			out = append(out, r)
			mu.Unlock()
		}(p)
	}
	wg.Wait()
	select {
	case err := <-errCh:
		if err != nil && err != ctx.Err() {
			return out, err
		}
	default:
	}
	return out, ctx.Err()
}

func reportErr(ch chan error, err error) {
	select {
	case ch <- err:
	default:
	}
}

func inspect(ctx context.Context, root, path string) (Repo, error) {
	name := filepath.Base(path)
	r := Repo{
		ID:    name,
		Name:  name,
		Path:  path,
		Root:  root,
		Stack: detectStack(path),
	}
	branch, err := git(ctx, path, "rev-parse", "--abbrev-ref", "HEAD")
	if err == nil {
		r.Branch = strings.TrimSpace(branch)
	}
	status, err := git(ctx, path, "status", "--porcelain")
	if err == nil {
		lines := nonEmpty(status)
		r.Changed = len(lines)
		r.Dirty = r.Changed > 0
	}
	stash, err := git(ctx, path, "rev-list", "--walk-reflogs", "--count", "refs/stash")
	if err == nil {
		r.Stash, _ = strconv.Atoi(strings.TrimSpace(stash))
	}
	ab, err := git(ctx, path, "rev-list", "--left-right", "--count", "@{upstream}...HEAD")
	if err == nil {
		parts := strings.Fields(strings.TrimSpace(ab))
		if len(parts) == 2 {
			r.Behind, _ = strconv.Atoi(parts[0])
			r.Ahead, _ = strconv.Atoi(parts[1])
		}
	}
	log, err := git(ctx, path, "log", "-1", "--format=%cI\t%s")
	if err == nil {
		line := strings.TrimSpace(log)
		when, msg, ok := strings.Cut(line, "\t")
		if ok {
			if t, perr := time.Parse(time.RFC3339, when); perr == nil {
				r.LastCommit = t
			}
			r.LastMessage = msg
		}
	}
	if remote, err := git(ctx, path, "remote", "get-url", "origin"); err == nil {
		r.Remote = strings.TrimSpace(remote)
	}
	r.Attention = r.Dirty || r.Behind > 0 || r.Ahead > 0
	return r, nil
}

func detectStack(path string) string {
	switch {
	case exists(filepath.Join(path, "go.mod")):
		return "go"
	case exists(filepath.Join(path, "Package.swift")) || hasSuffix(path, ".xcodeproj"):
		return "swift"
	case exists(filepath.Join(path, "app.json")) && exists(filepath.Join(path, "package.json")):
		return "expo"
	case exists(filepath.Join(path, "package.json")):
		return "ts"
	case exists(filepath.Join(path, "Cargo.toml")):
		return "rust"
	case exists(filepath.Join(path, "pyproject.toml")) || exists(filepath.Join(path, "requirements.txt")):
		return "python"
	default:
		return ""
	}
}

func git(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return stdout.String(), nil
}

func isGit(path string) bool {
	st, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil && (st.IsDir() || st.Mode().IsRegular())
}

func isDir(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func hasSuffix(dir, suffix string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), suffix) {
			return true
		}
	}
	return false
}

func ignoreSet(names []string) map[string]bool {
	m := map[string]bool{}
	for _, n := range names {
		if n != "" && n != ".git" {
			m[n] = true
		}
	}
	return m
}

func nonEmpty(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	return out
}
