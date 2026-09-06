package ports

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lum1n/devdash/internal/plugin"
	"github.com/lum1n/devdash/internal/repomatch"
	"github.com/lum1n/devdash/internal/scan"
)

const (
	pluginID     = "ports"
	refreshEvery = 8 * time.Second
)

// Plugin lists listen sockets whose cwd is a repo.
type Plugin struct {
	src Source

	mu      sync.RWMutex
	snap    snapshot
	updated time.Time
}

// New talks to ss + /proc.
func New() *Plugin { return &Plugin{src: liveSource{}} }

// Register adds the ports plugin to r.
func Register(r *plugin.Registry) { r.Register(New()) }

// ID implements plugin.Plugin.
func (p *Plugin) ID() string { return pluginID }

// Refresh lists listeners.
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
	n := 0
	for _, l := range p.snap.listeners {
		if looksLikeProject(l.Cwd) {
			n++
		}
	}
	data := OverviewData{Ready: p.snap.ready, Listening: n, Error: p.snap.err}
	return []plugin.Widget{{
		ID:      "ports",
		Title:   "ports",
		Summary: portsSummary(data),
		Kind:    "ports.overview",
		Data:    data,
	}}
}

// Commands lists palette actions.
func (p *Plugin) Commands() []plugin.Command {
	return []plugin.Command{{ID: "ports:open", Title: "open listening url"}}
}

// Annotate badges a repo that is serving.
func (p *Plugin) Annotate(repo scan.Repo) []plugin.Annotation {
	p.mu.RLock()
	defer p.mu.RUnlock()
	ls := p.listenersFor(repo)
	if len(ls) == 0 {
		return nil
	}
	return []plugin.Annotation{{
		Kind:   "port",
		Label:  fmt.Sprintf("%s :%d", ls[0].Label, ls[0].Port),
		Detail: ls[0].URL,
		Tone:   "ok",
	}}
}

// Project is the ports panel on a repo screen.
func (p *Plugin) Project(repo scan.Repo) []plugin.Widget {
	p.mu.RLock()
	defer p.mu.RUnlock()
	data := ProjectData{Ready: p.snap.ready, Listeners: p.listenersFor(repo)}
	return []plugin.Widget{{
		ID:      "ports.project",
		Title:   "ports",
		Summary: projectSummary(data),
		Kind:    "ports.project",
		Data:    data,
	}}
}

// Run opens the first (or targeted) listen URL.
func (p *Plugin) Run(_ context.Context, action string, repo scan.Repo, extra map[string]string) (plugin.Result, error) {
	if extra == nil {
		extra = map[string]string{}
	}
	switch action {
	case "open", "ports:open":
		p.mu.RLock()
		ls := p.listenersFor(repo)
		p.mu.RUnlock()
		url := strings.TrimSpace(firstNonEmpty(extra["target"], extra["url"]))
		if url == "" && len(ls) > 0 {
			url = ls[0].URL
		}
		if url == "" {
			return plugin.Result{}, fmt.Errorf("no listening url for %s", repo.ID)
		}
		detail, err := p.src.Open(url)
		if err != nil {
			return plugin.Result{}, err
		}
		return plugin.Result{OK: true, Action: "open", Detail: detail}, nil
	default:
		return plugin.Result{}, fmt.Errorf("unknown ports action %s", action)
	}
}

func (p *Plugin) listenersFor(repo scan.Repo) []Listener {
	var out []Listener
	for _, l := range p.snap.listeners {
		if repomatch.SameTree(repo.Path, l.Cwd) {
			out = append(out, l)
		}
	}
	if out == nil {
		return []Listener{}
	}
	return out
}

func looksLikeProject(cwd string) bool {
	if cwd == "" || cwd == "/" {
		return false
	}
	for d := cwd; d != "/" && d != "."; d = parent(d) {
		if st, err := os.Stat(filepath.Join(d, ".git")); err == nil && (st.IsDir() || st.Mode().IsRegular()) {
			return true
		}
		next := parent(d)
		if next == d {
			break
		}
	}
	return false
}

func parent(p string) string {
	d := filepath.Dir(p)
	if d == "" {
		return "/"
	}
	return d
}

func portsSummary(d OverviewData) string {
	if !d.Ready && d.Error != "" {
		return d.Error
	}
	if d.Listening == 0 {
		return "none listening"
	}
	return fmt.Sprintf("%d listening", d.Listening)
}

func projectSummary(d ProjectData) string {
	if len(d.Listeners) == 0 {
		return "none listening"
	}
	return fmt.Sprintf("%s :%d", d.Listeners[0].Label, d.Listeners[0].Port)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
