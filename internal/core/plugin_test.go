package core

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lum1n/devdash/internal/plugin"
	"github.com/lum1n/devdash/internal/scan"
)

type runStub struct{}

func (runStub) ID() string                             { return "acc" }
func (runStub) Widgets() []plugin.Widget               { return nil }
func (runStub) Commands() []plugin.Command             { return nil }
func (runStub) Annotate(scan.Repo) []plugin.Annotation { return nil }
func (runStub) Project(scan.Repo) []plugin.Widget      { return nil }
func (runStub) Run(_ context.Context, action string, repo scan.Repo, extra map[string]string) (plugin.Result, error) {
	return plugin.Result{OK: true, Action: action, Detail: repo.ID + ":" + extra["harness"]}, nil
}

func TestOpenDispatchesPluginAction(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("listen: 127.0.0.1:8789\nroots: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	reg := &plugin.Registry{}
	reg.Register(runStub{})
	app, err := New(cfgPath, reg)
	if err != nil {
		t.Fatal(err)
	}
	app.mu.Lock()
	app.repos = []scan.Repo{{ID: "sessh", Name: "sessh", Path: dir}}
	app.scannedAt = time.Now()
	app.mu.Unlock()

	res, err := app.Open(context.Background(), "sessh", "acc:claude")
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK || res.Detail != "sessh:claude" {
		t.Fatalf("open=%v", res)
	}
}
