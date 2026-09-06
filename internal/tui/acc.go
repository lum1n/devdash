package tui

import (
	"fmt"
	"strings"

	"github.com/lum1n/devdash/internal/plugin"
	"github.com/lum1n/devdash/internal/plugin/acc"
)

func accOverview(widgets []plugin.Widget) acc.OverviewData {
	for _, w := range widgets {
		if w.Kind == "acc.overview" {
			if d, ok := w.Data.(acc.OverviewData); ok {
				return d
			}
		}
	}
	return acc.OverviewData{}
}

func accProject(widgets []plugin.Widget) (acc.ProjectData, bool) {
	for _, w := range widgets {
		if w.Kind == "acc.project" {
			if d, ok := w.Data.(acc.ProjectData); ok {
				return d, true
			}
		}
	}
	return acc.ProjectData{}, false
}

func accOverviewLines(widgets []plugin.Widget, width int) string {
	d := accOverview(widgets)
	if d.Fleet.Agents == 0 && len(d.Plans) == 0 && d.Cost.Month == 0 {
		if accHeader(widgets) == "" {
			return ""
		}
	}
	var lines []string
	lines = append(lines, dimStyle.Render("-- acc"))
	watch := "off"
	if d.Fleet.Connected {
		watch = "live"
	}
	lines = append(lines, fmt.Sprintf("fleet %d  attn %d  watcher %s  skillcp %v",
		d.Fleet.Agents, d.Fleet.Attention, watch, d.Fleet.SkillcpOK))
	for _, p := range d.Plans {
		bar := meterBar(p.UsedPct, 12)
		lab := fmt.Sprintf("%-10s %s %3.0f%%", truncateRunes(p.Name, 10), bar, p.UsedPct)
		if p.UsedPct >= 90 {
			lab = errStyle.Render(lab)
		} else if p.UsedPct >= 75 {
			lab = warnStyle.Render(lab)
		} else {
			lab = okStyle.Render(lab)
		}
		lines = append(lines, lab)
	}
	if d.Cost.Month > 0 || d.Cost.Today > 0 {
		lines = append(lines, fmt.Sprintf("spend  today $%.2f  week $%.2f  30d $%.2f  %s",
			d.Cost.Today, d.Cost.Week, d.Cost.Month, floatSpark(d.Cost.Spark)))
	}
	if d.Fleet.Error != "" {
		lines = append(lines, dimStyle.Render(d.Fleet.Error))
	}
	for i, line := range lines {
		lines[i] = truncateRunes(line, width)
	}
	return strings.Join(lines, "\n")
}

func accProjectBlock(data acc.ProjectData, harnessIdx, sessionIdx, width int) string {
	var lines []string
	if len(data.Agents) == 0 {
		lines = append(lines, dimStyle.Render("no live agents"))
	}
	for _, a := range data.Agents {
		lines = append(lines, fmt.Sprintf("%s %s %s", a.Kind, a.State, truncateRunes(a.Session, 16)))
	}
	if data.Cost30d > 0 {
		lines = append(lines, fmt.Sprintf("30d $%.2f", data.Cost30d))
	}
	if len(data.Harnesses) > 0 {
		var parts []string
		for i, h := range data.Harnesses {
			lab := h.Label
			if !h.Ready {
				lab += "?"
			}
			if i == harnessIdx {
				lab = "[" + lab + "]"
			}
			parts = append(parts, lab)
		}
		lines = append(lines, "launch "+strings.Join(parts, "  "))
	}
	if len(data.Sessions) > 0 {
		var parts []string
		for i, s := range data.Sessions {
			if i >= 4 {
				break
			}
			title := s.Title
			if title == "" {
				title = s.ID
			}
			lab := s.Source + " " + truncateRunes(title, 10)
			if i == sessionIdx {
				lab = "[" + lab + "]"
			}
			parts = append(parts, lab)
		}
		lines = append(lines, "resume "+strings.Join(parts, "  "))
	}
	for i, line := range lines {
		lines[i] = truncateRunes(line, width)
	}
	return strings.Join(lines, "\n")
}

func meterBar(pct float64, width int) string {
	if width < 4 {
		width = 4
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	fill := int(pct) * width / 100
	return strings.Repeat("█", fill) + strings.Repeat("░", width-fill)
}

func floatSpark(values []float64) string {
	if len(values) == 0 {
		return ""
	}
	max := 0.0
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	blocks := []rune("▁▂▃▄▅▆▇█")
	var b strings.Builder
	for _, v := range values {
		idx := 0
		if max > 0 && v > 0 {
			idx = int(v / max * float64(len(blocks)-1))
			if idx >= len(blocks) {
				idx = len(blocks) - 1
			}
		}
		b.WriteRune(blocks[idx])
	}
	return b.String()
}

func firstReadyHarness(data acc.ProjectData) int {
	for i, h := range data.Harnesses {
		if h.ID == "claude" && h.Ready {
			return i
		}
	}
	for i, h := range data.Harnesses {
		if h.Ready {
			return i
		}
	}
	return 0
}
