package ui

import (
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// newViewport creates a configured viewport for rendering tmux pane output.
func newViewport(width, height int) viewport.Model {
	vp := viewport.New(width, height)
	vp.Style = lipgloss.NewStyle().Background(lipgloss.Color("0"))
	return vp
}
