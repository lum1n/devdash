package tmux

// Session is one tmux session matched to a repo (or listed live).
type Session struct {
	Name     string   `json:"name"`
	Windows  []Window `json:"windows"`
	Attached bool     `json:"attached,omitempty"`
}

// Window is one window inside a session.
type Window struct {
	Index   int    `json:"index"`
	Name    string `json:"name"`
	Path    string `json:"path,omitempty"`
	Command string `json:"command,omitempty"`
}

// OverviewData is the now-strip payload.
type OverviewData struct {
	Ready    bool   `json:"ready"`
	Sessions int    `json:"sessions"`
	Error    string `json:"error,omitempty"`
}

// ProjectData is the project-panel payload.
type ProjectData struct {
	Ready    bool      `json:"ready"`
	Sessions []Session `json:"sessions"`
}
