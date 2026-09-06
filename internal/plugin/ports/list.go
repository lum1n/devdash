package ports

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type snapshot struct {
	ready     bool
	listeners []Listener
	err       string
}

// Source lists listening sockets. Tests inject a fake.
type Source interface {
	Load() (snapshot, error)
	Open(url string) (string, error)
}

type liveSource struct{}

var ssUsers = regexp.MustCompile(`users:\(\("([^"]+)",pid=(\d+)`)

func (liveSource) Load() (snapshot, error) {
	if _, err := exec.LookPath("ss"); err != nil {
		return snapshot{err: "ss not on PATH"}, nil
	}
	out, err := exec.Command("ss", "-ltnpH").Output()
	if err != nil {
		return snapshot{ready: true, err: err.Error()}, nil
	}
	return snapshot{ready: true, listeners: parseSS(out)}, nil
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

func parseSS(raw []byte) []Listener {
	var out []Listener
	seen := map[string]bool{}
	for _, line := range bytes.Split(raw, []byte("\n")) {
		fields := strings.Fields(string(line))
		if len(fields) < 4 {
			continue
		}
		local := fields[3]
		if strings.EqualFold(fields[0], "LISTEN") && len(fields) > 4 {
			local = fields[3]
		}
		host, port, ok := splitHostPort(local)
		if !ok || skipPort(port) {
			continue
		}
		cmdName, pid := parseUsers(string(line))
		if pid == 0 || skipCommand(cmdName) {
			continue
		}
		cwd, err := os.Readlink(filepath.Join("/proc", strconv.Itoa(pid), "cwd"))
		if err != nil || cwd == "" {
			continue
		}
		key := fmt.Sprintf("%d:%d:%s", pid, port, cwd)
		if seen[key] {
			continue
		}
		seen[key] = true
		if cmdName == "" {
			if b, rerr := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "comm")); rerr == nil {
				cmdName = strings.TrimSpace(string(b))
			}
		}
		if skipCommand(cmdName) {
			continue
		}
		out = append(out, Listener{
			PID:     pid,
			Port:    port,
			Addr:    host,
			Cwd:     cwd,
			Command: cmdName,
			Label:   portLabel(port, cmdName),
			URL:     listenURL(host, port),
		})
	}
	return out
}

func parseUsers(line string) (string, int) {
	m := ssUsers.FindStringSubmatch(line)
	if len(m) < 3 {
		return "", 0
	}
	pid, _ := strconv.Atoi(m[2])
	return m[1], pid
}

func splitHostPort(local string) (string, int, bool) {
	local = strings.TrimSpace(local)
	if local == "" {
		return "", 0, false
	}
	// [::1]:3000 or 127.0.0.1:3000 or *:3000
	host, portStr, ok := strings.Cut(local, "]:")
	if ok {
		host = strings.TrimPrefix(host, "[")
	} else {
		host, portStr, ok = cutLastColon(local)
		if !ok {
			return "", 0, false
		}
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 {
		return "", 0, false
	}
	return host, port, true
}

func cutLastColon(s string) (string, string, bool) {
	i := strings.LastIndex(s, ":")
	if i < 0 {
		return "", "", false
	}
	return s[:i], s[i+1:], true
}

func skipPort(port int) bool {
	if port >= 32768 {
		return true
	}
	switch port {
	case 22, 25, 53, 67, 68, 123, 5353:
		return true
	default:
		return false
	}
}

func skipCommand(cmd string) bool {
	c := strings.ToLower(cmd)
	switch {
	case strings.Contains(c, "node-mainthread"), strings.Contains(c, "chrome"), strings.Contains(c, "electron"), strings.Contains(c, "cursor"):
		return true
	default:
		return false
	}
}

func portLabel(port int, cmd string) string {
	switch port {
	case 3000, 3001, 4173, 4321, 5173, 5174, 8000, 8080, 8081:
		return "web"
	case 8789:
		return "api"
	case 5432:
		return "postgres"
	case 6379:
		return "redis"
	}
	cmd = filepath.Base(strings.TrimSpace(cmd))
	if cmd != "" && cmd != "node" && cmd != "python" && cmd != "python3" {
		return cmd
	}
	return "port"
}

func listenURL(host string, port int) string {
	if host == "*" || host == "0.0.0.0" || host == "::" || host == "[::]" || host == "" {
		host = "127.0.0.1"
	}
	host = strings.Trim(host, "[]")
	scheme := "http"
	if port == 443 {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, host, port)
}

type errMsg string

func (e errMsg) Error() string { return string(e) }
