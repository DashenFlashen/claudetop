package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"claudetop/internal/session"
)

const sidebarWidth = 22

var (
	sidebarStyle = lipgloss.NewStyle().
			Width(sidebarWidth).
			Background(lipgloss.Color("236"))

	sidebarHeaderStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("236")).
				Foreground(lipgloss.Color("244")).
				Bold(true)

	sidebarItemStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("236")).
				Foreground(lipgloss.Color("252"))

	sidebarActiveStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("238")).
				Foreground(lipgloss.Color("255")).
				Bold(true)

	dotGrey   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	dotYellow = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	dotRed    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	dotGreen  = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	dotOrange = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

func statusDot(s *session.Session, tick int) string {
	switch s.Status {
	case session.StatusStarting:
		return dotGrey.Render("●")
	case session.StatusWorking:
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		return dotYellow.Render(frames[tick%len(frames)])
	case session.StatusNeedsInput:
		return dotRed.Render("●")
	case session.StatusPermission:
		return dotRed.Render("●!")
	case session.StatusDone:
		return dotGreen.Render("●")
	case session.StatusStuck:
		return dotOrange.Render("●")
	case session.StatusError:
		return dotRed.Render("●✗")
	default:
		return dotGrey.Render("●")
	}
}

// renderSidebar renders the session list sidebar.
// activeIdx is the currently focused session (-1 if none).
// tick is the animation counter for the working spinner.
func renderSidebar(sessions []*session.Session, activeIdx int, height int, tick int) string {
	var lines []string

	lines = append(lines, sidebarHeaderStyle.Width(sidebarWidth).Render("ACTIVE"))
	lines = append(lines, sidebarStyle.Render(""))

	if len(sessions) == 0 {
		lines = append(lines, sidebarItemStyle.Width(sidebarWidth).Render("  (none)"))
		lines = append(lines, sidebarItemStyle.Width(sidebarWidth).Render("  n: new session"))
	} else {
		for i, s := range sessions {
			dot := statusDot(s, tick)
			name := s.DisplayName()

			// Truncate name to fit within sidebar
			maxName := sidebarWidth - 5 // " N ● name"
			if len([]rune(name)) > maxName {
				runes := []rune(name)
				name = string(runes[:maxName-1]) + "…"
			}

			line := fmt.Sprintf(" %d %s %s", i+1, dot, name)

			style := sidebarItemStyle.Width(sidebarWidth)
			if i == activeIdx {
				style = sidebarActiveStyle.Width(sidebarWidth)
			}
			lines = append(lines, style.Render(line))
		}
	}

	// Footer hint
	lines = append(lines, sidebarStyle.Width(sidebarWidth).Render(""))
	lines = append(lines, sidebarItemStyle.Width(sidebarWidth).Render(" n new  x close"))

	// Pad to fill height
	for len(lines) < height {
		lines = append(lines, sidebarStyle.Width(sidebarWidth).Render(""))
	}
	if len(lines) > height {
		lines = lines[:height]
	}

	return strings.Join(lines, "\n")
}
