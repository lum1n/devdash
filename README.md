# Devdash

Local start dashboard for private git checkouts. A **Go** process is the source of truth: it scans roots, holds focus state, talks to plugins, and serves JSON. **TanStack Start** is the web UI. A **Bubble Tea** TUI shares the same core.

```text
browser  :3000  ── /api ──►  TanStack Start (dev or preview)
                                  │
                                  ▼
devdash serve  :8789   JSON API, scan, notes, diffs, plugins
        ▲
        │
devdash / tui  ──────────── same App
```

Bind to loopback by default. Nothing here is meant to be public.

## Requirements

| Piece | Why |
| --- | --- |
| Go 1.26+ | API, TUI, scanner |
| Node 22 + pnpm | web UI |
| git | every scan and diff |
| [ai-command-center](https://github.com/lum1n/ai-command-center) as `../ai-command-center` | `go.mod` replace; acc plugin |
| tmux (optional) | terminal shortcut, acc launch/resume |
| [agent-watcher](https://github.com/lum1n/agent-watcher) (optional) | live agent panes — consumed by acc, not a second plugin |

A folder appears in the list if it has its own `.git`, including nested under the root (`clients/acme`). Scan does not walk into a found repo, ignored names (`node_modules`, …), or hidden directories. A directory that merely sits inside another repo (no `.git` of its own) is skipped.

## Run locally

```bash
# API + scanner  (http://127.0.0.1:8789)
go run ./cmd/devdash serve

# Web  (http://127.0.0.1:3000, proxies /api to the Go process)
cd web && pnpm install && pnpm dev

# TUI (same core; no web required)
go run ./cmd/devdash
# or: go run ./cmd/devdash tui

# One-shot scan
go run ./cmd/devdash scan
go run ./cmd/devdash scan --json
```

Default root is `~/repos` when that directory exists. Override with config or `DEVDASH_ROOTS`.

### Install a binary

```bash
go build -o devdash ./cmd/devdash
install -m 755 devdash ~/.local/bin/devdash
```

`--config` and `DEVDASH_CONFIG` select the yaml file. Default is `~/.config/devdash/config.yaml` (or `$XDG_CONFIG_HOME/devdash/config.yaml`). On macOS, if that file is missing, an existing file under `~/Library/Application Support/devdash/` is used.

## Run with Docker

The image builds the API against the sibling acc module and serves the web preview. Mount your checkouts and config. Editor / tmux / agent launch stay on the host; the container is the dashboard.

`../ai-command-center` must exist next to this repo.

```bash
# optional: point at a different checkout tree
export DEVDASH_REPOS="$HOME/repos"

docker compose up --build
```

- Web: http://127.0.0.1:3000
- API: http://127.0.0.1:8789

```bash
docker compose up -d --build      # background
docker compose logs -f api
docker compose down
```

Compose sets `DEVDASH_LISTEN=0.0.0.0:8789` and `DEVDASH_ROOTS=/repos`, and bind-mounts:

- `${DEVDASH_REPOS:-$HOME/repos}` → `/repos` (read-only)
- `~/.config/devdash` → container config (focus, pins, next)
- `~/.config/acc` → acc config (read-only)

Ports are published on loopback only.

SSH workspaces inside Compose: the image has `ssh` and mounts `~/.ssh`. Linux `ssh` ignores macOS `UseKeychain` in that file and uses `SSH_AUTH_SOCK`. Docker Desktop exposes the agent at `/run/host-services/ssh-auth.sock` (compose maps it when present). A hop already open on the Mac (`ssh -L 8790:127.0.0.1:8789 host`) is reached as `host.docker.internal`, not container `127.0.0.1`.

`DEVDASH_ROOTS=/repos` only affects the in-container scan. It is not written back to `~/.config/devdash/config.yaml`.

If the active workspace is SSH and the hop is down, overview shows the error and the sidebar picker still works — switch to a local workspace.

## Config

Copy `config.example.yaml` to `~/.config/devdash/config.yaml`:

```yaml
listen: 127.0.0.1:8789
active: private
workspaces:
  - id: work
    name: work
    roots: [/home/you/work]
  - id: private
    name: private
    roots: [/home/you/repos]
  - id: private-remote
    name: private remote
    kind: ssh
    host: you@box
    url: 127.0.0.1:8790          # local hop (ssh -L)
    listen: 127.0.0.1:8789       # remote `devdash serve` bind
```

A workspace is a named scan context with its own roots and Focus (pins / next / archive). Existing `roots:` + `focus:` become workspace `local` on first load.

| Key / env | Role |
| --- | --- |
| `listen` / `DEVDASH_LISTEN` | API bind address |
| `workspaces` | named local or ssh contexts |
| `active` / `DEVDASH_WORKSPACE` | selected workspace id |
| `roots` / `DEVDASH_ROOTS` | scan overlay for local workspaces (not saved) |
| `DEVDASH_CONFIG` | config file path |
| `ignore` | child directory names to skip |
| `actions.editor` / `actions.term` | project shortcuts (`e`, `t`) |
| `focus.*` | legacy Focus; now stored per workspace |

**SSH workspace:** the remote host runs `devdash serve`. `url` is the local side of `ssh -L`. `listen` is where that remote process binds (default `127.0.0.1:8789`), not the local hop. `host` + `url` and Devdash opens `ssh -N -L`. Plugins and Focus on that hop are the remote machine's.

Switch with the sidebar workspace picker, settings, palette (`workspace · …`), or TUI `W`.

## Web

Sidebar chrome is stable: **overview**, **settings**, and a workspace `<select>`. Project, note, and diff do not add nav items — the header is a trail (`overview / repo / note`).

Below 640px the sidebar becomes a compact top bar, tables stack as cards (no horizontal scroll), and the workspace control uses the native picker. Desktop layout is unchanged.

## Daily loop

**Today** is the start queue: dirty, behind, live agents, last opened, pins, and per-project next actions.

| | Web | TUI |
| --- | --- | --- |
| Pin / unpin | project / today | `p` |
| Snooze 24h | today | `s` |
| Archive / unarchive | project / settings | `x` |
| Next action | project field | `n` |
| Notes | cards → own screen | `N` |
| Working-tree diff | click a dirty path | — |
| Commit diff | click a subject | — |
| Command palette | `:` or space | `:` or space |
| Workspace | sidebar picker | `W` |
| Filter repos | `/` | `/` `f` |
| Launch / resume acc | project acc panel | `h`/`a`  `H`/`R` |
| Copy path / hash | shortcut / hash | `c` / `y` |

A next action is one line in `focus.next`. Markdown notes live in `{repo}/notes/*.md`. Diffs use `@pierre/diffs` (unified or split) on their own screen: one file from the working tree, every file from a commit.

## Plugins

Compile-time Go interface in `internal/plugin`. A plugin can add overview widgets, palette commands, per-repo badges, project panels, and actions.

### acc (registered)

`internal/plugin/acc` wraps `ai-command-center/pkg/acc`. Config is `~/.config/acc/config.yaml` (same file as the acc TUI).

- Overview: fleet, plan meters, 30-day spend
- Project: agents on that path, launch / resume in tmux
- Badges: live agent state on the repo list

### agent-watcher (not a second plugin)

Live pane state already comes through acc:

1. acc discovers the socket (`watcher_socket`, `ACC_WATCHER_SOCKET`, or `{tmux socket}.agent-watcher.sock`)
2. acc fetches a snapshot
3. Devdash shows fleet counts, “watcher off” when disconnected, and per-repo agent badges

A separate agent-watcher plugin would duplicate those signals. Run the watcher on the **host**, next to tmux:

```bash
# socket path acc already looks up
sock="$(tmux display-message -p '#{socket_path}').agent-watcher.sock"
python3 ../agent-watcher/src/agent_watcher.py --listen "$sock"
```

Or set `watcher_socket` / `ACC_WATCHER_SOCKET` to that path. If the watcher is down, git status, notes, and diffs still work; the acc strip shows `watcher off`.

Another plugin is a new package plus one `Register` call in `internal/cli`. Bundled next to acc:

| | |
| --- | --- |
| `tmux` | live sessions matched by name or pane cwd; attach command |
| `ports` | listen sockets whose cwd is the repo; open the URL |
| `gh` | open PRs and review-requested via the `gh` CLI |

Each is optional at the Go boundary: drop its `Register` line and the panel disappears. Core scores any plugin annotation on Today by tone.

## API

`devdash serve` (default `127.0.0.1:8789`):

| | |
| --- | --- |
| `GET /api/health` | liveness |
| `GET /api/overview` | repos, today, palette, plugins |
| `GET /api/repos/{id}` | project detail |
| `GET /api/repos/{id}/diff/{ref}` | `ref=worktree&path=` or a commit hash |
| `GET/POST/PUT/DELETE /api/repos/{id}/notes` | markdown notes |
| `POST /api/scan` | rescan |
| `GET /api/workspaces` | workspace list |
| `POST /api/workspaces` | add workspace |
| `POST /api/workspaces/{id}/select` | switch workspace |
| `POST /api/plugins/{id}/run` | plugin action (acc, tmux, ports, gh) |

CORS allows `http://127.0.0.1:3000` and `http://localhost:3000`.

## Layout

```
cmd/devdash              CLI (tui default, serve, scan)
internal/core            overview, Today, palette, workspaces
internal/config          yaml + env + workspace migrate
internal/remote          SSH hop client
internal/notes           {repo}/notes/*.md
internal/scan            git walker + diffs
internal/api             JSON for the web app
internal/tui             Bubble Tea v2
internal/plugin          registry
internal/plugin/acc      ai-command-center embed
internal/plugin/tmux     live tmux sessions
internal/plugin/ports    listen sockets by cwd
internal/plugin/gh       GitHub PRs via gh
internal/repomatch       session/path → repo
web                      TanStack Start (mobile cards, header trail)
compose.yaml             API + web
Dockerfile               multi-stage api / web
```

## Develop

```bash
go test ./...
gofmt -w ./internal ./cmd
cd web && pnpm typecheck && pnpm test
```

Web talks to the API through `DEVDASH_API_URL` (default `http://127.0.0.1:8789`). After changing Go, restart `devdash serve`.
