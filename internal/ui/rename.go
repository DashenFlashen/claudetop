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
