package ports

// Listener is one TCP listen socket whose cwd is a git project.
type Listener struct {
	PID     int    `json:"pid"`
	Port    int    `json:"port"`
	Addr    string `json:"addr"`
	Cwd     string `json:"cwd"`
	Command string `json:"command"`
	Label   string `json:"label"`
	URL     string `json:"url"`
}

// OverviewData is the now-strip payload.
type OverviewData struct {
	Ready     bool   `json:"ready"`
	Listening int    `json:"listening"`
	Error     string `json:"error,omitempty"`
}

// ProjectData is the project-panel payload.
type ProjectData struct {
	Ready     bool       `json:"ready"`
	Listeners []Listener `json:"listeners"`
}
