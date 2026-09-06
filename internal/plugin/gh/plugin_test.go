package gh

import (
	"context"
	"testing"

	"github.com/lum1n/devdash/internal/scan"
)

type fakeSource struct {
	snap snapshot
	repo []pr
	last string
}

func (f *fakeSource) Load() (snapshot, error)      { return f.snap, nil }
func (f *fakeSource) RepoPRs(string) ([]pr, error) { return f.repo, nil }
func (f *fakeSource) Open(url string) (string, error) {
	f.last = url
	return url, nil
}

func TestParseGitHubRemote(t *testing.T) {
	cases := []string{
		"git@github.com:lum1n/sessh.git",
		"https://github.com/lum1n/sessh.git",
		"https://github.com/lum1n/sessh",
	}
	for _, remote := range cases {
		g, ok := parseGitHubRemote(remote)
		if !ok || g.Slug() != "lum1n/sessh" {
			t.Fatalf("%s -> %+v ok=%v", remote, g, ok)
		}
	}
	if _, ok := parseGitHubRemote("git@gitlab.com:x/y.git"); ok {
		t.Fatal("gitlab")
	}
}

func TestAnnotateReviewAndOpen(t *testing.T) {
	p := New()
	p.src = &fakeSource{
		snap: snapshot{ready: true, prs: []pr{{
			Number: 4, Title: "fix attach", URL: "https://github.com/lum1n/sessh/pull/4",
			Repository: repository{Name: "sessh", NameWithOwner: "lum1n/sessh"},
			Review:     true,
		}}},
		repo: []pr{{
			Number: 4, Title: "fix attach", URL: "https://github.com/lum1n/sessh/pull/4",
			Repository: repository{Name: "sessh", NameWithOwner: "lum1n/sessh"},
			Checks:     []check{{Conclusion: "FAILURE"}},
		}},
	}
	if err := p.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	repo := scan.Repo{ID: "sessh", Name: "sessh", Remote: "git@github.com:lum1n/sessh.git"}
	notes := p.Annotate(repo)
	if len(notes) != 1 || notes[0].Kind != "review" {
		t.Fatalf("notes=%v", notes)
	}
	if got := p.Annotate(scan.Repo{ID: "other", Name: "other"}); len(got) != 0 {
		t.Fatalf("other=%v", got)
	}
	proj := p.Project(repo)
	data, ok := proj[0].Data.(ProjectData)
	if !ok || len(data.Pulls) != 1 || data.Pulls[0].Checks != "fail" {
		t.Fatalf("project=%v", proj[0].Data)
	}
	res, err := p.Run(context.Background(), "open", repo, nil)
	if err != nil || res.Detail != "https://github.com/lum1n/sessh/pull/4" {
		t.Fatalf("run %v %v", res, err)
	}
}

func TestMatchesRepoByName(t *testing.T) {
	if !matchesRepo("devdash", "devdash", "", GitHubRepo{Owner: "lum1n", Name: "devdash"}) {
		t.Fatal("name")
	}
	if matchesRepo("sessh", "sessh", "", GitHubRepo{Owner: "lum1n", Name: "devdash"}) {
		t.Fatal("other")
	}
}
