package tmux

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/lum1n/devdash/internal/plugin"
	"github.com/lum1n/devdash/internal/repomatch"
	"github.com/lum1n/devdash/internal/scan"
)

const (
	pluginID     = "tmux"
	refreshEvery = 8 * time.Second
)

// Plugin lists tmux sessions and matches them to repos.
type Plugin struct {
	src Source

	mu      sync.RWMutex
	snap    snapshot
	updated time.Time
}

// New talks to the local tmux server.
func New() *Plugin { return &Plugin{src: liveSource{}} }

// Register adds the tmux plugin to r.
func Register(r *plugin.Registry) { r.Register(New()) }

// ID implements plugin.Plugin.
func (p *Plugin) ID() string { return pluginID }

// Refresh lists sessions.
func (p *Plugin) Refresh(ctx context.Context) error {
	_ = ctx
	p.mu.RLock()
	fresh := !p.updated.IsZero() && time.Since(p.updated) < refreshEvery
	p.mu.RUnlock()
	if fresh {
		return nil
	}
	snap, err := p.src.Load()
	p.mu.Lock()
	if err == nil {
		p.snap = snap
		p.updated = time.Now()
	} else if p.updated.IsZero() {
		p.snap.err = err.Error()
		p.updated = time.Now()
	}
	p.mu.Unlock()
	return err
}

// Widgets is the overview strip.
func (p *Plugin) Widgets() []plugin.Widget {
	p.mu.RLock()
	defer p.mu.RUnlock()
	data := OverviewData{Ready: p.snap.ready, Sessions: len(p.snap.sessions), Error: p.snap.err}
	return []plugin.Widget{{
		ID:      "tmux",
		Title:   "tmux",
		Summary: tmuxSummary(data),
		Kind:    "tmux.overview",
		Data:    data,
	}}
}

// Commands lists palette actions.
func (p *Plugin) Commands() []plugin.Command {
	return []plugin.Command{{ID: "tmux:attach", Title: "attach tmux"}}
}

// Annotate badges a repo that has a live session.
func (p *Plugin) Annotate(repo scan.Repo) []plugin.Annotation {
	p.mu.RLock()
	defer p.mu.RUnlock()
	sessions := p.sessionsFor(repo)
	if len(sessions) == 0 {
		return nil
	}
	return []plugin.Annotation{{
		Kind:   "tmux",
		Label:  "tmux",
		Detail: sessions[0].Name,
		Tone:   "ok",
	}}
}

// Project is the tmux panel on a repo screen.
func (p *Plugin) Project(repo scan.Repo) []plugin.Widget {
	p.mu.RLock()
	defer p.mu.RUnlock()
	data := ProjectData{Ready: p.snap.ready, Sessions: p.sessionsFor(repo)}
	return []plugin.Widget{{
		ID:      "tmux.project",
		Title:   "tmux",
		Summary: projectSummary(data),
		Kind:    "tmux.project",
		Data:    data,
	}}
}

// Run attaches (returns the attach command, same as acc).
func (p *Plugin) Run(_ context.Context, action string, repo scan.Repo, extra map[string]string) (plugin.Result, error) {
	if extra == nil {
		extra = map[string]string{}
	}
	switch action {
	case "attach", "tmux:attach":
		p.mu.RLock()
		sessions := p.sessionsFor(repo)
		p.mu.RUnlock()
		name := strings.TrimSpace(extra["target"])
		if name == "" && len(sessions) > 0 {
			name = sessions[0].Name
		}
		if name == "" {
			return plugin.Result{}, fmt.Errorf("no tmux session for %s", repo.ID)
		}
		detail, err := p.src.Attach(name)
		if err != nil {
			return plugin.Result{}, err
		}
		return plugin.Result{OK: true, Action: "attach", Detail: detail}, nil
	default:
		return plugin.Result{}, fmt.Errorf("unknown tmux action %s", action)
	}
}

func (p *Plugin) sessionsFor(repo scan.Repo) []Session {
	var out []Session
	for _, s := range p.snap.sessions {
		if sessionMatches(repo, s) {
			out = append(out, s)
		}
	}
	if out == nil {
		return []Session{}
	}
	return out
}

func sessionMatches(repo scan.Repo, s Session) bool {
	if repomatch.SessionHits(repo.Path, repo.ID, repo.Name, s.Name) {
		return true
	}
	for _, w := range s.Windows {
		if repomatch.SameTree(repo.Path, w.Path) {
			return true
		}
	}
	return false
}

func tmuxSummary(d OverviewData) string {
	if !d.Ready && d.Error != "" {
		return d.Error
	}
	if d.Sessions == 0 {
		return "no sessions"
	}
	return fmt.Sprintf("%d sessions", d.Sessions)
}

func projectSummary(d ProjectData) string {
	if len(d.Sessions) == 0 {
		return "no session"
	}
	if len(d.Sessions) == 1 {
		return d.Sessions[0].Name
	}
	return fmt.Sprintf("%d sessions", len(d.Sessions))
}
