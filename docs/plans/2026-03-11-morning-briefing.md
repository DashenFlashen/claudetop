# Morning Briefing Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Show a morning context screen on first daily launch with yesterday's git commits, inbox, parked sessions, standup draft (on demand), and a priorities text input that saves to `~/.claudetop/today.md`.

**Architecture:** Six tasks in dependency order: state/config changes first, then two new packages (git log discovery, today.md writer), then a renderer, then full app wiring, then help/hint updates. The briefing is a full-screen mode (`showBriefing bool` on Model), not an overlay — it replaces the entire View() while active. Inbox overlay works on top of it unchanged.

**Tech Stack:** Go, Bubbletea (bubbletea, bubbles/textinput), lipgloss, exec.Command for git log.

---

## Codebase Context

Read these files before starting any task:
- `internal/ui/app.go` — Model struct, overlay enum, Update/View patterns
- `internal/state/store.go` — State struct, InboxItem
- `internal/config/config.go` — Config, GeneralConfig
- `internal/ui/skilloutput.go` — example of how a full-screen overlay is rendered
- `internal/ui/inbox.go` — example rendering pattern

Module name: `claudetop`. Tests run with: `go test ./...`

---

## Task 1: State & Config changes

**Files:**
- Modify: `internal/state/store.go`
- Modify: `internal/config/config.go`
- Modify: `internal/state/store_test.go`

### Step 1: Write the failing tests

Add to `internal/state/store_test.go`:

```go
func TestLastBriefingDatePersists(t *testing.T) {
    dir := t.TempDir()
    t.Setenv("HOME", dir)

    s := &State{
        Sessions:         []*session.Session{},
        InboxItems:       []*InboxItem{},
        LastBriefingDate: "2026-03-11",
    }
    if err := Save(s); err != nil {
        t.Fatalf("save: %v", err)
    }

    loaded, err := Load()
    if err != nil {
        t.Fatalf("load: %v", err)
    }
    if loaded.LastBriefingDate != "2026-03-11" {
        t.Errorf("got %q, want %q", loaded.LastBriefingDate, "2026-03-11")
    }
}
```

Add to `internal/config/config.go` (no test file exists yet, skip config test — the default is exercised by the existing load test pattern):

### Step 2: Run tests to verify they fail

```bash
go test ./internal/state/...
```

Expected: FAIL with "cannot use literal (no field LastBriefingDate)"

### Step 3: Add `LastBriefingDate` to State

In `internal/state/store.go`, change the `State` struct:

```go
// State is the persisted application state.
type State struct {
	Sessions         []*session.Session `json:"sessions"`
	InboxItems       []*InboxItem       `json:"inbox_items,omitempty"`
	LastBriefingDate string             `json:"last_briefing_date,omitempty"`
}
```

### Step 4: Add `AutoBriefing` to GeneralConfig

In `internal/config/config.go`, change `GeneralConfig`:

```go
type GeneralConfig struct {
	RootDir      string `toml:"root_dir"`
	AutoBriefing bool   `toml:"auto_briefing"`
}
```

And in the `Load()` function, set the default:

```go
cfg := &Config{
    General: GeneralConfig{
        RootDir:      filepath.Dir(dir), // home is parent of ~/.claudetop
        AutoBriefing: true,
    },
}
```

### Step 5: Run tests to verify they pass

```bash
go test ./internal/state/... ./internal/config/...
```

Expected: PASS

### Step 6: Commit

```bash
git add internal/state/store.go internal/config/config.go internal/state/store_test.go
git commit -m "feat: add LastBriefingDate to state and AutoBriefing to config"
```

---

## Task 2: Git log discovery package

**Files:**
- Create: `internal/git/log.go`
- Create: `internal/git/log_test.go`

### Step 1: Write the failing tests

Create `internal/git/log_test.go`:

```go
package git_test

import (
	"os"
	"path/filepath"
	"testing"

	"claudetop/internal/git"
)

func TestReposUnder(t *testing.T) {
	root := t.TempDir()

	// Create two git repos and one non-git dir
	for _, name := range []string{"repo-a", "repo-b"} {
		if err := os.MkdirAll(filepath.Join(root, name, ".git"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "not-a-repo"), 0755); err != nil {
		t.Fatal(err)
	}

	repos, err := git.ReposUnder(root)
	if err != nil {
		t.Fatalf("ReposUnder: %v", err)
	}

	if len(repos) != 2 {
		t.Errorf("got %d repos, want 2: %v", len(repos), repos)
	}
	for _, r := range repos {
		if r != "repo-a" && r != "repo-b" {
			t.Errorf("unexpected repo name: %q", r)
		}
	}
}

func TestParseGitLog(t *testing.T) {
	input := "fix timeout in pipeline\nadd retry logic to connector\n"
	commits := git.ParseGitLog(input, "annotation-service")

	if len(commits) != 2 {
		t.Fatalf("got %d commits, want 2", len(commits))
	}
	if commits[0].Repo != "annotation-service" {
		t.Errorf("got repo %q, want %q", commits[0].Repo, "annotation-service")
	}
	if commits[0].Message != "fix timeout in pipeline" {
		t.Errorf("got message %q, want %q", commits[0].Message, "fix timeout in pipeline")
	}
}

func TestParseGitLogEmpty(t *testing.T) {
	commits := git.ParseGitLog("", "my-repo")
	if len(commits) != 0 {
		t.Errorf("expected no commits for empty input, got %d", len(commits))
	}
}
```

### Step 2: Run tests to verify they fail

```bash
go test ./internal/git/...
```

Expected: FAIL with "no such package"

### Step 3: Create `internal/git/log.go`

```go
package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CommitSummary represents a single git commit from yesterday.
type CommitSummary struct {
	Repo    string
	Message string
}

// ReposUnder returns names of immediate subdirectories of rootDir that contain a .git directory.
func ReposUnder(rootDir string) ([]string, error) {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, err
	}
	var repos []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		gitPath := filepath.Join(rootDir, e.Name(), ".git")
		if _, err := os.Stat(gitPath); err == nil {
			repos = append(repos, e.Name())
		}
	}
	return repos, nil
}

// ParseGitLog parses the output of `git log --format=%s` for the given repo name.
// Exported so it can be tested in isolation.
func ParseGitLog(output, repoName string) []CommitSummary {
	var commits []CommitSummary
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		commits = append(commits, CommitSummary{
			Repo:    repoName,
			Message: line,
		})
	}
	return commits
}

// YesterdayCommits returns all commits from yesterday (local time) across all git repos under rootDir.
// Errors from individual repos are silently skipped — a repo without git history is not an error.
func YesterdayCommits(rootDir string) ([]CommitSummary, error) {
	repos, err := ReposUnder(rootDir)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	afterDate := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, time.Local).Format("2006-01-02")
	beforeDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).Format("2006-01-02")

	var all []CommitSummary
	for _, repoName := range repos {
		repoPath := filepath.Join(rootDir, repoName)
		out, err := exec.Command("git", "-C", repoPath,
			"log",
			"--after="+afterDate,
			"--before="+beforeDate,
			"--format=%s",
			"--no-merges",
		).Output()
		if err != nil {
			continue // not a git repo or no commits
		}
		all = append(all, ParseGitLog(string(out), repoName)...)
	}
	return all, nil
}
```

### Step 4: Run tests to verify they pass

```bash
go test ./internal/git/...
```

Expected: PASS

### Step 5: Commit

```bash
git add internal/git/log.go internal/git/log_test.go
git commit -m "feat: add git log discovery package for yesterday's commits"
```

---

## Task 3: today.md writer

**Files:**
- Create: `internal/briefing/priorities.go`
- Create: `internal/briefing/priorities_test.go`

### Step 1: Write the failing tests

Create `internal/briefing/priorities_test.go`:

```go
package briefing_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"claudetop/internal/briefing"
)

func TestWritePriorities(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	path, err := briefing.WritePriorities("Resolve annotation timeout, Move CSV forward, Process inbox")
	if err != nil {
		t.Fatalf("WritePriorities: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	content := string(data)
	today := time.Now().Format("2006-01-02")
	if !strings.Contains(content, today) {
		t.Errorf("expected today's date %q in output, got:\n%s", today, content)
	}
	if !strings.Contains(content, "1. Resolve annotation timeout") {
		t.Errorf("expected numbered item 1, got:\n%s", content)
	}
	if !strings.Contains(content, "2. Move CSV forward") {
		t.Errorf("expected numbered item 2, got:\n%s", content)
	}
	if !strings.Contains(content, "3. Process inbox") {
		t.Errorf("expected numbered item 3, got:\n%s", content)
	}
}

func TestWritePrioritiesEmptyLinesSkipped(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	path, err := briefing.WritePriorities("item one, , item two")
	if err != nil {
		t.Fatalf("WritePriorities: %v", err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)
	if strings.Contains(content, "2. ") && !strings.Contains(content, "2. item two") {
		t.Errorf("empty item should be skipped: %s", content)
	}
}
```

### Step 2: Run tests to verify they fail

```bash
go test ./internal/briefing/...
```

Expected: FAIL with "no such package"

### Step 3: Create `internal/briefing/priorities.go`

```go
package briefing

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"claudetop/internal/config"
)

// WritePriorities saves today's priorities to ~/.claudetop/today.md.
// text is split on commas; each non-empty item becomes a numbered list entry.
// Returns the path written.
func WritePriorities(text string) (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create dir: %w", err)
	}

	path := filepath.Join(dir, "today.md")

	var items []string
	for _, part := range strings.Split(text, ",") {
		item := strings.TrimSpace(part)
		if item != "" {
			items = append(items, item)
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Priorities — %s\n\n", time.Now().Format("2006-01-02")))
	if len(items) == 0 {
		sb.WriteString(strings.TrimSpace(text) + "\n")
	} else {
		for i, item := range items {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, item))
		}
	}

	if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil {
		return "", fmt.Errorf("write today.md: %w", err)
	}
	return path, nil
}
```

### Step 4: Run tests to verify they pass

```bash
go test ./internal/briefing/...
```

Expected: PASS

### Step 5: Commit

```bash
git add internal/briefing/priorities.go internal/briefing/priorities_test.go
git commit -m "feat: add today.md writer for daily priorities"
```

---

## Task 4: Briefing renderer

**Files:**
- Create: `internal/ui/briefing.go`

No new tests for this task — it's a pure render function. Visual correctness is verified by running the app.

### Step 1: Create `internal/ui/briefing.go`

The renderer builds all "content lines" (everything above the divider), applies a scroll offset, and assembles the full screen by joining content area + fixed bottom section.

Bottom section is always 4 lines tall:
1. Divider
2. "TODAY'S PRIORITIES" label
3. Text input
4. Hint bar

Content area height = `height - 4`. Lines are sliced from `allLines[scrollOffset:]` up to content area height.

```go
package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"

	"claudetop/internal/git"
	"claudetop/internal/session"
	"claudetop/internal/state"
)

var (
	briefingBg = lipgloss.Color("232")

	briefingHeaderStyle = lipgloss.NewStyle().
				Background(briefingBg).
				Foreground(lipgloss.Color("255")).
				Bold(true).
				Padding(0, 2)

	briefingSectionStyle = lipgloss.NewStyle().
				Background(briefingBg).
				Foreground(lipgloss.Color("39")).
				Bold(true).
				Padding(0, 2)

	briefingDividerStyle = lipgloss.NewStyle().
				Background(briefingBg).
				Foreground(lipgloss.Color("238"))

	briefingTextStyle = lipgloss.NewStyle().
				Background(briefingBg).
				Foreground(lipgloss.Color("250")).
				Padding(0, 2)

	briefingDimStyle = lipgloss.NewStyle().
				Background(briefingBg).
				Foreground(lipgloss.Color("240")).
				Padding(0, 2)

	briefingHintStyle = lipgloss.NewStyle().
				Background(briefingBg).
				Foreground(lipgloss.Color("240")).
				Padding(0, 2)

	briefingPriorityLabelStyle = lipgloss.NewStyle().
					Background(briefingBg).
					Foreground(lipgloss.Color("220")).
					Bold(true).
					Padding(0, 2)

	briefingInputStyle = lipgloss.NewStyle().
				Background(briefingBg).
				Padding(0, 2)

	briefingDividerLineStyle = lipgloss.NewStyle().
					Background(briefingBg).
					Foreground(lipgloss.Color("236"))
)

// renderBriefing renders the full-screen morning briefing.
func renderBriefing(
	commits []git.CommitSummary,
	commitsLoading bool,
	inboxItems []*state.InboxItem,
	parkedSessions []*session.Session,
	standupOutput string,
	standupRunning bool,
	prioritiesInput textinput.Model,
	prioritiesFocused bool,
	scrollOffset int,
	width, height, tick int,
) string {
	bg := lipgloss.NewStyle().Background(briefingBg).Width(width)

	// Build all content lines (scrollable region)
	var lines []string

	// Header
	lines = append(lines, "")
	day := time.Now().Format("Monday 2 January")
	lines = append(lines, briefingHeaderStyle.Width(width).Render("GOOD MORNING  ·  "+day))
	lines = append(lines, "")

	// YESTERDAY section
	lines = append(lines, briefingSectionStyle.Width(width).Render("YESTERDAY"))
	lines = append(lines, briefingDividerStyle.Width(width).Render(strings.Repeat("─", width-4)))
	if commitsLoading {
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		lines = append(lines, briefingDimStyle.Width(width).Render(frames[tick%len(frames)]+"  Scanning repos..."))
	} else if len(commits) == 0 {
		lines = append(lines, briefingDimStyle.Width(width).Render("(no commits yesterday)"))
	} else {
		maxCommits := 10
		shown := commits
		if len(shown) > maxCommits {
			shown = shown[:maxCommits]
		}
		for _, c := range shown {
			line := fmt.Sprintf("%-20s · %s", c.Repo, c.Message)
			if len(line) > width-6 {
				line = line[:width-9] + "..."
			}
			lines = append(lines, briefingTextStyle.Width(width).Render(line))
		}
		if len(commits) > maxCommits {
			lines = append(lines, briefingDimStyle.Width(width).Render(fmt.Sprintf("+ %d more", len(commits)-maxCommits)))
		}
	}
	lines = append(lines, "")

	// INBOX section (only if non-empty)
	active := activeInboxItems(inboxItems)
	if len(active) > 0 {
		lines = append(lines, briefingSectionStyle.Width(width).Render(fmt.Sprintf("INBOX  %d items", len(active))))
		lines = append(lines, briefingDividerStyle.Width(width).Render(strings.Repeat("─", width-4)))
		maxShow := 2
		for i, item := range active {
			if i >= maxShow {
				break
			}
			content := item.Content
			if len([]rune(content)) > width-10 {
				runes := []rune(content)
				content = string(runes[:width-13]) + "…"
			}
			src := ""
			if item.Source != "manual" {
				src = "  — " + item.Source
			}
			lines = append(lines, briefingTextStyle.Width(width).Render("· "+content+src))
		}
		if len(active) > maxShow {
			lines = append(lines, briefingDimStyle.Width(width).Render(fmt.Sprintf("+ %d more  ·  b to open inbox", len(active)-maxShow)))
		} else {
			lines = append(lines, briefingDimStyle.Width(width).Render("b to open inbox"))
		}
		lines = append(lines, "")
	}

	// PARKED section (only if non-empty)
	if len(parkedSessions) > 0 {
		lines = append(lines, briefingSectionStyle.Width(width).Render("PARKED"))
		lines = append(lines, briefingDividerStyle.Width(width).Render(strings.Repeat("─", width-4)))
		for _, s := range parkedSessions {
			note := ""
			if s.ParkNote != "" {
				note = "  "" + s.ParkNote + """
			}
			lines = append(lines, briefingTextStyle.Width(width).Render("· "+s.DisplayName()+note))
		}
		lines = append(lines, "")
	}

	// STANDUP DRAFT section
	standupHint := "s to generate"
	if standupRunning {
		standupHint = "generating..."
	} else if standupOutput != "" {
		standupHint = "s to regenerate"
	}
	header := fmt.Sprintf("%-*s%s", width/2, "STANDUP DRAFT", standupHint)
	lines = append(lines, briefingSectionStyle.Width(width).Render(header))
	lines = append(lines, briefingDividerStyle.Width(width).Render(strings.Repeat("─", width-4)))
	if standupRunning {
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		lines = append(lines, briefingDimStyle.Width(width).Render(frames[tick%len(frames)]+"  Generating standup draft..."))
	} else if standupOutput == "" {
		lines = append(lines, briefingDimStyle.Width(width).Render("(not yet generated)"))
	} else {
		for _, l := range strings.Split(strings.TrimSpace(standupOutput), "\n") {
			lines = append(lines, briefingTextStyle.Width(width).Render(l))
		}
	}
	lines = append(lines, "")

	// Apply scroll offset and clip to content area height
	contentAreaHeight := height - 4 // reserve 4 lines for bottom section
	if contentAreaHeight < 1 {
		contentAreaHeight = 1
	}
	if scrollOffset > len(lines)-1 {
		scrollOffset = len(lines) - 1
	}
	if scrollOffset < 0 {
		scrollOffset = 0
	}
	visible := lines[scrollOffset:]
	if len(visible) > contentAreaHeight {
		visible = visible[:contentAreaHeight]
	}
	// Pad remaining lines with blank styled lines
	for len(visible) < contentAreaHeight {
		visible = append(visible, bg.Render(""))
	}

	// Bottom section (fixed, always visible)
	divider := briefingDividerLineStyle.Width(width).Render(strings.Repeat("─", width))
	label := briefingPriorityLabelStyle.Width(width).Render("TODAY'S PRIORITIES")
	inputLine := briefingInputStyle.Width(width).Render("> " + prioritiesInput.View())
	hint := "Tab: focus · Enter: save · Esc: skip · j/k: scroll · s: standup · b: inbox"
	hintLine := briefingHintStyle.Width(width).Render(hint)

	return strings.Join(append(visible, divider, label, inputLine, hintLine), "\n")
}

// newBriefingPrioritiesInput returns a configured textinput for the briefing priorities field.
func newBriefingPrioritiesInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "e.g. Resolve annotation timeout, move CSV export forward"
	ti.CharLimit = 300
	ti.Width = 70
	return ti
}
```

### Step 2: Verify it compiles

```bash
go build ./...
```

Expected: no errors

### Step 3: Commit

```bash
git add internal/ui/briefing.go
git commit -m "feat: add morning briefing renderer"
```

---

## Task 5: Wire briefing into app.go

**Files:**
- Modify: `internal/ui/app.go`

This is the largest task. Read `internal/ui/app.go` carefully before starting. Make changes in this order: message types → Model fields → New() → Init() → Update() → handleKey() → handleBriefingKey() → View().

### Step 1: Add imports and message types

At the top of `internal/ui/app.go`, add to the import block:
```go
"claudetop/internal/briefing"
"claudetop/internal/git"
```

After the existing message types (after `type pendingInboxSend struct`), add:

```go
type gitCommitsMsg struct {
	commits []git.CommitSummary
	err     error
}

type briefingStandupMsg struct {
	output string
	err    error
}
```

### Step 2: Add Model fields

In the `Model` struct, after `pendingInboxSend *pendingInboxSend`, add:

```go
showBriefing              bool
briefingScrollOffset      int
briefingCommits           []git.CommitSummary
briefingCommitsLoading    bool
briefingStandupOutput     string
briefingStandupRunning    bool
briefingPrioritiesInput   textinput.Model
briefingPrioritiesFocused bool
```

### Step 3: Update New() to trigger briefing on first daily open

In the `New()` function, after the existing `m.sidebarCursor` assignment block, add:

```go
today := time.Now().Format("2006-01-02")
if cfg.General.AutoBriefing && st.LastBriefingDate != today {
    m.showBriefing = true
    m.briefingCommitsLoading = true
    m.briefingPrioritiesInput = newBriefingPrioritiesInput()
}
```

### Step 4: Update Init() to fetch git commits when briefing is shown

Replace the current `Init()`:

```go
func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{tickCmd()}
	if m.showBriefing {
		cmds = append(cmds, fetchGitCommitsCmd(m.cfg.General.RootDir))
	}
	return tea.Batch(cmds...)
}
```

### Step 5: Handle new message types in Update()

In `Update()`, after the `case flushKeyMsg:` block and before `case tea.KeyMsg:`, add:

```go
case gitCommitsMsg:
    m.briefingCommitsLoading = false
    if msg.err == nil {
        m.briefingCommits = msg.commits
    }
    return m, nil

case briefingStandupMsg:
    m.briefingStandupRunning = false
    if msg.err != nil {
        m.briefingStandupOutput = "Error: " + msg.err.Error()
    } else {
        m.briefingStandupOutput = msg.output
    }
    return m, nil
```

### Step 6: Add briefing check in handleKey()

In `handleKey()`, after the overlay switch block and BEFORE the `if msg.Type == tea.KeyTab {` check, add:

```go
// Briefing mode (no overlay active)
if m.showBriefing {
    return m.handleBriefingKey(msg)
}
```

### Step 7: Add handleBriefingKey()

Add this method after `handleKey()`:

```go
func (m *Model) handleBriefingKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.briefingPrioritiesFocused {
		switch msg.Type {
		case tea.KeyEnter:
			text := strings.TrimSpace(m.briefingPrioritiesInput.Value())
			if text != "" {
				if err := briefing.WritePriorities(text); err != nil {
					m.setStatusMsg("Error: " + err.Error())
				}
			}
			m.closeBriefing()
			return m, nil
		case tea.KeyEsc:
			m.briefingPrioritiesFocused = false
			m.briefingPrioritiesInput.Blur()
			return m, nil
		}
		var cmd tea.Cmd
		m.briefingPrioritiesInput, cmd = m.briefingPrioritiesInput.Update(msg)
		return m, cmd
	}

	switch msg.Type {
	case tea.KeyEsc:
		m.closeBriefing()
		return m, nil
	case tea.KeyTab:
		m.briefingPrioritiesFocused = true
		m.briefingPrioritiesInput.Focus()
		return m, textinput.Blink
	}

	switch msg.String() {
	case "j":
		m.briefingScrollOffset++
	case "k":
		if m.briefingScrollOffset > 0 {
			m.briefingScrollOffset--
		}
	case "s":
		if !m.briefingStandupRunning {
			m.briefingStandupRunning = true
			m.briefingStandupOutput = ""
			return m, m.runBriefingStandup()
		}
	case "b":
		m.inboxCursor = 0
		m.overlay = overlayInbox
	}
	return m, nil
}

func (m *Model) closeBriefing() {
	m.showBriefing = false
	today := time.Now().Format("2006-01-02")
	m.store.LastBriefingDate = today
	m.saveState()
}

func (m *Model) runBriefingStandup() tea.Cmd {
	// Find the first skill with "standup" in the name (case-insensitive)
	var sk *config.SkillConfig
	for i := range m.cfg.Skills {
		if strings.Contains(strings.ToLower(m.cfg.Skills[i].Name), "standup") {
			sk = &m.cfg.Skills[i]
			break
		}
	}
	if sk == nil {
		return func() tea.Msg {
			return briefingStandupMsg{err: fmt.Errorf("no standup skill configured (add a skill with 'standup' in the name)")}
		}
	}
	command := sk.Command
	rootDir := m.cfg.General.RootDir
	return func() tea.Msg {
		parts := strings.Fields(command)
		if len(parts) == 0 {
			return briefingStandupMsg{err: fmt.Errorf("empty standup skill command")}
		}
		cmd := exec.Command(parts[0], parts[1:]...)
		cmd.Dir = rootDir
		out, err := cmd.CombinedOutput()
		return briefingStandupMsg{output: string(out), err: err}
	}
}
```

### Step 8: Add fetchGitCommitsCmd command function

After the existing command functions (e.g. after `runSkillOutput`), add:

```go
func fetchGitCommitsCmd(rootDir string) tea.Cmd {
	return func() tea.Msg {
		commits, err := git.YesterdayCommits(rootDir)
		return gitCommitsMsg{commits: commits, err: err}
	}
}
```

### Step 9: Update View() to render briefing

In `View()`, at the very top (after the `if m.width == 0` guard), add a briefing check BEFORE the overlay switch:

```go
// Briefing takes the full screen when active (overlays may still appear on top)
if m.showBriefing && m.overlay == overlayNone {
    parked := make([]*session.Session, 0)
    for _, s := range m.sessions {
        if s.Parked {
            parked = append(parked, s)
        }
    }
    return renderBriefing(
        m.briefingCommits, m.briefingCommitsLoading,
        m.store.InboxItems, parked,
        m.briefingStandupOutput, m.briefingStandupRunning,
        m.briefingPrioritiesInput, m.briefingPrioritiesFocused,
        m.briefingScrollOffset, m.width, m.height, m.tick,
    )
}
```

### Step 10: Verify it builds and all tests pass

```bash
go build ./... && go test ./...
```

Expected: builds cleanly, all tests pass

### Step 11: Commit

```bash
git add internal/ui/app.go
git commit -m "feat: wire morning briefing into app (trigger, keys, standup, git commits)"
```

---

## Task 6: Sidebar B key and help update

**Files:**
- Modify: `internal/ui/app.go` (handleSidebarKey)
- Modify: `internal/ui/help.go`

### Step 1: Add `B` to handleSidebarKey

In `handleSidebarKey()`, in the `switch msg.String()` block, add a new case after the `"b"` case:

```go
case "B":
    m.showBriefing = true
    m.briefingScrollOffset = 0
    m.briefingStandupOutput = ""
    m.briefingStandupRunning = false
    m.briefingCommitsLoading = true
    m.briefingPrioritiesInput = newBriefingPrioritiesInput()
    m.briefingPrioritiesFocused = false
    return m, fetchGitCommitsCmd(m.cfg.General.RootDir)
```

### Step 2: Add `B` and `b` hint to help entries

In `internal/ui/help.go`, in the `helpEntries` slice, add after the `{"b", "Open inbox"}` entry:

```go
{"B", "Reopen morning briefing"},
```

### Step 3: Run tests

```bash
go build ./... && go test ./...
```

Expected: PASS

### Step 4: Commit

```bash
git add internal/ui/app.go internal/ui/help.go
git commit -m "feat: add B key to reopen briefing, update help"
```

---

## Done

After all 6 tasks, run a final verification:

```bash
go build ./... && go test ./...
```

Then do a manual smoke test:
1. Delete or rename `~/.claudetop/state.json` so `LastBriefingDate` is empty
2. Run `./claudetop` — briefing should appear on startup
3. Press `s` — standup section should show generating spinner (or error if no standup skill)
4. Press `Tab` — cursor moves to priorities input
5. Type "item one, item two" and press Enter — briefing closes, check `~/.claudetop/today.md`
6. Restart claudetop — briefing should NOT appear (same day)
7. Press `Tab` to sidebar, press `B` — briefing reopens
