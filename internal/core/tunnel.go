package core

import (
	"context"
	"fmt"
	"net"
	"net/url"
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
	if base != "" && remote.New(base).Healthy(ctx) {
		a.mu.Lock()
		a.tunnels[ws.ID] = &sshTunnel{url: base}
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
		return fmt.Errorf("ssh not on PATH")
	}
	cmd := exec.CommandContext(ctx, "ssh", "-N", "-T",
		"-o", "BatchMode=yes",
		"-o", "ExitOnForwardFailure=yes",
		"-o", "ServerAliveInterval=30",
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
