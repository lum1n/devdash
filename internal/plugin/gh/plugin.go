package gh

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/lum1n/devdash/internal/plugin"
	"github.com/lum1n/devdash/internal/scan"
)

const (
	pluginID     = "gh"
	refreshEvery = 45 * time.Second
	repoTTL      = 45 * time.Second
)

// Plugin surfaces GitHub PRs via the gh CLI.
type Plugin struct {
	src Source

	mu      sync.RWMutex
	snap    snapshot
	updated time.Time
	repoPRs map[string]repoCache
}

type repoCache struct {
	when time.Time
	prs  []pr
	err  string
}

// New talks to gh.
func New() *Plugin { return &Plugin{src: liveSource{}, repoPRs: map[string]repoCache{}} }

// Register adds the gh plugin to r.
func Register(r *plugin.Registry) { r.Register(New()) }

// ID implements plugin.Plugin.
func (p *Plugin) ID() string { return pluginID }

// Refresh searches open and review-requested PRs.
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
	review := 0
	for _, pr := range p.snap.prs {
		if pr.Review {
			review++
		}
	}
	data := OverviewData{Ready: p.snap.ready, Open: len(p.snap.prs), Review: review, Error: p.snap.err}
	return []plugin.Widget{{
		ID:      "gh",
		Title:   "gh",
		Summary: ghSummary(data),
		Kind:    "gh.overview",
		Data:    data,
	}}
}

// Commands lists palette actions.
func (p *Plugin) Commands() []plugin.Command {
	return []plugin.Command{{ID: "gh:open", Title: "open pull request"}}
}

// Annotate badges a repo with review or an open PR.
func (p *Plugin) Annotate(repo scan.Repo) []plugin.Annotation {
	p.mu.RLock()
	prs := p.searchFor(repo)
	p.mu.RUnlock()
	if len(prs) == 0 {
		return nil
	}
	for _, pr := range prs {
		if pr.Review {
			return []plugin.Annotation{{
				Kind:   "review",
				Label:  fmt.Sprintf("review #%d", pr.Number),
				Detail: pr.Title,
				Tone:   "danger",
			}}
		}
	}
	return []plugin.Annotation{{
		Kind:   "pr",
		Label:  fmt.Sprintf("pr #%d", prs[0].Number),
		Detail: prs[0].Title,
		Tone:   "warn",
	}}
}

// Project lists open PRs for the repo.
func (p *Plugin) Project(repo scan.Repo) []plugin.Widget {
	slug, _ := parseGitHubRemote(repo.Remote)
	prs := p.projectPRs(repo, slug.Slug())
	data := ProjectData{Ready: p.ready(), Slug: slug.Slug(), Pulls: toPulls(prs)}
	return []plugin.Widget{{
		ID:      "gh.project",
		Title:   "gh",
		Summary: projectSummary(data),
		Kind:    "gh.project",
		Data:    data,
	}}
}

// Run opens the first (or targeted) PR url.
func (p *Plugin) Run(_ context.Context, action string, repo scan.Repo, extra map[string]string) (plugin.Result, error) {
	if extra == nil {
		extra = map[string]string{}
	}
	switch action {
	case "open", "gh:open":
		url := strings.TrimSpace(firstNonEmpty(extra["target"], extra["url"]))
		if url == "" {
			slug, _ := parseGitHubRemote(repo.Remote)
			prs := p.projectPRs(repo, slug.Slug())
			if len(prs) > 0 {
				url = prs[0].URL
			}
		}
		if url == "" {
			return plugin.Result{}, fmt.Errorf("no pull request for %s", repo.ID)
		}
		detail, err := p.src.Open(url)
		if err != nil {
			return plugin.Result{}, err
		}
		return plugin.Result{OK: true, Action: "open", Detail: detail}, nil
	default:
		return plugin.Result{}, fmt.Errorf("unknown gh action %s", action)
	}
}

func (p *Plugin) ready() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.snap.ready
}

func (p *Plugin) searchFor(repo scan.Repo) []pr {
	var out []pr
	for _, pr := range p.snap.prs {
		slug := GitHubRepo{}
		if pr.Repository.NameWithOwner != "" {
			if parsed, ok := splitOwnerName(pr.Repository.NameWithOwner); ok {
				slug = parsed
			}
		}
		if slug.Name == "" {
			slug.Name = pr.Repository.Name
		}
		if matchesRepo(repo.ID, repo.Name, repo.Remote, slug) {
			out = append(out, pr)
		}
	}
	return out
}

func (p *Plugin) projectPRs(repo scan.Repo, slug string) []pr {
	p.mu.RLock()
	search := p.searchFor(repo)
	cached, ok := p.repoPRs[slug]
	fresh := ok && slug != "" && time.Since(cached.when) < repoTTL
	p.mu.RUnlock()
	if slug == "" {
		return search
	}
	if fresh {
		return mergeReview(cached.prs, search)
	}
	listed, err := p.src.RepoPRs(slug)
	p.mu.Lock()
	if err == nil {
		p.repoPRs[slug] = repoCache{when: time.Now(), prs: listed}
	} else {
		p.repoPRs[slug] = repoCache{when: time.Now(), prs: search, err: err.Error()}
		listed = search
	}
	p.mu.Unlock()
	return mergeReview(listed, search)
}

func mergeReview(listed, search []pr) []pr {
	review := map[int]bool{}
	for _, p := range search {
		if p.Review {
			review[p.Number] = true
		}
	}
	if len(listed) == 0 {
		return search
	}
	for i := range listed {
		if review[listed[i].Number] {
			listed[i].Review = true
		}
	}
	return listed
}

func toPulls(prs []pr) []Pull {
	out := make([]Pull, 0, len(prs))
	for _, p := range prs {
		checks := ""
		if checksFail(p) {
			checks = "fail"
		}
		out = append(out, Pull{
			Number: p.Number,
			Title:  p.Title,
			URL:    p.URL,
			Draft:  p.IsDraft,
			Review: p.Review,
			Checks: checks,
			Repo:   p.Repository.NameWithOwner,
		})
	}
	return out
}

func ghSummary(d OverviewData) string {
	if !d.Ready && d.Error != "" {
		return d.Error
	}
	if d.Open == 0 {
		return "no open prs"
	}
	if d.Review > 0 {
		return fmt.Sprintf("%d open  ·  %d review", d.Open, d.Review)
	}
	return fmt.Sprintf("%d open", d.Open)
}

func projectSummary(d ProjectData) string {
	if len(d.Pulls) == 0 {
		return "no open prs"
	}
	return fmt.Sprintf("%d open", len(d.Pulls))
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
