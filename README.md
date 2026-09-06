# Devdash

Local start dashboard for private projects. A **Go** process scans roots, serves JSON, and runs the TUI. **TanStack Start** is the web UI.

```text
web (TanStack Start :3000)  ──JSON──►  devdash serve (:8789)
                                         │
devdash / devdash tui  ──────────────────┘  same core
```

## Quick start

```bash
# API + scanner
go run ./cmd/devdash serve

# TUI
go run ./cmd/devdash

# Web
cd web && pnpm install && pnpm dev
```

Open http://127.0.0.1:3000. Default root is `~/repos` when that directory exists.

```bash
devdash scan --json
```

## Config

`~/.config/devdash/config.yaml` (see `config.example.yaml`):

```yaml
listen: 127.0.0.1:8789
roots:
  - /home/you/repos
```

Add more roots from the overview form, the TUI (`A`), the palette, or by editing the file.

Today is the start queue: dirty, behind, live agents, last opened, pins, and per-project next actions. `p` pins/unpins, `s` snoozes 24h, `x` archives. `:` or space opens the command palette (jump, editor, tmux, launch, rescan).

TUI matches the web loop: `/` and `f` filter repos, `N` edits `notes/`, `n` sets next, `h`/`a` and `H`/`R` launch or resume acc, `w`/`d` slice commits, `c`/`y` copy path or hash.

A next action is a one-line note in `focus.next` (web project field, TUI `n`). Markdown notes live in `{repo}/notes/*.md` and are edited on the project screen.

## Plugins

Compile-time Go interface in `internal/plugin`. A plugin registers widgets, palette commands, per-repo annotations, project panels, and optional actions.

`acc` is registered in `internal/cli` and wraps `ai-command-center/pkg/acc`:

- overview: fleet, plan meters, 30-day spend sparkline
- project: agents on that path, launch / resume in tmux
- config: `~/.config/acc/config.yaml` (same as the acc TUI)

Another plugin is a new package plus one `Register` call.

## Layout

```
cmd/devdash            CLI (tui default, serve, scan)
internal/core          shared overview, Today queue, palette
internal/notes         markdown notes in {repo}/notes
internal/scan          git root walker
internal/api           JSON for the web app
internal/tui           Bubble Tea v2
internal/plugin        registry
internal/plugin/acc    ai-command-center embed
web                    TanStack Start
```
