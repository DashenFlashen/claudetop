# claudetop Iteration 2 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add session management quality features: Tab-based modal keyboard model, parked sessions, rename, prompt editor, and auto-naming.

**Architecture:** The keyboard model switches from a `;` leader key to a `sidebarFocused` boolean in the Model — when true, all keys route to sidebar commands; when false, all keys pass through to tmux. New overlays (park, rename, prompt editor) follow the same pattern as existing overlays. Auto-naming runs as a pure `tea.Cmd` that shells out to `claude -p`.

**Tech Stack:** Go, Bubbletea, bubbles (textinput, textarea, viewport), lipgloss. `claude` CLI must be in PATH for auto-naming.

---

## Context for the implementer

Read these files before starting:
- `docs/plans/2026-03-10-iteration2-design.md` — approved design
- `internal/ui/app.go` — main Bubbletea model (understand the overlay enum, message types, handleKey, handleSidebarKey)
- `internal/ui/sidebar.go` — sidebar render function
- `internal/session/session.go` — Session struct
- `internal/config/config.go` — Config struct and Dir()

The project uses Go modules. Run `go build -o claudetop .` from the repo root to verify a clean build. Run `go test ./...` for tests. All tests must pass after every task.

The overlay pattern: `app.go` has an `overlay int` enum. New overlays are added to this enum, handled in `handleKey` at the top (overlay handlers get priority), rendered in `View()`, and managed in the model by setting `m.overlay = overlayXxx`.

---

### Task 1: Config — AutoNameSessions field + Dir() error propagation

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/state/store.go`
- Modify: `main.go`
- Test: `internal/config/config_test.go`

**Step 1: Add `AutoNameSessions` to config struct**

In `internal/config/config.go`, update `GeneralConfig`:

```go
type GeneralConfig struct {
	RootDir          string `toml:"root_dir"`
	AutoNameSessions bool   `toml:"auto_name_sessions"`
}
```

Update the default config in `Load()` to set `AutoNameSessions: true`:

```go
cfg := &Config{
	General: GeneralConfig{
		RootDir:          home,
		AutoNameSessions: true,
	},
}
```

**Step 2: Change Dir() to return (string, error)**

Replace the current `Dir()` (which panics) with one that returns an error:

```go
// Dir returns the ~/.claudetop directory path.
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home directory: %w", err)
	}
	return filepath.Join(home, ".claudetop"), nil
}
```

**Step 3: Update Path() and EnsureDir() to handle the error**

```go
// Path returns the config file path.
func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

// EnsureDir creates ~/.claudetop if it doesn't exist.
func EnsureDir() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0755)
}
```

Update `Load()` to use the new `Path()`:

```go
func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("home dir: %w", err)
	}

	cfg := &Config{
		General: GeneralConfig{
			RootDir:          home,
			AutoNameSessions: true,
		},
	}

	path, err := Path()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	return cfg, nil
}
```

**Step 4: Update state/store.go to handle Dir() error**

In `internal/state/store.go`, replace `path()`:

```go
func path() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "state.json"), nil
}
```

Update `Load()` and `Save()` to call `path()` with error handling:

```go
func Load() (*State, error) {
	p, err := path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return &State{Sessions: []*session.Session{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read state: %w", err)
	}

	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}
	if s.Sessions == nil {
		s.Sessions = []*session.Session{}
	}
	return &s, nil
}

func Save(s *State) error {
	p, err := path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("write state: %w", err)
	}
	return os.Rename(tmp, p)
}
```

**Step 5: Update config_test.go — add AutoNameSessions assertion**

In `TestLoadDefaultsWhenMissing`, add:

```go
if !cfg.General.AutoNameSessions {
	t.Error("expected AutoNameSessions=true by default")
}
```

In `TestLoadFromFile`, add `auto_name_sessions = false` to the TOML and assert it:

```go
content := "[general]\nroot_dir = \"/tmp/repos\"\nauto_name_sessions = false\n"
// ...
if cfg.General.AutoNameSessions {
	t.Error("expected AutoNameSessions=false")
}
```

**Step 6: Run tests**

```
go test ./internal/config/... ./internal/state/...
```

Expected: PASS

**Step 7: Build to catch any missed Dir() callers**

```
go build -o claudetop .
```

Expected: clean build. Fix any compile errors from Dir()/Path() callers.

**Step 8: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go internal/state/store.go main.go
git commit -m "feat: AutoNameSessions config field; Dir() returns error instead of panicking"
```

---

### Task 2: Session model — Parked and ParkNote fields

**Files:**
- Modify: `internal/session/session.go`
- Test: `internal/state/store_test.go`

**Step 1: Add fields to Session struct**

In `internal/session/session.go`, add to the persisted fields block:

```go
type Session struct {
	// Persisted fields
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Dead      bool      `json:"dead,omitempty"`
	Parked    bool      `json:"parked,omitempty"`
	ParkNote  string    `json:"park_note,omitempty"`

	// Runtime-only (not persisted)
	Status       Status    `json:"-"`
	PaneContent  string    `json:"-"`
	LastOutputAt time.Time `json:"-"`
}
```

**Step 2: Write a failing test for Parked round-trip**

In `internal/state/store_test.go`, add:

```go
func TestSaveAndLoadParkedSession(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	os.MkdirAll(filepath.Join(tmp, ".claudetop"), 0755)

	sess := &session.Session{
		ID:        "parked-session",
		Name:      "milvus",
		CreatedAt: time.Now().Truncate(time.Second),
		Parked:    true,
		ParkNote:  "waiting for Björn",
	}
	s := &State{Sessions: []*session.Session{sess}}

	if err := Save(s); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if !loaded.Sessions[0].Parked {
		t.Error("expected Parked=true")
	}
	if loaded.Sessions[0].ParkNote != "waiting for Björn" {
		t.Errorf("expected ParkNote=%q, got %q", "waiting for Björn", loaded.Sessions[0].ParkNote)
	}
}
```

**Step 3: Run test to verify it fails**

```
go test ./internal/state/... -run TestSaveAndLoadParkedSession -v
```

Expected: FAIL (fields not on struct yet — but actually they are after Step 1, so this should pass)

**Step 4: Run all tests**

```
go test ./...
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/session/session.go internal/state/store_test.go
git commit -m "feat: add Parked and ParkNote fields to Session"
```

---

### Task 3: Code review cleanup — statusMsg expiry + remove redundant EnterAltScreen

**Files:**
- Modify: `internal/ui/app.go`
- Modify: `main.go` (already has `tea.WithAltScreen()` — verify)

**Step 1: Remove redundant `tea.EnterAltScreen` from `Init()`**

In `internal/ui/app.go`, find `Init()`:

```go
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,   // ← remove this line
		tickCmd(),
	)
}
```

Change to:

```go
func (m *Model) Init() tea.Cmd {
	return tickCmd()
}
```

Verify `main.go` has `tea.WithAltScreen()` in `NewProgram` call (it does: `tea.NewProgram(m, tea.WithAltScreen(), ...)`).

**Step 2: Add statusMsgAt field to Model**

In `internal/ui/app.go`, add to the `Model` struct:

```go
statusMsg   string    // transient message shown in status bar
statusMsgAt time.Time // when statusMsg was set (for expiry)
```

**Step 3: Add setStatusMsg helper**

Below the existing `saveState()` helper, add:

```go
// setStatusMsg sets a transient status bar message with an expiry timestamp.
func (m *Model) setStatusMsg(msg string) {
	m.statusMsg = msg
	m.statusMsgAt = time.Now()
}
```

**Step 4: Replace direct statusMsg assignments with setStatusMsg**

Find all places that set `m.statusMsg = "Error: ..."` (there are 3: in `sessionClosedMsg` handler, `errMsg` handler, and `saveState()`). Replace each with `m.setStatusMsg("Error: " + ...)`.

Leave `m.statusMsg = ""` (the clear in `sessionSpawnedMsg`) as a direct assignment — that's an intentional clear, not a timed message.

**Step 5: Add expiry check in tick handler**

In the `tickMsg` case in `Update`, add at the top of the handler (after `m.tick++`):

```go
const statusMsgTTL = 10 * time.Second
if m.statusMsg != "" && time.Since(m.statusMsgAt) > statusMsgTTL {
	m.statusMsg = ""
}
```

**Step 6: Build and test**

```
go build -o claudetop . && go test ./...
```

Expected: clean build, all tests pass.

**Step 7: Commit**

```bash
git add internal/ui/app.go
git commit -m "fix: remove redundant EnterAltScreen from Init; expire statusMsg after 10s"
```

---

### Task 4: Keyboard model overhaul — sidebarFocused, Tab routing, remove ; leader

This is the largest task. Read `internal/ui/app.go` completely before starting.

**Files:**
- Modify: `internal/ui/app.go`

**Step 1: Update Model struct**

Remove `leaderActive bool`. Add `sidebarFocused bool`. The struct becomes:

```go
type Model struct {
	cfg      *config.Config
	store    *state.State
	sessions []*session.Session

	activeIdx     int
	sidebarOpen   bool
	sidebarFocused bool   // true = sidebar has keyboard focus
	overlay       overlay
	tick          int
	statusMsg     string
	statusMsgAt   time.Time

	nameInput textinput.Model
	viewport  viewport.Model

	width  int
	height int
}
```

**Step 2: Update New() — initialize sidebarFocused**

If there are no live sessions on startup, start in sidebar mode (focused) so the user can press `n` immediately:

```go
func New(cfg *config.Config, st *state.State) *Model {
	m := &Model{
		cfg:         cfg,
		store:       st,
		sessions:    st.Sessions,
		activeIdx:   -1,
		sidebarOpen: true,
		nameInput:   newSessionInput(),
	}
	for i, s := range m.sessions {
		if !s.Dead {
			m.activeIdx = i
			break
		}
	}
	// Start in sidebar mode if no live sessions
	m.sidebarFocused = m.activeIdx < 0
	return m
}
```

**Step 3: Rewrite handleKey**

Replace the entire `handleKey` function with:

```go
func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Overlay handlers take priority
	switch m.overlay {
	case overlayHelp:
		if msg.Type == tea.KeyEsc || msg.String() == "?" {
			m.overlay = overlayNone
		}
		return m, nil
	case overlayNewSession:
		return m.handleNewSessionKey(msg)
	case overlayCloseConfirm:
		return m.handleCloseConfirmKey(msg)
	}

	// Tab: always switch sidebar focus when no overlay is open
	if msg.Type == tea.KeyTab {
		m.sidebarFocused = !m.sidebarFocused
		if m.sidebarFocused && !m.sidebarOpen {
			m.sidebarOpen = true
			m.resizeViewport()
		}
		return m, nil
	}

	// Sidebar show/hide toggle
	if msg.String() == "\\" {
		m.sidebarOpen = !m.sidebarOpen
		if !m.sidebarOpen {
			m.sidebarFocused = false
		}
		m.resizeViewport()
		return m, nil
	}

	if m.sidebarFocused {
		return m.handleSidebarKey(msg)
	}

	// Session focused: forward everything to tmux
	if m.activeIdx >= 0 && m.activeIdx < len(m.sessions) {
		s := m.sessions[m.activeIdx]
		return m, forwardKey(s.ID, msg)
	}
	return m, nil
}
```

**Step 4: Rewrite handleSidebarKey**

Replace with a much richer version that handles all TUI commands:

```go
func (m *Model) handleSidebarKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		m.sidebarFocused = false
		return m, nil
	case tea.KeyEsc:
		m.sidebarFocused = false
		return m, nil
	case tea.KeyUp:
		m.navigateSidebar(-1)
		return m, nil
	case tea.KeyDown:
		m.navigateSidebar(1)
		return m, nil
	}

	switch msg.String() {
	case "j":
		m.navigateSidebar(1)
	case "k":
		m.navigateSidebar(-1)
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		idx := int(msg.String()[0]-'0') - 1
		if idx < len(m.sessions) {
			m.switchSession(idx)
		}
	case "n":
		m.overlay = overlayNewSession
		m.nameInput = newSessionInput()
		return m, textinput.Blink
	case "x":
		if m.activeIdx >= 0 && m.activeIdx < len(m.sessions) {
			m.overlay = overlayCloseConfirm
		}
	case "?":
		m.overlay = overlayHelp
	case "q":
		return m, tea.Quit
	case "Q":
		m.killAllSessions()
		return m, tea.Quit
	case "e":
		return m, m.openEditor()
	}
	return m, nil
}
```

**Step 5: Add navigateSidebar helper**

```go
func (m *Model) navigateSidebar(dir int) {
	if len(m.sessions) == 0 {
		return
	}
	next := m.activeIdx + dir
	if next < 0 {
		next = len(m.sessions) - 1
	} else if next >= len(m.sessions) {
		next = 0
	}
	m.switchSession(next)
}
```

**Step 6: Delete handleLeaderKey**

Remove the entire `handleLeaderKey` method — it's replaced by `handleSidebarKey`.

**Step 7: Update sessionSpawnedMsg handler — set sidebarFocused = false after spawn**

In the `sessionSpawnedMsg` case, add `m.sidebarFocused = false` so focus moves to the new session:

```go
case sessionSpawnedMsg:
	m.statusMsg = ""
	m.sessions = append(m.sessions, msg.sess)
	m.store.Sessions = m.sessions
	m.saveState()
	m.switchSession(len(m.sessions) - 1)
	m.sidebarFocused = false
	return m, nil
```

**Step 8: Update hint bar in View()**

Find the hint line in `View()` and update it to show mode-aware hints. Pass `sidebarFocused` context:

```go
var hintText string
if m.sidebarFocused {
	hintText = " Tab: session   j/k navigate   n new   x close   q quit   \\ hide sidebar"
} else {
	hintText = " Tab: sidebar   \\ toggle   (all keys → Claude Code)"
}
hint := hintStyle.Width(m.width).Render(hintText)
```

**Step 9: Build**

```
go build -o claudetop .
```

Expected: clean build. This is a big refactor — fix any compile errors carefully.

**Step 10: Run tests**

```
go test ./...
```

Expected: PASS

**Step 11: Commit**

```bash
git add internal/ui/app.go
git commit -m "feat: Tab-based modal keyboard model — sidebarFocused replaces ; leader key"
```

---

### Task 5: Sidebar — PARKED section, focused border, navigation wrapping

**Files:**
- Modify: `internal/ui/sidebar.go`

**Step 1: Update renderSidebar signature**

Change to accept `sidebarFocused bool`:

```go
func renderSidebar(sessions []*session.Session, activeIdx int, height int, tick int, focused bool) string {
```

**Step 2: Add parkedDot style and PARKED section**

Add new styles:

```go
var (
	// ... existing styles ...
	dotHollow     = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	parkedNoteStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("240"))
	sidebarFocusedHeaderStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("39")). // cyan
		Bold(true)
)
```

**Step 3: Rewrite renderSidebar to split ACTIVE and PARKED**

```go
func renderSidebar(sessions []*session.Session, activeIdx int, height int, tick int, focused bool) string {
	var lines []string

	headerStyle := sidebarHeaderStyle
	if focused {
		headerStyle = sidebarFocusedHeaderStyle
	}

	// Separate active and parked
	var active, parked []*session.Session
	var activeIndices, parkedIndices []int
	for i, s := range sessions {
		if s.Parked {
			parked = append(parked, s)
			parkedIndices = append(parkedIndices, i)
		} else {
			active = append(active, s)
			activeIndices = append(activeIndices, i)
		}
	}

	lines = append(lines, headerStyle.Width(sidebarWidth).Render("ACTIVE"))
	lines = append(lines, sidebarStyle.Render(""))

	if len(active) == 0 {
		lines = append(lines, sidebarItemStyle.Width(sidebarWidth).Render("  (none)"))
		lines = append(lines, sidebarItemStyle.Width(sidebarWidth).Render("  n: new session"))
	} else {
		for _, origIdx := range activeIndices {
			lines = append(lines, renderSessionLine(sessions[origIdx], origIdx, activeIdx, tick))
		}
	}

	if len(parked) > 0 {
		lines = append(lines, sidebarStyle.Width(sidebarWidth).Render(""))
		lines = append(lines, headerStyle.Width(sidebarWidth).Render("PARKED"))
		lines = append(lines, sidebarStyle.Render(""))
		for _, origIdx := range parkedIndices {
			s := sessions[origIdx]
			lines = append(lines, renderSessionLine(s, origIdx, activeIdx, tick))
			if s.ParkNote != "" {
				note := "   \"" + s.ParkNote + "\""
				maxNote := sidebarWidth - 2
				if len([]rune(note)) > maxNote {
					runes := []rune(note)
					note = string(runes[:maxNote-1]) + "…"
				}
				lines = append(lines, parkedNoteStyle.Width(sidebarWidth).Render(note))
			}
		}
	}

	// Footer hint
	lines = append(lines, sidebarStyle.Width(sidebarWidth).Render(""))
	if focused {
		lines = append(lines, sidebarFocusedHeaderStyle.Width(sidebarWidth).Render(" n new  x close  p park"))
	} else {
		lines = append(lines, sidebarItemStyle.Width(sidebarWidth).Render(" Tab: focus sidebar"))
	}

	// Pad to fill height
	for len(lines) < height {
		lines = append(lines, sidebarStyle.Width(sidebarWidth).Render(""))
	}
	if len(lines) > height {
		lines = lines[:height]
	}

	return strings.Join(lines, "\n")
}
```

**Step 4: Extract renderSessionLine helper**

```go
func renderSessionLine(s *session.Session, idx, activeIdx, tick int) string {
	var dot string
	if s.Dead {
		dot = "✗"
	} else if s.Parked {
		dot = dotHollow.Render("○")
	} else {
		dot = statusDot(s, tick)
	}

	name := s.DisplayName()
	maxName := sidebarWidth - 5
	if len([]rune(name)) > maxName {
		runes := []rune(name)
		name = string(runes[:maxName-1]) + "…"
	}

	var line string
	if s.Dead {
		line = fmt.Sprintf(" %d ✗ %s [dead]", idx+1, name)
	} else {
		line = fmt.Sprintf(" %d %s %s", idx+1, dot, name)
	}

	style := sidebarItemStyle.Width(sidebarWidth)
	if idx == activeIdx {
		style = sidebarActiveStyle.Width(sidebarWidth)
	}
	if s.Dead {
		style = lipgloss.NewStyle().
			Width(sidebarWidth).
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("240"))
	}
	return style.Render(line)
}
```

**Step 5: Update renderSidebar call in app.go**

Find the `renderSidebar(...)` call in `View()` and add the `sidebarFocused` argument:

```go
sidebar := renderSidebar(m.sessions, m.activeIdx, mainHeight, m.tick, m.sidebarFocused)
```

**Step 6: Build and test**

```
go build -o claudetop . && go test ./...
```

Expected: clean build, all tests pass.

**Step 7: Commit**

```bash
git add internal/ui/sidebar.go internal/ui/app.go
git commit -m "feat: sidebar PARKED section with park note; cyan header when focused"
```

---

### Task 6: Park overlay

**Files:**
- Create: `internal/ui/park.go`
- Modify: `internal/ui/app.go`

**Step 1: Add overlayPark to the overlay enum in app.go**

In the `const` block:

```go
const (
	overlayNone overlay = iota
	overlayHelp
	overlayNewSession
	overlayCloseConfirm
	overlayPark
)
```

**Step 2: Add parkInput field to Model**

```go
parkInput textinput.Model
```

**Step 3: Create internal/ui/park.go**

```go
package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

func newParkInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "note (optional, e.g. \"waiting for Björn\")"
	ti.CharLimit = 100
	ti.Width = 54
	ti.Focus()
	return ti
}

func renderPark(input textinput.Model, width, height int) string {
	var lines []string
	lines = append(lines, newSessionTitleStyle.Render("Park Session"))
	lines = append(lines, "")
	lines = append(lines, "Note (optional):")
	lines = append(lines, input.View())
	lines = append(lines, "")
	lines = append(lines, newSessionHintStyle.Render("Enter: park   Esc: cancel"))

	content := newSessionOverlayStyle.Render(strings.Join(lines, "\n"))

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

**Step 4: Wire overlayPark in app.go**

In `handleKey`, add `overlayPark` to the overlay dispatch at the top:

```go
case overlayPark:
	return m.handleParkKey(msg)
```

Add `handleParkKey` method:

```go
func (m *Model) handleParkKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		if m.activeIdx >= 0 && m.activeIdx < len(m.sessions) {
			m.sessions[m.activeIdx].Parked = true
			m.sessions[m.activeIdx].ParkNote = strings.TrimSpace(m.parkInput.Value())
			m.store.Sessions = m.sessions
			m.saveState()
		}
		m.overlay = overlayNone
		return m, nil
	case tea.KeyEsc:
		m.overlay = overlayNone
		return m, nil
	}

	var cmd tea.Cmd
	m.parkInput, cmd = m.parkInput.Update(msg)
	return m, cmd
}
```

Add `p` case to `handleSidebarKey`:

```go
case "p":
	if m.activeIdx >= 0 && m.activeIdx < len(m.sessions) && !m.sessions[m.activeIdx].Parked {
		m.overlay = overlayPark
		m.parkInput = newParkInput()
		return m, textinput.Blink
	}
```

Add `u` case to `handleSidebarKey`:

```go
case "u":
	if m.activeIdx >= 0 && m.activeIdx < len(m.sessions) && m.sessions[m.activeIdx].Parked {
		m.sessions[m.activeIdx].Parked = false
		m.sessions[m.activeIdx].ParkNote = ""
		m.store.Sessions = m.sessions
		m.saveState()
	}
```

Add `overlayPark` to the `View()` switch:

```go
case overlayPark:
	return renderPark(m.parkInput, m.width, m.height)
```

**Step 5: Build and test**

```
go build -o claudetop . && go test ./...
```

Expected: clean build, all tests pass.

**Step 6: Commit**

```bash
git add internal/ui/park.go internal/ui/app.go
git commit -m "feat: park session overlay with optional note; u to unpark"
```

---

### Task 7: Rename overlay

**Files:**
- Create: `internal/ui/rename.go`
- Modify: `internal/ui/app.go`

**Step 1: Add overlayRename to the overlay enum**

```go
const (
	overlayNone overlay = iota
	overlayHelp
	overlayNewSession
	overlayCloseConfirm
	overlayPark
	overlayRename
)
```

**Step 2: Add renameInput field to Model**

```go
renameInput textinput.Model
```

**Step 3: Create internal/ui/rename.go**

```go
package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

func newRenameInput(current string) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "session name"
	ti.CharLimit = 50
	ti.Width = 44
	ti.SetValue(current)
	// Position cursor at end
	ti.CursorEnd()
	ti.Focus()
	return ti
}

func renderRename(input textinput.Model, width, height int) string {
	var lines []string
	lines = append(lines, newSessionTitleStyle.Render("Rename Session"))
	lines = append(lines, "")
	lines = append(lines, "Name:")
	lines = append(lines, input.View())
	lines = append(lines, "")
	lines = append(lines, newSessionHintStyle.Render("Enter: rename   Esc: cancel"))

	content := newSessionOverlayStyle.Render(strings.Join(lines, "\n"))

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

**Step 4: Wire overlayRename in app.go**

Add to overlay dispatch in `handleKey`:

```go
case overlayRename:
	return m.handleRenameKey(msg)
```

Add method:

```go
func (m *Model) handleRenameKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		name := strings.TrimSpace(m.renameInput.Value())
		if name != "" && m.activeIdx >= 0 && m.activeIdx < len(m.sessions) {
			m.sessions[m.activeIdx].Name = name
			m.store.Sessions = m.sessions
			m.saveState()
		}
		m.overlay = overlayNone
		return m, nil
	case tea.KeyEsc:
		m.overlay = overlayNone
		return m, nil
	}

	var cmd tea.Cmd
	m.renameInput, cmd = m.renameInput.Update(msg)
	return m, cmd
}
```

Add `r` case to `handleSidebarKey`:

```go
case "r":
	if m.activeIdx >= 0 && m.activeIdx < len(m.sessions) {
		m.overlay = overlayRename
		m.renameInput = newRenameInput(m.sessions[m.activeIdx].Name)
		return m, textinput.Blink
	}
```

Add to `View()` switch:

```go
case overlayRename:
	return renderRename(m.renameInput, m.width, m.height)
```

**Step 5: Build and test**

```
go build -o claudetop . && go test ./...
```

Expected: clean build, all tests pass.

**Step 6: Commit**

```bash
git add internal/ui/rename.go internal/ui/app.go
git commit -m "feat: rename session overlay pre-filled with current name"
```

---

### Task 8: Prompt editor overlay (`N`)

**Files:**
- Create: `internal/ui/prompteditor.go`
- Modify: `internal/ui/app.go`

**Step 1: Add overlayPromptEditor to the overlay enum**

```go
const (
	overlayNone overlay = iota
	overlayHelp
	overlayNewSession
	overlayCloseConfirm
	overlayPark
	overlayRename
	overlayPromptEditor
)
```

**Step 2: Add promptInput to Model**

Add the import `"github.com/charmbracelet/bubbles/textarea"` to `app.go`'s import block. Then add the field:

```go
promptInput textarea.Model
```

**Step 3: Add pendingPrompt to Model**

```go
pendingPrompt string // prompt text waiting to be sent after session spawns
```

**Step 4: Update sessionSpawnedMsg to carry prompt**

```go
type sessionSpawnedMsg struct {
	sess   *session.Session
	prompt string // empty for blank sessions
}
```

Update `spawnSession` signature:

```go
func spawnSession(name, rootDir, prompt string, sessionCount int) tea.Cmd {
	return func() tea.Msg {
		s := session.NewSession(name, sessionCount+1)
		if _, err := os.Stat(rootDir); err != nil {
			return errMsg{fmt.Errorf("root_dir %q does not exist: %w", rootDir, err)}
		}
		if err := tmux.Create(s.ID, rootDir); err != nil {
			return errMsg{fmt.Errorf("create session: %w", err)}
		}
		return sessionSpawnedMsg{sess: s, prompt: prompt}
	}
}
```

Update the existing caller in `handleNewSessionKey` to pass `""` for prompt:

```go
return m, spawnSession(name, m.cfg.General.RootDir, "", len(m.sessions))
```

**Step 5: Create internal/ui/prompteditor.go**

```go
package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
)

var promptEditorOverlayStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("240")).
	Background(lipgloss.Color("235")).
	Padding(1, 2).
	Width(66)

func newPromptEditor() textarea.Model {
	ta := textarea.New()
	ta.Placeholder = "Write your prompt for Claude..."
	ta.CharLimit = 2000
	ta.SetWidth(60)
	ta.SetHeight(8)
	ta.Focus()
	return ta
}

func renderPromptEditor(input textarea.Model, width, height int) string {
	var lines []string
	lines = append(lines, newSessionTitleStyle.Render("New Session — Write Prompt"))
	lines = append(lines, "")
	lines = append(lines, input.View())
	lines = append(lines, "")
	lines = append(lines, newSessionHintStyle.Render("Ctrl+S: spawn session   Esc: cancel"))

	content := promptEditorOverlayStyle.Render(strings.Join(lines, "\n"))

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

**Step 6: Wire overlayPromptEditor in app.go**

Add to overlay dispatch in `handleKey`:

```go
case overlayPromptEditor:
	return m.handlePromptEditorKey(msg)
```

Add method:

```go
func (m *Model) handlePromptEditorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlS:
		prompt := strings.TrimSpace(m.promptInput.Value())
		if prompt == "" {
			m.overlay = overlayNone
			return m, nil
		}
		name := m.uniqueName(fmt.Sprintf("session-%d", len(m.sessions)+1))
		m.overlay = overlayNone
		return m, spawnSession(name, m.cfg.General.RootDir, prompt, len(m.sessions))
	case tea.KeyEsc:
		m.overlay = overlayNone
		return m, nil
	}

	var cmd tea.Cmd
	m.promptInput, cmd = m.promptInput.Update(msg)
	return m, cmd
}
```

Add `N` case to `handleSidebarKey`:

```go
case "N":
	m.overlay = overlayPromptEditor
	m.promptInput = newPromptEditor()
	return m, m.promptInput.Focus()
```

Add to `View()` switch:

```go
case overlayPromptEditor:
	return renderPromptEditor(m.promptInput, m.width, m.height)
```

**Step 7: Run go mod tidy** (textarea is already in bubbles, no new dep needed, but tidy anyway):

```
go mod tidy
```

**Step 8: Build and test**

```
go build -o claudetop . && go test ./...
```

Expected: clean build, all tests pass.

**Step 9: Commit**

```bash
git add internal/ui/prompteditor.go internal/ui/app.go
git commit -m "feat: prompt editor overlay (N) with multi-line textarea; Ctrl+S to submit"
```

---

### Task 9: Auto-naming via `claude -p`

**Files:**
- Modify: `internal/ui/app.go`

**Step 1: Add sessionRenamedMsg**

```go
type sessionRenamedMsg struct {
	sessionID string
	name      string
}
```

**Step 2: Add sendPrompt command**

```go
// sendPrompt sends text to a tmux session after a startup delay.
func sendPrompt(sessionID, prompt string) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(2 * time.Second)
		if err := tmux.SendLiteralKey(sessionID, prompt); err != nil {
			return errMsg{err}
		}
		if err := tmux.SendKeys(sessionID, "Enter"); err != nil {
			return errMsg{err}
		}
		return nil
	}
}
```

**Step 3: Add autoName command**

```go
// autoName calls claude -p to summarize the prompt into a short session name.
// Returns nil on any failure — auto-naming is best-effort.
func autoName(sessionID, prompt string) tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("claude", "-p",
			"Summarize this task in 3 words max, lowercase, hyphen-separated, no punctuation. Output only the name, nothing else: "+prompt,
		).Output()
		if err != nil {
			return nil
		}
		name := strings.TrimSpace(string(out))
		if name == "" {
			return nil
		}
		return sessionRenamedMsg{sessionID: sessionID, name: name}
	}
}
```

Note: `exec` is already imported in `app.go`. If not, add `"os/exec"` to imports.

**Step 4: Handle sessionRenamedMsg in Update**

```go
case sessionRenamedMsg:
	for _, s := range m.sessions {
		if s.ID == msg.sessionID {
			s.Name = msg.name
			m.store.Sessions = m.sessions
			m.saveState()
			break
		}
	}
	return m, nil
```

**Step 5: Fire sendPrompt and autoName from sessionSpawnedMsg handler**

Update the `sessionSpawnedMsg` case:

```go
case sessionSpawnedMsg:
	m.statusMsg = ""
	m.sessions = append(m.sessions, msg.sess)
	m.store.Sessions = m.sessions
	m.saveState()
	m.switchSession(len(m.sessions) - 1)
	m.sidebarFocused = false

	var cmds []tea.Cmd
	if msg.prompt != "" {
		cmds = append(cmds, sendPrompt(msg.sess.ID, msg.prompt))
		if m.cfg.General.AutoNameSessions {
			cmds = append(cmds, autoName(msg.sess.ID, msg.prompt))
		}
	}
	return m, tea.Batch(cmds...)
```

**Step 6: Build and test**

```
go build -o claudetop . && go test ./...
```

Expected: clean build, all tests pass.

**Step 7: Commit**

```bash
git add internal/ui/app.go
git commit -m "feat: auto-name sessions via claude -p; prompt sent to tmux after 2s delay"
```

---

### Task 10: Update help overlay

**Files:**
- Modify: `internal/ui/help.go`

**Step 1: Rewrite helpEntries to reflect the new keyboard model**

```go
var helpEntries = []helpEntry{
	{"Session mode", ""},
	{"Tab", "Focus sidebar (enter command mode)"},
	{"\\", "Toggle sidebar visibility"},
	{"", ""},
	{"Sidebar mode", ""},
	{"Tab / Esc / Enter", "Return to session"},
	{"j / k / ↑ / ↓", "Navigate sessions"},
	{"1–9", "Jump to session by number"},
	{"", ""},
	{"n", "New blank session"},
	{"N", "New session with prompt editor"},
	{"x", "Close session (confirm)"},
	{"p", "Park session (optional note)"},
	{"u", "Unpark session"},
	{"r", "Rename session"},
	{"", ""},
	{"q", "Quit (sessions keep running)"},
	{"Q", "Quit and kill all sessions"},
	{"e", "Edit root CLAUDE.md in $EDITOR"},
	{"?", "Toggle this help"},
}
```

**Step 2: Build and test**

```
go build -o claudetop . && go test ./...
```

Expected: clean build, all tests pass.

**Step 3: Commit**

```bash
git add internal/ui/help.go
git commit -m "docs: update help overlay with new keyboard model"
```

---

### Task 11: Final review

**Step 1: Clean build**

```
go clean -cache && go build -o claudetop . && echo "clean build ok"
```

**Step 2: Run all tests**

```
go test ./... -v
```

Expected: all tests pass, no skips.

**Step 3: Manual smoke test checklist**

Run `./claudetop` and verify:

- [ ] Tab focuses/unfocuses sidebar (cyan "ACTIVE" header when focused)
- [ ] `\` toggles sidebar visibility without changing focus
- [ ] `j`/`k`/`↑`/`↓` navigate sessions in sidebar
- [ ] `n` creates blank session; `N` opens prompt editor (Ctrl+S to submit, Esc to cancel)
- [ ] `p` parks a session with optional note; note appears in PARKED section
- [ ] `u` unparks a session; moves back to ACTIVE
- [ ] `r` renames session with current name pre-filled
- [ ] Session spawned from `N` gets auto-named within ~5s
- [ ] `q` quits; `Q` kills all and quits
- [ ] Status bar error messages disappear after ~10s
- [ ] All keys pass through to Claude when in session mode (no ; needed)

**Step 4: Write DONE.md entry**

Append to `DONE.md`:

```markdown
## Iteration 2 — Session Management Quality

- Modal keyboard model: Tab focuses sidebar (command mode) / session (passthrough mode)
- `;` leader key removed; all commands accessed via sidebar
- Parked sessions with optional notes; PARKED section in sidebar
- Rename any session in place (`r`)
- Prompt editor (`N`) with multi-line textarea; Ctrl+S to submit
- Auto-naming via `claude -p` for sessions created with prompt editor
- statusMsg expires after 10s; redundant EnterAltScreen removed; config.Dir() propagates error
```

**Step 5: Commit**

```bash
git add DONE.md
git commit -m "chore: iteration 2 complete"
```

---

## Deviations to document in PLAN.md (if any occur)

Record unexpected design decisions here during implementation.
