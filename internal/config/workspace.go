package config

import (
	"os"
	"strings"
	"unicode"
)

const (
	KindLocal = "local"
	KindSSH   = "ssh"
)

// Workspace is a named scan context: local roots or a remote devdash API.
type Workspace struct {
	ID     string   `yaml:"id" json:"id"`
	Name   string   `yaml:"name,omitempty" json:"name"`
	Kind   string   `yaml:"kind,omitempty" json:"kind"`
	Roots  []string `yaml:"roots,omitempty" json:"roots,omitempty"`
	Ignore []string `yaml:"ignore,omitempty" json:"ignore,omitempty"`
	Host   string   `yaml:"host,omitempty" json:"host,omitempty"`
	URL    string   `yaml:"url,omitempty" json:"url,omitempty"`
	Listen string   `yaml:"listen,omitempty" json:"listen,omitempty"` // remote API bind, default 127.0.0.1:8789
	Focus  Focus    `yaml:"focus" json:"-"`
}

// IsSSH reports a remote hop.
func (w Workspace) IsSSH() bool {
	return w.Kind == KindSSH
}

// DisplayName is name or id.
func (w Workspace) DisplayName() string {
	if strings.TrimSpace(w.Name) != "" {
		return strings.TrimSpace(w.Name)
	}
	return w.ID
}

// NormalizeWorkspaces migrates legacy roots/focus into a default workspace.
func (c *Config) NormalizeWorkspaces() {
	if c.Active == "" {
		c.Active = strings.TrimSpace(os.Getenv(envPrefix + "_WORKSPACE"))
	}
	if len(c.Workspaces) == 0 {
		id := "local"
		c.Workspaces = []Workspace{{
			ID:    id,
			Name:  "local",
			Kind:  KindLocal,
			Roots: append([]string(nil), c.Roots...),
			Focus: c.Focus,
		}}
		if c.Active == "" {
			c.Active = id
		}
	}
	seen := map[string]int{}
	for i := range c.Workspaces {
		w := &c.Workspaces[i]
		if w.ID == "" {
			w.ID = slugID(w.Name)
		}
		if w.ID == "" {
			w.ID = "ws"
		}
		if n, ok := seen[w.ID]; ok {
			w.ID = w.ID + "-" + itoa(n+1)
		}
		seen[w.ID]++
		if w.Name == "" {
			w.Name = w.ID
		}
		if w.Kind == "" {
			if w.Host != "" || w.URL != "" {
				w.Kind = KindSSH
			} else {
				w.Kind = KindLocal
			}
		}
		if w.Kind != KindSSH {
			w.Kind = KindLocal
		}
		w.Focus.Normalize()
	}
	if c.workspaceIndex(c.Active) < 0 {
		c.Active = c.Workspaces[0].ID
	}
	c.syncLegacy()
}

func (c *Config) workspaceIndex(id string) int {
	id = strings.TrimSpace(id)
	for i, w := range c.Workspaces {
		if w.ID == id {
			return i
		}
	}
	return -1
}

// ActiveWorkspace is the selected workspace.
func (c *Config) ActiveWorkspace() Workspace {
	if i := c.workspaceIndex(c.Active); i >= 0 {
		return c.Workspaces[i]
	}
	if len(c.Workspaces) > 0 {
		return c.Workspaces[0]
	}
	return Workspace{ID: "local", Name: "local", Kind: KindLocal}
}

// ActivePtr is the selected workspace.
func (c *Config) ActivePtr() *Workspace {
	if i := c.workspaceIndex(c.Active); i >= 0 {
		return &c.Workspaces[i]
	}
	if len(c.Workspaces) > 0 {
		return &c.Workspaces[0]
	}
	return nil
}

// FocusPtr is focus for the active workspace.
func (c *Config) FocusPtr() *Focus {
	if ws := c.ActivePtr(); ws != nil {
		return &ws.Focus
	}
	c.Focus.Normalize()
	return &c.Focus
}

// ScanRoots are directories to walk for the active local workspace.
func (c *Config) ScanRoots() []string {
	if ws := c.ActivePtr(); ws != nil && !ws.IsSSH() {
		if len(ws.Roots) > 0 {
			return append([]string(nil), ws.Roots...)
		}
	}
	return append([]string(nil), c.Roots...)
}

// ScanIgnore merges global and workspace ignore lists.
func (c *Config) ScanIgnore() []string {
	seen := map[string]bool{}
	var out []string
	add := func(ids []string) {
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true
			out = append(out, id)
		}
	}
	add(c.Ignore)
	if ws := c.ActivePtr(); ws != nil {
		add(ws.Ignore)
	}
	return out
}

func (c *Config) syncLegacy() {
	ws := c.ActivePtr()
	if ws == nil || ws.IsSSH() {
		return
	}
	c.Roots = append([]string(nil), ws.Roots...)
	c.Focus = ws.Focus
}

func slugID(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return ""
	}
	var b strings.Builder
	prevDash := false
	for _, r := range name {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			prevDash = false
		case r == '-' || r == '_' || unicode.IsSpace(r):
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func itoa(n int) string {
	if n <= 0 {
		return "0"
	}
	var buf [8]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
