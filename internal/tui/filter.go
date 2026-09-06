package tui

import (
	"strings"
	"time"

	"github.com/lum1n/devdash/internal/scan"
)

type repoKind int

const (
	kindAll repoKind = iota
	kindDirty
	kindStale
	kindGo
	kindTS
	kindArchived
)

func (k repoKind) label() string {
	return []string{"all", "dirty", "stale", "go", "ts", "archived"}[k]
}

func (k repoKind) next() repoKind {
	return (k + 1) % 6
}

func (m model) visible() []scan.Repo {
	archived := map[string]bool{}
	for _, id := range m.ov.Focus.Archived {
		archived[id] = true
	}
	q := strings.ToLower(m.filter)
	now := time.Now()
	var out []scan.Repo
	for _, r := range m.ov.Repos {
		isArchived := archived[r.ID]
		if m.kind == kindArchived {
			if !isArchived {
				continue
			}
		} else if isArchived {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(r.Name), q) && !strings.Contains(strings.ToLower(r.Stack), q) {
			continue
		}
		switch m.kind {
		case kindDirty:
			if !r.Dirty {
				continue
			}
		case kindStale:
			if !r.LastCommit.IsZero() && now.Sub(r.LastCommit) <= 14*24*time.Hour {
				continue
			}
		case kindGo, kindTS:
			if r.Stack != m.kind.label() {
				continue
			}
		}
		out = append(out, r)
	}
	return out
}

func (m model) pinAction(id string) string {
	for _, p := range m.ov.Focus.Pinned {
		if p == id {
			return "unpin"
		}
	}
	if m.mode == modeToday && m.idx >= 0 && m.idx < len(m.ov.Today) && m.ov.Today[m.idx].Repo.ID == id && m.ov.Today[m.idx].Pinned {
		return "unpin"
	}
	return "pin"
}

func (m model) archiveAction(id string) string {
	for _, a := range m.ov.Focus.Archived {
		if a == id {
			return "unarchive"
		}
	}
	return "archive"
}
