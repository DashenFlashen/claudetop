package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"claudetop/internal/tmux"
)

// tmuxKeyName maps a Bubbletea key to a tmux send-keys name.
// literal=true means use -l flag (send as text); literal=false means named key.
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
		return msg.String(), true
	default:
		return msg.String(), true
	}
}

// scheduleKeyFlush returns a one-shot 15ms timer that fires flushKeyMsg,
// causing any buffered rune characters to be sent to tmux in one subprocess call.
func scheduleKeyFlush(sessionID string) tea.Cmd {
	return tea.Tick(15*time.Millisecond, func(t time.Time) tea.Msg {
		return flushKeyMsg{sessionID: sessionID}
	})
}

// sendLiteralText sends a string of text to a tmux session as a single subprocess call.
func sendLiteralText(sessionID, text string) tea.Cmd {
	return func() tea.Msg {
		if text != "" {
			tmux.SendLiteralKey(sessionID, text)
		}
		return nil
	}
}

// forwardKeyWithFlush flushes any buffered text then sends a special (non-rune) key,
// both in a single goroutine to guarantee ordering.
func forwardKeyWithFlush(sessionID, buffer string, msg tea.KeyMsg) tea.Cmd {
	return func() tea.Msg {
		if buffer != "" {
			tmux.SendLiteralKey(sessionID, buffer)
		}
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

// forwardKey sends a keypress to the active tmux session as a tea.Cmd.
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
