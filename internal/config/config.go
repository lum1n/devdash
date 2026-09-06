package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const envPrefix = "DEVDASH"

// Config is the on-disk operator config.
type Config struct {
	Listen     string      `yaml:"listen"`
	Active     string      `yaml:"active,omitempty"`
	Workspaces []Workspace `yaml:"workspaces,omitempty"`
	Roots      []string    `yaml:"roots,omitempty"`
	Ignore     []string    `yaml:"ignore,omitempty"`
	Actions    Actions     `yaml:"actions"`
	Focus      Focus       `yaml:"focus"`
}

// Actions are local shortcuts on the project screen.
type Actions struct {
	Editor string `yaml:"editor"`
	Term   string `yaml:"term"`
}

// Focus is the operator's Today queue state.
type Focus struct {
	Pinned   []string          `yaml:"pinned"`
	Archived []string          `yaml:"archived"`
	Snoozed  map[string]string `yaml:"snoozed"` // repo id -> RFC3339 until
	Opened   map[string]string `yaml:"opened"`  // repo id -> RFC3339 last open
	Next     map[string]string `yaml:"next"`    // repo id -> one-line next action
}

// Default returns secure local defaults.
func Default() Config {
	cfg := Config{
		Listen: "127.0.0.1:8789",
		Ignore: []string{"node_modules", ".git"},
	}
	cfg.Focus.Normalize()
	return cfg
}

// Path is the default config file location.
func Path() (string, error) {
	if p := os.Getenv(envPrefix + "_CONFIG"); p != "" {
		return p, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "devdash", "config.yaml"), nil
}

// Load reads path, or returns defaults if the file is missing.
func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		var err error
		path, err = Path()
		if err != nil {
			return cfg, err
		}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg = withDefaultRoot(cfg)
			cfg.Focus.Normalize()
			return finishLoad(cfg), nil
		}
		return cfg, fmt.Errorf("read config: %w", err)
	}
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Listen == "" {
		cfg.Listen = Default().Listen
	}
	cfg.Focus.Normalize()
	return finishLoad(cfg), nil
}

func finishLoad(cfg Config) Config {
	cfg = applyEnv(cfg)
	cfg.NormalizeWorkspaces()
	if v := strings.TrimSpace(os.Getenv(envPrefix + "_ROOTS")); v != "" {
		if ws := cfg.ActivePtr(); ws != nil && !ws.IsSSH() {
			ws.Roots = append([]string(nil), cfg.Roots...)
		}
	}
	cfg.syncLegacy()
	return cfg
}

func applyEnv(cfg Config) Config {
	if v := strings.TrimSpace(os.Getenv(envPrefix + "_LISTEN")); v != "" {
		cfg.Listen = v
	}
	if v := strings.TrimSpace(os.Getenv(envPrefix + "_WORKSPACE")); v != "" {
		cfg.Active = v
	}
	if v := strings.TrimSpace(os.Getenv(envPrefix + "_ROOTS")); v != "" {
		var roots []string
		for _, p := range filepath.SplitList(v) {
			p = strings.TrimSpace(p)
			if p != "" {
				roots = append(roots, p)
			}
		}
		if len(roots) > 0 {
			cfg.Roots = roots
		}
	}
	return cfg
}

// Save writes cfg to path, creating the parent directory.
func Save(path string, cfg Config) error {
	if path == "" {
		var err error
		path, err = Path()
		if err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	cfg.NormalizeWorkspaces()
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

func withDefaultRoot(cfg Config) Config {
	if len(cfg.Roots) > 0 {
		return cfg
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return cfg
	}
	repos := filepath.Join(home, "repos")
	if st, err := os.Stat(repos); err == nil && st.IsDir() {
		cfg.Roots = []string{repos}
	}
	return cfg
}
