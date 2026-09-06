package acc

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	acclib "github.com/lum1n/ai-command-center/pkg/acc"
	"github.com/lum1n/devdash/internal/plugin"
	"github.com/lum1n/devdash/internal/repomatch"
	"github.com/lum1n/devdash/internal/scan"
)

const (
	pluginID     = "acc"
	refreshEvery = 20 * time.Second
)

// Plugin wraps pkg/acc for the dashboard.
type Plugin struct {
	src     Source
	catalog []acclib.HarnessCatalog

	mu      sync.RWMutex
	snap    snapshot
	updated time.Time
}

// New loads ~/.config/acc and talks to the live acc stack.
func New() *Plugin {
	src := newLiveSource()
	return &Plugin{src: src, catalog: src.catalog}
}

// Register adds the acc plugin to r.
func Register(r *plugin.Registry) {
	r.Register(New())
}

// ID implements plugin.Plugin.
func (p *Plugin) ID() string { return pluginID }

// Refresh pulls fleet, usage, and plan meters.
func (p *Plugin) Refresh(ctx context.Context) error {
	p.mu.RLock()
	fresh := !p.updated.IsZero() && time.Since(p.updated) < refreshEvery
	p.mu.RUnlock()
	if fresh {
		return nil
	}
	snap, err := p.src.Load(ctx)
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

// Widgets is the overview now-strip.
func (p *Plugin) Widgets() []plugin.Widget {
	p.mu.RLock()
	defer p.mu.RUnlock()
	data := p.overview()
	return []plugin.Widget{{
		ID:      "acc",
		Title:   "acc",
		Summary: fleetSummary(data),
		Kind:    "acc.overview",
		Data:    data,
	}}
}

// Commands lists acc actions.
func (p *Plugin) Commands() []plugin.Command {
	return []plugin.Command{
		{ID: "acc:launch", Title: "launch agent", Keys: "a"},
		{ID: "acc:resume", Title: "resume last session"},
	}
}

// Annotate badges a repo that has live agents or spend.
func (p *Plugin) Annotate(repo scan.Repo) []plugin.Annotation {
	p.mu.RLock()
	defer p.mu.RUnlock()
	agents := p.agentsFor(repo)
	if len(agents) == 0 {
		return nil
	}
	attn := 0
	for _, a := range agents {
		if a.State == string(acclib.StateWaitingPermission) || a.State == string(acclib.StateErrored) {
			attn++
		}
	}
	tone := "ok"
	if attn > 0 {
		tone = "danger"
	} else if busy(agents[0].State) {
		tone = "warn"
	}
	return []plugin.Annotation{{
		Kind:   "agent",
		Label:  fmt.Sprintf("agent %s", agents[0].Kind),
		Detail: agents[0].State,
		Tone:   tone,
	}}
}

// Project is the acc panel on a repo screen.
func (p *Plugin) Project(repo scan.Repo) []plugin.Widget {
	p.mu.RLock()
	defer p.mu.RUnlock()
	data := p.project(repo)
	return []plugin.Widget{{
		ID:      "acc.project",
		Title:   "acc",
		Summary: projectSummary(data),
		Kind:    "acc.project",
		Data:    data,
	}}
}

// Run launches or resumes an agent in the repo.
func (p *Plugin) Run(ctx context.Context, action string, repo scan.Repo, extra map[string]string) (plugin.Result, error) {
	if extra == nil {
		extra = map[string]string{}
	}
	switch action {
	case "launch", "acc:launch":
		harness := extra["harness"]
		if harness == "" {
			harness = string(acclib.HarnessClaude)
		}
		detail, err := p.src.Launch(ctx, harness, repo.Path)
		if err != nil {
			return plugin.Result{}, err
		}
		return plugin.Result{OK: true, Action: "launch", Detail: detail}, nil
	case "resume", "acc:resume":
		source := extra["harness"]
		sessionID := extra["session"]
		if sessionID == "" {
			p.mu.RLock()
			for _, s := range p.snap.sessions {
				if repomatch.SameTree(repo.Path, s.Project) {
					source = string(s.Source)
					sessionID = s.ID
					break
				}
			}
			p.mu.RUnlock()
		}
		if sessionID == "" {
			return plugin.Result{}, fmt.Errorf("no session to resume")
		}
		detail, err := p.src.Resume(ctx, source, sessionID, repo.Path)
		if err != nil {
			return plugin.Result{}, err
		}
		return plugin.Result{OK: true, Action: "resume", Detail: detail}, nil
	default:
		return plugin.Result{}, fmt.Errorf("unknown acc action %s", action)
	}
}

func (p *Plugin) overview() OverviewData {
	d := p.snap.dash
	return OverviewData{
		Fleet: Fleet{
			Connected: d.Live.Connected,
			SkillcpOK: d.Inventory.SkillcpOK,
			Agents:    d.Counts.Total,
			Attention: d.Counts.Attention,
			Error:     firstNonEmpty(p.snap.err, d.Live.Error, d.Inventory.Error),
		},
		Plans: plansFrom(p.snap.plans),
		Cost: Cost{
			Today: p.snap.roll.Today.CostUSD,
			Week:  p.snap.roll.Week.CostUSD,
			Month: p.snap.roll.Totals.CostUSD,
			Spark: sparkFrom(p.snap.roll),
		},
	}
}

func (p *Plugin) project(repo scan.Repo) ProjectData {
	return ProjectData{
		Agents:    p.agentsFor(repo),
		Cost30d:   p.costFor(repo.Path),
		Harnesses: harnessesOnPATH(p.catalog),
		Sessions:  p.sessionsFor(repo.Path),
	}
}

func (p *Plugin) agentsFor(repo scan.Repo) []Agent {
	var out []Agent
	for _, a := range p.snap.dash.Live.Agents {
		if repomatch.MatchesRepo(repo.Path, repo.ID, repo.Name, a.Session, a.Path) {
			out = append(out, Agent{
				Session: a.Session,
				Kind:    string(a.Kind),
				State:   string(a.State),
				Path:    a.Path,
			})
		}
	}
	if out == nil {
		return []Agent{}
	}
	return out
}

func (p *Plugin) sessionsFor(path string) []Session {
	var out []Session
	for _, s := range p.snap.sessions {
		if !repomatch.SameTree(path, s.Project) {
			continue
		}
		when := ""
		if !s.UpdatedAt.IsZero() {
			when = s.UpdatedAt.UTC().Format(time.RFC3339)
		}
		out = append(out, Session{
			Source:    string(s.Source),
			ID:        s.ID,
			Title:     s.Title,
			UpdatedAt: when,
		})
	}
	if out == nil {
		return []Session{}
	}
	return out
}

func (p *Plugin) costFor(path string) float64 {
	var sum float64
	for _, row := range p.snap.roll.ByProject {
		if repomatch.SameTree(path, row.Key) {
			sum += row.Totals.CostUSD
		}
	}
	return sum
}

func fleetSummary(d OverviewData) string {
	parts := []string{fmt.Sprintf("%d agents", d.Fleet.Agents)}
	if d.Fleet.Attention > 0 {
		parts = append(parts, fmt.Sprintf("%d attn", d.Fleet.Attention))
	}
	parts = append(parts, fmtUSD(d.Cost.Month)+" 30d")
	if !d.Fleet.Connected {
		parts = append(parts, "watcher off")
	}
	return strings.Join(parts, "  ·  ")
}

func projectSummary(d ProjectData) string {
	parts := []string{fmt.Sprintf("%d agents", len(d.Agents))}
	if d.Cost30d > 0 {
		parts = append(parts, fmtUSD(d.Cost30d)+" 30d")
	}
	return strings.Join(parts, "  ·  ")
}

func busy(state string) bool {
	return state == string(acclib.StateThinking) || state == string(acclib.StateRunningTool)
}
