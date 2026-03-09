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
				Bold(true)

	newSessionHintStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244"))
)

// newSessionInput creates a configured text input for session naming.
func newSessionInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "session name (blank for auto)"
	ti.CharLimit = 50
	ti.Width = 44
	ti.Focus()
	return ti
}

// renderNewSession renders the new session prompt overlay centered in the terminal.
func renderNewSession(input textinput.Model, width, height int) string {
	var lines []string
	lines = append(lines, newSessionTitleStyle.Render("New Session"))
	lines = append(lines, "")
	lines = append(lines, "Name:")
	lines = append(lines, input.View())
	lines = append(lines, "")
	lines = append(lines, newSessionHintStyle.Render("Enter: create   Esc: cancel"))

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
