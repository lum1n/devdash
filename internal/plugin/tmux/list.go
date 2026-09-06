package tmux

import (
	"bytes"
	"os/exec"
	"strconv"
	"strings"
)

type snapshot struct {
	ready    bool
	sessions []Session
	err      string
}

// Source lists live tmux sessions. Tests inject a fake.
type Source interface {
	Load() (snapshot, error)
	Attach(session string) (string, error)
}

type liveSource struct{}

func (liveSource) Load() (snapshot, error) {
	if _, err := exec.LookPath("tmux"); err != nil {
		return snapshot{err: "tmux not on PATH"}, nil
	}
	out, err := exec.Command("tmux", "list-panes", "-a", "-F",
		"#{session_name}\t#{window_index}\t#{window_name}\t#{pane_current_path}\t#{pane_current_command}\t#{session_attached}").Output()
	if err != nil {
		return snapshot{ready: true, err: "no tmux server"}, nil
	}
	return snapshot{ready: true, sessions: parsePanes(out)}, nil
}

func (liveSource) Attach(session string) (string, error) {
	session = strings.TrimSpace(session)
	if session == "" {
		return "", errMsg("no tmux session")
	}
	cmd := "tmux attach -t " + session
	if exec.Command("tmux", "has-session", "-t", session).Run() != nil {
		return "", errMsg("session " + session + " gone")
	}
	return cmd, nil
}

func parsePanes(raw []byte) []Session {
	order := []string{}
	byName := map[string]*Session{}
	for _, line := range bytes.Split(raw, []byte("\n")) {
		fields := strings.Split(string(line), "\t")
		if len(fields) < 4 || strings.TrimSpace(fields[0]) == "" {
			continue
		}
		name := fields[0]
		s, ok := byName[name]
		if !ok {
			s = &Session{Name: name, Windows: []Window{}}
			byName[name] = s
			order = append(order, name)
		}
		if len(fields) > 5 && fields[5] == "1" {
			s.Attached = true
		}
		idx, _ := strconv.Atoi(fields[1])
		win := Window{Index: idx, Name: fields[2], Path: fields[3]}
		if len(fields) > 4 {
			win.Command = fields[4]
		}
		dup := false
		for _, w := range s.Windows {
			if w.Index == win.Index {
				dup = true
				break
			}
		}
		if !dup {
			s.Windows = append(s.Windows, win)
		}
	}
	out := make([]Session, 0, len(order))
	for _, name := range order {
		out = append(out, *byName[name])
	}
	return out
}

type errMsg string

func (e errMsg) Error() string { return string(e) }
