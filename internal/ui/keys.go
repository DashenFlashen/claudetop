package ui

import (
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
