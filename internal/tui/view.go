package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lum1n/devdash/internal/config"
	"github.com/lum1n/devdash/internal/core"
	"github.com/lum1n/devdash/internal/plugin"
	"github.com/lum1n/devdash/internal/plugin/acc"
	"github.com/lum1n/devdash/internal/scan"
)

func (m model) View() tea.View {
	var content string
	if m.width == 0 {
		content = "loading…"
	} else {
		header := m.viewHeader()
		tabs := m.viewTabs()
		footer := m.viewFooter()
		body := m.viewBody()
		content = m.fitFrame(header, tabs, body, footer)
	}
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m model) viewHeader() string {
	c := m.ov.Counts
	ws := m.ov.Workspace.Name
	if ws == "" {
		ws = m.ov.Workspace.ID
	}
	line := fmt.Sprintf("%s  %s  today %s  repos %s  dirty %s  behind %d  attn %s",
		titleStyle.Render("> devdash"),
		dimStyle.Render(ws),
		metricBig.Render(fmt.Sprintf("%d", c.Today)),
		metricBig.Render(fmt.Sprintf("%d", c.Repos)),
		metricBig.Render(fmt.Sprintf("%d", c.Dirty)),
		c.Behind,
		metricBig.Render(fmt.Sprintf("%d", c.Attention)),
	)
	if extra := pluginHeader(m.ov.Widgets); extra != "" {
		line += "  " + dimStyle.Render(extra)
	}
	if m.errMsg != "" {
		line += "  " + errStyle.Render(truncateRunes(m.errMsg, 40))
	}
	return truncateRunes(line, m.width)
}

func (m model) viewTabs() string {
	tabs := []struct {
		mode  viewMode
		label string
	}{
		{modeToday, "1 today"},
		{modeRepos, "2 repos"},
		{modeRoots, "3 ws"},
		{modeProject, "enter project"},
	}
	var parts []string
	for _, t := range tabs {
		if t.mode == m.mode {
			parts = append(parts, tabOn.Render(t.label))
		} else {
			parts = append(parts, tabOff.Render(t.label))
		}
	}
	return strings.Join(parts, " ")
}

func (m model) viewFooter() string {
	if m.adding {
		return helpStyle.Render("add root: " + m.addInput.View() + "  enter save  esc cancel")
	}
	if m.nexting {
		return helpStyle.Render("next: " + m.nextInput.View() + "  enter save  esc cancel")
	}
	if m.filtering {
		return helpStyle.Render("filter: " + m.filtInput.View() + "  enter apply  esc clear")
	}
	if m.noteTitle {
		return helpStyle.Render("title: " + m.titleInput.View() + "  enter create  esc cancel")
	}
	if m.noteEdit {
		dirty := ""
		if m.noteDirty {
			dirty = " unsaved"
		}
		return helpStyle.Render("note" + dirty + "  ctrl+s save  esc back  ^1-3 h  ^b ^i ^u ^k ^q ^l")
	}
	if m.notesMode {
		return helpStyle.Render("notes  j/k  enter edit  n new  D delete  esc back")
	}
	if m.palette {
		return helpStyle.Render("palette: " + m.palInput.View() + "  enter run  esc close")
	}
	help := "1-3 views  j/k  enter  e/t/a  p pin  s snooze  x archive  / filter  f kind  c copy  : palette  r  q"
	if m.mode == modeRepos {
		help = "2 repos  " + m.kind.label() + "  / filter  f cycle  j/k  enter  p/s/x  c copy  q"
	}
	if m.mode == modeProject {
		help = "esc back  n next  N notes  w/d slice  h/H acc  a launch  R resume  c/y copy  e/t/g  p/s/x  q"
	}
	if m.status != "" {
		help = m.status + "  ·  " + help
	}
	return helpStyle.Render(truncateRunes(help, m.width))
}

func (m model) viewBody() string {
	if m.palette {
		return m.viewPalette()
	}
	if m.noteEdit {
		return m.noteTA.View()
	}
	if m.notesMode {
		return m.viewNotes()
	}
	switch m.mode {
	case modeRoots:
		return m.viewRoots()
	case modeProject:
		return m.viewProject()
	case modeRepos:
		return m.viewRepos()
	default:
		return m.viewToday()
	}
}

func (m model) viewToday() string {
	if len(m.ov.Today) == 0 {
		empty := dimStyle.Render("// today is clear — pin with p, set next, or wait for dirty / behind / agents")
		if acc := accOverviewLines(m.ov.Widgets, m.width); acc != "" {
			return empty + "\n\n" + acc
		}
		return empty
	}
	var lines []string
	lines = append(lines, dimStyle.Render(fmt.Sprintf("%-22s %-28s %s", "name", "why", "score")))
	for i, item := range m.ov.Today {
		why := whyLabels(item.Why)
		mark := " "
		if item.Pinned {
			mark = "*"
		}
		plain := fmt.Sprintf("%s %-21s %-28s %d", mark, truncateRunes(item.Repo.Name, 21), truncateRunes(why, 28), item.Score)
		if i == m.idx {
			plain = selStyle.Render(truncateRunes(plain, m.width))
		}
		lines = append(lines, truncateRunes(plain, m.width))
	}
	if item, ok := m.todayAt(m.idx); ok {
		lines = append(lines, "")
		lines = append(lines, dimStyle.Render("-- "+item.Repo.Name))
		lines = append(lines, dimStyle.Render(item.Repo.Path))
		if item.Next != "" {
			lines = append(lines, okStyle.Render(item.Next))
		}
		if item.Repo.LastMessage != "" {
			lines = append(lines, item.Repo.LastMessage)
		}
	}
	if acc := accOverviewLines(m.ov.Widgets, m.width); acc != "" {
		lines = append(lines, "")
		lines = append(lines, acc)
	}
	return strings.Join(lines, "\n")
}

func (m model) viewNotes() string {
	var lines []string
	lines = append(lines, dimStyle.Render("-- notes  "+m.detail.Repo.Name+"  notes/"))
	if len(m.detail.Notes) == 0 {
		lines = append(lines, dimStyle.Render("// none — n to create"))
		return strings.Join(lines, "\n")
	}
	for i, n := range m.detail.Notes {
		line := fmt.Sprintf("%s  %s", n.Title, dimStyle.Render(n.Name))
		if i == m.noteIdx {
			line = selStyle.Render(truncateRunes(n.Title+"  "+n.Name, m.width))
		}
		lines = append(lines, truncateRunes(line, m.width))
	}
	return strings.Join(lines, "\n")
}

func (m model) viewPalette() string {
	items := m.paletteItems()
	if len(items) == 0 {
		return dimStyle.Render("// no matches")
	}
	var lines []string
	for i, item := range items {
		line := fmt.Sprintf("%-8s %s", item.Kind, item.Title)
		if item.Subtitle != "" {
			line += "  " + item.Subtitle
		}
		if i == m.palIdx {
			line = selStyle.Render(truncateRunes(line, m.width))
		}
		lines = append(lines, truncateRunes(line, m.width))
	}
	return strings.Join(lines, "\n")
}

func (m model) todayAt(i int) (core.TodayItem, bool) {
	if i >= 0 && i < len(m.ov.Today) {
		return m.ov.Today[i], true
	}
	return core.TodayItem{}, false
}

func whyLabels(why []core.Reason) string {
	var parts []string
	for _, w := range why {
		if w.ID == "next" && w.Detail != "" {
			parts = append(parts, w.Detail)
			continue
		}
		parts = append(parts, w.Label)
	}
	return strings.Join(parts, " · ")
}

func (m model) viewProject() string {
	d := m.detail
	r := d.Repo
	if r.Name == "" {
		return dimStyle.Render("// loading project")
	}
	var lines []string
	lines = append(lines, titleStyle.Render(r.Name)+"  "+dimStyle.Render(r.Stack+"  "+r.Branch))
	if d.Summary != "" {
		lines = append(lines, dimStyle.Render(d.Summary))
	}
	lines = append(lines, dimStyle.Render("-- next"))
	if d.Next != "" {
		lines = append(lines, okStyle.Render(d.Next))
	} else {
		lines = append(lines, dimStyle.Render("n to set"))
	}
	lines = append(lines, dimStyle.Render("-- notes  N to open"))
	if len(d.Notes) == 0 {
		lines = append(lines, dimStyle.Render("none"))
	}
	for i, n := range d.Notes {
		if i >= 6 {
			lines = append(lines, dimStyle.Render(fmt.Sprintf("  … %d more", len(d.Notes)-6)))
			break
		}
		lines = append(lines, "  "+n.Title)
	}
	var sigs []string
	for _, s := range d.Signals {
		lab := s.Label
		if s.Detail != "" {
			lab += " " + s.Detail
		}
		switch s.Tone {
		case "ok":
			sigs = append(sigs, okStyle.Render(lab))
		case "danger":
			sigs = append(sigs, errStyle.Render(lab))
		case "warn":
			sigs = append(sigs, warnStyle.Render(lab))
		default:
			sigs = append(sigs, dimStyle.Render(lab))
		}
	}
	lines = append(lines, dimStyle.Render("-- signals"))
	lines = append(lines, strings.Join(sigs, "  "))
	if data, ok := accProject(d.Plugins); ok {
		lines = append(lines, dimStyle.Render("-- acc  h harness  a launch  H session  R resume"))
		lines = append(lines, accProjectBlock(data, m.harnessIdx, m.sessionIdx, m.width))
	} else if accBody := accProjectLines(d.Plugins); accBody != "" {
		lines = append(lines, dimStyle.Render("-- acc"))
		lines = append(lines, accBody)
	}
	for _, w := range d.Plugins {
		if w.Kind == "acc.project" || !strings.HasSuffix(w.Kind, ".project") {
			continue
		}
		lines = append(lines, dimStyle.Render("-- "+w.Title))
		if w.Summary != "" {
			lines = append(lines, w.Summary)
		}
	}
	weekNote := ""
	if m.weekSel >= 0 && m.weekSel < len(d.Activity) {
		weekNote = fmt.Sprintf("  week %s", d.Activity[m.weekSel].Start.Format("01-02"))
	}
	dayNote := ""
	if m.daySel >= 0 {
		dayNote = "  " + []string{"su", "mo", "tu", "we", "th", "fr", "sa"}[m.daySel]
	}
	lines = append(lines, dimStyle.Render("-- activity 16w  "+fmt.Sprintf("%d commits", d.CommitN16w)+weekNote+dayNote+"  w/d"))
	lines = append(lines, sparkline(d.Activity))
	lines = append(lines, weekdayLine(d.Weekdays))
	if len(d.Files) > 0 {
		lines = append(lines, dimStyle.Render("-- working tree"))
		for i, f := range d.Files {
			if i >= 8 {
				lines = append(lines, dimStyle.Render(fmt.Sprintf("  … %d more", len(d.Files)-8)))
				break
			}
			lines = append(lines, fmt.Sprintf("  %-3s %s", f.Status, f.Path))
		}
	}
	commits := m.visibleCommits()
	if len(commits) > 0 {
		lines = append(lines, dimStyle.Render("-- commits  y copy hash"))
		for i, c := range commits {
			if i >= 8 {
				break
			}
			when := "—"
			if !c.When.IsZero() {
				when = c.When.Local().Format("01-02")
			}
			lines = append(lines, fmt.Sprintf("  %s  %-7s  %s", when, c.Hash, truncateRunes(c.Subject, 48)))
		}
	}
	return strings.Join(lines, "\n")
}

func pluginHeader(widgets []plugin.Widget) string {
	var parts []string
	for _, w := range widgets {
		if !strings.HasSuffix(w.Kind, ".overview") || w.Summary == "" {
			continue
		}
		parts = append(parts, w.Title+": "+w.Summary)
	}
	return strings.Join(parts, "  ·  ")
}

func accHeader(widgets []plugin.Widget) string {
	for _, w := range widgets {
		if w.Kind == "acc.overview" && w.Summary != "" {
			return w.Summary
		}
	}
	return ""
}

func accProjectLines(widgets []plugin.Widget) string {
	for _, w := range widgets {
		if w.Kind != "acc.project" {
			continue
		}
		data, ok := w.Data.(acc.ProjectData)
		if !ok {
			return w.Summary
		}
		var parts []string
		if len(data.Agents) == 0 {
			parts = append(parts, dimStyle.Render("no live agents"))
		}
		for _, a := range data.Agents {
			parts = append(parts, fmt.Sprintf("%s %s %s", a.Kind, a.State, a.Session))
		}
		if data.Cost30d > 0 {
			parts = append(parts, fmt.Sprintf("30d $%.2f", data.Cost30d))
		}
		if len(data.Sessions) > 0 {
			s := data.Sessions[0]
			parts = append(parts, fmt.Sprintf("resume %s %s", s.Source, truncateRunes(s.ID, 10)))
		}
		return strings.Join(parts, "  ·  ")
	}
	return ""
}

func sparkline(weeks []scan.WeekBucket) string {
	max := 1
	for _, w := range weeks {
		if w.Count > max {
			max = w.Count
		}
	}
	blocks := []rune("▁▂▃▄▅▆▇█")
	var b strings.Builder
	for _, w := range weeks {
		idx := 0
		if w.Count > 0 {
			idx = (w.Count * (len(blocks) - 1)) / max
			if idx >= len(blocks) {
				idx = len(blocks) - 1
			}
		}
		b.WriteRune(blocks[idx])
	}
	return okStyle.Render(b.String())
}

func weekdayLine(days [7]int) string {
	labels := []string{"su", "mo", "tu", "we", "th", "fr", "sa"}
	max := 1
	for _, n := range days {
		if n > max {
			max = n
		}
	}
	var parts []string
	for i, n := range days {
		fill := 0
		if n > 0 {
			fill = 1 + (n*4)/max
			if fill > 5 {
				fill = 5
			}
		}
		bar := strings.Repeat("█", fill) + strings.Repeat("░", 5-fill)
		parts = append(parts, fmt.Sprintf("%s %s", dimStyle.Render(labels[i]), bar))
	}
	return strings.Join(parts, "  ")
}

func (m model) viewRepos() string {
	rows := m.visible()
	if len(rows) == 0 {
		return dimStyle.Render("// no repos — add a root with A  / filter  f " + m.kind.label())
	}
	var lines []string
	lines = append(lines, dimStyle.Render(fmt.Sprintf("%-22s %-12s %-6s %-8s %s", "name", "branch", "stack", "status", "last"))+"  "+dimStyle.Render(m.kind.label()))
	for i, r := range rows {
		status := okStyle.Render("clean")
		if r.Dirty {
			status = warnStyle.Render(fmt.Sprintf("+%d", r.Changed))
		}
		if r.Behind > 0 {
			status += errStyle.Render(fmt.Sprintf(" ↓%d", r.Behind))
		}
		if r.Ahead > 0 {
			status += warnStyle.Render(fmt.Sprintf(" ↑%d", r.Ahead))
		}
		age := "—"
		if !r.LastCommit.IsZero() {
			age = r.LastCommit.Local().Format("01-02 15:04")
		}
		plain := fmt.Sprintf("%-22s %-12s %-6s", truncateRunes(r.Name, 22), truncateRunes(r.Branch, 12), r.Stack)
		row := plain + "  " + status + "  " + dimStyle.Render(age)
		if i == m.idx {
			row = selStyle.Render(truncateRunes(plain, m.width)) + "  " + status + "  " + dimStyle.Render(age)
		}
		lines = append(lines, truncateRunes(row, m.width))
	}
	if m.idx >= 0 && m.idx < len(rows) {
		sel := rows[m.idx]
		lines = append(lines, "")
		lines = append(lines, dimStyle.Render("-- "+sel.Name))
		lines = append(lines, dimStyle.Render(sel.Path))
		if sel.LastMessage != "" {
			lines = append(lines, sel.LastMessage)
		}
	}
	return strings.Join(lines, "\n")
}

func (m model) viewRoots() string {
	var lines []string
	lines = append(lines, dimStyle.Render("-- workspaces  W cycle"))
	if len(m.ov.Workspaces) == 0 {
		lines = append(lines, dimStyle.Render("// one local workspace from roots"))
	}
	for _, w := range m.ov.Workspaces {
		mark := " "
		if w.Active {
			mark = "*"
		}
		kind := w.Kind
		if kind == "" {
			kind = "local"
		}
		line := fmt.Sprintf("%s %-16s %-6s %s", mark, w.Name, kind, strings.Join(w.Roots, " "))
		if w.Host != "" {
			line += "  " + w.Host
		}
		lines = append(lines, truncateRunes(line, m.width))
	}
	lines = append(lines, dimStyle.Render("-- roots"))
	if len(m.ov.Roots) == 0 {
		lines = append(lines, dimStyle.Render("// none — A to add"))
	}
	for i, r := range m.ov.Roots {
		line := r
		if i == m.idx {
			line = selStyle.Render(truncateRunes(r, m.width))
		}
		lines = append(lines, truncateRunes(line, m.width))
	}
	f := m.ov.Focus
	lines = append(lines, dimStyle.Render("-- focus"))
	lines = append(lines, "pinned    "+joinOrDash(f.Pinned))
	lines = append(lines, "snoozed   "+joinOrDash(f.Snoozed))
	lines = append(lines, "archived  "+joinOrDash(f.Archived))
	if p, err := config.Path(); err == nil {
		lines = append(lines, dimStyle.Render(p))
	}
	return strings.Join(lines, "\n")
}

func joinOrDash(in []string) string {
	if len(in) == 0 {
		return "—"
	}
	return strings.Join(in, " · ")
}

func (m model) bodyHeight(header, tabs, footer string) int {
	used := lipgloss.Height(header) + lipgloss.Height(tabs) + lipgloss.Height(footer)
	h := m.height - used
	if h < 4 {
		h = 4
	}
	return h
}

func (m model) fitFrame(header, tabs, body, footer string) string {
	bodyH := m.bodyHeight(header, tabs, footer)
	body = clampLines(body, m.width, bodyH)
	frame := lipgloss.JoinVertical(lipgloss.Left, header, tabs, body, footer)
	lines := strings.Split(frame, "\n")
	for len(lines) < m.height {
		lines = append(lines, "")
	}
	if len(lines) > m.height {
		lines = lines[:m.height]
	}
	for i := range lines {
		lines[i] = truncateRunes(lines[i], m.width)
	}
	return strings.Join(lines, "\n")
}

func clampLines(s string, width, height int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	for i := range lines {
		lines[i] = truncateRunes(lines[i], width)
	}
	return strings.Join(lines, "\n")
}
