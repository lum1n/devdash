package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/lum1n/devdash/internal/config"
	"github.com/lum1n/devdash/internal/plugin"
	"github.com/lum1n/devdash/internal/scan"
)

// Overview is the shared payload for web and TUI.
type Overview struct {
	Workspace  WorkspaceInfo       `json:"workspace"`
	Workspaces []WorkspaceInfo     `json:"workspaces"`
	Roots      []string            `json:"roots"`
	Repos      []scan.Repo         `json:"repos"`
	Counts     Counts              `json:"counts"`
	Today      []TodayItem         `json:"today"`
	Palette    []PaletteItem       `json:"palette"`
	Focus      FocusView           `json:"focus"`
	Widgets    []plugin.Widget     `json:"widgets"`
	Commands   []plugin.Command    `json:"commands"`
	Notes      []plugin.Annotation `json:"annotations"`
	ScannedAt  time.Time           `json:"scanned_at"`
	ConfigPath string              `json:"config_path"`
}

// WorkspaceInfo is one named scan context.
type WorkspaceInfo struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Kind   string   `json:"kind"`
	Roots  []string `json:"roots,omitempty"`
	Host   string   `json:"host,omitempty"`
	URL    string   `json:"url,omitempty"`
	Active bool     `json:"active"`
	Ready  bool     `json:"ready"`
}

// Counts is the now-strip.
type Counts struct {
	Repos     int `json:"repos"`
	Dirty     int `json:"dirty"`
	Behind    int `json:"behind"`
	Ahead     int `json:"ahead"`
	Attention int `json:"attention"`
	Today     int `json:"today"`
	Archived  int `json:"archived"`
}

// App is the in-process dashboard core.
type App struct {
	mu        sync.RWMutex
	cfg       config.Config
	cfgPath   string
	plugins   *plugin.Registry
	repos     []scan.Repo
	notes     []plugin.Annotation
	scannedAt time.Time
	tunnels   map[string]*sshTunnel
}

// New loads config and an empty plugin registry.
func New(cfgPath string, plugins *plugin.Registry) (*App, error) {
	if plugins == nil {
		plugins = &plugin.Registry{}
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, err
	}
	if cfgPath == "" {
		cfgPath, _ = config.Path()
	}
	return &App{cfg: cfg, cfgPath: cfgPath, plugins: plugins, tunnels: map[string]*sshTunnel{}}, nil
}

// Config returns a copy of the current config.
func (a *App) Config() config.Config {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cfg
}

// Scan walks configured roots and refreshes the cache.
func (a *App) Scan(ctx context.Context) error {
	c, err := a.remote(ctx)
	if err != nil {
		return err
	}
	if c != nil {
		ov, err := remoteScan(c, ctx)
		if err != nil {
			return err
		}
		a.mu.Lock()
		a.repos = ov.Repos
		a.notes = ov.Notes
		a.scannedAt = time.Now()
		a.mu.Unlock()
		return nil
	}
	a.mu.RLock()
	roots := a.cfg.ScanRoots()
	ignore := a.cfg.ScanIgnore()
	a.mu.RUnlock()

	a.plugins.Refresh(ctx)

	repos, err := scan.Roots(ctx, roots, scan.Options{Ignore: ignore})
	if err != nil {
		return err
	}
	sort.Slice(repos, func(i, j int) bool {
		if repos[i].Attention != repos[j].Attention {
			return repos[i].Attention
		}
		if repos[i].LastCommit.Equal(repos[j].LastCommit) {
			return repos[i].Name < repos[j].Name
		}
		return repos[i].LastCommit.After(repos[j].LastCommit)
	})

	var notes []plugin.Annotation
	for _, r := range repos {
		notes = append(notes, a.plugins.Annotate(r)...)
	}

	a.mu.Lock()
	a.repos = repos
	a.notes = notes
	a.scannedAt = time.Now()
	a.mu.Unlock()
	return nil
}

// Overview returns the last scan, scanning once if empty.
func (a *App) Overview(ctx context.Context) (Overview, error) {
	c, err := a.remote(ctx)
	if err != nil {
		return Overview{}, err
	}
	if c != nil {
		ov, err := remoteOverview(c, ctx)
		if err != nil {
			return Overview{}, err
		}
		a.stampWorkspaces(&ov)
		normalizeOverview(&ov)
		ov.ConfigPath = a.cfgPath
		return ov, nil
	}
	a.mu.RLock()
	empty := a.scannedAt.IsZero()
	a.mu.RUnlock()
	if empty {
		if err := a.Scan(ctx); err != nil {
			return Overview{}, err
		}
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	widgets := a.plugins.Widgets()
	if widgets == nil {
		widgets = []plugin.Widget{}
	}
	commands := a.plugins.Commands()
	if commands == nil {
		commands = []plugin.Command{}
	}
	notes := append([]plugin.Annotation(nil), a.notes...)
	if notes == nil {
		notes = []plugin.Annotation{}
	}
	now := time.Now()
	focus := a.cfg.ActiveWorkspace().Focus
	today := buildToday(a.repos, notes, focus, now)
	palette := buildPalette(today, a.repos, commands, focus, a.workspaceInfos())
	ov := Overview{
		Roots:     a.cfg.ScanRoots(),
		Repos:     append([]scan.Repo(nil), a.repos...),
		Counts:    count(a.repos, focus, len(today)),
		Today:     today,
		Palette:   palette,
		Focus:     focusView(focus, now),
		Widgets:   widgets,
		Commands:  commands,
		Notes:     notes,
		ScannedAt: a.scannedAt,
	}
	a.stampWorkspacesLocked(&ov)
	normalizeOverview(&ov)
	ov.ConfigPath = a.cfgPath
	return ov, nil
}

// ConfigPath is the yaml file this process loaded.
func (a *App) ConfigPath() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cfgPath
}

func normalizeOverview(ov *Overview) {
	ov.Workspaces = nz(ov.Workspaces)
	ov.Roots = nz(ov.Roots)
	ov.Repos = nz(ov.Repos)
	ov.Today = nz(ov.Today)
	ov.Palette = nz(ov.Palette)
	ov.Widgets = nz(ov.Widgets)
	ov.Commands = nz(ov.Commands)
	ov.Notes = nz(ov.Notes)
	ov.Focus.Pinned = nz(ov.Focus.Pinned)
	ov.Focus.Archived = nz(ov.Focus.Archived)
	ov.Focus.Snoozed = nz(ov.Focus.Snoozed)
}

func nz[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// AddRoot appends an existing directory and persists config.
func (a *App) AddRoot(path string) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	st, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("root %s: %w", path, err)
	}
	if !st.IsDir() {
		return fmt.Errorf("root %s is not a directory", path)
	}

	c, err := a.remote(context.Background())
	if err != nil {
		return err
	}
	if c != nil {
		_, err := remoteAddRoot(c, context.Background(), path)
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	ws := a.cfg.ActivePtr()
	if ws == nil {
		return fmt.Errorf("no active workspace")
	}
	if ws.IsSSH() {
		return fmt.Errorf("add a root on the remote host")
	}
	for _, r := range ws.Roots {
		if samePath(r, path) {
			return nil
		}
	}
	ws.Roots = append(ws.Roots, path)
	a.cfg.NormalizeWorkspaces()
	return config.Save(a.cfgPath, a.cfg)
}

func count(repos []scan.Repo, focus config.Focus, todayN int) Counts {
	c := Counts{Today: todayN}
	for _, r := range repos {
		if focus.IsArchived(r.ID) {
			c.Archived++
			continue
		}
		c.Repos++
		if r.Dirty {
			c.Dirty++
		}
		if r.Behind > 0 {
			c.Behind++
		}
		if r.Ahead > 0 {
			c.Ahead++
		}
		if r.Attention {
			c.Attention++
		}
	}
	return c
}

func samePath(a, b string) bool {
	aa, err1 := filepath.Abs(a)
	bb, err2 := filepath.Abs(b)
	if err1 != nil || err2 != nil {
		return a == b
	}
	return aa == bb
}
