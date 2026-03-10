package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	helpOverlayStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Background(lipgloss.Color("235")).
				Padding(1, 2)

	helpTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Bold(true)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			Width(10)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250"))

	helpSectionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244"))
)

type helpEntry struct {
	key  string
	desc string
}

var helpEntries = []helpEntry{
	{"Always", ""},
	{"\\", "Toggle sidebar"},
	{"n", "New session"},
	{"x", "Close session (confirm)"},
	{"", ""},
	{"; prefix", ""},
	{";1-9", "Switch to session by number"},
	{";] / ;[", "Next / prev session"},
	{";?", "Help"},
	{";n", "New session"},
	{";q", "Quit (sessions keep running)"},
	{";Q", "Quit and kill all sessions"},
	{";e", "Edit root CLAUDE.md in $EDITOR"},
	{"", ""},
	{"Sidebar only", ""},
	{"1-9 / j / k", "Navigate sessions"},
	{"", ""},
	{"Esc", "Close overlay"},
}

// renderHelp renders the help overlay centered in the terminal.
func renderHelp(width, height int) string {
	var lines []string
	lines = append(lines, helpTitleStyle.Render("claudetop — keyboard shortcuts"))
	lines = append(lines, "")

	for _, e := range helpEntries {
		if e.key == "" && e.desc == "" {
			lines = append(lines, "")
			continue
		}
		if e.desc == "" {
			lines = append(lines, helpSectionStyle.Render(e.key))
			continue
		}
		lines = append(lines, helpKeyStyle.Render(e.key)+"  "+helpDescStyle.Render(e.desc))
	}

	content := helpOverlayStyle.Render(strings.Join(lines, "\n"))

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
