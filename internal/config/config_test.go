package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAppliesListenAndRootsEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("listen: 127.0.0.1:1\nroots:\n  - /old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEVDASH_LISTEN", "0.0.0.0:8789")
	t.Setenv("DEVDASH_ROOTS", "/repos"+string(os.PathListSeparator)+"/work")

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != "0.0.0.0:8789" {
		t.Fatalf("listen=%s", cfg.Listen)
	}
	if len(cfg.Roots) != 2 || cfg.Roots[0] != "/repos" || cfg.Roots[1] != "/work" {
		t.Fatalf("roots=%v", cfg.Roots)
	}
}

func TestResolvePathPrefersXDGOverOSUserConfig(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(home, ".config")
	osCfg := filepath.Join(home, "Library", "Application Support")
	if err := os.MkdirAll(filepath.Join(xdg, "devdash"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(osCfg, "devdash"), 0o700); err != nil {
		t.Fatal(err)
	}
	xdgFile := filepath.Join(xdg, "devdash", "config.yaml")
	osFile := filepath.Join(osCfg, "devdash", "config.yaml")
	if err := os.WriteFile(xdgFile, []byte("active: from-xdg\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(osFile, []byte("active: from-os\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got := resolvePath("", home, osCfg)
	if got != xdgFile {
		t.Fatalf("got %s want %s", got, xdgFile)
	}
}

func TestResolvePathFallsBackToExistingOSUserConfig(t *testing.T) {
	home := t.TempDir()
	osCfg := filepath.Join(home, "Library", "Application Support")
	if err := os.MkdirAll(filepath.Join(osCfg, "devdash"), 0o700); err != nil {
		t.Fatal(err)
	}
	osFile := filepath.Join(osCfg, "devdash", "config.yaml")
	if err := os.WriteFile(osFile, []byte("listen: 127.0.0.1:1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got := resolvePath("", home, osCfg)
	if got != osFile {
		t.Fatalf("got %s want %s", got, osFile)
	}
}

func TestResolvePathDefaultsToXDGWhenNothingExists(t *testing.T) {
	home := t.TempDir()
	osCfg := filepath.Join(home, "Library", "Application Support")
	want := filepath.Join(home, ".config", "devdash", "config.yaml")
	got := resolvePath("", home, osCfg)
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}
