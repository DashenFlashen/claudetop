package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
)

var promptEditorOverlayStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("240")).
	Background(lipgloss.Color("235")).
	Padding(1, 2).
	Width(66)

func newPromptEditor() textarea.Model {
	ta := textarea.New()
	ta.Placeholder = "Write your prompt for Claude..."
	ta.CharLimit = 2000
	ta.SetWidth(60)
	ta.SetHeight(8)
	ta.Focus()
	return ta
}

func renderPromptEditor(input textarea.Model, width, height int) string {
	var lines []string
	lines = append(lines, newSessionTitleStyle.Render("New Session — Write Prompt"))
	lines = append(lines, "")
	lines = append(lines, input.View())
	lines = append(lines, "")
	lines = append(lines, newSessionHintStyle.Render("Ctrl+S: spawn session   Esc: cancel"))

	content := promptEditorOverlayStyle.Render(strings.Join(lines, "\n"))

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
