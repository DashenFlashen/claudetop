package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"claudetop/internal/config"
	"claudetop/internal/session"
	"claudetop/internal/state"
	"claudetop/internal/tmux"
)

var separatorStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("236")).
	Foreground(lipgloss.Color("240"))

// Internal message types

type tickMsg time.Time

type paneContentMsg struct {
	sessionID string
	content   string
}

type sessionSpawnedMsg struct {
	sess   *session.Session
	prompt string // empty for blank sessions
}

type sessionClosedMsg struct {
	sessionID string
	err       error // non-nil if tmux kill failed
}

type errMsg struct{ err error }

type sessionRenamedMsg struct {
	sessionID string
	name      string
}

// overlay represents which (if any) overlay is currently shown.
type overlay int

const (
	overlayNone overlay = iota
	overlayHelp
	overlayNewSession
	overlayCloseConfirm
	overlayPark
	overlayRename
	overlayPromptEditor
)

// Model is the root Bubbletea model.
type Model struct {
	cfg      *config.Config
	store    *state.State
	sessions []*session.Session

	activeIdx      int
	sidebarCursor  int  // navigation cursor in sidebar (may differ from activeIdx)
	sidebarOpen    bool
	sidebarFocused bool // true = sidebar has keyboard focus
	overlay        overlay
	tick           int       // animation frame counter
	statusMsg      string    // transient message shown in status bar
	statusMsgAt    time.Time // when statusMsg was set (for expiry)

	nameInput     textinput.Model
	parkInput     textinput.Model
	renameInput   textinput.Model
	promptInput   textarea.Model
	pendingPrompt string // prompt text waiting to be sent after session spawns
	viewport      viewport.Model

	width  int
	height int
}

// New creates the app model from loaded config and state.
func New(cfg *config.Config, st *state.State) *Model {
	m := &Model{
		cfg:         cfg,
		store:       st,
		sessions:    st.Sessions,
		activeIdx:   -1,
		sidebarOpen: true,
		nameInput:   newSessionInput(),
	}
	// Start on the first live session
	for i, s := range m.sessions {
		if !s.Dead {
			m.activeIdx = i
			break
		}
	}
	// Always start in sidebar mode so the user can navigate immediately
	m.sidebarFocused = true
	m.sidebarCursor = m.activeIdx
	if m.sidebarCursor < 0 {
		m.sidebarCursor = 0
	}
	return m
}

func (m *Model) Init() tea.Cmd {
	return tickCmd()
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
		const statusMsgTTL = 10 * time.Second
		if m.statusMsg != "" && time.Since(m.statusMsgAt) > statusMsgTTL {
			m.statusMsg = ""
		}
		cmds := []tea.Cmd{tickCmd()}

		// Poll active session pane content every tick
		if m.activeIdx >= 0 && m.activeIdx < len(m.sessions) {
			s := m.sessions[m.activeIdx]
			if !s.Dead {
				cmds = append(cmds, capturePane(s.ID))
			}
		}

		// Every ~2s: poll all background sessions for pane content + update all statuses
		if m.tick%13 == 0 {
			for i, s := range m.sessions {
				if s.Dead {
					continue
				}
				// Poll background sessions (active session already polled above at full rate)
				if i != m.activeIdx {
					cmds = append(cmds, capturePane(s.ID))
				}
				// Update status using latest known content
				s.Status = session.Detect(s.PaneContent, s.LastOutputAt, s.CreatedAt, s.Status)
			}
		}

		return m, tea.Batch(cmds...)

	case paneContentMsg:
		for _, s := range m.sessions {
			if s.ID == msg.sessionID {
				if s.PaneContent != msg.content {
					s.PaneContent = msg.content
					s.LastOutputAt = time.Now()
					s.Status = session.Detect(s.PaneContent, s.LastOutputAt, s.CreatedAt, s.Status)
				}
				// If this is the active session, refresh the viewport
				if m.activeIdx >= 0 && m.sessions[m.activeIdx].ID == msg.sessionID {
					m.viewport.SetContent(msg.content)
					m.viewport.GotoBottom()
				}
				break
			}
		}
		return m, nil

	case sessionSpawnedMsg:
		m.statusMsg = ""
		m.sessions = append(m.sessions, msg.sess)
		m.store.Sessions = m.sessions
		m.saveState()
		m.switchSession(len(m.sessions) - 1)
		m.sidebarCursor = len(m.sessions) - 1
		m.sidebarFocused = false

		var cmds []tea.Cmd
		if msg.prompt != "" {
			cmds = append(cmds, sendPrompt(msg.sess.ID, msg.prompt))
			if m.cfg.General.AutoNameSessions {
				cmds = append(cmds, autoName(msg.sess.ID, msg.prompt))
			}
		}
		return m, tea.Batch(cmds...)

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

	case sessionClosedMsg:
		if msg.err != nil {
			m.setStatusMsg("Error: " + msg.err.Error())
		}
		for i, s := range m.sessions {
			if s.ID == msg.sessionID {
				m.sessions = append(m.sessions[:i], m.sessions[i+1:]...)
				m.store.Sessions = m.sessions
				m.saveState()
				if len(m.sessions) == 0 {
					m.activeIdx = -1
				} else if m.activeIdx >= len(m.sessions) {
					m.activeIdx = len(m.sessions) - 1
					m.loadSession(m.activeIdx)
				} else {
					m.loadSession(m.activeIdx)
				}
				// If new active session is dead, find next live one
				if m.activeIdx >= 0 && m.activeIdx < len(m.sessions) && m.sessions[m.activeIdx].Dead {
					for i, s := range m.sessions {
						if !s.Dead {
							m.activeIdx = i
							break
						}
					}
					// activeIdx may have changed; reload viewport with correct session
					m.loadSession(m.activeIdx)
				}
				// Clamp cursor to valid range after removal
				if m.sidebarCursor >= len(m.sessions) {
					m.sidebarCursor = len(m.sessions) - 1
				}
				if m.sidebarCursor < 0 {
					m.sidebarCursor = 0
				}
				break
			}
		}
		return m, nil

	case errMsg:
		m.setStatusMsg("Error: " + msg.err.Error())
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

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
	case overlayPark:
		return m.handleParkKey(msg)
	case overlayRename:
		return m.handleRenameKey(msg)
	case overlayPromptEditor:
		return m.handlePromptEditorKey(msg)
	}

	// Tab: toggle sidebar focus. Entering syncs cursor to active session;
	// leaving confirms the cursor selection.
	if msg.Type == tea.KeyTab {
		if m.sidebarFocused {
			// Confirm cursor selection and return to session
			if m.sidebarCursor >= 0 && m.sidebarCursor < len(m.sessions) {
				m.switchSession(m.sidebarCursor)
			}
			m.sidebarFocused = false
		} else {
			// Enter sidebar: place cursor on currently active session
			m.sidebarCursor = m.activeIdx
			if m.sidebarCursor < 0 {
				m.sidebarCursor = 0
			}
			m.sidebarFocused = true
			if !m.sidebarOpen {
				m.sidebarOpen = true
				m.resizeViewport()
			}
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


func (m *Model) handleSidebarKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		// Confirm cursor selection and return to session
		if m.sidebarCursor >= 0 && m.sidebarCursor < len(m.sessions) {
			m.switchSession(m.sidebarCursor)
		}
		m.sidebarFocused = false
		return m, nil
	case tea.KeyEsc:
		// Cancel: return to session without changing active session
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
			m.sidebarCursor = idx
			m.sidebarFocused = false
		}
		return m, nil
	case "n":
		m.overlay = overlayNewSession
		m.nameInput = newSessionInput()
		return m, textinput.Blink
	case "N":
		m.overlay = overlayPromptEditor
		m.promptInput = newPromptEditor()
		return m, m.promptInput.Focus()
	case "x":
		if m.activeIdx >= 0 && m.activeIdx < len(m.sessions) {
			m.overlay = overlayCloseConfirm
		}
	case "p":
		if m.activeIdx >= 0 && m.activeIdx < len(m.sessions) && !m.sessions[m.activeIdx].Parked {
			m.overlay = overlayPark
			m.parkInput = newParkInput()
			return m, textinput.Blink
		}
	case "r":
		if m.activeIdx >= 0 && m.activeIdx < len(m.sessions) {
			m.overlay = overlayRename
			m.renameInput = newRenameInput(m.sessions[m.activeIdx].Name)
			return m, textinput.Blink
		}
	case "u":
		if m.activeIdx >= 0 && m.activeIdx < len(m.sessions) && m.sessions[m.activeIdx].Parked {
			m.sessions[m.activeIdx].Parked = false
			m.sessions[m.activeIdx].ParkNote = ""
			m.store.Sessions = m.sessions
			m.saveState()
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

func (m *Model) handleNewSessionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		name := strings.TrimSpace(m.nameInput.Value())
		if name == "" {
			name = fmt.Sprintf("session-%d", len(m.sessions)+1)
		}
		name = m.uniqueName(name)
		m.overlay = overlayNone
		return m, spawnSession(name, m.cfg.General.RootDir, "", len(m.sessions))

	case tea.KeyEsc:
		m.overlay = overlayNone
		return m, nil
	}

	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

func (m *Model) handleCloseConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.overlay = overlayNone
	if msg.String() == "y" || msg.String() == "Y" {
		if m.activeIdx >= 0 && m.activeIdx < len(m.sessions) {
			return m, closeSession(m.sessions[m.activeIdx].ID)
		}
	}
	return m, nil
}

// saveState persists the current session list. On failure it sets m.statusMsg so
// the user sees the error rather than silently losing state on next restart.
func (m *Model) saveState() {
	if err := state.Save(m.store); err != nil {
		m.setStatusMsg("Error: " + err.Error())
	}
}

// setStatusMsg sets a transient status bar message with an expiry timestamp.
func (m *Model) setStatusMsg(msg string) {
	m.statusMsg = msg
	m.statusMsgAt = time.Now()
}

// Commands (pure functions — no model mutation)

func capturePane(sessionID string) tea.Cmd {
	return func() tea.Msg {
		content, err := tmux.CapturePane(sessionID)
		if err != nil {
			return nil // session may still be starting
		}
		return paneContentMsg{sessionID: sessionID, content: content}
	}
}

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

func closeSession(sessionID string) tea.Cmd {
	return func() tea.Msg {
		err := tmux.Kill(sessionID)
		return sessionClosedMsg{sessionID: sessionID, err: err}
	}
}

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

// Helpers

func (m *Model) switchSession(idx int) {
	m.activeIdx = idx
	m.loadSession(idx)
}

func (m *Model) navigateSidebar(dir int) {
	if len(m.sessions) == 0 {
		return
	}
	next := m.sidebarCursor + dir
	if next < 0 {
		next = len(m.sessions) - 1
	} else if next >= len(m.sessions) {
		next = 0
	}
	m.sidebarCursor = next
}

func (m *Model) loadSession(idx int) {
	if idx < 0 || idx >= len(m.sessions) {
		return
	}
	m.viewport.SetContent(m.sessions[idx].PaneContent)
	m.viewport.GotoBottom()
}

func (m *Model) resizeViewport() {
	vpWidth := m.width
	if m.sidebarOpen {
		vpWidth -= sidebarWidth + 2 // +1 for left border, +1 for separator column
	}
	vpHeight := m.height - 2 // status bar (1) + hint line (1)
	if vpWidth < 1 {
		vpWidth = 1
	}
	if vpHeight < 1 {
		vpHeight = 1
	}
	m.viewport = newViewport(vpWidth, vpHeight)
	m.loadSession(m.activeIdx)
}

func (m *Model) killAllSessions() {
	for _, s := range m.sessions {
		tmux.Kill(s.ID)
	}
	m.sessions = nil
	m.store.Sessions = nil
	m.saveState()
}

func (m *Model) openEditor() tea.Cmd {
	claudeMD := filepath.Join(m.cfg.General.RootDir, "CLAUDE.md")
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	cmd := exec.Command(editor, claudeMD)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return nil
	})
}

func (m *Model) uniqueName(name string) string {
	existing := map[string]bool{}
	for _, s := range m.sessions {
		existing[s.Name] = true
	}
	candidate := name
	for i := 2; existing[candidate]; i++ {
		candidate = fmt.Sprintf("%s-%d", name, i)
	}
	return candidate
}

// View renders the complete TUI.
func (m *Model) View() string {
	if m.width == 0 {
		return ""
	}

	// Overlays take the full screen
	switch m.overlay {
	case overlayHelp:
		return renderHelp(m.width, m.height)
	case overlayNewSession:
		return renderNewSession(m.nameInput, m.width, m.height)
	case overlayCloseConfirm:
		name := ""
		if m.activeIdx >= 0 && m.activeIdx < len(m.sessions) {
			name = m.sessions[m.activeIdx].DisplayName()
		}
		return renderConfirm(fmt.Sprintf("Kill session %q? (y/N)", name), m.width, m.height)
	case overlayPark:
		return renderPark(m.parkInput, m.width, m.height)
	case overlayRename:
		return renderRename(m.renameInput, m.width, m.height)
	case overlayPromptEditor:
		return renderPromptEditor(m.promptInput, m.width, m.height)
	}

	statusBar := renderStatusBar(m.sessions, m.width, m.statusMsg)
	mainHeight := m.height - 2

	var mainContent string
	if m.sidebarOpen {
		sidebar := renderSidebar(m.sessions, m.activeIdx, m.sidebarCursor, mainHeight, m.tick, m.sidebarFocused)
		// Left border: always present, cyan when sidebar is focused
		borderColor := lipgloss.Color("238")
		if m.sidebarFocused {
			borderColor = lipgloss.Color("39")
		}
		sidebar = lipgloss.NewStyle().
			Border(lipgloss.ThickBorder(), false, false, false, true).
			BorderForeground(borderColor).
			BorderBackground(lipgloss.Color("236")).
			Render(sidebar)
		// Separator column
		sep := ""
		for i := 0; i < mainHeight; i++ {
			if i > 0 {
				sep += "\n"
			}
			sep += separatorStyle.Render("│")
		}
		vp := m.viewport.View()
		mainContent = lipgloss.JoinHorizontal(lipgloss.Top, sidebar, sep, vp)
	} else {
		mainContent = m.viewport.View()
	}

	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Background(lipgloss.Color("0"))
	var hintText string
	if m.sidebarFocused {
		hintText = " Esc: cancel   j/k ↑↓: navigate   Enter/Tab: select   n new   x close   ? help"
	} else {
		hintText = " Tab: open sidebar   \\ toggle   (all keys → Claude Code)"
	}
	hint := hintStyle.Width(m.width).Render(hintText)

	return lipgloss.JoinVertical(lipgloss.Left, statusBar, mainContent, hint)
}

// renderConfirm renders a centered confirmation dialog.
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
