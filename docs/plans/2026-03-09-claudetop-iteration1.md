# claudetop Iteration 1 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a fullscreen TUI that manages Claude Code sessions backed by tmux, with sidebar, status detection, keyboard passthrough, and persistent state across restarts.

**Architecture:** Go + Bubbletea. A single Bubbletea model holds all UI state. Background polling via `tea.Tick` reads tmux pane output, updates session status, and refreshes the viewport. Keypresses go through to tmux via `tmux send-keys` unless prefixed with `;` or intercepted when the sidebar is focused.

**Tech Stack:** Go 1.22+, github.com/charmbracelet/bubbletea, github.com/charmbracelet/bubbles, github.com/charmbracelet/lipgloss, github.com/BurntSushi/toml

---

## Prerequisite: Install Go

**Step 1: Install Go**

```bash
brew install go
```

**Step 2: Verify**

```bash
go version
```

Expected: `go version go1.22.x darwin/arm64` (or similar)

---

## Task 1: Project scaffolding

**Files:**
- Create: `go.mod`
- Create: `main.go` (placeholder)
- Create: `internal/config/config.go` (empty package)
- Create: `internal/tmux/manager.go` (empty package)
- Create: `internal/state/store.go` (empty package)
- Create: `internal/session/session.go` (empty package)
- Create: `internal/ui/app.go` (empty package)
- Create: `PLAN.md`

**Step 1: Initialize Go module**

```bash
cd /Users/andersnordmark/work/personal/claudetop
go mod init claudetop
```

**Step 2: Create directory structure**

```bash
mkdir -p internal/config internal/tmux internal/state internal/session internal/ui
```

**Step 3: Add dependencies**

```bash
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/bubbles@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/BurntSushi/toml@latest
go mod tidy
```

**Step 4: Create placeholder main.go**

```go
// main.go
package main

import "fmt"

func main() {
	fmt.Println("claudetop")
}
```

**Step 5: Create empty package stubs**

Create `internal/config/config.go`:
```go
package config
```

Create `internal/tmux/manager.go`:
```go
package tmux
```

Create `internal/state/store.go`:
```go
package state
```

Create `internal/session/session.go`:
```go
package session
```

Create `internal/ui/app.go`:
```go
package ui
```

**Step 6: Create PLAN.md in project root**

```markdown
# claudetop Iteration 1 — Build Plan

## Components

- [ ] Task 1: Project scaffolding
- [ ] Task 2: Config loading
- [ ] Task 3: Session model
- [ ] Task 4: tmux manager
- [ ] Task 5: State store
- [ ] Task 6: Status detector
- [ ] Task 7: Status bar component
- [ ] Task 8: Sidebar component
- [ ] Task 9: Main viewport component
- [ ] Task 10: Help overlay
- [ ] Task 11: New session flow
- [ ] Task 12: Main app model
- [ ] Task 13: Main entry point
- [ ] Task 14: Integration testing (3+ concurrent sessions)
- [ ] Task 15: Phase 4 review
- [ ] Task 16: Phase 5 review
- [ ] Task 17: Phase 6 final + DONE.md

## Deviations from Spec

(record here as they occur)
```

**Step 7: Verify it compiles**

```bash
go build ./...
```

Expected: no output (clean build)

**Step 8: Commit**

```bash
git add go.mod go.sum main.go internal/ PLAN.md
git commit -m "feat: scaffold project structure"
```

---

## Task 2: Config loading

**Files:**
- Write: `internal/config/config.go`
- Write: `internal/config/config_test.go`

**Step 1: Write config.go**

```go
// internal/config/config.go
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	General GeneralConfig `toml:"general"`
}

type GeneralConfig struct {
	RootDir string `toml:"root_dir"`
}

// Dir returns the ~/.claudetop directory path.
func Dir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claudetop")
}

// Path returns the config file path.
func Path() string {
	return filepath.Join(Dir(), "config.toml")
}

// Load reads the config file, returning defaults if missing.
func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("home dir: %w", err)
	}

	cfg := &Config{
		General: GeneralConfig{
			RootDir: home,
		},
	}

	path := Path()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	return cfg, nil
}

// EnsureDir creates ~/.claudetop if it doesn't exist.
func EnsureDir() error {
	return os.MkdirAll(Dir(), 0755)
}
```

**Step 2: Write config_test.go**

```go
// internal/config/config_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultsWhenMissing(t *testing.T) {
	// Point to a temp dir with no config file
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.General.RootDir != tmp {
		t.Errorf("expected RootDir=%q, got %q", tmp, cfg.General.RootDir)
	}
}

func TestLoadFromFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	dir := filepath.Join(tmp, ".claudetop")
	os.MkdirAll(dir, 0755)

	content := `[general]
root_dir = "/tmp/repos"
`
	os.WriteFile(filepath.Join(dir, "config.toml"), []byte(content), 0644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.General.RootDir != "/tmp/repos" {
		t.Errorf("expected /tmp/repos, got %q", cfg.General.RootDir)
	}
}
```

**Step 3: Run tests**

```bash
go test ./internal/config/...
```

Expected: `PASS`

**Step 4: Update PLAN.md** — mark Task 2 done

**Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: config loading from ~/.claudetop/config.toml"
```

---

## Task 3: Session model

**Files:**
- Write: `internal/session/session.go`

**Step 1: Write session.go**

```go
// internal/session/session.go
package session

import (
	"fmt"
	"time"
)

// Status represents the detected state of a Claude Code session.
type Status int

const (
	StatusStarting    Status = iota // first 10 seconds
	StatusWorking                   // actively producing output
	StatusNeedsInput                // waiting for user to type
	StatusPermission                // waiting for command approval
	StatusDone                      // idle, task complete
	StatusStuck                     // working but silent > 2 minutes
	StatusError                     // error pattern detected
)

func (s Status) String() string {
	switch s {
	case StatusStarting:
		return "starting"
	case StatusWorking:
		return "working"
	case StatusNeedsInput:
		return "needs_input"
	case StatusPermission:
		return "permission"
	case StatusDone:
		return "done"
	case StatusStuck:
		return "stuck"
	case StatusError:
		return "error"
	default:
		return "unknown"
	}
}

// Dot returns the status indicator character.
func (s Status) Dot() string {
	return "●"
}

// Session represents a single Claude Code session.
type Session struct {
	// Persisted fields
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Dead      bool      `json:"dead,omitempty"`

	// Runtime-only (not persisted)
	Status       Status    `json:"-"`
	PaneContent  string    `json:"-"`
	LastOutputAt time.Time `json:"-"`
}

// TmuxName returns the namespaced tmux session name.
func (s *Session) TmuxName() string {
	return "ct-" + s.ID
}

// DisplayName returns the name to show in the sidebar.
func (s *Session) DisplayName() string {
	if s.Name != "" {
		return s.Name
	}
	return s.ID
}

// NewSession creates a session with a unique ID.
func NewSession(name string, index int) *Session {
	id := name
	if id == "" {
		id = fmt.Sprintf("session-%d", index)
	}
	return &Session{
		ID:        id,
		Name:      name,
		CreatedAt: time.Now(),
		Status:    StatusStarting,
	}
}
```

**Step 2: Verify it compiles**

```bash
go build ./internal/session/...
```

**Step 3: Update PLAN.md** — mark Task 3 done

**Step 4: Commit**

```bash
git add internal/session/session.go
git commit -m "feat: session model with status types"
```

---

## Task 4: tmux manager

**Files:**
- Write: `internal/tmux/manager.go`

> Note: tmux operations require a real tmux session; no unit tests — verified by integration test in Task 14.

**Step 1: Write manager.go**

```go
// internal/tmux/manager.go
package tmux

import (
	"fmt"
	"os/exec"
	"strings"
)

const prefix = "ct-"

// SessionName converts a session ID to a tmux session name.
func SessionName(id string) string {
	return prefix + id
}

// Create creates a new detached tmux session and starts claude in it.
func Create(id, rootDir string) error {
	name := SessionName(id)

	// Create detached session in root_dir
	out, err := exec.Command("tmux", "new-session", "-d", "-s", name, "-c", rootDir).CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmux new-session: %w\n%s", err, out)
	}

	// Start claude inside the session
	out, err = exec.Command("tmux", "send-keys", "-t", name, "claude", "Enter").CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmux send-keys claude: %w\n%s", err, out)
	}

	return nil
}

// Kill destroys a tmux session.
func Kill(id string) error {
	name := SessionName(id)
	out, err := exec.Command("tmux", "kill-session", "-t", name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmux kill-session: %w\n%s", err, out)
	}
	return nil
}

// Exists reports whether the tmux session is alive.
func Exists(id string) bool {
	return exec.Command("tmux", "has-session", "-t", SessionName(id)).Run() == nil
}

// CapturePane reads the current visible content of a tmux pane.
func CapturePane(id string) (string, error) {
	name := SessionName(id)
	out, err := exec.Command("tmux", "capture-pane", "-t", name, "-p", "-e").Output()
	if err != nil {
		return "", fmt.Errorf("tmux capture-pane: %w", err)
	}
	return string(out), nil
}

// SendKeys forwards a keystroke to a tmux session.
// key is the raw character or tmux key name (e.g. "Enter", "BSpace").
func SendKeys(id string, key string) error {
	name := SessionName(id)
	// Use empty string as the final arg so tmux doesn't append Enter
	out, err := exec.Command("tmux", "send-keys", "-t", name, key, "").CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmux send-keys: %w\n%s", err, out)
	}
	return nil
}

// SendLiteralKey sends a key using the -l flag (literal, no special interpretation).
func SendLiteralKey(id string, key string) error {
	name := SessionName(id)
	out, err := exec.Command("tmux", "send-keys", "-t", name, "-l", key).CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmux send-keys -l: %w\n%s", err, out)
	}
	return nil
}

// LiveSessions returns IDs of all ct- prefixed tmux sessions.
func LiveSessions() ([]string, error) {
	out, err := exec.Command("tmux", "list-sessions", "-F", "#{session_name}").Output()
	if err != nil {
		// No sessions running — not an error
		return nil, nil
	}

	var ids []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			ids = append(ids, strings.TrimPrefix(line, prefix))
		}
	}
	return ids, nil
}
```

**Step 2: Verify it compiles**

```bash
go build ./internal/tmux/...
```

**Step 3: Update PLAN.md** — mark Task 4 done

**Step 4: Commit**

```bash
git add internal/tmux/manager.go
git commit -m "feat: tmux session manager (create/kill/capture/send-keys)"
```

---

## Task 5: State store

**Files:**
- Write: `internal/state/store.go`
- Write: `internal/state/store_test.go`

**Step 1: Write store.go**

```go
// internal/state/store.go
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"claudetop/internal/session"
)

type State struct {
	Sessions []*session.Session `json:"sessions"`
}

func path() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claudetop", "state.json")
}

// Load reads state from disk. Returns empty state if file is missing.
func Load() (*State, error) {
	p := path()
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return &State{}, nil
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

// Save writes state to disk atomically.
func Save(s *State) error {
	p := path()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	// Write to temp file then rename for atomicity
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("write state: %w", err)
	}
	return os.Rename(tmp, p)
}
```

**Step 2: Write store_test.go**

```go
// internal/state/store_test.go
package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"claudetop/internal/session"
)

func TestLoadEmptyWhenMissing(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	s, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(s.Sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(s.Sessions))
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	os.MkdirAll(filepath.Join(tmp, ".claudetop"), 0755)

	sess := &session.Session{
		ID:        "test-session",
		Name:      "test",
		CreatedAt: time.Now().Truncate(time.Second),
	}
	s := &State{Sessions: []*session.Session{sess}}

	if err := Save(s); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(loaded.Sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(loaded.Sessions))
	}
	if loaded.Sessions[0].ID != "test-session" {
		t.Errorf("expected ID=test-session, got %q", loaded.Sessions[0].ID)
	}
}
```

**Step 3: Run tests**

```bash
go test ./internal/state/...
```

Expected: `PASS`

**Step 4: Update PLAN.md** — mark Task 5 done

**Step 5: Commit**

```bash
git add internal/state/
git commit -m "feat: state store with atomic JSON persistence"
```

---

## Task 6: Status detector

**Files:**
- Write: `internal/session/status.go`
- Write: `internal/session/status_test.go`

**Step 1: Write status.go**

```go
// internal/session/status.go
package session

import (
	"regexp"
	"strings"
	"time"
)

var (
	// Claude Code is asking for user input
	needsInputPatterns = []*regexp.Regexp{
		regexp.MustCompile(`\?\s*$`),
		regexp.MustCompile(`(?i)would you like`),
		regexp.MustCompile(`(?i)should i`),
		regexp.MustCompile(`(?i)do you want`),
		regexp.MustCompile(`(?i)shall i`),
	}

	// Claude Code wants permission to run a command or write a file
	permissionPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)do you want me to`),
		regexp.MustCompile(`(?i)allow.*to run`),
		regexp.MustCompile(`(?i)may i`),
		regexp.MustCompile(`\(y/N\)`),
		regexp.MustCompile(`\(Y/n\)`),
	}

	// Active tool use — Claude is processing
	workingPatterns = []*regexp.Regexp{
		regexp.MustCompile(`⏺`),
		regexp.MustCompile(`⠋|⠙|⠹|⠸|⠼|⠴|⠦|⠧|⠇|⠏`), // spinner chars
	}

	// Error condition
	errorPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?m)^Error:`),
		regexp.MustCompile(`(?m)^error:`),
	}
)

// lastNLines returns the last n lines of text.
func lastNLines(text string, n int) string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if len(lines) <= n {
		return text
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}

// Detect derives session status from pane output and timing.
// current is the previously detected status (needed for stuck detection).
func Detect(paneOutput string, lastOutputAt time.Time, createdAt time.Time, current Status) Status {
	age := time.Since(createdAt)
	if age < 10*time.Second {
		return StatusStarting
	}

	tail := lastNLines(paneOutput, 15)

	// Error check first — always surfaces
	for _, p := range errorPatterns {
		if p.MatchString(tail) {
			return StatusError
		}
	}

	// If output was very recent, we're working
	if time.Since(lastOutputAt) < 3*time.Second {
		return StatusWorking
	}

	// Permission request
	for _, p := range permissionPatterns {
		if p.MatchString(tail) {
			return StatusPermission
		}
	}

	// Needs input
	for _, p := range needsInputPatterns {
		if p.MatchString(tail) {
			return StatusNeedsInput
		}
	}

	// Stuck: was working but silent for 2+ minutes
	if current == StatusWorking && time.Since(lastOutputAt) > 2*time.Minute {
		return StatusStuck
	}

	return StatusDone
}
```

**Step 2: Write status_test.go**

```go
// internal/session/status_test.go
package session

import (
	"testing"
	"time"
)

func TestDetectStarting(t *testing.T) {
	now := time.Now()
	status := Detect("some output", now, now.Add(-5*time.Second), StatusStarting)
	if status != StatusStarting {
		t.Errorf("expected Starting for new session, got %v", status)
	}
}

func TestDetectWorking(t *testing.T) {
	createdAt := time.Now().Add(-30 * time.Second)
	lastOutput := time.Now().Add(-1 * time.Second) // recent output
	status := Detect("Running bash...", lastOutput, createdAt, StatusDone)
	if status != StatusWorking {
		t.Errorf("expected Working for recent output, got %v", status)
	}
}

func TestDetectNeedsInput(t *testing.T) {
	createdAt := time.Now().Add(-30 * time.Second)
	lastOutput := time.Now().Add(-10 * time.Second)
	output := "I've analyzed the code. Would you like me to proceed with the fix?"
	status := Detect(output, lastOutput, createdAt, StatusDone)
	if status != StatusNeedsInput {
		t.Errorf("expected NeedsInput, got %v", status)
	}
}

func TestDetectStuck(t *testing.T) {
	createdAt := time.Now().Add(-10 * time.Minute)
	lastOutput := time.Now().Add(-3 * time.Minute) // silent for 3 min
	status := Detect("Running tool...", lastOutput, createdAt, StatusWorking)
	if status != StatusStuck {
		t.Errorf("expected Stuck, got %v", status)
	}
}

func TestDetectError(t *testing.T) {
	createdAt := time.Now().Add(-30 * time.Second)
	lastOutput := time.Now().Add(-5 * time.Second)
	status := Detect("Error: connection refused", lastOutput, createdAt, StatusWorking)
	if status != StatusError {
		t.Errorf("expected Error, got %v", status)
	}
}

func TestDetectDone(t *testing.T) {
	createdAt := time.Now().Add(-30 * time.Second)
	lastOutput := time.Now().Add(-30 * time.Second)
	output := "I've completed the task. The fix has been applied successfully."
	status := Detect(output, lastOutput, createdAt, StatusDone)
	if status != StatusDone {
		t.Errorf("expected Done for silent non-question output, got %v", status)
	}
}
```

**Step 3: Run tests**

```bash
go test ./internal/session/...
```

Expected: `PASS`

**Step 4: Update PLAN.md** — mark Task 6 done

**Step 5: Commit**

```bash
git add internal/session/status.go internal/session/status_test.go
git commit -m "feat: status detector with pattern matching"
```

---

## Task 7: Status bar component

**Files:**
- Write: `internal/ui/statusbar.go`

**Step 1: Write statusbar.go**

```go
// internal/ui/statusbar.go
package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"

	"claudetop/internal/session"
)

var (
	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("250"))

	statusBarBold = lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("255")).
			Bold(true)

	needsInputStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("196")) // red
)

// renderStatusBar returns the top status line.
func renderStatusBar(sessions []*session.Session, width int) string {
	total := len(sessions)
	needsInput := 0
	for _, s := range sessions {
		if s.Status == session.StatusNeedsInput || s.Status == session.StatusPermission {
			needsInput++
		}
	}

	name := statusBarBold.Render("claudetop")

	counts := statusBarStyle.Render(fmt.Sprintf("  %d sessions", total))

	var attention string
	if needsInput > 0 {
		attention = "  " + needsInputStyle.Render(fmt.Sprintf("● %d needs input", needsInput))
	}

	clock := statusBarStyle.Render(time.Now().Format("15:04"))

	// Left side: name + counts + attention
	left := name + counts + attention

	// Pad to fill width, put clock on the right
	leftLen := lipgloss.Width(left)
	clockLen := lipgloss.Width(clock)
	padding := width - leftLen - clockLen
	if padding < 1 {
		padding = 1
	}

	return statusBarStyle.Width(width).Render(
		left + lipgloss.NewStyle().Background(lipgloss.Color("235")).Render(fmt.Sprintf("%*s", padding, "")) + clock,
	)
}
```

**Step 2: Verify it compiles**

```bash
go build ./internal/ui/...
```

**Step 3: Update PLAN.md** — mark Task 7 done

**Step 4: Commit**

```bash
git add internal/ui/statusbar.go
git commit -m "feat: status bar component"
```

---

## Task 8: Sidebar component

**Files:**
- Write: `internal/ui/sidebar.go`

**Step 1: Write sidebar.go**

```go
// internal/ui/sidebar.go
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"claudetop/internal/session"
)

const sidebarWidth = 22

var (
	sidebarStyle = lipgloss.NewStyle().
			Width(sidebarWidth).
			Background(lipgloss.Color("236"))

	sidebarHeaderStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("236")).
				Foreground(lipgloss.Color("244")).
				Bold(true)

	sidebarItemStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("236")).
				Foreground(lipgloss.Color("252"))

	sidebarActiveStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("238")).
				Foreground(lipgloss.Color("255")).
				Bold(true)

	// Status dot colors
	dotGrey   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	dotYellow = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	dotRed    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	dotGreen  = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	dotOrange = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

func statusDot(s *session.Session, tick int) string {
	switch s.Status {
	case session.StatusStarting:
		return dotGrey.Render("●")
	case session.StatusWorking:
		// Animate with a simple character cycle
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		return dotYellow.Render(frames[tick%len(frames)])
	case session.StatusNeedsInput:
		return dotRed.Render("●")
	case session.StatusPermission:
		return dotRed.Render("●!")
	case session.StatusDone:
		return dotGreen.Render("●")
	case session.StatusStuck:
		return dotOrange.Render("●")
	case session.StatusError:
		return dotRed.Render("●✗")
	default:
		return dotGrey.Render("●")
	}
}

// renderSidebar renders the session list sidebar.
// activeIdx is the index of the currently focused session (-1 if none).
// tick is an animation counter for working status.
func renderSidebar(sessions []*session.Session, activeIdx int, height int, tick int) string {
	var lines []string

	lines = append(lines, sidebarHeaderStyle.Width(sidebarWidth).Render("ACTIVE"))
	lines = append(lines, sidebarStyle.Render(""))

	if len(sessions) == 0 {
		lines = append(lines, sidebarItemStyle.Width(sidebarWidth).Render("  (none)"))
		lines = append(lines, sidebarItemStyle.Width(sidebarWidth).Render("  n: new session"))
	} else {
		for i, s := range sessions {
			num := fmt.Sprintf("%d", i+1)
			dot := statusDot(s, tick)
			name := s.DisplayName()

			// Truncate name to fit
			maxName := sidebarWidth - 4 // room for number + dot + spaces
			if len(name) > maxName {
				name = name[:maxName-1] + "…"
			}

			line := fmt.Sprintf(" %s %s %s", num, dot, name)

			style := sidebarItemStyle.Width(sidebarWidth)
			if i == activeIdx {
				style = sidebarActiveStyle.Width(sidebarWidth)
			}
			lines = append(lines, style.Render(line))
		}
	}

	// Footer hint
	lines = append(lines, strings.Repeat("\n", 1)) // spacer
	hint := sidebarItemStyle.Width(sidebarWidth).Render(" n new  x close")
	lines = append(lines, hint)

	// Pad to fill height
	for len(lines) < height {
		lines = append(lines, sidebarStyle.Width(sidebarWidth).Render(""))
	}

	// Trim to height
	if len(lines) > height {
		lines = lines[:height]
	}

	return strings.Join(lines, "\n")
}
```

**Step 2: Verify it compiles**

```bash
go build ./internal/ui/...
```

**Step 3: Update PLAN.md** — mark Task 8 done

**Step 4: Commit**

```bash
git add internal/ui/sidebar.go
git commit -m "feat: sidebar with status dots and animated working indicator"
```

---

## Task 9: Main viewport component

**Files:**
- Write: `internal/ui/viewport.go`

**Step 1: Write viewport.go**

```go
// internal/ui/viewport.go
package ui

import (
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

var viewportBorderStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

// newViewport creates a configured viewport for session output.
func newViewport(width, height int) viewport.Model {
	vp := viewport.New(width, height)
	vp.Style = lipgloss.NewStyle().
		Background(lipgloss.Color("0"))
	return vp
}
```

**Step 2: Verify it compiles**

```bash
go build ./internal/ui/...
```

**Step 3: Update PLAN.md** — mark Task 9 done

**Step 4: Commit**

```bash
git add internal/ui/viewport.go
git commit -m "feat: viewport wrapper for session output"
```

---

## Task 10: Help overlay

**Files:**
- Write: `internal/ui/help.go`

**Step 1: Write help.go**

```go
// internal/ui/help.go
package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	helpOverlayStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Background(lipgloss.Color("235")).
				Padding(1, 2)

	helpTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Bold(true).
			MarginBottom(1)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			Width(12)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250"))
)

type helpEntry struct {
	key  string
	desc string
}

var helpEntries = []helpEntry{
	{"Navigation", ""},
	{"1-9", "Switch to session"},
	{"] / [", "Next / prev session"},
	{"\\", "Toggle sidebar"},
	{"", ""},
	{"Sessions", ""},
	{"n", "New session"},
	{"x", "Close session"},
	{"", ""},
	{";-prefix", ""},
	{";?", "This help"},
	{";q", "Quit (sessions keep running)"},
	{";Q", "Quit and kill all sessions"},
	{";e", "Edit root CLAUDE.md"},
	{"", ""},
	{"Esc", "Close overlay"},
}

// renderHelp renders the help overlay centered in the terminal.
func renderHelp(width, height int) string {
	var lines []string
	lines = append(lines, helpTitleStyle.Render("claudetop — keyboard shortcuts"))

	for _, e := range helpEntries {
		if e.key == "" && e.desc == "" {
			lines = append(lines, "")
			continue
		}
		if e.desc == "" {
			// Section header
			lines = append(lines, lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				Render(e.key))
			continue
		}
		key := helpKeyStyle.Render(e.key)
		desc := helpDescStyle.Render(e.desc)
		lines = append(lines, key+"  "+desc)
	}

	content := helpOverlayStyle.Render(strings.Join(lines, "\n"))

	// Center in terminal
	contentWidth := lipgloss.Width(content)
	contentHeight := lipgloss.Height(content)

	leftPad := (width - contentWidth) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	topPad := (height - contentHeight) / 2
	if topPad < 0 {
		topPad = 0
	}

	var result []string
	emptyLine := strings.Repeat(" ", width)
	for i := 0; i < topPad; i++ {
		result = append(result, emptyLine)
	}
	for _, line := range strings.Split(content, "\n") {
		result = append(result, strings.Repeat(" ", leftPad)+line)
	}
	return strings.Join(result, "\n")
}
```

**Step 2: Verify it compiles**

```bash
go build ./internal/ui/...
```

**Step 3: Update PLAN.md** — mark Task 10 done

**Step 4: Commit**

```bash
git add internal/ui/help.go
git commit -m "feat: help overlay with keybinding reference"
```

---

## Task 11: New session flow (text input overlay)

**Files:**
- Write: `internal/ui/newsession.go`

**Step 1: Write newsession.go**

```go
// internal/ui/newsession.go
package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

var (
	newSessionOverlayStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Background(lipgloss.Color("235")).
				Padding(1, 2).
				Width(50)

	newSessionTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")).
				Bold(true).
				MarginBottom(1)

	newSessionHintStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				Italic(true)
)

// newSessionInput creates a configured text input for session name.
func newSessionInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "session name (blank for auto)"
	ti.CharLimit = 50
	ti.Width = 44
	ti.Focus()
	return ti
}

// renderNewSession renders the new session prompt overlay.
func renderNewSession(input textinput.Model, width, height int) string {
	var lines []string
	lines = append(lines, newSessionTitleStyle.Render("New Session"))
	lines = append(lines, "Name:")
	lines = append(lines, input.View())
	lines = append(lines, "")
	lines = append(lines, newSessionHintStyle.Render("Enter: create   Esc: cancel"))

	content := newSessionOverlayStyle.Render(strings.Join(lines, "\n"))

	contentWidth := lipgloss.Width(content)
	contentHeight := lipgloss.Height(content)

	leftPad := (width - contentWidth) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	topPad := (height - contentHeight) / 2
	if topPad < 0 {
		topPad = 0
	}

	var result []string
	emptyLine := strings.Repeat(" ", width)
	for i := 0; i < topPad; i++ {
		result = append(result, emptyLine)
	}
	for _, line := range strings.Split(content, "\n") {
		result = append(result, strings.Repeat(" ", leftPad)+line)
	}
	return strings.Join(result, "\n")
}
```

**Step 2: Verify it compiles**

```bash
go build ./internal/ui/...
```

**Step 3: Update PLAN.md** — mark Task 11 done

**Step 4: Commit**

```bash
git add internal/ui/newsession.go
git commit -m "feat: new session overlay with text input"
```

---

## Task 12: Main app model

**Files:**
- Write: `internal/ui/app.go` (replace placeholder)
- Write: `internal/ui/keys.go`

This is the core of the application. Read carefully before implementing.

### Key design decisions

1. **Keyboard passthrough**: When `sidebarFocused` is false and no overlay is shown, all keypresses are forwarded to the active tmux session via `tmux send-keys`, EXCEPT:
   - `\` → toggle sidebar (always intercepted)
   - `;` → leader key (always intercepted, arms `leaderActive`)
   - When `leaderActive`, next key is a TUI command

2. **Session focus**: The sidebar is always shown (when `sidebarOpen`). "Focus" means keyboard input goes to the active session's tmux pane.

3. **Polling**: A `tea.Tick` fires every 150ms to capture pane output for the active session and update status for all sessions.

4. **Numbering**: Sessions are numbered 1-N by slice position. Pressing `1` switches to sessions[0], etc.

**Step 1: Write keys.go**

```go
// internal/ui/keys.go
package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"claudetop/internal/tmux"
)

// tmuxKeyName maps a bubbletea key to a tmux send-keys name.
// Returns empty string if the key should be sent literally.
func tmuxKeyName(msg tea.KeyMsg) (name string, literal bool) {
	switch msg.Type {
	case tea.KeyEnter:
		return "Enter", false
	case tea.KeyBackspace:
		return "BSpace", false
	case tea.KeyDelete:
		return "DC", false
	case tea.KeyUp:
		return "Up", false
	case tea.KeyDown:
		return "Down", false
	case tea.KeyLeft:
		return "Left", false
	case tea.KeyRight:
		return "Right", false
	case tea.KeyTab:
		return "Tab", false
	case tea.KeyEsc:
		return "Escape", false
	case tea.KeyCtrlC:
		return "C-c", false
	case tea.KeyCtrlD:
		return "C-d", false
	case tea.KeyCtrlZ:
		return "C-z", false
	case tea.KeyCtrlL:
		return "C-l", false
	case tea.KeyCtrlA:
		return "C-a", false
	case tea.KeyCtrlE:
		return "C-e", false
	case tea.KeyCtrlU:
		return "C-u", false
	case tea.KeyCtrlK:
		return "C-k", false
	case tea.KeyCtrlW:
		return "C-w", false
	case tea.KeyRunes:
		return msg.String(), true // send literally
	default:
		return msg.String(), true
	}
}

// forwardKey sends a keypress to the active tmux session.
func forwardKey(sessionID string, msg tea.KeyMsg) tea.Cmd {
	return func() tea.Msg {
		name, literal := tmuxKeyName(msg)
		if name == "" {
			return nil
		}
		var err error
		if literal {
			err = tmux.SendLiteralKey(sessionID, name)
		} else {
			err = tmux.SendKeys(sessionID, name)
		}
		if err != nil {
			return errMsg{err}
		}
		return nil
	}
}
```

**Step 2: Write app.go**

```go
// internal/ui/app.go
package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"claudetop/internal/config"
	"claudetop/internal/session"
	"claudetop/internal/state"
	"claudetop/internal/tmux"
)

// Messages

type tickMsg time.Time
type paneContentMsg struct {
	sessionID string
	content   string
}
type errMsg struct{ err error }

// Model

type overlay int

const (
	overlayNone overlay = iota
	overlayHelp
	overlayNewSession
	overlayCloseConfirm
)

type Model struct {
	cfg     *config.Config
	store   *state.State
	sessions []*session.Session

	activeIdx    int  // index into sessions slice
	sidebarOpen  bool
	leaderActive bool
	overlay      overlay
	tick         int  // animation frame counter

	nameInput    textinput.Model
	viewport     viewport.Model

	width  int
	height int
}

func New(cfg *config.Config, st *state.State) *Model {
	return &Model{
		cfg:         cfg,
		store:       st,
		sessions:    st.Sessions,
		activeIdx:   -1,
		sidebarOpen: true,
		nameInput:   newSessionInput(),
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		tickCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeViewport()
		return m, nil

	case tickMsg:
		m.tick++
		cmds := []tea.Cmd{tickCmd()}

		// Poll active session pane content
		if m.activeIdx >= 0 && m.activeIdx < len(m.sessions) {
			s := m.sessions[m.activeIdx]
			if !s.Dead {
				cmds = append(cmds, m.captureActivePane(s.ID))
			}
		}

		// Update status for all sessions (every ~2s = every 13 ticks at 150ms)
		if m.tick%13 == 0 {
			for _, s := range m.sessions {
				if s.Dead {
					continue
				}
				newStatus := session.Detect(s.PaneContent, s.LastOutputAt, s.CreatedAt, s.Status)
				s.Status = newStatus
			}
		}

		return m, tea.Batch(cmds...)

	case paneContentMsg:
		for _, s := range m.sessions {
			if s.ID == msg.sessionID {
				if s.PaneContent != msg.content {
					s.PaneContent = msg.content
					s.LastOutputAt = time.Now()
				}
				if m.activeIdx >= 0 && m.sessions[m.activeIdx].ID == msg.sessionID {
					m.viewport.SetContent(msg.content)
					m.viewport.GotoBottom()
				}
				break
			}
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle overlays first
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

	// Leader key sequence
	if m.leaderActive {
		m.leaderActive = false
		return m.handleLeaderKey(msg)
	}

	// Always-intercepted keys (even when session is focused)
	switch msg.String() {
	case "\\":
		m.sidebarOpen = !m.sidebarOpen
		m.resizeViewport()
		return m, nil
	case ";":
		m.leaderActive = true
		return m, nil
	}

	// When sidebar is focused (no active session) or no sessions
	if m.activeIdx < 0 || len(m.sessions) == 0 {
		return m.handleSidebarKey(msg)
	}

	// Session is focused — sidebar navigation keys still work
	switch msg.String() {
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		idx := int(msg.String()[0]-'0') - 1
		if idx < len(m.sessions) {
			m.switchSession(idx)
		}
		return m, nil
	case "]":
		m.switchSession((m.activeIdx + 1) % len(m.sessions))
		return m, nil
	case "[":
		next := m.activeIdx - 1
		if next < 0 {
			next = len(m.sessions) - 1
		}
		m.switchSession(next)
		return m, nil
	case "n":
		m.overlay = overlayNewSession
		m.nameInput = newSessionInput()
		return m, textinput.Blink
	case "x":
		if m.activeIdx >= 0 {
			m.overlay = overlayCloseConfirm
		}
		return m, nil
	case "?":
		m.overlay = overlayHelp
		return m, nil
	}

	// Forward everything else to the active tmux session
	if m.activeIdx >= 0 && m.activeIdx < len(m.sessions) {
		s := m.sessions[m.activeIdx]
		if !s.Dead {
			return m, forwardKey(s.ID, msg)
		}
	}

	return m, nil
}

func (m *Model) handleLeaderKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "?":
		m.overlay = overlayHelp
	case "q":
		return m, tea.Quit
	case "Q":
		m.killAllSessions()
		return m, tea.Quit
	case "e":
		return m.openEditor()
	case "n":
		m.overlay = overlayNewSession
		m.nameInput = newSessionInput()
		return m, textinput.Blink
	}
	return m, nil
}

func (m *Model) handleSidebarKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		idx := int(msg.String()[0]-'0') - 1
		if idx < len(m.sessions) {
			m.switchSession(idx)
		}
	case "]", "j":
		if len(m.sessions) > 0 {
			next := (m.activeIdx + 1) % len(m.sessions)
			m.switchSession(next)
		}
	case "[", "k":
		if len(m.sessions) > 0 {
			next := m.activeIdx - 1
			if next < 0 {
				next = len(m.sessions) - 1
			}
			m.switchSession(next)
		}
	case "n":
		m.overlay = overlayNewSession
		m.nameInput = newSessionInput()
		return m, textinput.Blink
	case "x":
		if m.activeIdx >= 0 {
			m.overlay = overlayCloseConfirm
		}
	case "?":
		m.overlay = overlayHelp
	case ";":
		m.leaderActive = true
	}
	return m, nil
}

func (m *Model) handleNewSessionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		name := strings.TrimSpace(m.nameInput.Value())
		if name == "" {
			name = fmt.Sprintf("session-%d", len(m.sessions)+1)
		}
		m.overlay = overlayNone
		return m, m.spawnSession(name)

	case tea.KeyEsc:
		m.overlay = overlayNone
		return m, nil
	}

	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

func (m *Model) handleCloseConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.overlay = overlayNone
		if m.activeIdx >= 0 && m.activeIdx < len(m.sessions) {
			return m, m.closeSession(m.activeIdx)
		}
	default:
		m.overlay = overlayNone
	}
	return m, nil
}

// Commands

func (m *Model) captureActivePane(sessionID string) tea.Cmd {
	return func() tea.Msg {
		content, err := tmux.CapturePane(sessionID)
		if err != nil {
			return nil // session may have just started
		}
		return paneContentMsg{sessionID: sessionID, content: content}
	}
}

func (m *Model) spawnSession(name string) tea.Cmd {
	return func() tea.Msg {
		// Deduplicate name
		name = m.uniqueName(name)

		s := session.NewSession(name, len(m.sessions)+1)
		if err := tmux.Create(s.ID, m.cfg.General.RootDir); err != nil {
			return errMsg{err}
		}

		m.sessions = append(m.sessions, s)
		m.store.Sessions = m.sessions
		state.Save(m.store)

		m.switchSession(len(m.sessions) - 1)
		return nil
	}
}

func (m *Model) closeSession(idx int) tea.Cmd {
	return func() tea.Msg {
		s := m.sessions[idx]
		tmux.Kill(s.ID) // best effort

		m.sessions = append(m.sessions[:idx], m.sessions[idx+1:]...)
		m.store.Sessions = m.sessions
		state.Save(m.store)

		if m.activeIdx >= len(m.sessions) {
			m.activeIdx = len(m.sessions) - 1
		}
		if m.activeIdx >= 0 {
			m.loadSession(m.activeIdx)
		}
		return nil
	}
}

func (m *Model) killAllSessions() {
	for _, s := range m.sessions {
		tmux.Kill(s.ID)
	}
	m.sessions = nil
	m.store.Sessions = nil
	state.Save(m.store)
}

func (m *Model) openEditor() (tea.Model, tea.Cmd) {
	claudeMD := m.cfg.General.RootDir + "/CLAUDE.md"
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	cmd := exec.Command(editor, claudeMD)
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return nil
	})
}

// Helpers

func (m *Model) switchSession(idx int) {
	m.activeIdx = idx
	m.loadSession(idx)
}

func (m *Model) loadSession(idx int) {
	if idx < 0 || idx >= len(m.sessions) {
		return
	}
	s := m.sessions[idx]
	m.viewport.SetContent(s.PaneContent)
	m.viewport.GotoBottom()
}

func (m *Model) resizeViewport() {
	vpWidth := m.width
	if m.sidebarOpen {
		vpWidth -= sidebarWidth + 1 // +1 for separator
	}
	vpHeight := m.height - 2 // status bar + hint line
	if vpWidth < 1 {
		vpWidth = 1
	}
	if vpHeight < 1 {
		vpHeight = 1
	}
	m.viewport = newViewport(vpWidth, vpHeight)
	if m.activeIdx >= 0 && m.activeIdx < len(m.sessions) {
		m.viewport.SetContent(m.sessions[m.activeIdx].PaneContent)
		m.viewport.GotoBottom()
	}
}

func (m *Model) uniqueName(name string) string {
	existing := map[string]bool{}
	for _, s := range m.sessions {
		existing[s.ID] = true
	}
	candidate := name
	for i := 2; existing[candidate]; i++ {
		candidate = fmt.Sprintf("%s-%d", name, i)
	}
	return candidate
}

// View

func (m *Model) View() string {
	if m.width == 0 {
		return ""
	}

	// Handle overlays
	if m.overlay == overlayHelp {
		return renderHelp(m.width, m.height)
	}
	if m.overlay == overlayNewSession {
		return renderNewSession(m.nameInput, m.width, m.height)
	}
	if m.overlay == overlayCloseConfirm && m.activeIdx >= 0 {
		s := m.sessions[m.activeIdx]
		msg := fmt.Sprintf("Kill session %q? (y/N)", s.DisplayName())
		return renderConfirm(msg, m.width, m.height)
	}

	// Status bar
	statusBar := renderStatusBar(m.sessions, m.width)

	// Main content area
	mainHeight := m.height - 2

	var mainContent string
	if m.sidebarOpen {
		sidebar := renderSidebar(m.sessions, m.activeIdx, mainHeight, m.tick)
		vp := m.viewport.View()
		sep := lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("240")).
			Render(strings.Repeat("│\n", mainHeight))
		mainContent = lipgloss.JoinHorizontal(lipgloss.Top, sidebar, sep, vp)
	} else {
		mainContent = m.viewport.View()
	}

	// Hint line
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Background(lipgloss.Color("0"))
	hint := hintStyle.Width(m.width).Render(" \\ sidebar   ;? help   n new   x close")

	return lipgloss.JoinVertical(lipgloss.Left, statusBar, mainContent, hint)
}

// renderConfirm renders a simple confirmation prompt overlay.
func renderConfirm(msg string, width, height int) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Background(lipgloss.Color("235")).
		Padding(1, 2)

	content := style.Render(msg)
	leftPad := (width - lipgloss.Width(content)) / 2
	topPad := (height - lipgloss.Height(content)) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	if topPad < 0 {
		topPad = 0
	}

	var result []string
	for i := 0; i < topPad; i++ {
		result = append(result, "")
	}
	for _, line := range strings.Split(content, "\n") {
		result = append(result, strings.Repeat(" ", leftPad)+line)
	}
	return strings.Join(result, "\n")
}
```

**Step 3: Verify it compiles**

```bash
go build ./internal/ui/...
```

Fix any compilation errors before proceeding.

**Step 4: Update PLAN.md** — mark Task 12 done

**Step 5: Commit**

```bash
git add internal/ui/
git commit -m "feat: main app model with Bubbletea wiring"
```

---

## Task 13: Main entry point

**Files:**
- Write: `main.go` (replace placeholder)

**Step 1: Write main.go**

```go
// main.go
package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"claudetop/internal/config"
	"claudetop/internal/session"
	"claudetop/internal/state"
	"claudetop/internal/tmux"
	"claudetop/internal/ui"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "claudetop: config error: %v\n", err)
		os.Exit(1)
	}

	if err := config.EnsureDir(); err != nil {
		fmt.Fprintf(os.Stderr, "claudetop: cannot create config dir: %v\n", err)
		os.Exit(1)
	}

	// Load persisted state
	st, err := state.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "claudetop: state error: %v\n", err)
		os.Exit(1)
	}

	// Reconnect or mark dead
	reconnect(st)

	// Build and run the TUI
	m := ui.New(cfg, st)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

// reconnect checks each persisted session against live tmux sessions.
// Sessions whose tmux counterpart is gone are marked dead.
func reconnect(st *state.State) {
	live, _ := tmux.LiveSessions()
	liveSet := map[string]bool{}
	for _, id := range live {
		liveSet[id] = true
	}

	for _, s := range st.Sessions {
		if liveSet[s.ID] {
			s.Status = session.StatusDone // will be updated on first tick
			s.Dead = false
		} else {
			s.Dead = true
		}
	}

	// Remove dead sessions from state (clean up automatically)
	var alive []*session.Session
	for _, s := range st.Sessions {
		if !s.Dead {
			alive = append(alive, s)
		}
	}
	st.Sessions = alive
}
```

**Step 2: Verify it compiles**

```bash
go build -o claudetop .
```

Expected: binary produced, no errors.

**Step 3: Smoke test — launch it**

```bash
./claudetop
```

Expected: TUI opens, empty sidebar, status bar visible. Press `;q` to quit.

**Step 4: Create a minimal config for testing**

```bash
mkdir -p ~/.claudetop
cat > ~/.claudetop/config.toml << 'EOF'
[general]
root_dir = "/tmp"
EOF
```

**Step 5: Test the basic flow**

Run `./claudetop`, then:
- Press `n` → new session overlay appears
- Type a name → press Enter → session should spawn (visible in sidebar)
- Press `?` → help overlay appears → press Esc
- Press `\` → sidebar toggles
- Press `;q` → exits

**Step 6: Update PLAN.md** — mark Task 13 done

**Step 7: Commit**

```bash
git add main.go
git commit -m "feat: main entry point with reconnect logic"
```

---

## Task 14: Integration testing

### Setup

**Step 1: Set root_dir to a real directory**

```bash
cat > ~/.claudetop/config.toml << 'EOF'
[general]
root_dir = "/Users/andersnordmark/work/personal/claudetop"
EOF
```

**Step 2: Build fresh binary**

```bash
go build -o claudetop . && echo "build ok"
```

### Verification checklist

Run `./claudetop` and verify each item. Fix failures before proceeding.

**Step 3: Session spawns in correct root_dir**
- Press `n`, type `test-one`, Enter
- In a separate terminal: `tmux ls` — should see `ct-test-one`
- In sidebar: session appears
- In viewport: claude output visible

**Step 4: Status dot appears and changes**
- Create session, watch dot change from grey (starting) to yellow (working) to green (done)
- While claude is running: dot should be yellow/animated

**Step 5: 1-9 and ]/[ switch sessions**
- Create 3 sessions: test-one, test-two, test-three
- Press `1` → session 1 active
- Press `2` → session 2 active
- Press `]` → session 3 active
- Press `[` → back to session 2

**Step 6: \ toggles sidebar**
- Press `\` → sidebar hides, viewport expands
- Press `\` again → sidebar reappears

**Step 7: x closes session with confirmation**
- Press `x` → confirmation prompt appears with session name
- Press `y` → session disappears from sidebar
- Verify in terminal: `tmux ls` should not show the closed session

**Step 8: Status bar shows correct counts**
- With 3 sessions: status bar shows "3 sessions"
- When a session needs input: status bar shows "1 needs input" in red

**Step 9: TUI restart reconnects**
- Create 2 sessions, press `;q` to quit
- Run `./claudetop` again
- Both sessions should reappear in the sidebar

**Step 10: ? shows help overlay**
- Press `?` → help overlay appears with keybindings
- Press `Esc` → overlay closes

**Step 11: Config file is read correctly**
- Change root_dir in config to `/tmp`
- Restart claudetop, create a new session
- Verify: `tmux display-message -t ct-<name> -p '#{pane_current_path}'` should show `/tmp`

**Step 12: Test with 3 concurrent sessions**
- Create sessions: alpha, beta, gamma
- Type in alpha (forwarded to claude)
- Switch to beta, type
- Switch to gamma, type
- All three should have independent content

**Step 13: ;e opens CLAUDE.md**
- Press `;e` → $EDITOR opens with root CLAUDE.md
- Make a change, save and exit → back in claudetop

**Step 14: Update PLAN.md** — mark all checklist items verified, mark Task 14 done

**Step 15: Commit**

```bash
git add PLAN.md
git commit -m "test: integration testing complete - all Iteration 1 features verified"
```

---

## Task 15: Phase 4 review

Re-read the entire codebase with fresh eyes. Go through Section 9 (Keybindings) and Section 7 (Status System) of the spec against the implementation.

**Step 1: Check keybinding coverage**

Open `internal/ui/app.go` and verify:
- [ ] `\` toggles sidebar
- [ ] `1`-`9` switch sessions
- [ ] `]` next session (wraps)
- [ ] `[` previous session (wraps)
- [ ] `n` new blank session
- [ ] `x` close session with confirmation
- [ ] `?` help overlay
- [ ] `;q` quit (sessions keep running)
- [ ] `;Q` quit + kill all
- [ ] `;e` open CLAUDE.md in $EDITOR

**Step 2: Check status detection coverage**

Open `internal/session/status.go` and verify patterns cover:
- [ ] Starting (first 10s)
- [ ] Working (active output)
- [ ] Needs input (question patterns)
- [ ] Permission (y/N prompt)
- [ ] Done (idle after completion)
- [ ] Stuck (working + silent >2min)
- [ ] Error (error patterns)

**Step 3: Edge cases**
- [ ] What if tmux is not installed? → binary should fail with a clear error
- [ ] What if root_dir doesn't exist? → show error when creating session
- [ ] What if 0 sessions? → sidebar shows helpful message with `n` hint
- [ ] What if session N doesn't exist when pressing `5`? → silently ignore
- [ ] Session name collision → uniqueName() appends number suffix

**Step 4: Add tmux check at startup**

In `main.go`, add before everything else:

```go
// Verify tmux is available
if _, err := exec.LookPath("tmux"); err != nil {
    fmt.Fprintln(os.Stderr, "claudetop: tmux is required but not found in PATH")
    os.Exit(1)
}
```

Import `os/exec` in main.go.

**Step 5: Add root_dir validation in spawnSession**

In `app.go`'s `spawnSession`, before calling tmux.Create:

```go
if _, err := os.Stat(m.cfg.General.RootDir); err != nil {
    return errMsg{fmt.Errorf("root_dir %q does not exist", m.cfg.General.RootDir)}
}
```

**Step 6: Build and run**

```bash
go build -o claudetop . && ./claudetop
```

Fix any issues found. Record deviations in PLAN.md.

**Step 7: Update PLAN.md** — mark Task 15 done

**Step 8: Commit all fixes**

```bash
git add -p  # stage only relevant changes
git commit -m "fix: phase 4 review - edge cases and keybinding coverage"
```

---

## Task 16: Phase 5 review

Assume Phase 4 introduced bugs. Check with fresh eyes.

**Step 1: Test Phase 4 fixes**
- Verify tmux-not-found error message
- Verify root_dir-not-exist error when creating session
- Test 0 sessions startup (clear state.json first)

**Step 2: Regression check**
- Create 3 sessions, switch between them, close one, restart TUI
- All basic flows should still work

**Step 3: Code readability**
- Read through each file in `internal/ui/` — are function names clear?
- Are error messages helpful? (e.g. when tmux session creation fails)
- Is the polling logic easy to understand?

**Step 4: Fix anything found**

**Step 5: Update PLAN.md** — mark Task 16 done

**Step 6: Commit**

```bash
git add .
git commit -m "fix: phase 5 review - regressions and readability"
```

---

## Task 17: Phase 6 final + DONE.md

**Step 1: Clean build**

```bash
go clean -cache && go build -o claudetop . && echo "clean build ok"
```

Expected: no errors, no warnings.

**Step 2: Run all unit tests**

```bash
go test ./...
```

Expected: all pass.

**Step 3: Read PLAN.md** — confirm every item is marked done

**Step 4: Write DONE.md**

Create `DONE.md` in the project root:

```markdown
# claudetop Iteration 1 — Done

## What was built

- Fullscreen TUI for managing Claude Code sessions backed by tmux
- Session creation with `n` key, spawning `claude` in configured `root_dir`
- Sidebar with status dots (toggle with `\`)
- Status detection: starting / working (animated) / needs input / permission / done / stuck / error
- Session switching: `1`-`9`, `]`/`[`
- Session close: `x` with confirmation
- Status bar: session count, needs-input count, clock
- TUI restart reconnects to existing tmux sessions via state.json
- Config: `~/.claudetop/config.toml` with `root_dir`
- Help overlay: `?` or `;?`
- Keyboard passthrough: all keys forwarded to tmux pane when session focused
- Leader key: `;` prefix for TUI commands while session is focused
- BONUS: `;e` opens root CLAUDE.md in $EDITOR

## What works

All Iteration 1 success criteria verified:
- Sessions spawn in correct root_dir
- Sessions survive TUI restart (tmux + state.json)
- Status dots reflect actual Claude Code state
- Tested with 3 concurrent sessions

## Known limitations

- Status detection is heuristic (pattern-matching on pane output). Will occasionally misclassify.
  Tune patterns over time based on real Claude Code output.
- Pane rendering: captured via `tmux capture-pane -p -e`. ANSI escape sequences may not
  render perfectly in all terminal emulators. Focused session output may lag ~150ms.
- Session numbering shifts when a session is closed (session 2 of 3 becomes session 2 of 2).
  This is expected behavior.

## Deviations from spec

(See PLAN.md for details)

## Suggested next steps for Iteration 2

1. Auto-naming from first prompt (Section 8.2)
2. Manual rename with `r`
3. Repo tagging (Section 8.3)
4. Parked sessions (Section 8.4)
5. New session with prompt editor (`N`)
6. Split view (Section 6.1)
```

**Step 5: Commit everything**

```bash
git add DONE.md PLAN.md
git commit -m "docs: DONE.md - iteration 1 complete"
```

**Step 6: Final smoke test**

```bash
./claudetop
```

Create a session, verify it works, quit cleanly.

---

*Plan complete. All phases documented.*
