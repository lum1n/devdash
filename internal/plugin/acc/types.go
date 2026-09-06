package acc

// Fleet is live agent-watcher + skillcp health.
type Fleet struct {
	Connected bool   `json:"connected"`
	SkillcpOK bool   `json:"skillcp_ok"`
	Agents    int    `json:"agents"`
	Attention int    `json:"attention"`
	Error     string `json:"error,omitempty"`
}

// Plan is one subscription meter.
type Plan struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	UsedPct float64 `json:"used_pct"`
	Label   string  `json:"label,omitempty"`
	Health  string  `json:"health"`
	Detail  string  `json:"detail,omitempty"`
}

// Cost is a spend rollup plus sparkline points.
type Cost struct {
	Today float64   `json:"today"`
	Week  float64   `json:"week"`
	Month float64   `json:"month"`
	Spark []float64 `json:"spark"`
}

// Agent is one live pane matched to a repo.
type Agent struct {
	Session string `json:"session"`
	Kind    string `json:"kind"`
	State   string `json:"state"`
	Path    string `json:"path,omitempty"`
}

// Harness is a launchable agent CLI.
type Harness struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Ready bool   `json:"ready"`
}

// Session is a resumable harness session on this project.
type Session struct {
	Source    string `json:"source"`
	ID        string `json:"id"`
	Title     string `json:"title,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// OverviewData is the acc widget payload on the now-strip.
type OverviewData struct {
	Fleet Fleet  `json:"fleet"`
	Plans []Plan `json:"plans"`
	Cost  Cost   `json:"cost"`
}

// ProjectData is the acc panel on a project screen.
type ProjectData struct {
	Agents    []Agent   `json:"agents"`
	Cost30d   float64   `json:"cost_30d"`
	Harnesses []Harness `json:"harnesses"`
	Sessions  []Session `json:"sessions"`
}
