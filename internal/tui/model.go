package tui

import (
	"context"
	"strings"
	"time"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/atotto/clipboard"

	"github.com/lum1n/devdash/internal/core"
	"github.com/lum1n/devdash/internal/scan"
)

type viewMode int

const (
	modeToday viewMode = iota
	modeRepos
	modeRoots
	modeProject
)

type model struct {
	app        *core.App
	width      int
	height     int
	mode       viewMode
	ov         core.Overview
	idx        int
	filter     string
	kind       repoKind
	filtering  bool
	filtInput  textinput.Model
	status     string
	errMsg     string
	adding     bool
	addInput   textinput.Model
	nexting    bool
	nextInput  textinput.Model
	palette    bool
	palInput   textinput.Model
	palIdx     int
	detail     core.Project
	weekSel    int
	daySel     int
	harnessIdx int
	sessionIdx int
	notesMode  bool
	noteIdx    int
	noteEdit   bool
	noteTitle  bool
	noteDirty  bool
	noteName   string
	noteTA     textarea.Model
	titleInput textinput.Model
}

type overviewMsg struct {
	ov  core.Overview
	err error
}

type detailMsg struct {
	d   core.Project
	err error
}

type statusMsg string

// Run starts the Bubble Tea dashboard against the shared core.
func Run(app *core.App) error {
	p := tea.NewProgram(newModel(app))
	_, err := p.Run()
	return err
}

func newModel(app *core.App) model {
	ti := textinput.New()
	ti.Placeholder = "/path/to/repos"
	ti.CharLimit = 512
	ti.SetWidth(48)
	pal := textinput.New()
	pal.Placeholder = "jump · editor · tmux · rescan"
	pal.CharLimit = 128
	pal.SetWidth(48)
	ni := textinput.New()
	ni.Placeholder = "shipping iOS · blocked on API"
	ni.CharLimit = 160
	ni.SetWidth(48)
	fi := textinput.New()
	fi.Placeholder = "filter repos"
	fi.CharLimit = 64
	fi.SetWidth(32)
	return model{
		app:        app,
		status:     "scanning…",
		addInput:   ti,
		nextInput:  ni,
		palInput:   pal,
		filtInput:  fi,
		weekSel:    -1,
		daySel:     -1,
		noteTA:     newNoteEditor(),
		titleInput: newTitleInput(),
	}
}

func (m model) Init() tea.Cmd {
	return m.refresh(true)
}

func (m model) refresh(force bool) tea.Cmd {
	app := m.app
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if force {
			if err := app.Scan(ctx); err != nil {
				return overviewMsg{err: err}
			}
		}
		ov, err := app.Overview(ctx)
		return overviewMsg{ov: ov, err: err}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.width > 8 {
			m.noteTA.SetWidth(m.width - 2)
		}
		if m.height > 12 {
			m.noteTA.SetHeight(m.height - 10)
		}
		return m, nil
	case overviewMsg:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			m.status = "scan failed"
			return m, nil
		}
		m.ov = msg.ov
		m.errMsg = ""
		m.status = m.ov.ScannedAt.Local().Format("15:04:05")
		if n := m.rowCount(); m.idx >= n {
			m.idx = 0
		}
		return m, nil
	case detailMsg:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			m.status = "detail failed"
			return m, nil
		}
		m.detail = msg.d
		m.mode = modeProject
		m.status = m.detail.Repo.Name
		if data, ok := accProject(msg.d.Plugins); ok {
			m.harnessIdx = firstReadyHarness(data)
			m.sessionIdx = 0
		}
		return m, nil
	case noteLoadedMsg:
		m.noteEdit = true
		m.noteName = msg.n.Name
		m.noteDirty = false
		m.noteTA.SetValue(msg.n.Content)
		m.noteTA.Focus()
		m.status = msg.n.Name
		return m, nil
	case noteCreatedMsg:
		m.detail = msg.d
		m.noteTitle = false
		m.noteEdit = true
		m.noteName = msg.n.Name
		m.noteDirty = false
		m.noteTA.SetValue(msg.n.Content)
		m.noteTA.Focus()
		m.noteIdx = indexNote(msg.d.Notes, msg.n.Name)
		m.status = "created " + msg.n.Name
		return m, nil
	case noteSavedMsg:
		m.detail = msg.d
		m.noteDirty = false
		m.status = "saved " + m.noteName
		return m, nil
	case noteDeletedMsg:
		m.detail = msg.d
		m.noteEdit = false
		m.noteTA.Blur()
		if m.noteIdx >= len(m.detail.Notes) {
			m.noteIdx = 0
		}
		m.status = "deleted"
		return m, nil
	case statusMsg:
		m.status = string(msg)
		return m, nil
	case tea.KeyPressMsg:
		if m.adding {
			return m.updateAdd(msg)
		}
		if m.nexting {
			return m.updateNext(msg)
		}
		if m.filtering {
			return m.updateFilter(msg)
		}
		if m.palette {
			return m.updatePalette(msg)
		}
		if m.notesMode || m.noteEdit || m.noteTitle {
			return m.updateNotes(msg)
		}
		if m.mode == modeProject {
			return m.updateProject(msg)
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "1":
			m.mode = modeToday
		case "2":
			m.mode = modeRepos
		case "3":
			m.mode = modeRoots
		case "/":
			if m.mode == modeRepos {
				m.filtering = true
				m.filtInput.SetValue(m.filter)
				m.filtInput.Focus()
				m.status = "filter"
				return m, nil
			}
		case "f":
			if m.mode == modeRepos {
				m.kind = m.kind.next()
				m.idx = 0
				m.status = "filter " + m.kind.label()
			}
		case "c":
			if r, ok := m.selectedRepo(); ok {
				return m, copyText(r.Path)
			}
		case ":", "space", " ":
			m.palette = true
			m.palInput.SetValue("")
			m.palIdx = 0
			m.palInput.Focus()
			m.status = "palette"
			return m, nil
		case "r":
			m.status = "scanning…"
			return m, m.refresh(true)
		case "A":
			m.adding = true
			m.addInput.SetValue("")
			m.addInput.Focus()
			m.status = "add root"
		case "j", "down":
			m.move(1)
		case "k", "up":
			m.move(-1)
		case "enter":
			if r, ok := m.selectedRepo(); ok {
				m.status = "loading " + r.Name
				return m, m.loadDetail(r.ID)
			}
		case "e", "t", "a":
			if r, ok := m.selectedRepo(); ok {
				action := map[string]string{"e": "editor", "t": "term", "a": "acc:claude"}[msg.String()]
				return m, m.openCmd(r.ID, action)
			}
		case "p", "s", "x":
			if r, ok := m.selectedRepo(); ok {
				act := msg.String()
				if act == "p" {
					act = m.pinAction(r.ID)
				} else if act == "x" {
					act = m.archiveAction(r.ID)
				} else {
					act = "snooze"
				}
				return m, m.focusCmd(r.ID, act)
			}
		}
	}
	return m, nil
}

func (m model) updateAdd(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.adding = false
		m.addInput.Blur()
		m.status = "cancelled"
		return m, nil
	case "enter":
		path := strings.TrimSpace(m.addInput.Value())
		m.adding = false
		m.addInput.Blur()
		if path == "" {
			return m, nil
		}
		if err := m.app.AddRoot(path); err != nil {
			m.errMsg = err.Error()
			m.status = "add failed"
			return m, nil
		}
		m.status = "scanning…"
		return m, m.refresh(true)
	}
	var cmd tea.Cmd
	m.addInput, cmd = m.addInput.Update(msg)
	return m, cmd
}

func (m model) updateProject(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "backspace", "1":
		m.mode = modeRepos
		m.status = "repos"
		return m, nil
	case "r":
		return m, m.loadDetail(m.detail.Repo.ID)
	case ":", "space", " ":
		m.palette = true
		m.palInput.SetValue("")
		m.palIdx = 0
		m.palInput.Focus()
		m.status = "palette"
		return m, nil
	case "e", "t", "g":
		action := map[string]string{"e": "editor", "t": "term", "g": "url"}[msg.String()]
		return m, m.openCmd(m.detail.Repo.ID, action)
	case "c":
		return m, copyText(m.detail.Repo.Path)
	case "y":
		if commits := m.visibleCommits(); len(commits) > 0 {
			return m, copyText(commits[0].Hash)
		}
	case "a":
		return m, m.launchSelected()
	case "R":
		return m, m.resumeSelected()
	case "h":
		if data, ok := accProject(m.detail.Plugins); ok && len(data.Harnesses) > 0 {
			m.harnessIdx = (m.harnessIdx + 1) % len(data.Harnesses)
		}
		return m, nil
	case "H":
		if data, ok := accProject(m.detail.Plugins); ok && len(data.Sessions) > 0 {
			n := len(data.Sessions)
			if n > 4 {
				n = 4
			}
			m.sessionIdx = (m.sessionIdx + 1) % n
		}
		return m, nil
	case "p", "s", "x":
		id := m.detail.Repo.ID
		act := "snooze"
		if msg.String() == "p" {
			act = m.pinAction(id)
		} else if msg.String() == "x" {
			act = m.archiveAction(id)
		}
		return m, m.focusCmd(id, act)
	case "n":
		m.nexting = true
		m.nextInput.SetValue(m.detail.Next)
		m.nextInput.Focus()
		m.status = "next action"
		return m, nil
	case "N":
		m.notesMode = true
		m.status = "notes"
		return m, nil
	case "w":
		if n := len(m.detail.Activity); n > 0 {
			m.weekSel++
			if m.weekSel >= n {
				m.weekSel = -1
			}
		}
		return m, nil
	case "d":
		m.daySel++
		if m.daySel > 6 {
			m.daySel = -1
		}
		return m, nil
	}
	return m, nil
}

func (m model) updateFilter(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filtering = false
		m.filter = ""
		m.filtInput.SetValue("")
		m.filtInput.Blur()
		m.status = "repos"
		return m, nil
	case "enter":
		m.filtering = false
		m.filter = strings.TrimSpace(m.filtInput.Value())
		m.filtInput.Blur()
		m.idx = 0
		m.status = "filter " + m.kind.label()
		return m, nil
	}
	var cmd tea.Cmd
	m.filtInput, cmd = m.filtInput.Update(msg)
	m.filter = m.filtInput.Value()
	return m, cmd
}

func (m model) updateNext(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.nexting = false
		m.nextInput.Blur()
		m.status = "cancelled"
		return m, nil
	case "enter":
		text := strings.TrimSpace(m.nextInput.Value())
		m.nexting = false
		m.nextInput.Blur()
		id := m.detail.Repo.ID
		app := m.app
		return m, tea.Batch(
			func() tea.Msg {
				if err := app.SetNext(id, text); err != nil {
					return detailMsg{err: err}
				}
				d, err := app.Detail(context.Background(), id)
				return detailMsg{d: d, err: err}
			},
			m.refresh(false),
		)
	}
	var cmd tea.Cmd
	m.nextInput, cmd = m.nextInput.Update(msg)
	return m, cmd
}

func (m model) updatePalette(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.palette = false
		m.palInput.Blur()
		m.status = "today"
		return m, nil
	case "enter":
		items := m.paletteItems()
		m.palette = false
		m.palInput.Blur()
		if m.palIdx >= 0 && m.palIdx < len(items) {
			item := items[m.palIdx]
			if item.Action == "add-root" {
				m.adding = true
				m.addInput.SetValue("")
				m.addInput.Focus()
				m.status = "add root"
				return m, nil
			}
			return m, m.runPalette(item)
		}
		return m, nil
	case "down", "j", "ctrl+n":
		m.palIdx++
		if items := m.paletteItems(); m.palIdx >= len(items) {
			m.palIdx = 0
		}
		return m, nil
	case "up", "k", "ctrl+p":
		m.palIdx--
		if m.palIdx < 0 {
			if items := m.paletteItems(); len(items) > 0 {
				m.palIdx = len(items) - 1
			}
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.palInput, cmd = m.palInput.Update(msg)
	m.palIdx = 0
	return m, cmd
}

func (m model) runPalette(item core.PaletteItem) tea.Cmd {
	switch {
	case item.Action == "rescan":
		return m.refresh(true)
	case item.Kind == "jump" && item.RepoID != "":
		return m.loadDetail(item.RepoID)
	case item.Action == "editor" || item.Action == "term" || item.Action == "url":
		return m.openCmd(item.RepoID, item.Action)
	case item.Plugin != "":
		repo := item.RepoID
		if repo == "" && len(m.ov.Today) > 0 {
			repo = m.ov.Today[0].Repo.ID
		}
		if repo == "" {
			return func() tea.Msg { return statusMsg("pick a repo first") }
		}
		action := item.Action
		if action == "launch" || action == "acc:launch" {
			action = "acc:claude"
		}
		return m.openCmd(repo, action)
	default:
		return func() tea.Msg { return statusMsg(item.Title) }
	}
}

func (m model) launchSelected() tea.Cmd {
	harness := "claude"
	if data, ok := accProject(m.detail.Plugins); ok && m.harnessIdx >= 0 && m.harnessIdx < len(data.Harnesses) {
		harness = data.Harnesses[m.harnessIdx].ID
	}
	return m.pluginCmd(m.detail.Repo.ID, "launch", harness, "")
}

func (m model) resumeSelected() tea.Cmd {
	if data, ok := accProject(m.detail.Plugins); ok && m.sessionIdx >= 0 && m.sessionIdx < len(data.Sessions) {
		s := data.Sessions[m.sessionIdx]
		return m.pluginCmd(m.detail.Repo.ID, "resume", s.Source, s.ID)
	}
	return m.pluginCmd(m.detail.Repo.ID, "resume", "", "")
}

func (m model) pluginCmd(id, action, harness, session string) tea.Cmd {
	app := m.app
	return func() tea.Msg {
		res, err := app.RunPlugin(context.Background(), "acc", action, id, map[string]string{
			"harness": harness,
			"session": session,
		})
		if err != nil {
			return detailMsg{err: err}
		}
		return statusMsg(res.Detail)
	}
}

func copyText(s string) tea.Cmd {
	return func() tea.Msg {
		if s == "" {
			return statusMsg("nothing to copy")
		}
		if err := clipboard.WriteAll(s); err != nil {
			return statusMsg(s)
		}
		return statusMsg("copied " + truncateRunes(s, 40))
	}
}

func (m model) visibleCommits() []scan.Commit {
	list := m.detail.Commits
	if m.weekSel >= 0 && m.weekSel < len(m.detail.Activity) {
		start := m.detail.Activity[m.weekSel].Start
		end := start.Add(7 * 24 * time.Hour)
		var out []scan.Commit
		for _, c := range list {
			if !c.When.Before(start) && c.When.Before(end) {
				out = append(out, c)
			}
		}
		list = out
	}
	if m.daySel >= 0 {
		var out []scan.Commit
		for _, c := range list {
			if int(c.When.Weekday()) == m.daySel {
				out = append(out, c)
			}
		}
		list = out
	}
	return list
}

func (m model) openCmd(id, action string) tea.Cmd {
	return func() tea.Msg {
		res, err := m.app.Open(context.Background(), id, action)
		if err != nil {
			return detailMsg{err: err}
		}
		return statusMsg(res.Detail)
	}
}

func (m model) focusCmd(id, action string) tea.Cmd {
	app := m.app
	return func() tea.Msg {
		if err := app.SetFocus(id, action, 24); err != nil {
			return overviewMsg{err: err}
		}
		ov, err := app.Overview(context.Background())
		return overviewMsg{ov: ov, err: err}
	}
}

func (m model) loadDetail(id string) tea.Cmd {
	app := m.app
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		d, err := app.Detail(ctx, id)
		return detailMsg{d: d, err: err}
	}
}

func (m *model) move(delta int) {
	n := m.rowCount()
	if n == 0 {
		return
	}
	m.idx = (m.idx + delta + n) % n
}

func (m model) rowCount() int {
	if m.mode == modeToday {
		return len(m.ov.Today)
	}
	if m.mode == modeRoots {
		return len(m.ov.Roots)
	}
	return len(m.visible())
}

func (m model) selectedRepo() (scan.Repo, bool) {
	if m.mode == modeToday {
		if m.idx >= 0 && m.idx < len(m.ov.Today) {
			return m.ov.Today[m.idx].Repo, true
		}
		return scan.Repo{}, false
	}
	rows := m.visible()
	if m.idx >= 0 && m.idx < len(rows) {
		return rows[m.idx], true
	}
	return scan.Repo{}, false
}

func (m model) paletteItems() []core.PaletteItem {
	q := strings.ToLower(strings.TrimSpace(m.palInput.Value()))
	var out []core.PaletteItem
	for _, item := range m.ov.Palette {
		if q == "" || strings.Contains(strings.ToLower(item.Title+" "+item.Subtitle+" "+item.RepoID), q) {
			out = append(out, item)
		}
		if len(out) >= 12 {
			break
		}
	}
	return out
}

func truncateRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= n {
		return s
	}
	var b strings.Builder
	w := 0
	for _, r := range s {
		rw := 1
		if r == '\t' {
			rw = 2
		}
		if w+rw >= n {
			b.WriteRune('…')
			break
		}
		b.WriteRune(r)
		w += rw
	}
	return b.String()
}
