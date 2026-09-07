package core

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/lum1n/devdash/internal/notes"
	"github.com/lum1n/devdash/internal/plugin"
	"github.com/lum1n/devdash/internal/remote"
	"github.com/lum1n/devdash/internal/scan"
)

func (a *App) remote(ctx context.Context) (*remote.Client, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	a.mu.RLock()
	ws := a.cfg.ActiveWorkspace()
	a.mu.RUnlock()
	if !ws.IsSSH() {
		return nil, nil
	}
	if err := a.ensureTunnel(ctx, ws); err != nil {
		return nil, err
	}
	a.mu.RLock()
	base := a.remoteBaseLocked(ws.ID, ws.URL)
	a.mu.RUnlock()
	if base == "" {
		return nil, fmt.Errorf("workspace %s: set url or host", ws.ID)
	}
	return remote.New(base), nil
}

func (a *App) remoteBaseLocked(id, fallback string) string {
	if t := a.tunnels[id]; t != nil && t.url != "" {
		return t.url
	}
	return remote.NormalizeURL(fallback)
}

func remoteOverview(c *remote.Client, ctx context.Context) (Overview, error) {
	var ov Overview
	err := c.Do(ctx, http.MethodGet, "/api/overview", nil, &ov)
	return ov, err
}

func remoteScan(c *remote.Client, ctx context.Context) (Overview, error) {
	var ov Overview
	err := c.Do(ctx, http.MethodPost, "/api/scan", nil, &ov)
	return ov, err
}

func remoteDetail(c *remote.Client, ctx context.Context, id string) (Project, error) {
	var p Project
	err := c.Do(ctx, http.MethodGet, "/api/repos/"+url.PathEscape(id), nil, &p)
	return p, err
}

func remoteOpen(c *remote.Client, ctx context.Context, id, action string) (OpenResult, error) {
	var res OpenResult
	err := c.Do(ctx, http.MethodPost, "/api/repos/"+url.PathEscape(id)+"/open", map[string]string{"action": action}, &res)
	return res, err
}

func remoteSetFocus(c *remote.Client, ctx context.Context, id, action string, hours int) error {
	var ov Overview
	return c.Do(ctx, http.MethodPost, "/api/repos/"+url.PathEscape(id)+"/focus", map[string]any{
		"action": action, "hours": hours,
	}, &ov)
}

func remoteSetNext(c *remote.Client, ctx context.Context, id, text string) error {
	var ov Overview
	return c.Do(ctx, http.MethodPost, "/api/repos/"+url.PathEscape(id)+"/next", map[string]string{"text": text}, &ov)
}

func remoteDiff(c *remote.Client, ctx context.Context, id, ref, file string) (scan.Diff, error) {
	var d scan.Diff
	path := "/api/repos/" + url.PathEscape(id) + "/diff/" + url.PathEscape(ref)
	if file != "" {
		path += "?path=" + url.QueryEscape(file)
	}
	err := c.Do(ctx, http.MethodGet, path, nil, &d)
	return d, err
}

func remoteListNotes(c *remote.Client, ctx context.Context, id string) ([]notes.Meta, error) {
	var list []notes.Meta
	err := c.Do(ctx, http.MethodGet, "/api/repos/"+url.PathEscape(id)+"/notes", nil, &list)
	if list == nil {
		list = []notes.Meta{}
	}
	return list, err
}

func remoteCreateNote(c *remote.Client, ctx context.Context, id, title, content string) (notes.Note, error) {
	var n notes.Note
	err := c.Do(ctx, http.MethodPost, "/api/repos/"+url.PathEscape(id)+"/notes", map[string]string{
		"title": title, "content": content,
	}, &n)
	return n, err
}

func remoteReadNote(c *remote.Client, ctx context.Context, id, name string) (notes.Note, error) {
	var n notes.Note
	err := c.Do(ctx, http.MethodGet, "/api/repos/"+url.PathEscape(id)+"/notes/"+url.PathEscape(name), nil, &n)
	return n, err
}

func remoteWriteNote(c *remote.Client, ctx context.Context, id, name, content string) (notes.Note, error) {
	var n notes.Note
	err := c.Do(ctx, http.MethodPut, "/api/repos/"+url.PathEscape(id)+"/notes/"+url.PathEscape(name), map[string]string{
		"content": content,
	}, &n)
	return n, err
}

func remoteDeleteNote(c *remote.Client, ctx context.Context, id, name string) error {
	var out map[string]any
	return c.Do(ctx, http.MethodDelete, "/api/repos/"+url.PathEscape(id)+"/notes/"+url.PathEscape(name), nil, &out)
}

func remoteRunPlugin(c *remote.Client, ctx context.Context, pluginID, action, repo string, extra map[string]string) (plugin.Result, error) {
	body := map[string]string{"action": action, "repo": repo}
	for k, v := range extra {
		if v != "" {
			body[k] = v
		}
	}
	var res plugin.Result
	err := c.Do(ctx, http.MethodPost, "/api/plugins/"+url.PathEscape(pluginID)+"/run", body, &res)
	return res, err
}

func remoteAddRoot(c *remote.Client, ctx context.Context, path string) (Overview, error) {
	var ov Overview
	err := c.Do(ctx, http.MethodPost, "/api/roots", map[string]string{"path": path}, &ov)
	return ov, err
}
