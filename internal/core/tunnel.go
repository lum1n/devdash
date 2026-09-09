package core

import (
	"bytes"
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
	args := []string{
		"-N", "-T",
		"-o", "BatchMode=yes",
		"-o", "ExitOnForwardFailure=yes",
		"-o", "ServerAliveInterval=30",
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "UserKnownHostsFile=/tmp/devdash-known-hosts",
		"-o", "IgnoreUnknown=UseKeychain,AddKeysToAgent",
	}
	if runningInDocker() {
		args = append(args, "-o", "IdentityAgent=SSH_AUTH_SOCK")
	}
	args = append(args, "-L", localForward(localURL, remoteAddr), ws.Host)
	cmd := exec.Command("ssh", args...)
	var stderr bytes.Buffer
	cmd.Stdout = nil
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ssh tunnel: %w", err)
	}
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()

	sshGone, probeErr := waitHealthy(ctx, localURL, 15*time.Second, exited)
	if sshGone {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" && probeErr != nil {
			msg = probeErr.Error()
		}
		if msg == "" {
			msg = "ssh exited"
		}
		return fmt.Errorf("workspace %s: %s", ws.ID, msg)
	}
	if probeErr != nil {
		_ = cmd.Process.Kill()
		select {
		case <-exited:
		case <-time.After(2 * time.Second):
		}
		return fmt.Errorf("workspace %s: %s → %s %s has no /api/health (%v). On the remote, run `devdash serve` (listen %s)",
			ws.ID, ws.Host, localURL, remoteAddr, probeErr, remoteAddr)
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

func waitHealthy(ctx context.Context, base string, d time.Duration, abort <-chan error) (sshGone bool, err error) {
	deadline := time.Now().Add(d)
	c := remote.New(base)
	var last error
	for time.Now().Before(deadline) {
		select {
		case e := <-abort:
			return true, e
		default:
		}
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		probe, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		last = c.Ping(probe)
		cancel()
		if last == nil {
			return false, nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	if last == nil {
		return false, fmt.Errorf("no response")
	}
	return false, last
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
