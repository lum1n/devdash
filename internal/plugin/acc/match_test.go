package acc

import (
	"path/filepath"
	"testing"
)

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
