package notes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateListWriteDelete(t *testing.T) {
	dir := t.TempDir()
	n, err := Create(dir, "Shipping iOS", "")
	if err != nil {
		t.Fatal(err)
	}
	if n.Name != "shipping-ios.md" || n.Title != "Shipping iOS" {
		t.Fatalf("note=%+v", n)
	}
	if _, err := os.Stat(filepath.Join(dir, "notes", "shipping-ios.md")); err != nil {
		t.Fatal(err)
	}
	list, err := List(dir)
	if err != nil || len(list) != 1 {
		t.Fatalf("list=%v err=%v", list, err)
	}
	got, err := Write(dir, n.Name, "# Shipping iOS\n\n- [ ] TestFlight\n")
	if err != nil || !strings.Contains(got.Content, "TestFlight") {
		t.Fatalf("write=%+v err=%v", got, err)
	}
	if err := Delete(dir, n.Name); err != nil {
		t.Fatal(err)
	}
	list, err = List(dir)
	if err != nil || len(list) != 0 {
		t.Fatalf("after delete %v %v", list, err)
	}
}

func TestRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	if _, err := Read(dir, "../secret.md"); err == nil {
		t.Fatal("expected reject")
	}
	if _, err := Write(dir, "foo/bar.md", "x"); err == nil {
		t.Fatal("expected reject")
	}
	if _, err := Write(dir, "ok.md", "x"); err != nil {
		t.Fatal(err)
	}
}
