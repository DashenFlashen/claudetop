package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"claudetop/internal/config"
	"claudetop/internal/session"
)

const sidebarWidth = 26

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

func renderSessionLine(s *session.Session, idx, activeIdx, cursorIdx, tick int) string {
	// Dead sessions need more room for the "[dead]" suffix.
	maxName := sidebarWidth - 5
	if s.Dead {
		maxName = sidebarWidth - 12
	}
	if maxName < 1 {
		maxName = 1
	}

	name := s.DisplayName()
	if len([]rune(name)) > maxName {
		runes := []rune(name)
		name = string(runes[:maxName-1]) + "…"
	}

	// ▶ marks the session currently shown in the viewport when the cursor is elsewhere.
	prefix := " "
	if idx == activeIdx && idx != cursorIdx {
		prefix = "▶"
	}

	var line string
	if s.Dead {
		line = fmt.Sprintf("%s%d ✗ %s [dead]", prefix, idx+1, name)
	} else {
		var dot string
		if s.Parked {
			dot = dotHollow.Render("○")
		} else {
			dot = statusDot(s, tick)
		}
		line = fmt.Sprintf("%s%d %s %s", prefix, idx+1, dot, name)
	}

	style := sidebarItemStyle.Width(sidebarWidth)
	if idx == cursorIdx {
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
func renderSidebar(sessions []*session.Session, skills []config.SkillConfig, activeIdx, cursorIdx, height, tick int, focused bool) string {
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

	if len(activeIndices) == 0 {
		lines = append(lines, sidebarItemStyle.Width(sidebarWidth).Render("  (none)"))
		lines = append(lines, sidebarItemStyle.Width(sidebarWidth).Render("  n: new session"))
	} else {
		for _, origIdx := range activeIndices {
			lines = append(lines, renderSessionLine(sessions[origIdx], origIdx, activeIdx, cursorIdx, tick))
		}
	}

	if len(parkedIndices) > 0 {
		lines = append(lines, sidebarStyle.Width(sidebarWidth).Render(""))
		lines = append(lines, headerStyle.Width(sidebarWidth).Render("PARKED"))
		for _, origIdx := range parkedIndices {
			s := sessions[origIdx]
			lines = append(lines, renderSessionLine(s, origIdx, activeIdx, cursorIdx, tick))
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

	if len(skills) > 0 {
		lines = append(lines, sidebarStyle.Width(sidebarWidth).Render(""))
		lines = append(lines, headerStyle.Width(sidebarWidth).Render("SKILLS"))
		for _, sk := range skills {
			name := sk.Name
			maxName := sidebarWidth - 6 // " [x] " prefix
			if len([]rune(name)) > maxName {
				runes := []rune(name)
				name = string(runes[:maxName-1]) + "…"
			}
			line := fmt.Sprintf(" [%s] %s", sk.Key, name)
			lines = append(lines, sidebarItemStyle.Width(sidebarWidth).Render(line))
		}
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
