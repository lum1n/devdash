package plugin

import (
	"context"
	"testing"

	"github.com/lum1n/devdash/internal/scan"
)

type stub struct{}

func (stub) ID() string { return "stub" }
func (stub) Widgets() []Widget {
	return []Widget{{ID: "stub", Title: "stub", Summary: "ok", Kind: "stub"}}
}
func (stub) Commands() []Command { return nil }
func (stub) Annotate(repo scan.Repo) []Annotation {
	return []Annotation{{Kind: "badge", Label: repo.Name}}
}
func (stub) Project(repo scan.Repo) []Widget {
	return []Widget{{ID: "stub.project", Title: "stub", Summary: repo.Name, Kind: "stub.project"}}
}

type stubRunner struct{ stub }

func (stubRunner) Run(_ context.Context, action string, repo scan.Repo, extra map[string]string) (Result, error) {
	return Result{OK: true, Action: action, Detail: repo.ID + ":" + extra["harness"]}, nil
}

func TestRegistryCollectsPluginOutput(t *testing.T) {
	var r Registry
	r.Register(stub{})
	if got := r.Widgets(); len(got) != 1 || got[0].ID != "stub" {
		t.Fatalf("widgets=%v", got)
	}
	notes := r.Annotate(scan.Repo{ID: "sessh", Name: "sessh"})
	if len(notes) != 1 || notes[0].Plugin != "stub" || notes[0].RepoID != "sessh" {
		t.Fatalf("notes=%v", notes)
	}
	panels := r.Project(scan.Repo{ID: "sessh", Name: "sessh"})
	if len(panels) != 1 || panels[0].Summary != "sessh" {
		t.Fatalf("project=%v", panels)
	}
}

func TestRegistryRunDispatchesRunner(t *testing.T) {
	var r Registry
	r.Register(stubRunner{})
	res, err := r.Run(context.Background(), "stub", "launch", scan.Repo{ID: "sessh"}, map[string]string{"harness": "claude"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK || res.Detail != "sessh:claude" {
		t.Fatalf("result=%v", res)
	}
	if _, err := r.Run(context.Background(), "missing", "launch", scan.Repo{}, nil); err == nil {
		t.Fatal("expected missing plugin error")
	}
}
