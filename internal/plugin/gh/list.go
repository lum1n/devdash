package gh

import (
	"encoding/json"
	"os/exec"
	"strings"
)

type pr struct {
	Number         int        `json:"number"`
	Title          string     `json:"title"`
	URL            string     `json:"url"`
	IsDraft        bool       `json:"isDraft"`
	ReviewDecision string     `json:"reviewDecision"`
	Repository     repository `json:"repository"`
	Checks         []check    `json:"statusCheckRollup"`
	Review         bool       `json:"-"`
}

type repository struct {
	Name          string `json:"name"`
	NameWithOwner string `json:"nameWithOwner"`
}

type check struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

type snapshot struct {
	ready bool
	prs   []pr
	err   string
}

// Source talks to gh. Tests inject a fake.
type Source interface {
	Load() (snapshot, error)
	RepoPRs(slug string) ([]pr, error)
	Open(url string) (string, error)
}

type liveSource struct{}

func (liveSource) Load() (snapshot, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return snapshot{err: "gh not on PATH"}, nil
	}
	authored, err1 := searchPRs("--author=@me")
	review, err2 := searchPRs("--review-requested=@me")
	if err1 != nil && err2 != nil {
		return snapshot{ready: true, err: err1.Error()}, nil
	}
	seen := map[string]bool{}
	var out []pr
	for _, p := range authored {
		key := p.Repository.NameWithOwner + "#" + itoa(p.Number)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, p)
	}
	for _, p := range review {
		key := p.Repository.NameWithOwner + "#" + itoa(p.Number)
		p.Review = true
		if i := indexPR(out, key); i >= 0 {
			out[i].Review = true
			continue
		}
		out = append(out, p)
	}
	return snapshot{ready: true, prs: out}, nil
}

func (liveSource) RepoPRs(slug string) ([]pr, error) {
	if slug == "" {
		return nil, nil
	}
	out, err := exec.Command("gh", "pr", "list", "-R", slug, "--state", "open", "--limit", "20",
		"--json", "number,title,url,isDraft,reviewDecision,statusCheckRollup").Output()
	if err != nil {
		return nil, err
	}
	var prs []pr
	if err := json.Unmarshal(out, &prs); err != nil {
		return nil, err
	}
	for i := range prs {
		prs[i].Repository.NameWithOwner = slug
		if _, name, ok := strings.Cut(slug, "/"); ok {
			prs[i].Repository.Name = name
		}
	}
	return prs, nil
}

func (liveSource) Open(url string) (string, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return "", errMsg("no url")
	}
	bin := "xdg-open"
	if _, err := exec.LookPath(bin); err != nil {
		bin = "open"
	}
	cmd := exec.Command(bin, url)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return "", err
	}
	return url, nil
}

func searchPRs(filter string) ([]pr, error) {
	args := []string{"search", "prs", "--state=open", "--limit=40", "--json", "repository,number,title,url"}
	args = append(args, filter)
	out, err := exec.Command("gh", args...).Output()
	if err != nil {
		return nil, err
	}
	var prs []pr
	if err := json.Unmarshal(out, &prs); err != nil {
		return nil, err
	}
	return prs, nil
}

func indexPR(prs []pr, key string) int {
	for i, p := range prs {
		if p.Repository.NameWithOwner+"#"+itoa(p.Number) == key {
			return i
		}
	}
	return -1
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func checksFail(p pr) bool {
	for _, c := range p.Checks {
		if strings.EqualFold(c.Conclusion, "FAILURE") || strings.EqualFold(c.Conclusion, "TIMED_OUT") {
			return true
		}
	}
	return false
}

type errMsg string

func (e errMsg) Error() string { return string(e) }
