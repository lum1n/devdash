package acc

import (
	"path/filepath"
	"strings"
)

// SameTree reports whether candidate is repoPath or a subdirectory of it.
func SameTree(repoPath, candidate string) bool {
	repoPath = cleanPath(repoPath)
	candidate = cleanPath(candidate)
	if repoPath == "" || candidate == "" {
		return false
	}
	if repoPath == candidate {
		return true
	}
	prefix := repoPath + string(filepath.Separator)
	return strings.HasPrefix(candidate, prefix)
}

func cleanPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return filepath.Clean(p)
	}
	return filepath.Clean(abs)
}
