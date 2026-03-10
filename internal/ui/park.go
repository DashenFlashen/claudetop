package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

func newParkInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "note (optional, e.g. \"waiting for review\")"
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
