package tui

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/lum1n/devdash/internal/core"
	"github.com/lum1n/devdash/internal/notes"
)

type noteLoadedMsg struct{ n notes.Note }
type noteCreatedMsg struct {
	d core.Project
	n notes.Note
}
type noteSavedMsg struct{ d core.Project }
type noteDeletedMsg struct{ d core.Project }

func newNoteEditor() textarea.Model {
	ta := textarea.New()
	ta.Placeholder = "# note"
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	ta.SetHeight(12)
	ta.SetWidth(72)
	return ta
}

func newTitleInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "note title"
	ti.CharLimit = 80
	ti.SetWidth(40)
	return ti
}

func indexNote(list []notes.Meta, name string) int {
	for i, n := range list {
		if n.Name == name {
			return i
		}
	}
	return 0
}

func (m model) selectedNote() (notes.Meta, bool) {
	if m.noteIdx >= 0 && m.noteIdx < len(m.detail.Notes) {
		return m.detail.Notes[m.noteIdx], true
	}
	return notes.Meta{}, false
}

func (m model) updateNotes(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.noteTitle {
		return m.updateNoteTitle(msg)
	}
	if m.noteEdit {
		return m.updateNoteEdit(msg)
	}
	switch msg.String() {
	case "esc", "backspace":
		m.notesMode = false
		m.status = m.detail.Repo.Name
		return m, nil
	case "j", "down":
		if n := len(m.detail.Notes); n > 0 {
			m.noteIdx = (m.noteIdx + 1) % n
		}
	case "k", "up":
		if n := len(m.detail.Notes); n > 0 {
			m.noteIdx = (m.noteIdx - 1 + n) % n
		}
	case "enter":
		if n, ok := m.selectedNote(); ok {
			return m, m.openNote(n.Name)
		}
	case "n", "c":
		m.noteTitle = true
		m.titleInput.SetValue("")
		m.titleInput.Focus()
		m.status = "new note"
		return m, nil
	case "D":
		if n, ok := m.selectedNote(); ok {
			return m, m.deleteNote(n.Name)
		}
	}
	return m, nil
}

func (m model) updateNoteTitle(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.noteTitle = false
		m.titleInput.Blur()
		m.status = "notes"
		return m, nil
	case "enter":
		title := strings.TrimSpace(m.titleInput.Value())
		m.noteTitle = false
		m.titleInput.Blur()
		if title == "" {
			title = "note"
		}
		return m, m.createNote(title)
	}
	var cmd tea.Cmd
	m.titleInput, cmd = m.titleInput.Update(msg)
	return m, cmd
}

func (m model) updateNoteEdit(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.noteEdit = false
		m.noteTA.Blur()
		m.status = "notes"
		return m, nil
	case "ctrl+s":
		return m, m.saveNote()
	case "ctrl+1":
		return m.applyNoteMD("h1"), nil
	case "ctrl+2":
		return m.applyNoteMD("h2"), nil
	case "ctrl+3":
		return m.applyNoteMD("h3"), nil
	case "ctrl+b":
		return m.applyNoteMD("bold"), nil
	case "ctrl+i":
		return m.applyNoteMD("italic"), nil
	case "ctrl+u":
		return m.applyNoteMD("ul"), nil
	case "ctrl+o":
		return m.applyNoteMD("ol"), nil
	case "ctrl+k":
		return m.applyNoteMD("check"), nil
	case "ctrl+q":
		return m.applyNoteMD("quote"), nil
	case "ctrl+l":
		return m.applyNoteMD("link"), nil
	}
	var cmd tea.Cmd
	m.noteTA, cmd = m.noteTA.Update(msg)
	return m, cmd
}

func (m model) applyNoteMD(id string) model {
	content := m.noteTA.Value()
	line := m.noteTA.Line()
	sel := ""
	if m.noteTA.HasSelection() {
		sel = m.noteTA.SelectedText()
	}
	switch id {
	case "h1":
		content = mdPrefixLine(content, line, "# ", true)
	case "h2":
		content = mdPrefixLine(content, line, "## ", true)
	case "h3":
		content = mdPrefixLine(content, line, "### ", true)
	case "ul":
		content = mdPrefixLine(content, line, "- ", false)
	case "ol":
		content = mdPrefixLine(content, line, "1. ", false)
	case "check":
		content = mdPrefixLine(content, line, "- [ ] ", false)
	case "quote":
		content = mdPrefixLine(content, line, "> ", false)
	case "bold":
		content = mdWrap(content, sel, "**", "**", "bold")
	case "italic":
		content = mdWrap(content, sel, "_", "_", "italic")
	case "link":
		content = mdWrap(content, sel, "[", "](url)", "text")
	}
	m.noteTA.SetValue(content)
	m.noteDirty = true
	return m
}

func (m model) openNote(name string) tea.Cmd {
	app := m.app
	id := m.detail.Repo.ID
	return func() tea.Msg {
		n, err := app.ReadNote(id, name)
		if err != nil {
			return detailMsg{err: err}
		}
		return noteLoadedMsg{n: n}
	}
}

func (m model) createNote(title string) tea.Cmd {
	app := m.app
	id := m.detail.Repo.ID
	return func() tea.Msg {
		n, err := app.CreateNote(id, title, "")
		if err != nil {
			return detailMsg{err: err}
		}
		d, err := app.Detail(context.Background(), id)
		if err != nil {
			return detailMsg{err: err}
		}
		return noteCreatedMsg{d: d, n: n}
	}
}

func (m model) saveNote() tea.Cmd {
	app := m.app
	id := m.detail.Repo.ID
	name := m.noteName
	content := m.noteTA.Value()
	return func() tea.Msg {
		if _, err := app.WriteNote(id, name, content); err != nil {
			return detailMsg{err: err}
		}
		d, err := app.Detail(context.Background(), id)
		if err != nil {
			return detailMsg{err: err}
		}
		return noteSavedMsg{d: d}
	}
}

func (m model) deleteNote(name string) tea.Cmd {
	app := m.app
	id := m.detail.Repo.ID
	return func() tea.Msg {
		if err := app.DeleteNote(id, name); err != nil {
			return detailMsg{err: err}
		}
		d, err := app.Detail(context.Background(), id)
		if err != nil {
			return detailMsg{err: err}
		}
		return noteDeletedMsg{d: d}
	}
}
