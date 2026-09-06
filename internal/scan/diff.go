package scan

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	maxPatchBytes     = 1_500_000
	maxUntrackedBytes = 200_000
)

var commitRefRe = regexp.MustCompile(`^[0-9a-fA-F]{4,40}$`)

// Diff is a unified patch for the working tree or one commit.
type Diff struct {
	Ref       string   `json:"ref"`
	Title     string   `json:"title"`
	Subject   string   `json:"subject,omitempty"`
	Patch     string   `json:"patch"`
	Files     []string `json:"files"`
	Path      string   `json:"path,omitempty"`
	Empty     bool     `json:"empty"`
	Truncated bool     `json:"truncated,omitempty"`
}

// InspectDiff loads a commit patch, or one working-tree file when file is set.
func InspectDiff(ctx context.Context, repoPath, ref, file string) (Diff, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" || ref == "worktree" {
		return inspectWorktreeDiff(ctx, repoPath, file)
	}
	if !commitRefRe.MatchString(ref) {
		return Diff{}, fmt.Errorf("invalid diff ref %q", ref)
	}
	return inspectCommitDiff(ctx, repoPath, ref)
}

func inspectWorktreeDiff(ctx context.Context, repoPath, file string) (Diff, error) {
	file, err := cleanRelPath(file)
	if err != nil {
		return Diff{}, err
	}
	d := Diff{Ref: "worktree", Title: file, Path: file, Files: []string{}}
	patch, err := gitDiff(ctx, repoPath, "diff", "--no-color", "--no-ext-diff", "HEAD", "--", file)
	if err != nil {
		return Diff{}, err
	}
	if strings.TrimSpace(patch) == "" {
		st, sterr := os.Stat(filepath.Join(repoPath, filepath.FromSlash(file)))
		if sterr == nil && !st.IsDir() && st.Size() <= maxUntrackedBytes {
			part, uerr := gitDiff(ctx, repoPath, "diff", "--no-color", "--no-ext-diff", "--no-index", "--", "/dev/null", file)
			if uerr == nil && !strings.Contains(part, "Binary files") {
				patch = part
			}
		}
	}
	d.Patch = capPatch(patch, &d.Truncated)
	d.Files = patchFiles(d.Patch)
	if len(d.Files) == 0 && strings.TrimSpace(d.Patch) != "" {
		d.Files = []string{file}
	}
	d.Empty = strings.TrimSpace(d.Patch) == ""
	return d, nil
}

func cleanRelPath(rel string) (string, error) {
	rel = strings.TrimSpace(rel)
	if i := strings.LastIndex(rel, " -> "); i >= 0 {
		rel = strings.TrimSpace(rel[i+4:])
	}
	rel = strings.Trim(rel, `"`)
	rel = strings.ReplaceAll(rel, "\\", "/")
	if rel == "" || strings.HasPrefix(rel, "/") || strings.Contains(rel, "\x00") {
		return "", fmt.Errorf("working tree diff needs a file")
	}
	clean := path.Clean(rel)
	if clean == "." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("invalid path %q", rel)
	}
	return clean, nil
}

func inspectCommitDiff(ctx context.Context, path, ref string) (Diff, error) {
	if _, err := git(ctx, path, "rev-parse", "--verify", ref+"^{commit}"); err != nil {
		return Diff{}, fmt.Errorf("commit %s not found", ref)
	}
	subject, err := git(ctx, path, "log", "-1", "--format=%s", ref)
	if err != nil {
		return Diff{}, err
	}
	patch, err := gitDiff(ctx, path, "show", "--no-color", "--no-ext-diff", "--format=", "--patch", ref)
	if err != nil {
		return Diff{}, err
	}
	d := Diff{
		Ref:     ref,
		Title:   strings.TrimSpace(subject),
		Subject: strings.TrimSpace(subject),
		Files:   []string{},
	}
	if d.Title == "" {
		d.Title = ref
	}
	d.Patch = capPatch(patch, &d.Truncated)
	d.Files = patchFiles(d.Patch)
	d.Empty = strings.TrimSpace(d.Patch) == ""
	return d, nil
}

func gitDiff(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 1 {
			return stdout.String(), nil
		}
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%v: %s", err, strings.TrimSpace(stderr.String()))
		}
		return "", err
	}
	return stdout.String(), nil
}

func capPatch(patch string, truncated *bool) string {
	if len(patch) <= maxPatchBytes {
		return patch
	}
	*truncated = true
	cut := strings.LastIndex(patch[:maxPatchBytes], "\ndiff --git ")
	if cut <= 0 {
		return patch[:maxPatchBytes]
	}
	return patch[:cut+1]
}

var diffGitRe = regexp.MustCompile(`(?m)^diff --git a/(.+) b/(.+)$`)

func patchFiles(patch string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range diffGitRe.FindAllStringSubmatch(patch, -1) {
		name := m[2]
		if name == "/dev/null" {
			name = m[1]
		}
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	if out == nil {
		return []string{}
	}
	return out
}
