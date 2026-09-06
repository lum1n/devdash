package gh

// Pull is one open PR shown on a project panel.
type Pull struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	Draft  bool   `json:"draft"`
	Review bool   `json:"review"`
	Checks string `json:"checks,omitempty"`
	Repo   string `json:"repo,omitempty"`
}

// OverviewData is the now-strip payload.
type OverviewData struct {
	Ready  bool   `json:"ready"`
	Open   int    `json:"open"`
	Review int    `json:"review"`
	Error  string `json:"error,omitempty"`
}

// ProjectData is the project-panel payload.
type ProjectData struct {
	Ready bool   `json:"ready"`
	Slug  string `json:"slug,omitempty"`
	Pulls []Pull `json:"pulls"`
}
