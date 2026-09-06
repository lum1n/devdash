package acc

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	acclib "github.com/lum1n/ai-command-center/pkg/acc"
	"github.com/lum1n/ai-command-center/pkg/acc/sessions"
	"github.com/lum1n/ai-command-center/pkg/acc/subscription"
	"github.com/lum1n/ai-command-center/pkg/acc/usage"
)

type snapshot struct {
	dash     acclib.Dashboard
	roll     usage.Rollup
	plans    []subscription.Status
	sessions []sessions.Ref
	err      string
}

// Source loads acc state and starts agents. Tests inject a fake.
type Source interface {
	Load(ctx context.Context) (snapshot, error)
	Launch(ctx context.Context, harness, dir string) (string, error)
	Resume(ctx context.Context, source, sessionID, dir string) (string, error)
}

type liveSource struct {
	cfg     acclib.Config
	catalog []acclib.HarnessCatalog
}

func newLiveSource() *liveSource {
	cfg, err := acclib.LoadDefaultConfig()
	if err != nil {
		cfg = acclib.Config{SkillcpBinary: "skillcp", Subscriptions: subscription.DefaultConfig()}
	}
	return &liveSource{cfg: cfg, catalog: cfg.EffectiveCatalog()}
}

func (s *liveSource) Load(ctx context.Context) (snapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	var out snapshot
	inv, err := acclib.LoadInventory(ctx, s.cfg.SkillcpBinary)
	if err != nil {
		out.err = err.Error()
	}
	sock, sockErr := acclib.DiscoverWatcherSocket(s.cfg.WatcherSocket)
	if sockErr != nil && out.err == "" {
		out.err = sockErr.Error()
	}
	live, _ := acclib.FetchSnapshot(ctx, sock)
	out.dash = acclib.BuildDashboard(inv, live)

	if store, err := usage.OpenDefault(); err == nil {
		sc := usage.NewScanner(store, usage.DefaultPaths())
		if roll, qerr := sc.QuerySince(30*24*time.Hour, false); qerr == nil {
			out.roll = roll
		} else if out.err == "" {
			out.err = qerr.Error()
		}
		_ = store.Close()
	}

	if plans, err := subscription.FetchAll(ctx, s.cfg.Subscriptions); err == nil {
		out.plans = plans
	} else if out.err == "" {
		out.err = err.Error()
	}

	if refs, err := sessions.ListRecent(usage.DefaultPaths(), 4); err == nil {
		out.sessions = refs
	}
	return out, nil
}

func (s *liveSource) Launch(ctx context.Context, harness, dir string) (string, error) {
	res, err := acclib.RunLaunch(ctx, s.catalog, acclib.LaunchSpec{
		Dir:     dir,
		Harness: acclib.HarnessID(harness),
		Mode:    acclib.LaunchNewSession,
	})
	if err != nil {
		return "", err
	}
	return res.AttachCmd, nil
}

func (s *liveSource) Resume(ctx context.Context, source, sessionID, dir string) (string, error) {
	ref := sessions.Ref{Source: acclib.HarnessID(source), ID: sessionID, Project: dir}
	for _, r := range mustListRecent() {
		if string(r.Source) == source && r.ID == sessionID {
			ref = r
			if ref.Project == "" {
				ref.Project = dir
			}
			break
		}
	}
	res, err := sessions.RunResume(ref, "")
	if err != nil {
		return "", err
	}
	_ = ctx
	return res.AttachCmd, nil
}

func mustListRecent() []sessions.Ref {
	refs, err := sessions.ListRecent(usage.DefaultPaths(), 8)
	if err != nil {
		return nil
	}
	return refs
}

func harnessesOnPATH(catalog []acclib.HarnessCatalog) []Harness {
	out := make([]Harness, 0, len(catalog))
	for _, h := range catalog {
		_, err := exec.LookPath(h.Binary)
		out = append(out, Harness{
			ID:    string(h.ID),
			Label: h.Label,
			Ready: err == nil,
		})
	}
	return out
}

func sparkFrom(roll usage.Rollup) []float64 {
	tl := roll.Timeline
	if len(tl) == 0 {
		return []float64{}
	}
	const points = 24
	if len(tl) <= points {
		out := make([]float64, len(tl))
		for i, b := range tl {
			out[i] = b.Totals.CostUSD
		}
		return out
	}
	out := make([]float64, points)
	for i := 0; i < points; i++ {
		start := i * len(tl) / points
		end := (i + 1) * len(tl) / points
		var sum float64
		for _, b := range tl[start:end] {
			sum += b.Totals.CostUSD
		}
		out[i] = sum
	}
	return out
}

func plansFrom(in []subscription.Status) []Plan {
	out := make([]Plan, 0, len(in))
	for _, p := range in {
		if p.Health == subscription.HealthDisabled {
			continue
		}
		out = append(out, Plan{
			ID:      string(p.ID),
			Name:    p.Name,
			UsedPct: p.UsedPct,
			Label:   p.Label,
			Health:  p.Health,
			Detail:  firstNonEmpty(p.Detail, p.Error),
		})
	}
	return out
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func fmtUSD(n float64) string {
	if n <= 0 {
		return "$0"
	}
	if n < 0.01 {
		return fmt.Sprintf("$%.4f", n)
	}
	return fmt.Sprintf("$%.2f", n)
}
