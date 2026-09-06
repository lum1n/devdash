package ports

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/lum1n/devdash/internal/scan"
)

type fakeSource struct {
	snap snapshot
	last string
}

func (f *fakeSource) Load() (snapshot, error) { return f.snap, nil }
func (f *fakeSource) Open(url string) (string, error) {
	f.last = url
	return url, nil
}

func TestParseSS(t *testing.T) {
	raw := []byte("LISTEN 0 4096 127.0.0.1:3000 0.0.0.0:* users:((\"node\",pid=9,fd=23))\n")
	got := parseSS(raw)
	// cwd read will fail for pid 9 in tests; parseSS skips missing cwd.
	if len(got) != 0 {
		t.Fatalf("expected skip without /proc cwd, got %v", got)
	}
	host, port, ok := splitHostPort("127.0.0.1:5173")
	if !ok || host != "127.0.0.1" || port != 5173 {
		t.Fatalf("host=%s port=%d ok=%v", host, port, ok)
	}
	if u := listenURL("0.0.0.0", 3000); u != "http://127.0.0.1:3000" {
		t.Fatalf("url=%s", u)
	}
	if portLabel(3000, "node") != "web" || portLabel(8789, "devdash") != "api" {
		t.Fatal("labels")
	}
	if !skipPort(44263) || skipPort(3000) || !skipCommand("node-MainThread") {
		t.Fatal("skip")
	}
}

func TestAnnotateAndOpen(t *testing.T) {
	dir := t.TempDir()
	p := &Plugin{src: &fakeSource{snap: snapshot{
		ready: true,
		listeners: []Listener{{
			PID: 1, Port: 3000, Addr: "127.0.0.1", Cwd: dir, Command: "node",
			Label: "web", URL: "http://127.0.0.1:3000",
		}},
	}}}
	if err := p.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	repo := scan.Repo{ID: "devdash", Path: dir}
	notes := p.Annotate(repo)
	if len(notes) != 1 || notes[0].Label != "web :3000" {
		t.Fatalf("notes=%v", notes)
	}
	if got := p.Annotate(scan.Repo{ID: "x", Path: filepath.Join(t.TempDir(), "x")}); len(got) != 0 {
		t.Fatalf("other=%v", got)
	}
	res, err := p.Run(context.Background(), "open", repo, nil)
	if err != nil || res.Detail != "http://127.0.0.1:3000" {
		t.Fatalf("run %v %v", res, err)
	}
}

func TestParseUsersAndIPv6(t *testing.T) {
	cmd, pid := parseUsers(`users:(("devdash",pid=42,fd=4))`)
	if cmd != "devdash" || pid != 42 {
		t.Fatalf("cmd=%s pid=%d", cmd, pid)
	}
	host, port, ok := splitHostPort("[::1]:8789")
	if !ok || host != "::1" || port != 8789 {
		t.Fatalf("host=%s port=%d ok=%v", host, port, ok)
	}
}
