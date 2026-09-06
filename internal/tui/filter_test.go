package tui

import (
	"testing"
	"time"

	"github.com/lum1n/devdash/internal/core"
	"github.com/lum1n/devdash/internal/scan"
)

func TestVisibleRespectsKindAndArchive(t *testing.T) {
	now := time.Now()
	m := model{
		kind: kindDirty,
		ov: core.Overview{
			Repos: []scan.Repo{
				{ID: "clean", Name: "clean", LastCommit: now},
				{ID: "dirty", Name: "dirty", Dirty: true, LastCommit: now},
				{ID: "gone", Name: "gone", Dirty: true, LastCommit: now},
			},
			Focus: core.FocusView{Archived: []string{"gone"}, Pinned: []string{"dirty"}},
		},
	}
	got := m.visible()
	if len(got) != 1 || got[0].ID != "dirty" {
		t.Fatalf("dirty=%v", got)
	}
	m.kind = kindArchived
	got = m.visible()
	if len(got) != 1 || got[0].ID != "gone" {
		t.Fatalf("archived=%v", got)
	}
	if m.pinAction("dirty") != "unpin" {
		t.Fatal(m.pinAction("dirty"))
	}
	if m.archiveAction("gone") != "unarchive" {
		t.Fatal(m.archiveAction("gone"))
	}
}
