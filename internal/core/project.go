package core

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/lum1n/devdash/internal/notes"
	"github.com/lum1n/devdash/internal/plugin"
	"github.com/lum1n/devdash/internal/scan"
)

// OpenResult is the outcome of a project shortcut.
type OpenResult struct {
	OK     bool   `json:"ok"`
	Action string `json:"action"`
	Detail string `json:"detail,omitempty"`
}

// Repo finds a scanned repo by id or name.
func (a *App) Repo(id string) (scan.Repo, error) {
	id = cleanRepoID(id)
	if err := a.ensureScanned(context.Background()); err != nil {
		return scan.Repo{}, err
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	for _, r := range a.repos {
		if r.ID == id || r.Name == id {
			return r, nil
		}
	}
	return scan.Repo{}, fmt.Errorf("repo %s not found", id)
}

// Project is the project screen payload (git detail + plugin panels).
type Project struct {
	scan.Detail
	Plugins  []plugin.Widget `json:"plugins"`
	Next     string          `json:"next"`
	Notes    []notes.Meta    `json:"notes"`
	Archived bool            `json:"archived"`
}

// Detail loads the project screen for id.
func (a *App) Detail(ctx context.Context, id string) (Project, error) {
	if c, ok := a.remote(); ok {
		return remoteDetail(c, ctx, id)
	}
	repo, err := a.Repo(id)
	if err != nil {
		return Project{}, err
	}
	a.plugins.Refresh(ctx)
	d, err := scan.InspectDetail(ctx, repo.Root, repo.Path)
	if err != nil {
		return Project{}, err
	}
	d.Signals = append(d.Signals, pluginSignals(a.plugins.Annotate(d.Repo))...)
	panels := a.plugins.Project(d.Repo)
	if panels == nil {
		panels = []plugin.Widget{}
	}
	a.MarkOpened(d.Repo.ID)
	noteList, err := notes.List(d.Repo.Path)
	if err != nil {
		noteList = []notes.Meta{}
	}
	cfg := a.Config()
	focus := cfg.ActiveWorkspace().Focus
	return Project{
		Detail:   d,
		Plugins:  panels,
		Next:     focus.NextAction(d.Repo.ID),
		Notes:    noteList,
		Archived: focus.IsArchived(d.Repo.ID),
	}, nil
}

// RunPlugin dispatches a plugin action against a scanned repo.
func (a *App) RunPlugin(ctx context.Context, pluginID, action, repoID string, extra map[string]string) (plugin.Result, error) {
	if c, ok := a.remote(); ok {
		return remoteRunPlugin(c, ctx, pluginID, action, repoID, extra)
	}
	repo, err := a.Repo(repoID)
	if err != nil {
		return plugin.Result{}, err
	}
	return a.plugins.Run(ctx, pluginID, action, repo, extra)
}

// ListNotes lists markdown files in the repo notes/ directory.
func (a *App) ListNotes(id string) ([]notes.Meta, error) {
	if c, ok := a.remote(); ok {
		return remoteListNotes(c, context.Background(), id)
	}
	repo, err := a.Repo(id)
	if err != nil {
		return nil, err
	}
	return notes.List(repo.Path)
}

// ReadNote loads one markdown note.
func (a *App) ReadNote(id, name string) (notes.Note, error) {
	if c, ok := a.remote(); ok {
		return remoteReadNote(c, context.Background(), id, name)
	}
	repo, err := a.Repo(id)
	if err != nil {
		return notes.Note{}, err
	}
	return notes.Read(repo.Path, name)
}

// CreateNote adds a note under repo/notes.
func (a *App) CreateNote(id, title, content string) (notes.Note, error) {
	if c, ok := a.remote(); ok {
		return remoteCreateNote(c, context.Background(), id, title, content)
	}
	repo, err := a.Repo(id)
	if err != nil {
		return notes.Note{}, err
	}
	return notes.Create(repo.Path, title, content)
}

// WriteNote updates a note's markdown.
func (a *App) WriteNote(id, name, content string) (notes.Note, error) {
	if c, ok := a.remote(); ok {
		return remoteWriteNote(c, context.Background(), id, name, content)
	}
	repo, err := a.Repo(id)
	if err != nil {
		return notes.Note{}, err
	}
	return notes.Write(repo.Path, name, content)
}

// Diff loads a commit patch, or one working-tree file.
func (a *App) Diff(ctx context.Context, id, ref, file string) (scan.Diff, error) {
	if c, ok := a.remote(); ok {
		return remoteDiff(c, ctx, id, ref, file)
	}
	repo, err := a.Repo(id)
	if err != nil {
		return scan.Diff{}, err
	}
	return scan.InspectDiff(ctx, repo.Path, ref, file)
}

// DeleteNote removes a note file.
func (a *App) DeleteNote(id, name string) error {
	if c, ok := a.remote(); ok {
		return remoteDeleteNote(c, context.Background(), id, name)
	}
	repo, err := a.Repo(id)
	if err != nil {
		return err
	}
	return notes.Delete(repo.Path, name)
}

// Open runs a local shortcut for the project.
func (a *App) Open(ctx context.Context, id, action string) (OpenResult, error) {
	if c, ok := a.remote(); ok {
		return remoteOpen(c, ctx, id, action)
	}
	if res, err, ok := a.openPlugin(ctx, id, action); ok {
		return res, err
	}
	d, err := a.Detail(ctx, id)
	if err != nil {
		return OpenResult{}, err
	}
	cfg := a.Config()
	switch action {
	case "copy":
		return OpenResult{OK: true, Action: action, Detail: d.Repo.Path}, nil
	case "url":
		if d.RemoteURL == "" {
			return OpenResult{}, fmt.Errorf("no remote url")
		}
		if err := start(openURLCmd(d.RemoteURL)); err != nil {
			return OpenResult{}, err
		}
		a.MarkOpened(d.Repo.ID)
		return OpenResult{OK: true, Action: action, Detail: d.RemoteURL}, nil
	case "editor":
		bin := firstNonEmpty(cfg.Actions.Editor, os.Getenv("VISUAL"), os.Getenv("EDITOR"), "xdg-open")
		if err := start(exec.Command(bin, d.Repo.Path)); err != nil {
			return OpenResult{}, err
		}
		a.MarkOpened(d.Repo.ID)
		return OpenResult{OK: true, Action: action, Detail: bin + " " + d.Repo.Path}, nil
	case "term":
		cmd, detail := termCmd(cfg.Actions.Term, d.Repo.Path)
		if err := start(cmd); err != nil {
			return OpenResult{}, err
		}
		a.MarkOpened(d.Repo.ID)
		return OpenResult{OK: true, Action: action, Detail: detail}, nil
	default:
		return OpenResult{}, fmt.Errorf("unknown action %s", action)
	}
}

func (a *App) openPlugin(ctx context.Context, id, action string) (OpenResult, error, bool) {
	pluginID, rest, ok := strings.Cut(action, ":")
	if !ok || pluginID == "" || rest == "" {
		return OpenResult{}, nil, false
	}
	extra := map[string]string{}
	act := rest
	if pluginID == "acc" && rest != "launch" && rest != "resume" {
		extra["harness"] = rest
		act = "launch"
	}
	res, err := a.RunPlugin(ctx, pluginID, act, id, extra)
	if err == nil {
		a.MarkOpened(id)
	}
	return OpenResult{OK: res.OK, Action: res.Action, Detail: res.Detail}, err, true
}

func (a *App) ensureScanned(ctx context.Context) error {
	a.mu.RLock()
	empty := a.scannedAt.IsZero()
	a.mu.RUnlock()
	if empty {
		return a.Scan(ctx)
	}
	return nil
}

func pluginSignals(notes []plugin.Annotation) []scan.Signal {
	var out []scan.Signal
	for _, n := range notes {
		tone := n.Tone
		if tone == "" {
			tone = "muted"
		}
		out = append(out, scan.Signal{
			ID:     n.Plugin + ":" + n.Kind,
			Label:  n.Label,
			Detail: n.Detail,
			Tone:   tone,
		})
	}
	return out
}

func termCmd(configured, dir string) (*exec.Cmd, string) {
	if configured != "" {
		cmd := exec.Command(configured, dir)
		cmd.Dir = dir
		return cmd, configured
	}
	if _, err := exec.LookPath("tmux"); err == nil {
		return exec.Command("tmux", "new-window", "-c", dir), "tmux new-window"
	}
	return exec.Command("xdg-open", dir), "xdg-open"
}

func openURLCmd(url string) *exec.Cmd {
	if _, err := exec.LookPath("xdg-open"); err == nil {
		return exec.Command("xdg-open", url)
	}
	return exec.Command("open", url)
}

func start(cmd *exec.Cmd) error {
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}

func cleanRepoID(id string) string {
	id = strings.TrimSpace(id)
	cut := func(r rune) bool {
		switch r {
		case '"', '\'', '`', '/', '\\', ' ', '\t', '\n':
			return true
		default:
			return false
		}
	}
	for _, part := range strings.FieldsFunc(id, cut) {
		return part
	}
	return ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
