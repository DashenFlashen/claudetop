package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

var (
	captureOverlayStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("220")).
				Background(lipgloss.Color("235")).
				Padding(1, 2).
				Width(60)

	captureTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")).
				Bold(true)

	captureHintStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244"))
)

// newCaptureInput creates a configured text input for inbox capture.
func newCaptureInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "what's on your mind?"
	ti.CharLimit = 200
	ti.Width = 54
	ti.Focus()
	return ti
}

// renderCapture renders the capture overlay centered in the terminal.
func renderCapture(input textinput.Model, width, height int) string {
	var lines []string
	lines = append(lines, captureTitleStyle.Render("Capture"))
	lines = append(lines, "")
	lines = append(lines, input.View())
	lines = append(lines, "")
	lines = append(lines, captureHintStyle.Render("Enter: save   Esc: cancel"))

	content := captureOverlayStyle.Render(strings.Join(lines, "\n"))

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
