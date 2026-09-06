package gh

import (
	"net/url"
	"strings"
)

// GitHubRepo is owner/name parsed from a git remote.
type GitHubRepo struct {
	Owner string
	Name  string
}

func (g GitHubRepo) Slug() string {
	if g.Owner == "" || g.Name == "" {
		return ""
	}
	return g.Owner + "/" + g.Name
}

func parseGitHubRemote(remote string) (GitHubRepo, bool) {
	remote = strings.TrimSpace(remote)
	remote = strings.TrimSuffix(remote, ".git")
	if remote == "" {
		return GitHubRepo{}, false
	}
	if strings.HasPrefix(remote, "git@") {
		host, path, ok := strings.Cut(remote, ":")
		if !ok {
			return GitHubRepo{}, false
		}
		if !strings.EqualFold(strings.TrimPrefix(host, "git@"), "github.com") {
			return GitHubRepo{}, false
		}
		return splitOwnerName(path)
	}
	u, err := url.Parse(remote)
	if err != nil {
		return GitHubRepo{}, false
	}
	host := strings.ToLower(u.Hostname())
	if host != "github.com" && host != "www.github.com" {
		return GitHubRepo{}, false
	}
	return splitOwnerName(strings.TrimPrefix(u.Path, "/"))
}

func splitOwnerName(path string) (GitHubRepo, bool) {
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return GitHubRepo{}, false
	}
	return GitHubRepo{Owner: parts[0], Name: strings.TrimSuffix(parts[1], ".git")}, true
}

func matchesRepo(repoID, repoName, remote string, slug GitHubRepo) bool {
	if slug.Name != "" {
		if strings.EqualFold(slug.Name, repoID) || strings.EqualFold(slug.Name, repoName) {
			return true
		}
	}
	parsed, ok := parseGitHubRemote(remote)
	if !ok {
		return false
	}
	return strings.EqualFold(parsed.Slug(), slug.Slug())
}
