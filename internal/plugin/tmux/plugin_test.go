package tmux

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
func (f *fakeSource) Attach(session string) (string, error) {
	f.last = session
	return "tmux attach -t " + session, nil
}

func TestParsePanesAndMatch(t *testing.T) {
	raw := []byte("repos/devdash\t1\tdev\t/tmp/x\tnvim\t1\nsessh\t0\tmain\t/home/u/sessh\tnode\t0\n")
	got := parsePanes(raw)
	if len(got) != 2 || got[0].Name != "repos/devdash" || !got[0].Attached || len(got[0].Windows) != 1 {
		t.Fatalf("parsed=%v", got)
	}
}

func TestWidgetsAnnotateAttach(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "devdash")
	p := &Plugin{src: &fakeSource{snap: snapshot{
		ready: true,
		sessions: []Session{{
			Name:    "repos/devdash",
			Windows: []Window{{Index: 1, Name: "zsh", Path: filepath.Join(t.TempDir(), "store.db")}},
		}},
	}}}
	if err := p.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if w := p.Widgets(); len(w) != 1 || w[0].Kind != "tmux.overview" {
		t.Fatalf("widgets=%v", w)
	}
	repo := scan.Repo{ID: "devdash", Name: "devdash", Path: dir}
	notes := p.Annotate(repo)
	if len(notes) != 1 || notes[0].Kind != "tmux" || notes[0].Detail != "repos/devdash" {
		t.Fatalf("notes=%v", notes)
	}
	if got := p.Annotate(scan.Repo{ID: "other", Path: filepath.Join(t.TempDir(), "other")}); len(got) != 0 {
		t.Fatalf("other=%v", got)
	}
	res, err := p.Run(context.Background(), "attach", repo, nil)
	if err != nil || !res.OK || res.Detail != "tmux attach -t repos/devdash" {
		t.Fatalf("run %v %v", res, err)
	}
}

func TestParsePanesEmpty(t *testing.T) {
	if got := parsePanes(nil); len(got) != 0 {
		t.Fatalf("got=%v", got)
	}
}
