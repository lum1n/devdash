package core

import (
	"context"
	"fmt"
	"time"

	"github.com/lum1n/devdash/internal/config"
)

// SetFocus updates pin / archive / snooze for a repo and persists config.
func (a *App) SetFocus(id, action string, hours int) error {
	if c, ok := a.remote(); ok {
		return remoteSetFocus(c, context.Background(), id, action, hours)
	}
	if _, err := a.Repo(id); err != nil {
		return err
	}
	now := time.Now()
	a.mu.Lock()
	defer a.mu.Unlock()
	f := a.cfg.FocusPtr()
	f.Normalize()
	f.PruneSnoozes(now)
	switch action {
	case "pin":
		f.TogglePinned(id, true)
	case "unpin":
		f.TogglePinned(id, false)
	case "archive":
		f.ToggleArchived(id, true)
	case "unarchive":
		f.ToggleArchived(id, false)
	case "snooze":
		if hours <= 0 {
			hours = 24
		}
		f.Snooze(id, time.Duration(hours)*time.Hour, now)
	case "unsnooze":
		f.ClearSnooze(id)
	case "opened":
		f.TouchOpened(id, now)
	default:
		return fmt.Errorf("unknown focus action %s", action)
	}
	a.cfg.NormalizeWorkspaces()
	return config.Save(a.cfgPath, a.cfg)
}

// MarkOpened records that the operator visited a repo.
func (a *App) MarkOpened(id string) {
	_ = a.SetFocus(id, "opened", 0)
}

// SetNext stores the one-line next action for a repo.
func (a *App) SetNext(id, text string) error {
	if c, ok := a.remote(); ok {
		return remoteSetNext(c, context.Background(), id, text)
	}
	if _, err := a.Repo(id); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cfg.FocusPtr().SetNextAction(id, text)
	a.cfg.NormalizeWorkspaces()
	return config.Save(a.cfgPath, a.cfg)
}
