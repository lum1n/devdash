package plugin

import (
	"context"

	"github.com/lum1n/devdash/internal/scan"
)

// Annotation is a per-repo badge from a plugin.
type Annotation struct {
	Plugin string `json:"plugin"`
	Kind   string `json:"kind"`
	Label  string `json:"label"`
	Detail string `json:"detail,omitempty"`
	Tone   string `json:"tone,omitempty"` // ok | warn | danger
	RepoID string `json:"repo_id,omitempty"`
}

// Widget is an overview or project section contributed by a plugin.
type Widget struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Kind    string `json:"kind,omitempty"`
	Data    any    `json:"data,omitempty"`
}

// Command is a palette / TUI action.
type Command struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Keys  string `json:"keys,omitempty"`
}

// Result is the outcome of a plugin action.
type Result struct {
	OK     bool   `json:"ok"`
	Action string `json:"action"`
	Detail string `json:"detail,omitempty"`
}

// Plugin extends the dashboard without changing core scan.
type Plugin interface {
	ID() string
	Widgets() []Widget
	Commands() []Command
	Annotate(repo scan.Repo) []Annotation
	Project(repo scan.Repo) []Widget
}

// Refresher is an optional plugin that pulls live data during Scan.
type Refresher interface {
	Refresh(ctx context.Context) error
}

// Runner is an optional plugin that handles actions (launch, resume).
type Runner interface {
	Run(ctx context.Context, action string, repo scan.Repo, extra map[string]string) (Result, error)
}

// Registry holds compile-time plugins.
type Registry struct {
	plugins []Plugin
}

// Register adds p. Later plugins are a new package + Register call.
func (r *Registry) Register(p Plugin) {
	if p == nil {
		return
	}
	r.plugins = append(r.plugins, p)
}

// All returns registered plugins in registration order.
func (r *Registry) All() []Plugin {
	return append([]Plugin(nil), r.plugins...)
}

// Widgets collects overview widgets.
func (r *Registry) Widgets() []Widget {
	var out []Widget
	for _, p := range r.plugins {
		out = append(out, p.Widgets()...)
	}
	return out
}

// Commands collects palette commands.
func (r *Registry) Commands() []Command {
	var out []Command
	for _, p := range r.plugins {
		out = append(out, p.Commands()...)
	}
	return out
}

// Annotate collects badges for one repo.
func (r *Registry) Annotate(repo scan.Repo) []Annotation {
	var out []Annotation
	for _, p := range r.plugins {
		for _, a := range p.Annotate(repo) {
			if a.Plugin == "" {
				a.Plugin = p.ID()
			}
			a.RepoID = repo.ID
			out = append(out, a)
		}
	}
	return out
}

// Project collects per-repo plugin panels.
func (r *Registry) Project(repo scan.Repo) []Widget {
	var out []Widget
	for _, p := range r.plugins {
		out = append(out, p.Project(repo)...)
	}
	return out
}

// Refresh asks plugins that implement Refresher to update caches.
func (r *Registry) Refresh(ctx context.Context) {
	for _, p := range r.plugins {
		if rf, ok := p.(Refresher); ok {
			_ = rf.Refresh(ctx)
		}
	}
}

// Run dispatches an action to the named Runner plugin.
func (r *Registry) Run(ctx context.Context, pluginID, action string, repo scan.Repo, extra map[string]string) (Result, error) {
	for _, p := range r.plugins {
		if p.ID() != pluginID {
			continue
		}
		rn, ok := p.(Runner)
		if !ok {
			return Result{}, errPlugin("plugin " + pluginID + " has no actions")
		}
		return rn.Run(ctx, action, repo, extra)
	}
	return Result{}, errPlugin("plugin " + pluginID + " not registered")
}

type errPlugin string

func (e errPlugin) Error() string { return string(e) }
