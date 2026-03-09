# claudetop Iteration 1 — Design

*Date: 2026-03-09*
*Scope: MVP session manager*

## What We're Building

A fullscreen TUI that manages Claude Code sessions backed by tmux. The user never interacts with tmux directly — claudetop is the entire interface. Sessions persist across TUI restarts via tmux.

## Technology

- **Language:** Go
- **TUI framework:** Bubbletea + bubbles (list, viewport components)
- **Session backing:** tmux (exec calls, no third-party tmux library)
- **tmux namespace:** `ct-` prefix (e.g. `ct-session-1`, `ct-annotation-timeout`)

## Configuration

`~/.claudetop/config.toml` — minimal for Iteration 1:

```toml
[general]
root_dir = "/path/to/root"
```

## State Persistence

`~/.claudetop/state.json` — persists the session list so TUI restart can reconnect. On startup, each session is checked against live tmux sessions; missing ones are marked dead and offered for cleanup.

## Layout

```
┌──────────────────────────────────────────────┐
│ claudetop  3 sessions  1 needs input  09:41  │  ← status bar (always visible)
├──────────────────────────────────────────────┤
│                                              │
│            [ active session output ]         │  ← main viewport (tmux pane output)
│                                              │
└──────────────────────────────────────────────┘
  \ sidebar   ;? help
```

With sidebar open (~22 chars):

```
┌───────────────────┬──────────────────────────┐
│ ACTIVE            │                          │
│ 1 ● session-1 🟡  │   [ session output ]     │
│ 2 ● session-2 🔴  │                          │
│                   │                          │
└───────────────────┴──────────────────────────┘
```

## Components

1. **Config loader** — reads `~/.claudetop/config.toml`, provides `root_dir`
2. **tmux manager** — create/list/kill/reconnect tmux sessions, read pane output
3. **State store** — read/write `~/.claudetop/state.json`
4. **Status detector** — pattern-match pane output to determine session status
5. **Status bar** — top line: app name, session count, needs-input count, time
6. **Sidebar** — numbered session list with status dots, toggled with `\`
7. **Main viewport** — renders active tmux pane output
8. **Keybinding handler** — session switching, sidebar toggle, close, help, new
9. **Help overlay** — shows keybindings on `?` or `;?`
10. **App wiring** — connects all components into a Bubbletea model

## Session Status

Detected by pattern-matching pane output (heuristic):

| Status | Dot | Detection |
|--------|-----|-----------|
| Starting | grey ● | First 10s |
| Working | yellow ● (animated) | Streaming output / tool calls |
| Needs input | red ● | Question pattern + idle output |
| Done/idle | green ● | Completion pattern + silence |
| Stuck | orange ● | No output >2min while "working" |
| Error | red ● + ✗ | Error pattern |

## Keybindings (Iteration 1)

All keypresses pass through to Claude Code when a session is focused, except those prefixed with `;`.

| Key | Action |
|-----|--------|
| `1`–`9` | Switch to session by number |
| `]` / `[` | Next / previous session |
| `\` | Toggle sidebar |
| `n` | New blank session |
| `x` | Close focused session (with confirmation) |
| `;?` or `?` | Toggle help overlay |
| `;q` | Quit (sessions keep running) |
| `;e` | Open root CLAUDE.md in $EDITOR (bonus) |

## Success Criteria

- All Claude Code sessions go through claudetop
- Sessions survive TUI restart
- Status dots reflect actual state
- Works correctly with 3+ concurrent sessions
