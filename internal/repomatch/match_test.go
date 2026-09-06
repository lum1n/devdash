package repomatch

import (
	"path/filepath"
	"testing"
)

func TestMatchesRepoUsesSessionWhenPathIsCursorStore(t *testing.T) {
	root := filepath.Join(t.TempDir(), "devdash")
	store := filepath.Join(t.TempDir(), "store.db")
	if !MatchesRepo(root, "devdash", "devdash", "repos/devdash", store) {
		t.Fatal("session repos/devdash should match")
	}
	if !MatchesRepo(root, "devdash", "devdash", "devdash", store) {
		t.Fatal("session devdash should match")
	}
	if MatchesRepo(root, "devdash", "devdash", "sessh", store) {
		t.Fatal("other session")
	}
	if !MatchesRepo(root, "devdash", "devdash", "other", root) {
		t.Fatal("cwd path should still match")
	}
}

func TestSameTree(t *testing.T) {
	root := filepath.Join(t.TempDir(), "sessh")
	if !SameTree(root, root) {
		t.Fatal("same path")
	}
	if !SameTree(root, filepath.Join(root, "cmd")) {
		t.Fatal("child")
	}
	if SameTree(root, root+"-other") {
		t.Fatal("prefix sibling")
	}
	if SameTree(root, "") {
		t.Fatal("empty")
	}
}
