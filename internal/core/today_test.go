package core

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lum1n/devdash/internal/config"
	"github.com/lum1n/devdash/internal/plugin"
	"github.com/lum1n/devdash/internal/scan"
)

func TestBuildTodayScoresPluginNotes(t *testing.T) {
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	repos := []scan.Repo{{ID: "sessh", Name: "sessh", LastCommit: now}}
	notes := []plugin.Annotation{{
		RepoID: "sessh", Kind: "review", Label: "review #4", Tone: "danger",
	}}
	today := buildToday(repos, notes, config.Focus{}, now)
	if len(today) != 1 || today[0].Repo.ID != "sessh" {
		t.Fatalf("today=%v", today)
	}
	found := false
	for _, w := range today[0].Why {
		if w.ID == "review" {
			found = true
		}
	}
	if !found {
		t.Fatalf("why=%v", today[0].Why)
	}
}

func TestBuildTodayRanksPinnedAndDropsArchived(t *testing.T) {
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	repos := []scan.Repo{
		{ID: "clean", Name: "clean", LastCommit: now.Add(-time.Hour)},
		{ID: "dirty", Name: "dirty", Dirty: true, Changed: 3, LastCommit: now.Add(-time.Hour)},
		{ID: "pinned", Name: "pinned", LastCommit: now.Add(-48 * time.Hour)},
		{ID: "old", Name: "old", Dirty: true, Changed: 1, LastCommit: now.Add(-40 * 24 * time.Hour)},
		{ID: "gone", Name: "gone", Dirty: true, Changed: 9},
	}
	focus := config.Focus{
		Pinned:   []string{"pinned"},
		Archived: []string{"gone"},
		Snoozed:  map[string]string{"dirty": now.Add(2 * time.Hour).Format(time.RFC3339)},
	}
	focus.Normalize()
	focus.SetNextAction("clean", "ship TestFlight")
	today := buildToday(repos, nil, focus, now)
	if len(today) != 3 {
		t.Fatalf("today=%v", today)
	}
	if today[0].Repo.ID != "pinned" || !today[0].Pinned {
		t.Fatalf("first=%+v", today[0])
	}
	if today[1].Repo.ID != "old" {
		t.Fatalf("second=%+v", today[1])
	}
	if today[2].Repo.ID != "clean" || today[2].Next != "ship TestFlight" {
		t.Fatalf("next=%+v", today[2])
	}
}

func TestSetFocusPersistsPinAndPaletteHasJump(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("listen: 127.0.0.1:8789\nroots: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	app, err := New(cfgPath, &plugin.Registry{})
	if err != nil {
		t.Fatal(err)
	}
	app.mu.Lock()
	app.repos = []scan.Repo{{ID: "sessh", Name: "sessh", Dirty: true, Changed: 2, LastCommit: time.Now()}}
	app.scannedAt = time.Now()
	app.mu.Unlock()

	if err := app.SetFocus("sessh", "pin", 0); err != nil {
		t.Fatal(err)
	}
	ov, err := app.Overview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(ov.Today) != 1 || ov.Today[0].Repo.ID != "sessh" || !ov.Today[0].Pinned {
		t.Fatalf("today=%v", ov.Today)
	}
	if ov.Counts.Today != 1 {
		t.Fatalf("counts=%v", ov.Counts)
	}
	foundJump, foundRescan := false, false
	for _, p := range ov.Palette {
		if p.ID == "jump:sessh" {
			foundJump = true
		}
		if p.ID == "rescan" {
			foundRescan = true
		}
	}
	if !foundJump || !foundRescan {
		t.Fatalf("palette=%v", ov.Palette)
	}

	reloaded, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !reloaded.Focus.IsPinned("sessh") {
		t.Fatalf("focus=%v", reloaded.Focus)
	}
}

func TestRepoTrimsWrappingQuotes(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("listen: 127.0.0.1:8789\nroots: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	app, err := New(cfgPath, &plugin.Registry{})
	if err != nil {
		t.Fatal(err)
	}
	app.mu.Lock()
	app.repos = []scan.Repo{{ID: "kjeksdev", Name: "kjeksdev"}}
	app.scannedAt = time.Now()
	app.mu.Unlock()
	got, err := app.Repo(`kjeksdev""python3`)
	if err != nil || got.ID != "kjeksdev" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}

func TestSetNextPersistsAndRanksToday(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("listen: 127.0.0.1:8789\nroots: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	app, err := New(cfgPath, &plugin.Registry{})
	if err != nil {
		t.Fatal(err)
	}
	app.mu.Lock()
	app.repos = []scan.Repo{{ID: "sessh", Name: "sessh", LastCommit: time.Now()}}
	app.scannedAt = time.Now()
	app.mu.Unlock()

	if err := app.SetNext("sessh", "ship TestFlight"); err != nil {
		t.Fatal(err)
	}
	ov, err := app.Overview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(ov.Today) != 1 || ov.Today[0].Next != "ship TestFlight" {
		t.Fatalf("today=%v", ov.Today)
	}
	reloaded, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Focus.NextAction("sessh") != "ship TestFlight" {
		t.Fatalf("next=%v", reloaded.Focus.Next)
	}
}
