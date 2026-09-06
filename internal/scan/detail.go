package scan

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const activityWeeks = 16

// Detail is the project screen payload.
type Detail struct {
	Repo       Repo         `json:"repo"`
	Summary    string       `json:"summary"`
	Signals    []Signal     `json:"signals"`
	Activity   []WeekBucket `json:"activity"`
	Weekdays   [7]int       `json:"weekdays"`
	Commits    []Commit     `json:"commits"`
	Files      []FileChange `json:"files"`
	Authors    []Author     `json:"authors"`
	Shortcuts  []Shortcut   `json:"shortcuts"`
	RemoteURL  string       `json:"remote_url,omitempty"`
	CommitN16w int          `json:"commit_n_16w"`
}

// Signal is a status chip on the project screen.
type Signal struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Detail string `json:"detail,omitempty"`
	Tone   string `json:"tone"` // ok | warn | danger | muted
}

// WeekBucket is commits in one ISO-ish week (Monday start).
type WeekBucket struct {
	Start time.Time `json:"start"`
	Count int       `json:"count"`
}

// Commit is one recent log row.
type Commit struct {
	Hash    string    `json:"hash"`
	When    time.Time `json:"when"`
	Subject string    `json:"subject"`
	Author  string    `json:"author"`
}

// FileChange is one working-tree path from git status.
type FileChange struct {
	Path   string `json:"path"`
	Status string `json:"status"`
}

// Author is a short contributor rollup.
type Author struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// Shortcut is a project action the UI can run.
type Shortcut struct {
	ID      string `json:"id"`
	Key     string `json:"key"`
	Label   string `json:"label"`
	Enabled bool   `json:"enabled"`
}

// InspectDetail loads charts, signals, and working-tree facts for path.
func InspectDetail(ctx context.Context, root, path string) (Detail, error) {
	repo, err := inspect(ctx, root, path)
	if err != nil {
		return Detail{}, err
	}
	d := Detail{
		Repo:      repo,
		Summary:   readSummary(path),
		RemoteURL: RemoteWebURL(repo.Remote),
		Activity:  make([]WeekBucket, activityWeeks),
		Commits:   []Commit{},
		Files:     []FileChange{},
		Authors:   []Author{},
	}
	now := time.Now()
	start := weekStart(now).AddDate(0, 0, -7*(activityWeeks-1))
	for i := range d.Activity {
		d.Activity[i].Start = start.AddDate(0, 0, 7*i)
	}

	if dates, err := git(ctx, path, "log", "--since=16 weeks ago", "--format=%cI"); err == nil {
		for _, line := range nonEmpty(dates) {
			t, perr := time.Parse(time.RFC3339, strings.TrimSpace(line))
			if perr != nil {
				continue
			}
			d.CommitN16w++
			d.Weekdays[int(t.Weekday())]++
			ws := weekStart(t)
			idx := int(ws.Sub(start).Hours() / 24 / 7)
			if idx >= 0 && idx < activityWeeks {
				d.Activity[idx].Count++
			}
		}
	}

	if log, err := git(ctx, path, "log", "-20", "--format=%h\t%cI\t%an\t%s"); err == nil {
		for _, line := range nonEmpty(log) {
			parts := strings.SplitN(line, "\t", 4)
			if len(parts) < 4 {
				continue
			}
			c := Commit{Hash: parts[0], Author: parts[2], Subject: parts[3]}
			if t, perr := time.Parse(time.RFC3339, parts[1]); perr == nil {
				c.When = t
			}
			d.Commits = append(d.Commits, c)
		}
	}

	if status, err := git(ctx, path, "status", "--porcelain"); err == nil {
		for _, line := range nonEmpty(status) {
			if len(line) < 4 {
				continue
			}
			d.Files = append(d.Files, FileChange{
				Status: strings.TrimSpace(line[:2]),
				Path:   strings.TrimSpace(line[3:]),
			})
		}
	}

	if log, err := git(ctx, path, "log", "--since=30 days ago", "--format=%an"); err == nil {
		counts := map[string]int{}
		var order []string
		for _, name := range nonEmpty(log) {
			if counts[name] == 0 {
				order = append(order, name)
			}
			counts[name]++
		}
		for _, name := range order {
			d.Authors = append(d.Authors, Author{Name: name, Count: counts[name]})
		}
		for i := 0; i < len(d.Authors); i++ {
			for j := i + 1; j < len(d.Authors); j++ {
				if d.Authors[j].Count > d.Authors[i].Count {
					d.Authors[i], d.Authors[j] = d.Authors[j], d.Authors[i]
				}
			}
		}
		if len(d.Authors) > 5 {
			d.Authors = d.Authors[:5]
		}
	}

	d.Signals = signals(repo)
	d.Shortcuts = shortcuts(d)
	return d, nil
}

// RemoteWebURL turns a git remote into an https browse URL when possible.
func RemoteWebURL(remote string) string {
	s := strings.TrimSpace(remote)
	s = strings.TrimSuffix(s, ".git")
	switch {
	case strings.HasPrefix(s, "git@"):
		s = strings.TrimPrefix(s, "git@")
		host, path, ok := strings.Cut(s, ":")
		if ok {
			return "https://" + host + "/" + path
		}
	case strings.HasPrefix(s, "ssh://git@"):
		return "https://" + strings.TrimPrefix(s, "ssh://git@")
	}
	return s
}

func signals(r Repo) []Signal {
	var out []Signal
	if r.Dirty {
		out = append(out, Signal{ID: "dirty", Label: "dirty", Detail: itoa(r.Changed) + " files", Tone: "warn"})
	} else {
		out = append(out, Signal{ID: "clean", Label: "clean", Tone: "ok"})
	}
	if r.Ahead > 0 {
		out = append(out, Signal{ID: "ahead", Label: "ahead", Detail: itoa(r.Ahead), Tone: "warn"})
	}
	if r.Behind > 0 {
		out = append(out, Signal{ID: "behind", Label: "behind", Detail: itoa(r.Behind), Tone: "danger"})
	}
	if r.Stash > 0 {
		out = append(out, Signal{ID: "stash", Label: "stash", Detail: itoa(r.Stash), Tone: "warn"})
	}
	if r.Remote == "" {
		out = append(out, Signal{ID: "noremote", Label: "no remote", Tone: "muted"})
	}
	if r.Branch == "HEAD" || r.Branch == "" {
		out = append(out, Signal{ID: "detached", Label: "detached", Tone: "danger"})
	}
	if r.LastCommit.IsZero() || time.Since(r.LastCommit) > 14*24*time.Hour {
		out = append(out, Signal{ID: "stale", Label: "stale", Detail: "no commit in 14d", Tone: "muted"})
	}
	if r.Changed >= 20 {
		out = append(out, Signal{ID: "heavy", Label: "heavy dirty", Detail: itoa(r.Changed) + " paths", Tone: "danger"})
	}
	return out
}

func shortcuts(d Detail) []Shortcut {
	return []Shortcut{
		{ID: "editor", Key: "e", Label: "editor", Enabled: true},
		{ID: "term", Key: "t", Label: "tmux", Enabled: true},
		{ID: "url", Key: "g", Label: "remote", Enabled: d.RemoteURL != ""},
		{ID: "copy", Key: "c", Label: "copy path", Enabled: true},
	}
}

func weekStart(t time.Time) time.Time {
	t = t.Local()
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7
	}
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return d.AddDate(0, 0, -(wd - 1))
}

func readSummary(path string) string {
	for _, name := range []string{"README.md", "README", "readme.md"} {
		b, err := os.ReadFile(filepath.Join(path, name))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(line)
			line = strings.TrimLeft(line, "# ")
			if line != "" {
				if len(line) > 120 {
					return line[:117] + "…"
				}
				return line
			}
		}
	}
	return ""
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
