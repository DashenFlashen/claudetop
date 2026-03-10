package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// skillVPWidthOffset and skillVPHeightOffset are the amounts to subtract from
// the terminal dimensions to get the correct skill viewport content size.
// They are derived from the overlay layout: centering padding (4) + inner
// horizontal padding (4) = 8 wide; border (2) + inner vertical padding (2) +
// title/blank/hint rows (4) + centering margin (4) = 12 tall.
const (
	skillVPWidthOffset  = 8
	skillVPHeightOffset = 12
)

var (
	skillOverlayStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("39")).
				Background(lipgloss.Color("235"))

	skillTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true)

	skillLoadingStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244"))

	skillHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
)

// renderSkillOutput renders the skill output overlay.
// When running is true and output is empty, shows a loading spinner.
// When output is non-empty, shows a scrollable viewport of the output.
func renderSkillOutput(name, output string, running bool, vp viewport.Model, tick int, width, height int) string {
	const padding = 2
	const borderSize = 2
	overlayWidth := width - 2*padding
	if overlayWidth < 20 {
		overlayWidth = 20
	}

	var body string
	if running && output == "" {
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		spinner := frames[tick%len(frames)]
		body = skillLoadingStyle.Render(spinner + "  Running " + name + "...")
	} else {
		body = vp.View()
	}

	var hint string
	if !running {
		hint = skillHintStyle.Render("j/k: scroll   Esc: close")
	}

	innerWidth := overlayWidth - borderSize - 2 // border + padding
	var lines []string
	lines = append(lines, skillTitleStyle.Width(innerWidth).Render(name))
	lines = append(lines, "")
	lines = append(lines, body)
	if hint != "" {
		lines = append(lines, "")
		lines = append(lines, hint)
	}

	content := skillOverlayStyle.
		Width(overlayWidth).
		Padding(1, 2).
		Render(strings.Join(lines, "\n"))

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
