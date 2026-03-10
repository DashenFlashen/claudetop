# Inbox & Capture Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a lightweight inbox buffer to claudetop: capture text notes with `c`, browse and act on them with `b`, and launch Claude sessions pre-loaded with the note content.

**Architecture:** `InboxItem` is persisted in `state.json` alongside sessions. Two new overlays (`overlayCapture` and `overlayInbox`) follow the exact pattern of existing overlays (textinput + render function in dedicated file). "Start session from inbox" reuses `spawnSession` and fires a deferred tmux send-keys via a pending-send field on the model, so the session appears immediately but content arrives 2 seconds later when Claude has started.

**Tech Stack:** Go, Bubbletea (bubbles/textinput already imported), lipgloss, existing tmux package.

---

## Codebase orientation

Before starting, read these files:
- `internal/state/store.go` — State struct, Load, Save
- `internal/state/store_test.go` — test patterns (t.Setenv HOME, TempDir)
- `internal/ui/app.go` — Model struct, overlay enum, handleSidebarKey, View()
- `internal/ui/newsession.go` — overlay render function pattern to copy
- `internal/ui/statusbar.go` — renderStatusBar signature
- `internal/ui/help.go` — helpEntries slice

---

## Task 1: InboxItem data model

**Files:**
- Modify: `internal/state/store.go`
- Modify: `internal/state/store_test.go`

### Step 1: Add InboxItem type and update State

In `internal/state/store.go`, add after the imports block (before `State`):

```go
// InboxItem is a captured note in the inbox.
type InboxItem struct {
	ID      string    `json:"id"`
	Content string    `json:"content"`
	Source  string    `json:"source"`
	AddedAt time.Time `json:"added_at"`
	Parked  bool      `json:"parked,omitempty"`
}
```

Add `"fmt"` and `"time"` to imports (they're already present — `fmt` is there, add `time`).

Update `State`:
```go
type State struct {
	Sessions   []*session.Session `json:"sessions"`
	InboxItems []*InboxItem       `json:"inbox_items,omitempty"`
}
```

Update the `Load()` nil-check block (after `s.Sessions == nil` check):
```go
	if s.Sessions == nil {
		s.Sessions = []*session.Session{}
	}
	if s.InboxItems == nil {
		s.InboxItems = []*InboxItem{}
	}
```

Add a constructor for InboxItem (after `NewSession` in session.go — no, add it here in store.go):
```go
// NewInboxItem creates a new inbox item with a unique ID.
func NewInboxItem(content, source string) *InboxItem {
	return &InboxItem{
		ID:      fmt.Sprintf("%x", time.Now().UnixNano()),
		Content: content,
		Source:  source,
		AddedAt: time.Now(),
	}
}
```

### Step 2: Write failing test

In `internal/state/store_test.go`, add:

```go
func TestInboxItemSaveAndLoad(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	os.MkdirAll(filepath.Join(tmp, ".claudetop"), 0755)

	item := NewInboxItem("fix the thing", "manual")
	s := &State{
		Sessions:   []*session.Session{},
		InboxItems: []*InboxItem{item},
	}

	if err := Save(s); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(loaded.InboxItems) != 1 {
		t.Fatalf("expected 1 inbox item, got %d", len(loaded.InboxItems))
	}
	if loaded.InboxItems[0].Content != "fix the thing" {
		t.Errorf("expected Content=%q, got %q", "fix the thing", loaded.InboxItems[0].Content)
	}
	if loaded.InboxItems[0].Source != "manual" {
		t.Errorf("expected Source=%q, got %q", "manual", loaded.InboxItems[0].Source)
	}
}

func TestLoadEmptyInboxWhenMissing(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	s, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if s.InboxItems == nil {
		t.Error("expected InboxItems to be non-nil slice")
	}
	if len(s.InboxItems) != 0 {
		t.Errorf("expected 0 inbox items, got %d", len(s.InboxItems))
	}
}
```

### Step 3: Run test to verify it fails

```
go test ./internal/state/... -v -run TestInbox
```

Expected: FAIL (InboxItem undefined, field InboxItems not in State)

### Step 4: Implement (Step 1 above)

### Step 5: Run tests again

```
go test ./internal/state/... -v
```

Expected: all PASS

### Step 6: Commit

```bash
git add internal/state/store.go internal/state/store_test.go
git commit -m "feat: add InboxItem data model with persistence"
```

---

## Task 2: Capture overlay

**Files:**
- Create: `internal/ui/capture.go`
- Modify: `internal/ui/app.go`

### Step 1: Create the capture overlay render file

Create `internal/ui/capture.go`:

```go
package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

var (
	captureOverlayStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("220")).
				Background(lipgloss.Color("235")).
				Padding(1, 2).
				Width(60)

	captureTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")).
				Bold(true)

	captureHintStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244"))
)

// newCaptureInput creates a configured text input for inbox capture.
func newCaptureInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "what's on your mind?"
	ti.CharLimit = 200
	ti.Width = 54
	ti.Focus()
	return ti
}

// renderCapture renders the capture overlay centered in the terminal.
func renderCapture(input textinput.Model, width, height int) string {
	var lines []string
	lines = append(lines, captureTitleStyle.Render("Capture"))
	lines = append(lines, "")
	lines = append(lines, input.View())
	lines = append(lines, "")
	lines = append(lines, captureHintStyle.Render("Enter: save   Esc: cancel"))

	content := captureOverlayStyle.Render(strings.Join(lines, "\n"))

	contentWidth := lipgloss.Width(content)
	contentHeight := lipgloss.Height(content)

	leftPad := (width - contentWidth) / 2
	topPad := (height - contentHeight) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	if topPad < 0 {
		topPad = 0
	}

	var result []string
	for i := 0; i < topPad; i++ {
		result = append(result, strings.Repeat(" ", width))
	}
	for _, line := range strings.Split(content, "\n") {
		result = append(result, strings.Repeat(" ", leftPad)+line)
	}
	return strings.Join(result, "\n")
}
```

### Step 2: Update app.go — overlay enum, Model fields, handlers

**2a. Add overlay constants** (after `overlaySkillOutput` at line ~64):

```go
	overlayCapture
	overlayInbox
```

**2b. Add Model fields** (after `skillVP viewport.Model` field, before `width int`):

```go
	captureInput textinput.Model
	inboxCursor  int
```

**2c. Add `handleCaptureKey` method** (add after `handleSkillOutputKey`):

```go
func (m *Model) handleCaptureKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		content := strings.TrimSpace(m.captureInput.Value())
		if content != "" {
			item := state.NewInboxItem(content, "manual")
			m.store.InboxItems = append(m.store.InboxItems, item)
			m.saveState()
			m.setStatusMsg("Captured!")
		}
		m.overlay = overlayNone
		return m, nil
	case tea.KeyEsc:
		m.overlay = overlayNone
		return m, nil
	}

	var cmd tea.Cmd
	m.captureInput, cmd = m.captureInput.Update(msg)
	return m, cmd
}
```

**2d. Wire into `handleKey`** — add case to overlay switch (after `overlaySkillOutput` case at line ~293):

```go
	case overlayCapture:
		return m.handleCaptureKey(msg)
	case overlayInbox:
		return m.handleInboxKey(msg)
```

**2e. Add `c` case in `handleSidebarKey`** default switch, before the skill-check loop:

```go
	case "c":
		m.overlay = overlayCapture
		m.captureInput = newCaptureInput()
		return m, textinput.Blink
```

**2f. Add `overlayCapture` and `overlayInbox` cases in `View()` overlay switch** (after `overlaySkillOutput` case):

```go
	case overlayCapture:
		return renderCapture(m.captureInput, m.width, m.height)
	case overlayInbox:
		return renderInbox(m.store.InboxItems, m.inboxCursor, m.width, m.height)
```

### Step 3: Run tests and build

```
go build ./...
go test ./internal/state/... -v
```

Expected: builds cleanly, state tests all pass.

### Step 4: Commit

```bash
git add internal/ui/capture.go internal/ui/app.go
git commit -m "feat: add capture overlay (c in sidebar)"
```

---

## Task 3: Inbox view overlay

**Files:**
- Create: `internal/ui/inbox.go`
- Modify: `internal/ui/app.go`

### Step 1: Create the inbox overlay render file

Create `internal/ui/inbox.go`:

```go
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"claudetop/internal/state"
)

var (
	inboxOverlayStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("220")).
				Background(lipgloss.Color("235")).
				Padding(1, 2)

	inboxTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Bold(true)

	inboxItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	inboxCursorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("220")).
				Bold(true)

	inboxHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	inboxEmptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
)

// activeInboxItems returns only non-parked items.
func activeInboxItems(items []*state.InboxItem) []*state.InboxItem {
	var active []*state.InboxItem
	for _, item := range items {
		if !item.Parked {
			active = append(active, item)
		}
	}
	return active
}

// renderInbox renders the inbox view overlay centered in the terminal.
func renderInbox(items []*state.InboxItem, cursor int, width, height int) string {
	active := activeInboxItems(items)

	// Compute inner content width (overlay will be ~80% of terminal width, min 50)
	overlayWidth := width * 4 / 5
	if overlayWidth < 50 {
		overlayWidth = 50
	}
	if overlayWidth > 100 {
		overlayWidth = 100
	}
	innerWidth := overlayWidth - 6 // border(2) + padding(4)

	var lines []string
	title := fmt.Sprintf("INBOX (%d items)", len(active))
	lines = append(lines, inboxTitleStyle.Width(innerWidth).Render(title))
	lines = append(lines, "")

	if len(active) == 0 {
		lines = append(lines, inboxEmptyStyle.Render("(inbox empty)"))
		lines = append(lines, "")
		lines = append(lines, inboxHintStyle.Render("c: capture   Esc: close"))
	} else {
		for i, item := range active {
			prefix := "  "
			lineStyle := inboxItemStyle
			if i == cursor {
				prefix = "▶ "
				lineStyle = inboxCursorStyle
			}
			// Truncate content to fit
			content := item.Content
			maxContent := innerWidth - 5 // room for "N  " prefix
			if len([]rune(content)) > maxContent {
				runes := []rune(content)
				content = string(runes[:maxContent-1]) + "…"
			}
			line := fmt.Sprintf("%s%d  %s", prefix, i+1, content)
			lines = append(lines, lineStyle.Width(innerWidth).Render(line))
		}
		lines = append(lines, "")
		lines = append(lines, inboxHintStyle.Render("j/k: navigate   s: start session   d: dismiss   p: park   Esc: close"))
	}

	content := inboxOverlayStyle.Width(overlayWidth).Render(strings.Join(lines, "\n"))

	contentWidth := lipgloss.Width(content)
	contentHeight := lipgloss.Height(content)

	leftPad := (width - contentWidth) / 2
	topPad := (height - contentHeight) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	if topPad < 0 {
		topPad = 0
	}

	var result []string
	for i := 0; i < topPad; i++ {
		result = append(result, strings.Repeat(" ", width))
	}
	for _, line := range strings.Split(content, "\n") {
		result = append(result, strings.Repeat(" ", leftPad)+line)
	}
	return strings.Join(result, "\n")
}
```

### Step 2: Add handleInboxKey and pendingInboxSend to app.go

**2a. Add new message types** (after `skillOutputMsg` at line ~51):

```go
type inboxSessionSpawnedMsg struct {
	sess    *session.Session
	content string
}
```

**2b. Add pendingInboxSend struct and field** — add after `skillVP` fields in Model:

```go
	pendingInboxSend *pendingInboxSend
```

And add this struct definition near the message types at the top of app.go:

```go
// pendingInboxSend holds a scheduled tmux content send for an inbox-spawned session.
type pendingInboxSend struct {
	sessionID string
	content   string
	sendAt    time.Time
}
```

**2c. Handle `inboxSessionSpawnedMsg` in Update()** — add after `sessionSpawnedMsg` case:

```go
	case inboxSessionSpawnedMsg:
		m.statusMsg = ""
		m.sessions = append(m.sessions, msg.sess)
		m.store.Sessions = m.sessions
		m.saveState()
		m.switchSession(len(m.sessions) - 1)
		m.sidebarCursor = len(m.sessions) - 1
		m.sidebarFocused = false
		m.pendingInboxSend = &pendingInboxSend{
			sessionID: msg.sess.ID,
			content:   msg.content,
			sendAt:    time.Now().Add(2 * time.Second),
		}
		return m, nil
```

**2d. Fire pending send in `tickMsg` handler** — add at the end of the tick handler cmds slice, before `return m, tea.Batch(cmds...)`:

```go
		// Fire deferred inbox content send when Claude has had time to start
		if m.pendingInboxSend != nil && time.Now().After(m.pendingInboxSend.sendAt) {
			send := m.pendingInboxSend
			m.pendingInboxSend = nil
			cmds = append(cmds, sendInboxContent(send.sessionID, send.content))
		}
```

**2e. Add `sendInboxContent` command** (alongside `spawnSession`, `closeSession` etc.):

```go
// sendInboxContent sends captured text to a tmux session as if typed by the user.
func sendInboxContent(sessionID, content string) tea.Cmd {
	return func() tea.Msg {
		tmux.SendLiteralKey(sessionID, content)
		tmux.SendKeys(sessionID, "Enter")
		return nil
	}
}

// spawnSessionForInbox creates a new session and returns inboxSessionSpawnedMsg so the
// model can schedule the deferred content send.
func spawnSessionForInbox(name, rootDir, content string, sessionCount int) tea.Cmd {
	return func() tea.Msg {
		s := session.NewSession(name, sessionCount+1)
		if _, err := os.Stat(rootDir); err != nil {
			return errMsg{fmt.Errorf("root_dir %q does not exist: %w", rootDir, err)}
		}
		if err := tmux.Create(s.ID, rootDir); err != nil {
			return errMsg{fmt.Errorf("create session: %w", err)}
		}
		return inboxSessionSpawnedMsg{sess: s, content: content}
	}
}
```

**2f. Add `handleInboxKey` method** (after `handleCaptureKey`):

```go
func (m *Model) handleInboxKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	active := activeInboxItems(m.store.InboxItems)

	switch msg.Type {
	case tea.KeyEsc:
		m.overlay = overlayNone
		return m, nil
	}

	switch msg.String() {
	case "j":
		if len(active) > 0 {
			m.inboxCursor = (m.inboxCursor + 1) % len(active)
		}
	case "k":
		if len(active) > 0 {
			m.inboxCursor = (m.inboxCursor - 1 + len(active)) % len(active)
		}
	case "s":
		if m.inboxCursor < len(active) {
			item := active[m.inboxCursor]
			// Remove from inbox
			m.removeInboxItem(item.ID)
			m.overlay = overlayNone
			name := m.uniqueName("inbox")
			return m, spawnSessionForInbox(name, m.cfg.General.RootDir, item.Content, len(m.sessions))
		}
	case "d":
		if m.inboxCursor < len(active) {
			m.removeInboxItem(active[m.inboxCursor].ID)
			// Clamp cursor
			newActive := activeInboxItems(m.store.InboxItems)
			if m.inboxCursor >= len(newActive) && m.inboxCursor > 0 {
				m.inboxCursor--
			}
		}
	case "p":
		if m.inboxCursor < len(active) {
			active[m.inboxCursor].Parked = true
			m.saveState()
			// Clamp cursor
			newActive := activeInboxItems(m.store.InboxItems)
			if m.inboxCursor >= len(newActive) && m.inboxCursor > 0 {
				m.inboxCursor--
			}
		}
	}
	return m, nil
}

// removeInboxItem removes an inbox item by ID and saves state.
func (m *Model) removeInboxItem(id string) {
	items := m.store.InboxItems
	for i, item := range items {
		if item.ID == id {
			m.store.InboxItems = append(items[:i], items[i+1:]...)
			m.saveState()
			return
		}
	}
}
```

**2g. Add `b` case in `handleSidebarKey`** (alongside `c`):

```go
	case "b":
		m.inboxCursor = 0
		m.overlay = overlayInbox
```

### Step 3: Build

```
go build ./...
```

Expected: compiles cleanly.

### Step 4: Commit

```bash
git add internal/ui/inbox.go internal/ui/app.go
git commit -m "feat: add inbox view overlay with session launch"
```

---

## Task 4: Status bar inbox indicator

**Files:**
- Modify: `internal/ui/statusbar.go`
- Modify: `internal/ui/app.go` (call site only)

### Step 1: Update renderStatusBar signature

In `internal/ui/statusbar.go`, change signature:

```go
func renderStatusBar(sessions []*session.Session, inboxCount int, width int, statusMsg string) string {
```

Add inbox badge after the `needsInput` block, before the `statusMsg` block:

```go
	inboxBadgeStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("214"))

	if inboxCount > 0 {
		left += "  " + inboxBadgeStyle.Render(fmt.Sprintf("[INBOX: %d]", inboxCount))
	}
```

### Step 2: Update call site in app.go View()

Find the `renderStatusBar` call in `View()`:

```go
	statusBar := renderStatusBar(m.sessions, m.width, m.statusMsg)
```

Change to:

```go
	activeInbox := len(activeInboxItems(m.store.InboxItems))
	statusBar := renderStatusBar(m.sessions, activeInbox, m.width, m.statusMsg)
```

Note: `activeInboxItems` is defined in `inbox.go` in the same package, so it's accessible here.

### Step 3: Build and run

```
go build ./...
go test ./...
```

Expected: all pass.

### Step 4: Commit

```bash
git add internal/ui/statusbar.go internal/ui/app.go
git commit -m "feat: show inbox count in status bar"
```

---

## Task 5: Help and hint bar updates

**Files:**
- Modify: `internal/ui/help.go`
- Modify: `internal/ui/app.go` (hint text only)

### Step 1: Add inbox entries to help

In `internal/ui/help.go`, in `helpEntries`, add after the `p`/`u` entries (find the blank line before `"q"` entry):

```go
	{"c", "Capture note to inbox"},
	{"b", "Open inbox"},
	{"", ""},
```

### Step 2: Update sidebar hint bar text

In `internal/ui/app.go`, in `View()`, find:

```go
		hintText = "  Esc/Tab: session   j/k: navigate   n new   x close   r rename   R auto-name   ? help"
```

Change to:

```go
		hintText = "  Esc/Tab: session   j/k: navigate   n new   x close   r rename   R auto-name   c capture   b inbox   ? help"
```

### Step 3: Build and run all tests

```
go build ./...
go test ./...
```

Expected: all pass.

### Step 4: Commit

```bash
git add internal/ui/help.go internal/ui/app.go
git commit -m "feat: update help and hint bar with inbox shortcuts"
```

---

## Manual smoke test checklist

Run `go run ./cmd/claudetop` and verify:

1. **Capture**: Tab → `c` → type text → Enter → status shows "Captured!" → `[INBOX: 1]` in status bar
2. **Capture cancel**: Tab → `c` → Esc → no item added
3. **Inbox view**: Tab → `b` → see item list with cursor → j/k navigate
4. **Dismiss**: Tab → `b` → `d` → item removed, status bar updates
5. **Park**: Tab → `b` → `p` → item disappears from active list
6. **Start session**: Tab → `b` → `s` → new session appears, 2 seconds later item content is sent to Claude
7. **Persistence**: quit and restart → inbox items still present
8. **Empty inbox**: Tab → `b` with no items → "(inbox empty)" message
