package notes

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

const dirName = "notes"

var nameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,80}\.md$`)

// Meta is a note listing row.
type Meta struct {
	Name    string    `json:"name"`
	Title   string    `json:"title"`
	Updated time.Time `json:"updated_at"`
}

// Note is a markdown file in repo/notes.
type Note struct {
	Meta
	Content string `json:"content"`
}

// Dir is repoPath/notes.
func Dir(repoPath string) string {
	return filepath.Join(repoPath, dirName)
}

// List returns markdown notes, newest first. Missing dir is empty.
func List(repoPath string) ([]Meta, error) {
	entries, err := os.ReadDir(Dir(repoPath))
	if err != nil {
		if os.IsNotExist(err) {
			return []Meta{}, nil
		}
		return nil, err
	}
	var out []Meta
	for _, e := range entries {
		if e.IsDir() || !nameRe.MatchString(e.Name()) {
			continue
		}
		n, err := Read(repoPath, e.Name())
		if err != nil {
			continue
		}
		out = append(out, n.Meta)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Updated.Equal(out[j].Updated) {
			return out[i].Updated.After(out[j].Updated)
		}
		return out[i].Name < out[j].Name
	})
	if out == nil {
		return []Meta{}, nil
	}
	return out, nil
}

// Read loads one note.
func Read(repoPath, name string) (Note, error) {
	path, err := safePath(repoPath, name)
	if err != nil {
		return Note{}, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return Note{}, err
	}
	st, err := os.Stat(path)
	if err != nil {
		return Note{}, err
	}
	return Note{
		Meta:    Meta{Name: name, Title: titleOf(name, string(b)), Updated: st.ModTime()},
		Content: string(b),
	}, nil
}

// Create writes a new note from title and optional body.
func Create(repoPath, title, content string) (Note, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "note"
	}
	name, err := uniqueName(repoPath, slug(title))
	if err != nil {
		return Note{}, err
	}
	if strings.TrimSpace(content) == "" {
		content = "# " + title + "\n\n"
	}
	return Write(repoPath, name, content)
}

// Write creates notes/ if needed and saves markdown.
func Write(repoPath, name, content string) (Note, error) {
	path, err := safePath(repoPath, name)
	if err != nil {
		return Note{}, err
	}
	if err := os.MkdirAll(Dir(repoPath), 0o755); err != nil {
		return Note{}, err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return Note{}, err
	}
	return Read(repoPath, name)
}

// Delete removes a note. Missing file is ok.
func Delete(repoPath, name string) error {
	path, err := safePath(repoPath, name)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func uniqueName(repoPath, base string) (string, error) {
	if !strings.HasSuffix(base, ".md") {
		base += ".md"
	}
	if _, err := safePath(repoPath, base); err != nil {
		return "", err
	}
	dir := Dir(repoPath)
	name := base
	for i := 2; i < 50; i++ {
		if _, err := os.Stat(filepath.Join(dir, name)); os.IsNotExist(err) {
			return name, nil
		}
		stem := strings.TrimSuffix(base, ".md")
		name = fmt.Sprintf("%s-%d.md", stem, i)
	}
	return "", fmt.Errorf("too many notes named %s", base)
}

func safePath(repoPath, name string) (string, error) {
	raw := strings.TrimSpace(name)
	name = filepath.Base(raw)
	if raw == "" || name != raw || !nameRe.MatchString(name) {
		return "", fmt.Errorf("invalid note name %q", raw)
	}
	dir := Dir(repoPath)
	path := filepath.Join(dir, name)
	rel, err := filepath.Rel(dir, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("invalid note name %q", name)
	}
	return path, nil
}

func titleOf(name, content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			return strings.TrimSpace(strings.TrimLeft(line, "#"))
		}
		if line != "" {
			return line
		}
	}
	return strings.TrimSuffix(name, ".md")
}

func slug(title string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(title) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash && b.Len() > 0 {
			b.WriteByte('-')
			prevDash = true
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		s = "note"
	}
	if len(s) > 48 {
		s = s[:48]
	}
	return s + ".md"
}
