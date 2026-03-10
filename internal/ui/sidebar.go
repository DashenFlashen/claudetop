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

	dotHollow = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	parkedNoteStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("240"))

	sidebarFocusedHeaderStyle = lipgloss.NewStyle().
					Background(lipgloss.Color("236")).
					Foreground(lipgloss.Color("39")). // cyan
					Bold(true)
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
		return dotRed.Render("●[!]")
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

func renderSessionLine(s *session.Session, idx, activeIdx, tick int) string {
	var dot string
	if s.Dead {
		dot = "✗"
	} else if s.Parked {
		dot = dotHollow.Render("○")
	} else {
		dot = statusDot(s, tick)
	}

	name := s.DisplayName()
	maxName := sidebarWidth - 5
	if len([]rune(name)) > maxName {
		runes := []rune(name)
		name = string(runes[:maxName-1]) + "…"
	}

	var line string
	if s.Dead {
		line = fmt.Sprintf(" %d ✗ %s [dead]", idx+1, name)
	} else {
		line = fmt.Sprintf(" %d %s %s", idx+1, dot, name)
	}

	style := sidebarItemStyle.Width(sidebarWidth)
	if idx == activeIdx {
		style = sidebarActiveStyle.Width(sidebarWidth)
	}
	if s.Dead {
		style = lipgloss.NewStyle().
			Width(sidebarWidth).
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("240"))
	}
	return style.Render(line)
}

// renderSidebar renders the session list sidebar.
// activeIdx is the currently focused session (-1 if none).
// tick is the animation counter for the working spinner.
func renderSidebar(sessions []*session.Session, activeIdx int, height int, tick int, focused bool) string {
	var lines []string

	headerStyle := sidebarHeaderStyle
	if focused {
		headerStyle = sidebarFocusedHeaderStyle
	}

	// Separate active and parked
	var activeIndices, parkedIndices []int
	for i, s := range sessions {
		if s.Parked {
			parkedIndices = append(parkedIndices, i)
		} else {
			activeIndices = append(activeIndices, i)
		}
	}

	lines = append(lines, headerStyle.Width(sidebarWidth).Render("ACTIVE"))
	lines = append(lines, sidebarStyle.Render(""))

	if len(activeIndices) == 0 {
		lines = append(lines, sidebarItemStyle.Width(sidebarWidth).Render("  (none)"))
		lines = append(lines, sidebarItemStyle.Width(sidebarWidth).Render("  n: new session"))
	} else {
		for _, origIdx := range activeIndices {
			lines = append(lines, renderSessionLine(sessions[origIdx], origIdx, activeIdx, tick))
		}
	}

	if len(parkedIndices) > 0 {
		lines = append(lines, sidebarStyle.Width(sidebarWidth).Render(""))
		lines = append(lines, headerStyle.Width(sidebarWidth).Render("PARKED"))
		lines = append(lines, sidebarStyle.Render(""))
		for _, origIdx := range parkedIndices {
			s := sessions[origIdx]
			lines = append(lines, renderSessionLine(s, origIdx, activeIdx, tick))
			if s.ParkNote != "" {
				note := "   \"" + s.ParkNote + "\""
				maxNote := sidebarWidth - 2
				if len([]rune(note)) > maxNote {
					runes := []rune(note)
					note = string(runes[:maxNote-1]) + "…"
				}
				lines = append(lines, parkedNoteStyle.Width(sidebarWidth).Render(note))
			}
		}
	}

	// Footer hint
	lines = append(lines, sidebarStyle.Width(sidebarWidth).Render(""))
	if focused {
		lines = append(lines, sidebarFocusedHeaderStyle.Width(sidebarWidth).Render(" n new  x close  p park"))
	} else {
		lines = append(lines, sidebarItemStyle.Width(sidebarWidth).Render(" Tab: focus sidebar"))
	}

	// Pad to fill height
	for len(lines) < height {
		lines = append(lines, sidebarStyle.Width(sidebarWidth).Render(""))
	}
	if len(lines) > height {
		lines = lines[:height]
	}

	return strings.Join(lines, "\n")
}
