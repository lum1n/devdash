package acc

import (
	"context"
	"path/filepath"
	"testing"

	acclib "github.com/lum1n/ai-command-center/pkg/acc"
	"github.com/lum1n/ai-command-center/pkg/acc/sessions"
	"github.com/lum1n/ai-command-center/pkg/acc/subscription"
	"github.com/lum1n/ai-command-center/pkg/acc/usage"
	"github.com/lum1n/devdash/internal/scan"
)

type fakeSource struct {
	snap snapshot
	last string
}

func (f *fakeSource) Load(context.Context) (snapshot, error) { return f.snap, nil }
func (f *fakeSource) Launch(_ context.Context, harness, dir string) (string, error) {
	f.last = "launch " + harness + " " + dir
	return "tmux attach -t acc-" + harness, nil
}
func (f *fakeSource) Resume(_ context.Context, source, sessionID, _ string) (string, error) {
	f.last = "resume " + source + " " + sessionID
	return "tmux attach -t acc-resume", nil
}

func TestWidgetsAndAnnotateFromSnapshot(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sessh")
	p := &Plugin{
		src: &fakeSource{snap: snapshot{
			dash: acclib.Dashboard{
				Inventory: acclib.Inventory{SkillcpOK: true},
				Live: acclib.Snapshot{
					Connected: true,
					Agents: []acclib.AgentPane{{
						Session: "acc-claude-sessh",
						Kind:    acclib.HarnessClaude,
						State:   acclib.StateWaitingPermission,
						Path:    dir,
					}},
				},
				Counts: acclib.StateCounts{Total: 1, Attention: 1, WaitingPermission: 1},
			},
			roll: usage.Rollup{
				Totals:    usage.Totals{CostUSD: 1.25},
				Today:     usage.Totals{CostUSD: 0.10},
				ByProject: []usage.NamedTotal{{Key: dir, Totals: usage.Totals{CostUSD: 0.40}}},
			},
			plans: []subscription.Status{{
				ID: subscription.ProviderClaude, Name: "Claude", UsedPct: 62, Health: subscription.HealthOK, Label: "5h",
			}},
			sessions: []sessions.Ref{{Source: acclib.HarnessClaude, ID: "abc", Project: dir}},
		}},
		catalog: acclib.DefaultCatalog(),
	}
	if err := p.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}

	w := p.Widgets()
	if len(w) != 1 || w[0].Kind != "acc.overview" {
		t.Fatalf("widgets=%v", w)
	}
	data, ok := w[0].Data.(OverviewData)
	if !ok || data.Fleet.Agents != 1 || data.Fleet.Attention != 1 || data.Cost.Month != 1.25 {
		t.Fatalf("overview data=%v ok=%v", w[0].Data, ok)
	}
	if len(data.Plans) != 1 || data.Plans[0].UsedPct != 62 {
		t.Fatalf("plans=%v", data.Plans)
	}

	notes := p.Annotate(scan.Repo{ID: "sessh", Path: dir})
	if len(notes) != 1 || notes[0].Tone != "danger" || notes[0].Label != "agent claude" {
		t.Fatalf("notes=%v", notes)
	}

	cursorDir := filepath.Join(t.TempDir(), "devdash")
	p.snap.dash.Live.Agents = append(p.snap.dash.Live.Agents, acclib.AgentPane{
		Session: "repos/devdash",
		Kind:    acclib.HarnessCursor,
		State:   acclib.StateThinking,
		Path:    filepath.Join(t.TempDir(), "store.db"),
	})
	cursorNotes := p.Annotate(scan.Repo{ID: "devdash", Name: "devdash", Path: cursorDir})
	if len(cursorNotes) != 1 || cursorNotes[0].Label != "agent cursor" {
		t.Fatalf("cursor session notes=%v", cursorNotes)
	}
	if got := p.Annotate(scan.Repo{ID: "other", Path: filepath.Join(t.TempDir(), "other")}); len(got) != 0 {
		t.Fatalf("other notes=%v", got)
	}

	proj := p.Project(scan.Repo{ID: "sessh", Path: dir})
	if len(proj) != 1 {
		t.Fatalf("project=%v", proj)
	}
	pd, ok := proj[0].Data.(ProjectData)
	if !ok || pd.Cost30d != 0.40 || len(pd.Agents) != 1 || len(pd.Sessions) != 1 {
		t.Fatalf("project data=%v ok=%v", proj[0].Data, ok)
	}

	src := p.src.(*fakeSource)
	res, err := p.Run(context.Background(), "launch", scan.Repo{Path: dir}, map[string]string{"harness": "claude"})
	if err != nil || !res.OK {
		t.Fatalf("launch %v %v", res, err)
	}
	if src.last != "launch claude "+dir {
		t.Fatalf("last=%q", src.last)
	}
	res, err = p.Run(context.Background(), "resume", scan.Repo{Path: dir}, nil)
	if err != nil || res.Action != "resume" {
		t.Fatalf("resume %v %v", res, err)
	}
}
