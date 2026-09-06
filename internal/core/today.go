package core

import (
	"sort"
	"strings"
	"time"

	"github.com/lum1n/devdash/internal/config"
	"github.com/lum1n/devdash/internal/plugin"
	"github.com/lum1n/devdash/internal/scan"
)

const todayLimit = 8

// Reason is why a repo is on Today.
type Reason struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Detail string `json:"detail,omitempty"`
	Tone   string `json:"tone"`
}

// TodayItem is one row on the focus board.
type TodayItem struct {
	Repo     scan.Repo `json:"repo"`
	Score    int       `json:"score"`
	Why      []Reason  `json:"why"`
	Next     string    `json:"next,omitempty"`
	Pinned   bool      `json:"pinned"`
	Snoozed  bool      `json:"snoozed"`
	OpenedAt time.Time `json:"opened_at,omitempty"`
}

// PaletteItem is one command-palette row.
type PaletteItem struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"` // jump | action | plugin
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`
	Keys     string `json:"keys,omitempty"`
	RepoID   string `json:"repo_id,omitempty"`
	Action   string `json:"action,omitempty"`
	Plugin   string `json:"plugin,omitempty"`
}

// FocusView is a compact copy of operator focus lists.
type FocusView struct {
	Pinned   []string `json:"pinned"`
	Archived []string `json:"archived"`
	Snoozed  []string `json:"snoozed"`
}

func buildToday(repos []scan.Repo, notes []plugin.Annotation, focus config.Focus, now time.Time) []TodayItem {
	byRepo := notesByRepo(notes)
	var items []TodayItem
	for _, r := range repos {
		if focus.IsArchived(r.ID) {
			continue
		}
		if _, snoozed := focus.SnoozedUntil(r.ID, now); snoozed {
			continue
		}
		item := scoreRepo(r, byRepo[r.ID], focus, now)
		if item.Score <= 0 && !item.Pinned {
			continue
		}
		items = append(items, item)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Score != items[j].Score {
			return items[i].Score > items[j].Score
		}
		if !items[i].OpenedAt.Equal(items[j].OpenedAt) {
			return items[i].OpenedAt.After(items[j].OpenedAt)
		}
		return items[i].Repo.LastCommit.After(items[j].Repo.LastCommit)
	})
	if len(items) > todayLimit {
		items = items[:todayLimit]
	}
	if items == nil {
		return []TodayItem{}
	}
	return items
}

func scoreRepo(r scan.Repo, notes []plugin.Annotation, focus config.Focus, now time.Time) TodayItem {
	item := TodayItem{Repo: r, Pinned: focus.IsPinned(r.ID), Why: []Reason{}}
	if opened, ok := focus.LastOpened(r.ID); ok {
		item.OpenedAt = opened
	}
	if next := focus.NextAction(r.ID); next != "" {
		item.Next = next
		item.Score += 25
		item.Why = append(item.Why, Reason{ID: "next", Label: "next", Detail: next, Tone: "ok"})
	}
	if item.Pinned {
		item.Score += 100
		item.Why = append(item.Why, Reason{ID: "pinned", Label: "pinned", Tone: "ok"})
	}
	for _, n := range notes {
		tone := n.Tone
		if tone == "" {
			tone = "warn"
		}
		pts := noteScore(n.Kind, tone)
		if pts == 0 {
			continue
		}
		item.Score += pts
		item.Why = append(item.Why, Reason{ID: n.Kind, Label: n.Label, Detail: n.Detail, Tone: tone})
	}
	if r.Behind > 0 {
		item.Score += 40 + min(r.Behind, 10)
		item.Why = append(item.Why, Reason{ID: "behind", Label: "behind", Detail: itoa(r.Behind), Tone: "danger"})
	}
	if r.Dirty {
		item.Score += 20 + min(r.Changed, 15)
		detail := "+" + itoa(r.Changed)
		if !r.LastCommit.IsZero() && now.Sub(r.LastCommit) > 3*24*time.Hour {
			item.Score += 15
			detail += " · stale dirty"
		}
		item.Why = append(item.Why, Reason{ID: "dirty", Label: "dirty", Detail: detail, Tone: "warn"})
	}
	if r.Stash > 0 {
		item.Score += 8
		item.Why = append(item.Why, Reason{ID: "stash", Label: "stash", Detail: itoa(r.Stash), Tone: "warn"})
	}
	if r.Ahead > 0 && r.Behind == 0 {
		item.Score += 8
		item.Why = append(item.Why, Reason{ID: "ahead", Label: "ahead", Detail: itoa(r.Ahead), Tone: "warn"})
	}
	if !item.OpenedAt.IsZero() {
		age := now.Sub(item.OpenedAt)
		if age < 24*time.Hour {
			item.Score += 20
			item.Why = append(item.Why, Reason{ID: "opened", Label: "opened", Detail: "today", Tone: "ok"})
		} else if age < 7*24*time.Hour {
			item.Score += 10
			item.Why = append(item.Why, Reason{ID: "opened", Label: "opened", Detail: "this week", Tone: "muted"})
		}
	}
	return item
}

func buildPalette(today []TodayItem, repos []scan.Repo, commands []plugin.Command, focus config.Focus, spaces []WorkspaceInfo) []PaletteItem {
	out := []PaletteItem{
		{ID: "rescan", Kind: "action", Title: "rescan", Keys: "r", Action: "rescan"},
		{ID: "add-root", Kind: "action", Title: "add root", Keys: "A", Action: "add-root"},
	}
	for _, ws := range spaces {
		if ws.Active {
			continue
		}
		out = append(out, PaletteItem{
			ID:       "ws:" + ws.ID,
			Kind:     "workspace",
			Title:    "workspace · " + ws.Name,
			Subtitle: ws.Kind,
			Action:   "workspace",
		})
	}
	seen := map[string]bool{}
	addJump := func(r scan.Repo, subtitle string) {
		if seen[r.ID] || focus.IsArchived(r.ID) {
			return
		}
		seen[r.ID] = true
		out = append(out, PaletteItem{
			ID:       "jump:" + r.ID,
			Kind:     "jump",
			Title:    r.Name,
			Subtitle: firstNonEmpty(subtitle, r.Stack, r.Branch),
			RepoID:   r.ID,
			Action:   "open",
		})
		out = append(out,
			PaletteItem{ID: "editor:" + r.ID, Kind: "action", Title: "editor · " + r.Name, Keys: "e", RepoID: r.ID, Action: "editor"},
			PaletteItem{ID: "term:" + r.ID, Kind: "action", Title: "tmux · " + r.Name, Keys: "t", RepoID: r.ID, Action: "term"},
			PaletteItem{ID: "launch:" + r.ID, Kind: "plugin", Title: "launch claude · " + r.Name, Keys: "a", RepoID: r.ID, Action: "launch", Plugin: "acc"},
		)
	}
	for _, t := range today {
		addJump(t.Repo, whySummary(t.Why))
	}
	for _, r := range repos {
		addJump(r, r.Stack)
	}
	for _, c := range commands {
		out = append(out, PaletteItem{
			ID:     c.ID,
			Kind:   "plugin",
			Title:  c.Title,
			Keys:   c.Keys,
			Action: c.ID,
			Plugin: pluginIDFromCommand(c.ID),
		})
	}
	return out
}

func focusView(focus config.Focus, now time.Time) FocusView {
	var snoozed []string
	for id := range focus.Snoozed {
		if _, ok := focus.SnoozedUntil(id, now); ok {
			snoozed = append(snoozed, id)
		}
	}
	sort.Strings(snoozed)
	pinned := append([]string(nil), focus.Pinned...)
	archived := append([]string(nil), focus.Archived...)
	if pinned == nil {
		pinned = []string{}
	}
	if archived == nil {
		archived = []string{}
	}
	if snoozed == nil {
		snoozed = []string{}
	}
	return FocusView{Pinned: pinned, Archived: archived, Snoozed: snoozed}
}

func notesByRepo(notes []plugin.Annotation) map[string][]plugin.Annotation {
	out := map[string][]plugin.Annotation{}
	for _, n := range notes {
		out[n.RepoID] = append(out[n.RepoID], n)
	}
	return out
}

func whySummary(why []Reason) string {
	for _, w := range why {
		if w.ID == "next" && w.Detail != "" {
			return w.Detail
		}
	}
	if len(why) == 0 {
		return ""
	}
	return why[0].Label
}

func noteScore(kind, tone string) int {
	if kind == "agent" {
		if tone == "danger" {
			return 55
		}
		return 35
	}
	switch tone {
	case "danger":
		return 45
	case "warn":
		return 22
	case "ok":
		return 12
	default:
		return 8
	}
}

func pluginIDFromCommand(id string) string {
	pluginID, _, ok := strings.Cut(id, ":")
	if !ok {
		return id
	}
	return pluginID
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	neg := n < 0
	if n < 0 {
		n = -n
	}
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
