# Inbox & Capture Design

## Goal

Add a lightweight capture buffer to claudetop so thoughts, tasks, and prompts can be saved instantly and turned into Claude sessions with a single keystroke.

## Architecture

A new `InboxItem` type is persisted in `state.json` alongside sessions. Two new overlays are added to the existing overlay system: a capture overlay (single-line text input) and an inbox view overlay (navigable list with actions). The "start session" action reuses the existing tmux send-keys mechanism to transfer item content to a new Claude session.

## Data Model

```go
// InboxItem in internal/state/store.go
type InboxItem struct {
    ID      string    `json:"id"`
    Content string    `json:"content"`
    Source  string    `json:"source"`   // "manual" for now
    AddedAt time.Time `json:"added_at"`
    Parked  bool      `json:"parked"`
}
```

`State` gains `InboxItems []*InboxItem`. Active items = not parked. Persisted to same `state.json`.

## UI: Capture Overlay (`c` in sidebar mode)

```
╭──────────────────────────────────────╮
│ Capture                              │
│                                      │
│ > _                                  │
│                                      │
│ Enter: save   Esc: cancel            │
╰──────────────────────────────────────╯
```

- `bubbles/textinput` for single-line input
- `Enter`: creates InboxItem, closes overlay, shows "Captured!" status message
- `Esc`: cancels, discards input

## UI: Inbox View Overlay (`b` in sidebar mode)

```
╭─────────────────────────────────────────────────────╮
│ INBOX (3 items)                                     │
│                                                     │
│ ▶ 1  Review PR feedback from yesterday             │
│   2  Check deployment status for staging           │
│   3  Refactor auth module                          │
│                                                     │
│ s: start session   d: dismiss   p: park   Esc: close│
╰─────────────────────────────────────────────────────╯
```

- `j`/`k`: navigate items
- `s`: spawn new Claude session, send item content via tmux send-keys after 2s delay, dismiss item, close overlay
- `d`: remove item entirely
- `p`: park item (hide from active list)
- `Esc`: close overlay
- Empty state: "(inbox empty)" + hint to press `c` to capture

## UI: Status Bar Indicator

When active inbox items exist, status bar shows `[INBOX: N]` in amber (color 214) before the session info.

## "Start Session" Flow

1. Generate session ID, call `tmux.Create()` (spawns `claude`)
2. Set a 2s one-shot timer (tick-based, non-blocking) storing pending send: session ID + item content
3. On timer fire: `tmux.SendLiteralKey(id, content)` + `tmux.SendKeys(id, "Enter")`
4. Dismiss item from inbox
5. Switch viewport to new session, close inbox overlay

## Keyboard Bindings (sidebar mode)

| Key | Action |
|-----|--------|
| `c` | Open capture overlay |
| `b` | Open inbox view overlay |

## Out of Scope

- GitHub issue creation (iteration 6)
- Slack URL fetching (iteration 7)
- Unpark action from inbox view (TBD)
- Multiple sources beyond "manual"
