package core

import (
	"fmt"
	"time"

	"github.com/lum1n/devdash/internal/config"
)

// SetFocus updates pin / archive / snooze for a repo and persists config.
func (a *App) SetFocus(id, action string, hours int) error {
	if _, err := a.Repo(id); err != nil {
		return err
	}
	now := time.Now()
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cfg.Focus.Normalize()
	a.cfg.Focus.PruneSnoozes(now)
	switch action {
	case "pin":
		a.cfg.Focus.TogglePinned(id, true)
	case "unpin":
		a.cfg.Focus.TogglePinned(id, false)
	case "archive":
		a.cfg.Focus.ToggleArchived(id, true)
	case "unarchive":
		a.cfg.Focus.ToggleArchived(id, false)
	case "snooze":
		if hours <= 0 {
			hours = 24
		}
		a.cfg.Focus.Snooze(id, time.Duration(hours)*time.Hour, now)
	case "unsnooze":
		a.cfg.Focus.ClearSnooze(id)
	case "opened":
		a.cfg.Focus.TouchOpened(id, now)
	default:
		return fmt.Errorf("unknown focus action %s", action)
	}
	return config.Save(a.cfgPath, a.cfg)
}

// MarkOpened records that the operator visited a repo.
func (a *App) MarkOpened(id string) {
	_ = a.SetFocus(id, "opened", 0)
}

// SetNext stores the one-line next action for a repo.
func (a *App) SetNext(id, text string) error {
	if _, err := a.Repo(id); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cfg.Focus.SetNextAction(id, text)
	return config.Save(a.cfgPath, a.cfg)
}
