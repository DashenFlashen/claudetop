# claudetop Iteration 1 — Done

## What Was Built

A fullscreen TUI that manages Claude Code sessions backed by tmux. Sessions survive TUI
restarts. The user never interacts with tmux directly.

## Architecture Summary

- **Language:** Go
- **TUI:** Bubbletea (Elm-style message-passing), lipgloss for styling
- **Session backing:** tmux sessions with `ct-` namespace prefix
- **Config:** `~/.claudetop/config.toml` (root_dir only for now)
- **State:** `~/.claudetop/state.json` (atomic write via rename)

### Key design decisions

- Commands (`tea.Cmd`) are pure; model mutation happens only in `Update`.
  `sessionSpawnedMsg` and `sessionClosedMsg` (carrying session ID, not index) are
  the message types for async session lifecycle events.
- `;` is the leader key. While a session is focused, `;` arms the leader; the next
  keypress is a TUI command. `\` and `;` are always intercepted; all other keys pass
  through to tmux via `send-keys`.
- Active session pane is polled every ~150ms; background sessions every ~2s (tick%13).
- Dead sessions (no live tmux counterpart on reconnect) are kept in state with
  `Dead: true` and shown in the sidebar with a `✗ [dead]` label rather than silently
  dropped.

## Components

| Package | Responsibility |
|---|---|
| `internal/config` | Load `~/.claudetop/config.toml`, resolve config dir |
| `internal/session` | Session struct, status enum, heuristic status detection |
| `internal/tmux` | Create/kill/capture/send-keys, live session list |
| `internal/state` | Atomic JSON persistence of session list |
| `internal/ui` | Bubbletea model, status bar, sidebar, viewport, help, new-session overlay |

## Tests

```
ok  claudetop/internal/config
ok  claudetop/internal/session
ok  claudetop/internal/state
```

tmux and ui packages are not unit-tested (they wrap external processes / terminal I/O).

## Known Limitations

- **Stuck detection:** Once a session transitions to Done (silence + no question
  patterns), it won't re-enter Stuck even if it stops responding. Stuck only fires
  while the current status is Working. Heuristic limitation.

## Integration Testing

Run `./claudetop` manually and verify the checklist in PLAN.md Task 14:

- Session spawns in root_dir
- Sidebar shows session with correct status dot
- `1`–`9`, `]`, `[` switch sessions
- `\` toggles sidebar
- `x` closes session with confirmation
- Status bar counts are correct
- TUI restart reconnects existing tmux sessions
- `?` opens help overlay
- `;e` opens CLAUDE.md in $EDITOR
- Works correctly with 3+ concurrent sessions

---

## Iteration 2 — Session Management Quality

- Modal keyboard model: Tab focuses sidebar (command mode) / session (passthrough mode)
- `;` leader key removed; all commands accessed via sidebar
- Parked sessions with optional notes; PARKED section in sidebar
- Rename any session in place (`r`)
- Prompt editor (`N`) with multi-line textarea; Ctrl+S to submit
- Auto-naming via `claude -p` for sessions created with prompt editor
- statusMsg expires after 10s; redundant EnterAltScreen removed; config.Dir() propagates error
