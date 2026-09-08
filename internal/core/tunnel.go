package core

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/lum1n/devdash/internal/config"
	"github.com/lum1n/devdash/internal/remote"
)

type sshTunnel struct {
	cmd *exec.Cmd
	url string
}

func (a *App) ensureTunnel(ctx context.Context, ws config.Workspace) error {
	base := remote.NormalizeURL(ws.URL)
	candidates := append([]string{base}, dockerHostFallback(base)...)
	if hit := firstHealthy(ctx, candidates...); hit != "" {
		a.mu.Lock()
		a.tunnels[ws.ID] = &sshTunnel{url: hit}
		a.mu.Unlock()
		return nil
	}
	a.mu.RLock()
	existing := a.tunnels[ws.ID]
	a.mu.RUnlock()
	if existing != nil && existing.url != "" && remote.New(existing.url).Healthy(ctx) {
		return nil
	}
	if ws.Host == "" {
		if base == "" {
			return fmt.Errorf("workspace %s: set url or host", ws.ID)
		}
		return fmt.Errorf("workspace %s: %s not reachable (start remote `devdash serve` or ssh -L)", ws.ID, base)
	}
	localURL, remoteAddr := tunnelAddrs(ws)
	if _, err := exec.LookPath("ssh"); err != nil {
		hint := dockerHostFallback(base)
		if len(hint) > 0 {
			return fmt.Errorf("workspace %s: ssh not in this environment; run `ssh -L` on the host so %s is the hop (compose cannot reach 127.0.0.1 on the Mac)", ws.ID, hint[0])
		}
		return fmt.Errorf("workspace %s: ssh not on PATH", ws.ID)
	}
	cmd := exec.Command("ssh", "-N", "-T",
		"-o", "BatchMode=yes",
		"-o", "ExitOnForwardFailure=yes",
		"-o", "ServerAliveInterval=30",
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "UserKnownHostsFile=/tmp/devdash-known-hosts",
		"-L", localForward(localURL, remoteAddr),
		ws.Host,
	)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ssh tunnel: %w", err)
	}
	ready := waitHealthy(ctx, localURL, 8*time.Second)
	if !ready {
		_ = cmd.Process.Kill()
		return fmt.Errorf("workspace %s: tunnel up but %s has no devdash api", ws.ID, localURL)
	}
	a.mu.Lock()
	if old := a.tunnels[ws.ID]; old != nil && old.cmd != nil && old.cmd.Process != nil {
		_ = old.cmd.Process.Kill()
	}
	a.tunnels[ws.ID] = &sshTunnel{cmd: cmd, url: localURL}
	a.mu.Unlock()
	return nil
}

func tunnelAddrs(ws config.Workspace) (localURL, remoteAddr string) {
	localURL = remote.NormalizeURL(ws.URL)
	if localURL == "" {
		localURL = "http://127.0.0.1:8790"
	}
	remoteAddr = strings.TrimSpace(ws.Listen)
	if remoteAddr == "" {
		remoteAddr = "127.0.0.1:8789"
	}
	return localURL, remoteAddr
}

func localForward(localURL, remoteAddr string) string {
	u, err := url.Parse(localURL)
	host := "127.0.0.1"
	port := "8790"
	if err == nil && u.Host != "" {
		h, p, herr := net.SplitHostPort(u.Host)
		if herr == nil {
			host, port = h, p
		} else {
			host = u.Host
		}
	}
	if !strings.Contains(remoteAddr, ":") {
		remoteAddr = "127.0.0.1:" + remoteAddr
	}
	return host + ":" + port + ":" + remoteAddr
}

func waitHealthy(ctx context.Context, base string, d time.Duration) bool {
	deadline := time.Now().Add(d)
	c := remote.New(base)
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return false
		}
		if c.Healthy(ctx) {
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
	return c.Healthy(ctx)
}

func firstHealthy(ctx context.Context, urls ...string) string {
	for _, u := range urls {
		if u == "" {
			continue
		}
		if remote.New(u).Healthy(ctx) {
			return u
		}
	}
	return ""
}

func dockerHostFallback(base string) []string {
	if !runningInDocker() {
		return nil
	}
	alt := loopbackToDockerHost(base)
	if alt == "" || alt == base {
		return nil
	}
	return []string{alt}
}

func runningInDocker() bool {
	_, err := os.Stat("/.dockerenv")
	return err == nil
}

func loopbackToDockerHost(base string) string {
	u, err := url.Parse(strings.TrimSpace(base))
	if err != nil || u.Host == "" {
		return ""
	}
	host, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		return ""
	}
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		return ""
	}
	u.Host = net.JoinHostPort("host.docker.internal", port)
	return u.String()
}
