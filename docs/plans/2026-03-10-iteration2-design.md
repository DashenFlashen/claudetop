# claudetop Iteration 2 — Design

*Date: 2026-03-10*
*Scope: Session management quality*

## What We're Building

Six focused improvements to session management:

1. **Modal keyboard model** — Tab-based focus switching replaces the `;` leader key
2. **Parked sessions** — park a session with a note, unpark when ready
3. **Rename** — rename any session in place
4. **Prompt editor** — `N` opens a multi-line editor to write the initial prompt before spawning
5. **Auto-naming** — sessions spawned via `N` are named from the prompt using `claude -p`
6. **Code review cleanup** — three small carries from the iteration 1 review

Split view and repo tagging are explicitly deferred.

---

## 1. Keyboard Model

### Two focus states

**Session mode** (default): everything passes through to Claude Code. No exceptions.

**Sidebar mode**: sidebar has a bright cyan left border. All TUI commands work here. Claude Code gets nothing.

### Key routing

| Key | In session mode | In sidebar mode |
|-----|----------------|-----------------|
| `Tab` | → focus sidebar | → focus session |
| `\` | → toggle sidebar visible/hidden | → toggle sidebar visible/hidden |
| `Esc` | passes through | → return to session mode |
| Everything else | passes through | TUI command |

### Sidebar commands (when sidebar is focused)

| Key | Action |
|-----|--------|
| `j` / `k` / `↑` / `↓` | Navigate session list |
| `1`–`9` | Jump to session by number |
| `Enter` | Focus selected session (return to session mode) |
| `n` | New blank session |
| `N` | New session with prompt editor |
| `x` | Close highlighted session (with confirmation) |
| `p` | Park highlighted session |
| `u` | Unpark highlighted session (PARKED section only) |
| `r` | Rename highlighted session |
| `?` | Help overlay |
| `q` | Quit (sessions keep running) |
| `Q` | Quit and kill all sessions |
| `e` | Edit root CLAUDE.md in $EDITOR |

### Visual indicator

Status bar hint line changes with focus:
- Session mode: `Tab: sidebar   \ toggle   n new`
- Sidebar mode: `Tab: session   j/k navigate   n new   x close   q quit`

Sidebar left border: cyan (`#00afff`) when focused, invisible when not.

### Migration from `;` leader

The `;` leader key is removed. All commands are now sidebar-only. `\` remains as a secondary way to show/hide the sidebar without taking focus (useful for glancing at session status).

---

## 2. Parked Sessions

### Data model

Two new persisted fields on `Session`:

```go
Parked   bool   `json:"parked,omitempty"`
ParkNote string `json:"park_note,omitempty"`
```

### Sidebar layout

```
ACTIVE
 1 ⠹ annotation-inv
 2 ● csv-export

PARKED
 3 ○ milvus-vec
   "waiting for Björn"
```

- Parked sessions use `○` (hollow dot, grey)
- Park note renders on the line below, indented, dimmed
- PARKED section header only shown when at least one session is parked

### Behaviour

- `p` while a session is highlighted → park note overlay (single-line text input, optional). Enter to confirm, Esc to cancel.
- Session moves to PARKED section. Claude Code process keeps running.
- `u` while a PARKED session is highlighted → unpark immediately, move to ACTIVE.
- Parked sessions are still polled for status — they can transition to NeedsInput while parked.
- Dead sessions remain separate from parked sessions (different field, different display).

---

## 3. Rename

`r` while a session is highlighted → text input overlay pre-filled with the current name.

- Enter to confirm, Esc to cancel.
- Updates `session.Name`, saves state.
- tmux session ID is unchanged (stable ID was the whole point of the ID/Name decoupling from iteration 1).

---

## 4. Prompt Editor (`N`)

A multi-line text area for writing the full initial prompt before spawning a session.

- `bubbles/textarea` component
- `Ctrl+S` to submit, `Esc` to cancel
- On submit: session spawns as `session-N` immediately, prompt is sent to the tmux session via `send-keys`
- Auto-naming fires in background (see Section 5)

The existing `n` flow (single-line name input) is unchanged.

---

## 5. Auto-naming

Fires only from the `N` (prompt editor) flow, where the full prompt is known before spawn.

**Mechanism:**

```
claude -p "Summarize this task in 3 words, lowercase, hyphen-separated,
no punctuation. Output only the name, nothing else: {prompt}"
```

Runs as a `tea.Cmd` immediately after session spawn. When it returns, a `sessionRenamedMsg{sessionID, name}` updates the name and saves state. The session shows as `session-N` until the name arrives (~2–3s).

**Error handling:** best-effort. If `claude` is not in PATH, times out, or returns garbage, the session keeps its `session-N` name silently.

**Config:**

```toml
[general]
auto_name_sessions = true   # default true; set false to skip the claude -p call
```

Blank sessions (`n`) are never auto-named — they keep their name until manually renamed with `r`.

---

## 6. Code Review Cleanup

Three items carried from the iteration 1 code review:

**`statusMsg` expiry** — error messages in the status bar persist until cleared. Add a `statusMsgAt time.Time` field; clear `statusMsg` in the tick handler after 10 seconds.

**Redundant `tea.EnterAltScreen`** — `Init()` returns `tea.EnterAltScreen` but `NewProgram` already has `tea.WithAltScreen()`. Remove from `Init()`.

**`config.Dir()` error propagation** — currently panics on `os.UserHomeDir()` failure. Change signature to `Dir() (string, error)` and propagate through all callers.

---

## Components Affected

| Component | Change |
|-----------|--------|
| `internal/config/config.go` | Add `AutoNameSessions bool`; `Dir()` returns error |
| `internal/session/session.go` | Add `Parked`, `ParkNote` fields |
| `internal/ui/app.go` | New keyboard model, new message types, new overlays, statusMsg expiry |
| `internal/ui/sidebar.go` | PARKED section, `○` dot, park note line, focused border |
| `internal/ui/help.go` | Updated keybindings |
| `internal/ui/park.go` | New: park note overlay |
| `internal/ui/rename.go` | New: rename overlay |
| `internal/ui/prompteditor.go` | New: multi-line prompt editor overlay |

---

## Success Criteria

- All Claude Code sessions can be named meaningfully (auto or manual)
- Sessions can be parked with a note and unparked without losing work
- Tab switches focus cleanly; no keypress leaks to Claude when sidebar is focused
- `N` flow spawns session and auto-names it within ~3 seconds
- No regression in existing iteration 1 behaviour
