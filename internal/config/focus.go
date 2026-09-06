package config

import (
	"slices"
	"strings"
	"time"
)

// Normalize fills maps and drops empty ids.
func (f *Focus) Normalize() {
	if f.Snoozed == nil {
		f.Snoozed = map[string]string{}
	}
	if f.Opened == nil {
		f.Opened = map[string]string{}
	}
	if f.Next == nil {
		f.Next = map[string]string{}
	}
	f.Pinned = compactIDs(f.Pinned)
	f.Archived = compactIDs(f.Archived)
}

// IsPinned reports whether id is pinned.
func (f Focus) IsPinned(id string) bool {
	return slices.Contains(f.Pinned, id)
}

// IsArchived reports whether id is archived.
func (f Focus) IsArchived(id string) bool {
	return slices.Contains(f.Archived, id)
}

// SnoozedUntil returns the snooze expiry when it is still in the future.
func (f Focus) SnoozedUntil(id string, now time.Time) (time.Time, bool) {
	raw := f.Snoozed[id]
	if raw == "" {
		return time.Time{}, false
	}
	until, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, false
	}
	if !until.After(now) {
		return time.Time{}, false
	}
	return until, true
}

// LastOpened returns the last-open time when known.
func (f Focus) LastOpened(id string) (time.Time, bool) {
	raw := f.Opened[id]
	if raw == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// TogglePinned adds or removes id from the pin list.
func (f *Focus) TogglePinned(id string, on bool) {
	f.Normalize()
	if on {
		f.Archived = removeID(f.Archived, id)
		if !f.IsPinned(id) {
			f.Pinned = append(f.Pinned, id)
		}
		return
	}
	f.Pinned = removeID(f.Pinned, id)
}

// ToggleArchived adds or removes id from the archive list.
func (f *Focus) ToggleArchived(id string, on bool) {
	f.Normalize()
	if on {
		f.Pinned = removeID(f.Pinned, id)
		if !f.IsArchived(id) {
			f.Archived = append(f.Archived, id)
		}
		return
	}
	f.Archived = removeID(f.Archived, id)
}

// Snooze hides id from Today until now+d.
func (f *Focus) Snooze(id string, d time.Duration, now time.Time) {
	f.Normalize()
	if d <= 0 {
		d = 24 * time.Hour
	}
	f.Snoozed[id] = now.Add(d).UTC().Format(time.RFC3339)
}

// ClearSnooze removes a snooze.
func (f *Focus) ClearSnooze(id string) {
	f.Normalize()
	delete(f.Snoozed, id)
}

// NextAction returns the one-line next action for id.
func (f Focus) NextAction(id string) string {
	return strings.TrimSpace(f.Next[id])
}

// SetNextAction stores or clears the one-line next action.
func (f *Focus) SetNextAction(id, text string) {
	f.Normalize()
	text = strings.TrimSpace(text)
	if text == "" {
		delete(f.Next, id)
		return
	}
	if len(text) > 160 {
		text = strings.TrimSpace(text[:160])
	}
	f.Next[id] = text
}

// TouchOpened records a visit.
func (f *Focus) TouchOpened(id string, now time.Time) {
	f.Normalize()
	f.Opened[id] = now.UTC().Format(time.RFC3339)
}

// PruneSnoozes drops expired entries.
func (f *Focus) PruneSnoozes(now time.Time) {
	f.Normalize()
	for id, raw := range f.Snoozed {
		until, err := time.Parse(time.RFC3339, raw)
		if err != nil || !until.After(now) {
			delete(f.Snoozed, id)
		}
	}
}

func compactIDs(in []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, id := range in {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func removeID(in []string, id string) []string {
	var out []string
	for _, v := range in {
		if v != id {
			out = append(out, v)
		}
	}
	return out
}
