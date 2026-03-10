package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

// Internal message types

type tickMsg time.Time
type flushKeyMsg struct{ sessionID string }

type paneContentMsg struct {
	sessionID string
	content   string
}

type sessionSpawnedMsg struct {
	sess *session.Session
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

type skillOutputMsg struct {
	name   string
	output string
	err    error
}

type inboxSessionSpawnedMsg struct {
	sess    *session.Session
	content string
}

// pendingInboxSend holds a scheduled tmux content send for an inbox-spawned session.
type pendingInboxSend struct {
	sessionID string
	content   string
	sendAt    time.Time
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
	overlaySkillOutput
	overlayCapture
	overlayInbox
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

	nameInput   textinput.Model
	parkInput   textinput.Model
	renameInput textinput.Model
	viewport    viewport.Model

	skillName    string         // name of skill shown in output overlay
	skillOutput  string         // output from completed skill run ("" = still running)
	skillRunning bool           // true while subprocess is executing
	skillVP      viewport.Model // scrollable viewport for skill output

	keyBuffer    string         // accumulated rune keystrokes pending tmux send
	captureInput textinput.Model
	inboxCursor  int

	pendingInboxSend *pendingInboxSend

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
		m.resizeSkillVP()
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

		// Fire deferred inbox content send when Claude has had time to start
		if m.pendingInboxSend != nil && time.Now().After(m.pendingInboxSend.sendAt) {
			send := m.pendingInboxSend
			m.pendingInboxSend = nil
			cmds = append(cmds, sendInboxContent(send.sessionID, send.content))
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
		return m, nil

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

	case skillOutputMsg:
		m.skillRunning = false
		if msg.err != nil {
			m.skillOutput = "Error: " + msg.err.Error()
		} else {
			m.skillOutput = msg.output
		}
		m.resizeSkillVP()
		m.skillVP.SetContent(m.skillOutput)
		return m, nil

	case sessionRenamedMsg:
		for _, s := range m.sessions {
			if s.ID == msg.sessionID {
				s.Name = msg.name
				m.store.Sessions = m.sessions
				m.saveState()
				m.setStatusMsg("Renamed to: " + msg.name)
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

	case flushKeyMsg:
		if m.keyBuffer != "" {
			buf := m.keyBuffer
			m.keyBuffer = ""
			return m, sendLiteralText(msg.sessionID, buf)
		}
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
	case overlaySkillOutput:
		return m.handleSkillOutputKey(msg)
	case overlayCapture:
		return m.handleCaptureKey(msg)
	case overlayInbox:
		return m.handleInboxKey(msg)
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

	// Session focused: forward everything to tmux.
	// Rune keys are buffered and flushed together to reduce subprocess spawns.
	if m.activeIdx >= 0 && m.activeIdx < len(m.sessions) {
		s := m.sessions[m.activeIdx]
		if msg.Type == tea.KeyRunes {
			m.keyBuffer += msg.String()
			return m, scheduleKeyFlush(s.ID)
		}
		buf := m.keyBuffer
		m.keyBuffer = ""
		return m, forwardKeyWithFlush(s.ID, buf, msg)
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
	case "R":
		if m.sidebarCursor >= 0 && m.sidebarCursor < len(m.sessions) {
			s := m.sessions[m.sidebarCursor]
			if !s.Dead && s.PaneContent != "" {
				m.setStatusMsg("Auto-naming session...")
				return m, autoNameFromContent(s.ID, s.PaneContent)
			}
		}
	case "x":
		if m.sidebarCursor >= 0 && m.sidebarCursor < len(m.sessions) {
			m.overlay = overlayCloseConfirm
		}
	case "p":
		if m.sidebarCursor >= 0 && m.sidebarCursor < len(m.sessions) && !m.sessions[m.sidebarCursor].Parked {
			m.overlay = overlayPark
			m.parkInput = newParkInput()
			return m, textinput.Blink
		}
	case "r":
		if m.sidebarCursor >= 0 && m.sidebarCursor < len(m.sessions) {
			m.overlay = overlayRename
			m.renameInput = newRenameInput(m.sessions[m.sidebarCursor].Name)
			return m, textinput.Blink
		}
	case "u":
		if m.sidebarCursor >= 0 && m.sidebarCursor < len(m.sessions) && m.sessions[m.sidebarCursor].Parked {
			m.sessions[m.sidebarCursor].Parked = false
			m.sessions[m.sidebarCursor].ParkNote = ""
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
	case "c":
		m.overlay = overlayCapture
		m.captureInput = newCaptureInput()
		return m, textinput.Blink
	case "b":
		m.inboxCursor = 0
		m.overlay = overlayInbox
	case "e":
		return m, m.openEditor()
	default:
		// Check configured skills
		for _, sk := range m.cfg.Skills {
			if msg.String() == sk.Key {
				return m.launchSkill(sk)
			}
		}
	}
	return m, nil
}

func (m *Model) launchSkill(sk config.SkillConfig) (tea.Model, tea.Cmd) {
	switch sk.Mode {
	case "output":
		if m.skillRunning {
			m.setStatusMsg(sk.Name + " is already running")
			return m, nil
		}
		m.overlay = overlaySkillOutput
		m.skillName = sk.Name
		m.skillOutput = ""
		m.skillRunning = true
		return m, runSkillOutput(sk, m.cfg.General.RootDir)
	case "interactive":
		name := m.uniqueName(sk.Name)
		m.sidebarFocused = false
		return m, spawnSessionWithCommand(name, m.cfg.General.RootDir, sk.Command, len(m.sessions))
	}
	return m, nil
}

func (m *Model) handleSkillOutputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.overlay = overlayNone
		return m, nil
	}
	switch msg.String() {
	case "j":
		m.skillVP.LineDown(1)
	case "k":
		m.skillVP.LineUp(1)
	case "q":
		m.overlay = overlayNone
		return m, nil
	}
	return m, nil
}

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
			m.removeInboxItem(item.ID)
			m.overlay = overlayNone
			name := m.uniqueName("inbox")
			return m, spawnSessionForInbox(name, m.cfg.General.RootDir, item.Content, len(m.sessions))
		}
	case "d":
		if m.inboxCursor < len(active) {
			m.removeInboxItem(active[m.inboxCursor].ID)
			newActive := activeInboxItems(m.store.InboxItems)
			if m.inboxCursor >= len(newActive) && m.inboxCursor > 0 {
				m.inboxCursor--
			}
		}
	case "p":
		if m.inboxCursor < len(active) {
			active[m.inboxCursor].Parked = true
			m.saveState()
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

func (m *Model) handleParkKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		if m.sidebarCursor >= 0 && m.sidebarCursor < len(m.sessions) {
			m.sessions[m.sidebarCursor].Parked = true
			m.sessions[m.sidebarCursor].ParkNote = strings.TrimSpace(m.parkInput.Value())
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
		if name != "" && m.sidebarCursor >= 0 && m.sidebarCursor < len(m.sessions) {
			m.sessions[m.sidebarCursor].Name = name
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

func (m *Model) handleNewSessionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		name := strings.TrimSpace(m.nameInput.Value())
		if name == "" {
			name = fmt.Sprintf("session-%d", len(m.sessions)+1)
		}
		name = m.uniqueName(name)
		m.overlay = overlayNone
		return m, spawnSession(name, m.cfg.General.RootDir, len(m.sessions))

	case tea.KeyEsc:
		m.overlay = overlayNone
		return m, nil
	}

	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

func (m *Model) handleCloseConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "y" || msg.String() == "Y":
		m.overlay = overlayNone
		if m.sidebarCursor >= 0 && m.sidebarCursor < len(m.sessions) {
			return m, closeSession(m.sessions[m.sidebarCursor].ID)
		}
	case msg.String() == "n" || msg.String() == "N" || msg.Type == tea.KeyEsc:
		m.overlay = overlayNone
	}
	// All other keys ignored — accidental Enter does not cancel
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

// newSession creates a tmux session running claude and returns the session, or an error message.
func newSession(name, rootDir string, sessionCount int) (*session.Session, tea.Msg) {
	s := session.NewSession(name, sessionCount+1)
	if _, err := os.Stat(rootDir); err != nil {
		return nil, errMsg{fmt.Errorf("root_dir %q does not exist: %w", rootDir, err)}
	}
	if err := tmux.Create(s.ID, rootDir); err != nil {
		return nil, errMsg{fmt.Errorf("create session: %w", err)}
	}
	return s, nil
}

func spawnSession(name, rootDir string, sessionCount int) tea.Cmd {
	return func() tea.Msg {
		s, errM := newSession(name, rootDir, sessionCount)
		if errM != nil {
			return errM
		}
		return sessionSpawnedMsg{sess: s}
	}
}

// spawnSessionForInbox creates a new session and returns inboxSessionSpawnedMsg so the
// model can schedule the deferred content send.
func spawnSessionForInbox(name, rootDir, content string, sessionCount int) tea.Cmd {
	return func() tea.Msg {
		s, errM := newSession(name, rootDir, sessionCount)
		if errM != nil {
			return errM
		}
		return inboxSessionSpawnedMsg{sess: s, content: content}
	}
}

func closeSession(sessionID string) tea.Cmd {
	return func() tea.Msg {
		err := tmux.Kill(sessionID)
		return sessionClosedMsg{sessionID: sessionID, err: err}
	}
}

// sendInboxContent sends captured text to a tmux session as if typed by the user.
func sendInboxContent(sessionID, content string) tea.Cmd {
	return func() tea.Msg {
		tmux.SendLiteralKey(sessionID, content)
		tmux.SendKeys(sessionID, "Enter")
		return nil
	}
}

// spawnSessionWithCommand spawns a tmux session running the given command instead of bare claude.
func spawnSessionWithCommand(name, rootDir, command string, sessionCount int) tea.Cmd {
	return func() tea.Msg {
		s := session.NewSession(name, sessionCount+1)
		if _, err := os.Stat(rootDir); err != nil {
			return errMsg{fmt.Errorf("root_dir %q does not exist: %w", rootDir, err)}
		}
		if err := tmux.CreateWithCommand(s.ID, rootDir, command); err != nil {
			return errMsg{fmt.Errorf("create session: %w", err)}
		}
		return sessionSpawnedMsg{sess: s}
	}
}

// runSkillOutput runs a skill command as a subprocess and returns its combined output.
// Best-effort: errors are captured and shown in the overlay.
// Note: command is split on whitespace; quoted arguments with spaces are not supported.
func runSkillOutput(sk config.SkillConfig, rootDir string) tea.Cmd {
	return func() tea.Msg {
		parts := strings.Fields(sk.Command)
		if len(parts) == 0 {
			return skillOutputMsg{name: sk.Name, err: fmt.Errorf("empty command")}
		}
		cmd := exec.Command(parts[0], parts[1:]...)
		cmd.Dir = rootDir
		out, err := cmd.CombinedOutput()
		return skillOutputMsg{name: sk.Name, output: string(out), err: err}
	}
}

// autoNameFromContent calls claude -p to summarize the pane content into a short session name.
// Returns nil on any failure — auto-naming is best-effort.
// Pane content is truncated to stay within OS argument length limits.
func autoNameFromContent(sessionID, paneContent string) tea.Cmd {
	const maxContent = 3000
	if len(paneContent) > maxContent {
		paneContent = paneContent[len(paneContent)-maxContent:]
	}
	return func() tea.Msg {
		out, err := exec.Command("claude", "-p",
			"Based on this terminal session content, summarize the task in 3 words max, lowercase, hyphen-separated, no punctuation. Output only the name, nothing else:\n\n"+paneContent,
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
	m.keyBuffer = ""
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
	// -2 for viewport left+right border (always present, colour changes with focus)
	vpWidth := m.width - 2
	if m.sidebarOpen {
		// -1 for sidebar ThickBorder left; viewport left border replaces separator column
		vpWidth -= sidebarWidth + 1
	}
	// -2 for viewport top+bottom border; -2 for status bar + hint line
	vpHeight := m.height - 4
	if vpWidth < 1 {
		vpWidth = 1
	}
	if vpHeight < 1 {
		vpHeight = 1
	}
	m.viewport = newViewport(vpWidth, vpHeight)
	m.loadSession(m.activeIdx)
}

func (m *Model) resizeSkillVP() {
	w := m.width - skillVPWidthOffset
	h := m.height - skillVPHeightOffset
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	m.skillVP.Width = w
	m.skillVP.Height = h
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
		if m.sidebarCursor >= 0 && m.sidebarCursor < len(m.sessions) {
			name = m.sessions[m.sidebarCursor].DisplayName()
		}
		return renderConfirm(fmt.Sprintf("Kill session %q? (y/N)", name), m.width, m.height)
	case overlayPark:
		return renderPark(m.parkInput, m.width, m.height)
	case overlayRename:
		return renderRename(m.renameInput, m.width, m.height)
	case overlaySkillOutput:
		return renderSkillOutput(m.skillName, m.skillOutput, m.skillRunning, m.skillVP, m.tick, m.width, m.height)
	case overlayCapture:
		return renderCapture(m.captureInput, m.width, m.height)
	case overlayInbox:
		return renderInbox(m.store.InboxItems, m.inboxCursor, m.width, m.height)
	}

	statusBar := renderStatusBar(m.sessions, m.width, m.statusMsg)
	mainHeight := m.height - 2

	// Viewport border: green when session focused, dim when sidebar focused
	vpBorderColor := lipgloss.Color("237")
	if !m.sidebarFocused {
		vpBorderColor = lipgloss.Color("82")
	}
	vp := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(vpBorderColor).
		Render(m.viewport.View())

	var mainContent string
	if m.sidebarOpen {
		sidebar := renderSidebar(m.sessions, m.cfg.Skills, m.activeIdx, m.sidebarCursor, mainHeight, m.tick, m.sidebarFocused)
		// Left border: always present, cyan when sidebar is focused
		sidebarBorderColor := lipgloss.Color("238")
		if m.sidebarFocused {
			sidebarBorderColor = lipgloss.Color("39")
		}
		sidebar = lipgloss.NewStyle().
			Border(lipgloss.ThickBorder(), false, false, false, true).
			BorderForeground(sidebarBorderColor).
			BorderBackground(lipgloss.Color("236")).
			Render(sidebar)
		mainContent = lipgloss.JoinHorizontal(lipgloss.Top, sidebar, vp)
	} else {
		mainContent = vp
	}

	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Background(lipgloss.Color("0"))
	var badgeStyle lipgloss.Style
	var modeBadge, hintText string
	if m.sidebarFocused {
		badgeStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("39")).
			Foreground(lipgloss.Color("0")).
			Bold(true).
			Padding(0, 1)
		modeBadge = "SIDEBAR"
		hintText = "  Esc/Tab: session   j/k: navigate   n new   x close   r rename   R auto-name   ? help"
	} else {
		badgeStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("34")).
			Foreground(lipgloss.Color("0")).
			Bold(true).
			Padding(0, 1)
		modeBadge = "SESSION"
		hintText = "  Tab: sidebar   \\ toggle   (keys → Claude Code)"
	}
	badge := badgeStyle.Render(modeBadge)
	hint := lipgloss.JoinHorizontal(lipgloss.Top, badge, hintStyle.Width(m.width-lipgloss.Width(badge)).Render(hintText))

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
