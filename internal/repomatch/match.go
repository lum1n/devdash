package repomatch

import (
	"path/filepath"
	"strings"
)

// MatchesRepo reports whether a live pane belongs to repo.
// Watcher path is the project cwd for some harnesses, and a store file
// (e.g. ~/.config/cursor/chats/.../store.db) for Cursor — then we match
// the tmux session name (sessh, repos/devdash) to the repo id/name/path.
func MatchesRepo(repoPath, repoID, repoName, session, candidate string) bool {
	if SameTree(repoPath, candidate) {
		return true
	}
	return SessionHits(repoPath, repoID, repoName, session)
}

// SameTree reports whether candidate is repoPath or a subdirectory of it.
func SameTree(repoPath, candidate string) bool {
	repoPath = Clean(repoPath)
	candidate = Clean(candidate)
	if repoPath == "" || candidate == "" {
		return false
	}
	if repoPath == candidate {
		return true
	}
	prefix := repoPath + string(filepath.Separator)
	return strings.HasPrefix(candidate, prefix)
}

// SessionHits matches a tmux session name to a scanned repo.
func SessionHits(repoPath, repoID, repoName, session string) bool {
	session = strings.TrimSpace(session)
	if session == "" {
		return false
	}
	base := filepath.Base(session)
	for _, name := range []string{repoID, repoName, filepath.Base(Clean(repoPath))} {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if strings.EqualFold(session, name) || strings.EqualFold(base, name) {
			return true
		}
	}
	abs := Clean(repoPath)
	if abs == "" {
		return false
	}
	if strings.HasSuffix(abs, string(filepath.Separator)+session) {
		return true
	}
	return strings.HasSuffix(abs, string(filepath.Separator)+base)
}

// Clean absolutizes p when possible.
func Clean(p string) string {
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
