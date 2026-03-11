# Morning Briefing Design

*Iteration 5 — claudetop*
*Date: 2026-03-11*

---

## Goal

Show a morning context screen on first daily launch. Surfaces yesterday's git activity,
inbox items, parked sessions, and a priorities input. Standup draft is generated on
demand with `s` and streams into the screen while you type priorities in parallel.

---

## Screen Layout

Full-screen view that replaces the session view on first open each day. Scrollable
content area in the top portion, priorities input pinned to the bottom.

```
┌─ GOOD MORNING ──────────────────────────────────────────────────────┐
│  Tuesday 11 March                                                    │
│                                                                      │
│  YESTERDAY                                                           │
│  ────────────────────────────────────────────────────────────────    │
│  annotation-service · fix timeout in pipeline (3h ago)              │
│  annotation-service · add retry logic to connector (5h ago)         │
│                                                                      │
│  INBOX  3 items                                                      │
│  ────────────────────────────────────────────────────────────────    │
│  · "can you check why annotation is timing out" — Erik               │
│  · "review PR #892 when you have a moment" — Maja                    │
│  + 1 more · b to open inbox                                          │
│                                                                      │
│  PARKED                                                              │
│  ────────────────────────────────────────────────────────────────    │
│  · milvus-vectors  "waiting for Björn to reply"                      │
│                                                                      │
│  STANDUP DRAFT                              s to generate            │
│  ────────────────────────────────────────────────────────────────    │
│  (not yet generated)                                                 │
│                                                                      │
├──────────────────────────────────────────────────────────────────────│
│  TODAY'S PRIORITIES                                                  │
│  > _                                                                 │
│                                                                      │
│  Tab: focus here · Enter: save and continue · Esc: skip              │
└──────────────────────────────────────────────────────────────────────┘
```

---

## Sections

### Header
Date formatted as "Tuesday 11 March". No year.

### Yesterday
Runs `git log --since=midnight-yesterday --until=midnight-today --oneline` across
each subdirectory under `root_dir` that is a git repo (detected by `.git` dir
presence). Shows `repo-name · commit message (Xh ago)`. Truncated to fit width.
If no commits, shows "(no commits yesterday)".

Discovery is one level deep — direct subdirectories of root_dir. Scanned at startup,
runs in a goroutine, results sent back as a tea.Msg.

### Inbox
Shows first 2 items inline (first ~60 chars of content + source name). If more than 2,
shows `+ N more`. Shows `b to open inbox` as a hint. `b` opens the inbox overlay on
top of the briefing; closing inbox returns to briefing.

If inbox is empty, section is omitted entirely.

### Parked
Lists parked sessions: name + park note (if set). If none, section is omitted.

### Standup Draft
Before generating: shows `(not yet generated)`.
After `s` is pressed: streams content from the standup skill into this section.
Uses the same mechanism as output-mode skills (spawn tmux session, send skill command,
poll CapturePane on ticker, update section content). The standup skill key is looked
up from config — whichever skill has `run_on_briefing = true`, or falls back to the
first skill with "standup" in the name, or allows any skill key to be pressed.

Actually, simpler: `s` is hardcoded to the standup skill for now. Config can add
`run_on_briefing = true` in a future iteration.

The standup section has a max display height of 8 lines. If output is longer, the
full briefing content scrolls with j/k.

`s` can be pressed again to regenerate (kills the previous session and starts a new one).

`s` only works when the priorities text input is NOT focused. When input is focused,
`s` types the character 's'.

### Today's Priorities
A single-line text input pinned to the bottom of the screen (below a horizontal
divider). Not focused by default — Tab moves focus to it. When focused, all
keystrokes go to the input including `s`, `b`, `j`, `k`.

On Enter (when input is focused): saves content to `~/.claudetop/today.md` and
closes the briefing. On Esc (regardless of focus): closes without saving.

`~/.claudetop/today.md` format:
```markdown
# Priorities — 2026-03-11

1. Resolve annotation pipeline timeout
2. Move CSV export forward
3. Process inbox
```

Each line of the input is treated as one priority item. The file is overwritten
each time.

---

## Trigger Logic

On startup, after loading state, claudetop checks:
- `state.LastBriefingDate` (new field, `string` in `YYYY-MM-DD` format)
- If it does not match today's date AND `auto_briefing = true` in config (new field,
  defaults to `true`), show the briefing before the session view

After briefing is dismissed (Enter or Esc), update `state.LastBriefingDate = today`
and save state.

The briefing can be reopened manually with `;B` from anywhere.

---

## Navigation

| Key | Action |
|-----|--------|
| `j` / `↓` | Scroll content up (when input not focused) |
| `k` / `↑` | Scroll content down (when input not focused) |
| `s` | Generate / regenerate standup draft (when input not focused) |
| `b` | Open inbox overlay |
| `Tab` | Focus priorities text input |
| `Enter` | Save priorities + close briefing (when input focused) |
| `Esc` | Close briefing without saving priorities |
| `;B` | Reopen briefing from anywhere |

---

## Config Changes

Add to `GeneralConfig`:
```toml
[general]
auto_briefing = true   # show briefing on first open each day (default true)
```

Add to `State`:
```json
{ "last_briefing_date": "2026-03-11" }
```

---

## State Changes

- `State.LastBriefingDate string` — stores `"YYYY-MM-DD"` of last completed briefing.
  Empty string means never shown.

---

## Out of Scope (This Iteration)

- Priorities injection into new sessions (today.md written but not injected yet)
- `run_on_briefing` config flag on skills
- Session history tracking (closed sessions not shown — only git commits)
- Multi-line priorities editor (single-line textinput is enough)
