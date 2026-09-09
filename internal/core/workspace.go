package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lum1n/devdash/internal/config"
	"github.com/lum1n/devdash/internal/remote"
)

// Workspaces is the local workspace list (never proxied).
func (a *App) Workspaces(ctx context.Context) []WorkspaceInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.workspaceInfosCtx(ctx)
}

func (a *App) workspaceInfos() []WorkspaceInfo {
	return a.workspaceInfosCtx(context.Background())
}

func (a *App) workspaceInfosCtx(ctx context.Context) []WorkspaceInfo {
	out := make([]WorkspaceInfo, 0, len(a.cfg.Workspaces))
	for _, w := range a.cfg.Workspaces {
		info := WorkspaceInfo{
			ID:     w.ID,
			Name:   w.DisplayName(),
			Kind:   w.Kind,
			Roots:  append([]string(nil), w.Roots...),
			Host:   w.Host,
			URL:    remote.NormalizeURL(w.URL),
			Active: w.ID == a.cfg.Active,
			Ready:  !w.IsSSH(),
		}
		if w.IsSSH() {
			base := a.remoteBaseLocked(w.ID, w.URL)
			info.URL = base
			if t := a.tunnels[w.ID]; t != nil && t.url != "" {
				info.Ready = true
			}
		}
		out = append(out, info)
	}
	return out
}

func (a *App) stampWorkspaces(ov *Overview) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	a.stampWorkspacesLocked(ov)
}

func (a *App) stampWorkspacesLocked(ov *Overview) {
	list := a.workspaceInfos()
	ov.Workspaces = list
	ov.Workspace = WorkspaceInfo{ID: a.cfg.Active, Name: a.cfg.ActiveWorkspace().DisplayName(), Kind: a.cfg.ActiveWorkspace().Kind, Active: true}
	for _, w := range list {
		if w.Active {
			ov.Workspace = w
			break
		}
	}
}

// SelectWorkspace switches the active workspace and rescans when local.
func (a *App) SelectWorkspace(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	a.mu.Lock()
	idx := -1
	for i, w := range a.cfg.Workspaces {
		if w.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		a.mu.Unlock()
		return fmt.Errorf("workspace %s not found", id)
	}
	ws := a.cfg.Workspaces[idx]
	a.cfg.Active = ws.ID
	a.cfg.NormalizeWorkspaces()
	a.repos = nil
	a.notes = nil
	a.scannedAt = time.Time{}
	path := a.cfgPath
	cfg := a.cfg
	a.mu.Unlock()
	if err := config.Save(path, cfg); err != nil {
		return err
	}
	if !ws.IsSSH() {
		return a.Scan(ctx)
	}
	return nil
}

// AddWorkspace appends a workspace and selects it.
func (a *App) AddWorkspace(ctx context.Context, w config.Workspace) error {
	if w.Name == "" && w.ID == "" {
		return fmt.Errorf("workspace name is required")
	}
	if w.ID == "" {
		w.ID = slugWorkspace(w.Name)
	}
	if w.Kind == "" {
		if w.Host != "" || w.URL != "" {
			w.Kind = config.KindSSH
		} else {
			w.Kind = config.KindLocal
		}
	}
	a.mu.Lock()
	for _, existing := range a.cfg.Workspaces {
		if existing.ID == w.ID {
			a.mu.Unlock()
			return fmt.Errorf("workspace %s already exists", w.ID)
		}
	}
	w.Focus.Normalize()
	a.cfg.Workspaces = append(a.cfg.Workspaces, w)
	a.mu.Unlock()
	return a.SelectWorkspace(ctx, w.ID)
}

func slugWorkspace(name string) string {
	return strings.TrimSpace(strings.ToLower(strings.ReplaceAll(name, " ", "-")))
}
